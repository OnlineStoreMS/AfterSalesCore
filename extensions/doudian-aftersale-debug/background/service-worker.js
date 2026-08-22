/** OSMS 抖店售后【诊断】— 仅手动执行，过程日志上报云端临时目录 */
const STORAGE = {
  device: 'aftersalePlugin',
  apiBase: 'aftersaleApiBase',
  lastSync: 'aftersaleLastSync',
  lastError: 'aftersaleLastError',
  lastSyncAt: 'aftersaleLastSyncAt',
}

const DEFAULT_API_BASE = 'https://osms.zfcycle.com/apps/aftersales/api/v1'
const WORKBENCH_URL = 'https://fxg.jinritemai.com/ffa/merchant-aftersale-workbench/aftersale/list'
const SERVICE_ORDER_URL = 'https://fxg.jinritemai.com/ffa/task-order/service'

let syncing = false
let opening = false
let currentRun = null

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}

function clip(v) {
  try {
    const s = JSON.stringify(v)
    if (s.length <= 8000) return JSON.parse(s)
    return { _truncated: true, preview: s.slice(0, 8000) }
  } catch {
    return String(v).slice(0, 2000)
  }
}

function startRun(kind) {
  const run = {
    runId: `${kind}-${Date.now()}-${Math.random().toString(16).slice(2, 8)}`,
    kind,
    t0: Date.now(),
    events: [],
    log(step, data, level = 'info') {
      const ev = {
        ms: Date.now() - run.t0,
        at: new Date().toISOString(),
        level,
        step,
        data: data == null ? undefined : clip(data),
      }
      run.events.push(ev)
      console.log(`[diag ${run.runId}] ${level} ${step}`, data)
    },
    err(step, e) {
      const msg = e instanceof Error ? e.message : String(e)
      run.log(
        step,
        { error: msg, stack: e instanceof Error ? String(e.stack || '').slice(0, 1500) : undefined },
        'error',
      )
    },
  }
  currentRun = run
  return run
}

function logStep(step, data, level) {
  if (currentRun) currentRun.log(step, data, level || 'info')
}

function probePageFn() {
  const textOf = (el) => String(el?.innerText || el?.textContent || '').replace(/\s+/g, ' ').trim()
  const nodes = Array.from(document.querySelectorAll('iframe'))
  const iframes = nodes.map((f) => ({
    src: String(f.src || '').slice(0, 220),
    w: f.offsetWidth,
    h: f.offsetHeight,
    sameOrigin: false,
  }))
  for (let i = 0; i < nodes.length; i++) {
    try {
      const d = nodes[i].contentDocument
      iframes[i].sameOrigin = !!d
      if (d) {
        iframes[i].childHref = String(d.location?.href || '').slice(0, 220)
        iframes[i].childTabs = Array.from(d.querySelectorAll('.auxo-tabs-tab, [role="tab"]'))
          .map(textOf)
          .filter(Boolean)
          .slice(0, 20)
        iframes[i].childTitles = Array.from(d.querySelectorAll('[class*="groupTitle"]'))
          .map(textOf)
          .filter(Boolean)
          .slice(0, 12)
      }
    } catch {
      /* cross-origin */
    }
  }
  return {
    href: location.href,
    title: document.title,
    readyState: document.readyState,
    iframeCount: iframes.length,
    iframes,
    tabs: Array.from(document.querySelectorAll('.auxo-tabs-tab, [role="tab"]'))
      .map(textOf)
      .filter(Boolean)
      .slice(0, 30),
    groupTitles: Array.from(document.querySelectorAll('[class*="groupTitle"]'))
      .map(textOf)
      .filter(Boolean)
      .slice(0, 16),
    menuLabels: Array.from(
      document.querySelectorAll('.auxo-menu-item, .auxo-menu-submenu-title, [role="menuitem"]'),
    )
      .map(textOf)
      .filter((t) => t && t.length <= 16)
      .slice(0, 40),
  }
}

async function probeTab(tabId, label) {
  try {
    const tab = await chrome.tabs.get(tabId)
    const injected = await chrome.scripting.executeScript({
      target: { tabId },
      func: probePageFn,
    })
    const probe = {
      tab: {
        id: tab.id,
        url: tab.url,
        status: tab.status,
        discarded: !!tab.discarded,
        active: !!tab.active,
        title: tab.title,
      },
      page: injected?.[0]?.result,
    }
    logStep(label || 'probe', probe)
    return probe
  } catch (e) {
    if (currentRun) currentRun.err(label || 'probe', e)
    return null
  }
}

async function uploadRun(run, extra) {
  const body = {
    runId: run.runId,
    kind: run.kind,
    ok: !!extra.ok,
    error: extra.error || '',
    durationMs: Date.now() - run.t0,
    version: chrome.runtime.getManifest().version,
    meta: extra.meta || {},
    events: run.events,
  }
  await chrome.storage.local.set({
    lastDiagRunId: run.runId,
    lastDiagUpload: '',
    [STORAGE.lastError]: extra.error || '',
  })
  const device = await getDevice()
  if (!device?.pluginKey) {
    await chrome.storage.local.set({ lastDiagUpload: '未绑定，仅本地记录' })
    return { uploaded: false }
  }
  try {
    const data = await api('/plugin/debug-log', { auth: true, body })
    await chrome.storage.local.set({ lastDiagUpload: data?.name || 'ok' })
    return { uploaded: true, name: data?.name }
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    await chrome.storage.local.set({ lastDiagUpload: `上报失败: ${msg}` })
    return { uploaded: false, error: msg }
  }
}

function isFxgUsable(tab) {
  return !!(
    tab?.id &&
    tab.url &&
    tab.url.includes('fxg.jinritemai.com') &&
    !/login|passport/i.test(tab.url)
  )
}

function scoreFxgTab(tab, savedId) {
  let s = 0
  if (tab.id === savedId) s += 40
  if (tab.active) s += 30
  if (String(tab.url || '').includes('merchant-aftersale-workbench')) s += 25
  if (String(tab.url || '').includes('/ffa/')) s += 10
  if (!tab.discarded) s += 20
  return s
}

async function findFxgTab() {
  const stored = await chrome.storage.local.get(['aftersaleFxgTabId'])
  const savedId = Number(stored.aftersaleFxgTabId || 0)
  const all = await chrome.tabs.query({ url: ['https://fxg.jinritemai.com/*'] })
  const usable = all.filter(isFxgUsable)
  const live = usable.filter((t) => !t.discarded)
  const pool = live.length ? live : usable
  pool.sort((a, b) => scoreFxgTab(b, savedId) - scoreFxgTab(a, savedId))
  return pool[0] || null
}

async function rememberFxgTab(tabId) {
  await chrome.storage.local.set({ aftersaleFxgTabId: tabId })
}

async function waitTabComplete(tabId, timeoutMs = 25000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    let tab
    try {
      tab = await chrome.tabs.get(tabId)
    } catch {
      throw new Error('抖店标签已关闭')
    }
    if (tab.status === 'complete') {
      if (isFxgUsable(tab)) return tab
      if (/login|passport/i.test(tab.url || '')) {
        throw new Error('请先在打开的标签页登录抖店后台')
      }
    }
    await sleep(300)
  }
  return chrome.tabs.get(tabId)
}

async function ensureTabAwake(tab) {
  if (!tab?.id) throw new Error('请先打开抖店后台')
  if (tab.discarded) {
    await chrome.tabs.reload(tab.id)
    return waitTabComplete(tab.id)
  }
  return tab
}

function normalizeApiBase(input) {
  let u = String(input || '').trim().replace(/\/+$/, '')
  if (!u) u = DEFAULT_API_BASE.replace(/\/api\/v1$/, '')
  if (!/\/api\/v1$/i.test(u)) {
    if (/\/api$/i.test(u)) u += '/v1'
    else u += '/api/v1'
  }
  return u
}

function displayApiBase(apiBase) {
  return String(apiBase || '').replace(/\/api\/v1$/i, '')
}

async function getApiBase() {
  const data = await chrome.storage.local.get([STORAGE.apiBase])
  return normalizeApiBase(data[STORAGE.apiBase] || DEFAULT_API_BASE)
}

async function setApiBase(v) {
  await chrome.storage.local.set({ [STORAGE.apiBase]: normalizeApiBase(v) })
}

async function getDevice() {
  const data = await chrome.storage.local.get([STORAGE.device])
  return data[STORAGE.device] || null
}

async function setDevice(device) {
  if (!device) await chrome.storage.local.remove(STORAGE.device)
  else await chrome.storage.local.set({ [STORAGE.device]: device })
}

async function api(path, { method = 'POST', body, auth = false } = {}) {
  const apiBase = await getApiBase()
  const headers = { 'Content-Type': 'application/json' }
  if (auth) {
    const device = await getDevice()
    if (!device?.pluginKey || !device?.pluginSecret) {
      throw new Error('尚未绑定店铺')
    }
    headers['X-Plugin-Key'] = device.pluginKey
    headers['X-Plugin-Secret'] = device.pluginSecret
  }
  const res = await fetch(`${apiBase}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : '{}',
  })
  let json = null
  try {
    json = await res.json()
  } catch {
    throw new Error(`请求失败 HTTP ${res.status}`)
  }
  if (!res.ok || json?.code !== 200) {
    throw new Error(json?.message || `请求失败 HTTP ${res.status}`)
  }
  return json.data
}

function installPageLogisticsExtractor() {
  if (window.__osmsAftersaleLogisticsInstalled) return true
  window.__osmsAftersaleLogisticsInstalled = true
  const fiberOf = (el) => {
    if (!el) return null
    const key = Object.keys(el).find(
      (k) => k.startsWith('__reactFiber') || k.startsWith('__reactInternalInstance'),
    )
    return key ? el[key] : null
  }
  const recordOf = (el) => {
    let n = fiberOf(el)
    for (let i = 0; i < 24 && n; i++) {
      const p = n.memoizedProps || n.pendingProps
      if (p && p.record) return p.record
      n = n.return
    }
    return null
  }
  const fromRecord = (rec) => {
    if (!rec) return null
    const info = rec.parentRecord?.after_sale_info || rec.after_sale_info || {}
    const id = String(info.after_sale_id || rec.parentRecord?.after_sale_id || '')
    const code = String(info.return_logistics_code || rec.return_logistics_code || '').trim()
    if (!id) return null
    return { id, code }
  }
  const collectMap = () => {
    const map = {}
    const table = document.querySelector('table')
    let n = fiberOf(table)
    for (let i = 0; i < 40 && n; i++) {
      const p = n.memoizedProps || n.pendingProps
      const data = p?.data || p?.dataSource
      if (Array.isArray(data) && data.length) {
        for (const rec of data) {
          const hit = fromRecord(rec)
          if (hit?.code) map[hit.id] = hit.code
        }
        break
      }
      n = n.return
    }
    document.querySelectorAll('table tr').forEach((tr) => {
      const hit = fromRecord(recordOf(tr))
      if (hit?.code) map[hit.id] = hit.code
    })
    return map
  }
  document.addEventListener('osms-aftersale-need-logistics', () => {
    document.dispatchEvent(new CustomEvent('osms-aftersale-logistics', { detail: collectMap() }))
  })
  return true
}

async function installLogisticsExtractor(tabId) {
  try {
    await chrome.scripting.executeScript({
      target: { tabId, allFrames: true },
      world: 'MAIN',
      func: installPageLogisticsExtractor,
    })
  } catch (e) {
    console.warn('install logistics extractor failed', e)
  }
}

function installPageServiceExtractor() {
  if (window.__osmsServiceExtractorInstalled) return true
  window.__osmsServiceExtractorInstalled = true
  const fiberOf = (el) => {
    if (!el) return null
    const key = Object.keys(el).find(
      (k) => k.startsWith('__reactFiber') || k.startsWith('__reactInternalInstance'),
    )
    return key ? el[key] : null
  }
  const recordOf = (el) => {
    let n = fiberOf(el)
    for (let i = 0; i < 24 && n; i++) {
      const p = n.memoizedProps || n.pendingProps
      if (p && p.record) return p.record
      n = n.return
    }
    return null
  }
  const serialize = (rec) => {
    if (!rec) return null
    const os = rec.orderService || {}
    const oi = rec.orderInfo || {}
    const ui = rec.userInfo || {}
    const log = rec.lastLogInfo || {}
    const id = String(os.service_id || rec.mixTaskOrderId || '')
    if (!id) return null
    const tags = Array.isArray(rec.tagList)
      ? rec.tagList.map((t) => t?.tagName).filter(Boolean).join('、')
      : ''
    return {
      platformServiceId: id,
      orderNo: String(os.order_id || oi.orderId || ''),
      productTitle: String(oi.orderTitle || ''),
      productImage: String(oi.img || ''),
      productContent: String(oi.orderContent || ''),
      buyerNick: String(ui.userName || ''),
      createSource: String(rec.createSourceDesc || ''),
      businessType: String(rec.taskOrderBusinessTypeDesc || ''),
      orderType: String(rec.mixTaskOrderTypeDesc || ''),
      tags,
      status: String(os.service_type_desc || ''),
      timeoutText: '',
      delayEndTime: Number(os.delay_end_time || 0),
      delayTimeLeft: Number(os.delay_time_left || 0),
      detail: String(os.detail || ''),
      solution: String(os.deal_suggest || ''),
      lastLog: String(log.lastLogContent || ''),
      lastLogTime: String(log.lastLogTime || ''),
      createTime: String(rec.createTimeDesc || ''),
      rawJson: JSON.stringify({
        service_id: id,
        delay_end_time: os.delay_end_time,
        delay_time_left: os.delay_time_left,
        service_status: os.service_status,
        service_type: os.service_type,
      }),
    }
  }
  const collectRows = () => {
    const rows = []
    document.querySelectorAll('table tbody tr').forEach((tr) => {
      const rec = serialize(recordOf(tr))
      if (!rec) return
      const cells = Array.from(tr.children).map((td) => String(td.innerText || '').trim())
      if (cells.length >= 5) {
        const progressLines = String(cells[4] || '').split('\n').map((s) => s.trim()).filter(Boolean)
        if (!rec.status) rec.status = progressLines[0] || rec.status
        if (!rec.timeoutText) rec.timeoutText = progressLines.slice(1).join(' ')
        if ((!rec.solution || rec.solution === '_') && cells[5] && cells[5] !== '_') rec.solution = cells[5]
      }
      rows.push(rec)
    })
    return rows
  }
  document.addEventListener('osms-service-need-rows', () => {
    document.dispatchEvent(new CustomEvent('osms-service-rows', { detail: collectRows() }))
  })
  return true
}

async function installServiceExtractor(tabId) {
  try {
    await chrome.scripting.executeScript({
      target: { tabId, allFrames: true },
      world: 'MAIN',
      func: installPageServiceExtractor,
    })
  } catch (e) {
    console.warn('install service extractor failed', e)
  }
}

async function collectFromTab(tabId) {
  logStep('collect.workbench.start', { tabId })
  await injectWorkbench(tabId)
  await installLogisticsExtractor(tabId)
  for (let i = 0; i < 20; i++) {
    try {
      const ping = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_PING_WORKBENCH' })
      logStep('collect.workbench.ping', { try: i, ping })
      if (!ping?.ready) throw new Error('工作台卡片未就绪')
      const res = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_COLLECT' })
      logStep('collect.workbench.result', {
        try: i,
        ok: !!res?.ok,
        error: res?.error,
        cardCount: res?.cards?.length,
        ticketCount: res?.tickets?.length,
        shop: { id: res?.platformShopId, name: res?.platformShopName },
      })
      if (res?.ok) return res
      throw new Error(res?.error || '采集失败')
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      if (i === 0 || i === 19) logStep('collect.workbench.retry', { try: i, error: msg }, 'warn')
      if (i === 19) {
        await probeTab(tabId, 'collect.workbench.fail.probe')
        if (msg.includes('Receiving end')) {
          throw new Error('未进入售后工作台：页面脚本未就绪，请刷新当前抖店标签后再同步')
        }
        throw new Error(msg)
      }
      await injectWorkbench(tabId)
      await sleep(500)
    }
  }
  throw new Error('采集失败')
}

async function injectMenu(tabId) {
  await chrome.scripting.executeScript({
    target: { tabId },
    files: ['content/menu.js'],
  })
}

async function injectWorkbench(tabId) {
  await chrome.scripting.executeScript({
    target: { tabId },
    files: ['content/workbench.js'],
  })
}
async function clickDoudianMenu(tabId, target, force = false) {
  logStep('menu.click', { tabId, target, force })
  await probeTab(tabId, `menu.probe.before.${target}`)
  for (let i = 0; i < 10; i++) {
    try {
      await injectMenu(tabId)
      const res = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_CLICK_MENU', target, force })
      logStep('menu.click.result', { try: i, target, res })
      if (res?.ok) {
        await probeTab(tabId, `menu.probe.after.${target}`)
        return res
      }
      throw new Error(res?.error || '未找到左侧菜单')
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      if (i === 9) {
        await probeTab(tabId, `menu.fail.probe.${target}`)
        throw new Error(msg.includes('Receiving end') ? '请刷新抖店页面后再同步' : msg)
      }
      await sleep(500)
    }
  }
  throw new Error('菜单点击失败')
}

async function pingWorkbenchReady(tabId) {
  try {
    await injectWorkbench(tabId)
    const ping = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_PING_WORKBENCH' })
    logStep('workbench.ping', ping)
    return !!ping?.ready
  } catch (e) {
    logStep('workbench.ping.error', { error: e instanceof Error ? e.message : String(e) }, 'warn')
    return false
  }
}

async function waitWorkbenchReady(tabId, tries = 75) {
  for (let i = 0; i < tries; i++) {
    if (await pingWorkbenchReady(tabId)) return
    await sleep(400)
  }
  await probeTab(tabId, 'workbench.ready.timeout.probe')
  throw new Error('售后工作台未就绪，请确认左侧菜单可点击「售后 → 售后工作台」')
}

async function prepareWorkbench(tabId) {
  logStep('prepare.workbench', { tabId })
  try {
    await injectMenu(tabId)
  } catch (e) {
    if (currentRun) currentRun.err('inject.menu', e)
  }
  if (await pingWorkbenchReady(tabId)) {
    logStep('prepare.workbench.alreadyReady', { tabId })
    return
  }
  await clickDoudianMenu(tabId, 'workbench')
  await waitWorkbenchReady(tabId)
}

async function openWorkbenchTab() {
  if (opening) return { ok: false, error: '正在打开抖店工作台' }
  const run = startRun('open')
  opening = true
  try {
    let tab = await findFxgTab()
    logStep('open.findTab', tab ? { id: tab.id, url: tab.url, discarded: tab.discarded, active: tab.active } : null)
    if (tab?.id) {
      tab = await ensureTabAwake(tab)
      await chrome.tabs.update(tab.id, { active: true })
      try {
        await chrome.windows.update(tab.windowId, { focused: true })
      } catch (e) {
        logStep('open.focusWindow', { error: e instanceof Error ? e.message : String(e) }, 'warn')
      }
    } else {
      logStep('open.createTab', { url: WORKBENCH_URL })
      tab = await chrome.tabs.create({ url: WORKBENCH_URL, active: true })
      tab = await waitTabComplete(tab.id)
    }
    await rememberFxgTab(tab.id)
    await probeTab(tab.id, 'open.probe.afterFocus')
    await prepareWorkbench(tab.id)
    await uploadRun(run, { ok: true, meta: { tabId: tab.id, url: tab.url } })
    return { ok: true }
  } catch (e) {
    if (currentRun) currentRun.err('open.fail', e)
    const msg = e instanceof Error ? e.message : String(e)
    await uploadRun(run, { ok: false, error: msg })
    return { ok: false, error: msg }
  } finally {
    opening = false
  }
}

async function injectServiceContent(tabId) {
  try {
    await chrome.scripting.executeScript({
      target: { tabId },
      files: ['content/service-order.js'],
    })
  } catch (e) {
    if (currentRun) currentRun.err('inject.service', e)
  }
  await installServiceExtractor(tabId)
}

async function waitServiceReady(tabId, tries = 75) {
  for (let i = 0; i < tries; i++) {
    try {
      await injectServiceContent(tabId)
      const ping = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_PING_SERVICE' })
      if (i === 0 || ping?.ready) logStep('service.ping', { try: i, ping })
      if (ping?.ready) return true
    } catch (e) {
      if (i === 0) logStep('service.ping.error', { try: i, error: e instanceof Error ? e.message : String(e) }, 'warn')
    }
    await sleep(400)
  }
  return false
}

async function collectServiceOrders(tabId) {
  logStep('service.start', { tabId })
  try {
    try {
      await clickDoudianMenu(tabId, 'service', true)
    } catch (e) {
      if (currentRun) currentRun.err('service.menu', e)
    }
    let ready = await waitServiceReady(tabId, 50)
    logStep('service.ready.afterMenu', { ready })
    if (!ready) {
      const tab = await chrome.tabs.get(tabId)
      logStep('service.fallback.nav', { url: tab.url })
      if (!String(tab.url || '').includes('/ffa/task-order/service')) {
        await chrome.tabs.update(tabId, { url: SERVICE_ORDER_URL })
        await sleep(2500)
      }
      ready = await waitServiceReady(tabId, 60)
      logStep('service.ready.afterNav', { ready })
    }
    if (!ready) {
      await probeTab(tabId, 'service.notReady.probe')
      throw new Error('服务工单页面未就绪，请确认可点击左侧菜单「售后 → 服务工单」')
    }
    await injectServiceContent(tabId)
    for (let i = 0; i < 8; i++) {
      try {
        const res = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_COLLECT_SERVICE' })
        logStep('service.collect', {
          try: i,
          ok: !!res?.ok,
          error: res?.error,
          tabCount: res?.tabs?.length,
          orderCount: res?.orders?.length,
          tabs: res?.tabs,
        })
        if (res?.ok) return res
        throw new Error(res?.error || '服务工单采集失败')
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e)
        if (i === 7) {
          await probeTab(tabId, 'service.collect.fail.probe')
          throw new Error(msg)
        }
        await injectServiceContent(tabId)
        await sleep(500)
      }
    }
    throw new Error('服务工单采集失败')
  } finally {
    try {
      await clickDoudianMenu(tabId, 'workbench', true)
      await waitWorkbenchReady(tabId)
    } catch (e) {
      if (currentRun) currentRun.err('service.returnWorkbench', e)
    }
  }
}

async function syncNow() {
  if (syncing) return { ok: false, error: '正在同步' }
  const run = startRun('sync')
  const device = await getDevice()
  if (!device) return { ok: false, error: '尚未绑定店铺' }
  let tab = await findFxgTab()
  logStep('sync.findTab', tab ? { id: tab.id, url: tab.url, discarded: tab.discarded, active: tab.active } : null)
  if (!tab?.id) {
    await uploadRun(run, { ok: false, error: '请先打开抖店后台（点「打开抖店工作台」即可）' })
    return { ok: false, error: '请先打开抖店后台（点「打开抖店工作台」即可）' }
  }
  tab = await ensureTabAwake(tab)
  await rememberFxgTab(tab.id)
  syncing = true
  const keepAlive = setInterval(() => {
    chrome.runtime.getPlatformInfo(() => {})
  }, 15000)
  try {
    await probeTab(tab.id, 'sync.probe.start')
    await prepareWorkbench(tab.id)
    const collected = await collectFromTab(tab.id)
    let serviceOrders = []
    let serviceError = ''
    try {
      const service = await collectServiceOrders(tab.id)
      serviceOrders = service.orders || []
    } catch (e) {
      serviceError = e instanceof Error ? e.message : String(e)
      if (currentRun) currentRun.err('sync.service', e)
    }
    const payload = {
      platformShopId: collected.platformShopId || '',
      platformShopName: collected.platformShopName || '',
      cards: collected.cards || [],
      tickets: collected.tickets || [],
    }
    if (!serviceError) payload.serviceOrders = serviceOrders
    logStep('sync.upload', {
      cards: payload.cards.length,
      tickets: payload.tickets.length,
      serviceOrders: serviceOrders.length,
      serviceError,
    })
    const data = await api('/plugin/sync', { auth: true, body: payload })
    const patch = { ...device }
    if (payload.platformShopId) patch.platformShopId = payload.platformShopId
    if (payload.platformShopName) patch.platformShopName = payload.platformShopName
    await setDevice(patch)
    const now = new Date().toLocaleString()
    await chrome.storage.local.set({
      [STORAGE.lastSync]: now,
      [STORAGE.lastError]: serviceError,
      [STORAGE.lastSyncAt]: Date.now(),
    })
    const out = {
      ok: true,
      cardCount: data?.cardCount ?? payload.cards.length,
      ticketCount: data?.ticketCount ?? payload.tickets.length,
      serviceOrderCount: data?.serviceOrderCount ?? serviceOrders.length,
      serviceError,
      lastSyncAt: data?.lastSyncAt || now,
    }
    await uploadRun(run, {
      ok: !serviceError,
      error: serviceError,
      meta: { cardCount: out.cardCount, ticketCount: out.ticketCount, serviceOrderCount: out.serviceOrderCount },
    })
    return out
  } catch (e) {
    if (currentRun) currentRun.err('sync.fail', e)
    const msg = e instanceof Error ? e.message : String(e)
    await chrome.storage.local.set({ [STORAGE.lastError]: msg })
    await uploadRun(run, { ok: false, error: msg })
    return { ok: false, error: msg }
  } finally {
    clearInterval(keepAlive)
    syncing = false
  }
}

async function bind(bindCode, apiBase) {
  if (apiBase) await setApiBase(apiBase)
  const code = String(bindCode || '').trim().toUpperCase()
  if (code.length < 4) throw new Error('请输入绑定码')
  const data = await api('/plugin/bind', { body: { bindCode: code } })
  await setDevice({
    shopId: data.shopId,
    shopName: data.shopName,
    platform: data.platform,
    pluginKey: data.pluginKey,
    pluginSecret: data.pluginSecret,
  })
  return data
}

chrome.tabs.onUpdated.addListener((tabId, info, tab) => {
  if (info.status !== 'complete') return
  if (!isFxgUsable(tab)) return
  rememberFxgTab(tabId)
})

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  const reply = (p) => sendResponse(p)
  ;(async () => {
    if (msg?.type === 'AFTERSALE_GET_STATUS') {
      const [device, apiBase, extra] = await Promise.all([
        getDevice(),
        getApiBase(),
        chrome.storage.local.get([STORAGE.lastSync, STORAGE.lastError, 'lastDiagRunId', 'lastDiagUpload']),
      ])
      reply({
        ok: true,
        diagnostic: true,
        version: chrome.runtime.getManifest().version,
        bound: !!device,
        shopName: device?.shopName || '',
        platform: device?.platform || '',
        online: !!device,
        heartbeatError: '',
        lastSync: extra[STORAGE.lastSync] || '',
        lastSyncError: extra[STORAGE.lastError] || '',
        lastDiagRunId: extra.lastDiagRunId || '',
        lastDiagUpload: extra.lastDiagUpload || '',
        apiBase,
        apiBaseDisplay: displayApiBase(apiBase),
      })
      return
    }
    if (msg?.type === 'AFTERSALE_SAVE_API') {
      await setApiBase(msg.apiBase)
      reply({ ok: true })
      return
    }
    if (msg?.type === 'AFTERSALE_BIND') {
      try {
        const data = await bind(msg.bindCode, msg.apiBase)
        reply({ ok: true, shopName: data.shopName })
      } catch (e) {
        reply({ ok: false, error: e instanceof Error ? e.message : String(e) })
      }
      return
    }
    if (msg?.type === 'AFTERSALE_UNBIND') {
      await setDevice(null)
      reply({ ok: true })
      return
    }
    if (msg?.type === 'AFTERSALE_SYNC_NOW') {
      reply(await syncNow())
      return
    }
    if (msg?.type === 'AFTERSALE_OPEN_WORKBENCH') {
      reply(await openWorkbenchTab())
      return
    }
    if (msg?.type === 'AFTERSALE_WORKBENCH_READY') {
      reply({ ok: true })
    }
  })()
  return true
})
