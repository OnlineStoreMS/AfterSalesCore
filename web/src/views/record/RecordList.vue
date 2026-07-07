<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search, VideoCamera } from '@element-plus/icons-vue'
import {
  batchDeleteEdgeRecords,
  fetchEdgeRecords,
  RECORD_STATUS_MAP,
  RECORD_TYPE_LABEL,
  type RecordType,
} from '../../api/edgeRecord'
import { fetchEdgeDevices } from '../../api/edgeDevice'

const route = useRoute()
const router = useRouter()
const recordType = computed(() => (route.meta.recordType as RecordType) || 'unboxing')
const typeLabel = computed(() => RECORD_TYPE_LABEL[recordType.value])
const createPath = computed(() => `/${recordType.value}/create`)

const tableData = ref<Awaited<ReturnType<typeof fetchEdgeRecords>>['list']>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(15)
const trackingNo = ref('')
const edgeId = ref('')
const loading = ref(false)
const selectedIds = ref<number[]>([])
const edgeOptions = ref<{ edgeId: string; name: string }[]>([])

async function loadData() {
  loading.value = true
  try {
    const data = await fetchEdgeRecords({
      type: recordType.value,
      trackingNo: trackingNo.value || undefined,
      edgeId: edgeId.value || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    tableData.value = data.list
    total.value = data.total
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadEdgeOptions() {
  try {
    const devices = await fetchEdgeDevices()
    edgeOptions.value = devices.map((d) => ({ edgeId: d.edgeId, name: d.name }))
  } catch { /* ignore */ }
}

onMounted(() => {
  loadEdgeOptions()
  loadData()
})

function handleSearch() {
  page.value = 1
  loadData()
}

function openDetail(row: { id: number }) {
  router.push(`/${recordType.value}/${row.id}`)
}

function onPageChange(p: number) {
  page.value = p
  loadData()
}

function onSelectionChange(rows: { id: number }[]) {
  selectedIds.value = rows.map((r) => r.id)
}

function statusLabel(s: string) {
  return RECORD_STATUS_MAP[s]?.label || s
}

function statusType(s: string) {
  return RECORD_STATUS_MAP[s]?.type || 'info'
}

async function handleBatchDelete() {
  if (!selectedIds.value.length) return
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${selectedIds.value.length} 条记录？仅可删除云端浏览器录制的记录。`, '批量删除')
    const { deleted } = await batchDeleteEdgeRecords(selectedIds.value)
    ElMessage.success(`已删除 ${deleted} 条`)
    selectedIds.value = []
    loadData()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error((e as Error).message || '删除失败')
  }
}
</script>

<template>
  <div class="record-list">
    <el-card v-loading="loading">
      <template #header>
        <span>{{ typeLabel }}记录</span>
        <el-button type="primary" :icon="Plus" @click="router.push(createPath)">
          录制{{ typeLabel }}
        </el-button>
      </template>

      <div class="toolbar">
        <el-input
          v-model="trackingNo"
          placeholder="按快递单号搜索"
          :prefix-icon="Search"
          clearable
          style="width: 240px"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <el-select v-model="edgeId" placeholder="录制端" clearable style="width: 180px" @change="handleSearch">
          <el-option v-for="opt in edgeOptions" :key="opt.edgeId" :label="`${opt.name} (${opt.edgeId})`" :value="opt.edgeId" />
        </el-select>
        <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>
        <el-button v-if="selectedIds.length" type="danger" @click="handleBatchDelete">批量删除 ({{ selectedIds.length }})</el-button>
      </div>

      <el-table :data="tableData" stripe border @selection-change="onSelectionChange">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="trackingNo" label="快递单号" min-width="150">
          <template #default="{ row }">
            <el-link type="primary" @click="openDetail(row)">{{ row.trackingNo }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="录制端" min-width="140">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.edgeName || row.edgeId }}</el-tag>
            <span class="edge-id">{{ row.edgeId }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="视频" width="100" align="center">
          <template #default="{ row }">
            <span v-if="row.videoUrl"><el-icon><VideoCamera /></el-icon> 有</span>
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="photoCount" label="照片" width="80" align="center" />
        <el-table-column prop="createdAt" label="创建时间" width="170" />
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="onPageChange"
        />
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.record-list :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.toolbar { display: flex; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.pager { margin-top: 16px; display: flex; justify-content: flex-end; }
.muted { color: #909399; }
.edge-id { margin-left: 6px; color: #909399; font-size: 12px; }
</style>
