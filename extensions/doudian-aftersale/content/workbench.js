/** 抖店售后工作台采集：快捷筛选卡片 + 各卡片售后单 */

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}

function textOf(el) {
  return String(el?.innerText || el?.textContent || '').replace(/\s+/g, ' ').trim()
}

function parseCards() {
  const cards = []
  const titles = Array.from(document.querySelectorAll('[class*="groupTitle"]'))
  let sort = 0
  for (const titleEl of titles) {
    const groupName = textOf(titleEl)
    if (!groupName || groupName.length > 24) continue
    if (!/紧急|待商家|待消费者|纠纷/.test(groupName)) continue
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
  let platformShopId = ''
  let platformShopName = ''
  try {
    const html = document.documentElement.innerHTML.slice(0, 400000)
    const idMatch = html.match(/shopid[=:]["']?(\d{5,})/i) || html.match(/"shop_id"\s*:\s*"?(\d{5,})/)
    if (idMatch) platformShopId = idMatch[1]
    const nameMatch = html.match(/shopname=([^&"'<]+)/i)
    if (nameMatch) platformShopName = decodeURIComponent(nameMatch[1])
  } catch {
    /* ignore */
  }
  return { platformShopId, platformShopName }
}

function pick(text, re) {
  const m = text.match(re)
  return m ? String(m[1] || '').trim() : ''
}

function guessTitle(lines) {
  const skip = /编号|金额|申请|退款|物流|状态|件数|原因|操作|商品信息|订单信息|纠纷|售后信息|共\d+条/
  return (
    lines
      .filter((l) => l.length >= 8 && !skip.test(l) && !/^[¥\d.\s]+$/.test(l) && !/^\d+$/.test(l))
      .sort((a, b) => b.length - a.length)[0] || ''
  )
}

function parseTicketText(raw) {
  const text = String(raw || '').replace(/\s+/g, ' ').trim()
  const platformAftersaleId = pick(text, /售后编号\s*([0-9]+)/)
  if (!platformAftersaleId) return null
  const qty = Number(pick(text, /申请件数\s*(\d+)/) || pick(text, /购买件数\s*(\d+)/) || 0)
  const lines = String(raw || '')
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
  const sku = lines.find((l) => /【|规格|款/.test(l) && l.length < 80 && l.length > 2) || ''
  return {
    platformAftersaleId,
    orderNo: pick(text, /订单编号\s*([0-9]+)/),
    productTitle: guessTitle(lines),
    productImage: '',
    sku,
    qty: Number.isFinite(qty) ? qty : 0,
    payAmount: pick(text, /应付金额\s*¥?\s*([\d.]+)/),
    refundAmount: pick(text, /售后退款\s*¥?\s*([\d.]+)/),
    aftersaleType: pick(text, /(退货退款|仅退款|换货|补寄|维修|未发货退款|已发货退款)/),
    reason: pick(text, /申请原因\s*([^申待物纠操]{2,40})/),
    status: pick(text, /(待商家收货|待商家审核|待消费者退货|待消费者处理|同意退款|已完成|已关闭|仲裁平台处理中)/),
    timeoutText: pick(text, /(\d+小时\d+分[^ ]*自动同意|\d+分[^ ]*后自动[^\s]{0,8})/),
    dispute: pick(text, /(未介入|仲裁待协商|仲裁待举证|仲裁平台处理中|平台主动处理)/),
    logistics: pick(text, /((?:买家退货|订单发货)[^操共]{0,40})/),
    applyTime: pick(text, /申请时间\s*([\d/:-]+\s*[\d:]*)/),
    rawJson: JSON.stringify({ text: text.slice(0, 2000) }),
  }
}

function collectVisibleTickets() {
  const hits = Array.from(document.querySelectorAll('*')).filter((el) => {
    if (el.children && el.children.length > 12) return false
    const t = el.childNodes.length ? textOf(el) : ''
    return t.includes('售后编号') && /\d{8,}/.test(t)
  })
  const rows = []
  const seen = new Set()
  for (const el of hits) {
    let cur = el
    let row = null
    for (let i = 0; i < 10 && cur; i++) {
      const t = textOf(cur)
      if (t.includes('售后编号') && t.includes('订单编号') && t.length > 40 && t.length < 4000) {
        row = cur
        break
      }
      cur = cur.parentElement
    }
    if (!row) continue
    const parsed = parseTicketText(row.innerText || '')
    if (!parsed || seen.has(parsed.platformAftersaleId)) continue
    seen.add(parsed.platformAftersaleId)
    rows.push(parsed)
  }
  return rows
}

function findNextPage() {
  const nodes = Array.from(document.querySelectorAll('li, button, a, span, div'))
  return nodes.find((el) => {
    const t = textOf(el)
    if (t !== '下一页') return false
    const cls = String(el.className || '')
    const parentCls = String(el.parentElement?.className || '')
    const disabled =
      el.getAttribute('aria-disabled') === 'true' ||
      el.hasAttribute('disabled') ||
      /disabled|is-disabled/.test(cls + parentCls)
    return !disabled
  })
}

async function collectCardTickets(expectedCount) {
  const all = []
  const seen = new Set()
  const maxPages = Math.max(1, Math.ceil((expectedCount || 10) / 10) + 2)
  for (let p = 0; p < maxPages; p++) {
    await sleep(p === 0 ? 200 : 700)
    const rows = collectVisibleTickets()
    for (const r of rows) {
      if (seen.has(r.platformAftersaleId)) continue
      seen.add(r.platformAftersaleId)
      all.push(r)
    }
    if (expectedCount && all.length >= expectedCount) break
    const next = findNextPage()
    if (!next) break
    next.click()
    await sleep(900)
  }
  return all
}

async function collectAll() {
  const cardRows = parseCards()
  if (!cardRows.length) {
    throw new Error('未找到快捷筛选卡片，请确认当前是售后工作台列表页')
  }
  const ticketMap = new Map()
  for (const card of cardRows) {
    if (!card.count) continue
    try {
      card.el.scrollIntoView({ block: 'nearest' })
    } catch {
      /* ignore */
    }
    card.el.click()
    await sleep(900)
    const rows = await collectCardTickets(card.count)
    for (const t of rows) {
      const existing = ticketMap.get(t.platformAftersaleId)
      if (existing) {
        if (!existing.cardKeys.includes(card.cardKey)) existing.cardKeys.push(card.cardKey)
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

chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
  if (msg?.type !== 'AFTERSALE_COLLECT') return
  collectAll()
    .then((data) => sendResponse(data))
    .catch((e) => sendResponse({ ok: false, error: e instanceof Error ? e.message : String(e) }))
  return true
})

if (!window.__OSMS_AFTERSALE_READY__) {
  window.__OSMS_AFTERSALE_READY__ = true
  setTimeout(() => {
    try {
      chrome.runtime.sendMessage({ type: 'AFTERSALE_WORKBENCH_READY' })
    } catch {
      /* ignore */
    }
  }, 2000)
}
