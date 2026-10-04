<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Download, Search } from '@element-plus/icons-vue'
import { exportPublicReturnShare, fetchPublicReturnShare } from '../../api/publicReturnShare'
import type { LogisticsTrack, ReturnPackage } from '../../api/shop'
import { displayTrackDetail } from '../../utils/ticketLogistics'

const route = useRoute()
const token = computed(() => String(route.params.token || '').trim())
const loading = ref(false)
const exporting = ref(false)
const tableData = ref<ReturnPackage[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const sharerName = ref('')
const returnLocation = ref('')
const missing = ref(false)

function seq(index: number) {
  return (page.value - 1) * pageSize.value + index + 1
}

async function loadData() {
  if (!token.value) {
    missing.value = true
    return
  }
  loading.value = true
  try {
    const data = await fetchPublicReturnShare(token.value, {
      keyword: keyword.value || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    tableData.value = data.list || []
    total.value = data.total
    sharerName.value = data.name || ''
    returnLocation.value = data.returnLocation || ''
    missing.value = false
    document.title = (sharerName.value ? `${sharerName.value} · ` : '') + '退回件'
  } catch (e) {
    missing.value = true
    ElMessage.error((e as Error).message || '链接无效或已停用')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadData()
}

async function exportExcel() {
  exporting.value = true
  try {
    const { blob, filename } = await exportPublicReturnShare(token.value)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
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

onMounted(loadData)
</script>

<template>
  <div v-loading="loading" class="public-share">
    <div class="frame">
      <header class="hero">
        <div class="brand">退回件</div>
        <h1>{{ sharerName || '退回件分享' }}</h1>
        <p v-if="returnLocation" class="meta">退回地 {{ returnLocation }}</p>
        <p class="hint">刷新本页始终显示最新结果。不含店铺名，按序号查看。</p>
      </header>

      <div v-if="missing && !loading" class="empty">分享链接无效或已停用</div>

      <template v-else>
        <div class="toolbar">
          <el-input
            v-model="keyword"
            clearable
            placeholder="物流单号 / 订单号 / 售后编号 / 商品"
            style="width: 280px"
            @keyup.enter="handleSearch"
          />
          <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
          <el-button :icon="Download" :loading="exporting" @click="exportExcel">导出 Excel</el-button>
          <span class="total">共 {{ total }} 条</span>
        </div>

        <el-table :data="tableData" stripe border>
          <el-table-column label="序号" width="70" align="center">
            <template #default="{ $index }">{{ seq($index) }}</template>
          </el-table-column>
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
              <div>购买件数 {{ row.buyQty || row.qty || 0 }} 件</div>
              <div class="sub">订单 {{ row.orderNo || '—' }}</div>
              <div class="sub">售后 {{ row.platformAftersaleId }}</div>
            </template>
          </el-table-column>
          <el-table-column label="售后信息" min-width="220">
            <template #default="{ row }">
              <div>{{ row.aftersaleType || '已发货退款' }}</div>
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
          <el-table-column label="分发备注" min-width="180">
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
      </template>
    </div>
  </div>
</template>

<style scoped>
.public-share { min-height: 100vh; background: #f5f7fa; padding: 24px 16px 48px; }
.frame { max-width: 1280px; margin: 0 auto; background: #fff; border-radius: 12px; padding: 24px; box-shadow: 0 8px 24px #0000000d; }
.hero { margin-bottom: 20px; }
.brand { color: #409eff; font-weight: 600; font-size: 13px; letter-spacing: 0.08em; }
h1 { margin: 6px 0; font-size: 24px; }
.meta { margin: 0; color: #303133; }
.hint { margin: 6px 0 0; color: #909399; font-size: 13px; }
.empty { padding: 48px 0; text-align: center; color: #909399; }
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
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
</style>
