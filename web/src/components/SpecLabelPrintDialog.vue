<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Printer } from '@element-plus/icons-vue'
import {
  LABEL_SIZE_PRESETS,
  loadSavedLabelSize,
  printSpecLabels,
  resolveSpecText,
  saveLabelSize,
  type LabelSize,
  type SpecLabelItem,
} from '../utils/specLabelPrint'

const props = defineProps<{
  modelValue: boolean
  items: SpecLabelItem[]
}>()

const emit = defineEmits<{
  'update:modelValue': [boolean]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const size = ref<LabelSize>(loadSavedLabelSize())
const presetId = ref('')
const copies = ref(1)
const customW = ref(40)
const customH = ref(50)

const previewItems = computed(() =>
  props.items.map((it) => ({
    ...it,
    copies: Math.min(Math.max(Number(copies.value) || 1, 1), 99),
  })),
)

const totalSheets = computed(() =>
  previewItems.value.reduce((n, it) => n + (Math.min(Math.max(Number(it.copies) || 1, 1), 99)), 0),
)

function matchPreset(s: LabelSize) {
  const hit = LABEL_SIZE_PRESETS.find(
    (p) => p.widthMm === s.widthMm && p.heightMm === s.heightMm,
  )
  presetId.value = hit?.id || 'custom'
  customW.value = s.widthMm
  customH.value = s.heightMm
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    const saved = loadSavedLabelSize()
    size.value = { ...saved }
    matchPreset(saved)
    const firstCopies = props.items[0]?.copies
    copies.value = firstCopies && firstCopies > 0 ? Math.min(firstCopies, 99) : 1
  },
)

function onPresetChange(id: string) {
  if (id === 'custom') {
    size.value = {
      widthMm: Math.max(10, Number(customW.value) || 40),
      heightMm: Math.max(10, Number(customH.value) || 30),
    }
    return
  }
  const p = LABEL_SIZE_PRESETS.find((x) => x.id === id)
  if (!p) return
  size.value = { widthMm: p.widthMm, heightMm: p.heightMm }
  customW.value = p.widthMm
  customH.value = p.heightMm
}

function onCustomChange() {
  presetId.value = 'custom'
  size.value = {
    widthMm: Math.max(10, Math.min(200, Number(customW.value) || 40)),
    heightMm: Math.max(10, Math.min(300, Number(customH.value) || 30)),
  }
}

const previewScale = computed(() => {
  // 屏幕预览：把 mm 映射到 px，大标签缩小
  const maxEdge = Math.max(size.value.widthMm, size.value.heightMm)
  if (maxEdge >= 100) return 2.2
  if (maxEdge >= 70) return 2.8
  return 3.4
})

function doPrint() {
  if (!props.items.length) {
    ElMessage.warning('没有可打印的订单')
    return
  }
  const missing = props.items.filter((it) => !String(it.orderNo || '').trim())
  if (missing.length) {
    ElMessage.warning('存在缺少订单号的记录，请检查后再打')
    return
  }
  saveLabelSize(size.value)
  const ok = printSpecLabels(previewItems.value, size.value)
  if (!ok) {
    ElMessage.error('无法打开打印窗口，请允许浏览器弹窗后重试')
    return
  }
  ElMessage.success(`已调起打印（${totalSheets.value} 张）`)
  visible.value = false
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="打印商品规格标签"
    width="720px"
    destroy-on-close
    append-to-body
  >
    <div class="toolbar">
      <span class="field-label">标签尺寸</span>
      <el-select v-model="presetId" style="width: 160px" @change="onPresetChange">
        <el-option
          v-for="p in LABEL_SIZE_PRESETS"
          :key="p.id"
          :label="p.label"
          :value="p.id"
        />
        <el-option label="自定义" value="custom" />
      </el-select>
      <template v-if="presetId === 'custom'">
        <el-input-number
          v-model="customW"
          :min="10"
          :max="200"
          :step="1"
          controls-position="right"
          style="width: 110px"
          @change="onCustomChange"
        />
        <span>×</span>
        <el-input-number
          v-model="customH"
          :min="10"
          :max="300"
          :step="1"
          controls-position="right"
          style="width: 110px"
          @change="onCustomChange"
        />
        <span class="unit">mm</span>
      </template>
      <span class="field-label">每单份数</span>
      <el-input-number v-model="copies" :min="1" :max="99" controls-position="right" style="width: 110px" />
      <span class="hint">共 {{ items.length }} 单 · {{ totalSheets }} 张 · {{ size.widthMm }}×{{ size.heightMm }} mm</span>
    </div>

    <p class="tip">
      打印前按标签纸尺寸排版：主文案为商品规格，底部为订单号便于追溯。实际出纸尺寸由浏览器「更多设置 → 纸张尺寸」或标签打印机驱动决定，请选择与预设一致的纸张。
    </p>

    <div class="preview-board">
      <div
        v-for="(it, idx) in items.slice(0, 6)"
        :key="`${it.orderNo}-${idx}`"
        class="preview-label"
        :style="{
          width: `${size.widthMm * previewScale}px`,
          height: `${size.heightMm * previewScale}px`,
        }"
      >
        <div class="preview-inner">
          <div class="preview-spec">{{ resolveSpecText(it) }}</div>
          <div class="preview-order">
            <span>订单</span>
            <span class="mono">{{ it.orderNo || '—' }}</span>
          </div>
        </div>
      </div>
      <div v-if="items.length > 6" class="more">… 另有 {{ items.length - 6 }} 单</div>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :icon="Printer" @click="doPrint">打印</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.field-label {
  color: #606266;
  font-size: 13px;
  white-space: nowrap;
}
.unit { color: #909399; font-size: 13px; }
.hint { margin-left: auto; color: #909399; font-size: 12px; }
.tip {
  margin: 0 0 12px;
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}
.preview-board {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: flex-start;
  min-height: 120px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 8px;
}
.preview-label {
  background: #fff;
  border: 1px dashed #c0c4cc;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}
.preview-inner {
  width: 100%;
  height: 100%;
  padding: 6%;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  box-sizing: border-box;
}
.preview-spec {
  font-weight: 700;
  font-size: 13px;
  line-height: 1.3;
  word-break: break-word;
  overflow: hidden;
  flex: 1;
}
.preview-order {
  display: flex;
  gap: 6px;
  align-items: baseline;
  border-top: 1px solid #303133;
  padding-top: 4px;
  font-size: 11px;
  font-weight: 600;
}
.preview-order .mono {
  font-family: ui-monospace, Menlo, Consolas, monospace;
  word-break: break-all;
}
.more {
  align-self: center;
  color: #909399;
  font-size: 13px;
}
</style>
