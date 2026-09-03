<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import {
  fetchServiceOrders,
  fetchShops,
  type MarketplaceShop,
  type ServiceOrder,
  type ServiceTabCount,
} from '../../api/shop'

const loading = ref(false)
const shops = ref<MarketplaceShop[]>([])
const tableData = ref<ServiceOrder[]>([])
const tabs = ref<ServiceTabCount[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const shopId = ref<number | undefined>()
const statusTab = ref('待处理')
const keyword = ref('')
const nowTick = ref(Date.now())
let tickTimer = 0

const tabCounts = computed(() => {
  const map = Object.fromEntries(tabs.value.map((t) => [t.statusTab, t.count]))
  return {
    待处理: map['待处理'] || 0,
    处理中: map['处理中'] || 0,
    已逾期: map['已逾期'] || 0,
  }
})

function remainSecondsOf(row: ServiceOrder) {
  void nowTick.value
  if (row.deadlineAt) {
    const t = Date.parse(row.deadlineAt)
    if (!Number.isNaN(t)) return Math.floor((t - Date.now()) / 1000)
  }
  return Number(row.remainSeconds || 0)
}

function remainLabel(row: ServiceOrder) {
  const action = row.timeoutAction || '逾期'
  if (row.statusTab === '已逾期' || (row.deadlineAt && remainSecondsOf(row) <= 0)) {
    return action ? `已超时 · ${action}` : '已超时'
  }
  const sec = remainSecondsOf(row)
  if (row.deadlineAt || sec > 0) {
    const h = Math.floor(Math.max(0, sec) / 3600)
    const m = Math.floor((Math.max(0, sec) % 3600) / 60)
    if (h > 0) return `${h}小时${m}分后${action}`
    return `${m}分后${action}`
  }
  return row.timeoutText || '—'
}

function timeoutTone(row: ServiceOrder) {
  if (row.statusTab === '已逾期') return 'danger'
  const sec = remainSecondsOf(row)
  if (!row.deadlineAt && !row.timeoutText && !sec) return ''
  if (sec <= 0 || sec <= 4 * 3600) return 'danger'
  if (sec <= 12 * 3600) return 'warning'
  return 'ok'
}

function statusTone(tab: string) {
  if (tab === '已逾期') return 'danger'
  if (tab === '待处理') return 'warning'
  if (tab === '处理中') return 'primary'
  return 'info'
}

async function loadShops() {
  try {
    shops.value = await fetchShops()
  } catch {
    shops.value = []
  }
}

async function loadData() {
  loading.value = true
  try {
    const data = await fetchServiceOrders({
      shopId: shopId.value || undefined,
      statusTab: statusTab.value || undefined,
      keyword: keyword.value || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    tableData.value = data.list
    total.value = data.total
    tabs.value = data.tabs || []
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadData()
}

onMounted(() => {
  loadShops()
  loadData()
  tickTimer = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)
})
onUnmounted(() => {
  if (tickTimer) window.clearInterval(tickTimer)
})
</script>

<template>
  <div v-loading="loading" class="service-list">
    <div class="page-head">
      <div>
        <h2 class="page-title">服务工单</h2>
        <p class="desc">默认看待处理，并按建议 / 最新记录时间最近的在前。采集近 30 天待处理、处理中、已逾期。时效按剩余时间变色，可在通知管理勾选「待处理服务工单」按等级提醒。</p>
      </div>
    </div>

    <el-card>
      <div class="toolbar">
        <el-select
          v-model="shopId"
          clearable
          placeholder="全部店铺"
          style="width: 180px"
          @change="handleSearch"
        >
          <el-option
            v-for="shop in shops"
            :key="shop.id"
            :label="shop.name"
            :value="shop.id"
          />
        </el-select>
        <el-select
          v-model="statusTab"
          clearable
          placeholder="全部状态"
          style="width: 150px"
          @change="handleSearch"
        >
          <el-option :label="`待处理 ${tabCounts['待处理']}`" value="待处理" />
          <el-option :label="`处理中 ${tabCounts['处理中']}`" value="处理中" />
          <el-option :label="`已逾期 ${tabCounts['已逾期']}`" value="已逾期" />
        </el-select>
        <el-input
          v-model="keyword"
          clearable
          placeholder="工单ID / 订单号 / 商品 / 买家"
          style="width: 280px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <span class="total">共 {{ total }} 条</span>
      </div>

      <el-table :data="tableData" stripe border>
        <el-table-column prop="shopName" label="店铺" width="140" />
        <el-table-column label="商品 / 订单" min-width="240">
          <template #default="{ row }">
            <div class="product">
              <img v-if="row.productImage" class="thumb" :src="row.productImage" alt="" />
              <div class="product-meta">
                <div class="title">{{ row.productTitle || '—' }}</div>
                <div v-if="row.productContent" class="sub">{{ row.productContent }}</div>
                <div class="sub">订单 {{ row.orderNo || '—' }}</div>
                <div class="sub">工单 {{ row.platformServiceId }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="买家 / 来源" min-width="160">
          <template #default="{ row }">
            <div>{{ row.buyerNick || '—' }}</div>
            <div class="sub">{{ row.createSource || '—' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="140">
          <template #default="{ row }">
            <div>{{ row.businessType || '—' }}</div>
            <div class="sub">{{ row.orderType || '' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="140">
          <template #default="{ row }">
            <el-tag :type="statusTone(row.statusTab)" size="small">{{ row.statusTab }}</el-tag>
            <div class="status-text" :class="statusTone(row.statusTab)">{{ row.status || '—' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="时效" min-width="160">
          <template #default="{ row }">
            <span class="timeout" :class="timeoutTone(row)">{{ remainLabel(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="建议 / 最新记录" min-width="200">
          <template #default="{ row }">
            <div v-if="row.solution">{{ row.solution }}</div>
            <div v-if="row.lastLog" class="sub">{{ row.lastLog }}</div>
            <div v-if="row.lastLogTime" class="sub">{{ row.lastLogTime }}</div>
            <div v-if="row.detail && !row.lastLog" class="sub">{{ row.detail }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="syncedAt" label="同步时间" width="170" />
      </el-table>

      <div class="pager">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadData"
          @size-change="() => { page = 1; loadData() }"
        />
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.page-head { margin-bottom: 16px; }
.page-title { margin: 0 0 6px; font-size: 22px; }
.desc { color: #606266; margin: 0; }
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.total { margin-left: auto; color: #909399; font-size: 13px; }
.product { display: flex; gap: 10px; align-items: flex-start; }
.thumb { width: 48px; height: 48px; border-radius: 4px; object-fit: cover; flex-shrink: 0; background: #f5f7fa; }
.product-meta { min-width: 0; }
.title { font-weight: 600; line-height: 1.4; }
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
.status-text { font-size: 12px; margin-top: 6px; font-weight: 600; }
.status-text.danger { color: #f56c6c; }
.status-text.warning { color: #e6a23c; }
.status-text.primary { color: #409eff; }
.status-text.info { color: #909399; }
.timeout { font-weight: 600; }
.timeout.danger { color: #f56c6c; }
.timeout.warning { color: #e6a23c; }
.timeout.ok { color: #67c23a; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
