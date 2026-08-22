/** 抖店服务工单采集：待处理 / 处理中 / 已逾期 */

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}

function textOf(el) {
  return String(el?.innerText || el?.textContent || '').replace(/\s+/g, ' ').trim()
}

async function waitUntil(pred, tries = 30, gap = 200) {
  let last = null
  for (let i = 0; i < tries; i++) {
    last = pred()
    if (last) return last
    await sleep(gap)
  }
  return last
}

function parseListTotal() {
  const el = document.querySelector('.auxo-pagination-total-text')
  const m = textOf(el).match(/共\s*(\d+)\s*条/)
  return m ? Number(m[1]) : null
}

function currentPage() {
  const el = document.querySelector('.auxo-pagination-item-active')
  const n = Number(textOf(el))
  return Number.isFinite(n) && n > 0 ? n : 1
}

function findNextPage() {
  const next = document.querySelector('.auxo-pagination-next')
  if (!next) return null
  const cls = String(next.className || '')
  if (/disabled/i.test(cls) || next.getAttribute('aria-disabled') === 'true') return null
  const btn = next.querySelector('button, a') || next
  if (btn.disabled || btn.getAttribute('disabled') != null) return null
  return btn
}

async function ensureAllTypeTab() {
  const nodes = document.querySelectorAll('.auxo-tabs-tab')
  for (const el of nodes) {
    const t = textOf(el)
    if (t !== '全部工单' && !t.startsWith('全部工单')) continue
    if (!/active/.test(String(el.className || ''))) {
      el.click()
      await sleep(400)
    }
    return
  }
}

async function ensureFirstPage() {
  if (currentPage() <= 1) return
  const first =
    document.querySelector('.auxo-pagination-item-1') ||
    document.querySelector('li.auxo-pagination-item[title="1"]')
  if (!first) return
  first.click()
  await waitUntil(() => (currentPage() <= 1 ? true : null), 15, 150)
}

function parseStatusTabs() {
  const tabs = []
  const nodes = document.querySelectorAll('.sp-tabs-button .auxo-tabs-tab, [class*="sp-tabs"] .auxo-tabs-tab')
  for (const el of nodes) {
    const t = textOf(el)
    const m = t.match(/^(全部|待处理|处理中|已完结|已逾期)\s*(\d+)/)
    if (!m) continue
    tabs.push({
      key: m[1],
      label: m[1],
      count: Number(m[2] || 0),
      selected: /active/.test(String(el.className || '')),
      el,
    })
  }
  return tabs
}

function refreshMainRows() {
  return new Promise((resolve) => {
    const timer = setTimeout(() => {
      document.removeEventListener('osms-service-rows', onMsg)
      resolve([])
    }, 500)
    function onMsg(e) {
      clearTimeout(timer)
      document.removeEventListener('osms-service-rows', onMsg)
      resolve(Array.isArray(e.detail) ? e.detail : [])
    }
    document.addEventListener('osms-service-rows', onMsg)
    document.dispatchEvent(new Event('osms-service-need-rows'))
  })
}

function parseDomRows() {
  const table = document.querySelector('table')
  if (!table) return []
  const trs = Array.from(table.querySelectorAll('tbody tr'))
  const rows = []
  for (const tr of trs) {
    const cells = Array.from(tr.children).map((td) => String(td.innerText || '').trim())
    if (cells.length < 5) continue
    const orderText = cells[0] || ''
    const sourceText = cells[2] || ''
    const typeText = cells[3] || ''
    const progressText = cells[4] || ''
    const id = (sourceText.match(/工单ID[:：]?\s*(\d+)/) || [])[1] || ''
    if (!id) continue
    const orderNo = (orderText.match(/(\d{15,})/) || [])[1] || ''
    const progressLines = progressText.split('\n').map((s) => s.trim()).filter(Boolean)
    const lastLines = String(cells[6] || '').split('\n').map((s) => s.trim()).filter(Boolean)
    rows.push({
      platformServiceId: id,
      orderNo,
      productTitle: orderText.split('\n').filter((l) => l && !/^\d{15,}$/.test(l) && l !== orderNo)[1] || '',
      productImage: '',
      productContent: (orderText.match(/总价.*/) || [])[0] || '',
      buyerNick: cells[1] || '',
      createSource: sourceText.split('\n')[0] || '',
      businessType: typeText.split('\n')[0] || '',
      orderType: typeText.split('\n')[1] || '',
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

async function visibleOrders() {
  const main = await refreshMainRows()
  const dom = parseDomRows()
  if (!main.length) return dom
  const byId = new Map(dom.map((r) => [r.platformServiceId, r]))
  return main.map((r) => {
    const d = byId.get(r.platformServiceId)
    if (!d) return r
    return {
      ...r,
      timeoutText: r.timeoutText || d.timeoutText,
      status: r.status || d.status,
      solution: r.solution && r.solution !== '_' ? r.solution : d.solution,
      lastLog: r.lastLog || d.lastLog,
      lastLogTime: r.lastLogTime || d.lastLogTime,
      tags: r.tags || d.tags,
      productTitle: r.productTitle || d.productTitle,
      buyerNick: r.buyerNick || d.buyerNick,
    }
  })
}

function visibleIds() {
  return parseDomRows().map((r) => r.platformServiceId)
}

async function waitForPageChange(prevIds) {
  const prev = new Set(prevIds || [])
  return (
    (await waitUntil(() => {
      const ids = visibleIds()
      if (!ids.length) return null
      if (ids.some((id) => !prev.has(id))) return ids
      return null
    }, 20, 250)) || []
  )
}

async function clearListViaZeroTab(tabs) {
  const zero = tabs.find((t) => t.count === 0 && t.key !== '全部')
  if (!zero?.el) return
  zero.el.click()
  await waitUntil(() => {
    if (!/active/.test(String(zero.el.className || ''))) return null
    const total = parseListTotal()
    const n = visibleIds().length
    if ((total === 0 || total == null) && n === 0) return true
    if (total === 0) return n === 0 ? true : null
    return null
  }, 20, 200)
}

async function collectTabOrders(tab) {
  tab.el.click()
  await waitUntil(() => {
    if (!/active/.test(String(tab.el.className || ''))) return null
    const total = parseListTotal()
    if (total != null && total !== tab.count) return null
    return true
  }, 30, 200)
  await ensureFirstPage()
  const ready = await waitUntil(() => {
    if (currentPage() > 1) return null
    const total = parseListTotal()
    if (total != null && total !== tab.count) return null
    const n = visibleIds().length
    if (tab.count === 0) return n === 0 ? true : null
    if (n === 0) return null
    if (n > Math.min(10, tab.count)) return null
    return true
  }, 25, 200)
  if (tab.count === 0) return []
  if (!ready) return []

  const all = []
  const seen = new Set()
  const add = (rows) => {
    for (const r of rows || []) {
      const id = String(r.platformServiceId || '').trim()
      if (!id || seen.has(id)) continue
      seen.add(id)
      all.push({ ...r, statusTab: tab.key })
    }
  }
  add(await visibleOrders())
  const maxPages = Math.max(1, Math.ceil((tab.count || 10) / 10) + 1)
  for (let p = 1; p < maxPages; p++) {
    if (all.length >= tab.count) break
    const next = findNextPage()
    if (!next) break
    const prevIds = visibleIds()
    next.click()
    await waitForPageChange(prevIds)
    add(await visibleOrders())
  }
  return all
}

async function collectAll() {
  await ensureAllTypeTab()
  const tabs = await waitUntil(() => {
    const found = parseStatusTabs()
    return found.length ? found : null
  }, 40, 250)
  if (!tabs?.length) {
    throw new Error('未找到服务工单状态页签，请确认当前是服务工单页面')
  }
  await clearListViaZeroTab(tabs)
  const targets = tabs.filter((t) => t.key !== '全部' && t.key !== '已完结' && t.count > 0)
  const orderMap = new Map()
  for (const tab of targets) {
    const rows = await collectTabOrders(tab)
    for (const r of rows) {
      orderMap.set(r.platformServiceId, r)
    }
  }
  for (const tab of targets) {
    const got = Array.from(orderMap.values()).filter((o) => o.statusTab === tab.key).length
    if (got === 0 || (tab.key === '待处理' && got < tab.count)) {
      throw new Error(`${tab.key}应有${tab.count}条，实际采集${got}条`)
    }
  }
  return {
    ok: true,
    tabs: tabs.map(({ el, ...rest }) => rest),
    orders: Array.from(orderMap.values()),
  }
}

if (!window.__osmsServiceOrderInjected) {
  window.__osmsServiceOrderInjected = true
  chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
    if (msg?.type === 'AFTERSALE_PING_SERVICE') {
      const tabs = parseStatusTabs()
      sendResponse({
        ok: true,
        ready: tabs.some((t) => t.key === '待处理' || t.key === '处理中' || t.key === '已逾期'),
      })
      return
    }
    if (msg?.type !== 'AFTERSALE_COLLECT_SERVICE') return
    collectAll()
      .then((data) => sendResponse(data))
      .catch((e) => sendResponse({ ok: false, error: e instanceof Error ? e.message : String(e) }))
    return true
  })
}
