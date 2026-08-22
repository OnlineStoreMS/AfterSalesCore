<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import {
  PLUGIN_STATUS_MAP,
  fetchShopWorkbench,
  fetchShopServiceOrders,
  requestShopSync,
  type FilterCard,
  type MarketplaceShop,
  type AftersaleTicket,
  type ServiceOrder,
} from '../../api/shop'

const route = useRoute()
const router = useRouter()
const shopId = computed(() => Number(route.params.id))

const loading = ref(false)
const shop = ref<MarketplaceShop | null>(null)
const cards = ref<FilterCard[]>([])
const tickets = ref<AftersaleTicket[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const activeCardKey = ref('')
const lastSyncAt = ref('')
const requestingSync = ref(false)
const nowTick = ref(Date.now())
let tickTimer = 0

const serviceOrders = ref<ServiceOrder[]>([])
const serviceTotal = ref(0)
const servicePage = ref(1)
const servicePageSize = ref(20)
const serviceKeyword = ref('')
const activeServiceTab = ref('待处理')

const groupedCards = computed(() => {
  const groups: { name: string; items: FilterCard[] }[] = []
  const index = new Map<string, number>()
  for (const card of cards.value) {
    const name = card.groupName || '其他'
    let i = index.get(name)
    if (i === undefined) {
      i = groups.length
      index.set(name, i)
      groups.push({ name, items: [] })
    }
    groups[i].items.push(card)
  }
  return groups
})

async function loadData() {
  if (!shopId.value) return
  loading.value = true
  try {
    const data = await fetchShopWorkbench(shopId.value, {
      cardKey: activeCardKey.value || undefined,
      keyword: keyword.value || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    shop.value = data.shop
    cards.value = data.cards
    tickets.value = data.tickets
    total.value = data.total
    lastSyncAt.value = data.lastSyncAt || ''
    await loadService()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadService() {
  if (!shopId.value) return
  const data = await fetchShopServiceOrders(shopId.value, {
    statusTab: activeServiceTab.value || undefined,
    keyword: serviceKeyword.value || undefined,
    page: servicePage.value,
    pageSize: servicePageSize.value,
  })
  serviceOrders.value = data.list || []
  serviceTotal.value = data.total || 0
}

onMounted(() => {
  loadData()
  tickTimer = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)
})
onUnmounted(() => {
  if (tickTimer) window.clearInterval(tickTimer)
})
watch(shopId, () => {
  activeCardKey.value = ''
  keyword.value = ''
  page.value = 1
  activeServiceTab.value = '待处理'
  serviceKeyword.value = ''
  servicePage.value = 1
  loadData()
})

function selectCard(card?: FilterCard) {
  const key = card?.cardKey || ''
  activeCardKey.value = activeCardKey.value === key ? '' : key
  page.value = 1
  loadData()
}

function handleSearch() {
  page.value = 1
  loadData()
}

function handleServiceSearch() {
  servicePage.value = 1
  loadService()
}

function statusType(s: string) {
  return PLUGIN_STATUS_MAP[s as keyof typeof PLUGIN_STATUS_MAP]?.type || 'info'
}

function statusLabel(s: string) {
  return PLUGIN_STATUS_MAP[s as keyof typeof PLUGIN_STATUS_MAP]?.label || s
}

function urgentGroup(name: string) {
  return name === '紧急'
}

function remainSecondsOf(row: AftersaleTicket | ServiceOrder) {
  void nowTick.value
  if (row.deadlineAt) {
    const t = Date.parse(row.deadlineAt)
    if (!Number.isNaN(t)) return Math.max(0, Math.floor((t - Date.now()) / 1000))
  }
  return Math.max(0, Number(row.remainSeconds || 0))
}

function formatRemain(sec: number) {
  if (sec <= 0) return '已超时'
  const d = Math.floor(sec / 86400)
  const h = Math.floor((sec % 86400) / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  const parts: string[] = []
  if (d) parts.push(`${d}天`)
  if (h) parts.push(`${h}小时`)
  if (d || h || m) parts.push(`${m}分`)
  if (!d && h < 6) parts.push(`${s}秒`)
  return parts.join('') || '不足1分'
}

function remainClass(sec: number) {
  if (sec <= 0 || sec < 6 * 3600) return 'danger'
  if (sec < 24 * 3600) return 'warning'
  return ''
}

async function handleRequestSync() {
  if (!shopId.value) return
  requestingSync.value = true
  try {
    const item = await requestShopSync(shopId.value)
    shop.value = item
    ElMessage.success('已请求同步，插件下次心跳（约 1 分钟内）会采集')
  } catch (e) {
    ElMessage.error((e as Error).message || '请求失败')
  } finally {
    requestingSync.value = false
  }
}
</script>

<template>
  <div v-loading="loading" class="workbench">
    <div class="page-head">
      <div>
        <el-button text type="primary" @click="router.push('/shops')">← 店铺管理</el-button>
        <h2 class="page-title">{{ shop?.name || '售后工作台' }}</h2>
        <p class="desc">
          {{ shop?.platformLabel }}
          <template v-if="shop?.platformShopName"> · {{ shop.platformShopName }}</template>
          <template v-if="lastSyncAt"> · 最近同步 {{ lastSyncAt }}</template>
        </p>
      </div>
      <div class="head-actions">
        <el-button
          v-if="shop && shop.pluginStatus !== 'unbound'"
          :loading="requestingSync"
          @click="handleRequestSync"
        >
          {{ shop.syncRequested ? '已请求同步' : '请求插件同步' }}
        </el-button>
        <el-tag v-if="shop" :type="statusType(shop.pluginStatus)" size="large">
          {{ statusLabel(shop.pluginStatus) }}
        </el-tag>
      </div>
    </div>

    <el-card class="filter-card">
      <div class="filter-title">售后工作台</div>
      <el-empty v-if="!groupedCards.length" description="暂无卡片数据，请在抖店工作台打开插件并同步" :image-size="72" />
      <div v-else class="groups">
        <div v-for="group in groupedCards" :key="group.name" class="group">
          <div class="group-title" :class="{ urgent: urgentGroup(group.name) }">{{ group.name }}</div>
          <div class="group-items">
            <button
              v-for="card in group.items"
              :key="card.cardKey"
              type="button"
              class="card-item"
              :class="{ active: activeCardKey === card.cardKey, hot: card.count > 0 }"
              @click="selectCard(card)"
            >
              <span class="card-label">{{ card.cardLabel }}</span>
              <span class="card-count">{{ card.count }}</span>
            </button>
          </div>
        </div>
      </div>
    </el-card>

    <el-card>
      <div class="toolbar">
        <el-input
          v-model="keyword"
          clearable
          placeholder="售后编号 / 订单号 / 商品 / 状态 / 退货单号"
          style="width: 280px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <el-button v-if="activeCardKey" @click="selectCard()">清除卡片筛选</el-button>
        <span class="total">共 {{ total }} 条售后单</span>
      </div>

      <el-table :data="tickets" stripe border>
        <el-table-column label="商品信息" min-width="280">
          <template #default="{ row }">
            <div class="product">
              <img v-if="row.productImage" class="thumb" :src="row.productImage" alt="" />
              <div class="product-meta">
                <div class="title">{{ row.productTitle || '—' }}</div>
                <div v-if="row.sku" class="sub">{{ row.sku }}</div>
                <div v-if="row.productTags" class="tags">{{ row.productTags }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="订单信息" min-width="200">
          <template #default="{ row }">
            <div>应付金额 ¥{{ row.payAmount || '—' }}</div>
            <div class="sub">购买件数 {{ row.buyQty || row.qty || 0 }} 件</div>
            <div class="sub">订单 {{ row.orderNo || '—' }}</div>
            <div class="sub">售后 {{ row.platformAftersaleId }}</div>
            <div v-if="row.tags" class="sub">{{ row.tags }}</div>
          </template>
        </el-table-column>
        <el-table-column label="售后信息" min-width="180">
          <template #default="{ row }">
            <div>{{ row.aftersaleType || '—' }}</div>
            <div class="sub">售后退款 ¥{{ row.refundAmount || '—' }}</div>
            <div class="sub">申请件数 {{ row.qty || 0 }} 件</div>
            <div v-if="row.reason" class="sub">申请原因 {{ row.reason }}</div>
            <div v-if="row.applyTime" class="sub">申请时间 {{ row.applyTime }}</div>
          </template>
        </el-table-column>
        <el-table-column label="售后状态" min-width="190">
          <template #default="{ row }">
            <div>{{ row.status || '—' }}</div>
            <div
              v-if="row.deadlineAt || row.timeoutText"
              class="timeout"
              :class="remainClass(remainSecondsOf(row))"
            >
              <template v-if="row.deadlineAt || row.remainSeconds">
                <template v-if="remainSecondsOf(row) <= 0">
                  已超时<span v-if="row.timeoutAction"> · {{ row.timeoutAction }}</span>
                </template>
                <template v-else>
                  剩余 {{ formatRemain(remainSecondsOf(row)) }}
                  <span v-if="row.timeoutAction">后{{ row.timeoutAction }}</span>
                </template>
              </template>
              <template v-else>{{ row.timeoutText }}</template>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="dispute" label="纠纷仲裁" width="120" />
        <el-table-column label="物流信息" min-width="180">
          <template #default="{ row }">
            <pre class="logistics">{{ row.logistics || '—' }}</pre>
            <div v-if="row.returnLogisticsNo" class="tracking">退货单号 {{ row.returnLogisticsNo }}</div>
          </template>
        </el-table-column>
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

    <el-card class="filter-card">
      <div class="filter-title">服务工单（待处理）</div>
      <div class="toolbar" style="margin-top: 12px">
        <el-input
          v-model="serviceKeyword"
          clearable
          placeholder="工单ID / 订单号 / 商品 / 买家"
          style="width: 280px"
          @keyup.enter="handleServiceSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleServiceSearch">查询</el-button>
        <span class="total">共 {{ serviceTotal }} 条</span>
      </div>
      <el-table :data="serviceOrders" stripe border>
        <el-table-column label="订单 / 工单" min-width="260">
          <template #default="{ row }">
            <div class="product">
              <img v-if="row.productImage" class="thumb" :src="row.productImage" alt="" />
              <div class="product-meta">
                <div class="title">{{ row.productTitle || '—' }}</div>
                <div v-if="row.productContent" class="sub">{{ row.productContent }}</div>
                <div class="sub">订单 {{ row.orderNo || '—' }}</div>
                <div class="sub">工单 {{ row.platformServiceId }}</div>
                <div v-if="row.tags" class="tags">{{ row.tags }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="买家 / 来源" min-width="140">
          <template #default="{ row }">
            <div>{{ row.buyerNick || '—' }}</div>
            <div class="sub">{{ row.createSource || '—' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="工单类型" min-width="130">
          <template #default="{ row }">
            <div>{{ row.businessType || '—' }}</div>
            <div class="sub">{{ row.orderType }}</div>
          </template>
        </el-table-column>
        <el-table-column label="处理进度" min-width="180">
          <template #default="{ row }">
            <div>{{ row.status || '—' }}</div>
            <div
              v-if="row.deadlineAt || row.timeoutText"
              class="timeout"
              :class="remainClass(remainSecondsOf(row))"
            >
              <template v-if="row.deadlineAt || row.remainSeconds">
                <template v-if="remainSecondsOf(row) <= 0">
                  已逾期<span v-if="row.timeoutAction"> · {{ row.timeoutAction }}</span>
                </template>
                <template v-else>
                  剩余 {{ formatRemain(remainSecondsOf(row)) }}
                  <span v-if="row.timeoutAction">后{{ row.timeoutAction }}</span>
                </template>
              </template>
              <template v-else>{{ row.timeoutText }}</template>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="处理方案" min-width="120">
          <template #default="{ row }">{{ row.solution || '—' }}</template>
        </el-table-column>
        <el-table-column label="最近更新" min-width="160">
          <template #default="{ row }">
            <div>{{ row.lastLog || '—' }}</div>
            <div class="sub">{{ row.lastLogTime || row.createTime }}</div>
          </template>
        </el-table-column>
        <el-table-column label="工单要求" min-width="220">
          <template #default="{ row }">
            <div class="detail">{{ row.detail || '—' }}</div>
          </template>
        </el-table-column>
      </el-table>
      <div class="pager">
        <el-pagination
          v-model:current-page="servicePage"
          v-model:page-size="servicePageSize"
          :total="serviceTotal"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @current-change="loadService"
          @size-change="() => { servicePage = 1; loadService() }"
        />
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}
.head-actions { display: flex; align-items: center; gap: 10px; }
.page-title { margin: 4px 0 6px; font-size: 22px; }
.desc { color: #606266; margin: 0; }
.filter-card { margin-bottom: 16px; }
.filter-title { font-weight: 600; margin-bottom: 12px; }
.groups { display: flex; flex-direction: column; gap: 14px; }
.group-title { font-size: 13px; color: #606266; margin-bottom: 8px; }
.group-title.urgent { color: #f56c6c; font-weight: 600; }
.group-items { display: flex; flex-wrap: wrap; gap: 8px; }
.card-item {
  display: flex;
  align-items: center;
  gap: 8px;
  border: 1px solid #ebeef5;
  background: #fff;
  border-radius: 8px;
  padding: 8px 12px;
  cursor: pointer;
  font: inherit;
}
.card-item:hover { border-color: #409eff; }
.card-item.active { border-color: #409eff; background: #ecf5ff; }
.card-label { color: #303133; }
.card-count { font-weight: 700; color: #909399; min-width: 1.2em; text-align: right; }
.card-item.hot .card-count { color: #409eff; }
.card-item.active .card-count { color: #409eff; }
.card-item.danger .card-count { color: #f56c6c; }
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.total { margin-left: auto; color: #909399; font-size: 13px; }
.product { display: flex; gap: 10px; align-items: flex-start; }
.thumb { width: 48px; height: 48px; border-radius: 4px; object-fit: cover; flex-shrink: 0; background: #f5f7fa; }
.product-meta { min-width: 0; }
.product .title { font-weight: 600; line-height: 1.4; }
.tags { color: #69718c; font-size: 12px; margin-top: 4px; }
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
.timeout { color: #e6a23c; font-size: 12px; margin-top: 2px; line-height: 1.4; }
.timeout.warning { color: #e6a23c; font-weight: 600; }
.timeout.danger { color: #f56c6c; font-weight: 700; }
.logistics { margin: 0; font: inherit; white-space: pre-line; color: #303133; }
.tracking { color: #409eff; font-size: 12px; margin-top: 4px; word-break: break-all; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
.detail { font-size: 12px; color: #606266; line-height: 1.45; display: -webkit-box; -webkit-line-clamp: 4; -webkit-box-orient: vertical; overflow: hidden; }
</style>
