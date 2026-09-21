<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Download, Search } from '@element-plus/icons-vue'
import {
  RETURN_EXPORT_FIELDS,
  exportReturnPackages,
  fetchReturnPackages,
  fetchShops,
  type LogisticsTrack,
  type MarketplaceShop,
  type ReturnPackage,
} from '../../api/shop'
import { dateRangeDefaultTime, dateShortcuts } from '../../utils/date'
import { displayTrackDetail } from '../../utils/ticketLogistics'

const loading = ref(false)
const shops = ref<MarketplaceShop[]>([])
const tableData = ref<ReturnPackage[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const shopId = ref<number | undefined>()
const keyword = ref('')
const returnRange = ref<[string, string] | null>(null)
const applyRange = ref<[string, string] | null>(null)
const exportVisible = ref(false)
const exporting = ref(false)
const exportFields = ref(RETURN_EXPORT_FIELDS.map((f) => f.key))

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
    const data = await fetchReturnPackages({
      shopId: shopId.value || undefined,
      keyword: keyword.value || undefined,
      returnFrom: returnRange.value?.[0] || undefined,
      returnTo: returnRange.value?.[1] || undefined,
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

function openExport() {
  exportVisible.value = true
}

async function confirmExport() {
  if (!exportFields.value.length) {
    ElMessage.warning('请至少勾选一个导出字段')
    return
  }
  exporting.value = true
  try {
    const { blob, filename } = await exportReturnPackages({
      shopId: shopId.value || undefined,
      keyword: keyword.value || undefined,
      returnFrom: returnRange.value?.[0] || undefined,
      returnTo: returnRange.value?.[1] || undefined,
      applyFrom: applyRange.value?.[0] || undefined,
      applyTo: applyRange.value?.[1] || undefined,
      fields: exportFields.value,
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
    exportVisible.value = false
    ElMessage.success('已开始下载')
  } catch (e) {
    ElMessage.error((e as Error).message || '导出失败')
  } finally {
    exporting.value = false
  }
}

function hasTracks(row: ReturnPackage) {
  return Boolean(row.tracks?.length)
}

function trackDetail(track: LogisticsTrack) {
  return displayTrackDetail(track)
}

onMounted(() => {
  loadShops()
  loadData()
})
</script>

<template>
  <div v-loading="loading" class="return-list">
    <div class="page-head">
      <div>
        <h2 class="page-title">退回管理</h2>
        <p class="desc">默认按物流退回时间排序，没有退回时间则按申请时间，最近的在前。悬停物流单号可查看最近 5 条轨迹。</p>
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
        <span class="field-label">物流退回时间</span>
        <el-date-picker
          v-model="returnRange"
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
          placeholder="物流单号 / 退回地 / 订单号 / 售后编号 / 商品"
          style="width: 300px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <el-button :icon="Download" @click="openExport">导出 Excel</el-button>
        <span class="total">共 {{ total }} 条退回件</span>
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
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="订单信息" min-width="220">
          <template #default="{ row }">
            <div>应付金额 ¥{{ row.payAmount || '—' }}</div>
            <div class="sub">购买件数 {{ row.buyQty || row.qty || 0 }} 件</div>
            <div class="sub">订单 {{ row.orderNo || '—' }}</div>
            <div class="sub">售后 {{ row.platformAftersaleId }}</div>
          </template>
        </el-table-column>
        <el-table-column label="售后信息" min-width="220">
          <template #default="{ row }">
            <div>{{ row.aftersaleType || '已发货退款' }}</div>
            <div class="sub">售后退款 ¥{{ row.refundAmount || '—' }}</div>
            <div class="sub">申请件数 {{ row.qty || 0 }} 件</div>
            <div v-if="row.reason" class="sub">申请原因 {{ row.reason }}</div>
            <div v-if="row.applyTime" class="sub">申请时间 {{ row.applyTime }}</div>
          </template>
        </el-table-column>
        <el-table-column label="物流单号" min-width="180">
          <template #default="{ row }">
            <el-popover
              placement="left-start"
              :width="360"
              trigger="hover"
              :disabled="!hasTracks(row)"
              popper-class="return-track-popper"
            >
              <template #reference>
                <div class="logistics-cell" :class="{ link: hasTracks(row) }">
                  <div class="tracking">{{ row.logisticsNo || '—' }}</div>
                  <div v-if="row.carrier" class="sub">{{ row.carrier }}</div>
                  <div v-if="row.logistics" class="sub">{{ row.logistics }}</div>
                </div>
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
        <el-table-column label="分发备注" min-width="200">
          <template #default="{ row }">
            <div class="location">{{ row.fenFaRemark || '—' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="退回地" min-width="220">
          <template #default="{ row }">
            <div class="location">{{ row.returnLocation || '—' }}</div>
          </template>
        </el-table-column>
        <el-table-column label="物流退回时间" width="170">
          <template #default="{ row }">{{ row.returnTime || '—' }}</template>
        </el-table-column>
        <el-table-column label="申请时间" width="170">
          <template #default="{ row }">{{ row.applyTime || '—' }}</template>
        </el-table-column>
        <el-table-column prop="syncedAt" label="同步时间" width="170" />
      </el-table>

      <el-dialog v-model="exportVisible" title="导出 Excel" width="480px">
        <p class="export-hint">按当前筛选条件导出，最多 1500 条。商品信息会带上图片；退回地默认导出最新两条轨迹。</p>
        <el-checkbox-group v-model="exportFields" class="export-fields">
          <el-checkbox v-for="f in RETURN_EXPORT_FIELDS" :key="f.key" :label="f.key">
            {{ f.label }}
          </el-checkbox>
        </el-checkbox-group>
        <template #footer>
          <el-button @click="exportVisible = false">取消</el-button>
          <el-button type="primary" :loading="exporting" @click="confirmExport">导出</el-button>
        </template>
      </el-dialog>

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
.logistics-cell.link { cursor: pointer; }
.tracking { color: #409eff; word-break: break-all; }
.location { white-space: pre-wrap; line-height: 1.5; word-break: break-word; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
.export-hint { margin: 0 0 12px; color: #606266; font-size: 13px; line-height: 1.5; }
.export-fields { display: flex; flex-wrap: wrap; gap: 8px 16px; }
</style>

<style>
.return-track-popper .track-meta { color: #909399; font-size: 12px; margin-bottom: 8px; }
.return-track-popper .track-list { margin: 0; padding: 0; list-style: none; }
.return-track-popper .track-list li { position: relative; padding: 0 0 12px 14px; border-left: 2px solid #e4e7ed; }
.return-track-popper .track-list li:last-child { padding-bottom: 0; border-left-color: transparent; }
.return-track-popper .track-list li::before {
  content: '';
  position: absolute;
  left: -6px;
  top: 4px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #c0c4cc;
}
.return-track-popper .track-list li:first-child::before { background: #409eff; }
.return-track-popper .track-title { font-weight: 600; line-height: 1.4; }
.return-track-popper .track-date { color: #909399; font-size: 12px; margin-top: 2px; }
.return-track-popper .track-detail { color: #606266; font-size: 12px; margin-top: 2px; line-height: 1.5; word-break: break-word; }
</style>
