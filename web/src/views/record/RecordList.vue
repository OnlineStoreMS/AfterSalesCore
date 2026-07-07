<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Download, Plus, Search } from '@element-plus/icons-vue'
import {
  batchDeleteEdgeRecords,
  deleteEdgeRecord,
  fetchEdgeRecords,
  downloadEdgeRecordVideo,
  RECORD_STATUS_MAP,
  RECORD_TYPE_LABEL,
  type EdgeRecordListItem,
  type RecordType,
} from '../../api/edgeRecord'
import { fetchEdgeDevices } from '../../api/edgeDevice'
import OrderGoodsCell from '../../components/OrderGoodsCell.vue'

const LOAD_GOODS_KEY = 'aftersales_record_load_goods'

const route = useRoute()
const router = useRouter()
const recordType = computed(() => (route.meta.recordType as RecordType) || 'unboxing')
const typeLabel = computed(() => RECORD_TYPE_LABEL[recordType.value])
const createPath = computed(() => `/${recordType.value}/create`)

const loadGoods = ref(localStorage.getItem(LOAD_GOODS_KEY) === '1')
const goodsLoading = ref(false)

const tableData = ref<EdgeRecordListItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(15)
const trackingNo = ref('')
const edgeId = ref('')
const loading = ref(false)
const downloadingId = ref<number | null>(null)
const selectedIds = ref<number[]>([])
const edgeOptions = ref<{ edgeId: string; name: string }[]>([])

async function loadData() {
  loading.value = true
  goodsLoading.value = loadGoods.value
  try {
    const data = await fetchEdgeRecords({
      type: recordType.value,
      trackingNo: trackingNo.value || undefined,
      edgeId: edgeId.value || undefined,
      withGoods: loadGoods.value,
      page: page.value,
      pageSize: pageSize.value,
    })
    tableData.value = data.list
    total.value = data.total
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
    goodsLoading.value = false
  }
}

function onLoadGoodsChange(val: boolean) {
  localStorage.setItem(LOAD_GOODS_KEY, val ? '1' : '0')
  if (val) {
    loadData()
  } else {
    tableData.value = tableData.value.map((row) => ({ ...row, goods: undefined }))
  }
}

async function loadEdgeOptions() {
  try {
    const devices = await fetchEdgeDevices()
    edgeOptions.value = devices.map((d) => ({ edgeId: d.edgeId, name: d.name }))
  } catch { /* ignore */ }
}

function resetAndLoad() {
  page.value = 1
  selectedIds.value = []
  loadData()
}

onMounted(() => {
  loadEdgeOptions()
  loadData()
})

watch(recordType, () => {
  trackingNo.value = ''
  edgeId.value = ''
  resetAndLoad()
})

function handleSearch() {
  page.value = 1
  loadData()
}

function openDetail(row: EdgeRecordListItem) {
  router.push(`/${recordType.value}/${row.id}`)
}

function onPageChange(p: number) {
  page.value = p
  loadData()
}

function onSelectionChange(rows: EdgeRecordListItem[]) {
  selectedIds.value = rows.map((r) => r.id)
}

function statusLabel(s: string) {
  return RECORD_STATUS_MAP[s]?.label || s
}

function statusType(s: string) {
  return RECORD_STATUS_MAP[s]?.type || 'info'
}

function edgeLabel(row: EdgeRecordListItem) {
  return row.edgeName || row.edgeId
}

async function handleDownload(row: EdgeRecordListItem) {
  if (!row.videoUrl) {
    ElMessage.warning('该记录暂无视频')
    return
  }
  downloadingId.value = row.id
  try {
    await downloadEdgeRecordVideo(row.id)
  } catch (e) {
    ElMessage.error((e as Error).message || '下载失败')
  } finally {
    downloadingId.value = null
  }
}

async function handleDelete(row: EdgeRecordListItem) {
  try {
    await ElMessageBox.confirm(`确定删除单号 ${row.trackingNo} 的记录？`, '删除')
    await deleteEdgeRecord(row.id)
    ElMessage.success('已删除')
    loadData()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error((e as Error).message || '删除失败')
  }
}

async function handleBatchDelete() {
  if (!selectedIds.value.length) return
  try {
    await ElMessageBox.confirm(`确定删除选中的 ${selectedIds.value.length} 条记录？`, '批量删除')
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
    <el-card v-loading="loading" class="record-card">
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
          class="toolbar-input"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <el-select v-model="edgeId" placeholder="录制端" clearable class="toolbar-select" @change="handleSearch">
          <el-option
            v-for="opt in edgeOptions"
            :key="opt.edgeId"
            :label="opt.name"
            :value="opt.edgeId"
          />
        </el-select>
        <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>
        <el-switch
          v-model="loadGoods"
          inline-prompt
          active-text="商品"
          inactive-text="商品"
          :loading="goodsLoading"
          @change="onLoadGoodsChange"
        />
        <span class="goods-tip muted">开启后按快递单号查询商品（较慢）</span>
        <el-button v-if="selectedIds.length" type="danger" @click="handleBatchDelete">
          批量删除 ({{ selectedIds.length }})
        </el-button>
      </div>

      <el-table :data="tableData" stripe border class="record-table" @selection-change="onSelectionChange">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="trackingNo" label="快递单号" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            <el-link type="primary" @click="openDetail(row)">{{ row.trackingNo }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="商品" min-width="280">
          <template #default="{ row }">
            <OrderGoodsCell :goods="row.goods" />
          </template>
        </el-table-column>
        <el-table-column label="录制端" min-width="120" show-overflow-tooltip>
          <template #default="{ row }">
            <el-tooltip :content="row.edgeId" placement="top">
              <el-tag size="small" type="info">{{ edgeLabel(row) }}</el-tag>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="photoCount" label="照片" min-width="72" align="center" />
        <el-table-column prop="createdAt" label="创建时间" min-width="168" show-overflow-tooltip />
        <el-table-column label="操作" min-width="168" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openDetail(row)">查看</el-button>
            <el-button
              type="primary"
              link
              :icon="Download"
              :disabled="!row.videoUrl"
              :loading="downloadingId === row.id"
              @click="handleDownload(row)"
            >
              下载
            </el-button>
            <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
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
.record-list {
  width: 100%;
}
.record-card {
  width: 100%;
}
.record-list :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.record-list :deep(.el-card__body) {
  width: 100%;
}
.record-table {
  width: 100%;
}
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
  align-items: center;
}
.toolbar-input {
  width: 240px;
  max-width: 100%;
}
.toolbar-select {
  width: 160px;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.goods-tip {
  font-size: 12px;
}
.muted {
  color: #909399;
}
</style>
