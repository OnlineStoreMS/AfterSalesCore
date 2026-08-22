<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import {
  PLUGIN_STATUS_MAP,
  fetchShopWorkbench,
  type FilterCard,
  type MarketplaceShop,
  type AftersaleTicket,
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
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
watch(shopId, () => {
  activeCardKey.value = ''
  keyword.value = ''
  page.value = 1
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

function statusType(s: string) {
  return PLUGIN_STATUS_MAP[s as keyof typeof PLUGIN_STATUS_MAP]?.type || 'info'
}

function statusLabel(s: string) {
  return PLUGIN_STATUS_MAP[s as keyof typeof PLUGIN_STATUS_MAP]?.label || s
}

function urgentGroup(name: string) {
  return name === '紧急'
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
      <el-tag v-if="shop" :type="statusType(shop.pluginStatus)" size="large">
        {{ statusLabel(shop.pluginStatus) }}
      </el-tag>
    </div>

    <el-card class="filter-card">
      <div class="filter-title">快捷筛选</div>
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
          placeholder="售后编号 / 订单号 / 商品 / 状态"
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
        <el-table-column label="售后状态" min-width="150">
          <template #default="{ row }">
            <div>{{ row.status || '—' }}</div>
            <div v-if="row.timeoutText" class="timeout">{{ row.timeoutText }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="dispute" label="纠纷仲裁" width="120" />
        <el-table-column label="物流信息" min-width="150">
          <template #default="{ row }">
            <pre class="logistics">{{ row.logistics || '—' }}</pre>
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
  </div>
</template>

<style scoped>
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
}
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
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.total { margin-left: auto; color: #909399; font-size: 13px; }
.product { display: flex; gap: 10px; align-items: flex-start; }
.thumb { width: 48px; height: 48px; border-radius: 4px; object-fit: cover; flex-shrink: 0; background: #f5f7fa; }
.product-meta { min-width: 0; }
.product .title { font-weight: 600; line-height: 1.4; }
.tags { color: #69718c; font-size: 12px; margin-top: 4px; }
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
.timeout { color: #e6a23c; font-size: 12px; margin-top: 2px; }
.logistics { margin: 0; font: inherit; white-space: pre-line; color: #303133; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
