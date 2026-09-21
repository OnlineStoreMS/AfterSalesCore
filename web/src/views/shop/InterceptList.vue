<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import AftersaleStatusCell from '../../components/AftersaleStatusCell.vue'
import TicketLogisticsCell from '../../components/TicketLogisticsCell.vue'
import { fetchInterceptOrders, fetchShops, type InterceptOrder, type MarketplaceShop } from '../../api/shop'
import { parseTicketLogistics } from '../../utils/ticketLogistics'

const loading = ref(false)
const shops = ref<MarketplaceShop[]>([])
const tableData = ref<InterceptOrder[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const shopId = ref<number | undefined>()
const keyword = ref('')

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
    const data = await fetchInterceptOrders({
      shopId: shopId.value || undefined,
      keyword: keyword.value || undefined,
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

function handleSearch() {
  page.value = 1
  loadData()
}

onMounted(() => {
  loadShops()
  loadData()
})
</script>

<template>
  <div v-loading="loading" class="intercept-list">
    <div class="page-head">
      <div>
        <h2 class="page-title">需商家拦截快递</h2>
        <p class="desc">汇总各店铺「需商家拦截快递」的售后单，以及已发货退款成功里订单发货为待取件的单据。买家退货待取件不计入。按申请时间最近的在前。</p>
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
          placeholder="订单号 / 售后编号 / 商品 / 物流单号"
          style="width: 320px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <span class="total">共 {{ total }} 条</span>
      </div>

      <el-table :data="tableData" stripe border>
        <el-table-column prop="shopName" label="店铺" width="140" />
        <el-table-column label="原因" width="150">
          <template #default="{ row }">
            <div v-if="row.needIntercept" class="tag-danger">需商家拦截快递</div>
            <div v-if="row.awaitPickup" class="tag-danger">待取件</div>
          </template>
        </el-table-column>
        <el-table-column label="商品信息" min-width="220">
          <template #default="{ row }">
            <div class="product">
              <img v-if="row.productImage" class="thumb" :src="row.productImage" alt="" />
              <div class="product-meta">
                <div class="title">{{ row.productTitle || '—' }}</div>
                <div v-if="row.sku" class="sub">{{ row.sku }}</div>
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
          </template>
        </el-table-column>
        <el-table-column label="售后信息" min-width="200">
          <template #default="{ row }">
            <div>{{ row.aftersaleType || '—' }}</div>
            <div class="sub">售后退款 ¥{{ row.refundAmount || '—' }}</div>
            <div class="sub">申请件数 {{ row.qty || 0 }} 件</div>
            <div v-if="row.reason" class="sub">申请原因 {{ row.reason }}</div>
            <div v-if="row.applyTime" class="sub">申请时间 {{ row.applyTime }}</div>
          </template>
        </el-table-column>
        <el-table-column label="售后状态" min-width="220">
          <template #default="{ row }">
            <AftersaleStatusCell
              :status="row.status"
              :timeout-text="row.timeoutText"
              :timeout-action="row.timeoutAction"
              :timeout-display="row.timeoutDisplay"
              :deadline-at="row.deadlineAt"
              :remain-seconds="row.remainSeconds"
            />
          </template>
        </el-table-column>
        <el-table-column label="物流信息" min-width="200">
          <template #default="{ row }">
            <TicketLogisticsCell
              v-bind="parseTicketLogistics(row)"
              :tracks="row.tracks"
              :raw="row.logisticsStatus || row.logistics"
              :carrier="row.carrier"
            />
          </template>
        </el-table-column>
        <el-table-column label="申请时间" width="170">
          <template #default="{ row }">{{ row.applyTime || '—' }}</template>
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
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
.total { margin-left: auto; color: #909399; font-size: 13px; }
.product { display: flex; gap: 10px; align-items: flex-start; }
.thumb { width: 48px; height: 48px; border-radius: 4px; object-fit: cover; flex-shrink: 0; background: #f5f7fa; }
.product-meta { min-width: 0; }
.title { font-weight: 600; line-height: 1.4; }
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
.tag-danger { color: #f56c6c; font-weight: 600; line-height: 1.5; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
