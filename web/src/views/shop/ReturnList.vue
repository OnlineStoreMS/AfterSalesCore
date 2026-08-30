<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import {
  fetchReturnPackages,
  fetchShops,
  type MarketplaceShop,
  type ReturnPackage,
} from '../../api/shop'

const loading = ref(false)
const shops = ref<MarketplaceShop[]>([])
const tableData = ref<ReturnPackage[]>([])
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
    const data = await fetchReturnPackages({
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
  <div v-loading="loading" class="return-list">
    <div class="page-head">
      <div>
        <h2 class="page-title">退回管理</h2>
        <p class="desc">已发货退款且物流为「订单发货 / 已退回」的退回件，退回地取物流轨迹倒数第二条。</p>
      </div>
    </div>

    <el-card>
      <div class="toolbar">
        <el-select
          v-model="shopId"
          clearable
          placeholder="全部店铺"
          style="width: 200px"
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
          placeholder="物流单号 / 退回地 / 订单号 / 售后编号 / 商品"
          style="width: 320px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <span class="total">共 {{ total }} 条退回件</span>
      </div>

      <el-table :data="tableData" stripe border>
        <el-table-column prop="shopName" label="店铺" width="140" />
        <el-table-column label="商品信息" min-width="260">
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
        <el-table-column label="订单 / 售后" min-width="200">
          <template #default="{ row }">
            <div>订单 {{ row.orderNo || '—' }}</div>
            <div class="sub">售后 {{ row.platformAftersaleId }}</div>
            <div class="sub">{{ row.aftersaleType || '已发货退款' }} · {{ row.status || '—' }}</div>
            <div v-if="row.applyTime" class="sub">申请 {{ row.applyTime }}</div>
          </template>
        </el-table-column>
        <el-table-column label="物流单号" min-width="170">
          <template #default="{ row }">
            <div class="tracking">{{ row.logisticsNo || '—' }}</div>
            <div v-if="row.carrier" class="sub">{{ row.carrier }}</div>
            <div v-if="row.logistics" class="sub">{{ row.logistics }}</div>
            <div v-if="row.shipTime" class="sub">发货 {{ row.shipTime }}</div>
          </template>
        </el-table-column>
        <el-table-column label="退回地" min-width="260">
          <template #default="{ row }">
            <div class="location">{{ row.returnLocation || '—' }}</div>
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
.tracking { color: #409eff; word-break: break-all; }
.location { white-space: pre-wrap; line-height: 1.5; word-break: break-word; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
