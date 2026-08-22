<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Document, Refresh, Warning } from '@element-plus/icons-vue'
import {
  fetchPluginDebugLog,
  fetchPluginDebugLogs,
  type PluginDebugLogItem,
  type PluginDebugLogRecord,
} from '../../api/pluginDebug'

const loading = ref(false)
const dir = ref('')
const list = ref<PluginDebugLogItem[]>([])
const current = ref<PluginDebugLogRecord | null>(null)
const currentName = ref('')

async function loadList() {
  loading.value = true
  try {
    const data = await fetchPluginDebugLogs()
    dir.value = data.dir
    list.value = data.list || []
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function openLog(row: PluginDebugLogItem) {
  currentName.value = row.name
  try {
    current.value = await fetchPluginDebugLog(row.name)
  } catch (e) {
    ElMessage.error((e as Error).message || '读取失败')
  }
}

onMounted(loadList)
</script>

<template>
  <div class="page">
    <div class="head">
      <div>
        <h2>插件诊断日志</h2>
        <p class="hint">临时目录，进程重启后会清空。仅诊断版插件在手动打开/同步时上报，不含定时任务。</p>
        <p v-if="dir" class="dir">{{ dir }}</p>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="loadList">刷新</el-button>
    </div>

    <el-alert
      type="warning"
      :closable="false"
      show-icon
      :icon="Warning"
      title="这是临时排障入口。正式插件请继续用「OSMS 抖店售后工作台」。"
      style="margin-bottom: 12px"
    />

    <el-table :data="list" v-loading="loading" stripe height="280" @row-click="openLog">
      <el-table-column prop="receivedAt" label="收到时间" width="180" />
      <el-table-column prop="kind" label="动作" width="90" />
      <el-table-column label="结果" width="80">
        <template #default="{ row }">
          <el-tag :type="row.ok ? 'success' : 'danger'" size="small">{{ row.ok ? '成功' : '失败' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="shopName" label="店铺" width="140" />
      <el-table-column prop="error" label="错误" min-width="180" show-overflow-tooltip />
      <el-table-column prop="durationMs" label="耗时ms" width="90" />
      <el-table-column prop="version" label="版本" width="100" />
    </el-table>

    <div v-if="current" class="detail">
      <h3>
        <el-icon><Document /></el-icon>
        {{ currentName }}
      </h3>
      <p class="meta">
        {{ current.kind }} · shop {{ current.shopId }} {{ current.shopName }} · {{ current.durationMs }}ms · v{{ current.version }}
      </p>
      <el-timeline>
        <el-timeline-item
          v-for="(ev, i) in current.events"
          :key="i"
          :timestamp="`${ev.ms}ms`"
          :type="ev.level === 'error' ? 'danger' : ev.level === 'warn' ? 'warning' : 'primary'"
        >
          <strong>{{ ev.step }}</strong>
          <pre v-if="ev.data != null">{{ JSON.stringify(ev.data, null, 2) }}</pre>
        </el-timeline-item>
      </el-timeline>
    </div>
  </div>
</template>

<style scoped>
.page { padding: 16px 20px 32px; }
.head { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; }
h2 { margin: 0 0 6px; font-size: 18px; }
.hint, .dir, .meta { color: #64748b; font-size: 13px; margin: 0 0 4px; }
.detail { margin-top: 20px; }
.detail h3 { display: flex; align-items: center; gap: 6px; font-size: 15px; }
pre {
  margin: 6px 0 0;
  padding: 8px 10px;
  background: #0f172a0d;
  border-radius: 8px;
  font-size: 12px;
  max-height: 240px;
  overflow: auto;
  white-space: pre-wrap;
}
</style>
