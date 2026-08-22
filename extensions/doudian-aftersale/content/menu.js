/** 抖店左侧菜单：售后 → 服务工单 / 售后工作台 */

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms))
}

function textOf(el) {
  return String(el?.innerText || el?.textContent || '').replace(/\s+/g, ' ').trim()
}

function visible(el) {
  if (!el || !el.getClientRects) return false
  if (!el.getClientRects().length) return false
  const st = window.getComputedStyle(el)
  return st.visibility !== 'hidden' && st.display !== 'none'
}

function siderRoot() {
  return (
    document.querySelector('.auxo-layout-sider') ||
    document.querySelector('[class*="layout-sider"]') ||
    document.querySelector('[class*="side-menu"]') ||
    document.querySelector('[class*="sideMenu"]') ||
    document.querySelector('aside') ||
    document.querySelector('.auxo-menu') ||
    document.body
  )
}

const MENU_TARGETS = {
  service: {
    name: '服务工单',
    hrefNeedles: ['/ffa/task-order/service'],
    readyNeedles: ['/ffa/task-order/service'],
    parent: '售后',
  },
  workbench: {
    name: '售后工作台',
    hrefNeedles: ['/ffa/merchant-aftersale-workbench'],
    readyNeedles: ['/ffa/merchant-aftersale-workbench'],
    parent: '售后',
  },
}

function hrefMatch(el, needles) {
  const href = `${el.getAttribute?.('href') || ''} ${el.getAttribute?.('to') || ''} ${el.dataset?.path || ''}`
  return needles.some((n) => href.includes(n))
}

function findByHref(needles, root) {
  const scope = root || document
  const links = Array.from(scope.querySelectorAll('a[href], [data-path], [to]'))
  return links.find((el) => hrefMatch(el, needles) && visible(el)) ||
    links.find((el) => hrefMatch(el, needles)) ||
    null
}

function ownLabel(el) {
  const own = Array.from(el.childNodes)
    .filter((n) => n.nodeType === Node.TEXT_NODE)
    .map((n) => String(n.textContent || '').trim())
    .filter(Boolean)
    .join('')
  if (own) return own
  const title = el.querySelector?.('.auxo-menu-title-content, [class*="menu-title"], [class*="menuTitle"]')
  if (title) {
    const t = textOf(title)
    if (t && t.length <= 16) return t
  }
  const t = textOf(el)
  return t.length <= 16 ? t : ''
}

function findByLabel(name, root) {
  const scope = root || document
  const nodes = Array.from(
    scope.querySelectorAll(
      'a, button, li, span, div, p, .auxo-menu-item, .auxo-menu-submenu-title, [class*="menu-item"], [class*="menuItem"], [role="menuitem"]',
    ),
  )
  const hits = nodes.filter((el) => ownLabel(el) === name || textOf(el) === name)
  hits.sort((a, b) => textOf(a).length - textOf(b).length)
  return hits.find((el) => visible(el)) || hits[0] || null
}

function findMenuItem(target) {
  const sider = siderRoot()
  return (
    findByHref(target.hrefNeedles, sider) ||
    findByLabel(target.name, sider) ||
    findByHref(target.hrefNeedles, document) ||
    findByLabel(target.name, document)
  )
}

function clickEl(el) {
  if (!el) return false
  const btn = el.closest('a, button, li, [role="menuitem"], .auxo-menu-item, .auxo-menu-submenu-title') || el
  btn.dispatchEvent(new MouseEvent('pointerdown', { bubbles: true, cancelable: true, view: window }))
  btn.dispatchEvent(new MouseEvent('mousedown', { bubbles: true, cancelable: true, view: window }))
  btn.dispatchEvent(new MouseEvent('mouseup', { bubbles: true, cancelable: true, view: window }))
  btn.click()
  return true
}

async function expandParent(parentName) {
  if (!parentName) return
  const sider = siderRoot()
  const parent = findByLabel(parentName, sider) || findByLabel(parentName, document)
  if (!parent) return
  const submenu = parent.closest('.auxo-menu-submenu') || parent.parentElement
  const opened = /open|expanded|active/i.test(`${submenu?.className || ''} ${parent.className || ''}`)
  if (opened) return
  parent.dispatchEvent(new MouseEvent('mouseover', { bubbles: true, cancelable: true, view: window }))
  parent.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true, cancelable: true, view: window }))
  clickEl(parent)
  await sleep(350)
}

function sameOriginDocs() {
  const list = []
  const walk = (doc) => {
    if (!doc || list.includes(doc)) return
    list.push(doc)
    for (const iframe of doc.querySelectorAll('iframe')) {
      try {
        walk(iframe.contentDocument)
      } catch {
        /* cross-origin */
      }
    }
  }
  walk(document)
  return list
}

function hasWorkbenchCards() {
  for (const doc of sameOriginDocs()) {
    if (
      doc.querySelectorAll('[class*="groupTitle"]').length > 0 &&
      doc.querySelectorAll('[class*="groupItems"]').length > 0
    ) {
      return true
    }
  }
  return false
}

function tabTexts() {
  const texts = []
  for (const doc of sameOriginDocs()) {
    const nodes = doc.querySelectorAll('.auxo-tabs-tab, [role="tab"], [class*="tabs-tab"]')
    for (const el of nodes) texts.push(textOf(el))
  }
  return texts
}

function hasServiceStatusTabs() {
  const tabs = tabTexts()
  const hit = (re) => tabs.some((t) => re.test(t))
  return (
    hit(/待处理/) &&
    (hit(/处理中/) || hit(/已逾期/) || hit(/已完结/))
  )
}

function alreadyThere(target) {
  if (target.name === '服务工单') {
    return hasServiceStatusTabs()
  }
  if (target.name === '售后工作台') {
    return hasWorkbenchCards()
  }
  const href = location.href || ''
  return target.readyNeedles.some((n) => href.includes(n))
}

async function clickMenu(key, force) {
  const target = MENU_TARGETS[key]
  if (!target) throw new Error(`未知菜单 ${key}`)
  if (alreadyThere(target)) {
    return { ok: true, already: true, url: location.href, name: target.name }
  }
  let item = findMenuItem(target)
  if (!item || !visible(item)) {
    await expandParent(target.parent)
    item = findMenuItem(target)
  }
  if (!item) {
    throw new Error(`未找到左侧菜单「${target.parent} → ${target.name}」`)
  }
  clickEl(item)
  const tries = target.name === '服务工单' ? 80 : 40
  for (let i = 0; i < tries; i++) {
    if (alreadyThere(target)) {
      return { ok: true, url: location.href, name: target.name }
    }
    await sleep(250)
  }
  throw new Error(`已点击「${target.name}」，但页面未切换，请确认已登录抖店后台`)
}

window.__osmsClickMenu = clickMenu

if (!window.__osmsDoudianMenu) {
  window.__osmsDoudianMenu = true
  chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
    if (msg?.type !== 'AFTERSALE_CLICK_MENU') return
    clickMenu(msg.target, !!msg.force)
      .then((data) => sendResponse(data))
      .catch((e) => sendResponse({ ok: false, error: e instanceof Error ? e.message : String(e) }))
    return true
  })
}
