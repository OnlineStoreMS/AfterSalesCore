<script setup lang="ts">
import { useRouter } from 'vue-router'
import { VideoCamera, Search } from '@element-plus/icons-vue'

const router = useRouter()
</script>

<template>
  <div class="dashboard">
    <el-card>
      <template #header>售后中心</template>
      <p class="desc">管理退货包裹开箱视频：录制、识别快递单号、拍摄问题凭证，并按单号检索。</p>
      <div class="actions">
        <el-button type="primary" size="large" :icon="VideoCamera" @click="router.push('/unboxing/create')">
          开始录制开箱
        </el-button>
        <el-button size="large" :icon="Search" @click="router.push('/unboxing')">
          按快递单号查询
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 720px;
}
.desc {
  color: #606266;
  margin: 0 0 20px;
  line-height: 1.6;
}
.actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
</style>
