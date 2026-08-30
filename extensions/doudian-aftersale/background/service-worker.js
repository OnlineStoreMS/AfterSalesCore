/** OSMS 抖店售后工作台 — 绑定、心跳、采集上报 */
const STORAGE = {
  device: 'aftersalePlugin',
  apiBase: 'aftersaleApiBase',
  lastSync: 'aftersaleLastSync',
  lastError: 'aftersaleLastError',
  lastSyncAt: 'aftersaleLastSyncAt',
  workLogs: 'aftersaleWorkLogs',
}

const DEFAULT_API_BASE = 'https://osms.zfcycle.com/apps/aftersales/api/v1'
const HEARTBEAT_ALARM = 'aftersale-heartbeat'
const AUTO_SYNC_ALARM = 'aftersale-autosync'

const WORKBENCH_URL = 'https://fxg.jinritemai.com/ffa/merchant-aftersale-workbench/aftersale/list'

let syncing = false
let opening = false
let lastHeartbeatError = ''
let lastHeartbeatErrorAt = 0

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}

function topFrame(tabId) {
  return { tabId, frameIds: [0] }
}

async function injectFiles(tabId, files, world) {
  const opts = { target: topFrame(tabId), files }
  if (world) opts.world = world
  try {
    await chrome.scripting.executeScript(opts)
  } catch {
    const fallback = { target: { tabId }, files }
    if (world) fallback.world = world
    await chrome.scripting.executeScript(fallback)
  }
}

async function sendTop(tabId, msg) {
  return chrome.tabs.sendMessage(tabId, msg, { frameId: 0 })
}

async function callPage(tabId, fnName, fnArgs = [], world) {
  const opts = {
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
  }
  if (world) opts.world = world
  const [inj] = await chrome.scripting.executeScript(opts)
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

async function readWorkLogs() {
  const data = await chrome.storage.local.get([STORAGE.workLogs])
  return Array.isArray(data[STORAGE.workLogs]) ? data[STORAGE.workLogs] : []
}

async function workSnapshot() {
  const [device, extra, logs] = await Promise.all([
    getDevice(),
    chrome.storage.local.get(['heartbeatError', STORAGE.lastSync, STORAGE.lastError]),
    readWorkLogs(),
  ])
  return {
    bound: !!device,
    shopName: device?.shopName || '',
    online: !!device && !extra.heartbeatError,
    syncing,
    lastSync: extra[STORAGE.lastSync] || '',
    lastSyncError: extra[STORAGE.lastError] || '',
    logs,
  }
}

async function broadcastWork(extra = {}) {
  const snap = await workSnapshot()
  const msg = { type: 'AFTERSALE_WORK_LOG', ...snap, ...extra }
  const tabs = await chrome.tabs.query({ url: ['https://fxg.jinritemai.com/*'] })
  for (const tab of tabs) {
    if (!tab.id) continue
    try {
      await chrome.tabs.sendMessage(tab.id, msg)
    } catch {
      /* 页面尚未注入面板 */
    }
  }
}

async function workLog(msg, level = 'info') {
  const line = { t: Date.now(), level, msg: String(msg || '') }
  const logs = [...(await readWorkLogs()).slice(-39), line]
  await chrome.storage.local.set({ [STORAGE.workLogs]: logs })
  await broadcastWork({ logs })
}

async function injectPanel(tabId) {
  try {
    await injectFiles(tabId, ['content/panel.js'])
  } catch (e) {
    console.warn('inject panel failed', e)
  }
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
    if (lastHeartbeatError) {
      lastHeartbeatError = ''
      await workLog('心跳已恢复', 'ok')
    }
    if (data?.syncNow && !syncing) {
      await workLog('服务端请求同步')
      await syncNow()
    }
    return { ok: true, data }
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    await chrome.storage.local.set({ heartbeatError: msg })
    if (msg !== lastHeartbeatError || Date.now() - lastHeartbeatErrorAt > 5 * 60 * 1000) {
      lastHeartbeatError = msg
      lastHeartbeatErrorAt = Date.now()
      await workLog(`心跳失败：${msg}`, 'error')
    }
    return { ok: false, error: msg }
  }
}

function installPageLogisticsExtractor() {
  if (window.__osmsAftersaleDriverV20) return true
  window.__osmsAftersaleDriverV20 = true
  window.__osmsAftersaleLogisticsInstalled = true
  const fiberOf = (el) => {
    if (!el) return null
    const key = Object.keys(el).find(
      (k) => k.startsWith('__reactFiber') || k.startsWith('__reactInternalInstance'),
    )
    return key ? el[key] : null
  }
  const ui = (name) =>
    document.querySelector(`.aurora-${name}`) || document.querySelector(`.auxo-${name}`)
  const propsOf = (el, pred) => {
    let n = fiberOf(el)
    for (let i = 0; i < 30 && n; i++) {
      const p = n.memoizedProps || n.pendingProps
      if (p && pred(p)) return p
      n = n.return
    }
    return null
  }
  const setPageSize = (size) => {
    const changer = ui('pagination-options-size-changer')
    const pager = ui('pagination')
    const change = propsOf(changer, (p) => typeof p.changeSize === 'function')
    if (change?.changeSize) {
      change.changeSize(size)
      return { ok: true, via: 'changeSize' }
    }
    const pagerP = propsOf(
      pager,
      (p) => typeof p.onChange === 'function' && ('pageSize' in p || 'total' in p),
    )
    if (typeof pagerP?.onShowSizeChange === 'function') {
      pagerP.onShowSizeChange(1, size)
      return { ok: true, via: 'onShowSizeChange' }
    }
    if (typeof pagerP?.onChange === 'function') {
      pagerP.onChange(1, size)
      return { ok: true, via: 'onChange' }
    }
    return { ok: false }
  }
  const goPage = (page, size) => {
    const pagerP = propsOf(
      ui('pagination'),
      (p) => typeof p.onChange === 'function' && ('pageSize' in p || 'total' in p),
    )
    if (typeof pagerP?.onChange === 'function') {
      pagerP.onChange(page, size || pagerP.pageSize || 10)
      return { ok: true, via: 'onChange' }
    }
    return { ok: false }
  }
  document.addEventListener('osms-aftersale-drive', (e) => {
    const d = e.detail || {}
    let result = { ok: false }
    try {
      if (d.action === 'setPageSize') result = setPageSize(Number(d.size) || 100)
      else if (d.action === 'goPage') result = goPage(Number(d.page) || 1, Number(d.size) || 10)
    } catch (err) {
      result = { ok: false, error: String(err) }
    }
    document.dispatchEvent(new CustomEvent('osms-aftersale-drive-done', { detail: result }))
  })
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

async function collectFromTab(tabId) {
  await injectWorkbench(tabId)
  await installLogisticsExtractor(tabId)
  for (let i = 0; i < 20; i++) {
    try {
      const ping = await callPage(tabId, '__osmsWorkbenchReady', [], 'MAIN')
      let ready = ping === true
      if (ping?.missing) {
        const isoPing = await callPage(tabId, '__osmsWorkbenchReady')
        ready = isoPing === true
      }
      if (ping?.missing && !ready) {
        const msgPing = await sendTop(tabId, { type: 'AFTERSALE_PING_WORKBENCH' })
        ready = !!msgPing?.ready
      }
      if (!ready) throw new Error(ping?.error || '工作台卡片未就绪')
      let res = await callPage(tabId, '__osmsCollectWorkbench', [], 'MAIN')
      if (res?.missing) res = await callPage(tabId, '__osmsCollectWorkbench')
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
  try {
    await injectFiles(tabId, ['content/workbench.js'], 'MAIN')
  } catch (e) {
    console.warn('inject workbench MAIN failed', e)
  }
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
    await injectPanel(tabId)
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
    await workLog('正在打开售后工作台')
    await prepareWorkbench(tab.id)
    await workLog('已打开售后工作台', 'ok')
    return { ok: true }
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    await workLog(`打开工作台失败：${msg}`, 'error')
    return { ok: false, error: msg }
  } finally {
    opening = false
  }
}

async function syncNow() {
  if (syncing) return { ok: false, error: '正在同步' }
  const device = await getDevice()
  if (!device) {
    await workLog('尚未绑定店铺', 'error')
    return { ok: false, error: '尚未绑定店铺' }
  }
  let tab = await findFxgTab()
  if (!tab?.id) {
    await workLog('未找到抖店标签，请先打开抖店工作台', 'error')
    return { ok: false, error: '请先打开抖店后台（点「打开抖店工作台」即可）' }
  }
  tab = await ensureTabAwake(tab)
  await rememberFxgTab(tab.id)
  syncing = true
  await broadcastWork()
  const keepAlive = setInterval(() => {
    chrome.runtime.getPlatformInfo(() => {})
  }, 15000)
  try {
    await workLog('开始同步售后工作台')
    await prepareWorkbench(tab.id)
    await workLog('正在采集退回件、卡片与售后单')
    const collected = await collectFromTab(tab.id)
    const payload = {
      platformShopId: collected.platformShopId || '',
      platformShopName: collected.platformShopName || '',
      cards: collected.cards || [],
      tickets: collected.tickets || [],
    }
    if (Array.isArray(collected.returns) && collected.returns.length) payload.returns = collected.returns
    const short = (collected.cardStats || []).filter((s) => s.expected && s.got < s.expected)
    for (const s of short) {
      await workLog(
        `${s.label} 只采到 ${s.got}/${s.expected}` +
          (s.pageSize != null ? `（页大小${s.pageSize} 可见${s.visible}）` : ''),
        'error',
      )
    }
    if (collected.returnStats?.error) {
      await workLog(`退回件采集失败：${collected.returnStats.error}`, 'error')
    } else if (Array.isArray(payload.returns)) {
      const withNo = collected.returnStats?.withNo ?? payload.returns.filter((x) => x.logisticsNo).length
      const st = collected.returnStats || {}
      await workLog(
        `退回件 ${payload.returns.length} 条` +
          (st.filteredTotal != null ? `（已发货退款/退款成功 ${st.filteredTotal}` : '') +
          (st.pages != null ? `，已扫 ${st.pages}/${st.pageCount || st.pages} 页` : '') +
          (st.scanned != null ? `、看过 ${st.scanned} 单` : '') +
          (st.filteredTotal != null ? '）' : '') +
          `，其中 ${withNo} 条已取到单号`,
      )
    }
    await workLog(
      `已采集 ${payload.cards.length} 张卡片、${payload.tickets.length} 条售后单` +
        (Array.isArray(payload.returns) ? `、${payload.returns.length} 条退回件` : '') +
        '，正在上报',
    )
    const data = await api('/plugin/sync', { auth: true, body: payload })
    const patch = { ...device }
    if (payload.platformShopId) patch.platformShopId = payload.platformShopId
    if (payload.platformShopName) patch.platformShopName = payload.platformShopName
    await setDevice(patch)
    const now = new Date().toLocaleString()
    await chrome.storage.local.set({
      [STORAGE.lastSync]: now,
      [STORAGE.lastError]: '',
      [STORAGE.lastSyncAt]: Date.now(),
    })
    const cardCount = data?.cardCount ?? payload.cards.length
    const ticketCount = data?.ticketCount ?? payload.tickets.length
    const returnCount = data?.returnCount ?? payload.returns?.length ?? 0
    await workLog(`同步完成：卡片 ${cardCount} / 售后单 ${ticketCount} / 退回件 ${returnCount}`, 'ok')
    return {
      ok: true,
      cardCount,
      ticketCount,
      returnCount,
      lastSyncAt: data?.lastSyncAt || now,
    }
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    await chrome.storage.local.set({ [STORAGE.lastError]: msg })
    await workLog(`同步失败：${msg}`, 'error')
    return { ok: false, error: msg }
  } finally {
    clearInterval(keepAlive)
    syncing = false
    await broadcastWork()
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
  await chrome.alarms.clear(AUTO_SYNC_ALARM)
  await workLog(`已绑定店铺：${data.shopName || data.shopId}`, 'ok')
  await heartbeat()
  return data
}

async function ensureAlarms() {
  const existing = await chrome.alarms.getAll()
  const names = new Set(existing.map((a) => a.name))
  if (!names.has(HEARTBEAT_ALARM)) {
    await chrome.alarms.create(HEARTBEAT_ALARM, { periodInMinutes: 1 })
  }
  if (names.has(AUTO_SYNC_ALARM)) {
    await chrome.alarms.clear(AUTO_SYNC_ALARM)
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
})

chrome.tabs.onUpdated.addListener((tabId, info, tab) => {
  if (info.status !== 'complete') return
  if (!isFxgUsable(tab)) return
  rememberFxgTab(tabId)
  injectPanel(tabId)
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
      const snap = await workSnapshot()
      reply({
        ok: true,
        version: chrome.runtime.getManifest().version,
        bound: snap.bound,
        shopName: snap.shopName,
        platform: device?.platform || '',
        online: snap.online,
        heartbeatError: extra.heartbeatError || '',
        lastSync: snap.lastSync,
        lastSyncError: snap.lastSyncError,
        syncing: snap.syncing,
        logs: snap.logs,
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
      await workLog('已解除绑定')
      reply({ ok: true })
      return
    }
    if (msg?.type === 'AFTERSALE_CLEAR_WORKLOG') {
      await chrome.storage.local.set({ [STORAGE.workLogs]: [] })
      await workLog('已清除工作记录')
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
