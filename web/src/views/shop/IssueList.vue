<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CopyDocument, Plus, Search } from '@element-plus/icons-vue'
import { fetchShops, type MarketplaceShop } from '../../api/shop'
import {
  buildIssueHandleText,
  buildMergedIssueHandleText,
  copyText,
  createIssue,
  deleteIssue,
  fetchIssueMeta,
  fetchIssues,
  lookupIssueSource,
  updateIssue,
  updateIssueStatus,
  type AftersaleIssue,
  type IssueLookup,
  type IssueMeta,
  type IssueUpsertBody,
} from '../../api/issue'

const loading = ref(false)
const shops = ref<MarketplaceShop[]>([])
const meta = ref<IssueMeta>({ problemTypes: [], handleMethods: [], statuses: [] })
const list = ref<AftersaleIssue[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const shopId = ref<number | undefined>()
const status = ref('pending')
const problemType = ref('')
const keyword = ref('')
const selected = ref<AftersaleIssue[]>([])

const dialogVisible = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const lookupQ = ref('')
const lookingUp = ref(false)
const lookupHits = ref<IssueLookup[]>([])

const form = reactive<IssueUpsertBody>({
  shopId: undefined,
  shopName: '',
  orderNo: '',
  platformOrderId: '',
  platformAftersaleId: '',
  productTitle: '',
  productImage: '',
  skuSpecs: '',
  buyerName: '',
  buyerPhone: '',
  buyerAddress: '',
  hasAftersale: false,
  aftersaleStatus: '',
  aftersaleType: '',
  logistics: '',
  returnLogisticsNo: '',
  shipLogisticsNo: '',
  problemType: '',
  problemTypeCustom: '',
  handleMethod: '',
  handleMethodCustom: '',
  problemNote: '',
  handleNote: '',
  status: 'pending',
})

const isCustomProblem = computed(() => form.problemType === 'custom')
const isCustomHandle = computed(() => form.handleMethod === 'custom')

async function loadShops() {
  try {
    shops.value = await fetchShops()
  } catch {
    shops.value = []
  }
}

async function loadMeta() {
  try {
    meta.value = await fetchIssueMeta()
  } catch {
    /* ignore */
  }
}

async function loadData() {
  loading.value = true
  try {
    const data = await fetchIssues({
      shopId: shopId.value || undefined,
      status: status.value || undefined,
      problemType: problemType.value || undefined,
      keyword: keyword.value || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e: any) {
    ElMessage.error(e.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  page.value = 1
  loadData()
}

function onSelectionChange(rows: AftersaleIssue[]) {
  selected.value = rows
}

function resetForm() {
  editingId.value = null
  lookupQ.value = ''
  lookupHits.value = []
  Object.assign(form, {
    shopId: undefined,
    shopName: '',
    orderNo: '',
    platformOrderId: '',
    platformAftersaleId: '',
    productTitle: '',
    productImage: '',
    skuSpecs: '',
    buyerName: '',
    buyerPhone: '',
    buyerAddress: '',
    hasAftersale: false,
    aftersaleStatus: '',
    aftersaleType: '',
    logistics: '',
    returnLogisticsNo: '',
    shipLogisticsNo: '',
    problemType: '',
    problemTypeCustom: '',
    handleMethod: '',
    handleMethodCustom: '',
    problemNote: '',
    handleNote: '',
    status: 'pending',
  })
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

function openEdit(row: AftersaleIssue) {
  editingId.value = row.id
  lookupHits.value = []
  lookupQ.value = row.orderNo || row.platformAftersaleId || ''
  Object.assign(form, {
    shopId: row.shopId,
    shopName: row.shopName,
    orderNo: row.orderNo || '',
    platformOrderId: row.platformOrderId || '',
    platformAftersaleId: row.platformAftersaleId || '',
    productTitle: row.productTitle,
    productImage: row.productImage || '',
    skuSpecs: row.skuSpecs || '',
    buyerName: row.buyerName || '',
    buyerPhone: row.buyerPhone || '',
    buyerAddress: row.buyerAddress || '',
    hasAftersale: row.hasAftersale,
    aftersaleStatus: row.aftersaleStatus || '',
    aftersaleType: row.aftersaleType || '',
    logistics: row.logistics || '',
    returnLogisticsNo: row.returnLogisticsNo || '',
    shipLogisticsNo: row.shipLogisticsNo || '',
    problemType: row.problemType,
    problemTypeCustom: row.problemTypeCustom || '',
    handleMethod: row.handleMethod || '',
    handleMethodCustom: row.handleMethodCustom || '',
    problemNote: row.problemNote || '',
    handleNote: row.handleNote || '',
    status: row.status,
  })
  dialogVisible.value = true
}

function applyLookup(hit: IssueLookup) {
  form.shopId = hit.shopId
  form.shopName = hit.shopName || form.shopName
  form.orderNo = hit.orderNo || ''
  form.platformOrderId = hit.platformOrderId || ''
  form.platformAftersaleId = hit.platformAftersaleId || ''
  form.productTitle = hit.productTitle || ''
  form.productImage = hit.productImage || ''
  form.skuSpecs = hit.skuSpecs || ''
  form.buyerName = hit.buyerName || ''
  form.buyerPhone = hit.buyerPhone || ''
  form.buyerAddress = hit.buyerAddress || ''
  form.hasAftersale = !!hit.hasAftersale
  form.aftersaleStatus = hit.aftersaleStatus || ''
  form.aftersaleType = hit.aftersaleType || ''
  form.logistics = hit.logistics || ''
  form.returnLogisticsNo = hit.returnLogisticsNo || ''
  form.shipLogisticsNo = hit.shipLogisticsNo || ''
}

async function doLookup() {
  const q = lookupQ.value.trim()
  if (!q) {
    ElMessage.warning('请输入订单号或售后单号')
    return
  }
  lookingUp.value = true
  try {
    const data = await lookupIssueSource(q)
    lookupHits.value = data.list || []
    if (!lookupHits.value.length) {
      ElMessage.warning('未找到订单或售后单')
      return
    }
    if (lookupHits.value.length === 1) {
      applyLookup(lookupHits.value[0])
      ElMessage.success('已带入订单/售后信息')
    }
  } catch (e: any) {
    ElMessage.error(e.message || '检索失败')
  } finally {
    lookingUp.value = false
  }
}

async function saveIssue() {
  if (!form.shopName?.trim() || !form.productTitle?.trim()) {
    ElMessage.warning('店铺和商品信息必填，请先检索订单/售后单')
    return
  }
  if (!form.orderNo?.trim() && !form.platformAftersaleId?.trim()) {
    ElMessage.warning('订单号或售后单号至少填一项')
    return
  }
  if (!form.problemType) {
    ElMessage.warning('请选择问题类型')
    return
  }
  if (form.problemType === 'custom' && !form.problemTypeCustom?.trim()) {
    ElMessage.warning('请填写自定义问题类型')
    return
  }
  if (form.handleMethod === 'custom' && !form.handleMethodCustom?.trim()) {
    ElMessage.warning('请填写自定义处理方式')
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await updateIssue(editingId.value, { ...form })
      ElMessage.success('已更新')
    } else {
      await createIssue({ ...form })
      ElMessage.success('已创建')
    }
    dialogVisible.value = false
    await loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function onCopyAddress(row: AftersaleIssue) {
  const text = [row.buyerName, row.buyerPhone, row.buyerAddress].filter(Boolean).join(' ').trim()
  if (!text) {
    ElMessage.warning('暂无地址')
    return
  }
  const ok = await copyText(text)
  if (ok) ElMessage.success('地址已复制')
  else ElMessage.error('复制失败')
}

async function onHandleOne(row: AftersaleIssue) {
  const text = buildIssueHandleText(row)
  if (!text.trim()) {
    ElMessage.warning('暂无可复制内容')
    return
  }
  const ok = await copyText(text)
  if (ok) {
    ElMessage.success('已复制处理信息')
    if (row.status === 'pending') {
      try {
        await updateIssueStatus(row.id, 'processing')
        await loadData()
      } catch {
        /* ignore status bump failure */
      }
    }
  } else {
    ElMessage.error('复制失败')
  }
}

async function onHandleSelected() {
  if (!selected.value.length) {
    ElMessage.warning('请先勾选问题单')
    return
  }
  const types = new Set(selected.value.map((r) => r.problemTypeLabel || r.problemType))
  if (types.size > 1) {
    ElMessage.warning('请勾选同一问题类型后再合并复制')
    return
  }
  const text = buildMergedIssueHandleText(selected.value)
  const ok = await copyText(text)
  if (ok) {
    ElMessage.success(`已合并复制 ${selected.value.length} 条`)
    for (const row of selected.value) {
      if (row.status === 'pending') {
        try {
          await updateIssueStatus(row.id, 'processing')
        } catch {
          /* ignore */
        }
      }
    }
    await loadData()
  } else {
    ElMessage.error('复制失败')
  }
}

async function onStatusChange(row: AftersaleIssue, next: string) {
  try {
    await updateIssueStatus(row.id, next)
    ElMessage.success('状态已更新')
    await loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '更新失败')
  }
}

async function onDelete(row: AftersaleIssue) {
  try {
    await ElMessageBox.confirm(`确认删除问题单「${row.productTitle}」？`, '删除确认', { type: 'warning' })
    await deleteIssue(row.id)
    ElMessage.success('已删除')
    await loadData()
  } catch (e: any) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error(e.message || '删除失败')
  }
}

function statusTone(s: string) {
  if (s === 'pending') return 'warning'
  if (s === 'processing') return 'primary'
  if (s === 'done') return 'success'
  return 'info'
}

function onShopPick(id?: number) {
  const shop = shops.value.find((s) => s.id === id)
  if (shop) form.shopName = shop.name
}

onMounted(async () => {
  await Promise.all([loadShops(), loadMeta()])
  await loadData()
})

watch([status, problemType], () => handleSearch())
</script>

<template>
  <div v-loading="loading" class="issue-list">
    <div class="page-head">
      <div>
        <h2 class="page-title">售后问题记录</h2>
        <p class="desc">
          人工处理类问题单。可通过订单号/售后单号带入店铺、商品、地址；有售后单时同步售后状态与物流。侧栏气泡为近 7 天待处理数量。
        </p>
      </div>
      <div class="head-actions">
        <el-button type="primary" :icon="Plus" @click="openCreate">新建问题单</el-button>
        <el-button :disabled="!selected.length" @click="onHandleSelected">处理（合并复制）</el-button>
      </div>
    </div>

    <el-card>
      <div class="toolbar">
        <el-select v-model="shopId" clearable placeholder="全部店铺" style="width: 160px" @change="handleSearch">
          <el-option v-for="shop in shops" :key="shop.id" :label="shop.name" :value="shop.id" />
        </el-select>
        <el-select v-model="status" clearable placeholder="全部状态" style="width: 130px">
          <el-option v-for="s in meta.statuses" :key="s.value" :label="s.label" :value="s.value" />
        </el-select>
        <el-select v-model="problemType" clearable placeholder="问题类型" style="width: 150px">
          <el-option v-for="p in meta.problemTypes" :key="p.value" :label="p.label" :value="p.value" />
        </el-select>
        <el-input
          v-model="keyword"
          clearable
          placeholder="订单号 / 售后单号 / 商品 / 买家"
          style="width: 280px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <span class="total">共 {{ total }} 条</span>
      </div>

      <el-table :data="list" stripe border @selection-change="onSelectionChange">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="shopName" label="店铺" width="120" />
        <el-table-column label="商品 / 单号" min-width="240">
          <template #default="{ row }">
            <div class="product">
              <img v-if="row.productImage" class="thumb" :src="row.productImage" alt="" />
              <div class="product-meta">
                <div class="title">{{ row.productTitle || '—' }}</div>
                <div v-if="row.skuSpecs" class="spec">{{ row.skuSpecs }}</div>
                <div class="sub">订单 {{ row.orderNo || '—' }}</div>
                <div v-if="row.platformAftersaleId" class="sub">售后 {{ row.platformAftersaleId }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="问题" min-width="160">
          <template #default="{ row }">
            <div>{{ row.problemTypeLabel }}</div>
            <div v-if="row.problemNote" class="sub">{{ row.problemNote }}</div>
          </template>
        </el-table-column>
        <el-table-column label="处理方式" min-width="140">
          <template #default="{ row }">
            <div>{{ row.handleMethodLabel || '—' }}</div>
            <div v-if="row.handleNote" class="sub">{{ row.handleNote }}</div>
          </template>
        </el-table-column>
        <el-table-column label="买家地址" min-width="200">
          <template #default="{ row }">
            <div>{{ row.buyerName || '' }} {{ row.buyerPhone || '' }}</div>
            <div class="sub">{{ row.buyerAddress || '—' }}</div>
            <el-button
              v-if="row.buyerAddress || row.buyerPhone"
              link
              type="primary"
              size="small"
              :icon="CopyDocument"
              @click="onCopyAddress(row)"
            >复制地址</el-button>
          </template>
        </el-table-column>
        <el-table-column label="售后信息" min-width="180">
          <template #default="{ row }">
            <template v-if="row.hasAftersale">
              <div>{{ row.aftersaleStatus || '—' }}</div>
              <div v-if="row.aftersaleType" class="sub">{{ row.aftersaleType }}</div>
              <div v-if="row.logistics" class="sub">{{ row.logistics }}</div>
              <div v-if="row.returnLogisticsNo" class="sub">退货 {{ row.returnLogisticsNo }}</div>
              <div v-if="row.shipLogisticsNo" class="sub">发货 {{ row.shipLogisticsNo }}</div>
            </template>
            <span v-else class="muted">无售后单</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-select
              :model-value="row.status"
              size="small"
              style="width: 100px"
              @change="(v: string) => onStatusChange(row, v)"
            >
              <el-option
                v-for="s in meta.statuses"
                :key="s.value"
                :label="s.label"
                :value="s.value"
              />
            </el-select>
            <el-tag class="status-tag" :type="statusTone(row.status)" size="small">{{ row.statusLabel }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="160" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="onHandleOne(row)">处理</el-button>
            <el-button link @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="onDelete(row)">删除</el-button>
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

    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '编辑问题单' : '新建问题单'"
      width="760px"
      destroy-on-close
    >
      <div class="lookup-bar">
        <el-input
          v-model="lookupQ"
          clearable
          placeholder="输入订单号或售后单号检索"
          @keyup.enter="doLookup"
        />
        <el-button type="primary" :loading="lookingUp" @click="doLookup">检索</el-button>
      </div>
      <div v-if="lookupHits.length > 1" class="lookup-hits">
        <button
          v-for="(hit, idx) in lookupHits"
          :key="idx"
          type="button"
          class="hit"
          @click="applyLookup(hit)"
        >
          <div class="title">{{ hit.shopName }} · {{ hit.productTitle || '—' }}</div>
          <div class="sub">
            订单 {{ hit.orderNo || '—' }}
            <template v-if="hit.platformAftersaleId"> · 售后 {{ hit.platformAftersaleId }}</template>
            · {{ hit.hasAftersale ? hit.aftersaleStatus || '有售后' : '仅订单' }}
          </div>
        </button>
      </div>

      <el-form label-width="100px" class="issue-form">
        <el-form-item label="店铺" required>
          <el-select
            v-model="form.shopId"
            filterable
            clearable
            placeholder="选择店铺"
            style="width: 100%"
            @change="onShopPick"
          >
            <el-option v-for="shop in shops" :key="shop.id" :label="shop.name" :value="shop.id" />
          </el-select>
          <el-input v-model="form.shopName" placeholder="店铺名称" style="margin-top: 8px" />
        </el-form-item>
        <el-form-item label="商品" required>
          <el-input v-model="form.productTitle" placeholder="商品标题" />
          <el-input v-model="form.skuSpecs" placeholder="商品规格" style="margin-top: 8px" />
        </el-form-item>
        <el-form-item label="单号">
          <div class="row2">
            <el-input v-model="form.orderNo" placeholder="订单号" />
            <el-input v-model="form.platformAftersaleId" placeholder="售后单号" />
          </div>
        </el-form-item>
        <el-form-item label="买家地址">
          <div class="row2">
            <el-input v-model="form.buyerName" placeholder="姓名" />
            <el-input v-model="form.buyerPhone" placeholder="手机" />
          </div>
          <el-input
            v-model="form.buyerAddress"
            type="textarea"
            :rows="2"
            placeholder="完整收货地址"
            style="margin-top: 8px"
          />
        </el-form-item>
        <el-form-item v-if="form.hasAftersale" label="售后信息">
          <div class="row2">
            <el-input v-model="form.aftersaleStatus" placeholder="售后状态" />
            <el-input v-model="form.aftersaleType" placeholder="售后类型" />
          </div>
          <el-input v-model="form.logistics" placeholder="物流信息" style="margin-top: 8px" />
          <div class="row2" style="margin-top: 8px">
            <el-input v-model="form.returnLogisticsNo" placeholder="退货单号" />
            <el-input v-model="form.shipLogisticsNo" placeholder="发货单号" />
          </div>
        </el-form-item>
        <el-form-item label="问题类型" required>
          <el-select v-model="form.problemType" placeholder="选择类型" style="width: 100%">
            <el-option v-for="p in meta.problemTypes" :key="p.value" :label="p.label" :value="p.value" />
          </el-select>
          <el-input
            v-if="isCustomProblem"
            v-model="form.problemTypeCustom"
            placeholder="自定义问题类型"
            style="margin-top: 8px"
          />
        </el-form-item>
        <el-form-item label="问题记录">
          <el-input v-model="form.problemNote" type="textarea" :rows="2" placeholder="问题说明" />
        </el-form-item>
        <el-form-item label="处理方式">
          <el-select v-model="form.handleMethod" clearable placeholder="选择处理方式" style="width: 100%">
            <el-option v-for="h in meta.handleMethods" :key="h.value" :label="h.label" :value="h.value" />
          </el-select>
          <el-input
            v-if="isCustomHandle"
            v-model="form.handleMethodCustom"
            placeholder="自定义处理方式"
            style="margin-top: 8px"
          />
        </el-form-item>
        <el-form-item label="处理说明">
          <el-input v-model="form.handleNote" type="textarea" :rows="2" placeholder="处理备注" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width: 160px">
            <el-option v-for="s in meta.statuses" :key="s.value" :label="s.label" :value="s.value" />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveIssue">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.page-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
}
.page-title { margin: 0 0 6px; font-size: 22px; }
.desc { color: #606266; margin: 0; line-height: 1.5; }
.head-actions { display: flex; gap: 8px; flex-shrink: 0; }
.toolbar { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; flex-wrap: wrap; }
.total { margin-left: auto; color: #909399; font-size: 13px; }
.product { display: flex; gap: 10px; align-items: flex-start; }
.thumb { width: 48px; height: 48px; border-radius: 4px; object-fit: cover; flex-shrink: 0; background: #f5f7fa; }
.product-meta { min-width: 0; }
.title { font-weight: 600; line-height: 1.4; }
.spec { color: #303133; font-size: 13px; margin-top: 4px; }
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
.muted { color: #c0c4cc; }
.status-tag { margin-top: 6px; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
.lookup-bar { display: flex; gap: 8px; margin-bottom: 12px; }
.lookup-hits { display: flex; flex-direction: column; gap: 8px; margin-bottom: 12px; }
.hit {
  text-align: left;
  border: 1px solid #ebeef5;
  background: #fff;
  border-radius: 8px;
  padding: 10px 12px;
  cursor: pointer;
  font: inherit;
}
.hit:hover { border-color: #409eff; background: #ecf5ff; }
.row2 { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; width: 100%; }
.issue-form :deep(.el-form-item) { margin-bottom: 14px; }
</style>
