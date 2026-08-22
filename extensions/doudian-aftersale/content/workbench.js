/** 抖店售后工作台采集：快捷筛选卡片 + 表格成对行（表头订单行 + 商品明细行） */

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}

function textOf(el) {
  return String(el?.innerText || el?.textContent || '').replace(/\s+/g, ' ').trim()
}

function pick(text, re) {
  const m = String(text || '').match(re)
  return m ? String(m[1] || '').trim() : ''
}

function parseCards() {
  const cards = []
  const titles = Array.from(document.querySelectorAll('[class*="groupTitle"]'))
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
  const nameEl =
    document.querySelector('[class*="headerShopName"]') ||
    document.querySelector('[class*="userName"]')
  let platformShopName = textOf(nameEl)
  let platformShopId = ''
  try {
    const html = document.documentElement.innerHTML.slice(0, 400000)
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
  const key = Object.keys(el).find((k) => k.startsWith('__reactFiber'))
  return key ? el[key] : null
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
    const timer = setTimeout(() => {
      document.removeEventListener('osms-aftersale-logistics', onMsg)
      resolve(logisticsMap)
    }, 400)
    function onMsg(e) {
      clearTimeout(timer)
      document.removeEventListener('osms-aftersale-logistics', onMsg)
      logisticsMap = e.detail && typeof e.detail === 'object' ? e.detail : {}
      resolve(logisticsMap)
    }
    document.addEventListener('osms-aftersale-logistics', onMsg)
    document.dispatchEvent(new Event('osms-aftersale-need-logistics'))
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
  const table = document.querySelector('table')
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
  el.click()
  await sleep(80)
}

async function waitEmptyList() {
  await waitUntil(() => {
    const total = parseListTotal()
    if (total === 0) return true
    if (total == null && collectVisibleTickets().length === 0) return true
    return null
  }, 20, 150)
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

function cardTableState(card) {
  if (!isCardSelected(card.el)) return null
  if (currentPage() > 1) return null
  const total = parseListTotal()
  const rows = collectVisibleTickets()
  if (card.count === 0) {
    if (total === 0 || rows.length === 0) return { rows, total: total ?? 0 }
    return null
  }
  if (total !== card.count) return null
  const maxFirst = Math.min(10, card.count)
  if (rows.length === 0 || rows.length > maxFirst) return null
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

async function collectCardTickets(card, zeroCard) {
  if (zeroCard?.el && zeroCard.el !== card.el) {
    await clickCard(zeroCard.el)
    await waitEmptyList()
  }
  await clickCard(card.el)
  await waitUntil(() => {
    if (!isCardSelected(card.el)) return null
    if (parseListTotal() !== card.count) return null
    return true
  }, 30, 200)
  await ensureFirstPage()
  await waitUntil(() => cardTableState(card), 25, 200)
  const all = []
  const seen = new Set()
  const add = (rows) => {
    for (const r of rows || []) {
      if (!r.platformAftersaleId || seen.has(r.platformAftersaleId)) continue
      seen.add(r.platformAftersaleId)
      all.push(r)
    }
  }
  add(await visibleTicketsWithLogistics())
  const expected = card.count || 0
  const maxPages = Math.max(1, Math.ceil((expected || 10) / 10) + 1)
  for (let p = 1; p < maxPages; p++) {
    if (expected && all.length >= expected) break
    const next = findNextPage()
    if (!next) break
    const prevIds = collectVisibleTickets().map((r) => r.platformAftersaleId)
    next.click()
    await waitForPageChange(prevIds)
    add(await visibleTicketsWithLogistics())
  }
  return all
}

async function collectAll() {
  const cardRows = parseCards()
  if (!cardRows.length) {
    throw new Error('未找到快捷筛选卡片，请确认当前是售后工作台列表页')
  }
  const zeroCard = cardRows.find((c) => !c.count)
  const ticketMap = new Map()
  for (const card of cardRows) {
    if (!card.count) continue
    const rows = await collectCardTickets(card, zeroCard)
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
  }
}

if (!window.__osmsWorkbenchInjected) {
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
      chrome.runtime.sendMessage({ type: 'AFTERSALE_WORKBENCH_READY' })
    } catch {
      /* ignore */
    }
  }, 2000)
}
