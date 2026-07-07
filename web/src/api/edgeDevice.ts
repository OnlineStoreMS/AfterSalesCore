import client, { unwrap } from './client'

export interface EdgeDevice {
  id: number
  edgeId: string
  name: string
  baseUrl: string
  status: 'online' | 'offline' | 'unknown'
  lastSeenAt?: string
  remark?: string
  createdAt: string
  updatedAt: string
}

export const EDGE_STATUS_MAP: Record<string, { label: string; type: '' | 'success' | 'warning' | 'info' | 'danger' }> = {
  online: { label: '在线', type: 'success' },
  offline: { label: '离线', type: 'danger' },
  unknown: { label: '未知', type: 'info' },
}

export async function fetchEdgeDevices() {
  return unwrap<EdgeDevice[]>(await client.get('/edge-devices'))
}

export async function createEdgeDevice(data: {
  edgeId: string
  name: string
  baseUrl?: string
  remark?: string
}) {
  return unwrap<EdgeDevice>(await client.post('/edge-devices', data))
}

export async function updateEdgeDevice(id: number, data: {
  name?: string
  baseUrl?: string
  remark?: string
}) {
  return unwrap<EdgeDevice>(await client.put(`/edge-devices/${id}`, data))
}

export async function deleteEdgeDevice(id: number) {
  return unwrap<{ deleted: boolean }>(await client.delete(`/edge-devices/${id}`))
}

export async function probeEdgeDevice(id: number) {
  return unwrap<EdgeDevice>(await client.post(`/edge-devices/${id}/probe`))
}

export async function syncEdgeDevices() {
  return unwrap<EdgeDevice[]>(await client.post('/edge-devices/sync'))
}
