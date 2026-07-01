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

/** 先单独获取摄像头，再可选合并麦克风（避免 Windows 上只授权麦克风而无画面） */
export async function requestCameraStream(withAudio = true): Promise<MediaStream> {
  const videoAttempts: MediaStreamConstraints[] = [
    { video: { width: { ideal: 1280 }, height: { ideal: 720 }, frameRate: { ideal: 30 } } },
    { video: { width: { ideal: 640 }, height: { ideal: 480 }, frameRate: { ideal: 30 } } },
    { video: true },
  ]

  let videoStream: MediaStream | null = null
  let lastError: unknown

  for (const constraints of videoAttempts) {
    try {
      videoStream = await navigator.mediaDevices.getUserMedia(constraints)
      if (videoStream.getVideoTracks().length > 0) break
      videoStream.getTracks().forEach((t) => t.stop())
      videoStream = null
    } catch (e) {
      lastError = e
      if (e instanceof DOMException && e.name === 'NotAllowedError') {
        throw e
      }
    }
  }

  if (!videoStream?.getVideoTracks().length) {
    throw lastError ?? new DOMException('No camera found', 'NotFoundError')
  }

  if (withAudio) {
    try {
      const audioStream = await navigator.mediaDevices.getUserMedia({
        audio: { echoCancellation: true, noiseSuppression: true },
      })
      audioStream.getAudioTracks().forEach((t) => videoStream!.addTrack(t))
    } catch {
      // 无麦克风时仍可仅录制视频
    }
  }

  return videoStream
}

export function hasLiveVideoTrack(stream: MediaStream): boolean {
  const track = stream.getVideoTracks()[0]
  return !!track && track.readyState === 'live'
}

export function pickRecorderMimeType(): string {
  const candidates = [
    'video/webm;codecs=vp8,opus',
    'video/webm;codecs=vp9,opus',
    'video/webm;codecs=vp8',
    'video/webm',
  ]
  return candidates.find((m) => MediaRecorder.isTypeSupported(m)) ?? ''
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
