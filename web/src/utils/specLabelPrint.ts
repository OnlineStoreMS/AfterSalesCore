/** 商品规格标签：尺寸预设与打印排版 */

export type SpecLabelItem = {
  orderNo: string
  sku: string
  productTitle?: string
  shopName?: string
  aftersaleId?: string
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

const SIZE_STORAGE_KEY = 'aftersales.specLabel.size'

export function loadSavedLabelSize(): LabelSize {
  try {
    const raw = localStorage.getItem(SIZE_STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as LabelSize
      if (parsed?.widthMm > 0 && parsed?.heightMm > 0) return parsed
    }
  } catch {
    /* ignore */
  }
  return { widthMm: 40, heightMm: 50 }
}

export function saveLabelSize(size: LabelSize) {
  try {
    localStorage.setItem(SIZE_STORAGE_KEY, JSON.stringify(size))
  } catch {
    /* ignore */
  }
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

function labelHtml(item: SpecLabelItem, size: LabelSize, tone: LayoutTone): string {
  const spec = escapeHtml(resolveSpecText(item))
  const orderNo = escapeHtml(String(item.orderNo || '').trim() || '—')
  const title = escapeHtml(String(item.productTitle || '').trim())
  const shop = escapeHtml(String(item.shopName || '').trim())
  const aftersale = escapeHtml(String(item.aftersaleId || '').trim())

  const extras: string[] = []
  if (tone === 'roomy' && title) {
    extras.push(`<div class="title">${title}</div>`)
  }
  if (tone === 'roomy' && shop) {
    extras.push(`<div class="meta">店铺 ${shop}</div>`)
  }
  if (tone !== 'compact' && aftersale) {
    extras.push(`<div class="meta">售后 ${aftersale}</div>`)
  }

  return `<div class="label tone-${tone}" style="width:${size.widthMm}mm;height:${size.heightMm}mm">
  <div class="inner">
    <div class="spec">${spec}</div>
    <div class="order"><span class="order-key">订单</span><span class="order-no">${orderNo}</span></div>
    ${extras.join('\n')}
  </div>
</div>`
}

export function buildSpecLabelPrintHtml(items: SpecLabelItem[], size: LabelSize): string {
  const tone = layoutTone(size)
  const sheets = expandCopies(items)
  const body = sheets.map((it) => labelHtml(it, size, tone)).join('\n')
  return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8" />
<title>商品规格标签</title>
<style>
  @page {
    size: ${size.widthMm}mm ${size.heightMm}mm;
    margin: 0;
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
    width: 100%;
    height: 100%;
    padding: 1.6mm 1.8mm;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    font-family: "Microsoft YaHei", "PingFang SC", "Noto Sans SC", sans-serif;
  }
  .spec {
    font-weight: 700;
    line-height: 1.25;
    word-break: break-word;
    overflow: hidden;
    flex: 1 1 auto;
  }
  .order {
    flex: 0 0 auto;
    margin-top: 1mm;
    display: flex;
    align-items: baseline;
    gap: 1mm;
    border-top: 0.3mm solid #000;
    padding-top: 1mm;
    font-family: ui-monospace, "SF Mono", Menlo, Consolas, monospace;
  }
  .order-key {
    flex: 0 0 auto;
    font-weight: 700;
  }
  .order-no {
    flex: 1 1 auto;
    word-break: break-all;
    font-weight: 600;
  }
  .title, .meta {
    flex: 0 0 auto;
    margin-top: 0.8mm;
    color: #222;
    word-break: break-word;
    overflow: hidden;
  }

  /* 尺寸自适应：紧凑 / 常规 / 宽松 */
  .tone-compact .spec { font-size: 3.2mm; max-height: 18mm; }
  .tone-compact .order { font-size: 2.2mm; }
  .tone-compact .order-key { font-size: 2mm; }
  .tone-compact .inner { padding: 1.2mm 1.4mm; }

  .tone-normal .spec { font-size: 4mm; max-height: 32mm; }
  .tone-normal .order { font-size: 2.6mm; }
  .tone-normal .meta { font-size: 2.2mm; line-height: 1.3; max-height: 8mm; }

  .tone-roomy .spec { font-size: 7mm; max-height: 55mm; }
  .tone-roomy .order { font-size: 3.6mm; }
  .tone-roomy .title { font-size: 3.2mm; line-height: 1.35; max-height: 28mm; }
  .tone-roomy .meta { font-size: 2.8mm; line-height: 1.35; }

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

/** 打开独立窗口并调起系统打印对话框；@page size 交给浏览器/驱动适配标签纸。 */
export function printSpecLabels(items: SpecLabelItem[], size: LabelSize): boolean {
  if (!items.length) return false
  const html = buildSpecLabelPrintHtml(items, size)
  const w = window.open('', '_blank', 'noopener,noreferrer,width=520,height=720')
  if (!w) return false
  w.document.open()
  w.document.write(html)
  w.document.close()
  const run = () => {
    try {
      w.focus()
      w.print()
    } finally {
      // 部分浏览器打印对话框关闭后才安全关闭窗口
      setTimeout(() => {
        try {
          w.close()
        } catch {
          /* ignore */
        }
      }, 400)
    }
  }
  if (w.document.readyState === 'complete') {
    setTimeout(run, 120)
  } else {
    w.addEventListener('load', () => setTimeout(run, 80))
  }
  return true
}
