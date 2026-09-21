<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'

const props = defineProps<{
  status?: string
  timeoutText?: string
  timeoutAction?: string
  timeoutDisplay?: string
  deadlineAt?: string
  remainSeconds?: number
}>()

const nowTick = ref(Date.now())
let tickTimer = 0

onMounted(() => {
  tickTimer = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)
})
onUnmounted(() => {
  if (tickTimer) window.clearInterval(tickTimer)
})

const remainSeconds = computed(() => {
  void nowTick.value
  if (props.deadlineAt) {
    const t = Date.parse(props.deadlineAt)
    if (!Number.isNaN(t)) return Math.max(0, Math.floor((t - Date.now()) / 1000))
  }
  return Math.max(0, Number(props.remainSeconds || 0))
})

const hasClock = computed(() => Boolean(props.deadlineAt) || Number(props.remainSeconds) > 0)

function formatRemain(sec: number) {
  if (sec <= 0) return '已超时'
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  const parts: string[] = []
  if (d) parts.push(`${d}天`)
  if (h) parts.push(`${h}小时`)
  if (d || h || m) parts.push(`${m}分`)
  if (!d && h < 6) parts.push(`${s}秒`)
  return parts.join('') || '不足1分'
}

const timeoutLine = computed(() => {
  if (hasClock.value) {
    if (remainSeconds.value <= 0) {
      return props.timeoutAction ? `已超时 · ${props.timeoutAction}` : (props.timeoutText || props.timeoutDisplay || '已超时')
    }
    const remain = formatRemain(remainSeconds.value)
    return props.timeoutAction ? `剩余 ${remain}后${props.timeoutAction}` : `剩余 ${remain}`
  }
  return (props.timeoutDisplay || props.timeoutText || '').trim()
})

const timeoutTone = computed(() => {
  if (!timeoutLine.value) return ''
  const sec = remainSeconds.value
  if (!hasClock.value) return ''
  if (sec <= 0 || sec < 6 * 3600) return 'danger'
  if (sec < 24 * 3600) return 'warning'
  return ''
})
</script>

<template>
  <div class="aftersale-status">
    <div>{{ status || '—' }}</div>
    <div v-if="timeoutLine" class="timeout" :class="timeoutTone">{{ timeoutLine }}</div>
  </div>
</template>

<style scoped>
.timeout { color: #e6a23c; font-size: 12px; margin-top: 2px; line-height: 1.4; white-space: normal; }
.timeout.warning { color: #e6a23c; font-weight: 600; }
.timeout.danger { color: #f56c6c; font-weight: 700; }
</style>
