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

function alreadyThere(target) {
  const href = location.href || ''
  if (target.readyNeedles.some((n) => href.includes(n))) return true
  if (target.name === '服务工单') {
    const tabs = Array.from(document.querySelectorAll('.auxo-tabs-tab')).map((el) => textOf(el))
    return tabs.some((t) => /^待处理\s*\d+/.test(t)) && tabs.some((t) => /^处理中\s*\d+/.test(t))
  }
  if (target.name === '售后工作台') {
    return !!document.querySelector('[class*="groupTitle"]') && !!document.querySelector('[class*="groupItems"]')
  }
  return false
}

async function clickMenu(key) {
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
  for (let i = 0; i < 40; i++) {
    if (alreadyThere(target)) {
      return { ok: true, url: location.href, name: target.name }
    }
    await sleep(250)
  }
  throw new Error(`已点击「${target.name}」，但页面未切换，请确认已登录抖店后台`)
}

if (!window.__osmsDoudianMenu) {
  window.__osmsDoudianMenu = true
  chrome.runtime.onMessage.addListener((msg, _sender, sendResponse) => {
    if (msg?.type !== 'AFTERSALE_CLICK_MENU') return
    clickMenu(msg.target)
      .then((data) => sendResponse(data))
      .catch((e) => sendResponse({ ok: false, error: e instanceof Error ? e.message : String(e) }))
    return true
  })
}
