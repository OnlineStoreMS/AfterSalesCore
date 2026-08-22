const statusEl = document.getElementById('status')
const shopEl = document.getElementById('shop')
const syncEl = document.getElementById('sync')
const apiBaseEl = document.getElementById('apiBase')
const bindBlock = document.getElementById('bindBlock')
const boundBlock = document.getElementById('boundBlock')
const bindCodeEl = document.getElementById('bindCode')

function setStatus(text, cls) {
  statusEl.className = cls || 'warn'
  statusEl.textContent = text
}

function refresh() {
  chrome.runtime.sendMessage({ type: 'AFTERSALE_GET_STATUS' }, (st) => {
    if (chrome.runtime.lastError) {
      setStatus(chrome.runtime.lastError.message, 'bad')
      return
    }
    if (!st?.ok) {
      setStatus(st?.error || '状态读取失败', 'bad')
      return
    }
    apiBaseEl.value = st.apiBaseDisplay || st.apiBase || ''
    if (!st.bound) {
      bindBlock.hidden = false
      boundBlock.hidden = true
      shopEl.textContent = ''
      syncEl.textContent = ''
      const diagEl = document.getElementById('diag')
      if (diagEl) diagEl.textContent = ''
      setStatus(`诊断版未绑定 · v${st.version}`, 'warn')
      return
    }
    bindBlock.hidden = true
    boundBlock.hidden = false
    shopEl.textContent = `店铺：${st.shopName || '-'} · ${st.platform || 'doudian'}`
    syncEl.textContent = st.lastSync
      ? `最近同步：${st.lastSync}${st.lastSyncError ? ` · ${st.lastSyncError}` : ''}`
      : '尚未同步'
    const diagEl = document.getElementById('diag')
    if (diagEl) {
      diagEl.textContent = st.lastDiagRunId
        ? `日志：${st.lastDiagRunId} · ${st.lastDiagUpload || '未上报'}`
        : '日志：尚未上报'
    }
    setStatus(`诊断版已绑定 · 仅手动 · v${st.version}`, 'ok')
  })
}

document.getElementById('saveApiBtn').addEventListener('click', () => {
  chrome.runtime.sendMessage({ type: 'AFTERSALE_SAVE_API', apiBase: apiBaseEl.value }, (res) => {
    if (!res?.ok) setStatus(res?.error || '保存失败', 'bad')
    else setStatus('地址已保存', 'ok')
    refresh()
  })
})

document.getElementById('bindBtn').addEventListener('click', () => {
  setStatus('绑定中…', 'warn')
  chrome.runtime.sendMessage(
    { type: 'AFTERSALE_BIND', bindCode: bindCodeEl.value, apiBase: apiBaseEl.value },
    (res) => {
      if (!res?.ok) {
        setStatus(res?.error || '绑定失败', 'bad')
        return
      }
      bindCodeEl.value = ''
      setStatus('绑定成功', 'ok')
      refresh()
    },
  )
})

document.getElementById('unbindBtn').addEventListener('click', () => {
  chrome.runtime.sendMessage({ type: 'AFTERSALE_UNBIND' }, () => refresh())
})

document.getElementById('syncBtn').addEventListener('click', () => {
  setStatus('同步中…', 'warn')
  chrome.runtime.sendMessage({ type: 'AFTERSALE_SYNC_NOW' }, (res) => {
    if (!res?.ok) setStatus(res?.error || '同步失败', 'bad')
    else if (res.serviceError) setStatus(`已同步售后 ${res.ticketCount || 0}，工单采集失败：${res.serviceError}`, 'warn')
    else setStatus(`已同步售后 ${res.ticketCount || 0} / 工单 ${res.serviceOrderCount || 0}`, 'ok')
    refresh()
  })
})

document.getElementById('openBtn').addEventListener('click', () => {
  setStatus('正在打开抖店工作台…', 'warn')
  chrome.runtime.sendMessage({ type: 'AFTERSALE_OPEN_WORKBENCH' }, (res) => {
    if (chrome.runtime.lastError) {
      setStatus(chrome.runtime.lastError.message, 'bad')
      return
    }
    if (!res?.ok) setStatus(res?.error || '打开失败', 'bad')
    else setStatus('已打开售后工作台，可点立即同步', 'ok')
  })
})

refresh()
