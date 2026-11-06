<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import TicketLogisticsCell from '../../components/TicketLogisticsCell.vue'
import {
  fetchShopTicketsByKind,
  fetchShops,
  markShopTicketHandled,
  transferShopTicketToIssue,
  type AftersaleTicket,
  type MarketplaceShop,
  type ShopTicketKind,
} from '../../api/shop'
import { fetchIssueMeta, type IssueMeta } from '../../api/issue'
import { dateRangeDefaultTime, dateShortcuts } from '../../utils/date'
import { parseTicketLogistics, signedTimeFromTracks, displayTrackDetail } from '../../utils/ticketLogistics'

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
const reason = ref('')
const reasons = ref<string[]>([])
const status = ref('')
const statuses = ref<string[]>([])
const handleStatus = ref('')
const signedRange = ref<[string, string] | null>(null)
const issueMeta = ref<IssueMeta>({ problemTypes: [], handleMethods: [], statuses: [] })
const issueVisible = ref(false)
const issueSaving = ref(false)
const issueTicket = ref<AftersaleTicket | null>(null)
const issueForm = reactive({
  problemType: '',
  problemTypeCustom: '',
  handleMethod: '',
  handleMethodCustom: '',
  problemNote: '',
  handleNote: '',
})
const sortBy = ref('remainTime')
const sortOrder = ref<'asc' | 'desc'>('asc')
const sortReady = ref(false)
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
      reason: showReasonFilter.value ? reason.value || undefined : undefined,
      status: showStatusFilter.value ? status.value || undefined : undefined,
      handleStatus: isSignedKind.value ? handleStatus.value || undefined : undefined,
      signedFrom: isSignedKind.value ? signedRange.value?.[0] || undefined : undefined,
      signedTo: isSignedKind.value ? signedRange.value?.[1] || undefined : undefined,
      sortBy: isSignedKind.value ? sortBy.value : undefined,
      sortOrder: isSignedKind.value ? sortOrder.value : undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    tickets.value = data.list
    total.value = data.total
    if (data.reasons) reasons.value = data.reasons
    if (data.statuses) statuses.value = data.statuses
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

function onSortChange(payload: { prop?: string; order?: string | null }) {
  if (payload.prop === 'signedTime') {
    sortBy.value = 'signedTime'
    sortOrder.value = payload.order === 'ascending' ? 'asc' : 'desc'
  } else {
    sortBy.value = 'remainTime'
    sortOrder.value = payload.order === 'descending' ? 'desc' : 'asc'
  }
  if (!sortReady.value) return
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

function signedTimeOf(row: AftersaleTicket) {
  return signedTimeFromTracks(row.tracks, row.signedTime)
}

function pickupPointOf(row: AftersaleTicket) {
  const direct = String(row.pickupPoint || '').trim()
  if (direct) return direct
  const track = row.tracks?.[0]
  if (!track) return ''
  const line = [track.date, track.title, displayTrackDetail(track)].filter(Boolean).join(' ').trim()
  return line || String(track.text || '').trim()
}

const handleStatusOptions = [
  { value: '', label: '全部' },
  { value: 'pending', label: '未处理' },
  { value: 'handled', label: '已处理' },
  { value: 'issue', label: '已转问题记录' },
]
const isCustomProblem = computed(() => issueForm.problemType === 'custom')
const isCustomHandle = computed(() => issueForm.handleMethod === 'custom')

function handleStatusTone(s?: string) {
  if (s === 'handled') return 'success'
  if (s === 'issue') return 'primary'
  return 'warning'
}

async function onMarkHandled(row: AftersaleTicket, handled: boolean) {
  try {
    await ElMessageBox.confirm(
      handled ? `将售后单 ${row.platformAftersaleId} 标记为已处理？` : `取消售后单 ${row.platformAftersaleId} 的已处理标记？`,
      handled ? '标记已处理' : '取消已处理',
      { type: handled ? 'success' : 'warning' },
    )
    await markShopTicketHandled(row.id, handled)
    ElMessage.success(handled ? '已标记处理' : '已取消标记')
    await loadData()
  } catch (e: any) {
    if (e === 'cancel' || e === 'close') return
    ElMessage.error(e.message || '操作失败')
  }
}

function openTransfer(row: AftersaleTicket) {
  issueTicket.value = row
  issueForm.problemType = ''
  issueForm.problemTypeCustom = ''
  issueForm.handleMethod = ''
  issueForm.handleMethodCustom = ''
  issueForm.problemNote = ''
  issueForm.handleNote = ''
  issueVisible.value = true
}

async function confirmTransfer() {
  const row = issueTicket.value
  if (!row) return
  if (!issueForm.problemType) {
    ElMessage.warning('请选择问题类型')
    return
  }
  if (issueForm.problemType === 'custom' && !issueForm.problemTypeCustom.trim()) {
    ElMessage.warning('请填写自定义问题类型')
    return
  }
  if (issueForm.handleMethod === 'custom' && !issueForm.handleMethodCustom.trim()) {
    ElMessage.warning('请填写自定义处理方式')
    return
  }
  issueSaving.value = true
  try {
    await transferShopTicketToIssue(row.id, {
      shopId: row.shopId,
      shopName: row.shopName,
      orderNo: row.orderNo,
      platformAftersaleId: row.platformAftersaleId,
      productTitle: row.productTitle,
      productImage: row.productImage,
      skuSpecs: row.sku,
      aftersaleStatus: row.status,
      aftersaleType: row.aftersaleType,
      logistics: row.logistics,
      returnLogisticsNo: row.returnLogisticsNo,
      shipLogisticsNo: row.shipLogisticsNo,
      hasAftersale: true,
      problemType: issueForm.problemType,
      problemTypeCustom: issueForm.problemTypeCustom.trim() || undefined,
      handleMethod: issueForm.handleMethod || undefined,
      handleMethodCustom: issueForm.handleMethodCustom.trim() || undefined,
      problemNote: issueForm.problemNote.trim() || undefined,
      handleNote: issueForm.handleNote.trim() || undefined,
      status: 'pending',
    })
    ElMessage.success('已转到售后问题记录')
    issueVisible.value = false
    await loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '转出失败')
  } finally {
    issueSaving.value = false
  }
}

const isPickupKind = computed(() => kind.value === 'buyer-return-pickup')
const showReasonFilter = computed(() =>
  kind.value === 'review-shipped-refund' ||
  kind.value === 'return-refund' ||
  kind.value === 'exchange',
)
const showStatusFilter = computed(() => kind.value === 'return-refund')
const isSignedKind = computed(() => kind.value === 'buyer-return-signed')
const showHandleFilter = computed(() => kind.value === 'buyer-return-signed')
const isDisputeKind = computed(() => kind.value === 'dispute')
const keywordPlaceholder = computed(() =>
  isPickupKind.value
    ? '售后编号 / 订单号 / 商品 / 退货单号 / 代收点'
    : '售后编号 / 订单号 / 商品 / 退货单号',
)

onMounted(async () => {
  loadShops()
  try {
    issueMeta.value = await fetchIssueMeta()
  } catch {
    /* ignore */
  }
  if (isSignedKind.value) handleStatus.value = 'pending'
  await loadData()
  sortReady.value = true
  tickTimer = window.setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)
})
onUnmounted(() => {
  if (tickTimer) window.clearInterval(tickTimer)
})
watch(kind, async () => {
  keyword.value = ''
  reason.value = ''
  reasons.value = []
  status.value = ''
  statuses.value = []
  handleStatus.value = kind.value === 'buyer-return-signed' ? 'pending' : ''
  signedRange.value = null
  sortBy.value = 'remainTime'
  sortOrder.value = 'asc'
  sortReady.value = false
  page.value = 1
  await loadData()
  await nextTick()
  sortReady.value = true
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
        <el-select
          v-if="showReasonFilter"
          v-model="reason"
          clearable
          filterable
          placeholder="申请原因"
          style="width: 200px"
          @change="handleSearch"
        >
          <el-option v-for="item in reasons" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select
          v-if="showStatusFilter"
          v-model="status"
          clearable
          filterable
          placeholder="售后状态"
          style="width: 200px"
          @change="handleSearch"
        >
          <el-option v-for="item in statuses" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select
          v-if="showHandleFilter"
          v-model="handleStatus"
          placeholder="处理状态"
          style="width: 160px"
          @change="handleSearch"
        >
          <el-option v-for="item in handleStatusOptions" :key="item.value || 'all'" :label="item.label" :value="item.value" />
        </el-select>
        <template v-if="isSignedKind">
          <span class="field-label">签收时间</span>
          <el-date-picker
            v-model="signedRange"
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
        </template>
        <el-input
          v-model="keyword"
          clearable
          :placeholder="keywordPlaceholder"
          style="width: 360px"
          @keyup.enter="handleSearch"
        />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <span class="total">共 {{ total }} 条</span>
      </div>

      <el-table
        :data="tickets"
        stripe
        border
        :default-sort="isSignedKind ? { prop: 'remainTime', order: 'ascending' } : undefined"
        @sort-change="onSortChange"
      >
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
        <el-table-column
          label="售后状态"
          min-width="180"
          :prop="isSignedKind ? 'remainTime' : undefined"
          :sortable="isSignedKind ? 'custom' : false"
        >
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
        <el-table-column v-if="isDisputeKind" prop="dispute" label="纠纷仲裁" width="140" />
        <el-table-column label="物流信息" min-width="220">
          <template #default="{ row }">
            <TicketLogisticsCell
              v-bind="parseTicketLogistics(row)"
              :tracks="row.tracks"
              :raw="row.logistics"
            />
          </template>
        </el-table-column>
        <el-table-column v-if="isPickupKind" label="代收点" min-width="220">
          <template #default="{ row }">
            <div class="location">{{ pickupPointOf(row) || '—' }}</div>
          </template>
        </el-table-column>
        <el-table-column
          v-if="isSignedKind"
          prop="signedTime"
          label="签收时间"
          width="170"
          sortable="custom"
        >
          <template #default="{ row }">{{ signedTimeOf(row) || '—' }}</template>
        </el-table-column>
        <el-table-column v-if="isSignedKind" label="处理状态" width="120">
          <template #default="{ row }">
            <el-tag :type="handleStatusTone(row.handleStatus)" size="small">
              {{ row.handleStatusLabel || '未处理' }}
            </el-tag>
            <div v-if="row.handledByName" class="sub">{{ row.handledByName }}</div>
          </template>
        </el-table-column>
        <el-table-column v-if="isSignedKind" label="操作" width="168" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.handleStatus !== 'handled' && row.handleStatus !== 'issue'"
              link
              type="primary"
              @click="onMarkHandled(row, true)"
            >标记已处理</el-button>
            <el-button
              v-else-if="row.handleStatus === 'handled'"
              link
              @click="onMarkHandled(row, false)"
            >取消处理</el-button>
            <el-button
              v-if="row.handleStatus !== 'issue'"
              link
              type="primary"
              @click="openTransfer(row)"
            >转问题记录</el-button>
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

    <el-dialog v-model="issueVisible" title="转到售后问题记录" width="560px" destroy-on-close>
      <p v-if="issueTicket" class="issue-preview">
        {{ issueTicket.shopName }} · {{ issueTicket.productTitle || '—' }}
        <span class="sub">售后 {{ issueTicket.platformAftersaleId }} / 订单 {{ issueTicket.orderNo || '—' }}</span>
      </p>
      <el-form label-width="96px">
        <el-form-item label="问题类型" required>
          <el-select v-model="issueForm.problemType" placeholder="选择类型" style="width: 100%">
            <el-option v-for="p in issueMeta.problemTypes" :key="p.value" :label="p.label" :value="p.value" />
          </el-select>
          <el-input
            v-if="isCustomProblem"
            v-model="issueForm.problemTypeCustom"
            placeholder="自定义问题类型"
            style="margin-top: 8px"
          />
        </el-form-item>
        <el-form-item label="问题记录">
          <el-input v-model="issueForm.problemNote" type="textarea" :rows="2" placeholder="问题说明" />
        </el-form-item>
        <el-form-item label="处理方式">
          <el-select v-model="issueForm.handleMethod" clearable placeholder="选择处理方式" style="width: 100%">
            <el-option v-for="h in issueMeta.handleMethods" :key="h.value" :label="h.label" :value="h.value" />
          </el-select>
          <el-input
            v-if="isCustomHandle"
            v-model="issueForm.handleMethodCustom"
            placeholder="自定义处理方式"
            style="margin-top: 8px"
          />
        </el-form-item>
        <el-form-item label="处理说明">
          <el-input v-model="issueForm.handleNote" type="textarea" :rows="2" placeholder="处理备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="issueVisible = false">取消</el-button>
        <el-button type="primary" :loading="issueSaving" @click="confirmTransfer">转入问题记录</el-button>
      </template>
    </el-dialog>
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
.tags { color: #69718c; font-size: 12px; margin-top: 4px; }
.sub { color: #909399; font-size: 12px; margin-top: 2px; }
.timeout { color: #e6a23c; font-size: 12px; margin-top: 2px; line-height: 1.4; }
.timeout.warning { color: #e6a23c; font-weight: 600; }
.timeout.danger { color: #f56c6c; font-weight: 700; }
.location { white-space: pre-wrap; line-height: 1.5; word-break: break-word; }
.pager { display: flex; justify-content: flex-end; margin-top: 16px; }
.issue-preview { margin: 0 0 12px; line-height: 1.5; }
</style>
