/** 商品规格标签：尺寸预设与打印排版 */

import { pinyin } from 'pinyin-pro'
import { code128Svg } from './code128'

export type SpecLabelItem = {
  id?: number
  orderNo: string
  sku: string
  productTitle?: string
  shopName?: string
  aftersaleId?: string
  /** 入库时间（退回物流签收时间） */
  inboundAt?: string
  /** 已打印次数（展示用） */
  labelPrintCount?: number
  /** 该行默认打印份数（如申请件数） */
  copies?: number
}

export type LabelSizePreset = {
  id: string
  label: string
  widthMm: number
  heightMm: number
}

export const LABEL_SIZE_PRESETS: LabelSizePreset[] = [
  { id: '40x30', label: '40×30 mm', widthMm: 40, heightMm: 30 },
  { id: '40x50', label: '40×50 mm', widthMm: 40, heightMm: 50 },
  { id: '50x30', label: '50×30 mm', widthMm: 50, heightMm: 30 },
  { id: '60x40', label: '60×40 mm', widthMm: 60, heightMm: 40 },
  { id: '70x50', label: '70×50 mm', widthMm: 70, heightMm: 50 },
  { id: '76x130', label: '76×130 mm', widthMm: 76, heightMm: 130 },
  { id: '100x50', label: '100×50 mm', widthMm: 100, heightMm: 50 },
]

export type LabelSize = { widthMm: number; heightMm: number }

/** portrait=纵向（按预设宽×高）；landscape=横向（宽高对调） */
export type LabelOrientation = 'portrait' | 'landscape'

export type LabelPrintPrefs = {
  size: LabelSize
  orientation: LabelOrientation
}

const PREFS_STORAGE_KEY = 'aftersales.specLabel.prefs'
const SIZE_STORAGE_KEY = 'aftersales.specLabel.size'

export function resolvePrintSize(size: LabelSize, orientation: LabelOrientation = 'portrait'): LabelSize {
  if (orientation === 'landscape') {
    return { widthMm: size.heightMm, heightMm: size.widthMm }
  }
  return { widthMm: size.widthMm, heightMm: size.heightMm }
}

export function loadSavedLabelPrefs(): LabelPrintPrefs {
  try {
    const raw = localStorage.getItem(PREFS_STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<LabelPrintPrefs>
      const w = Number(parsed?.size?.widthMm)
      const h = Number(parsed?.size?.heightMm)
      const orientation: LabelOrientation =
        parsed?.orientation === 'landscape' ? 'landscape' : 'portrait'
      if (w > 0 && h > 0) return { size: { widthMm: w, heightMm: h }, orientation }
    }
  } catch {
    /* ignore */
  }
  // 兼容旧版只存尺寸的 key
  try {
    const legacy = localStorage.getItem(SIZE_STORAGE_KEY)
    if (legacy) {
      const parsed = JSON.parse(legacy) as LabelSize
      if (parsed?.widthMm > 0 && parsed?.heightMm > 0) {
        return { size: { widthMm: parsed.widthMm, heightMm: parsed.heightMm }, orientation: 'portrait' }
      }
    }
  } catch {
    /* ignore */
  }
  return { size: { widthMm: 40, heightMm: 50 }, orientation: 'portrait' }
}

export function loadSavedLabelSize(): LabelSize {
  return loadSavedLabelPrefs().size
}

export function saveLabelPrefs(prefs: LabelPrintPrefs) {
  try {
    localStorage.setItem(PREFS_STORAGE_KEY, JSON.stringify(prefs))
    localStorage.setItem(SIZE_STORAGE_KEY, JSON.stringify(prefs.size))
  } catch {
    /* ignore */
  }
}

export function saveLabelSize(size: LabelSize, orientation: LabelOrientation = 'portrait') {
  saveLabelPrefs({ size, orientation })
}

export function orderBarcodeSvg(orderNo: string): string {
  return code128Svg(orderNo)
}

/** 店铺名拼音首字母，小写。例：杭州精胜单车 → hzjsdc */
export function shopPinyinInitials(name: string): string {
  const raw = String(name || '').trim()
  if (!raw) return ''
  const parts = pinyin(raw, {
    pattern: 'first',
    toneType: 'none',
    type: 'array',
    nonZh: 'consecutive',
  })
  return parts.join('').toLowerCase().replace(/[^a-z0-9]/g, '')
}

export function resolveSpecText(item: SpecLabelItem): string {
  const sku = String(item.sku || '').trim()
  if (sku) return sku
  return String(item.productTitle || '').trim() || '（无规格）'
}

type LayoutTone = 'compact' | 'normal' | 'roomy'

function layoutTone(size: LabelSize): LayoutTone {
  const area = size.widthMm * size.heightMm
  if (size.heightMm <= 32 || area <= 1400) return 'compact'
  if (size.heightMm >= 100 || area >= 6000) return 'roomy'
  return 'normal'
}

function escapeHtml(s: string): string {
  return String(s || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function expandCopies(items: SpecLabelItem[]): SpecLabelItem[] {
  const out: SpecLabelItem[] = []
  for (const item of items) {
    const n = Math.min(Math.max(Number(item.copies) || 1, 1), 99)
    for (let i = 0; i < n; i++) out.push(item)
  }
  return out
}

function specFontMm(spec: string, size: LabelSize, reservedMm: number): number {
  const width = Math.max(size.widthMm - 6, 8)
  const height = Math.max(size.heightMm - reservedMm, 6)
  const chars = Math.max([...spec].length, 1)
  let font = Math.min(7.2, height / 1.25, width / 2)
  for (; font >= 2.1; font -= 0.15) {
    const perLine = Math.max(1, Math.floor(width / (font * 0.95)))
    const lines = Math.ceil(chars / perLine)
    if (lines * font * 1.22 <= height) return Math.round(font * 10) / 10
  }
  return 2.1
}

function labelHtml(item: SpecLabelItem, size: LabelSize, tone: LayoutTone): string {
  const specRaw = resolveSpecText(item)
  const spec = escapeHtml(specRaw)
  const orderRaw = String(item.orderNo || '').trim()
  const orderNo = escapeHtml(orderRaw || '—')
  const barcode = orderRaw ? code128Svg(orderRaw) : ''
  const barcodeH = tone === 'compact' ? 8 : tone === 'roomy' ? 16 : 12
  const reserved = barcodeH + 10
  const specSize = specFontMm(specRaw, size, reserved)
  const barcodeBlock = barcode
    ? `<div class="barcode" style="height:${barcodeH}mm">${barcode}</div>`
    : ''
  const shopMark = escapeHtml(shopPinyinInitials(String(item.shopName || '')))
  const shopBlock = shopMark ? `<div class="shop-mark">${shopMark}</div>` : ''

  return `<div class="label tone-${tone}" style="width:${size.widthMm}mm;height:${size.heightMm}mm">
  <div class="inner">
    ${barcodeBlock}
    <div class="order-no">${orderNo}</div>
    <div class="spec" style="font-size:${specSize}mm">${spec}</div>
    ${shopBlock}
  </div>
</div>`
}

export function buildSpecLabelPrintHtml(
  items: SpecLabelItem[],
  size: LabelSize,
  orientation: LabelOrientation = 'portrait',
): string {
  const printSize = resolvePrintSize(size, orientation)
  const tone = layoutTone(printSize)
  const sheets = expandCopies(items)
  const body = sheets.map((it) => labelHtml(it, printSize, tone)).join('\n')
  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8" />
<title>商品规格标签</title>
<style>
  @page {
    size: ${printSize.widthMm}mm ${printSize.heightMm}mm;
    margin: 2mm;
  }
  * { box-sizing: border-box; }
  html, body {
    margin: 0;
    padding: 0;
    background: #fff;
    color: #000;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }
  .label {
    page-break-after: always;
    break-after: page;
    overflow: hidden;
    border: 0;
  }
  .label:last-child {
    page-break-after: auto;
    break-after: auto;
  }
  .inner {
    position: relative;
    width: 100%;
    height: 100%;
    padding: 1.6mm 2mm 2mm;
    display: flex;
    flex-direction: column;
    font-family: "Microsoft YaHei", "PingFang SC", "Noto Sans SC", sans-serif;
  }
  .barcode {
    flex: 0 0 auto;
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .barcode svg {
    width: 92%;
    height: 100%;
    margin: 0 auto;
    display: block;
    shape-rendering: crispEdges;
  }
  .order-no {
    flex: 0 0 auto;
    margin-top: 0.4mm;
    text-align: center;
    font-family: ui-monospace, "SF Mono", Menlo, Consolas, monospace;
    font-weight: 700;
    letter-spacing: 0.1mm;
    word-break: break-all;
    line-height: 1.15;
  }
  .spec {
    flex: 1 1 auto;
    min-height: 0;
    margin-top: 3.6mm;
    padding-bottom: 3.2mm;
    font-weight: 700;
    line-height: 1.22;
    word-break: break-word;
    overflow: hidden;
  }
  .shop-mark {
    position: absolute;
    right: 1.4mm;
    bottom: 1.2mm;
    font-size: 2.4mm;
    font-weight: 700;
    letter-spacing: 0.15mm;
    line-height: 1;
    font-family: ui-monospace, "SF Mono", Menlo, Consolas, monospace;
  }
  .tone-compact .shop-mark { font-size: 2mm; right: 1mm; bottom: 0.8mm; }
  .tone-compact .spec { margin-top: 2.4mm; }

  .tone-compact .order-no { font-size: 2.2mm; }
  .tone-compact .inner { padding: 1.2mm 1.6mm 1.4mm; }
  .tone-normal .order-no { font-size: 2.6mm; }
  .tone-roomy .order-no { font-size: 3.4mm; }

  @media screen {
    body {
      padding: 12px;
      background: #f0f2f5;
      display: flex;
      flex-wrap: wrap;
      gap: 12px;
      align-content: flex-start;
    }
    .label {
      background: #fff;
      box-shadow: 0 1px 4px rgba(0,0,0,.12);
      outline: 1px dashed #c0c4cc;
      page-break-after: auto;
    }
  }
</style>
</head>
<body>
${body}
</body>
</html>`
}

const PRINT_IFRAME_ID = 'aftersales-spec-label-print-frame'

/** 用隐藏 iframe 调起打印（不依赖弹窗，避免被浏览器拦截）。 */
export function printSpecLabels(
  items: SpecLabelItem[],
  size: LabelSize,
  orientation: LabelOrientation = 'portrait',
): boolean {
  if (!items.length || typeof document === 'undefined') return false
  const html = buildSpecLabelPrintHtml(items, size, orientation)

  let frame = document.getElementById(PRINT_IFRAME_ID) as HTMLIFrameElement | null
  if (frame) {
    frame.remove()
  }
  frame = document.createElement('iframe')
  frame.id = PRINT_IFRAME_ID
  frame.setAttribute('aria-hidden', 'true')
  frame.style.cssText =
    'position:fixed;right:0;bottom:0;width:0;height:0;border:0;opacity:0;pointer-events:none;'
  document.body.appendChild(frame)

  const win = frame.contentWindow
  const doc = frame.contentDocument || win?.document
  if (!win || !doc) {
    frame.remove()
    return false
  }

  doc.open()
  doc.write(html)
  doc.close()

  const cleanup = () => {
    setTimeout(() => {
      try {
        frame?.remove()
      } catch {
        /* ignore */
      }
    }, 800)
  }

  const run = () => {
    try {
      win.focus()
      win.print()
    } catch {
      cleanup()
      return
    }
    cleanup()
  }

  // 等样式/字体就绪后再调打印
  if (doc.readyState === 'complete') {
    setTimeout(run, 150)
  } else {
    frame.addEventListener('load', () => setTimeout(run, 100), { once: true })
    setTimeout(run, 400)
  }
  return true
}
