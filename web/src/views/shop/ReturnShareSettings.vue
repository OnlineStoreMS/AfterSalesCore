<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument, Plus } from '@element-plus/icons-vue'
import {
  createReturnShare,
  deleteReturnShare,
  fetchReturnShares,
  returnSharePageUrl,
  updateReturnShare,
  type ReturnShare,
} from '../../api/shop'

const loading = ref(false)
const list = ref<ReturnShare[]>([])
const dialogVisible = ref(false)
const saving = ref(false)
const editing = ref<ReturnShare | null>(null)
const form = ref({ name: '', returnLocation: '', enabled: true })

async function load() {
  loading.value = true
  try {
    list.value = await fetchReturnShares()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

function shareUrl(row: ReturnShare) {
  return returnSharePageUrl(row.token)
}

async function copyUrl(row: ReturnShare) {
  const url = shareUrl(row)
  try {
    await navigator.clipboard.writeText(url)
    ElMessage.success('分享地址已复制')
  } catch {
    ElMessage.info(url)
  }
}

function openCreate() {
  editing.value = null
  form.value = { name: '', returnLocation: '', enabled: true }
  dialogVisible.value = true
}

function openEdit(row: ReturnShare) {
  editing.value = row
  form.value = { name: row.name, returnLocation: row.returnLocation, enabled: row.enabled }
  dialogVisible.value = true
}

async function save() {
  if (!form.value.name.trim()) {
    ElMessage.warning('请填写分享者')
    return
  }
  if (!form.value.returnLocation.trim()) {
    ElMessage.warning('请填写退回地过滤')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateReturnShare(editing.value.id, {
        name: form.value.name.trim(),
        returnLocation: form.value.returnLocation.trim(),
        enabled: form.value.enabled,
      })
      ElMessage.success('已保存')
    } else {
      const created = await createReturnShare({
        name: form.value.name.trim(),
        returnLocation: form.value.returnLocation.trim(),
        enabled: form.value.enabled,
      })
      await copyUrl(created)
      ElMessage.success('已新增分享者，分享地址已复制')
    }
    dialogVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleEnabled(row: ReturnShare) {
  try {
    await updateReturnShare(row.id, { enabled: !row.enabled })
    ElMessage.success(row.enabled ? '已停用' : '已启用')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '操作失败')
  }
}

async function remove(row: ReturnShare) {
  await ElMessageBox.confirm(`删除分享者「${row.name}」后，原分享地址将失效。`, '确认删除')
  try {
    await deleteReturnShare(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e) {
    if ((e as Error).message === 'cancel') return
    ElMessage.error((e as Error).message || '删除失败')
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="share-settings">
    <div class="page-head">
      <div>
        <h2 class="page-title">分享设置</h2>
        <p class="desc">为供货商配置分享者。每个分享者对应一条固定地址，按退回地过滤实时刷新最新退回件，不展示店铺名。</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="openCreate">新增分享者</el-button>
    </div>

    <el-card>
      <el-table :data="list" stripe border>
        <el-table-column prop="name" label="分享者" min-width="140" />
        <el-table-column prop="returnLocation" label="退回地过滤" min-width="220">
          <template #default="{ row }">
            <div class="location">{{ row.returnLocation }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="分享地址" min-width="280">
          <template #default="{ row }">
            <div class="url">{{ shareUrl(row) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link :icon="CopyDocument" @click="copyUrl(row)">复制地址</el-button>
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
            <el-button type="primary" link @click="toggleEnabled(row)">{{ row.enabled ? '停用' : '启用' }}</el-button>
            <el-button type="danger" link @click="remove(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑分享者' : '新增分享者'" width="520px">
      <el-form label-width="100px">
        <el-form-item label="分享者" required>
          <el-input v-model="form.name" placeholder="如供货商名称" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="退回地过滤" required>
          <el-input
            v-model="form.returnLocation"
            type="textarea"
            :rows="3"
            placeholder="匹配退回地包含的关键字，多个用逗号或换行分隔"
          />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; margin-bottom: 16px; }
.page-title { margin: 0 0 6px; font-size: 22px; }
.desc { color: #606266; margin: 0; max-width: 720px; line-height: 1.5; }
.location, .url { white-space: pre-wrap; word-break: break-all; line-height: 1.5; }
.url { color: #409eff; font-size: 12px; }
</style>
