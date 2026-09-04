<script setup lang="ts">
import { computed } from 'vue'
import type { LogisticsTrack } from '../api/shop'
import type { LogisticsLine } from '../utils/ticketLogistics'

const props = defineProps<{
  lines: LogisticsLine[]
  shipNo?: string
  returnNo?: string
  tracks?: LogisticsTrack[]
  raw?: string
  carrier?: string
}>()

const displayTracks = computed(() => (props.tracks || []).slice(0, 5))
const hasTracks = computed(() => displayTracks.value.length > 0)
/** 有退货单号时轨迹归属退货；仅发货时轨迹归属发货 */
const shipTracksEnabled = computed(() => hasTracks.value && !props.returnNo)
const returnTracksEnabled = computed(() => hasTracks.value && !!props.returnNo)

function trackDetail(track: LogisticsTrack) {
  return track.detail || (track.title || track.date ? '' : track.text || '')
}
</script>

<template>
  <div class="ticket-logistics">
    <template v-if="lines.length">
      <div v-for="(line, i) in lines" :key="i" class="logistics-line">
        <template v-if="line.status">
          {{ line.label }}
          <span v-if="line.tone" :class="line.tone">{{ line.status }}</span>
          <span v-else>{{ line.status }}</span>
        </template>
        <span v-else :class="line.tone">{{ line.label }}</span>
      </div>
    </template>
    <pre v-else class="logistics">{{ raw || '—' }}</pre>

    <el-popover
      v-if="shipNo"
      placement="left-start"
      :width="360"
      trigger="hover"
      :disabled="!shipTracksEnabled"
      popper-class="ticket-track-popper"
    >
      <template #reference>
        <div class="tracking" :class="{ link: shipTracksEnabled }">发货单号 {{ shipNo }}</div>
      </template>
      <div class="track-pop">
        <div v-if="shipNo || carrier" class="track-meta">
          {{ [carrier, shipNo].filter(Boolean).join(' ') }}
        </div>
        <ul class="track-list">
          <li v-for="(track, i) in displayTracks" :key="i">
            <div class="track-title">{{ track.title || '物流记录' }}</div>
            <div v-if="track.date" class="track-date">{{ track.date }}</div>
            <div v-if="trackDetail(track)" class="track-detail">{{ trackDetail(track) }}</div>
          </li>
        </ul>
      </div>
    </el-popover>

    <el-popover
      v-if="returnNo"
      placement="left-start"
      :width="360"
      trigger="hover"
      :disabled="!returnTracksEnabled"
      popper-class="ticket-track-popper"
    >
      <template #reference>
        <div class="tracking" :class="{ link: returnTracksEnabled }">退货单号 {{ returnNo }}</div>
      </template>
      <div class="track-pop">
        <div v-if="returnNo || carrier" class="track-meta">
          {{ [carrier, returnNo].filter(Boolean).join(' ') }}
        </div>
        <ul class="track-list">
          <li v-for="(track, i) in displayTracks" :key="i">
            <div class="track-title">{{ track.title || '物流记录' }}</div>
            <div v-if="track.date" class="track-date">{{ track.date }}</div>
            <div v-if="trackDetail(track)" class="track-detail">{{ trackDetail(track) }}</div>
          </li>
        </ul>
      </div>
    </el-popover>

    <div v-if="carrier" class="sub">{{ carrier }}</div>
  </div>
</template>

<style scoped>
.logistics { margin: 0; font: inherit; white-space: pre-line; color: #303133; }
.logistics-line { line-height: 1.5; color: #303133; }
.logistics-line .danger { color: #f56c6c; font-weight: 600; }
.logistics-line .warning { color: #e6a23c; font-weight: 600; }
.logistics-line .ok { color: #67c23a; font-weight: 600; }
.tracking { color: #409eff; font-size: 12px; margin-top: 4px; word-break: break-all; }
.tracking.link { cursor: pointer; }
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
</style>

<style>
.ticket-track-popper .track-meta { color: #909399; font-size: 12px; margin-bottom: 8px; }
.ticket-track-popper .track-list { margin: 0; padding: 0; list-style: none; }
.ticket-track-popper .track-list li { position: relative; padding: 0 0 12px 14px; border-left: 2px solid #e4e7ed; }
.ticket-track-popper .track-list li:last-child { padding-bottom: 0; border-left-color: transparent; }
.ticket-track-popper .track-list li::before {
  content: '';
  position: absolute;
  left: -6px;
  top: 4px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #c0c4cc;
}
.ticket-track-popper .track-list li:first-child::before { background: #409eff; }
.ticket-track-popper .track-title { font-weight: 600; line-height: 1.4; }
.ticket-track-popper .track-date { color: #909399; font-size: 12px; margin-top: 2px; }
.ticket-track-popper .track-detail { color: #606266; font-size: 12px; margin-top: 2px; line-height: 1.5; word-break: break-word; }
</style>
