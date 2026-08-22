/** OSMS 抖店售后工作台 — 绑定、心跳、采集上报 */
const STORAGE = {
  device: 'aftersalePlugin',
  apiBase: 'aftersaleApiBase',
  lastSync: 'aftersaleLastSync',
  lastError: 'aftersaleLastError',
  lastSyncAt: 'aftersaleLastSyncAt',
}

const DEFAULT_API_BASE = 'https://osms.zfcycle.com/apps/aftersales/api/v1'
const HEARTBEAT_ALARM = 'aftersale-heartbeat'
const AUTO_SYNC_ALARM = 'aftersale-autosync'
const AUTO_SYNC_MINUTES = 5

const WORKBENCH_URL = 'https://fxg.jinritemai.com/ffa/merchant-aftersale-workbench/aftersale/list'
const SERVICE_ORDER_URL = 'https://fxg.jinritemai.com/ffa/task-order/service'

let syncing = false
let opening = false

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}

function topFrame(tabId) {
  return { tabId, frameIds: [0] }
}

async function injectFiles(tabId, files) {
  try {
    await chrome.scripting.executeScript({
      target: topFrame(tabId),
      files,
    })
  } catch {
    await chrome.scripting.executeScript({
      target: { tabId },
      files,
    })
  }
}

async function sendTop(tabId, msg) {
  return chrome.tabs.sendMessage(tabId, msg, { frameId: 0 })
}

function inspectServiceTabsFn() {
  const textOf = (el) => String(el?.innerText || el?.textContent || '').replace(/\s+/g, ' ').trim()
  const texts = []
  const walk = (doc) => {
    if (!doc) return
    for (const el of doc.querySelectorAll('.auxo-tabs-tab, [role="tab"]')) {
      const t = textOf(el)
      if (t) texts.push(t)
    }
    for (const iframe of doc.querySelectorAll('iframe')) {
      try {
        walk(iframe.contentDocument)
      } catch {
        /* cross-origin */
      }
    }
  }
  walk(document)
  const ready =
    (texts.some((t) => /待处理/.test(t)) &&
      texts.some((t) => /处理中|已逾期|已完结/.test(t))) ||
    (/task-order\/service/.test(String(location.href || '')) &&
      texts.some((t) => /全部工单|待处理|已逾期/.test(t)))
  return { ready, href: String(location.href || ''), tabs: texts.slice(0, 40) }
}

async function pageHasServiceTabs(tabId) {
  try {
    const results = await chrome.scripting.executeScript({
      target: { tabId, allFrames: true },
      func: inspectServiceTabsFn,
    })
    return (results || []).map((r) => r.result).find((r) => r?.ready) || { ready: false }
  } catch {
    return { ready: false }
  }
}

async function collectPendingInPage() {
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
  const textOf = (el) => String(el?.innerText || el?.textContent || '').replace(/\s+/g, ' ').trim()
  const chips = () =>
    [...document.querySelectorAll('div, span, button')].filter(
      (el) => /^(今天|近7天|近30天|近90天)$/.test(textOf(el)) && el.children.length <= 2,
    )
  const selectedChip = () => {
    const cs = chips()
    if (!cs.length) return ''
    const min = Math.min(...cs.map((c) => String(c.className || '').split(/\s+/).filter(Boolean).length))
    const sel = cs.find((c) => String(c.className || '').split(/\s+/).filter(Boolean).length > min)
    return sel ? textOf(sel) : ''
  }
  const statusTabs = () => {
    const out = []
    const keys = new Set()
    for (const el of document.querySelectorAll('.auxo-tabs-tab, [role="tab"]')) {
      const t = textOf(el)
      if (!t || /工单/.test(t)) continue
      const m = t.match(/^(全部|待处理|处理中|已完结|已逾期)\s*[（(]?\s*(\d+)/)
      if (!m || keys.has(m[1])) continue
      keys.add(m[1])
      out.push({
        key: m[1],
        count: Number(m[2]),
        active: /active/.test(String(el.className || '')) || el.getAttribute('aria-selected') === 'true',
        el,
      })
    }
    return out
  }
  const parseRows = () => {
    const table = document.querySelector('table')
    if (!table) return []
    const rows = []
    for (const tr of table.querySelectorAll('tbody tr')) {
      const cells = [...tr.children].map((td) => String(td.innerText || '').trim())
      if (cells.length < 5) continue
      const id = (String(cells[2] || '').match(/工单ID[:：]?\s*(\d+)/) || [])[1] || ''
      if (!id) continue
      const orderText = cells[0] || ''
      const progressLines = String(cells[4] || '').split('\n').map((s) => s.trim()).filter(Boolean)
      const lastLines = String(cells[6] || '').split('\n').map((s) => s.trim()).filter(Boolean)
      rows.push({
        platformServiceId: id,
        orderNo: (orderText.match(/(\d{15,})/) || [])[1] || '',
        productTitle:
          orderText.split('\n').filter((l) => l && !/^\d{15,}$/.test(l) && l !== (orderText.match(/(\d{15,})/) || [])[1])[1] ||
          '',
        productImage: '',
        productContent: (orderText.match(/总价.*/) || [])[0] || '',
        buyerNick: cells[1] || '',
        createSource: String(cells[2] || '').split('\n')[0] || '',
        businessType: String(cells[3] || '').split('\n')[0] || '',
        orderType: String(cells[3] || '').split('\n')[1] || '',
        tags: orderText.split('\n').filter((l) => /进线|催|紧急|重复/.test(l)).join('、'),
        status: progressLines[0] || '',
        timeoutText: progressLines.slice(1).join(' '),
        delayEndTime: 0,
        delayTimeLeft: 0,
        detail: '',
        solution: cells[5] && cells[5] !== '_' ? cells[5] : '',
        lastLog: lastLines[0] || '',
        lastLogTime: lastLines[1] || '',
        createTime: lastLines[1] || '',
      })
    }
    return rows
  }
  if (!statusTabs().some((t) => t.key === '待处理')) {
    return { ok: false, missing: true }
  }
  if (selectedChip() && selectedChip() !== '近30天') {
    const el = chips().find((n) => textOf(n) === '近30天')
    if (el) {
      el.click()
      for (let i = 0; i < 20; i++) {
        await sleep(150)
        if (selectedChip() === '近30天') break
      }
      await sleep(400)
    }
  }
  const collectKey = async (key) => {
    const tab = statusTabs().find((t) => t.key === key)
    if (!tab || !tab.count) return []
    if (!tab.active) {
      tab.el.click()
      for (let i = 0; i < 20; i++) {
        await sleep(150)
        if (statusTabs().find((x) => x.key === key)?.active) break
      }
      await sleep(400)
    }
    return parseRows().map((r) => ({ ...r, statusTab: key }))
  }
  const pending = await collectKey('待处理')
  return JSON.parse(
    JSON.stringify({
      ok: true,
      tabs: statusTabs().map(({ el, ...rest }) => rest),
      orders: pending,
    }),
  )
}

async function runInAllFrames(tabId, func) {
  const results = await chrome.scripting.executeScript({
    target: { tabId, allFrames: true },
    func,
  })
  const values = (results || []).map((r) => r.result).filter(Boolean)
  return values.find((r) => r.ok) || values.find((r) => r.error) || values[0] || null
}

async function callPage(tabId, fnName, fnArgs = []) {
  const [inj] = await chrome.scripting.executeScript({
    target: topFrame(tabId),
    func: async (name, args) => {
      const fn = window[name]
      if (typeof fn !== 'function') return { ok: false, missing: name }
      try {
        const data = await fn(...args)
        return JSON.parse(JSON.stringify(data == null ? { ok: true } : data))
      } catch (e) {
        return { ok: false, error: e instanceof Error ? e.message : String(e) }
      }
    },
    args: [fnName, fnArgs],
  })
  return inj?.result
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

async function heartbeat() {
  const device = await getDevice()
  if (!device) return { ok: false, skipped: true }
  try {
    const data = await api('/plugin/heartbeat', {
      auth: true,
      body: {
        platformShopId: device.platformShopId || '',
        platformShopName: device.platformShopName || '',
      },
    })
    await chrome.storage.local.set({ heartbeatError: '' })
    return { ok: true, data }
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    await chrome.storage.local.set({ heartbeatError: msg })
    return { ok: false, error: msg }
  }
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
  await injectWorkbench(tabId)
  await installLogisticsExtractor(tabId)
  for (let i = 0; i < 20; i++) {
    try {
      const ping = await callPage(tabId, '__osmsWorkbenchReady')
      let ready = ping === true
      if (ping?.missing) {
        const msgPing = await sendTop(tabId, { type: 'AFTERSALE_PING_WORKBENCH' })
        ready = !!msgPing?.ready
      }
      if (!ready) throw new Error(ping?.error || '工作台卡片未就绪')
      const res = await callPage(tabId, '__osmsCollectWorkbench')
      if (res?.missing) {
        const viaMsg = await sendTop(tabId, { type: 'AFTERSALE_COLLECT' })
        if (viaMsg?.ok) return viaMsg
        throw new Error(viaMsg?.error || '采集失败')
      }
      if (res?.ok) return res
      throw new Error(res?.error || '采集失败')
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      if (i === 19) {
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
  await injectFiles(tabId, ['content/menu.js'])
}

async function injectWorkbench(tabId) {
  await injectFiles(tabId, ['content/workbench.js'])
}

async function clickDoudianMenu(tabId, target, force = false) {
  for (let i = 0; i < 10; i++) {
    try {
      await injectMenu(tabId)
      const viaFn = await callPage(tabId, '__osmsClickMenu', [target, force])
      if (viaFn && !viaFn.missing) {
        if (viaFn.ok) return viaFn
        throw new Error(viaFn.error || '未找到左侧菜单')
      }
      const res = await sendTop(tabId, { type: 'AFTERSALE_CLICK_MENU', target, force })
      if (res?.ok) return res
      throw new Error(res?.error || '未找到左侧菜单')
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      if (i === 9) throw new Error(msg.includes('Receiving end') ? '请刷新抖店页面后再同步' : msg)
      await sleep(500)
    }
  }
  throw new Error('菜单点击失败')
}

async function pingWorkbenchReady(tabId) {
  try {
    await injectWorkbench(tabId)
    const ping = await callPage(tabId, '__osmsWorkbenchReady')
    if (ping === true) return true
    if (ping?.missing) {
      const msgPing = await sendTop(tabId, { type: 'AFTERSALE_PING_WORKBENCH' })
      return !!msgPing?.ready
    }
    return false
  } catch {
    return false
  }
}

async function waitWorkbenchReady(tabId, tries = 75) {
  for (let i = 0; i < tries; i++) {
    if (await pingWorkbenchReady(tabId)) return
    await sleep(400)
  }
  throw new Error('售后工作台未就绪，请确认左侧菜单可点击「售后 → 售后工作台」')
}

async function prepareWorkbench(tabId) {
  try {
    await injectMenu(tabId)
  } catch (e) {
    console.warn('inject menu.js failed', e)
  }
  if (await pingWorkbenchReady(tabId)) return
  await clickDoudianMenu(tabId, 'workbench')
  await waitWorkbenchReady(tabId)
}

async function openWorkbenchTab() {
  if (opening) return { ok: false, error: '正在打开抖店工作台' }
  opening = true
  try {
    let tab = await findFxgTab()
    if (tab?.id) {
      tab = await ensureTabAwake(tab)
      await chrome.tabs.update(tab.id, { active: true })
      try {
        await chrome.windows.update(tab.windowId, { focused: true })
      } catch {
        /* ignore */
      }
    } else {
      tab = await chrome.tabs.create({ url: WORKBENCH_URL, active: true })
      tab = await waitTabComplete(tab.id)
    }
    await rememberFxgTab(tab.id)
    await prepareWorkbench(tab.id)
    return { ok: true }
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) }
  } finally {
    opening = false
  }
}

async function waitServiceReady(tabId, tries = 40) {
  for (let i = 0; i < tries; i++) {
    try {
      if ((await pageHasServiceTabs(tabId)).ready) return true
    } catch {
      /* 切页过程中探测可能失败 */
    }
    await sleep(400)
  }
  return false
}

async function collectServiceOrders(tabId, { skipReturn } = {}) {
  try {
    const already = await pageHasServiceTabs(tabId)
    if (!already.ready) {
      try {
        await clickDoudianMenu(tabId, 'service')
      } catch (e) {
        console.warn('click service menu failed', e)
      }
    }
    let ready = await waitServiceReady(tabId, 40)
    if (!ready) {
      const tab = await chrome.tabs.get(tabId)
      if (!String(tab.url || '').includes('/ffa/task-order/service')) {
        await chrome.tabs.update(tabId, { url: SERVICE_ORDER_URL })
        await sleep(2500)
      }
      ready = await waitServiceReady(tabId, 40)
    }
    if (!ready) {
      throw new Error('服务工单页面未就绪，请打开左侧「售后 → 服务工单」')
    }
    const res = await runInAllFrames(tabId, collectPendingInPage)
    if (res?.ok) return res
    throw new Error(res?.error || '未读到待处理列表')
  } finally {
    if (!skipReturn) {
      try {
        await clickDoudianMenu(tabId, 'workbench')
        await waitWorkbenchReady(tabId)
      } catch (e) {
        console.warn('return to workbench failed', e)
      }
    }
  }
}

async function syncNow() {
  if (syncing) return { ok: false, error: '正在同步' }
  const device = await getDevice()
  if (!device) return { ok: false, error: '尚未绑定店铺' }
  let tab = await findFxgTab()
  if (!tab?.id) return { ok: false, error: '请先打开抖店后台（点「打开抖店工作台」即可）' }
  tab = await ensureTabAwake(tab)
  await rememberFxgTab(tab.id)
  syncing = true
  const keepAlive = setInterval(() => {
    chrome.runtime.getPlatformInfo(() => {})
  }, 15000)
  try {
    let serviceOrders = []
    let serviceError = ''
    const alreadyService = await pageHasServiceTabs(tab.id)
    if (alreadyService.ready) {
      try {
        const service = await collectServiceOrders(tab.id, { skipReturn: true })
        serviceOrders = service.orders || []
      } catch (e) {
        serviceError = e instanceof Error ? e.message : String(e)
      }
    }
    await prepareWorkbench(tab.id)
    const collected = await collectFromTab(tab.id)
    if (!alreadyService.ready || serviceError) {
      try {
        const service = await collectServiceOrders(tab.id)
        serviceOrders = service.orders || []
        serviceError = ''
      } catch (e) {
        serviceError = e instanceof Error ? e.message : String(e)
      }
    }
    const payload = {
      platformShopId: collected.platformShopId || '',
      platformShopName: collected.platformShopName || '',
      cards: collected.cards || [],
      tickets: collected.tickets || [],
    }
    if (!serviceError) payload.serviceOrders = serviceOrders
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
    return {
      ok: true,
      cardCount: data?.cardCount ?? payload.cards.length,
      ticketCount: data?.ticketCount ?? payload.tickets.length,
      serviceOrderCount: data?.serviceOrderCount ?? serviceOrders.length,
      serviceError,
      lastSyncAt: data?.lastSyncAt || now,
    }
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    await chrome.storage.local.set({ [STORAGE.lastError]: msg })
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
  await chrome.alarms.create(HEARTBEAT_ALARM, { periodInMinutes: 1 })
  await chrome.alarms.create(AUTO_SYNC_ALARM, { periodInMinutes: AUTO_SYNC_MINUTES })
  await heartbeat()
  return data
}

async function ensureAlarms() {
  const existing = await chrome.alarms.getAll()
  const names = new Set(existing.map((a) => a.name))
  if (!names.has(HEARTBEAT_ALARM)) {
    await chrome.alarms.create(HEARTBEAT_ALARM, { periodInMinutes: 1 })
  }
  if (!names.has(AUTO_SYNC_ALARM)) {
    await chrome.alarms.create(AUTO_SYNC_ALARM, { periodInMinutes: AUTO_SYNC_MINUTES })
  }
}

chrome.runtime.onInstalled.addListener(() => {
  ensureAlarms()
})

chrome.runtime.onStartup.addListener(() => {
  ensureAlarms()
})

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === HEARTBEAT_ALARM) heartbeat()
  if (alarm.name === AUTO_SYNC_ALARM) maybeAutoSync(AUTO_SYNC_MINUTES * 60 * 1000 - 30000)
})

async function maybeAutoSync(minIntervalMs = 60000) {
  const device = await getDevice()
  if (!device || syncing || opening) return
  const tab = await findFxgTab()
  if (!tab?.id) return
  const extra = await chrome.storage.local.get([STORAGE.lastSyncAt])
  const last = Number(extra[STORAGE.lastSyncAt] || 0)
  if (Date.now() - last < minIntervalMs) return
  return syncNow()
}

chrome.tabs.onUpdated.addListener((tabId, info, tab) => {
  if (info.status !== 'complete') return
  if (!isFxgUsable(tab)) return
  rememberFxgTab(tabId)
  setTimeout(() => maybeAutoSync(60000), 8000)
})

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  const reply = (p) => sendResponse(p)
  ;(async () => {
    if (msg?.type === 'AFTERSALE_GET_STATUS') {
      const [device, apiBase, extra] = await Promise.all([
        getDevice(),
        getApiBase(),
        chrome.storage.local.get(['heartbeatError', STORAGE.lastSync, STORAGE.lastError]),
      ])
      if (device) await ensureAlarms()
      reply({
        ok: true,
        version: chrome.runtime.getManifest().version,
        bound: !!device,
        shopName: device?.shopName || '',
        platform: device?.platform || '',
        online: !!device && !extra.heartbeatError,
        heartbeatError: extra.heartbeatError || '',
        lastSync: extra[STORAGE.lastSync] || '',
        lastSyncError: extra[STORAGE.lastError] || '',
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
      await chrome.alarms.clear(HEARTBEAT_ALARM)
      await chrome.alarms.clear(AUTO_SYNC_ALARM)
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
      maybeAutoSync(60000)
      reply({ ok: true })
    }
  })()
  return true
})
