<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import TicketLogisticsCell from '../../components/TicketLogisticsCell.vue'
import {
  fetchShopTicketsByKind,
  fetchShops,
  type AftersaleTicket,
  type MarketplaceShop,
  type ShopTicketKind,
} from '../../api/shop'
import { parseTicketLogistics } from '../../utils/ticketLogistics'

const route = useRoute()
const kind = computed(() => (route.meta.kind as ShopTicketKind) || 'buyer-return-pickup')
const pageTitle = computed(() => (route.meta.title as string) || '售后单')
const pageDesc = computed(() => (route.meta.desc as string) || '')

const loading = ref(false)
const shops = ref<MarketplaceShop[]>([])
const tickets = ref<AftersaleTicket[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const shopId = ref<number | undefined>()
const keyword = ref('')
const nowTick = ref(Date.now())
let tickTimer = 0

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
    const data = await fetchShopTicketsByKind({
      kind: kind.value,
      shopId: shopId.value || undefined,
      keyword: keyword.value || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    tickets.value = data.list
    total.value = data.total
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

function remainSecondsOf(row: AftersaleTicket) {
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
watch(kind, () => {
  keyword.value = ''
  page.value = 1
  loadData()
})
</script>

<template>
  <div v-loading="loading" class="ticket-list">
    <div class="page-head">
      <div>
        <h2 class="page-title">{{ pageTitle }}</h2>
        <p class="desc">{{ pageDesc }}</p>
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
        <el-input
          v-model="keyword"
          clearable
          placeholder="售后编号 / 订单号 / 商品 / 退货单号"
          style="width: 320px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <span class="total">共 {{ total }} 条</span>
      </div>

      <el-table :data="tickets" stripe border>
        <el-table-column prop="shopName" label="店铺" width="140" />
        <el-table-column label="商品信息" min-width="240">
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
        <el-table-column label="售后状态" min-width="180">
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
        <el-table-column label="物流信息" min-width="220">
          <template #default="{ row }">
            <TicketLogisticsCell
              v-bind="parseTicketLogistics(row)"
              :tracks="row.tracks"
              :raw="row.logistics"
            />
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
.page-head { margin-bottom: 16px; }
.page-title { margin: 0 0 6px; font-size: 22px; }
.desc { color: #606266; margin: 0; }
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.total { margin-left: auto; color: #909399; font-size: 13px; }
.product { display: flex; gap: 10px; align-items: flex-start; }
.thumb { width: 48px; height: 48px; border-radius: 4px; object-fit: cover; flex-shrink: 0; background: #f5f7fa; }
.product-meta { min-width: 0; }
.title { font-weight: 600; line-height: 1.4; }
.tags { color: #69718c; font-size: 12px; margin-top: 4px; }
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
.timeout { color: #e6a23c; font-size: 12px; margin-top: 2px; line-height: 1.4; }
.timeout.warning { color: #e6a23c; font-weight: 600; }
.timeout.danger { color: #f56c6c; font-weight: 700; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
