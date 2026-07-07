<script setup lang="ts">
import type { EdgeRecordGoods } from '../api/edgeRecord'

defineProps<{
  goods?: EdgeRecordGoods[]
}>()
</script>

<template>
  <div v-if="goods?.length" class="goods-list">
    <div v-for="(g, idx) in goods" :key="idx" class="goods-line">
      <div class="goods-row">
        <el-image
          v-if="g.picUrl"
          :src="g.picUrl"
          :preview-src-list="[g.picUrl]"
          fit="cover"
          class="goods-pic"
          preview-teleported
        />
        <div class="goods-info">
          <div class="goods-title">{{ g.title || '—' }}</div>
          <div v-if="g.skuName" class="goods-sku muted">{{ g.skuName }}<span v-if="g.num"> x{{ g.num }}</span></div>
        </div>
      </div>
    </div>
  </div>
  <span v-else class="muted">—</span>
</template>

<style scoped>
.goods-line + .goods-line {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed #ebeef5;
}
.goods-row {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}
.goods-pic {
  width: 56px;
  height: 56px;
  flex-shrink: 0;
  border-radius: 4px;
  border: 1px solid #ebeef5;
}
.goods-info {
  min-width: 0;
  line-height: 1.5;
}
.goods-title {
  font-size: 13px;
  word-break: break-word;
}
.goods-sku {
  font-size: 12px;
}
.muted {
  color: #909399;
}
</style>
