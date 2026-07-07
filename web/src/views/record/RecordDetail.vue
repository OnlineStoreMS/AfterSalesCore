<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Download, Picture } from '@element-plus/icons-vue'
import {
  fetchEdgeRecord,
  downloadEdgeRecordVideo,
  RECORD_STATUS_MAP,
  RECORD_TYPE_LABEL,
  type RecordType,
} from '../../api/edgeRecord'

const route = useRoute()
const router = useRouter()
const recordType = computed(() => (route.meta.recordType as RecordType) || 'unboxing')
const typeLabel = computed(() => RECORD_TYPE_LABEL[recordType.value])
const recordId = computed(() => Number(route.params.id))
const listPath = computed(() => `/${recordType.value}`)

const loading = ref(false)
const downloading = ref(false)
const record = ref<Awaited<ReturnType<typeof fetchEdgeRecord>> | null>(null)

async function loadData() {
  loading.value = true
  try {
    record.value = await fetchEdgeRecord(recordId.value)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

function statusLabel(s: string) {
  return RECORD_STATUS_MAP[s]?.label || s
}

function statusType(s: string) {
  return RECORD_STATUS_MAP[s]?.type || 'info'
}

async function handleDownload() {
  downloading.value = true
  try {
    await downloadEdgeRecordVideo(recordId.value)
  } catch (e) {
    ElMessage.error((e as Error).message || '下载失败')
  } finally {
    downloading.value = false
  }
}
</script>

<template>
  <div v-loading="loading" class="record-detail">
    <div class="top-bar">
      <el-button :icon="ArrowLeft" text @click="router.push(listPath)">返回列表</el-button>
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
          <span>{{ typeLabel }}详情 · {{ record.trackingNo }}</span>
          <el-tag :type="statusType(record.status)" size="small">{{ statusLabel(record.status) }}</el-tag>
        </div>
      </template>

      <el-descriptions :column="2" border class="meta">
        <el-descriptions-item label="快递单号">{{ record.trackingNo }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ typeLabel }}</el-descriptions-item>
        <el-descriptions-item label="录制端">{{ record.edgeName || record.edgeId }}</el-descriptions-item>
        <el-descriptions-item label="Edge ID">
          <el-tag size="small">{{ record.edgeId }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ record.createdAt || '-' }}</el-descriptions-item>
        <el-descriptions-item label="完成时间">{{ record.completedAt || '-' }}</el-descriptions-item>
        <el-descriptions-item label="备注" :span="2">{{ record.remark || '-' }}</el-descriptions-item>
      </el-descriptions>

      <section v-if="record.videoUrl" class="section">
        <h3>{{ typeLabel }}视频</h3>
        <div class="video-wrap">
          <video :src="record.videoUrl" controls playsinline class="video-player" />
        </div>
      </section>
      <section v-else class="section"><el-empty description="暂无视频" /></section>

      <section class="section">
        <h3><el-icon><Picture /></el-icon> 照片（{{ record.photos?.length || 0 }}）</h3>
        <div v-if="record.photos?.length" class="photo-grid">
          <div v-for="(photo, idx) in record.photos" :key="idx" class="photo-item">
            <el-image
              :src="photo.url"
              :preview-src-list="record.photos.map((p) => p.url)"
              fit="cover"
              class="photo-img"
            />
          </div>
        </div>
        <el-empty v-else description="暂无照片" />
      </section>
    </el-card>
  </div>
</template>

<style scoped>
.top-bar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
.header-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.meta { margin-bottom: 24px; }
.section { margin-top: 24px; }
.section h3 { display: flex; align-items: center; gap: 6px; margin: 0 0 12px; font-size: 16px; font-weight: 600; }
.video-wrap { max-width: 720px; background: #000; border-radius: 8px; overflow: hidden; }
.video-player { width: 100%; max-height: 480px; display: block; }
.photo-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 16px; }
.photo-item { border: 1px solid #ebeef5; border-radius: 8px; overflow: hidden; }
.photo-img { width: 100%; height: 160px; display: block; }
</style>
