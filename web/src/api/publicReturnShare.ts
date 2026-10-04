import axios from 'axios'
import type { ApiResponse, PageData } from './client'
import type { ReturnPackage } from './shop'

const publicClient = axios.create({
  baseURL: (import.meta.env.BASE_URL || '/') + 'api/v1/public',
  timeout: 120000,
  headers: { 'Content-Type': 'application/json' },
})

publicClient.interceptors.response.use(
  (res) => {
    if (res.config.responseType === 'blob') return res
    const body = res.data as ApiResponse
    if (body.code !== 200) {
      return Promise.reject(new Error(body.message || '请求失败'))
    }
    return res
  },
  (err) => {
    const msg = err.response?.data?.message || err.message || '请求失败'
    return Promise.reject(new Error(msg))
  },
)

export interface PublicReturnSharePage extends PageData<ReturnPackage> {
  name: string
  returnLocation: string
}

export async function fetchPublicReturnShare(token: string, params?: { keyword?: string; page?: number; pageSize?: number; sortBy?: string; sortOrder?: string }) {
  const res = await publicClient.get(`/return-shares/${encodeURIComponent(token)}`, { params })
  return (res.data as ApiResponse<PublicReturnSharePage>).data as PublicReturnSharePage
}

export async function exportPublicReturnShare(token: string) {
  const res = await publicClient.post(`/return-shares/${encodeURIComponent(token)}/export`, {}, {
    responseType: 'blob',
    timeout: 120000,
  })
  const blob = res.data as Blob
  if (blob.type && blob.type.includes('application/json')) {
    const text = await blob.text()
    try {
      const body = JSON.parse(text) as { message?: string }
      throw new Error(body.message || '导出失败')
    } catch (e) {
      if (e instanceof Error && e.message !== '导出失败') throw e
      throw new Error('导出失败')
    }
  }
  const header = String(res.headers['content-disposition'] || '')
  const star = header.match(/filename\*=UTF-8''([^;]+)/i)
  const plain = header.match(/filename="?([^";]+)"?/i)
  const filename = decodeURIComponent(star?.[1] || '') || plain?.[1] || '退回件.xlsx'
  return { blob, filename }
}
