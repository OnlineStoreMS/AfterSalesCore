import client, { unwrap } from './client'

export interface PluginDebugLogItem {
  name: string
  runId: string
  kind: string
  ok: boolean
  error?: string
  durationMs: number
  version: string
  shopId: number
  shopName: string
  receivedAt: string
  size: number
}

export interface PluginDebugEvent {
  ms: number
  at: string
  level: string
  step: string
  data?: unknown
}

export interface PluginDebugLogRecord {
  runId: string
  kind: string
  ok: boolean
  error?: string
  durationMs: number
  version: string
  shopId: number
  shopName: string
  receivedAt: string
  meta?: Record<string, unknown>
  events: PluginDebugEvent[]
}

export async function fetchPluginDebugLogs() {
  return unwrap<{ dir: string; list: PluginDebugLogItem[] }>(await client.get('/plugin-debug-logs'))
}

export async function fetchPluginDebugLog(name: string) {
  return unwrap<PluginDebugLogRecord>(
    await client.get(`/plugin-debug-logs/${encodeURIComponent(name)}`),
  )
}
