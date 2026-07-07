<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Monitor } from '@element-plus/icons-vue'
import {
  createEdgeDevice,
  deleteEdgeDevice,
  fetchEdgeDevices,
  probeEdgeDevice,
  syncEdgeDevices,
  updateEdgeDevice,
  EDGE_STATUS_MAP,
  type EdgeDevice,
} from '../../api/edgeDevice'

const loading = ref(false)
const tableData = ref<EdgeDevice[]>([])
const dialogVisible = ref(false)
const editing = ref<EdgeDevice | null>(null)
const form = ref({ edgeId: '', name: '', baseUrl: '', remark: '' })

async function loadData() {
  loading.value = true
  try {
    tableData.value = await fetchEdgeDevices()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

function statusLabel(s: string) {
  return EDGE_STATUS_MAP[s]?.label || s
}

function statusType(s: string) {
  return EDGE_STATUS_MAP[s]?.type || 'info'
}

function openCreate() {
  editing.value = null
  form.value = { edgeId: '', name: '', baseUrl: '', remark: '' }
  dialogVisible.value = true
}

function openEdit(row: EdgeDevice) {
  if (row.edgeId === 'cloud') {
    ElMessage.info('云端浏览器录制端不可编辑')
    return
  }
  editing.value = row
  form.value = { edgeId: row.edgeId, name: row.name, baseUrl: row.baseUrl, remark: row.remark || '' }
  dialogVisible.value = true
}

async function handleSave() {
  try {
    if (editing.value) {
      await updateEdgeDevice(editing.value.id, {
        name: form.value.name,
        baseUrl: form.value.baseUrl,
        remark: form.value.remark,
      })
      ElMessage.success('已更新')
    } else {
      await createEdgeDevice(form.value)
      ElMessage.success('已添加')
    }
    dialogVisible.value = false
    loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function handleProbe(row: EdgeDevice) {
  if (!row.baseUrl) {
    ElMessage.warning('请先配置 base_url')
    return
  }
  try {
    await probeEdgeDevice(row.id)
    ElMessage.success('探测完成')
    loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '探测失败')
  }
}

async function handleSync() {
  try {
    tableData.value = await syncEdgeDevices()
    ElMessage.success('已从记录同步 edge_id')
  } catch (e) {
    ElMessage.error((e as Error).message || '同步失败')
  }
}

async function handleDelete(row: EdgeDevice) {
  if (row.edgeId === 'cloud') return
  try {
    await ElMessageBox.confirm(`确定删除录制端 ${row.name}？`, '删除')
    await deleteEdgeDevice(row.id)
    ElMessage.success('已删除')
    loadData()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error((e as Error).message || '删除失败')
  }
}
</script>

<template>
  <div class="edge-device-list">
    <el-card v-loading="loading">
      <template #header>
        <div class="header">
          <span><el-icon><Monitor /></el-icon> 本地录制端管理</span>
          <div class="actions">
            <el-button :icon="Refresh" @click="handleSync">从记录同步</el-button>
            <el-button type="primary" :icon="Plus" @click="openCreate">添加录制端</el-button>
          </div>
        </div>
      </template>

      <p class="hint">配置 BoxEdge 的访问地址后，系统将定期探测在线状态（GET /api/health）。</p>

      <el-table :data="tableData" stripe border>
        <el-table-column prop="edgeId" label="Edge ID" min-width="120" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="baseUrl" label="访问地址" min-width="220" show-overflow-tooltip />
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastSeenAt" label="最近在线" width="170" />
        <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="handleProbe(row)">探测</el-button>
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
            <el-button v-if="row.edgeId !== 'cloud'" type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑录制端' : '添加录制端'" width="480px">
      <el-form label-width="100px">
        <el-form-item label="Edge ID" required>
          <el-input v-model="form.edgeId" :disabled="!!editing" placeholder="如 edge-001" />
        </el-form-item>
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="售后本地录制端" />
        </el-form-item>
        <el-form-item label="访问地址">
          <el-input v-model="form.baseUrl" placeholder="http://192.168.x.x:8080" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.actions { display: flex; gap: 8px; }
.hint { color: #909399; margin: 0 0 16px; font-size: 13px; }
</style>
