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
const WORKBENCH_MATCH = 'https://fxg.jinritemai.com/ffa/merchant-aftersale-workbench/*'

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

async function findWorkbenchTab() {
  const tabs = await chrome.tabs.query({ url: [WORKBENCH_MATCH] })
  return tabs.find((t) => t.id) || null
}

async function collectFromTab(tabId) {
  for (let i = 0; i < 8; i++) {
    try {
      const res = await chrome.tabs.sendMessage(tabId, { type: 'AFTERSALE_COLLECT' })
      if (res?.ok) return res
      throw new Error(res?.error || '采集失败')
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      if (i === 7) throw new Error(msg.includes('Receiving end') ? '请先打开并停留在抖店售后工作台页面' : msg)
      await sleep(400)
    }
  }
  throw new Error('采集失败')
}

async function syncNow() {
  if (syncing) return { ok: false, error: '正在同步' }
  const device = await getDevice()
  if (!device) return { ok: false, error: '尚未绑定店铺' }
  const tab = await findWorkbenchTab()
  if (!tab?.id) return { ok: false, error: '请先打开抖店售后工作台' }
  syncing = true
  const keepAlive = setInterval(() => {
    chrome.runtime.getPlatformInfo(() => {})
  }, 15000)
  try {
    const collected = await collectFromTab(tab.id)
    const payload = {
      platformShopId: collected.platformShopId || '',
      platformShopName: collected.platformShopName || '',
      cards: collected.cards || [],
      tickets: collected.tickets || [],
    }
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
    return {
      ok: true,
      cardCount: data?.cardCount ?? payload.cards.length,
      ticketCount: data?.ticketCount ?? payload.tickets.length,
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
  const tab = await findWorkbenchTab()
  if (!tab?.id) return
  const extra = await chrome.storage.local.get([STORAGE.lastSyncAt])
  const last = Number(extra[STORAGE.lastSyncAt] || 0)
  if (Date.now() - last < minIntervalMs) return
  return syncNow()
}

chrome.tabs.onUpdated.addListener((tabId, info, tab) => {
  if (info.status !== 'complete') return
  if (!tab.url || !tab.url.includes('/ffa/merchant-aftersale-workbench/')) return
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
