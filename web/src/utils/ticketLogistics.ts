export type LogisticsTone = 'danger' | 'warning' | 'ok' | ''

export interface LogisticsLine {
  label: string
  status?: string
  tone?: LogisticsTone
}

const LOGISTICS_STATUS_RE = /已退回|已取消|待取件|已签收|运输中|已发货/

function chunkAfter(text: string, label: string, stops: string[]) {
  const i = text.indexOf(label)
  if (i < 0) return ''
  let rest = text.slice(i + label.length)
  let cut = rest.length
  for (const stop of stops) {
    const j = rest.indexOf(stop)
    if (j >= 0 && j < cut) cut = j
  }
  return rest.slice(0, cut)
}

export function buyerReturnTone(status?: string): LogisticsTone {
  if (status === '已签收') return 'danger'
  if (status === '待取件') return 'warning'
  if (status === '运输中') return 'ok'
  return ''
}

export function parseTicketLogistics(row: {
  logistics?: string
  logisticsBuyerStatus?: string
  logisticsShipStatus?: string
  needIntercept?: boolean
  returnLogisticsNo?: string
  shipLogisticsNo?: string
  logisticsNo?: string
  awaitPickup?: boolean
}) {
  const text = row.logistics || ''
  const hasBuyer = text.includes('买家退货')
  const hasShip = text.includes('订单发货')
  const buyerStatus =
    row.logisticsBuyerStatus ||
    (hasBuyer ? chunkAfter(text, '买家退货', ['订单发货', '需商家拦截快递']).match(LOGISTICS_STATUS_RE)?.[0] || '' : '')
  const shipStatus =
    row.logisticsShipStatus ||
    (hasShip ? chunkAfter(text, '订单发货', ['买家退货', '需商家拦截快递']).match(LOGISTICS_STATUS_RE)?.[0] || '' : '')
  const intercept = row.needIntercept || (hasShip && text.includes('需商家拦截快递'))
  const lines: LogisticsLine[] = []
  if (hasBuyer) lines.push({ label: '买家退货', status: buyerStatus, tone: buyerReturnTone(buyerStatus) })
  if (hasShip) lines.push({ label: '订单发货', status: shipStatus })
  if (intercept) lines.push({ label: '需商家拦截快递', tone: 'danger' })
  if (row.awaitPickup && !lines.some((l) => l.label === '待取件')) {
    lines.push({ label: '待取件', tone: 'danger' })
  }
  return {
    lines,
    shipNo: row.shipLogisticsNo || (!row.returnLogisticsNo ? row.logisticsNo || '' : '') || '',
    returnNo: row.returnLogisticsNo || '',
  }
}

export function collapseDuplicatedText(s: string) {
  const text = String(s || '').replace(/\s+/g, ' ').trim()
  if (text.length < 24) return text
  const almostSame = (a: string, b: string) => {
    if (!a || !b) return false
    const x = a.replace(/[。！!．.\s]+$/g, '')
    const y = b.replace(/[。！!．.\s]+$/g, '')
    if (x === y && x.length >= 12) return true
    const longer = x.length >= y.length ? x : y
    const shorter = x.length >= y.length ? y : x
    return shorter.length >= 12 && longer.startsWith(shorter) && shorter.length / longer.length >= 0.85
  }
  const maxSplit = Math.min(text.length - 12, Math.floor(text.length * 0.6))
  for (let len = maxSplit; len >= 12; len--) {
    const a = text.slice(0, len).trim()
    const b = text.slice(len).trim()
    if (almostSame(a, b)) return a.length >= b.length ? a : b
  }
  return text
}

export function displayTrackDetail(track: {
  date?: string
  title?: string
  detail?: string
  text?: string
}) {
  const date = String(track.date || '').trim()
  const title = String(track.title || '').trim()
  let detail = String(track.detail || '').trim()
  if (!detail) detail = title || date ? '' : String(track.text || '').trim()
  for (const part of [date, title]) {
    if (part && detail.startsWith(part)) detail = detail.slice(part.length).trim()
  }
  return collapseDuplicatedText(detail)
}
