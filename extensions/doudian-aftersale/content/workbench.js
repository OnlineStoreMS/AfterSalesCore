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
  const typeHit = lines.find((l) => /已发货退款|未发货退款|退货退款|仅退款|换货|补寄|维修/.test(l))
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
    logistics: formatTicketLogistics(logisticsRaw),
    returnLogisticsNo,
    shipLogisticsNo:
      logisticsRaw.includes('订单发货') || needsIntercept(logisticsRaw)
        ? shipLogisticsNoFromRecord(findTableRecord(logisticsCell) || findTableRecord(tr))
        : '',
    applyTime: pick(afterText, /申请时间\s*([\d/]+(?:\s*[\d:]+)?)/),
    orderInfo: orderText,
    aftersaleInfo: afterText,
    rawJson: JSON.stringify({
      headerTags: headerInfo.tags,
      product: textOf(productCell).slice(0, 400),
      order: orderText.slice(0, 800),
      after: afterText.slice(0, 800),
      status: statusText.slice(0, 120),
      logistics: formatTicketLogistics(logisticsRaw).slice(0, 200),
      returnLogisticsNo,
      shipLogisticsNo:
        logisticsRaw.includes('订单发货') || needsIntercept(logisticsRaw)
          ? shipLogisticsNoFromRecord(findTableRecord(logisticsCell) || findTableRecord(tr))
          : '',
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
  const rawHit = logisticsMap[ticket.platformAftersaleId]
  let returnNo = String(ticket.returnLogisticsNo || '').trim()
  let shipNo = String(ticket.shipLogisticsNo || '').trim()
  if (typeof rawHit === 'string') {
    returnNo = returnNo || rawHit.trim()
  } else if (rawHit && typeof rawHit === 'object') {
    returnNo = returnNo || String(rawHit.returnLogisticsNo || rawHit.return || '').trim()
    shipNo = shipNo || String(rawHit.shipLogisticsNo || rawHit.ship || '').trim()
  }
  if (!returnNo && !shipNo) return ticket
  let raw = ticket.rawJson
  try {
    const o = JSON.parse(raw || '{}')
    if (returnNo) o.returnLogisticsNo = returnNo
    if (shipNo) o.shipLogisticsNo = shipNo
    raw = JSON.stringify(o)
  } catch {
    /* ignore */
  }
  return {
    ...ticket,
    returnLogisticsNo: returnNo || ticket.returnLogisticsNo || '',
    shipLogisticsNo: shipNo || ticket.shipLogisticsNo || '',
    rawJson: raw,
  }
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
  for (let j = headerIndex + 1; j < trs.length && j <= headerIndex + 5; j++) {
    const tr = trs[j]
    const txt = tr.innerText || ''
    if (/empty/i.test(tr.className) || !txt.trim()) continue
    if (txt.includes('售后编号') && txt.includes('订单编号')) break
    if (tr.children.length >= 5 && /应付金额|售后退款|商品/.test(txt)) return tr
    if (tr.children.length >= 5 && !txt.includes('售后编号')) return tr
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
    let parsed = null
    if (data && data.children.length >= 5) {
      parsed = parseDataRow(data, headers, headerInfo)
    }
    if (!parsed?.platformAftersaleId) {
      parsed = {
        platformAftersaleId: headerInfo.platformAftersaleId,
        orderNo: headerInfo.orderNo,
        aftersaleType: headerInfo.aftersaleType,
        tags: headerInfo.tags,
        productTitle: '',
        productImage: '',
        sku: '',
        productTags: '',
        qty: 0,
        buyQty: 0,
        payAmount: '',
        refundAmount: '',
        reason: '',
        status: '',
        timeoutText: '',
        dispute: '',
        logistics: '',
        returnLogisticsNo: '',
        shipLogisticsNo: '',
        applyTime: '',
        orderInfo: '',
        aftersaleInfo: '',
        rawJson: JSON.stringify({ headerOnly: true, tags: headerInfo.tags }),
      }
    }
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

function cardTypeAliases(type) {
  if (type === '已发货退款') return ['已发货退款', '仅退款']
  if (type === '未发货退款') return ['未发货退款', '仅退款']
  return [type]
}

function rowTypeBlob(row) {
  return `${row?.aftersaleType || ''} ${row?.tags || ''} ${row?.aftersaleInfo || ''}`
}

function rowConflictsCardType(blob, type) {
  if (type === '已发货退款') return /换货|补寄|维修|未发货退款/.test(blob)
  if (type === '未发货退款') return /换货|补寄|维修|已发货退款|退货退款/.test(blob)
  if (type === '换货') return /已发货退款|未发货退款|仅退款|退货退款|补寄|维修/.test(blob)
  return false
}

function rowMatchesCardType(row, type) {
  const blob = rowTypeBlob(row)
  if (rowConflictsCardType(blob, type)) return false
  if (cardTypeAliases(type).some((a) => blob.includes(a))) return true
  return type === '已发货退款' || type === '未发货退款'
}

function rowsMatchCardType(card, rows) {
  const label = String(card?.cardLabel || '')
  if (!label || /全部|紧急|临期|催|投诉|重复/.test(label)) return true
  const type = cardTypeKeyword(label)
  if (!type) return true
  const list = rows || []
  if (!list.length) return false
  const hit = list.filter((r) => rowMatchesCardType(r, type))
  return hit.length >= Math.ceil(list.length * 0.6)
}

function ticketIds(rows) {
  return (rows || []).map((r) => r.platformAftersaleId).filter(Boolean)
}

function emptyPrevCard() {
  return {
    cardKey: '',
    groupName: '',
    cardLabel: '',
    type: '',
    count: null,
    visibleIds: [],
    collectedIds: [],
  }
}

function isAggregateCard(card) {
  return /全部|紧急|临期|催|投诉|重复/.test(String(card?.cardLabel || ''))
}

function cardKind(card) {
  const type = cardTypeKeyword(card?.cardLabel)
  if (type) return 'type'
  if (isAggregateCard(card)) return 'aggregate'
  return 'status'
}

function sameGroupSameCount(prev, card) {
  return prev?.groupName && prev.groupName === card.groupName && prev.count === card.count
}

function expectNewIds(prev, card) {
  if (!prev?.cardKey || prev.cardKey === card.cardKey) return false
  const a = cardKind(prev)
  const b = cardKind(card)
  if (a === 'type' && b === 'type' && prev.type !== cardTypeKeyword(card.cardLabel)) return true
  if (a === 'aggregate' && b === 'type') return true
  if (a === 'type' && b === 'aggregate') return true
  if (a === 'status' && b === 'type') return true
  if (a === 'type' && b === 'status') return true
  if (a === 'aggregate' && b === 'status' && sameGroupSameCount(prev, card)) return false
  if (a === 'status' && b === 'aggregate' && sameGroupSameCount(prev, card)) return false
  if (a === 'status' && b === 'status' && sameGroupSameCount(prev, card)) return false
  return prev.count !== card.count || prev.cardKey !== card.cardKey
}

function rowsHaveUniqueType(card, rows) {
  const type = cardTypeKeyword(card?.cardLabel)
  if (!type) return true
  const list = rows || []
  if (!list.length) return false
  const hit = list.filter((r) => {
    const blob = rowTypeBlob(r)
    return blob.includes(type)
  })
  return hit.length >= Math.ceil(list.length * 0.6)
}

function stillStaleList(card, ctx, rows) {
  const prev = ctx?.prev || emptyPrevCard()
  const ids = ticketIds(rows)
  if (!ids.length) return false
  const type = cardTypeKeyword(card.cardLabel)
  const prevVisible = ctx?.beforeIds?.length ? ctx.beforeIds : prev.visibleIds || []
  const prevCollected = prev.collectedIds || []

  if (prev.type && type && prev.type !== type && prevCollected.length) {
    if (ids.every((id) => prevCollected.includes(id))) return true
  }
  if (!expectNewIds(prev, card)) return false
  if (prevVisible.length && ids.every((id) => prevVisible.includes(id))) {
    if (prev.type && type && prev.type !== type) return true
    if (rowsHaveUniqueType(card, rows)) return false
    if (ctx?.sawTotalChange && !prev.type) return false
    return true
  }
  return false
}

function cardTableState(card, ctx) {
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
  if (stillStaleList(card, ctx, rows)) return null
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

function listBelongsToCard(card, ctx) {
  const live = liveCard(card)
  if (!isCardSelected(live.el)) return false
  if (parseListTotal() !== card.count) return false
  const rows = collectVisibleTickets()
  if (stillStaleList(card, ctx, rows)) return false
  return rowsMatchCardType(card, rows)
}

async function ensureCardSelected(card) {
  const live = liveCard(card)
  if (isCardSelected(live.el)) return
  await clickCard(live.el)
}

function groupAggregateCard(card) {
  return parseCards().find((c) => c.groupName === card.groupName && isAggregateCard(c) && c.count > 0)
}

async function clearSelectedInGroup(card) {
  for (const c of parseCards()) {
    if (c.groupName !== card.groupName || !isCardSelected(c.el)) continue
    await clickCard(c.el)
    await sleep(200)
  }
  await waitUntil(() => {
    const selected = parseCards().some((c) => c.groupName === card.groupName && isCardSelected(c.el))
    return selected ? null : true
  }, 20, 150)
}

async function switchToCard(card, ctx) {
  if (expectNewIds(ctx.prev, card)) {
    await clearSelectedInGroup(card)
    await waitUntil(() => {
      const t = parseListTotal()
      if (t == null) return null
      if (ctx.prev.count != null && ctx.prev.count !== card.count && t === ctx.prev.count) return null
      return true
    }, 40, 200)
    await sleep(400)
  }
  const live = liveCard(card)
  if (!isCardSelected(live.el)) await clickCard(live.el)
}

async function collectCardTickets(card, prev = emptyPrevCard()) {
  const beforeIds = ticketIds(collectVisibleTickets())
  const beforeTotal = parseListTotal()
  const ctx = {
    prev,
    beforeIds,
    beforeTotal,
    sawTotalChange: beforeTotal != null && beforeTotal !== card.count,
  }
  const matched = () => {
    const total = parseListTotal()
    if (beforeTotal != null && total != null && total !== beforeTotal) ctx.sawTotalChange = true
    return listBelongsToCard(card, ctx) ? true : null
  }
  await switchToCard(card, ctx)
  await waitUntil(matched, 50, 200)
  if (!matched()) {
    await switchToCard(card, ctx)
    await waitUntil(matched, 40, 200)
  }
  if (!matched()) {
    const agg = groupAggregateCard(card)
    if (agg && agg.cardKey !== card.cardKey) {
      await clickCard(liveCard(agg).el)
      await waitUntil(() => {
        const t = parseListTotal()
        return t != null && t !== card.count ? true : null
      }, 40, 200)
      await sleep(400)
      await clickCard(liveCard(card).el)
      await waitUntil(matched, 50, 200)
    }
  }
  if (!matched()) return emptyCardResult()
  await setLargestPageSize(card.count)
  if (!matched()) {
    await ensureCardSelected(card)
    await waitUntil(matched, 50, 200)
  }
  if (!matched()) return emptyCardResult()
  await ensureFirstPage()
  await waitUntil(() => cardTableState(card, ctx), 50, 200)
  if (!listBelongsToCard(card, ctx)) return emptyCardResult()
  const all = []
  const seen = new Set()
  const expected = card.count || 0
  const add = (rows) => {
    if (!listBelongsToCard(card, ctx)) return
    if (stillStaleList(card, ctx, rows)) return
    if (!rowsMatchCardType(card, rows)) return
    if ((rows || []).length > expected) return
    for (const r of rows || []) {
      if (!r.platformAftersaleId || seen.has(r.platformAftersaleId)) continue
      if (expected && all.length >= expected) break
      seen.add(r.platformAftersaleId)
      all.push(r)
    }
  }
  add(await enrichAllTicketLogistics(await visibleTicketsWithLogistics()))
  if (expected && all.length < expected) {
    await setLargestPageSize(expected)
    if (listBelongsToCard(card, ctx)) add(await enrichAllTicketLogistics(await visibleTicketsWithLogistics()))
  }
  for (let p = 2; expected && all.length < expected && p <= 20; p++) {
    if (!listBelongsToCard(card, ctx)) break
    const prevIds = all.map((r) => r.platformAftersaleId)
    const moved = await goToPage(p)
    if (!moved) {
      const next = findNextPage()
      if (!next) break
      fireClick(next)
      await waitForPageChange(prevIds)
    }
    add(await enrichAllTicketLogistics(await visibleTicketsWithLogistics()))
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

function needsIntercept(text) {
  const t = String(text || '')
  return t.includes('订单发货') && t.includes('需商家拦截快递')
}

function chunkAfterLabel(text, label, stops) {
  const src = String(text || '')
  const i = src.indexOf(label)
  if (i < 0) return ''
  let rest = src.slice(i + label.length)
  let cut = rest.length
  for (const stop of stops || []) {
    const j = rest.indexOf(stop)
    if (j >= 0 && j < cut) cut = j
  }
  return rest.slice(0, cut)
}

function formatTicketLogistics(raw) {
  const text = String(raw || '')
  const lines = []
  if (text.includes('买家退货')) {
    const st = matchLogisticsKeyword(chunkAfterLabel(text, '买家退货', ['订单发货', '需商家拦截快递']))
    lines.push(st ? `买家退货 ${st}` : '买家退货')
  }
  if (text.includes('订单发货')) {
    const st = matchLogisticsKeyword(chunkAfterLabel(text, '订单发货', ['买家退货', '需商家拦截快递']))
    lines.push(st ? `订单发货 ${st}` : '订单发货')
  }
  if (text.includes('需商家拦截快递')) lines.push('需商家拦截快递')
  return lines.join('\n') || text
}

function logisticsStatusOf(text, tracks) {
  const raw = String(text || '')
  if (raw.includes('已取消')) return '已取消'
  const ship = matchLogisticsKeyword(chunkAfterLabel(raw, '订单发货', ['买家退货', '需商家拦截快递']))
  if (ship) return ship
  const latest = tracks?.[0]?.title || tracks?.[0]?.text || ''
  const fromLatest = matchLogisticsKeyword(latest)
  if (fromLatest && fromLatest !== '已发货') return fromLatest
  return matchLogisticsKeyword([latest, raw, ...(tracks || []).map((x) => x.text || x.title)].join('\n'))
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

function logisticsCodesFromRecord(record) {
  const items = []
  if (!record || typeof record !== 'object') return items
  const seen = new Set()
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
        if (code) items.push({ key: k, code })
      } else if (val && typeof val === 'object') {
        walk(val, depth + 1)
      }
    }
  }
  walk(record, 0)
  return items
}

function logisticsNoFromRecord(record) {
  return logisticsCodesFromRecord(record)[0]?.code || ''
}

function shipLogisticsNoFromRecord(record) {
  const items = logisticsCodesFromRecord(record)
  const ship = items.find((x) => /send|ship|deliver/i.test(x.key) && !/return|refund/i.test(x.key))
  const generic = items.find((x) => /logistics_code/i.test(x.key) && !/return/i.test(x.key))
  return ship?.code || generic?.code || ''
}

function parseLogisticsDrawer() {
  const body = queryAny('.auxo-drawer-open .auxo-drawer-body, .aurora-drawer-open .aurora-drawer-body')
  if (!body) return null
  const info = {}
  for (const lab of body.querySelectorAll('[class*="base-info-label"], [class*="label"]')) {
    const key = textOf(lab)
    const val = textOf(lab.nextElementSibling || lab.parentElement).replace(key, '').trim()
    if (key.includes('发货物流单号') && !info.shipLogisticsNo) {
      info.shipLogisticsNo = cleanTrackingNo(val)
      if (!info.logisticsNo) info.logisticsNo = info.shipLogisticsNo
    } else if (key.includes('退货物流单号') && !info.returnLogisticsNo) {
      info.returnLogisticsNo = cleanTrackingNo(val)
    } else if (/物流单号|运单号|快递单号/.test(key) && !info.logisticsNo) {
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
    info.shipLogisticsNo = info.shipLogisticsNo || cleanTrackingNo(pick(blob, /发货物流单号\s*([A-Za-z0-9-]{8,32})/))
    info.returnLogisticsNo = info.returnLogisticsNo || cleanTrackingNo(pick(blob, /退货物流单号\s*([A-Za-z0-9-]{8,32})/))
    info.logisticsNo = info.logisticsNo || info.shipLogisticsNo || trackingNoFromText(blob)
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
  const tracks = (parsed?.tracks || []).slice(0, 5)
  return {
    logisticsNo: parsed?.logisticsNo || fallbackNo,
    returnLogisticsNo: parsed?.returnLogisticsNo || '',
    shipLogisticsNo: parsed?.shipLogisticsNo || '',
    carrier: parsed?.carrier || '',
    shipTime: parsed?.shipTime || '',
    returnLocation: secondLastTrack(tracks),
    tracks,
  }
}

function logisticsClickTarget(cell, want) {
  if (!cell) return null
  const nodes = cell.querySelectorAll('[class*="clickable"], [class*="logisticsText"], a, span')
  for (const el of nodes) {
    if (textOf(el).includes(want)) return el
  }
  return cell.querySelector('[class*="clickable"], [class*="logisticsText"]') || cell
}

async function readShipLogistics(ticket) {
  closeLogisticsDrawer()
  await sleep(120)
  const pair = findDataRowByAftersaleId(ticket.platformAftersaleId)
  const cell = pair ? cellByName(pair.data, ['物流信息'], pair.headers) : null
  const fallbackNo =
    cleanTrackingNo(ticket.shipLogisticsNo) ||
    shipLogisticsNoFromRecord(findTableRecord(cell || pair?.data || pair?.header))
  const clickEl = logisticsClickTarget(cell, '订单发货') || logisticsClickTarget(cell, '发货')
  if (!clickEl) return { shipLogisticsNo: fallbackNo, tracks: [] }
  fireClick(clickEl)
  const parsed = await waitUntil(() => {
    const d = parseLogisticsDrawer()
    if (d && (d.shipLogisticsNo || d.logisticsNo || (d.tracks && d.tracks.length))) return d
    return null
  }, 25, 200)
  closeLogisticsDrawer()
  await waitUntil(() => (parseLogisticsDrawer() ? null : true), 15, 120)
  const tracks = (parsed?.tracks || []).slice(0, 5)
  return {
    shipLogisticsNo: parsed?.shipLogisticsNo || parsed?.logisticsNo || fallbackNo,
    logisticsNo: parsed?.logisticsNo || parsed?.shipLogisticsNo || fallbackNo,
    carrier: parsed?.carrier || '',
    shipTime: parsed?.shipTime || '',
    tracks,
  }
}

function applyTicketTracks(ticket, tracks) {
  const list = (tracks || []).slice(0, 5)
  if (!list.length) return ticket
  ticket.trackJson = JSON.stringify(list)
  try {
    const o = JSON.parse(ticket.rawJson || '{}')
    o.tracks = list
    ticket.rawJson = JSON.stringify(o)
  } catch {
    /* ignore */
  }
  return ticket
}

function hasBuyerReturnLogistics(ticket) {
  const text = String(ticket?.logistics || '')
  return text.includes('买家退货') || !!String(ticket?.returnLogisticsNo || '').trim()
}

function hasOrderShipLogistics(ticket) {
  const text = String(ticket?.logistics || '')
  return text.includes('订单发货') || needsIntercept(text) || !!String(ticket?.shipLogisticsNo || '').trim()
}

/** 所有售后单：买家退货采退回单号+轨迹；仅订单发货采发货单号+轨迹；轨迹最多 5 条 */
async function enrichAllTicketLogistics(rows) {
  const out = []
  for (const t of rows || []) {
    const row = { ...t, logistics: formatTicketLogistics(t.logistics) }
    const buyer = hasBuyerReturnLogistics(row)
    const ship = hasOrderShipLogistics(row)
    const anyHint = String(row.logistics || '').trim() || buyer || ship

    if (buyer || (anyHint && !ship)) {
      if (!row.trackJson || !row.returnLogisticsNo) {
        try {
          const extra = await readReturnLogistics(row)
          const retNo =
            cleanTrackingNo(extra.returnLogisticsNo) ||
            cleanTrackingNo(extra.logisticsNo) ||
            row.returnLogisticsNo ||
            ''
          if (retNo) row.returnLogisticsNo = retNo
          if (extra.shipLogisticsNo && !row.shipLogisticsNo) {
            row.shipLogisticsNo = cleanTrackingNo(extra.shipLogisticsNo)
          }
          if (!row.trackJson) applyTicketTracks(row, extra.tracks)
        } catch {
          /* keep row */
        }
      }
    }

    if (ship && (!row.shipLogisticsNo || (!buyer && !row.trackJson))) {
      try {
        const extra = await readShipLogistics(row)
        if (extra.shipLogisticsNo) row.shipLogisticsNo = cleanTrackingNo(extra.shipLogisticsNo)
        if (!buyer && !row.trackJson) applyTicketTracks(row, extra.tracks)
      } catch {
        /* keep row */
      }
    }

    out.push(row)
  }
  return out
}

function applyRefundTracksToTickets(ticketMap, returns, shippedRefunds) {
  const tracksOf = (item) => {
    try {
      return JSON.parse(item?.trackJson || '[]')
    } catch {
      return []
    }
  }
  for (const item of [...(returns || []), ...(shippedRefunds || [])]) {
    const t = ticketMap.get(item.platformAftersaleId)
    if (!t || t.trackJson) continue
    applyTicketTracks(t, tracksOf(item))
    if (!t.returnLogisticsNo && item.logisticsNo && hasBuyerReturnLogistics(t)) {
      t.returnLogisticsNo = item.logisticsNo
    }
    if (!t.shipLogisticsNo && (item.shipLogisticsNo || (!hasBuyerReturnLogistics(t) && item.logisticsNo))) {
      t.shipLogisticsNo = item.shipLogisticsNo || item.logisticsNo || ''
    }
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
  const tracks = (extra.tracks || []).slice(0, 5)
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
      tracks,
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
      if (hasBuyerReturnLogistics(t)) {
        extra = await readReturnLogistics(t)
      } else {
        const ship = await readShipLogistics(t)
        extra = {
          logisticsNo: ship.shipLogisticsNo || ship.logisticsNo || '',
          shipLogisticsNo: ship.shipLogisticsNo || '',
          carrier: ship.carrier || '',
          shipTime: ship.shipTime || '',
          returnLocation: '',
          tracks: ship.tracks || [],
        }
      }
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
  let prevCard = emptyPrevCard()
  for (const card of cardRows) {
    if (!card.count) continue
    const rows = await collectCardTickets(card, prevCard)
    if (rows.length) {
      prevCard = {
        cardKey: card.cardKey,
        groupName: card.groupName,
        cardLabel: card.cardLabel,
        type: cardTypeKeyword(card.cardLabel),
        count: card.count,
        visibleIds: ticketIds(rows),
        collectedIds: ticketIds(rows),
      }
    }
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
        if (!existing.shipLogisticsNo && t.shipLogisticsNo) {
          existing.shipLogisticsNo = t.shipLogisticsNo
        }
        if (!existing.trackJson && t.trackJson) {
          existing.trackJson = t.trackJson
        }
      } else {
        ticketMap.set(t.platformAftersaleId, { ...t, cardKeys: [card.cardKey] })
      }
    }
  }
  await resetAftersaleFilters()
  applyRefundTracksToTickets(ticketMap, returns, shippedRefunds)
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
