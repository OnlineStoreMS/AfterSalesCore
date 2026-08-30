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
  nextSyncAt?: string
  syncRequested?: boolean
  pendingTicketCount?: number
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
  timeoutAction?: string
  deadlineAt?: string
  remainSeconds?: number
  dispute?: string
  logistics?: string
  logisticsBuyerStatus?: string
  logisticsShipStatus?: string
  needIntercept?: boolean
  returnLogisticsNo?: string
  shipLogisticsNo?: string
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

export interface PluginSetting {
  pluginSyncIntervalMin: number
}

export const PLUGIN_SYNC_OPTIONS: { value: number; label: string }[] = [
  { value: 15, label: '每 15 分钟' },
  { value: 30, label: '每 30 分钟' },
  { value: 60, label: '每 1 小时' },
  { value: 120, label: '每 2 小时' },
  { value: 360, label: '每 6 小时' },
  { value: 720, label: '每 12 小时' },
  { value: 1440, label: '每 24 小时' },
]

export async function fetchPluginSetting() {
  return unwrap<PluginSetting>(await client.get('/plugin-settings'))
}

export async function savePluginSetting(data: PluginSetting) {
  return unwrap<PluginSetting>(await client.put('/plugin-settings', data))
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

export async function requestShopSync(id: number) {
  return unwrap<MarketplaceShop>(await client.post(`/shops/${id}/request-sync`))
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

export interface ReturnPackage {
  id: number
  shopId: number
  shopName: string
  platformAftersaleId: string
  orderNo: string
  productTitle: string
  productImage?: string
  sku: string
  qty: number
  buyQty?: number
  payAmount: string
  refundAmount: string
  aftersaleType: string
  reason: string
  status: string
  orderInfo?: string
  aftersaleInfo?: string
  logistics?: string
  logisticsNo: string
  carrier?: string
  returnLocation: string
  shipTime?: string
  applyTime?: string
  returnTime?: string
  syncedAt: string
}

export interface ShippedRefund {
  id: number
  shopId: number
  shopName: string
  platformAftersaleId: string
  orderNo: string
  productTitle: string
  productImage?: string
  sku: string
  productTags?: string
  tags?: string
  qty: number
  buyQty?: number
  payAmount: string
  refundAmount: string
  aftersaleType: string
  reason: string
  status: string
  orderInfo?: string
  aftersaleInfo?: string
  logistics?: string
  logisticsStatus?: string
  logisticsNo?: string
  carrier?: string
  shipTime?: string
  tracks?: LogisticsTrack[]
  alert: boolean
  applyTime?: string
  syncedAt: string
}

export interface LogisticsTrack {
  date?: string
  title?: string
  detail?: string
  text?: string
}

export async function fetchReturnPackages(params?: {
  shopId?: number
  keyword?: string
  returnFrom?: string
  returnTo?: string
  applyFrom?: string
  applyTo?: string
  page?: number
  pageSize?: number
}) {
  return unwrap<PageData<ReturnPackage>>(await client.get('/return-packages', { params }))
}

export async function fetchShippedRefunds(params?: {
  shopId?: number
  keyword?: string
  status?: string
  alertOnly?: boolean
  applyFrom?: string
  applyTo?: string
  page?: number
  pageSize?: number
}) {
  return unwrap<PageData<ShippedRefund>>(await client.get('/shipped-refunds', {
    params: {
      ...params,
      alertOnly: params?.alertOnly ? '1' : undefined,
    },
  }))
}

export interface InterceptOrder {
  id: number
  shopId: number
  shopName: string
  source: string
  needIntercept: boolean
  awaitPickup: boolean
  platformAftersaleId: string
  orderNo: string
  productTitle: string
  productImage?: string
  sku: string
  qty: number
  buyQty?: number
  payAmount: string
  refundAmount: string
  aftersaleType: string
  reason: string
  status: string
  logistics?: string
  logisticsStatus?: string
  logisticsNo?: string
  shipLogisticsNo?: string
  returnLogisticsNo?: string
  carrier?: string
  applyTime?: string
  syncedAt: string
}

export async function fetchInterceptOrders(params?: {
  shopId?: number
  keyword?: string
  page?: number
  pageSize?: number
}) {
  return unwrap<PageData<InterceptOrder>>(await client.get('/intercept-orders', { params }))
}
