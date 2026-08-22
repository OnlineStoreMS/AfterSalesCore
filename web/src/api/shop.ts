import client, { unwrap, type PageData } from './client'

export type ShopPlatform = 'doudian' | 'taobao' | 'pinduoduo'
export type PluginStatus = 'unbound' | 'online' | 'offline'

export interface MarketplaceShop {
  id: number
  name: string
  platform: ShopPlatform
  platformLabel: string
  bindCode: string
  pluginStatus: PluginStatus
  pluginAvailable: boolean
  platformShopId?: string
  platformShopName?: string
  lastSyncAt?: string
  lastSeenAt?: string
  remark?: string
  createdAt: string
  updatedAt: string
}

export interface FilterCard {
  groupName: string
  cardKey: string
  cardLabel: string
  count: number
  sortOrder: number
}

export interface AftersaleTicket {
  id: number
  platformAftersaleId: string
  orderNo: string
  productTitle: string
  productImage?: string
  sku: string
  productTags?: string
  tags?: string
  qty: number
  buyQty: number
  payAmount: string
  refundAmount: string
  aftersaleType: string
  reason: string
  status: string
  timeoutText?: string
  dispute?: string
  logistics?: string
  applyTime?: string
  cardKeys: string[]
  syncedAt: string
}

export interface ShopWorkbench {
  shop: MarketplaceShop
  cards: FilterCard[]
  tickets: AftersaleTicket[]
  total: number
  page: number
  pageSize: number
  lastSyncAt?: string
}

export const PLATFORM_OPTIONS: { value: ShopPlatform; label: string }[] = [
  { value: 'doudian', label: '抖店' },
  { value: 'taobao', label: '淘宝' },
  { value: 'pinduoduo', label: '拼多多' },
]

export const PLUGIN_STATUS_MAP: Record<PluginStatus, { label: string; type: '' | 'success' | 'warning' | 'info' | 'danger' }> = {
  unbound: { label: '未绑定', type: 'info' },
  online: { label: '在线', type: 'success' },
  offline: { label: '离线', type: 'danger' },
}

export async function fetchShops() {
  return unwrap<MarketplaceShop[]>(await client.get('/shops'))
}

export async function fetchShop(id: number) {
  return unwrap<MarketplaceShop>(await client.get(`/shops/${id}`))
}

export async function createShop(data: { name: string; platform?: ShopPlatform; remark?: string }) {
  return unwrap<MarketplaceShop>(await client.post('/shops', data))
}

export async function updateShop(id: number, data: { name?: string; remark?: string }) {
  return unwrap<MarketplaceShop>(await client.put(`/shops/${id}`, data))
}

export async function deleteShop(id: number) {
  return unwrap<{ deleted: boolean }>(await client.delete(`/shops/${id}`))
}

export async function resetShopBind(id: number) {
  return unwrap<MarketplaceShop>(await client.post(`/shops/${id}/reset-bind`))
}

export async function fetchShopWorkbench(id: number, params?: {
  cardKey?: string
  keyword?: string
  page?: number
  pageSize?: number
}) {
  return unwrap<ShopWorkbench>(await client.get(`/shops/${id}/workbench`, { params }))
}

export async function fetchShopTickets(id: number, params?: {
  cardKey?: string
  keyword?: string
  page?: number
  pageSize?: number
}) {
  return unwrap<PageData<AftersaleTicket>>(await client.get(`/shops/${id}/tickets`, { params }))
}
