/** 抖店服务工单采集：待处理 / 处理中 / 已逾期 */

let pageDoc = document

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

function sameOriginDocs() {
  const list = []
  const walk = (doc) => {
    if (!doc || list.includes(doc)) return
    list.push(doc)
    for (const iframe of doc.querySelectorAll('iframe')) {
      try {
        walk(iframe.contentDocument)
      } catch {
        /* cross-origin */
      }
    }
  }
  walk(document)
  return list
}

function queryTabs(doc) {
  const root = doc || document
  const nodes = []
  const seen = new Set()
  const push = (el) => {
    if (!el || seen.has(el)) return
    seen.add(el)
    nodes.push(el)
  }
  const scan = (scope) => {
    if (!scope?.querySelectorAll) return
    for (const el of scope.querySelectorAll(
      '.sp-tabs-button .auxo-tabs-tab, [class*="sp-tabs"] .auxo-tabs-tab, .auxo-tabs-nav .auxo-tabs-tab, .auxo-tabs-tab, [role="tab"]',
    )) {
      push(el)
    }
    for (const el of scope.querySelectorAll('*')) {
      if (el.shadowRoot) scan(el.shadowRoot)
    }
  }
  scan(root)
  return nodes
}

function $all(selector, root = pageDoc) {
  return Array.from((root || document).querySelectorAll(selector))
}

function $(selector, root = pageDoc) {
  return (root || document).querySelector(selector)
}

const STATUS_TAB_RE =
  /^(全部|待处理|处理中|已完结|已逾期)(?:工单)?\s*[（(]?\s*(\d+)\s*[)）]?/
const STATUS_LABEL_RE = /^(全部|待处理|处理中|已完结|已逾期)$/

function parseTabEl(el) {
  const t = textOf(el)
  if (!t || t.length > 28) return null
  let m = t.match(STATUS_TAB_RE)
  if (!m) {
    m = t.match(STATUS_LABEL_RE)
    if (!m) return null
  }
  let count = m[2] != null ? Number(m[2]) : NaN
  if (!Number.isFinite(count)) {
    const badge = el.querySelector?.(
      '[class*="badge"], [class*="count"], [class*="num"], .auxo-badge-count, sup',
    )
    const raw = textOf(badge).replace(/[^\d]/g, '')
    count = raw ? Number(raw) : 0
  }
  const selected =
    /active/.test(String(el.className || '')) ||
    el.getAttribute?.('aria-selected') === 'true'
  return { key: m[1], label: m[1], count, selected, el }
}

function parseStatusTabsIn(doc) {
  const tabs = []
  const keys = new Set()
  for (const el of queryTabs(doc)) {
    const tab = parseTabEl(el)
    if (!tab) continue
    if (keys.has(tab.key)) continue
    keys.add(tab.key)
    tabs.push(tab)
  }
  return tabs
}

function locateServicePage() {
  for (const doc of sameOriginDocs()) {
    const tabs = parseStatusTabsIn(doc)
    if (tabs.some((t) => t.key === '待处理' || t.key === '处理中' || t.key === '已逾期')) {
      pageDoc = doc
      return tabs
    }
  }
  pageDoc = document
  return parseStatusTabsIn(document)
}

function parseStatusTabs() {
  return locateServicePage()
}

function servicePageReady() {
  return locateServicePage().some((t) => t.key === '待处理' || t.key === '处理中' || t.key === '已逾期')
}

function parseListTotal() {
  const el = $('.auxo-pagination-total-text')
  const m = textOf(el).match(/共\s*(\d+)\s*条/)
  return m ? Number(m[1]) : null
}

function currentPage() {
  const el = $('.auxo-pagination-item-active')
  const n = Number(textOf(el))
  return Number.isFinite(n) && n > 0 ? n : 1
}

function findNextPage() {
  const next = $('.auxo-pagination-next')
  if (!next) return null
  const cls = String(next.className || '')
  if (/disabled/i.test(cls) || next.getAttribute('aria-disabled') === 'true') return null
  const btn = next.querySelector('button, a') || next
  if (btn.disabled || btn.getAttribute('disabled') != null) return null
  return btn
}

async function ensureAllTypeTab() {
  for (const doc of [pageDoc, ...sameOriginDocs()]) {
    const nodes = $all('.auxo-tabs-tab, [role="tab"]', doc)
    for (const el of nodes) {
      const t = textOf(el)
      if (t !== '全部工单' && !/^全部工单/.test(t)) continue
      if (!/active/.test(String(el.className || '')) && el.getAttribute('aria-selected') !== 'true') {
        el.click()
        await sleep(600)
      }
      return
    }
  }
}

async function ensureFirstPage() {
  if (currentPage() <= 1) return
  const first = $('.auxo-pagination-item-1') || $('li.auxo-pagination-item[title="1"]')
  if (!first) return
  first.click()
  await waitUntil(() => (currentPage() <= 1 ? true : null), 15, 150)
}

function refreshMainRows() {
  return new Promise((resolve) => {
    const roots = [...new Set([pageDoc, document].filter(Boolean))]
    const timer = setTimeout(() => {
      for (const root of roots) root.removeEventListener('osms-service-rows', onMsg)
      resolve([])
    }, 800)
    function onMsg(e) {
      clearTimeout(timer)
      for (const root of roots) root.removeEventListener('osms-service-rows', onMsg)
      resolve(Array.isArray(e.detail) ? e.detail : [])
    }
    for (const root of roots) {
      root.addEventListener('osms-service-rows', onMsg)
      root.dispatchEvent(new Event('osms-service-need-rows'))
    }
  })
}

function parseDomRows() {
  const table = $('table')
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
  const tabs = await waitUntil(() => {
    const found = locateServicePage()
    return found.some((t) => t.key === '待处理' || t.key === '处理中' || t.key === '已逾期') ? found : null
  }, 80, 400)
  if (!tabs?.length) {
    throw new Error('未找到服务工单状态页签，请确认已打开服务工单')
  }
  await ensureAllTypeTab()
  const latest = locateServicePage()
  const useTabs = latest.length ? latest : tabs
  await clearListViaZeroTab(useTabs)
  const targets = useTabs.filter((t) => t.key !== '全部' && t.key !== '已完结' && t.count > 0)
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
    tabs: useTabs.map(({ el, ...rest }) => rest),
    orders: Array.from(orderMap.values()),
  }
}

if (!window.__osmsServiceOrderInjected) {
  window.__osmsServiceOrderInjected = true
  chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
    if (msg?.type === 'AFTERSALE_PING_SERVICE') {
      const tabs = locateServicePage()
      sendResponse({
        ok: true,
        ready: servicePageReady(),
        diag: {
          href: location.href,
          iframeCount: document.querySelectorAll('iframe').length,
          pageIsTop: pageDoc === document,
          tabs: tabs.map(({ el, ...rest }) => rest),
        },
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
