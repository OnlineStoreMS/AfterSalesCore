<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Download, Picture } from '@element-plus/icons-vue'
import {
  fetchUnboxingRecord,
  getUnboxingVideoDownload,
  UNBOXING_STATUS_MAP,
  type UnboxingRecord,
} from '../../api/unboxing'

const route = useRoute()
const router = useRouter()
const recordId = computed(() => Number(route.params.id))

const loading = ref(false)
const downloading = ref(false)
const record = ref<UnboxingRecord | null>(null)

async function loadData() {
  loading.value = true
  try {
    record.value = await fetchUnboxingRecord(recordId.value)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

function statusLabel(s: string) {
  return UNBOXING_STATUS_MAP[s]?.label || s
}

function statusType(s: string) {
  return UNBOXING_STATUS_MAP[s]?.type || 'info'
}

function formatDuration(sec?: number) {
  if (!sec) return '-'
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return m > 0 ? `${m}分${s}秒` : `${s}秒`
}

function formatSize(bytes?: number) {
  if (!bytes) return '-'
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

async function handleDownload() {
  downloading.value = true
  try {
    const { url, filename } = await getUnboxingVideoDownload(recordId.value)
    const a = document.createElement('a')
    a.href = url
    a.download = filename || `unboxing-${record.value?.trackingNo || recordId.value}.webm`
    a.target = '_blank'
    a.rel = 'noopener'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  } catch (e) {
    ElMessage.error((e as Error).message || '下载失败')
  } finally {
    downloading.value = false
  }
}
</script>

<template>
  <div v-loading="loading" class="unboxing-detail">
    <div class="top-bar">
      <el-button :icon="ArrowLeft" text @click="router.push('/unboxing')">返回列表</el-button>
      <el-button
        v-if="record?.videoUrl"
        type="primary"
        :icon="Download"
        :loading="downloading"
        @click="handleDownload"
      >
        下载视频
      </el-button>
    </div>

    <el-card v-if="record">
      <template #header>
        <div class="header-row">
          <span>开箱详情 · {{ record.trackingNo }}</span>
          <el-tag :type="statusType(record.status)" size="small">
            {{ statusLabel(record.status) }}
          </el-tag>
        </div>
      </template>

      <el-descriptions :column="2" border class="meta">
        <el-descriptions-item label="快递单号">{{ record.trackingNo }}</el-descriptions-item>
        <el-descriptions-item label="操作人">{{ record.operatorName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ record.createdAt || '-' }}</el-descriptions-item>
        <el-descriptions-item label="视频时长">{{ formatDuration(record.videoDurationSec) }}</el-descriptions-item>
        <el-descriptions-item label="视频大小">{{ formatSize(record.videoSize) }}</el-descriptions-item>
        <el-descriptions-item label="备注">{{ record.remark || '-' }}</el-descriptions-item>
      </el-descriptions>

      <section v-if="record.videoUrl" class="section">
        <h3>开箱视频</h3>
        <div class="video-wrap">
          <video :src="record.videoUrl" controls playsinline class="video-player" />
        </div>
      </section>

      <section v-else class="section">
        <el-empty description="暂无视频" />
      </section>

      <section class="section">
        <h3>
          <el-icon><Picture /></el-icon>
          问题照片（{{ record.photos?.length || 0 }}）
        </h3>
        <div v-if="record.photos?.length" class="photo-grid">
          <div v-for="photo in record.photos" :key="photo.id" class="photo-item">
            <el-image
              :src="photo.photoUrl"
              :preview-src-list="record.photos.map((p) => p.photoUrl)"
              fit="cover"
              class="photo-img"
            />
            <p v-if="photo.issueRemark" class="photo-remark">{{ photo.issueRemark }}</p>
          </div>
        </div>
        <el-empty v-else description="暂无问题照片" />
      </section>
    </el-card>
  </div>
</template>

<style scoped>
.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.meta {
  margin-bottom: 24px;
}
.section {
  margin-top: 24px;
}
.section h3 {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 12px;
  font-size: 16px;
  font-weight: 600;
}
.video-wrap {
  max-width: 720px;
  background: #000;
  border-radius: 8px;
  overflow: hidden;
}
.video-player {
  width: 100%;
  max-height: 480px;
  display: block;
}
.photo-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
}
.photo-item {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  overflow: hidden;
}
.photo-img {
  width: 100%;
  height: 160px;
  display: block;
}
.photo-remark {
  margin: 0;
  padding: 8px 10px;
  font-size: 13px;
  color: #606266;
  background: #fafafa;
}
</style>
