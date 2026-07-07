import client, { unwrap, type PageData } from './client'
import { downloadAuthenticated } from '../utils/download'

export type RecordType = 'unboxing' | 'packing'

export interface EdgeRecordPhoto {
  url: string
}

export interface EdgeRecord {
  id: number
  edgeId: string
  edgeName?: string
  type: RecordType
  trackingNo: string
  status: string
  videoUrl?: string
  videoSize?: number
  videoDurationSec?: number
  remark?: string
  photos: EdgeRecordPhoto[]
  createdAt?: string
  updatedAt?: string
  completedAt?: string
}

export interface EdgeRecordGoods {
  title?: string
  skuName?: string
  picUrl?: string
  num?: number
}

export interface EdgeRecordListItem {
  id: number
  edgeId: string
  edgeName?: string
  type: RecordType
  trackingNo: string
  status: string
  videoUrl?: string
  videoDurationSec?: number
  photoCount: number
  goods?: EdgeRecordGoods[]
  remark?: string
  createdAt: string
  completedAt?: string
}

export const RECORD_STATUS_MAP: Record<string, { label: string; type: '' | 'success' | 'warning' | 'info' | 'danger' }> = {
  draft: { label: '草稿', type: 'info' },
  recording: { label: '录制中', type: 'warning' },
  completed: { label: '已完成', type: 'success' },
}

export const RECORD_TYPE_LABEL: Record<RecordType, string> = {
  unboxing: '开箱',
  packing: '打包',
}

export async function fetchEdgeRecordStats() {
  return unwrap<{ unboxingCount: number; packingCount: number }>(await client.get('/edge-records/stats'))
}

export async function fetchEdgeRecords(params: {
  type?: RecordType
  trackingNo?: string
  edgeId?: string
  page?: number
  pageSize?: number
}) {
  const res = await client.get('/edge-records', { params })
  return unwrap<PageData<EdgeRecordListItem>>(res)
}

export async function fetchEdgeRecord(id: number) {
  return unwrap<EdgeRecord>(await client.get(`/edge-records/${id}`))
}

export async function createEdgeRecord(data: { type: RecordType; trackingNo: string; remark?: string }) {
  return unwrap<EdgeRecord>(await client.post('/edge-records', data))
}

export async function uploadEdgeRecordVideo(id: number, file: File, durationSec: number) {
  const form = new FormData()
  form.append('file', file)
  form.append('durationSec', String(durationSec))
  return unwrap<EdgeRecord>(
    await client.post(`/edge-records/${id}/video`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 300000,
    }),
  )
}

export async function uploadEdgeRecordPhoto(id: number, file: File) {
  const form = new FormData()
  form.append('file', file)
  return unwrap<EdgeRecord>(
    await client.post(`/edge-records/${id}/photos`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  )
}

export async function completeEdgeRecord(id: number, data?: { remark?: string }) {
  return unwrap<EdgeRecord>(await client.post(`/edge-records/${id}/complete`, data || {}))
}

export async function deleteEdgeRecord(id: number) {
  return unwrap<{ deleted: boolean }>(await client.delete(`/edge-records/${id}`))
}

export async function batchDeleteEdgeRecords(ids: number[]) {
  return unwrap<{ deleted: number }>(await client.post('/edge-records/batch-delete', { ids }))
}

export async function downloadEdgeRecordVideo(id: number) {
  await downloadAuthenticated(`/api/v1/admin/edge-records/${id}/video/download`, `video-${id}.webm`)
}
