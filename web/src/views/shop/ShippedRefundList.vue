<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import {
  fetchShippedRefunds,
  fetchShops,
  type LogisticsTrack,
  type MarketplaceShop,
  type ShippedRefund,
} from '../../api/shop'
import { dateRangeDefaultTime, dateShortcuts } from '../../utils/date'

const loading = ref(false)
const shops = ref<MarketplaceShop[]>([])
const tableData = ref<ShippedRefund[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const shopId = ref<number | undefined>()
const keyword = ref('')
const status = ref('')
const applyRange = ref<[string, string] | null>(null)

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
    const data = await fetchShippedRefunds({
      shopId: shopId.value || undefined,
      keyword: keyword.value || undefined,
      status: status.value || undefined,
      applyFrom: applyRange.value?.[0] || undefined,
      applyTo: applyRange.value?.[1] || undefined,
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

function isAlert(row: ShippedRefund) {
  return row.alert || ['待取件', '已签收', '运输中'].includes(row.logisticsStatus || '')
}

function hasTracks(row: ShippedRefund) {
  return Boolean(row.tracks?.length)
}

function trackDetail(track: LogisticsTrack) {
  return track.detail || (track.title || track.date ? '' : track.text || '')
}

onMounted(() => {
  loadShops()
  loadData()
})
</script>

<template>
  <div v-loading="loading" class="shipped-list">
    <div class="page-head">
      <div>
        <h2 class="page-title">已发货退款成功</h2>
        <p class="desc">已发货退款且退款成功、尚未退回的售后单。物流为待取件、已签收、运输中时标红，可在通知管理里推送紧急飞书卡片。</p>
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
          v-model="status"
          clearable
          placeholder="物流状态"
          style="width: 140px"
          @change="handleSearch"
        >
          <el-option label="待取件" value="待取件" />
          <el-option label="已签收" value="已签收" />
          <el-option label="运输中" value="运输中" />
          <el-option label="已发货" value="已发货" />
        </el-select>
        <span class="field-label">申请时间</span>
        <el-date-picker
          v-model="applyRange"
          type="datetimerange"
          range-separator="至"
          start-placeholder="开始"
          end-placeholder="结束"
          value-format="YYYY-MM-DD HH:mm:ss"
          :shortcuts="dateShortcuts"
          :default-time="dateRangeDefaultTime"
          clearable
          style="width: 360px"
          @change="handleSearch"
        />
        <el-input
          v-model="keyword"
          clearable
          placeholder="订单号 / 售后编号 / 商品 / 物流 / 订单信息"
          style="width: 300px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <span class="total">共 {{ total }} 条</span>
      </div>

      <el-table :data="tableData" stripe border>
        <el-table-column prop="shopName" label="店铺" width="140" />
        <el-table-column label="商品信息" min-width="240">
          <template #default="{ row }">
            <div class="product">
              <img v-if="row.productImage" class="thumb" :src="row.productImage" alt="" />
              <div class="product-meta">
                <div class="title">{{ row.productTitle || '—' }}</div>
                <div v-if="row.sku" class="sub">{{ row.sku }}</div>
                <div v-if="row.productTags" class="sub">{{ row.productTags }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="订单信息" min-width="220">
          <template #default="{ row }">
            <pre v-if="row.orderInfo" class="cell-pre">{{ row.orderInfo }}</pre>
            <template v-else>
              <div>应付金额 ¥{{ row.payAmount || '—' }}</div>
              <div class="sub">购买件数 {{ row.buyQty || row.qty || 0 }} 件</div>
              <div class="sub">订单 {{ row.orderNo || '—' }}</div>
              <div class="sub">售后 {{ row.platformAftersaleId }}</div>
            </template>
          </template>
        </el-table-column>
        <el-table-column label="售后信息" min-width="220">
          <template #default="{ row }">
            <pre v-if="row.aftersaleInfo" class="cell-pre">{{ row.aftersaleInfo }}</pre>
            <template v-else>
              <div>{{ row.aftersaleType || '已发货退款' }}</div>
              <div class="sub">售后退款 ¥{{ row.refundAmount || '—' }}</div>
              <div class="sub">申请件数 {{ row.qty || 0 }} 件</div>
              <div v-if="row.reason" class="sub">申请原因 {{ row.reason }}</div>
              <div v-if="row.applyTime" class="sub">申请时间 {{ row.applyTime }}</div>
            </template>
          </template>
        </el-table-column>
        <el-table-column label="物流信息" width="120">
          <template #default="{ row }">
            <el-popover
              placement="left-start"
              :width="360"
              trigger="hover"
              :disabled="!hasTracks(row)"
              popper-class="shipped-track-popper"
            >
              <template #reference>
                <span class="logistics-status" :class="{ danger: isAlert(row), link: hasTracks(row) }">
                  {{ row.logisticsStatus || '—' }}
                </span>
              </template>
              <div class="track-pop">
                <div v-if="row.logisticsNo || row.carrier" class="track-meta">
                  {{ [row.carrier, row.logisticsNo].filter(Boolean).join(' ') }}
                </div>
                <ul class="track-list">
                  <li v-for="(track, i) in (row.tracks || []).slice(0, 5)" :key="i">
                    <div class="track-title">{{ track.title || '物流记录' }}</div>
                    <div v-if="track.date" class="track-date">{{ track.date }}</div>
                    <div v-if="trackDetail(track)" class="track-detail">{{ trackDetail(track) }}</div>
                  </li>
                </ul>
              </div>
            </el-popover>
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
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.field-label { color: #606266; font-size: 13px; white-space: nowrap; }
.total { margin-left: auto; color: #909399; font-size: 13px; }
.product { display: flex; gap: 10px; align-items: flex-start; }
.thumb { width: 48px; height: 48px; border-radius: 4px; object-fit: cover; flex-shrink: 0; background: #f5f7fa; }
.product-meta { min-width: 0; }
.title { font-weight: 600; line-height: 1.4; }
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
.cell-pre { margin: 0; white-space: pre-wrap; line-height: 1.5; word-break: break-word; font: inherit; color: inherit; }
.logistics-status { font-weight: 600; }
.logistics-status.link { color: #409eff; cursor: pointer; }
.logistics-status.danger { color: #f56c6c; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>

<style>
.shipped-track-popper .track-meta { color: #909399; font-size: 12px; margin-bottom: 8px; }
.shipped-track-popper .track-list { margin: 0; padding: 0; list-style: none; }
.shipped-track-popper .track-list li { position: relative; padding: 0 0 12px 14px; border-left: 2px solid #e4e7ed; }
.shipped-track-popper .track-list li:last-child { padding-bottom: 0; border-left-color: transparent; }
.shipped-track-popper .track-list li::before {
  content: '';
  position: absolute;
  left: -6px;
  top: 4px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #c0c4cc;
}
.shipped-track-popper .track-list li:first-child::before { background: #409eff; }
.shipped-track-popper .track-title { font-weight: 600; line-height: 1.4; }
.shipped-track-popper .track-date { color: #909399; font-size: 12px; margin-top: 2px; }
.shipped-track-popper .track-detail { color: #606266; font-size: 12px; margin-top: 2px; line-height: 1.5; word-break: break-word; }
</style>
