<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Camera, Delete, VideoCamera, VideoPause, VideoPlay } from '@element-plus/icons-vue'
import {
  completeUnboxingRecord,
  createUnboxingRecord,
  uploadUnboxingPhoto,
  uploadUnboxingVideo,
} from '../../api/unboxing'
import { captureVideoFrame, scanBarcodeFromVideo } from '../../utils/barcodeScan'
import { isValidTrackingNo, normalizeTrackingNo } from '../../utils/trackingNo'

interface CapturedPhoto {
  blob: Blob
  previewUrl: string
  issueRemark: string
}

const router = useRouter()
const step = ref(0)
const stepTitles = ['识别单号', '录制视频', '问题照片', '确认提交']

const videoRef = ref<HTMLVideoElement | null>(null)
const stream = ref<MediaStream | null>(null)
const cameraReady = ref(false)
const cameraError = ref('')

const trackingNo = ref('')
const scanning = ref(false)
const scanTimer = ref<number | null>(null)

const recording = ref(false)
const recordStartAt = ref(0)
const recordDurationSec = ref(0)
const mediaRecorder = ref<MediaRecorder | null>(null)
const recordedChunks = ref<Blob[]>([])
const videoBlob = ref<Blob | null>(null)
const videoPreviewUrl = ref('')

const photos = ref<CapturedPhoto[]>([])
const remark = ref('')
const submitting = ref(false)

const canNextFromTracking = computed(() => isValidTrackingNo(trackingNo.value))
const hasVideo = computed(() => !!videoBlob.value)
const recordDurationLabel = computed(() => {
  const sec = recordDurationSec.value
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
})

function cameraUnavailableMessage(): string {
  if (!window.isSecureContext) {
    return '摄像头需要在安全环境下使用，请通过 HTTPS 或 localhost 访问（局域网 IP 的 HTTP 地址不支持摄像头），也可改用手动输入快递单号。'
  }
  if (!navigator.mediaDevices?.getUserMedia) {
    return '当前浏览器不支持摄像头 API，请更换浏览器或改用手动输入快递单号。'
  }
  return '无法访问摄像头，请检查浏览器权限设置。'
}

async function initCamera() {
  cameraError.value = ''
  cameraReady.value = false
  if (!navigator.mediaDevices?.getUserMedia) {
    cameraError.value = cameraUnavailableMessage()
    return
  }
  try {
    stopCamera()
    const media = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: 'environment', width: { ideal: 1280 }, height: { ideal: 720 } },
      audio: true,
    })
    stream.value = media
    if (videoRef.value) {
      videoRef.value.srcObject = media
      await videoRef.value.play()
    }
    cameraReady.value = true
  } catch (e) {
    cameraError.value = (e as Error).message || '无法访问摄像头，请检查权限'
  }
}

function stopCamera() {
  if (scanTimer.value) {
    window.clearInterval(scanTimer.value)
    scanTimer.value = null
  }
  if (mediaRecorder.value && mediaRecorder.value.state !== 'inactive') {
    mediaRecorder.value.stop()
  }
  mediaRecorder.value = null
  stream.value?.getTracks().forEach((t) => t.stop())
  stream.value = null
  if (videoRef.value) {
    videoRef.value.srcObject = null
  }
}

function startScanLoop() {
  if (scanTimer.value || !cameraReady.value) return
  scanning.value = true
  scanTimer.value = window.setInterval(async () => {
    if (!videoRef.value || step.value !== 0) return
    const code = await scanBarcodeFromVideo(videoRef.value)
    if (code) {
      trackingNo.value = code
      stopScanLoop()
      ElMessage.success(`已识别单号：${code}`)
    }
  }, 800)
}

function stopScanLoop() {
  scanning.value = false
  if (scanTimer.value) {
    window.clearInterval(scanTimer.value)
    scanTimer.value = null
  }
}

function onTrackingInput() {
  trackingNo.value = normalizeTrackingNo(trackingNo.value)
}

function startRecording() {
  if (!stream.value) {
    ElMessage.warning('摄像头未就绪')
    return
  }
  recordedChunks.value = []
  revokeVideoPreview()
  const mime = MediaRecorder.isTypeSupported('video/webm;codecs=vp9')
    ? 'video/webm;codecs=vp9'
    : MediaRecorder.isTypeSupported('video/webm')
      ? 'video/webm'
      : ''
  const recorder = mime
    ? new MediaRecorder(stream.value, { mimeType: mime })
    : new MediaRecorder(stream.value)
  recorder.ondataavailable = (ev) => {
    if (ev.data.size > 0) recordedChunks.value.push(ev.data)
  }
  recorder.onstop = () => {
    const blob = new Blob(recordedChunks.value, { type: recorder.mimeType || 'video/webm' })
    videoBlob.value = blob
    videoPreviewUrl.value = URL.createObjectURL(blob)
    recordDurationSec.value = Math.max(1, Math.round((Date.now() - recordStartAt.value) / 1000))
    recording.value = false
  }
  mediaRecorder.value = recorder
  recordStartAt.value = Date.now()
  recorder.start(1000)
  recording.value = true
}

function stopRecording() {
  if (mediaRecorder.value && mediaRecorder.value.state !== 'inactive') {
    mediaRecorder.value.stop()
  }
}

function revokeVideoPreview() {
  if (videoPreviewUrl.value) {
    URL.revokeObjectURL(videoPreviewUrl.value)
    videoPreviewUrl.value = ''
  }
  videoBlob.value = null
}

function capturePhoto() {
  if (!videoRef.value || !cameraReady.value) {
    ElMessage.warning('摄像头未就绪')
    return
  }
  const canvas = captureVideoFrame(videoRef.value)
  canvas.toBlob((blob) => {
    if (!blob) return
    photos.value.push({
      blob,
      previewUrl: URL.createObjectURL(blob),
      issueRemark: '',
    })
    ElMessage.success('已拍摄照片')
  }, 'image/jpeg', 0.92)
}

function removePhoto(index: number) {
  const item = photos.value[index]
  if (item) URL.revokeObjectURL(item.previewUrl)
  photos.value.splice(index, 1)
}

async function goNext() {
  if (step.value === 0) {
    if (!canNextFromTracking.value) {
      ElMessage.warning('请输入或扫描有效的快递单号')
      return
    }
    stopScanLoop()
    step.value = 1
    return
  }
  if (step.value === 1) {
    if (!hasVideo.value) {
      ElMessage.warning('请先完成视频录制')
      return
    }
    step.value = 2
    return
  }
  if (step.value === 2) {
    step.value = 3
  }
}

function goPrev() {
  if (step.value === 0) {
    router.push('/unboxing')
    return
  }
  if (step.value === 1 && recording.value) {
    stopRecording()
  }
  step.value -= 1
  if (step.value === 0) startScanLoop()
}

async function handleSubmit() {
  if (!canNextFromTracking.value || !videoBlob.value) {
    ElMessage.warning('请完成单号识别与视频录制')
    return
  }
  submitting.value = true
  try {
    const record = await createUnboxingRecord({
      trackingNo: trackingNo.value,
      remark: remark.value || undefined,
    })
    const videoFile = new File([videoBlob.value], `unboxing-${trackingNo.value}.webm`, {
      type: videoBlob.value.type || 'video/webm',
    })
    await uploadUnboxingVideo(record.id, videoFile, recordDurationSec.value)
    for (const photo of photos.value) {
      const file = new File([photo.blob], `issue-${Date.now()}.jpg`, { type: 'image/jpeg' })
      await uploadUnboxingPhoto(record.id, file, photo.issueRemark || undefined)
    }
    await completeUnboxingRecord(record.id, {
      videoDurationSec: recordDurationSec.value,
      remark: remark.value || undefined,
    })
    ElMessage.success('开箱记录已提交')
    router.push(`/unboxing/${record.id}`)
  } catch (e) {
    ElMessage.error((e as Error).message || '提交失败')
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  await initCamera()
  startScanLoop()
})

onBeforeUnmount(() => {
  stopScanLoop()
  stopCamera()
  revokeVideoPreview()
  photos.value.forEach((p) => URL.revokeObjectURL(p.previewUrl))
})
</script>

<template>
  <div class="unboxing-create">
    <div class="top-bar">
      <el-button :icon="ArrowLeft" text @click="goPrev">返回</el-button>
      <span class="title">录制开箱</span>
    </div>

    <el-steps :active="step" finish-status="success" align-center class="steps">
      <el-step v-for="(t, i) in stepTitles" :key="i" :title="t" />
    </el-steps>

    <el-card>
      <div class="camera-section">
        <div class="camera-wrap">
          <video
            ref="videoRef"
            autoplay
            playsinline
            muted
            class="camera-preview"
            :class="{ hidden: step === 3 && videoPreviewUrl }"
          />
          <video
            v-if="step >= 1 && videoPreviewUrl"
            :src="videoPreviewUrl"
            controls
            playsinline
            class="camera-preview overlay"
          />
          <div v-if="!cameraReady && !cameraError" class="camera-placeholder">正在启动摄像头…</div>
          <div v-if="cameraError" class="camera-placeholder error">{{ cameraError }}</div>
        </div>

        <div v-if="step === 0" class="panel">
          <p class="hint">将快递条码对准摄像头自动识别，或手动输入单号。</p>
          <el-input
            v-model="trackingNo"
            placeholder="快递单号"
            size="large"
            clearable
            @input="onTrackingInput"
          />
          <div class="actions">
            <el-tag v-if="scanning" type="info">扫描中…</el-tag>
            <el-button v-if="!scanning" @click="startScanLoop">重新扫描</el-button>
            <el-button type="primary" :disabled="!canNextFromTracking" @click="goNext">下一步</el-button>
          </div>
        </div>

        <div v-else-if="step === 1" class="panel">
          <p class="hint">请全程录制开箱过程，确保包裹外观与内物清晰可见。</p>
          <div class="record-status">
            <el-tag :type="recording ? 'danger' : 'info'" size="large">
              {{ recording ? `录制中 ${recordDurationLabel}` : hasVideo ? `已录制 ${recordDurationLabel}` : '未开始' }}
            </el-tag>
          </div>
          <div class="actions">
            <el-button
              v-if="!recording"
              type="danger"
              :icon="VideoCamera"
              @click="startRecording"
            >
              {{ hasVideo ? '重新录制' : '开始录制' }}
            </el-button>
            <el-button v-else type="warning" :icon="VideoPause" @click="stopRecording">停止录制</el-button>
            <el-button type="primary" :disabled="!hasVideo || recording" @click="goNext">下一步</el-button>
          </div>
        </div>

        <div v-else-if="step === 2" class="panel">
          <p class="hint">如有货损、少件等问题，请拍摄凭证照片（可选）。</p>
          <div class="actions">
            <el-button type="primary" :icon="Camera" @click="capturePhoto">拍摄照片</el-button>
            <el-button type="primary" plain @click="goNext">跳过 / 下一步</el-button>
          </div>
          <div v-if="photos.length" class="photo-list">
            <div v-for="(photo, idx) in photos" :key="idx" class="photo-card">
              <img :src="photo.previewUrl" alt="问题照片" />
              <el-input
                v-model="photo.issueRemark"
                placeholder="问题说明（可选）"
                size="small"
              />
              <el-button type="danger" link :icon="Delete" @click="removePhoto(idx)">删除</el-button>
            </div>
          </div>
        </div>

        <div v-else class="panel">
          <el-descriptions :column="1" border>
            <el-descriptions-item label="快递单号">{{ trackingNo }}</el-descriptions-item>
            <el-descriptions-item label="视频时长">{{ recordDurationLabel }}</el-descriptions-item>
            <el-descriptions-item label="问题照片">{{ photos.length }} 张</el-descriptions-item>
          </el-descriptions>
          <el-form-item label="备注" class="remark-field">
            <el-input v-model="remark" type="textarea" :rows="2" placeholder="可选备注" />
          </el-form-item>
          <div class="actions">
            <el-button @click="goPrev">上一步</el-button>
            <el-button type="primary" :loading="submitting" :icon="VideoPlay" @click="handleSubmit">
              提交开箱记录
            </el-button>
          </div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.top-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}
.title {
  font-size: 18px;
  font-weight: 600;
}
.steps {
  margin-bottom: 20px;
}
.camera-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.camera-wrap {
  position: relative;
  background: #000;
  border-radius: 8px;
  overflow: hidden;
  max-width: 720px;
  aspect-ratio: 16 / 9;
}
.camera-preview {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.camera-preview.hidden {
  visibility: hidden;
  position: absolute;
  inset: 0;
}
.camera-preview.overlay {
  position: absolute;
  inset: 0;
  visibility: visible;
  background: #000;
}
.camera-placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  background: #1a1a1a;
}
.camera-placeholder.error {
  color: #f56c6c;
  padding: 16px;
  text-align: center;
}
.panel {
  max-width: 720px;
}
.hint {
  color: #606266;
  margin: 0 0 12px;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  margin-top: 12px;
}
.record-status {
  margin-bottom: 4px;
}
.photo-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 12px;
  margin-top: 16px;
}
.photo-card {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.photo-card img {
  width: 100%;
  height: 120px;
  object-fit: cover;
  border-radius: 4px;
}
.remark-field {
  margin-top: 16px;
}
</style>
