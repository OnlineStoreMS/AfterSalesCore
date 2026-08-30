/** 抖店页右下角：售后同步工作记录（对齐快递助手打单面板） */
;(() => {
  if (window !== window.top) return
  if (window.__osmsAftersalePanel) return
  window.__osmsAftersalePanel = true

  const PANEL_ID = 'osms-aftersale-panel'
  const POS_KEY = 'aftersalePanelPos'
  const state = {
    bound: false,
    shopName: '',
    online: true,
    syncing: false,
    lastSync: '',
    lastError: '',
    logs: [],
  }

  function escapeHtml(s) {
    return String(s || '')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
  }

  function formatTime(ts) {
    const d = ts ? new Date(ts) : new Date()
    if (Number.isNaN(d.getTime())) return ''
    return d.toLocaleTimeString()
  }

  function clampPos(left, top, el) {
    const w = el.offsetWidth || 320
    const h = el.offsetHeight || 80
    const maxL = Math.max(8, window.innerWidth - w - 8)
    const maxT = Math.max(8, window.innerHeight - h - 8)
    return {
      left: Math.min(maxL, Math.max(8, left)),
      top: Math.min(maxT, Math.max(8, top)),
    }
  }

  function applyPos(el, pos) {
    if (!pos || !Number.isFinite(pos.left) || !Number.isFinite(pos.top)) return
    el.style.left = `${pos.left}px`
    el.style.top = `${pos.top}px`
    el.style.right = 'auto'
    el.style.bottom = 'auto'
  }

  function bindDrag(el) {
    const hd = el.querySelector('.hd')
    if (!hd || hd.dataset.dragBound) return
    hd.dataset.dragBound = '1'
    let dragging = false
    let startX = 0
    let startY = 0
    let startL = 0
    let startT = 0
    hd.addEventListener('mousedown', (e) => {
      if (e.button !== 0) return
      if (e.target instanceof Element && e.target.closest('button')) return
      const r = el.getBoundingClientRect()
      dragging = true
      startX = e.clientX
      startY = e.clientY
      startL = r.left
      startT = r.top
      applyPos(el, { left: startL, top: startT })
      e.preventDefault()
    })
    window.addEventListener('mousemove', (e) => {
      if (!dragging) return
      applyPos(el, clampPos(startL + e.clientX - startX, startT + e.clientY - startY, el))
    })
    window.addEventListener('mouseup', () => {
      if (!dragging) return
      dragging = false
      const r = el.getBoundingClientRect()
      const pos = clampPos(r.left, r.top, el)
      applyPos(el, pos)
      try {
        chrome.storage.local.set({ [POS_KEY]: pos })
      } catch {
        /* ignore */
      }
    })
    window.addEventListener('resize', () => {
      const r = el.getBoundingClientRect()
      if (el.style.left) applyPos(el, clampPos(r.left, r.top, el))
    })
  }

  function applyPayload(data) {
    if (!data) return
    if (typeof data.bound === 'boolean') state.bound = data.bound
    if (data.shopName != null) state.shopName = data.shopName
    if (typeof data.online === 'boolean') state.online = data.online
    if (typeof data.syncing === 'boolean') state.syncing = data.syncing
    if (data.lastSync != null) state.lastSync = data.lastSync
    if (data.lastSyncError != null) state.lastError = data.lastSyncError
    if (Array.isArray(data.logs)) state.logs = data.logs
    renderPanel()
  }

  function renderPanel() {
    let el = document.getElementById(PANEL_ID)
    if (!el) {
      el = document.createElement('div')
      el.id = PANEL_ID
      el.innerHTML = `
        <div class="hd">
          <strong>OSMS 售后同步</strong>
          <button type="button" data-act="min">—</button>
        </div>
        <div class="bd">
          <div class="status"></div>
          <div class="logs"></div>
          <div class="actions">
            <button type="button" data-act="sync">立即同步</button>
            <button type="button" data-act="open">打开工作台</button>
            <button type="button" data-act="clear">清除</button>
          </div>
          <div class="tip">保持抖店标签打开。服务端约 5 分钟调度同步，也可点「立即同步」。</div>
        </div>
      `
      const style = document.createElement('style')
      style.textContent = `
        #${PANEL_ID}{
          position:fixed; right:16px; bottom:16px; z-index:2147483646;
          width:320px; max-height:52vh; overflow:hidden;
          background:#0f172a; color:#e2e8f0; border-radius:12px;
          box-shadow:0 12px 40px rgba(0,0,0,.35); font:12px/1.45 system-ui,sans-serif;
        }
        #${PANEL_ID}.min .bd{display:none}
        #${PANEL_ID} .hd{
          display:flex; align-items:center; justify-content:space-between;
          padding:10px 12px; background:#1e293b; cursor:move; user-select:none;
        }
        #${PANEL_ID} .hd button,#${PANEL_ID} .actions button{
          border:0; border-radius:6px; padding:4px 8px; cursor:pointer;
          background:#334155; color:#f8fafc;
        }
        #${PANEL_ID} .bd{padding:10px 12px 12px}
        #${PANEL_ID} .status{margin-bottom:8px; color:#93c5fd}
        #${PANEL_ID} .logs{
          max-height:180px; overflow:auto; background:#020617; border-radius:8px;
          padding:8px; color:#cbd5e1; margin-bottom:8px;
        }
        #${PANEL_ID} .logs .line{margin:0 0 4px}
        #${PANEL_ID} .logs .error{color:#fca5a5}
        #${PANEL_ID} .logs .ok{color:#86efac}
        #${PANEL_ID} .actions{display:flex; gap:6px; flex-wrap:wrap}
        #${PANEL_ID} .actions button[data-act="sync"]{background:#2563eb}
        #${PANEL_ID} .actions button:disabled{opacity:.55; cursor:default}
        #${PANEL_ID} .tip{margin-top:8px; color:#94a3b8}
      `
      document.documentElement.appendChild(style)
      document.documentElement.appendChild(el)
      bindDrag(el)
      try {
        chrome.storage.local.get(POS_KEY, (res) => {
          if (res?.[POS_KEY]) applyPos(el, res[POS_KEY])
        })
      } catch {
        /* ignore */
      }
      el.addEventListener('click', (e) => {
        const t = e.target
        if (!(t instanceof HTMLElement)) return
        const act = t.getAttribute('data-act')
        if (act === 'min') el.classList.toggle('min')
        if (act === 'sync') {
          chrome.runtime.sendMessage({ type: 'AFTERSALE_SYNC_NOW' }, () => refresh())
        }
        if (act === 'open') {
          chrome.runtime.sendMessage({ type: 'AFTERSALE_OPEN_WORKBENCH' }, () => refresh())
        }
        if (act === 'clear') {
          chrome.runtime.sendMessage({ type: 'AFTERSALE_CLEAR_WORKLOG' }, () => refresh())
        }
      })
    }

    const status = el.querySelector('.status')
    const logs = el.querySelector('.logs')
    const syncBtn = el.querySelector('[data-act="sync"]')
    if (status) {
      if (!state.bound) {
        status.textContent = '未绑定店铺 · 请在插件弹窗填写绑定码'
      } else {
        const parts = [
          state.shopName || '已绑定',
          state.online ? '在线' : '心跳异常',
        ]
        if (state.syncing) parts.push('同步中…')
        else if (state.lastSync) parts.push(`最近同步 ${state.lastSync}`)
        else parts.push('尚未同步')
        if (state.lastError && !state.syncing) parts.push(state.lastError)
        status.textContent = parts.join(' · ')
      }
    }
    if (logs) {
      const rows = state.logs.slice(-40)
      logs.innerHTML = rows.length
        ? rows
            .map((l) => {
              const cls = l.level === 'error' ? 'error' : l.level === 'ok' ? 'ok' : ''
              return `<div class="line ${cls}">[${escapeHtml(formatTime(l.t))}] ${escapeHtml(l.msg)}</div>`
            })
            .join('')
        : '<div class="line">等待同步…</div>'
      logs.scrollTop = logs.scrollHeight
    }
    if (syncBtn instanceof HTMLButtonElement) syncBtn.disabled = !state.bound || state.syncing
  }

  function refresh() {
    chrome.runtime.sendMessage({ type: 'AFTERSALE_GET_STATUS' }, (st) => {
      if (chrome.runtime.lastError || !st?.ok) return
      applyPayload(st)
    })
  }

  chrome.runtime.onMessage.addListener((msg) => {
    if (msg?.type === 'AFTERSALE_WORK_LOG') applyPayload(msg)
  })

  chrome.storage.onChanged.addListener((changes, area) => {
    if (area !== 'local') return
    if (changes.aftersaleWorkLogs || changes.aftersaleLastSync || changes.aftersaleLastError) {
      refresh()
    }
  })

  renderPanel()
  refresh()
})()
