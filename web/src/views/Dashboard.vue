<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Box, Monitor, VideoCamera, Search } from '@element-plus/icons-vue'
import { fetchEdgeRecordStats } from '../api/edgeRecord'
import { fetchEdgeDevices, EDGE_STATUS_MAP } from '../api/edgeDevice'

const router = useRouter()
const stats = ref({ unboxingCount: 0, packingCount: 0 })
const edgeDevices = ref<Awaited<ReturnType<typeof fetchEdgeDevices>>>([])
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    const [s, devices] = await Promise.all([fetchEdgeRecordStats(), fetchEdgeDevices()])
    stats.value = s
    edgeDevices.value = devices.filter((d) => d.edgeId !== 'cloud')
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

function edgeStatusLabel(status: string) {
  return EDGE_STATUS_MAP[status]?.label || status
}

function edgeStatusType(status: string) {
  return EDGE_STATUS_MAP[status]?.type || 'info'
}
</script>

<template>
  <div v-loading="loading" class="dashboard">
    <h2 class="page-title">售后工作台</h2>
    <p class="desc">管理云端浏览器录制与 BoxEdge 本地录制端同步的记录，支持开箱与打包全流程。</p>

    <div class="card-grid">
      <el-card shadow="hover" class="action-card" @click="router.push('/unboxing/create')">
        <el-icon :size="32" color="#409eff"><VideoCamera /></el-icon>
        <h3>录制开箱</h3>
        <p>使用浏览器摄像头录制退货开箱</p>
      </el-card>
      <el-card shadow="hover" class="action-card" @click="router.push('/packing/create')">
        <el-icon :size="32" color="#67c23a"><Box /></el-icon>
        <h3>录制打包</h3>
        <p>使用浏览器摄像头录制发货打包</p>
      </el-card>
      <el-card shadow="hover" class="action-card" @click="router.push('/unboxing')">
        <el-icon :size="32" color="#e6a23c"><Search /></el-icon>
        <h3>开箱记录</h3>
        <p>已归档 {{ stats.unboxingCount }} 条</p>
      </el-card>
      <el-card shadow="hover" class="action-card" @click="router.push('/packing')">
        <el-icon :size="32" color="#909399"><Search /></el-icon>
        <h3>打包记录</h3>
        <p>已归档 {{ stats.packingCount }} 条</p>
      </el-card>
    </div>

    <el-card class="edge-card">
      <template #header>
        <div class="edge-header">
          <span><el-icon><Monitor /></el-icon> 本地录制端状态</span>
          <el-button type="primary" link @click="router.push('/edge-devices')">管理录制端</el-button>
        </div>
      </template>
      <el-table v-if="edgeDevices.length" :data="edgeDevices" stripe>
        <el-table-column prop="edgeId" label="Edge ID" min-width="120" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="baseUrl" label="地址" min-width="200" show-overflow-tooltip />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="edgeStatusType(row.status)" size="small">{{ edgeStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastSeenAt" label="最近在线" width="170" />
      </el-table>
      <el-empty v-else description="暂无本地录制端，可在「录制端管理」中添加" />
    </el-card>
  </div>
</template>

<style scoped>
.dashboard { max-width: 960px; }
.page-title { margin: 0 0 8px; font-size: 22px; }
.desc { color: #606266; margin: 0 0 24px; line-height: 1.6; }
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}
.action-card {
  cursor: pointer;
  text-align: center;
  transition: transform 0.15s;
}
.action-card:hover { transform: translateY(-2px); }
.action-card h3 { margin: 12px 0 6px; font-size: 16px; }
.action-card p { margin: 0; color: #909399; font-size: 13px; }
.edge-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.edge-card :deep(.el-card__header) { padding: 12px 20px; }
</style>
