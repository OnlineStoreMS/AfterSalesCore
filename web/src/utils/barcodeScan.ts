import { BrowserMultiFormatReader } from '@zxing/browser'
import { extractTrackingNoFromText, normalizeTrackingNo } from './trackingNo'

export async function scanBarcodeFromVideo(video: HTMLVideoElement): Promise<string | null> {
  const reader = new BrowserMultiFormatReader()
  try {
    const result = await reader.decodeOnceFromVideoElement(video)
    const text = result.getText()
    return extractTrackingNoFromText(text) || normalizeTrackingNo(text)
  } catch {
    return null
  }
}

export async function scanBarcodeFromImage(img: HTMLImageElement | HTMLCanvasElement): Promise<string | null> {
  const reader = new BrowserMultiFormatReader()
  try {
    const result = await reader.decodeFromImageElement(img as HTMLImageElement)
    const text = result.getText()
    return extractTrackingNoFromText(text) || normalizeTrackingNo(text)
  } catch {
    return null
  }
}

export function captureVideoFrame(video: HTMLVideoElement): HTMLCanvasElement {
  const canvas = document.createElement('canvas')
  canvas.width = video.videoWidth || 640
  canvas.height = video.videoHeight || 480
  const ctx = canvas.getContext('2d')
  if (ctx) {
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height)
  }
  return canvas
}
