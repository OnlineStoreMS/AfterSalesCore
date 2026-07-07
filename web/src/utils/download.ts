import { getToken } from './auth'

export function filenameFromDisposition(header: string | null, fallback: string): string {
  if (!header) return fallback
  const utf8 = header.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8) return decodeURIComponent(utf8[1])
  const plain = header.match(/filename="([^"]+)"/i)
  if (plain) return plain[1]
  return fallback
}

export async function saveResponseBlob(res: Response, fallbackName: string) {
  const blob = await res.blob()
  const filename = filenameFromDisposition(res.headers.get('Content-Disposition'), fallbackName)
  const objectUrl = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = objectUrl
  a.download = filename
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(objectUrl)
}

export async function downloadAuthenticated(url: string, fallbackName: string) {
  const token = getToken()
  const res = await fetch(url, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) {
    let msg = '下载失败'
    try {
      const data = await res.json()
      msg = data.message || msg
    } catch {
      /* non-json error body */
    }
    throw new Error(msg)
  }
  await saveResponseBlob(res, fallbackName)
}
