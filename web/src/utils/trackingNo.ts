/** 从条码/文本中提取常见快递单号 */
export function normalizeTrackingNo(raw: string): string {
  return raw.trim().toUpperCase().replace(/\s+/g, '')
}

export function extractTrackingNoFromText(text: string): string | null {
  const normalized = text.toUpperCase()
  const patterns = [
    /\b(SF\d{10,15})\b/,
    /\b(YT\d{10,20})\b/,
    /\b(JD[A-Z0-9]{10,20})\b/,
    /\b(7[0-9]{12,16})\b/,
    /\b([0-9]{12,20})\b/,
    /\b([A-Z]{2}\d{9}[A-Z]{2})\b/,
  ]
  for (const p of patterns) {
    const m = normalized.match(p)
    if (m?.[1]) return normalizeTrackingNo(m[1])
  }
  return null
}

export function isValidTrackingNo(v: string): boolean {
  const s = normalizeTrackingNo(v)
  if (s.length < 8 || s.length > 32) return false
  return /^[A-Z0-9-]+$/.test(s)
}
