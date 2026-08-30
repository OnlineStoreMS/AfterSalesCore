/** 抖店售后工作台采集：快捷筛选卡片 + 表格成对行（表头订单行 + 商品明细行） */

let pageDoc = document

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}

function textOf(el) {
  return String(el?.innerText || el?.textContent || '').replace(/\s+/g, ' ').trim()
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

function $all(selector, root = pageDoc) {
  return Array.from((root || document).querySelectorAll(selector))
}

function $(selector, root = pageDoc) {
  return (root || document).querySelector(selector)
}

function queryAny(selector) {
  for (const doc of sameOriginDocs()) {
    const el = doc.querySelector(selector)
    if (el) return el
  }
  return document.querySelector(selector)
}

function queryAnyAll(selector) {
  const out = []
  for (const doc of sameOriginDocs()) {
    out.push(...doc.querySelectorAll(selector))
  }
  if (!out.length) out.push(...document.querySelectorAll(selector))
  return out
}

function ui(name) {
  return queryAny(`.aurora-${name}, .auxo-${name}`)
}

function uiAll(name) {
  const aurora = queryAnyAll(`.aurora-${name}`)
  return aurora.length ? aurora : queryAnyAll(`.auxo-${name}`)
}

function locateWorkbenchPage() {
  for (const doc of sameOriginDocs()) {
    if (
      doc.querySelectorAll('[class*="groupTitle"]').length > 0 &&
      doc.querySelectorAll('[class*="groupItems"]').length > 0
    ) {
      pageDoc = doc
      return true
    }
  }
  pageDoc = document
  return false
}

function pick(text, re) {
  const m = String(text || '').match(re)
  return m ? String(m[1] || '').trim() : ''
}

function parseCards() {
  locateWorkbenchPage()
  const cards = []
  const titles = $all('[class*="groupTitle"]')
  let sort = 0
  for (const titleEl of titles) {
    const groupName = textOf(titleEl)
    if (!groupName || groupName.length > 24) continue
    if (/自定义|服务体验|工作台|筛选/.test(groupName)) continue
    const root = titleEl.parentElement
    if (!root) continue
    const itemRoot = root.querySelector('[class*="groupItems"]') || root
    const items = Array.from(itemRoot.children).filter((el) => {
      const label = el.querySelector('[class*="label"]')
      return !!textOf(label)
    })
    for (const it of items) {
      const label = textOf(it.querySelector('[class*="label"]'))
      if (!label) continue
      const nums = (textOf(it).match(/\d+/g) || []).map(Number)
      const count = nums.length ? nums[nums.length - 1] : 0
      cards.push({
        groupName,
        cardLabel: label,
        cardKey: `${groupName}:${label}`,
        count,
        sortOrder: sort++,
        el: it,
      })
    }
  }
  return cards
}

function shopMeta() {
  locateWorkbenchPage()
  const nameEl =
    $('[class*="headerShopName"]') ||
    $('[class*="userName"]') ||
    document.querySelector('[class*="headerShopName"]') ||
    document.querySelector('[class*="userName"]')
  let platformShopName = textOf(nameEl)
  let platformShopId = ''
  try {
    const html = `${document.documentElement.innerHTML}\n${pageDoc?.documentElement?.innerHTML || ''}`.slice(0, 400000)
    const idMatch = html.match(/shopid[=:]["']?(\d{5,})/i) || html.match(/"shop_id"\s*:\s*"?(\d{5,})/)
    if (idMatch) platformShopId = idMatch[1]
    if (!platformShopName) {
      const nameMatch = html.match(/shopname=([^&"'<]+)/i)
      if (nameMatch) platformShopName = decodeURIComponent(nameMatch[1])
    }
  } catch {
    /* ignore */
  }
  return { platformShopId, platformShopName }
}

function headerCells(table) {
  const ths = Array.from(table.querySelectorAll('thead th, tr:first-child th'))
  if (ths.length) return ths.map((th) => textOf(th))
  const first = table.querySelector('tr')
  return first ? Array.from(first.children).map((c) => textOf(c)) : []
}

function cellByName(row, names, headers) {
  for (const name of names) {
    const i = headers.findIndex((h) => h.includes(name))
    if (i >= 0 && row.children[i]) return row.children[i]
  }
  return null
}

function productImageSrc(cell) {
  const imgs = Array.from(cell?.querySelectorAll('img') || [])
  const hit = imgs.find((img) => {
    const src = img.currentSrc || img.src || ''
    const w = img.naturalWidth || img.width || 0
    return src && !src.startsWith('data:') && w >= 32
  })
  return hit ? hit.currentSrc || hit.src : ''
}

function parseHeaderRow(tr) {
  let kora = {}
  const koraEl = tr.querySelector('[data-kora-json]')
  if (koraEl) {
    try {
      kora = JSON.parse(koraEl.getAttribute('data-kora-json') || '{}')
    } catch {
      kora = {}
    }
  }
  const text = tr.innerText || ''
  const orderNo = String(kora.order_id || pick(text, /订单编号\s*([0-9]+)/) || '')
  const aftersaleId = String(kora.after_sale_id || pick(text, /售后编号\s*([0-9]+)/) || '')
  const lines = String(text)
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
  const skip = /^(订单编号|售后编号|[0-9]{8,}|[^\w\u4e00-\u9fa5]*\*)$/
  const typeHit = lines.find((l) => /退货退款|仅退款|换货|补寄|维修/.test(l))
  const tags = lines.filter((l) => !skip.test(l) && l !== typeHit && l.length < 20)
  return {
    orderNo,
    platformAftersaleId: aftersaleId,
    aftersaleType: typeHit || '',
    tags: tags.join('、'),
  }
}

function reactFiber(el) {
  if (!el) return null
  const key = Object.keys(el).find(
    (k) => k.startsWith('__reactFiber') || k.startsWith('__reactInternalInstance'),
  )
  return key ? el[key] : null
}

function reactProps(el, pred, depth = 30) {
  let n = reactFiber(el)
  for (let i = 0; i < depth && n; i++) {
    const p = n.memoizedProps || n.pendingProps
    if (p && pred(p)) return p
    n = n.return
  }
  return null
}

function pagerProps() {
  return reactProps(
    ui('pagination'),
    (p) => typeof p.onChange === 'function' && ('total' in p || 'pageSize' in p),
  )
}

function changeSizeFn() {
  const p = reactProps(
    ui('pagination-options-size-changer'),
    (x) => typeof x.changeSize === 'function',
  )
  return p?.changeSize || null
}

function isShown(el) {
  if (!el) return false
  const r = el.getBoundingClientRect()
  return r.width > 2 && r.height > 2
}

function findTableRecord(el) {
  let n = reactFiber(el)
  for (let i = 0; i < 20 && n; i++) {
    const p = n.memoizedProps || n.pendingProps
    if (p?.record) return p.record
    n = n.return
  }
  return null
}

function returnLogisticsNoFromRecord(record, aftersaleId) {
  const info = record?.parentRecord?.after_sale_info || record?.after_sale_info || {}
  const id = String(info.after_sale_id || record?.parentRecord?.after_sale_id || '')
  if (aftersaleId && id && id !== String(aftersaleId)) return ''
  return String(info.return_logistics_code || record?.return_logistics_code || '').trim()
}

function parseDataRow(tr, headers, headerInfo) {
  const productCell = cellByName(tr, ['商品信息'], headers) || tr.children[1]
  const orderCell = cellByName(tr, ['订单信息'], headers)
  const afterCell = cellByName(tr, ['售后信息'], headers)
  const statusCell = cellByName(tr, ['售后状态'], headers)
  const disputeCell = cellByName(tr, ['纠纷仲裁'], headers)
  const logisticsCell = cellByName(tr, ['物流信息'], headers)

  const productTitle = textOf(
    productCell?.querySelector('[class*="name-"]') ||
      productCell?.querySelector('[class*="ellipsis"]'),
  )
  const sku = textOf(productCell?.querySelector('[class*="spec-"]'))
  const productTags = Array.from(productCell?.querySelectorAll('[class*="tag"]') || [])
    .map((el) => textOf(el))
    .filter((t) => t && t.length <= 12)
    .filter((t, i, arr) => arr.indexOf(t) === i)
    .join('、')

  const orderText = textOf(orderCell)
  const afterText = textOf(afterCell)
  const statusText = (statusCell?.innerText || '').trim()
  const statusLines = statusText.split('\n').map((s) => s.trim()).filter(Boolean)
  const logisticsRaw = (logisticsCell?.innerText || '')
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
    .join('\n')

  const applyQty = Number(pick(afterText, /申请件数\s*(\d+)/) || 0)
  const buyQty = Number(pick(orderText, /购买件数\s*(\d+)/) || 0)
  const returnLogisticsNo = returnLogisticsNoFromRecord(
    findTableRecord(logisticsCell) || findTableRecord(tr),
    headerInfo.platformAftersaleId,
  )

  return {
    platformAftersaleId: headerInfo.platformAftersaleId,
    orderNo: headerInfo.orderNo,
    productTitle: productTitle || '',
    productImage: productImageSrc(productCell),
    sku,
    productTags,
    tags: headerInfo.tags,
    qty: Number.isFinite(applyQty) ? applyQty : 0,
    buyQty: Number.isFinite(buyQty) ? buyQty : 0,
    payAmount: pick(orderText, /应付金额\s*¥?\s*([\d.]+)/),
    refundAmount: pick(afterText, /售后退款\s*¥?\s*([\d.]+)/),
    aftersaleType: headerInfo.aftersaleType || pick(afterText, /(退货退款|仅退款|换货|补寄|维修|未发货退款|已发货退款)/),
    reason: pick(afterText, /申请原因\s*(.+?)(?=\s*申请时间|$)/),
    status: statusLines[0] || '',
    timeoutText: statusLines.slice(1).join(' '),
    dispute: textOf(disputeCell),
    logistics: logisticsRaw,
    returnLogisticsNo,
    applyTime: pick(afterText, /申请时间\s*([\d/]+(?:\s*[\d:]+)?)/),
    rawJson: JSON.stringify({
      headerTags: headerInfo.tags,
      product: textOf(productCell).slice(0, 400),
      order: orderText.slice(0, 200),
      after: afterText.slice(0, 300),
      status: statusText.slice(0, 120),
      logistics: logisticsRaw.slice(0, 200),
      returnLogisticsNo,
    }),
  }
}

let logisticsMap = {}

function refreshLogisticsMap() {
  return new Promise((resolve) => {
    const roots = [...new Set([pageDoc, document].filter(Boolean))]
    const timer = setTimeout(() => {
      for (const root of roots) root.removeEventListener('osms-aftersale-logistics', onMsg)
      resolve(logisticsMap)
    }, 600)
    function onMsg(e) {
      clearTimeout(timer)
      for (const root of roots) root.removeEventListener('osms-aftersale-logistics', onMsg)
      logisticsMap = e.detail && typeof e.detail === 'object' ? e.detail : {}
      resolve(logisticsMap)
    }
    for (const root of roots) {
      root.addEventListener('osms-aftersale-logistics', onMsg)
      root.dispatchEvent(new Event('osms-aftersale-need-logistics'))
    }
  })
}

function applyLogistics(ticket) {
  const code = String(ticket.returnLogisticsNo || logisticsMap[ticket.platformAftersaleId] || '').trim()
  if (!code) return ticket
  let raw = ticket.rawJson
  try {
    const o = JSON.parse(raw || '{}')
    o.returnLogisticsNo = code
    raw = JSON.stringify(o)
  } catch {
    /* ignore */
  }
  return { ...ticket, returnLogisticsNo: code, rawJson: raw }
}

async function visibleTicketsWithLogistics() {
  await refreshLogisticsMap()
  return collectVisibleTickets().map(applyLogistics)
}

function collectVisibleTickets() {
  const table = $('table')
  if (!table) return []
  const headers = headerCells(table)
  const trs = Array.from(table.querySelectorAll('tbody tr, tr'))
  const rows = []
  const seen = new Set()
  for (let i = 0; i < trs.length; i++) {
    const tr = trs[i]
    const txt = tr.innerText || ''
    const isHeader =
      txt.includes('售后编号') &&
      txt.includes('订单编号') &&
      (tr.className.includes('level-0') || tr.children.length <= 3)
    if (!isHeader) continue
    const headerInfo = parseHeaderRow(tr)
    if (!headerInfo.platformAftersaleId || seen.has(headerInfo.platformAftersaleId)) continue
    const data = trs[i + 1]
    if (!data || data.children.length < 5) continue
    if (!/应付金额|售后退款|商品/.test(data.innerText || '')) continue
    const parsed = parseDataRow(data, headers, headerInfo)
    if (!parsed.platformAftersaleId) continue
    seen.add(parsed.platformAftersaleId)
    rows.push(parsed)
  }
  return rows
}

function isCardSelected(el) {
  return /selected/i.test(String(el?.className || ''))
}

function fireClick(el) {
  if (!el) return
  el.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true, cancelable: true, view: window }))
  el.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true, view: window }))
  el.dispatchEvent(new MouseEvent('mouseup', { bubbles: true, cancelable: true, view: window }))
  el.click()
}

function findPageItem(n) {
  return (
    ui(`pagination-item-${n}`) ||
    queryAny(`li.aurora-pagination-item[title="${n}"], li.auxo-pagination-item[title="${n}"]`)
  )
}

function parseListTotal() {
  for (const el of uiAll('pagination-total-text')) {
    const m = textOf(el).match(/共\s*(\d+)\s*条/)
    if (m) return Number(m[1])
  }
  return null
}

function currentPage() {
  const el = ui('pagination-item-active')
  const n = Number(textOf(el))
  return Number.isFinite(n) && n > 0 ? n : 1
}

function currentPageSize() {
  const changer = ui('pagination-options-size-changer')
  const item = changer?.querySelector(
    '.aurora-select-content-value, .auxo-select-selection-item, [title*="条/页"]',
  )
  const raw = item?.getAttribute('title') || textOf(item) || ''
  const n = Number((raw.match(/(\d+)/) || [])[1])
  return Number.isFinite(n) && n > 0 ? n : 10
}

function visibleHeaderCount() {
  let n = 0
  for (const doc of sameOriginDocs()) {
    for (const tr of doc.querySelectorAll('table tr')) {
      const t = tr.innerText || ''
      if (t.includes('售后编号') && t.includes('订单编号')) n++
    }
  }
  return n
}

function findNextPage() {
  const next = ui('pagination-next')
  if (!next) return null
  const cls = String(next.className || '')
  if (/disabled/i.test(cls) || next.getAttribute('aria-disabled') === 'true') return null
  const btn = next.querySelector('button, a') || next
  if (btn.disabled || btn.getAttribute('disabled') != null) return null
  return btn
}

async function waitFirstPageRows(expected) {
  const size = currentPageSize()
  const need = expected > 0 ? Math.min(expected, size >= 50 ? expected : Math.max(size, 10)) : 0
  if (!need) {
    await sleep(400)
    return visibleHeaderCount()
  }
  await waitUntil(() => (visibleHeaderCount() >= need ? true : null), 50, 200)
  return visibleHeaderCount()
}

function drivePage(detail) {
  return new Promise((resolve) => {
    const roots = [...new Set([pageDoc, document].filter(Boolean))]
    const timer = setTimeout(() => {
      cleanup()
      resolve(null)
    }, 800)
    function onDone(e) {
      cleanup()
      resolve(e.detail || { ok: true })
    }
    function cleanup() {
      clearTimeout(timer)
      for (const root of roots) root.removeEventListener('osms-aftersale-drive-done', onDone)
    }
    for (const root of roots) {
      root.addEventListener('osms-aftersale-drive-done', onDone)
      root.dispatchEvent(new CustomEvent('osms-aftersale-drive', { detail }))
    }
  })
}

async function applyPageSize(size) {
  const driven = await drivePage({ action: 'setPageSize', size })
  if (driven?.ok) return true
  const changeSize = changeSizeFn()
  const pager = pagerProps()
  try {
    if (typeof changeSize === 'function') {
      changeSize(size)
      return true
    }
    if (typeof pager?.onShowSizeChange === 'function') {
      pager.onShowSizeChange(1, size)
      return true
    }
    if (typeof pager?.onChange === 'function') {
      pager.onChange(1, size)
      return true
    }
  } catch {
    /* fall through */
  }
  const changer = ui('pagination-options-size-changer')
  if (!changer) return false
  fireClick(changer.querySelector('.aurora-select-content, .auxo-select-selector') || changer)
  const opt = await waitUntil(() => {
    let best = null
    let bestN = 0
    for (const el of queryAnyAll(
      '.aurora-select-item, .aurora-select-item-option, .auxo-select-item, .auxo-select-item-option, [class*="select-item-option"]',
    )) {
      if (!isShown(el)) continue
      const t = textOf(el)
      if (!/条\/页/.test(t) && !/^\d+$/.test(t)) continue
      const n = Number((t.match(/(\d+)/) || [])[1])
      if (n === size || (size >= 50 && n >= 50 && n > bestN)) {
        bestN = n
        best = el
      }
    }
    return best
  }, 20, 150)
  if (opt) fireClick(opt)
  return !!opt
}

async function setLargestPageSize(expected) {
  for (let i = 0; i < 3 && currentPageSize() < 50; i++) {
    await applyPageSize(100)
    if (await waitUntil(() => (currentPageSize() >= 50 ? true : null), 20, 150)) break
  }
  await waitFirstPageRows(expected)
  return currentPageSize()
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

async function clickCard(el) {
  if (!el) return
  try {
    el.scrollIntoView({ block: 'nearest' })
  } catch {
    /* ignore */
  }
  fireClick(el)
  await sleep(80)
}

function liveCard(card) {
  locateWorkbenchPage()
  return parseCards().find((c) => c.cardKey === card.cardKey) || card
}

async function waitEmptyList() {
  await waitUntil(() => {
    const total = parseListTotal()
    if (total === 0) return true
    if (total == null && collectVisibleTickets().length === 0) return true
    return null
  }, 20, 150)
}

async function goToPage(n) {
  if (currentPage() === n) return true
  const prevIds = collectVisibleTickets().map((r) => r.platformAftersaleId)
  const driven = await drivePage({ action: 'goPage', page: n, size: currentPageSize() || 10 })
  const pager = pagerProps()
  try {
    if (driven?.ok) {
      /* MAIN world already changed page */
    } else if (typeof pager?.onChange === 'function') {
      pager.onChange(n, currentPageSize() || 10)
    } else {
      const item = findPageItem(n)
      if (item) fireClick(item)
      else {
        const next = findNextPage()
        if (!next) return false
        fireClick(next)
      }
    }
  } catch {
    return false
  }
  return !!(await waitUntil(() => {
    if (currentPage() !== n) return null
    const rows = collectVisibleTickets()
    if (!prevIds.length) return true
    if (rows.some((r) => r.platformAftersaleId && !prevIds.includes(r.platformAftersaleId))) return true
    return null
  }, 40, 200))
}

async function ensureFirstPage() {
  if (currentPage() <= 1) return
  await goToPage(1)
}

function cardTableState(card) {
  const live = liveCard(card)
  if (!isCardSelected(live.el)) return null
  if (currentPage() > 1) return null
  const total = parseListTotal()
  const rows = collectVisibleTickets()
  if (card.count === 0) {
    if (total === 0 || rows.length === 0) return { rows, total: total ?? 0 }
    return null
  }
  if (total !== card.count) return null
  if (rows.length === 0 || rows.length > card.count) return null
  if (currentPageSize() >= 50 && rows.length < card.count) return null
  return { rows, total }
}

async function waitForPageChange(prevIds) {
  const prev = new Set(prevIds || [])
  return (
    (await waitUntil(() => {
      const rows = collectVisibleTickets()
      if (!rows.length) return null
      if (rows.some((r) => !prev.has(r.platformAftersaleId))) return rows
      return null
    }, 20, 250)) || []
  )
}

function emptyCardResult() {
  const all = []
  all.pageSize = currentPageSize()
  all.visible = visibleHeaderCount()
  return all
}

function listBelongsToCard(card) {
  const live = liveCard(card)
  if (!isCardSelected(live.el)) return false
  return parseListTotal() === card.count
}

async function collectCardTickets(card) {
  const matched = () => (listBelongsToCard(card) ? true : null)
  for (let i = 0; i < 3 && !matched(); i++) {
    await clickCard(liveCard(card).el)
    await waitUntil(matched, 40, 200)
  }
  if (!matched()) return emptyCardResult()
  await setLargestPageSize(card.count)
  if (!matched()) {
    await clickCard(liveCard(card).el)
    await waitUntil(matched, 40, 200)
  }
  if (!matched()) return emptyCardResult()
  await ensureFirstPage()
  await waitUntil(() => cardTableState(card), 50, 200)
  if (!listBelongsToCard(card)) return emptyCardResult()
  const all = []
  const seen = new Set()
  const expected = card.count || 0
  const add = (rows) => {
    if (!listBelongsToCard(card)) return
    if ((rows || []).length > expected) return
    for (const r of rows || []) {
      if (!r.platformAftersaleId || seen.has(r.platformAftersaleId)) continue
      if (expected && all.length >= expected) break
      seen.add(r.platformAftersaleId)
      all.push(r)
    }
  }
  add(await visibleTicketsWithLogistics())
  if (expected && all.length < expected) {
    await setLargestPageSize(expected)
    if (listBelongsToCard(card)) add(await visibleTicketsWithLogistics())
  }
  for (let p = 2; expected && all.length < expected && p <= 20; p++) {
    if (!listBelongsToCard(card)) break
    const prevIds = all.map((r) => r.platformAftersaleId)
    const moved = await goToPage(p)
    if (!moved) {
      const next = findNextPage()
      if (!next) break
      fireClick(next)
      await waitForPageChange(prevIds)
    }
    add(await visibleTicketsWithLogistics())
  }
  all.pageSize = currentPageSize()
  all.visible = visibleHeaderCount()
  return all
}

async function collectAll() {
  locateWorkbenchPage()
  const cardRows = parseCards()
  if (!cardRows.length) {
    throw new Error('未找到快捷筛选卡片，请确认当前是售后工作台列表页')
  }
  const ticketMap = new Map()
  const cardStats = []
  for (const card of cardRows) {
    if (!card.count) continue
    const rows = await collectCardTickets(card)
    cardStats.push({
      cardKey: card.cardKey,
      label: `${card.groupName}·${card.cardLabel}`,
      expected: card.count,
      got: rows.length,
      pageSize: rows.pageSize,
      visible: rows.visible,
    })
    for (const t of rows) {
      const existing = ticketMap.get(t.platformAftersaleId)
      if (existing) {
        if (!existing.cardKeys.includes(card.cardKey)) existing.cardKeys.push(card.cardKey)
        if (!existing.returnLogisticsNo && t.returnLogisticsNo) {
          existing.returnLogisticsNo = t.returnLogisticsNo
        }
      } else {
        ticketMap.set(t.platformAftersaleId, { ...t, cardKeys: [card.cardKey] })
      }
    }
  }
  const meta = shopMeta()
  return {
    ok: true,
    ...meta,
    cards: cardRows.map(({ el, ...rest }) => rest),
    tickets: Array.from(ticketMap.values()),
    cardStats,
  }
}

window.__osmsCollectWorkbench = collectAll
window.__osmsWorkbenchReady = () => parseCards().length > 0

if (!window.__osmsWorkbenchInjected && typeof chrome !== 'undefined' && chrome.runtime?.onMessage) {
  window.__osmsWorkbenchInjected = true
  chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
    if (msg?.type === 'AFTERSALE_PING_WORKBENCH') {
      sendResponse({ ok: true, ready: parseCards().length > 0 })
      return
    }
    if (msg?.type !== 'AFTERSALE_COLLECT') return
    collectAll()
      .then((data) => sendResponse(data))
      .catch((e) => sendResponse({ ok: false, error: e instanceof Error ? e.message : String(e) }))
    return true
  })
  setTimeout(() => {
    try {
      if (parseCards().length) chrome.runtime.sendMessage({ type: 'AFTERSALE_WORKBENCH_READY' })
    } catch {
      /* ignore */
    }
  }, 4000)
}
