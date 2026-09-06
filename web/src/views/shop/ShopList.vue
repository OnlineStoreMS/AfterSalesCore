<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Shop } from '@element-plus/icons-vue'
import {
  PLATFORM_OPTIONS,
  PLUGIN_STATUS_MAP,
  PLUGIN_SYNC_OPTIONS,
  createShopFromAgent,
  deleteShop,
  enableAgentCollect,
  fetchAgentOnlineShops,
  fetchPluginSetting,
  fetchShops,
  resetShopBind,
  requestShopSync,
  savePluginSetting,
  updateShop,
  type MarketplaceShop,
  type ShopPlatform,
} from '../../api/shop'

const router = useRouter()
const loading = ref(false)
const tableData = ref<MarketplaceShop[]>([])
const dialogVisible = ref(false)
const editing = ref<MarketplaceShop | null>(null)
const syncMinutes = ref(30)
const savingSync = ref(false)
const onlineShops = ref<Array<{
  platform: string
  platformShopId: string
  platformShopName: string
  browserChannel: string
  agentName: string
}>>([])
const loadingOnline = ref(false)

const form = ref({
  platform: 'doudian' as ShopPlatform,
  platformShopId: '',
  platformShopName: '',
  jobType: 'doudian.aftersale',
  name: '',
  remark: '',
  intervalMinutes: 30,
})

const jobOptions = [
  { value: 'doudian.aftersale', label: '抖店售后单抓取' },
]

async function loadData() {
  loading.value = true
  try {
    const [shops, setting] = await Promise.all([fetchShops(), fetchPluginSetting()])
    tableData.value = shops
    syncMinutes.value = setting.pluginSyncIntervalMin || 30
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadOnlineShops() {
  loadingOnline.value = true
  try {
    onlineShops.value = await fetchAgentOnlineShops(form.value.platform)
  } catch (e) {
    onlineShops.value = []
    ElMessage.error((e as Error).message || '加载 Agents 上线店铺失败')
  } finally {
    loadingOnline.value = false
  }
}

watch(() => form.value.platform, () => {
  form.value.platformShopId = ''
  form.value.platformShopName = ''
  if (dialogVisible.value && !editing.value) loadOnlineShops()
})

async function saveSyncInterval() {
  savingSync.value = true
  try {
    const setting = await savePluginSetting({ pluginSyncIntervalMin: syncMinutes.value })
    syncMinutes.value = setting.pluginSyncIntervalMin
    ElMessage.success('已保存，并已更新各店铺采集任务的执行间隔')
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    savingSync.value = false
  }
}

onMounted(loadData)

function statusLabel(s: string) {
  return PLUGIN_STATUS_MAP[s as keyof typeof PLUGIN_STATUS_MAP]?.label || s
}

function statusType(s: string) {
  return PLUGIN_STATUS_MAP[s as keyof typeof PLUGIN_STATUS_MAP]?.type || 'info'
}

async function openCreate() {
  editing.value = null
  form.value = {
    platform: 'doudian',
    platformShopId: '',
    platformShopName: '',
    jobType: 'doudian.aftersale',
    name: '',
    remark: '',
    intervalMinutes: syncMinutes.value || 30,
  }
  dialogVisible.value = true
  await loadOnlineShops()
}

function openEdit(row: MarketplaceShop) {
  editing.value = row
  form.value = {
    platform: row.platform,
    platformShopId: row.platformShopId || '',
    platformShopName: row.platformShopName || '',
    jobType: 'doudian.aftersale',
    name: row.name,
    remark: row.remark || '',
    intervalMinutes: syncMinutes.value || 30,
  }
  dialogVisible.value = true
}

function onPickShop(id: string) {
  form.value.platformShopId = id
  const s = onlineShops.value.find((x) => x.platformShopId === id)
  if (s) {
    form.value.platformShopName = s.platformShopName
    if (!form.value.name) form.value.name = s.platformShopName || s.platformShopId
  }
}

async function handleSave() {
  try {
    if (editing.value) {
      await updateShop(editing.value.id, {
        name: form.value.name,
        platformShopId: form.value.platformShopId,
        platformShopName: form.value.platformShopName,
        remark: form.value.remark,
      })
      ElMessage.success('已更新')
    } else {
      if (!form.value.platformShopId) {
        ElMessage.error('请选择 Agents 中心已上线店铺')
        return
      }
      await createShopFromAgent({
        platform: form.value.platform,
        platformShopId: form.value.platformShopId,
        platformShopName: form.value.platformShopName,
        jobType: form.value.jobType,
        name: form.value.name,
        intervalMinutes: form.value.intervalMinutes,
      })
      ElMessage.success('已创建采集任务并触发首次执行')
    }
    dialogVisible.value = false
    loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function handleEnable(row: MarketplaceShop) {
  try {
    await enableAgentCollect(row.id)
    ElMessage.success('已启用 Agent 采集')
    loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '启用失败')
  }
}

async function handleReset(row: MarketplaceShop) {
  try {
    await ElMessageBox.confirm('重置后原采集凭证立即失效，需重新启用。', '重置采集')
    await resetShopBind(row.id)
    ElMessage.success('已重置')
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

async function handleRequestSync(row: MarketplaceShop) {
  try {
    await requestShopSync(row.id)
    ElMessage.success('已请求立即执行采集')
    loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '请求失败')
  }
}
</script>

<template>
  <div class="shop-list">
    <el-card v-loading="loading">
      <template #header>
        <div class="header">
          <span><el-icon><Shop /></el-icon> 店铺采集</span>
          <el-button type="primary" :icon="Plus" @click="openCreate">添加采集</el-button>
        </div>
      </template>

      <p class="hint">
        选择 Agents 已上线店铺，创建一次「售后单采集」任务即可（含上报地址等参数）。之后按间隔反复执行同一任务，不会每次新建任务；「请求同步」是立即再执行一次。
      </p>
      <div class="sync-setting">
        <span class="sync-label">自动采集间隔</span>
        <el-select v-model="syncMinutes" style="width: 160px">
          <el-option
            v-for="opt in PLUGIN_SYNC_OPTIONS"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>
        <el-button type="primary" plain :loading="savingSync" @click="saveSyncInterval">保存间隔</el-button>
        <span class="sync-tip">到期后触发已有采集任务再执行，不会重复创建任务。</span>
      </div>

      <el-table :data="tableData" stripe border>
        <el-table-column prop="name" label="店铺" min-width="180">
          <template #default="{ row }">
            <div class="shop-title">
              <span class="shop-name">{{ row.name }}</span>
              <span
                v-if="row.pendingTicketCount"
                class="count-dot"
                :title="`待处理售后 ${row.pendingTicketCount} 单`"
              >{{ row.pendingTicketCount > 99 ? '99+' : row.pendingTicketCount }}</span>
            </div>
            <div v-if="row.platformShopId" class="sub">店铺 ID：{{ row.platformShopId }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="platformLabel" label="平台" width="100" />
        <el-table-column label="采集" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="!row.pluginAvailable" type="warning" size="small">未提供</el-tag>
            <el-tag v-else :type="statusType(row.pluginStatus)" size="small">{{ statusLabel(row.pluginStatus) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastSyncAt" label="最近同步" width="170">
          <template #default="{ row }">{{ row.lastSyncAt || '—' }}</template>
        </el-table-column>
        <el-table-column prop="nextSyncAt" label="下次同步" width="190">
          <template #default="{ row }">{{ row.nextSyncAt || '—' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="360" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="openWorkbench(row)">工作台</el-button>
            <el-button
              v-if="row.pluginAvailable && row.pluginStatus === 'unbound'"
              type="primary"
              link
              @click="handleEnable(row)"
            >启用采集</el-button>
            <el-button
              v-if="row.pluginStatus !== 'unbound'"
              type="primary"
              link
              @click="handleRequestSync(row)"
            >
              {{ row.syncRequested ? '已请求执行' : '立即执行' }}
            </el-button>
            <el-button type="primary" link @click="openEdit(row)">编辑</el-button>
            <el-button type="primary" link @click="handleReset(row)">重置采集</el-button>
            <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑店铺' : '添加采集任务'" width="560px">
      <el-form label-width="120px">
        <template v-if="!editing">
          <el-form-item label="平台类型" required>
            <el-select v-model="form.platform" style="width: 100%">
              <el-option v-for="p in PLATFORM_OPTIONS" :key="p.value" :label="p.label" :value="p.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="上线店铺" required>
            <el-select
              :model-value="form.platformShopId"
              filterable
              :loading="loadingOnline"
              placeholder="选择 Agents 中心已上线店铺"
              style="width: 100%"
              @change="onPickShop"
            >
              <el-option
                v-for="s in onlineShops"
                :key="s.platformShopId"
                :label="`${s.platformShopName || s.platformShopId}（${s.platformShopId} · ${s.agentName || '节点'} · ${s.browserChannel || '-'}）`"
                :value="s.platformShopId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="任务类型" required>
            <el-select v-model="form.jobType" style="width: 100%">
              <el-option v-for="j in jobOptions" :key="j.value" :label="j.label" :value="j.value" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.jobType === 'doudian.aftersale'" label="采集间隔" required>
            <el-select v-model="form.intervalMinutes" style="width: 100%">
              <el-option
                v-for="opt in PLUGIN_SYNC_OPTIONS"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="显示名称">
            <el-input v-model="form.name" placeholder="默认用店铺名称" />
          </el-form-item>
        </template>
        <template v-else>
          <el-form-item label="店铺名称" required>
            <el-input v-model="form.name" />
          </el-form-item>
          <el-form-item label="平台店铺 ID">
            <el-input v-model="form.platformShopId" />
          </el-form-item>
          <el-form-item label="平台店铺名">
            <el-input v-model="form.platformShopName" />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="form.remark" type="textarea" :rows="2" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">{{ editing ? '保存' : '创建采集任务' }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.header { display: flex; justify-content: space-between; align-items: center; }
.hint { color: #606266; margin: 0 0 12px; line-height: 1.6; }
.sync-setting { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-bottom: 16px; }
.sync-label { font-weight: 500; }
.sync-tip { color: #909399; font-size: 13px; }
.shop-title { display: flex; align-items: center; gap: 6px; }
.shop-name { font-weight: 500; }
.count-dot {
  min-width: 18px; height: 18px; padding: 0 5px; border-radius: 9px;
  background: #f56c6c; color: #fff; font-size: 12px; line-height: 18px; text-align: center;
}
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
</style>
