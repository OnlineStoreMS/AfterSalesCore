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
const FXG_MATCHES = [
  'https://fxg.jinritemai.com/ffa/merchant-aftersale-workbench/*',
  'https://fxg.jinritemai.com/ffa/task-order/*',
  'https://fxg.jinritemai.com/ffa/*',
  'https://fxg.jinritemai.com/*',
]

let syncing = false

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
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

async function findFxgTab() {
  const stored = await chrome.storage.local.get(['aftersaleFxgTabId'])
  const savedId = Number(stored.aftersaleFxgTabId || 0)
  if (savedId) {
    try {
      const tab = await chrome.tabs.get(savedId)
      if (tab?.id && tab.url && tab.url.includes('fxg.jinritemai.com') && !/login|passport/i.test(tab.url)) {
        return tab
      }
    } catch {
      /* tab closed */
    }
  }
  for (const url of FXG_MATCHES) {
    const tabs = await chrome.tabs.query({ url: [url] })
    const hit = tabs.find((t) => t.id && t.url && !/login|passport/i.test(t.url || ''))
    if (hit) return hit
  }
  return null
}

async function rememberFxgTab(tabId) {
  await chrome.storage.local.set({ aftersaleFxgTabId: tabId })
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
      target: { tabId },
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
      target: { tabId },
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
  for (let i = 0; i < 8; i++) {
    try {
      const res = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_COLLECT' })
      if (res?.ok) return res
      throw new Error(res?.error || '采集失败')
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      if (i === 7) throw new Error(msg.includes('Receiving end') ? '未进入售后工作台，请保持抖店后台打开' : msg)
      await injectWorkbench(tabId)
      await sleep(400)
    }
  }
  throw new Error('采集失败')
}

async function injectMenu(tabId) {
  try {
    await chrome.scripting.executeScript({
      target: { tabId },
      files: ['content/menu.js'],
    })
  } catch (e) {
    console.warn('inject menu.js failed', e)
  }
}

async function injectWorkbench(tabId) {
  try {
    await chrome.scripting.executeScript({
      target: { tabId },
      files: ['content/workbench.js'],
    })
  } catch (e) {
    console.warn('inject workbench.js failed', e)
  }
}

async function clickDoudianMenu(tabId, target, force = false) {
  await injectMenu(tabId)
  for (let i = 0; i < 6; i++) {
    try {
      const res = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_CLICK_MENU', target, force })
      if (res?.ok) return res
      throw new Error(res?.error || '未找到左侧菜单')
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      if (i === 5) throw new Error(msg.includes('Receiving end') ? '请刷新抖店页面后再同步' : msg)
      await injectMenu(tabId)
      await sleep(400)
    }
  }
  throw new Error('菜单点击失败')
}

async function waitWorkbenchReady(tabId) {
  for (let i = 0; i < 50; i++) {
    try {
      await injectWorkbench(tabId)
      const ping = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_PING_WORKBENCH' })
      if (ping?.ready) return
    } catch {
      /* SPA 切换后脚本可能尚未注入 */
    }
    await sleep(400)
  }
  throw new Error('售后工作台未就绪，请确认左侧菜单可点击「售后 → 售后工作台」')
}

async function prepareWorkbench(tabId) {
  await injectMenu(tabId)
  await injectWorkbench(tabId)
  await clickDoudianMenu(tabId, 'workbench', true)
  await waitWorkbenchReady(tabId)
}

async function injectServiceContent(tabId) {
  try {
    await chrome.scripting.executeScript({
      target: { tabId },
      files: ['content/service-order.js'],
    })
  } catch (e) {
    console.warn('inject service-order.js failed', e)
  }
  await installServiceExtractor(tabId)
}

async function waitServiceReady(tabId) {
  for (let i = 0; i < 50; i++) {
    try {
      await injectServiceContent(tabId)
      const ping = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_PING_SERVICE' })
      if (ping?.ready) return chrome.tabs.get(tabId)
    } catch {
      /* SPA 切换后脚本可能尚未注入 */
    }
    await sleep(400)
  }
  throw new Error('服务工单页面未就绪，请确认可点击左侧菜单「售后 → 服务工单」')
}

async function collectServiceOrders(tabId) {
  await clickDoudianMenu(tabId, 'service')
  try {
    await waitServiceReady(tabId)
    await injectServiceContent(tabId)
    for (let i = 0; i < 8; i++) {
      try {
        const res = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_COLLECT_SERVICE' })
        if (res?.ok) return res
        throw new Error(res?.error || '服务工单采集失败')
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e)
        if (i === 7) throw new Error(msg)
        await sleep(400)
      }
    }
    throw new Error('服务工单采集失败')
  } finally {
    try {
      await clickDoudianMenu(tabId, 'workbench', true)
      await waitWorkbenchReady(tabId)
    } catch (e) {
      console.warn('return to workbench failed', e)
    }
  }
}

async function syncNow() {
  if (syncing) return { ok: false, error: '正在同步' }
  const device = await getDevice()
  if (!device) return { ok: false, error: '尚未绑定店铺' }
  const tab = await findFxgTab()
  if (!tab?.id) return { ok: false, error: '请先打开抖店后台（任意页面即可，插件会自动点售后工作台）' }
  await rememberFxgTab(tab.id)
  syncing = true
  const keepAlive = setInterval(() => {
    chrome.runtime.getPlatformInfo(() => {})
  }, 15000)
  try {
    await prepareWorkbench(tab.id)
    const collected = await collectFromTab(tab.id)
    let serviceOrders = []
    let serviceError = ''
    try {
      const service = await collectServiceOrders(tab.id)
      serviceOrders = service.orders || []
    } catch (e) {
      serviceError = e instanceof Error ? e.message : String(e)
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
  if (!device || syncing) return
  const tab = await findFxgTab()
  if (!tab?.id) return
  const extra = await chrome.storage.local.get([STORAGE.lastSyncAt])
  const last = Number(extra[STORAGE.lastSyncAt] || 0)
  if (Date.now() - last < minIntervalMs) return
  return syncNow()
}

chrome.tabs.onUpdated.addListener((tabId, info, tab) => {
  if (info.status !== 'complete') return
  if (!tab.url || !tab.url.includes('fxg.jinritemai.com')) return
  if (/login|passport/i.test(tab.url)) return
  rememberFxgTab(tabId)
  setTimeout(() => maybeAutoSync(60000), 2500)
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
    if (msg?.type === 'AFTERSALE_WORKBENCH_READY') {
      maybeAutoSync(60000)
      reply({ ok: true })
    }
  })()
  return true
})
