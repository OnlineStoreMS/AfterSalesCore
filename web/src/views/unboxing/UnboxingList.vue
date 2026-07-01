<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus, Search, VideoCamera } from '@element-plus/icons-vue'
import {
  fetchUnboxingRecords,
  UNBOXING_STATUS_MAP,
  type UnboxingListItem,
} from '../../api/unboxing'

const router = useRouter()
const tableData = ref<UnboxingListItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const trackingNo = ref('')
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    const data = await fetchUnboxingRecords({
      trackingNo: trackingNo.value || undefined,
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

onMounted(loadData)

function handleSearch() {
  page.value = 1
  loadData()
}

function openDetail(row: UnboxingListItem) {
  router.push(`/unboxing/${row.id}`)
}

function onPageChange(p: number) {
  page.value = p
  loadData()
}

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
</script>

<template>
  <div class="unboxing-list">
    <el-card v-loading="loading">
      <template #header>
        <span>开箱记录</span>
        <el-button type="primary" :icon="Plus" @click="router.push('/unboxing/create')">
          录制开箱
        </el-button>
      </template>

      <div class="toolbar">
        <el-input
          v-model="trackingNo"
          placeholder="按快递单号搜索"
          :prefix-icon="Search"
          clearable
          style="width: 280px"
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>
      </div>

      <el-table :data="tableData" stripe border>
        <el-table-column prop="trackingNo" label="快递单号" min-width="160">
          <template #default="{ row }">
            <el-link type="primary" @click="openDetail(row)">{{ row.trackingNo }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="视频时长" width="110" align="center">
          <template #default="{ row }">
            <span v-if="row.videoDurationSec">
              <el-icon style="vertical-align: -2px"><VideoCamera /></el-icon>
              {{ formatDuration(row.videoDurationSec) }}
            </span>
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="photoCount" label="问题照片" width="100" align="center" />
        <el-table-column prop="operatorName" label="操作人" width="120" />
        <el-table-column prop="createdAt" label="创建时间" width="170" />
        <el-table-column label="操作" width="100" fixed="right">
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
.unboxing-list :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.muted {
  color: #909399;
}
</style>
