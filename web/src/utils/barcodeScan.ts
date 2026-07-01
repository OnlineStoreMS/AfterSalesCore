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

/** 依次尝试多种约束，兼容 USB 摄像头（无 environment facingMode）及无麦克风环境 */
export async function requestCameraStream(): Promise<MediaStream> {
  const attempts: MediaStreamConstraints[] = [
    {
      video: { facingMode: { ideal: 'environment' }, width: { ideal: 1280 }, height: { ideal: 720 } },
      audio: { echoCancellation: true },
    },
    {
      video: { width: { ideal: 1280 }, height: { ideal: 720 } },
      audio: true,
    },
    { video: true, audio: true },
    { video: true },
  ]

  let lastError: unknown
  for (const constraints of attempts) {
    try {
      return await navigator.mediaDevices.getUserMedia(constraints)
    } catch (e) {
      lastError = e
      if (e instanceof DOMException && e.name === 'NotAllowedError') {
        throw e
      }
    }
  }
  throw lastError
}

export function formatCameraError(err: unknown): string {
  if (err instanceof DOMException) {
    switch (err.name) {
      case 'NotFoundError':
      case 'OverconstrainedError':
        return '未找到可用摄像头，请确认 USB 摄像头已连接并在系统中可用，或改用手动输入快递单号。'
      case 'NotAllowedError':
        return '浏览器已拒绝摄像头权限，请在地址栏站点设置中允许摄像头后刷新页面。'
      case 'NotReadableError':
        return '摄像头无法打开，可能被其他程序占用，请关闭后重试。'
      default:
        break
    }
  }
  return (err as Error).message || '无法访问摄像头，请检查权限与设备连接'
}
