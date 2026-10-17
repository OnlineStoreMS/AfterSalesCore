/** CODE128（数字串优先用 Code C，条更窄，适合订单号）。 */

const PATTERNS = [
  '11011001100', '11001101100', '11001100110', '10010011000', '10010001100',
  '10001001100', '10011001000', '10011000100', '10001100100', '11001001000',
  '11001000100', '11000100100', '10110011100', '10011011100', '10011001110',
  '10111001100', '10011101100', '10011100110', '11001110010', '11001011100',
  '11001001110', '11011100100', '11001110100', '11101101110', '11101001100',
  '11100101100', '11100100110', '11101100100', '11100110100', '11100110010',
  '11011011000', '11011000110', '11000110110', '10100011000', '10001011000',
  '10001000110', '10110001000', '10001101000', '10001100010', '11010001000',
  '11000101000', '11000100010', '10110111000', '10110001110', '10001101110',
  '10111011000', '10111000110', '10001110110', '11101110110', '11010001110',
  '11000101110', '11011101000', '11011100010', '11011101110', '11101011000',
  '11101000110', '11100010110', '11101101000', '11101100010', '11100011010',
  '11101111010', '11001000010', '11110001010', '10100110000', '10100001100',
  '10010110000', '10010000110', '10000101100', '10000100110', '10110010000',
  '10110000100', '10011010000', '10011000010', '10000110100', '10000110010',
  '11000010010', '11001010000', '11110111010', '11000010100', '10001111010',
  '10100111100', '10010111100', '10010011110', '10111100100', '10011110100',
  '10011110010', '11110100100', '11110010100', '11110010010', '11011011110',
  '11011110110', '11110110110', '10101111000', '10100011110', '10001011110',
  '10111101000', '10111100010', '11110101000', '11110100010', '10111011110',
  '10111101110', '11101011110', '11110101110', '11010000100', '11010010000',
  '11010011100', '1100011101011',
]

const START_B = 104
const START_C = 105
const CODE_C = 99
const STOP = 106

function encode(raw: string): number[] {
  const text = String(raw || '').trim()
  if (!text) return []
  const codes: number[] = []
  const digits = /^\d+$/.test(text)
  if (digits && text.length >= 2) {
    let i = 0
    if (text.length % 2 === 1) {
      codes.push(START_B, text.charCodeAt(0) - 32, CODE_C)
      i = 1
    } else {
      codes.push(START_C)
    }
    for (; i + 1 < text.length; i += 2) {
      codes.push(Number(text.slice(i, i + 2)))
    }
  } else {
    codes.push(START_B)
    for (const ch of text) {
      const v = ch.charCodeAt(0) - 32
      if (v < 0 || v > 94) return []
      codes.push(v)
    }
  }
  let sum = codes[0]
  for (let i = 1; i < codes.length; i++) sum += codes[i] * i
  codes.push(sum % 103, STOP)
  return codes
}

/** 订单号条形码 SVG。无法编码时返回空字符串。 */
export function code128Svg(raw: string): string {
  if (PATTERNS.length !== 107) return ''
  const codes = encode(raw)
  if (!codes.length) return ''
  let bits = ''
  for (const code of codes) {
    const pat = PATTERNS[code]
    if (!pat) return ''
    bits += pat
  }
  const quiet = 10
  const barH = 40
  const total = bits.length + quiet * 2
  let x = quiet
  let rects = ''
  for (let i = 0; i < bits.length; ) {
    if (bits[i] !== '1') {
      x += 1
      i += 1
      continue
    }
    let w = 1
    while (bits[i + w] === '1') w += 1
    rects += `<rect x="${x}" y="0" width="${w}" height="${barH}"/>`
    x += w
    i += w
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${total} ${barH}" preserveAspectRatio="xMidYMid meet" role="img"><g fill="#000">${rects}</g></svg>`
}
