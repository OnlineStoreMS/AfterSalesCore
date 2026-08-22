import client, { unwrap } from './client'

export interface NotificationScenarioOption {
  key: string
  label: string
  group?: string
}

export interface NotificationShopOption {
  id: string
  name: string
}

export interface NotificationConfig {
  enabled: boolean
  webhookUrl: string
  secret?: string
  secretSet?: boolean
  appId?: string
  appSecret?: string
  appSecretSet?: boolean
  pollIntervalMinutes: number
  scenarios: string[]
  shopIds?: string[]
}

export interface NotificationState {
  lastRunAt?: string
  lastRunOk?: boolean
  lastError?: string
  lastSentCount?: number
  lastBarcodeError?: string
}

export interface NotificationView {
  config: NotificationConfig
  state: NotificationState
  scenarios: NotificationScenarioOption[]
  shops: NotificationShopOption[]
}

export async function getNotification() {
  return unwrap<NotificationView>(await client.get('/notifications'))
}

export async function saveNotification(body: NotificationConfig) {
  return unwrap<NotificationView>(await client.put('/notifications', body))
}

export async function testNotification(text?: string) {
  return unwrap<{ ok: boolean }>(await client.post('/notifications/test', { text }))
}

export async function testBarcodeNotification() {
  return unwrap<{ ok: boolean }>(await client.post('/notifications/test-barcode'))
}

export async function runNotification() {
  return unwrap<{
    sent: number
    skipped: number
    barcodeWarnings?: number
    lastBarcodeError?: string
    error?: string
  }>(await client.post('/notifications/run'))
}

export async function resetNotificationState() {
  return unwrap<{ cleared: number; view: NotificationView }>(await client.post('/notifications/reset-state'))
}
