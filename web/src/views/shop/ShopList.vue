<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument, Plus, Shop } from '@element-plus/icons-vue'
import {
  PLATFORM_OPTIONS,
  PLUGIN_STATUS_MAP,
  createShop,
  deleteShop,
  fetchShops,
  resetShopBind,
  updateShop,
  type MarketplaceShop,
  type ShopPlatform,
} from '../../api/shop'

const router = useRouter()
const loading = ref(false)
const tableData = ref<MarketplaceShop[]>([])
const dialogVisible = ref(false)
const editing = ref<MarketplaceShop | null>(null)
const form = ref({ name: '', platform: 'doudian' as ShopPlatform, remark: '' })

async function loadData() {
  loading.value = true
  try {
    tableData.value = await fetchShops()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

function statusLabel(s: string) {
  return PLUGIN_STATUS_MAP[s as keyof typeof PLUGIN_STATUS_MAP]?.label || s
}

function statusType(s: string) {
  return PLUGIN_STATUS_MAP[s as keyof typeof PLUGIN_STATUS_MAP]?.type || 'info'
}

function openCreate() {
  editing.value = null
  form.value = { name: '', platform: 'doudian', remark: '' }
  dialogVisible.value = true
}

function openEdit(row: MarketplaceShop) {
  editing.value = row
  form.value = { name: row.name, platform: row.platform, remark: row.remark || '' }
  dialogVisible.value = true
}

async function handleSave() {
  try {
    if (editing.value) {
      await updateShop(editing.value.id, { name: form.value.name, remark: form.value.remark })
      ElMessage.success('已更新')
    } else {
      const shop = await createShop(form.value)
      ElMessage.success(shop.pluginAvailable ? '已添加，请用绑定码连接插件' : '已添加。该平台插件尚未提供，可先保存店铺')
    }
    dialogVisible.value = false
    loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function copyBind(row: MarketplaceShop) {
  try {
    await navigator.clipboard.writeText(row.bindCode)
    ElMessage.success('绑定码已复制')
  } catch {
    ElMessage.info(row.bindCode)
  }
}

async function handleReset(row: MarketplaceShop) {
  try {
    await ElMessageBox.confirm('重置后原插件密钥立即失效，需要重新填写绑定码。', '重置绑定码')
    const shop = await resetShopBind(row.id)
    ElMessage.success(`新绑定码：${shop.bindCode}`)
    loadData()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error((e as Error).message || '重置失败')
  }
}

async function handleDelete(row: MarketplaceShop) {
  try {
    await ElMessageBox.confirm(`确定删除店铺「${row.name}」及其售后数据？`, '删除')
    await deleteShop(row.id)
    ElMessage.success('已删除')
    loadData()
  } catch (e) {
    if (e !== 'cancel') ElMessage.error((e as Error).message || '删除失败')
  }
}

function openWorkbench(row: MarketplaceShop) {
  router.push(`/shops/${row.id}`)
}
</script>

<template>
  <div class="shop-list">
    <el-card v-loading="loading">
      <template #header>
        <div class="header">
          <span><el-icon><Shop /></el-icon> 店铺管理</span>
          <el-button type="primary" :icon="Plus" @click="openCreate">添加店铺</el-button>
        </div>
      </template>

      <p class="hint">
        新增店铺后生成绑定码。抖店插件安装后填入售后管理地址和绑定码，即可把工作台「快捷筛选」卡片与售后单同步到这里。
      </p>

      <el-table :data="tableData" stripe border>
        <el-table-column prop="name" label="店铺" min-width="140">
          <template #default="{ row }">
            <div class="shop-name">{{ row.name }}</div>
            <div v-if="row.platformShopName" class="sub">平台：{{ row.platformShopName }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="platformLabel" label="平台" width="100" />
        <el-table-column label="绑定码" width="180">
          <template #default="{ row }">
            <code class="bind-code">{{ row.bindCode }}</code>
            <el-button type="primary" link :icon="CopyDocument" @click="copyBind(row)">复制</el-button>
          </template>
        </el-table-column>
        <el-table-column label="插件" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="!row.pluginAvailable" type="warning" size="small">未提供</el-tag>
            <el-tag v-else :type="statusType(row.pluginStatus)" size="small">{{ statusLabel(row.pluginStatus) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastSyncAt" label="最近同步" width="170">
          <template #default="{ row }">{{ row.lastSyncAt || '—' }}</template>
        </el-table-column>
        <el-table-column prop="lastSeenAt" label="最近在线" width="170">
          <template #default="{ row }">{{ row.lastSeenAt || '—' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openWorkbench(row)">工作台</el-button>
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
            <el-button type="primary" link @click="handleReset(row)">重置绑定</el-button>
            <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑店铺' : '添加店铺'" width="480px">
      <el-form label-width="100px">
        <el-form-item label="店铺名称" required>
          <el-input v-model="form.name" placeholder="如 甄选美妆抖店" />
        </el-form-item>
        <el-form-item label="平台类型" required>
          <el-select v-model="form.platform" :disabled="!!editing" style="width: 100%">
            <el-option v-for="p in PLATFORM_OPTIONS" :key="p.value" :label="p.label" :value="p.value" />
          </el-select>
        </el-form-item>
        <el-alert
          v-if="!editing && form.platform !== 'doudian'"
          type="warning"
          :closable="false"
          show-icon
          title="该平台插件尚未提供，可先创建店铺，绑定与同步暂不可用。"
          style="margin-bottom: 12px"
        />
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
.hint { color: #909399; margin: 0 0 16px; font-size: 13px; line-height: 1.6; }
.bind-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  letter-spacing: 0.08em;
  margin-right: 4px;
}
.shop-name { font-weight: 600; }
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
</style>
