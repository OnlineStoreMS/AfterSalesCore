import client, { unwrap, type PageData } from './client'

export interface IssueOption {
  value: string
  label: string
}

export interface IssueMeta {
  problemTypes: IssueOption[]
  handleMethods: IssueOption[]
  statuses: IssueOption[]
}

export interface IssueLookup {
  orderNo?: string
  platformOrderId?: string
  platformAftersaleId?: string
  shopId?: number
  shopName: string
  productTitle: string
  productImage?: string
  skuSpecs?: string
  buyerName?: string
  buyerPhone?: string
  buyerAddress?: string
  hasAftersale: boolean
  aftersaleStatus?: string
  aftersaleType?: string
  logistics?: string
  returnLogisticsNo?: string
  shipLogisticsNo?: string
  source: string
}

export interface AftersaleIssue {
  id: number
  shopId?: number
  shopName: string
  orderNo?: string
  platformOrderId?: string
  platformAftersaleId?: string
  productTitle: string
  productImage?: string
  skuSpecs?: string
  buyerName?: string
  buyerPhone?: string
  buyerAddress?: string
  hasAftersale: boolean
  aftersaleStatus?: string
  aftersaleType?: string
  logistics?: string
  returnLogisticsNo?: string
  shipLogisticsNo?: string
  problemType: string
  problemTypeCustom?: string
  problemTypeLabel: string
  handleMethod?: string
  handleMethodCustom?: string
  handleMethodLabel?: string
  problemNote?: string
  handleNote?: string
  status: string
  statusLabel: string
  operatorName?: string
  completedAt?: string | null
  createdAt: string
  updatedAt: string
}

export type IssueUpsertBody = {
  shopId?: number
  shopName: string
  orderNo?: string
  platformOrderId?: string
  platformAftersaleId?: string
  productTitle: string
  productImage?: string
  skuSpecs?: string
  buyerName?: string
  buyerPhone?: string
  buyerAddress?: string
  hasAftersale?: boolean
  aftersaleStatus?: string
  aftersaleType?: string
  logistics?: string
  returnLogisticsNo?: string
  shipLogisticsNo?: string
  problemType: string
  problemTypeCustom?: string
  handleMethod?: string
  handleMethodCustom?: string
  problemNote?: string
  handleNote?: string
  status?: string
}

export async function fetchIssueMeta() {
  return unwrap<IssueMeta>(await client.get('/issue-records/meta'))
}

export async function lookupIssueSource(q: string) {
  return unwrap<{ list: IssueLookup[] }>(await client.get('/issue-records/lookup', { params: { q } }))
}

export async function fetchIssues(params?: {
  shopId?: number
  status?: string
  problemType?: string
  keyword?: string
  page?: number
  pageSize?: number
}) {
  return unwrap<PageData<AftersaleIssue>>(await client.get('/issue-records', { params }))
}

export async function createIssue(body: IssueUpsertBody) {
  return unwrap<AftersaleIssue>(await client.post('/issue-records', body))
}

export async function updateIssue(id: number, body: IssueUpsertBody) {
  return unwrap<AftersaleIssue>(await client.put(`/issue-records/${id}`, body))
}

export async function updateIssueStatus(id: number, status: string) {
  return unwrap<AftersaleIssue>(await client.patch(`/issue-records/${id}/status`, { status }))
}

export async function deleteIssue(id: number) {
  return unwrap<{ ok: boolean }>(await client.delete(`/issue-records/${id}`))
}

/** 处理复制：地址 / 规格 / 问题 / 处理 组合文本 */
export function buildIssueHandleText(row: AftersaleIssue) {
  const lines: string[] = []
  const addr = [row.buyerName, row.buyerPhone, row.buyerAddress].filter(Boolean).join(' ').trim()
  if (addr) lines.push(`地址：${addr}`)
  if (row.skuSpecs?.trim()) lines.push(`规格：${row.skuSpecs.trim()}`)
  else if (row.productTitle?.trim()) lines.push(`商品：${row.productTitle.trim()}`)
  const problem = [row.problemTypeLabel, row.problemNote].filter(Boolean).join(' · ')
  if (problem) lines.push(`问题：${problem}`)
  const handle = [row.handleMethodLabel, row.handleNote].filter(Boolean).join(' · ')
  if (handle) lines.push(`处理：${handle}`)
  if (row.orderNo) lines.push(`订单：${row.orderNo}`)
  if (row.platformAftersaleId) lines.push(`售后：${row.platformAftersaleId}`)
  if (row.shopName) lines.push(`店铺：${row.shopName}`)
  return lines.join('\n')
}

export function buildMergedIssueHandleText(rows: AftersaleIssue[]) {
  return rows.map((r, i) => `【${i + 1}】\n${buildIssueHandleText(r)}`).join('\n\n')
}

export async function copyText(text: string) {
  const value = String(text || '').trim()
  if (!value) return false
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(value)
      return true
    }
  } catch {
    /* fallback */
  }
  const ta = document.createElement('textarea')
  ta.value = value
  ta.style.position = 'fixed'
  ta.style.left = '-9999px'
  document.body.appendChild(ta)
  ta.select()
  const ok = document.execCommand('copy')
  document.body.removeChild(ta)
  return ok
}
