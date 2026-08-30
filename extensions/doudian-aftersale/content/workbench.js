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
  let logisticsRaw = (logisticsCell?.innerText || '')
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
    .join('\n')
  const rowText = tr.innerText || ''
  if (!logisticsRaw.includes('已退回') && rowText.includes('已退回')) {
    const extra = pick(rowText, /(订单发货[^\n]*已退回|[^\n]*已退回)/) || '已退回'
    logisticsRaw = [logisticsRaw, extra].filter(Boolean).join('\n')
  }

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
    orderInfo: orderText,
    aftersaleInfo: afterText,
    rawJson: JSON.stringify({
      headerTags: headerInfo.tags,
      product: textOf(productCell).slice(0, 400),
      order: orderText.slice(0, 800),
      after: afterText.slice(0, 800),
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

function workbenchTable() {
  for (const table of queryAnyAll('table')) {
    const t = textOf(table)
    if (t.includes('物流信息') && (t.includes('售后编号') || t.includes('商品信息'))) return table
  }
  return $('table')
}

function nextDataRow(trs, headerIndex) {
  for (let j = headerIndex + 1; j < trs.length && j <= headerIndex + 3; j++) {
    const tr = trs[j]
    const txt = tr.innerText || ''
    if (/empty/i.test(tr.className) || !txt.trim()) continue
    if (txt.includes('售后编号') && txt.includes('订单编号')) break
    if (tr.children.length >= 5 && /应付金额|售后退款|商品/.test(txt)) return tr
  }
  return null
}

function collectVisibleTickets() {
  const table = workbenchTable()
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
    const data = nextDataRow(trs, i)
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
  await sleep(280)
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

function cardTypeKeyword(label) {
  const t = String(label || '')
  if (/已发货退款/.test(t)) return '已发货退款'
  if (/未发货退款/.test(t)) return '未发货退款'
  if (/退货退款/.test(t)) return '退货退款'
  if (/仅退款/.test(t)) return '仅退款'
  if (/换货/.test(t)) return '换货'
  if (/补寄/.test(t)) return '补寄'
  if (/维修/.test(t)) return '维修'
  return ''
}

function rowsMatchCardType(card, rows) {
  const label = String(card?.cardLabel || '')
  if (!label || /全部|紧急|临期|催|投诉|重复/.test(label)) return true
  const type = cardTypeKeyword(label)
  if (!type) return true
  const list = rows || []
  if (!list.length) return false
  const hit = list.filter((r) => `${r.aftersaleType || ''} ${r.tags || ''}`.includes(type))
  return hit.length >= Math.ceil(list.length * 0.6)
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
  if (!rowsMatchCardType(card, rows)) return null
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
  if (parseListTotal() !== card.count) return false
  return rowsMatchCardType(card, collectVisibleTickets())
}

async function ensureCardSelected(card) {
  const live = liveCard(card)
  if (isCardSelected(live.el)) return
  await clickCard(live.el)
}

async function collectCardTickets(card) {
  const matched = () => (listBelongsToCard(card) ? true : null)
  for (let i = 0; i < 4 && !matched(); i++) {
    await ensureCardSelected(card)
    await waitUntil(matched, 50, 200)
    if (matched()) break
    const live = liveCard(card)
    if (isCardSelected(live.el)) await sleep(400)
    else await clickCard(live.el)
  }
  if (!matched()) return emptyCardResult()
  await setLargestPageSize(card.count)
  if (!matched()) {
    await ensureCardSelected(card)
    await waitUntil(matched, 50, 200)
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
    if (!rowsMatchCardType(card, rows)) return
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

function isReturnedOutbound(ticket) {
  return String(ticket?.logistics || '').includes('已退回')
}

function matchLogisticsKeyword(s) {
  const t = String(s || '')
  if (t.includes('已退回')) return '已退回'
  if (t.includes('已取消')) return '已取消'
  if (t.includes('待取件')) return '待取件'
  if (t.includes('已签收')) return '已签收'
  if (t.includes('运输中')) return '运输中'
  if (t.includes('已发货')) return '已发货'
  return ''
}

function logisticsStatusOf(text, tracks) {
  if (String(text || '').includes('已取消')) return '已取消'
  const latest = tracks?.[0]?.title || tracks?.[0]?.text || ''
  const fromLatest = matchLogisticsKeyword(latest)
  if (fromLatest && fromLatest !== '已发货') return fromLatest
  return matchLogisticsKeyword([latest, text, ...(tracks || []).map((x) => x.text || x.title)].join('\n'))
}

function hasShipStatus(text) {
  return /\d+\s*\/\s*\d+\s*已/.test(String(text || ''))
}

function logisticsFilledCount() {
  return collectVisibleTickets().filter((t) => hasShipStatus(t.logistics)).length
}

function tableScroller() {
  const table = workbenchTable()
  if (!table) return null
  return (
    table.closest('.aurora-table-body, .auxo-table-body') ||
    table.parentElement?.querySelector('.aurora-table-body, .auxo-table-body') ||
    table.closest('[class*="table-body"]') ||
    table.parentElement
  )
}

async function scrollWorkbenchTable() {
  const scroller = tableScroller()
  if (!scroller) return
  const view = scroller.clientHeight || 400
  const max = Math.max(scroller.scrollHeight || 0, workbenchTable()?.scrollHeight || 0)
  for (let y = 0; y <= max + view; y += Math.max(180, Math.floor(view * 0.65))) {
    scroller.scrollTop = y
    await sleep(400)
  }
  scroller.scrollTop = 0
  await sleep(600)
}

function returnedVisibleCount() {
  return Math.max(
    collectVisibleTickets().filter(isReturnedOutbound).length,
    countVisibleReturnedRows(),
  )
}

async function ensureReturnPageSize() {
  for (let i = 0; i < 3 && currentPageSize() > 10; i++) {
    await applyPageSize(10)
    if (await waitUntil(() => (currentPageSize() <= 10 ? true : null), 20, 150)) break
  }
  await waitListSettled()
  return currentPageSize() || 10
}

async function waitReturnListReady(expected, opts = {}) {
  const pageSize = currentPageSize() || 10
  const needRows = expected > 0 ? Math.min(expected, pageSize) : 0
  await waitUntil(() => {
    const rows = collectVisibleTickets()
    if (needRows && rows.length < needRows) return null
    const filled = logisticsFilledCount()
    if (rows.length && filled < Math.ceil(rows.length * 0.85)) return null
    return true
  }, 40, 400)
  await sleep(400)
  await scrollWorkbenchTable()
  const deadline = Date.now() + (opts.maxWait || 12000)
  let last = -1
  let stable = 0
  while (Date.now() < deadline) {
    const n = returnedVisibleCount()
    if (n > 0) {
      if (n === last) stable += 1
      else stable = 0
      last = n
      if (stable >= 2) return n
    }
    await sleep(600)
  }
  return returnedVisibleCount()
}

function ticketListFingerprint() {
  const rows = collectVisibleTickets()
  return {
    total: parseListTotal(),
    ids: rows.map((t) => t.platformAftersaleId).filter(Boolean).join('|'),
    visible: rows.length,
    returned: rows.filter(isReturnedOutbound).length,
  }
}

function countVisibleReturnedRows() {
  const table = workbenchTable()
  if (!table) return 0
  return Array.from(table.querySelectorAll('tr')).filter((tr) => {
    const t = tr.innerText || ''
    return t.includes('已退回') && (t.includes('订单发货') || t.includes('物流'))
  }).length
}

function closeOpenSelects() {
  document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
  document.dispatchEvent(new KeyboardEvent('keyup', { key: 'Escape', bubbles: true }))
}

async function waitListSettled() {
  let last = `${parseListTotal()}|${visibleHeaderCount()}`
  for (let i = 0; i < 12; i++) {
    await sleep(400)
    const cur = `${parseListTotal()}|${visibleHeaderCount()}`
    if (cur === last && visibleHeaderCount() >= 0) return parseListTotal()
    last = cur
  }
  return parseListTotal()
}

function findButton(re) {
  for (const el of queryAnyAll('button')) {
    if (re.test(textOf(el)) && isShown(el)) return el
  }
  return null
}

function findSelectByLabel(label) {
  const wraps = queryAnyAll('[class*="labelWrapper"], label, .auxo-form-item, .aurora-form-item')
  for (const w of wraps) {
    if (!textOf(w).includes(label)) continue
    let p = w
    for (let i = 0; i < 6 && p; i++) {
      const sel = p.querySelector('.auxo-select, .aurora-select')
      if (sel) return sel
      p = p.parentElement
    }
  }
  return null
}

function currentSelectValue(label) {
  const sel = findSelectByLabel(label)
  if (!sel) return ''
  return textOf(
    sel.querySelector('.auxo-select-selection-item, .aurora-select-content-value, [class*="selection-item"]'),
  )
}

async function pickSelectByLabel(label, want) {
  const sel = findSelectByLabel(label)
  if (!sel) return false
  if (currentSelectValue(label) === want) return true
  fireClick(sel.querySelector('.auxo-select-selector, .aurora-select-content, .aurora-select-selector') || sel)
  const opt = await waitUntil(() => {
    let hit = null
    for (const el of queryAnyAll(
      '.auxo-select-item, .auxo-select-item-option, .aurora-select-item, .aurora-select-item-option, [class*="select-item"]',
    )) {
      if (!isShown(el)) continue
      const t = textOf(el)
      if (t === want || t.includes(want)) hit = el
    }
    return hit
  }, 20, 150)
  if (opt) fireClick(opt)
  await sleep(200)
  return currentSelectValue(label) === want || currentSelectValue(label).includes(want)
}

async function clearSelectedCards() {
  for (const card of parseCards()) {
    if (!isCardSelected(card.el)) continue
    fireClick(card.el)
    await sleep(250)
  }
}

function closeLogisticsDrawer() {
  const close =
    queryAny('.auxo-drawer-open .auxo-drawer-close, .aurora-drawer-open .aurora-drawer-close') ||
    queryAny('.auxo-drawer-open [class*="drawer-close"], .aurora-drawer-open [class*="drawer-close"]')
  if (close) fireClick(close)
}

function cleanTrackingNo(val) {
  const s = String(val || '').replace(/\s+/g, '')
  if (!s || s.length < 8 || s.length > 32) return ''
  if (!/^[A-Za-z0-9-]+$/.test(s)) return ''
  if (/售后|订单|编号/.test(s)) return ''
  return s
}

function trackingNoFromText(text) {
  const t = String(text || '')
  return (
    cleanTrackingNo(pick(t, /(?:退货|发货|换货)?物流单号[:：\s]*([A-Za-z0-9-]{8,32})/)) ||
    cleanTrackingNo(pick(t, /(?:运单号|快递单号)[:：\s]*([A-Za-z0-9-]{8,32})/))
  )
}

function logisticsNoFromRecord(record) {
  if (!record || typeof record !== 'object') return ''
  const seen = new Set()
  const codes = []
  const walk = (v, depth) => {
    if (!v || depth > 5 || seen.has(v)) return
    if (typeof v === 'object') seen.add(v)
    if (Array.isArray(v)) {
      v.forEach((x) => walk(x, depth + 1))
      return
    }
    if (typeof v !== 'object') return
    for (const [k, val] of Object.entries(v)) {
      if (/logistics_code|tracking_no|waybill/i.test(k) && typeof val === 'string') {
        const code = cleanTrackingNo(val)
        if (code) codes.push(code)
      } else if (val && typeof val === 'object') {
        walk(val, depth + 1)
      }
    }
  }
  walk(record, 0)
  return codes[0] || ''
}

function parseLogisticsDrawer() {
  const body = queryAny('.auxo-drawer-open .auxo-drawer-body, .aurora-drawer-open .aurora-drawer-body')
  if (!body) return null
  const info = {}
  for (const lab of body.querySelectorAll('[class*="base-info-label"], [class*="label"]')) {
    const key = textOf(lab)
    const val = textOf(lab.nextElementSibling || lab.parentElement).replace(key, '').trim()
    if (/物流单号|运单号|快递单号/.test(key) && !info.logisticsNo) {
      info.logisticsNo = cleanTrackingNo(val)
    } else if (/物流商|快递公司/.test(key) && !info.carrier) {
      info.carrier = val
    } else if (key.includes('发货时间') && !info.shipTime) {
      info.shipTime = val
    }
  }
  const tracks = []
  for (const el of body.querySelectorAll('.auxo-timeline-item, .aurora-timeline-item')) {
    const date = textOf(el.querySelector('[class*="date"]'))
    const title = textOf(el.querySelector('[class*="itemTitle"]'))
    const detail = textOf(el.querySelector('[class*="itemDetail"]'))
    const text = [date, title, detail].filter(Boolean).join(' ')
    if (text) tracks.push({ date, title, detail, text })
  }
  if (!info.logisticsNo) {
    const clone = body.cloneNode(true)
    clone.querySelectorAll('style, script').forEach((n) => n.remove())
    const blob = textOf(clone)
    info.logisticsNo = trackingNoFromText(blob)
    if (!info.carrier) info.carrier = pick(blob, /(?:发货物流商|物流商|快递公司)\s*([^\s发退换]+)/)
    if (!info.shipTime) info.shipTime = pick(blob, /发货时间\s*([\d/]+(?:\s*[\d:]+)?)/)
  }
  return { ...info, tracks }
}

function secondLastTrack(tracks) {
  if (!tracks?.length) return ''
  if (tracks.length === 1) return tracks[0].text || ''
  // 抽屉轨迹为新→旧；退回地取时间正序倒数第二条（展示列表第 2 条）
  return tracks[1].text || ''
}

function returnTimeFromTracks(tracks, applyTime, shipTime) {
  const hit =
    (tracks || []).find((t) => t.title === '已退回' || String(t.text || '').includes('已退回')) ||
    tracks?.[0]
  const date = String(hit?.date || '').trim()
  if (!date) return ''
  if (/^\d{4}/.test(date)) return date.replace(/-/g, '/')
  const year = pick(applyTime, /^(\d{4})/) || pick(shipTime, /^(\d{4})/) || String(new Date().getFullYear())
  return `${year}/${date}`
}

function findDataRowByAftersaleId(aftersaleId) {
  const table = workbenchTable()
  if (!table) return null
  const headers = headerCells(table)
  const trs = Array.from(table.querySelectorAll('tbody tr, tr'))
  for (let i = 0; i < trs.length; i++) {
    const tr = trs[i]
    const txt = tr.innerText || ''
    const isHeader =
      txt.includes('售后编号') &&
      txt.includes('订单编号') &&
      (tr.className.includes('level-0') || tr.children.length <= 3)
    if (!isHeader) continue
    const headerInfo = parseHeaderRow(tr)
    if (headerInfo.platformAftersaleId !== String(aftersaleId)) continue
    const data = nextDataRow(trs, i)
    return { header: tr, data, headers }
  }
  return null
}

async function readReturnLogistics(ticket) {
  closeLogisticsDrawer()
  await sleep(120)
  const pair = findDataRowByAftersaleId(ticket.platformAftersaleId)
  const cell = pair ? cellByName(pair.data, ['物流信息'], pair.headers) : null
  const fallbackNo =
    cleanTrackingNo(ticket.returnLogisticsNo) ||
    trackingNoFromText(ticket.logistics) ||
    trackingNoFromText(cell?.innerText) ||
    logisticsNoFromRecord(findTableRecord(cell || pair?.data || pair?.header))
  const clickEl =
    cell?.querySelector('[class*="clickable"], [class*="logisticsText"]') || cell
  if (!clickEl) {
    return { logisticsNo: fallbackNo, carrier: '', shipTime: '', returnLocation: '', tracks: [] }
  }
  fireClick(clickEl)
  const parsed = await waitUntil(() => {
    const d = parseLogisticsDrawer()
    if (d && (d.logisticsNo || (d.tracks && d.tracks.length))) return d
    return null
  }, 25, 200)
  closeLogisticsDrawer()
  await waitUntil(() => (parseLogisticsDrawer() ? null : true), 15, 120)
  const tracks = parsed?.tracks || []
  return {
    logisticsNo: parsed?.logisticsNo || fallbackNo,
    carrier: parsed?.carrier || '',
    shipTime: parsed?.shipTime || '',
    returnLocation: secondLastTrack(tracks),
    tracks,
  }
}

async function resetAftersaleFilters() {
  const prevTotal = parseListTotal()
  const hadCard = parseCards().some((c) => isCardSelected(c.el))
  const reset = findButton(/重\s*置/)
  if (reset) {
    fireClick(reset)
    await waitUntil(() => {
      if (parseCards().some((c) => isCardSelected(c.el))) return null
      const t = parseListTotal()
      if (t == null) return null
      if (hadCard && prevTotal != null && t === prevTotal) return null
      return true
    }, 40, 200)
    await waitListSettled()
    await sleep(800)
    return
  }
  await pickSelectByLabel('售后类型', '全部')
  await pickSelectByLabel('售后状态', '全部')
  closeOpenSelects()
  const query = findButton(/查\s*询/)
  if (query) fireClick(query)
  await waitListSettled()
}

function shippedRefundHeaderCount() {
  const table = workbenchTable()
  if (!table) return { total: 0, shipped: 0 }
  let total = 0
  let shipped = 0
  for (const tr of table.querySelectorAll('tr')) {
    const t = tr.innerText || ''
    if (!t.includes('售后编号') || !t.includes('订单编号')) continue
    total += 1
    if (t.includes('已发货退款')) shipped += 1
  }
  return { total, shipped }
}

function isShippedRefundListReady() {
  if (!currentSelectValue('售后类型').includes('已发货退款')) return false
  if (!currentSelectValue('售后状态').includes('退款成功')) return false
  const now = ticketListFingerprint()
  if (now.total == null || now.visible === 0) return false
  const headers = shippedRefundHeaderCount()
  if (headers.total === 0) return false
  return headers.shipped >= Math.ceil(headers.total * 0.7)
}

async function applyReturnFilters() {
  if (parseCards().some((c) => isCardSelected(c.el))) {
    await resetAftersaleFilters()
  }
  const before = ticketListFingerprint()
  const typeOk = await pickSelectByLabel('售后类型', '已发货退款')
  await sleep(300)
  const statusOk = await pickSelectByLabel('售后状态', '退款成功')
  await sleep(300)
  if (!typeOk || !statusOk) {
    throw new Error('无法设置售后类型/售后状态筛选')
  }
  closeOpenSelects()
  await waitUntil(() => {
    const open = queryAnyAll('.auxo-select-dropdown, .aurora-select-dropdown').some(isShown)
    return open ? null : true
  }, 15, 150)
  const query = findButton(/查\s*询/)
  if (!query) throw new Error('未找到查询按钮')
  fireClick(query)
  const ready = await waitUntil(() => {
    if (!isShippedRefundListReady()) return null
    const now = ticketListFingerprint()
    if (before.ids && now.ids === before.ids && now.total === before.total && (before.total || 0) <= 20) {
      return null
    }
    return now
  }, 50, 400)
  if (!ready) {
    fireClick(query)
    const retry = await waitUntil(() => (isShippedRefundListReady() ? ticketListFingerprint() : null), 40, 400)
    if (!retry) throw new Error('退回件筛选未刷新列表')
  }
  await waitListSettled()
}

function toReturnItem(ticket, extra) {
  const tracks = extra.tracks || []
  return {
    platformAftersaleId: ticket.platformAftersaleId,
    orderNo: ticket.orderNo,
    productTitle: ticket.productTitle,
    productImage: ticket.productImage,
    sku: ticket.sku,
    qty: ticket.qty,
    buyQty: ticket.buyQty || 0,
    payAmount: ticket.payAmount,
    refundAmount: ticket.refundAmount,
    aftersaleType: ticket.aftersaleType || '已发货退款',
    reason: ticket.reason,
    status: ticket.status,
    orderInfo: ticket.orderInfo || '',
    aftersaleInfo: ticket.aftersaleInfo || '',
    logistics: ticket.logistics,
    logisticsNo: extra.logisticsNo || '',
    carrier: extra.carrier || '',
    returnLocation: extra.returnLocation || '',
    shipTime: extra.shipTime || '',
    applyTime: ticket.applyTime,
    returnTime: returnTimeFromTracks(tracks, ticket.applyTime, extra.shipTime),
    trackJson: JSON.stringify(tracks),
    rawJson: JSON.stringify({
      orderInfo: ticket.orderInfo || '',
      aftersaleInfo: ticket.aftersaleInfo || '',
      logistics: ticket.logistics,
      logisticsNo: extra.logisticsNo || '',
      carrier: extra.carrier || '',
      returnLocation: extra.returnLocation || '',
      tracks: tracks.slice(0, 8),
    }),
  }
}

function toShippedRefundItem(ticket, extra) {
  const tracks = (extra?.tracks || []).slice(0, 5)
  return {
    platformAftersaleId: ticket.platformAftersaleId,
    orderNo: ticket.orderNo,
    productTitle: ticket.productTitle,
    productImage: ticket.productImage,
    sku: ticket.sku,
    productTags: ticket.productTags || '',
    tags: ticket.tags || '',
    qty: ticket.qty,
    buyQty: ticket.buyQty || 0,
    payAmount: ticket.payAmount,
    refundAmount: ticket.refundAmount,
    aftersaleType: ticket.aftersaleType || '已发货退款',
    reason: ticket.reason,
    status: ticket.status,
    orderInfo: ticket.orderInfo || '',
    aftersaleInfo: ticket.aftersaleInfo || '',
    logistics: ticket.logistics,
    logisticsStatus: logisticsStatusOf(ticket.logistics, tracks),
    logisticsNo: extra?.logisticsNo || '',
    carrier: extra?.carrier || '',
    shipTime: extra?.shipTime || '',
    trackJson: JSON.stringify(tracks),
    applyTime: ticket.applyTime,
    rawJson: JSON.stringify({
      orderInfo: ticket.orderInfo || '',
      aftersaleInfo: ticket.aftersaleInfo || '',
      logistics: ticket.logistics,
      logisticsNo: extra?.logisticsNo || '',
      productTags: ticket.productTags || '',
      tags: ticket.tags || '',
      tracks,
    }),
  }
}

async function collectVisibleRefundBuckets(seenReturned, seenShipped) {
  const returns = []
  const shipped = []
  for (const t of collectVisibleTickets()) {
    if (!t.platformAftersaleId) continue
    if (isReturnedOutbound(t)) {
      if (seenReturned.has(t.platformAftersaleId)) continue
      seenReturned.add(t.platformAftersaleId)
      let extra = { logisticsNo: '', carrier: '', shipTime: '', returnLocation: '', tracks: [] }
      try {
        extra = await readReturnLogistics(t)
      } catch {
        /* keep row without track */
      }
      returns.push(toReturnItem(t, extra))
      continue
    }
    if (seenShipped.has(t.platformAftersaleId)) continue
    seenShipped.add(t.platformAftersaleId)
    let extra = { logisticsNo: '', carrier: '', shipTime: '', returnLocation: '', tracks: [] }
    try {
      extra = await readReturnLogistics(t)
    } catch {
      /* keep row without track */
    }
    shipped.push(toShippedRefundItem(t, extra))
  }
  return { returns, shipped }
}

async function collectReturnPackages() {
  locateWorkbenchPage()
  closeLogisticsDrawer()
  await sleep(800)
  await applyReturnFilters()
  await ensureReturnPageSize()
  const expected = parseListTotal() ?? 0
  if (expected > 0) {
    await ensureFirstPage()
    await waitListSettled()
    await waitReturnListReady(expected, { maxWait: 15000 })
    await sleep(600)
  }
  const pageSize = currentPageSize() || 10
  const pageCount = expected > 0 ? Math.max(1, Math.ceil(expected / pageSize)) : 1
  const seenReturned = new Set()
  const seenShipped = new Set()
  const seenTickets = new Set()
  const items = []
  const shippedItems = []
  let pages = 0

  async function collectCurrentPage() {
    for (const t of collectVisibleTickets()) {
      if (t.platformAftersaleId) seenTickets.add(t.platformAftersaleId)
    }
    const bucket = await collectVisibleRefundBuckets(seenReturned, seenShipped)
    items.push(...bucket.returns)
    shippedItems.push(...bucket.shipped)
  }

  for (let p = 1; p <= Math.min(pageCount, 30); p++) {
    if (p > 1) {
      const prevIds = collectVisibleTickets().map((r) => r.platformAftersaleId)
      const moved = await goToPage(p)
      if (!moved) {
        const next = findNextPage()
        if (!next) break
        fireClick(next)
        await waitForPageChange(prevIds)
      }
      await waitListSettled()
      await waitReturnListReady(expected, { maxWait: 8000 })
    }
    pages++
    await collectCurrentPage()
    if (!findNextPage()) break
  }
  while (expected && seenTickets.size < expected && findNextPage() && pages < 30) {
    const prevIds = collectVisibleTickets().map((r) => r.platformAftersaleId)
    const next = findNextPage()
    fireClick(next)
    await waitForPageChange(prevIds)
    await waitListSettled()
    await waitReturnListReady(expected, { maxWait: 8000 })
    pages++
    await collectCurrentPage()
  }
  if (!items.length && expected >= 5) {
    await scrollWorkbenchTable()
    await sleep(1500)
    await collectCurrentPage()
  }
  const leftoverReturned = countVisibleReturnedRows()
  if (!items.length && leftoverReturned) {
    throw new Error(`页面有 ${leftoverReturned} 条已退回，但未解析到售后单`)
  }
  if (expected >= 10 && !items.length && !shippedItems.length) {
    throw new Error(`已发货退款/退款成功 ${expected} 条已出现，但列表未采到明细`)
  }
  return {
    items,
    shippedItems,
    stats: {
      filteredTotal: expected,
      scanned: seenTickets.size,
      pages,
      pageSize,
      pageCount,
      returned: items.length,
      shipped: shippedItems.length,
      withNo: items.filter((x) => x.logisticsNo).length + shippedItems.filter((x) => x.logisticsNo).length,
    },
  }
}

async function collectAll() {
  locateWorkbenchPage()
  const cardRows = parseCards()
  if (!cardRows.length) {
    throw new Error('未找到快捷筛选卡片，请确认当前是售后工作台列表页')
  }
  let returns
  let shippedRefunds
  let returnStats
  try {
    const collected = await collectReturnPackages()
    returnStats = collected.stats
    if (collected.items?.length) returns = collected.items
    if (collected.shippedItems?.length) shippedRefunds = collected.shippedItems
  } catch (e) {
    returnStats = { error: e instanceof Error ? e.message : String(e) }
  }
  await resetAftersaleFilters()
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
  await resetAftersaleFilters()
  const meta = shopMeta()
  return {
    ok: true,
    ...meta,
    cards: cardRows.map(({ el, ...rest }) => rest),
    tickets: Array.from(ticketMap.values()),
    cardStats,
    returns,
    shippedRefunds,
    returnStats,
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
