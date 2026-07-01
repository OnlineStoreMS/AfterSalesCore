import client, { unwrap, type PageData } from './client'

export interface UnboxingPhoto {
  id: number
  photoUrl: string
  issueRemark?: string
  sortOrder: number
  createdAt?: string
}

export interface UnboxingRecord {
  id: number
  trackingNo: string
  videoUrl?: string
  videoSize?: number
  videoDurationSec?: number
  videoMimeType?: string
  status: string
  remark?: string
  operatorId?: number
  operatorName?: string
  createdAt?: string
  photos: UnboxingPhoto[]
}

export interface UnboxingListItem {
  id: number
  trackingNo: string
  status: string
  videoUrl?: string
  videoDurationSec?: number
  photoCount: number
  operatorName?: string
  createdAt: string
}

export const UNBOXING_STATUS_MAP: Record<string, { label: string; type: '' | 'success' | 'warning' | 'info' }> = {
  draft: { label: '草稿', type: 'info' },
  completed: { label: '已完成', type: 'success' },
}

export async function fetchUnboxingRecords(params: {
  trackingNo?: string
  page?: number
  pageSize?: number
}) {
  const res = await client.get('/unboxing-records', { params })
  return unwrap<PageData<UnboxingListItem>>(res)
}

export async function fetchUnboxingRecord(id: number) {
  return unwrap<UnboxingRecord>(await client.get(`/unboxing-records/${id}`))
}

export async function createUnboxingRecord(data: { trackingNo: string; remark?: string }) {
  return unwrap<UnboxingRecord>(await client.post('/unboxing-records', data))
}

export async function uploadUnboxingVideo(id: number, file: File, durationSec: number) {
  const form = new FormData()
  form.append('file', file)
  form.append('durationSec', String(durationSec))
  return unwrap<UnboxingRecord>(
    await client.post(`/unboxing-records/${id}/video`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 300000,
    }),
  )
}

export async function uploadUnboxingPhoto(id: number, file: File, issueRemark?: string) {
  const form = new FormData()
  form.append('file', file)
  if (issueRemark) form.append('issueRemark', issueRemark)
  return unwrap<UnboxingRecord>(
    await client.post(`/unboxing-records/${id}/photos`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  )
}

export async function completeUnboxingRecord(id: number, data?: { videoDurationSec?: number; remark?: string }) {
  return unwrap<UnboxingRecord>(await client.post(`/unboxing-records/${id}/complete`, data || {}))
}

export async function getUnboxingVideoDownload(id: number) {
  return unwrap<{ url: string; filename: string }>(await client.get(`/unboxing-records/${id}/video/download`))
}
