<template>
  <div ref="rootRef" class="file-upload" @paste="handlePaste" tabindex="0">
    <!-- 单文件模式 -->
    <template v-if="!multiple && maxCount === 1">
      <div v-if="fileList.length === 0" class="upload-area">
        <el-upload
          drag
          :show-file-list="false"
          :http-request="handleUpload"
          :accept="accept"
          class="uploader"
        >
          <el-icon :size="24"><UploadFilled /></el-icon>
          <div class="upload-text">{{ placeholder }}</div>
          <div v-if="tip" class="upload-tip">{{ tip }}</div>
        </el-upload>
        <div class="material-picker-entry">
          <MaterialPicker :accept="accept" @select="selectMaterial" />
        </div>
      </div>
      <div v-else class="file-item single">
        <el-image v-if="isImage(fileList[0])" :src="fileList[0]" fit="cover" class="file-thumb" />
        <div v-else class="file-icon-wrap">
          <i :class="'iconfont icon-' + fileIcon" class="file-icon"></i>
          <span class="file-ext">{{ fileExt(fileList[0]) }}</span>
        </div>
        <div class="file-overlay">
          <el-button type="danger" :icon="Delete" size="small" circle @click="removeFile(0)" />
        </div>
      </div>
    </template>

    <!-- 多文件模式 -->
    <template v-else>
      <div v-if="fileList.length > 0" class="file-grid">
        <div v-for="(url, index) in fileList" :key="index" class="file-item">
          <el-image v-if="isImage(url)" :src="previewUrl(url)" fit="cover" class="file-thumb" />
          <div v-else class="file-icon-wrap">
            <i :class="'iconfont icon-' + getFileIcon(url)" class="file-icon"></i>
            <span class="file-ext">{{ fileTitle(url) }}</span>
          </div>
          <div class="file-overlay">
            <el-button type="danger" :icon="Delete" size="small" circle @click="removeFile(index)" />
          </div>
        </div>
        <div v-if="fileList.length < maxCount" class="file-item add-btn">
          <el-upload
            :show-file-list="false"
            :http-request="handleUpload"
            :accept="accept"
            multiple
            class="uploader"
          >
            <el-icon :size="24"><Plus /></el-icon>
            <span class="add-text">上传</span>
          </el-upload>
        </div>
        <div v-if="fileList.length < maxCount" class="file-item add-btn material-add-btn">
          <MaterialPicker :accept="accept" @select="selectMaterial" />
        </div>
      </div>
      <div v-else class="upload-area">
        <el-upload
          drag
          :show-file-list="false"
          :http-request="handleUpload"
          :accept="accept"
          multiple
          class="uploader"
        >
          <el-icon :size="24"><UploadFilled /></el-icon>
          <div class="upload-text">{{ placeholder }}</div>
          <div v-if="tip" class="upload-tip">{{ tip }}</div>
        </el-upload>
      </div>
    </template>

    <el-progress v-if="uploading" :percentage="uploadProgress" :stroke-width="3" class="upload-progress" />
  </div>
</template>

<script setup>
import MaterialPicker from '@/components/MaterialPicker.vue'
import { httpPost } from '@/utils/http'
import { replaceImg } from '@/utils/libs'
import { Delete, Plus, UploadFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { computed, onMounted, onUnmounted, ref } from 'vue'

const props = defineProps({
  modelValue: { type: [String, Array], default: '' },
  accept: { type: String, default: '*' },
  multiple: { type: Boolean, default: false },
  maxCount: { type: Number, default: 1 },
  maxSize: { type: Number, default: 50 },
  tip: { type: String, default: '' },
  placeholder: { type: String, default: '拖拽文件到此处，或点击上传' },
  previewMap: { type: Object, default: () => ({}) },
})

const emit = defineEmits(['update:modelValue', 'upload-success'])

const uploading = ref(false)
const uploadProgress = ref(0)
const rootRef = ref(null)

const fileList = computed({
  get() {
    if (props.multiple || props.maxCount > 1) {
      return Array.isArray(props.modelValue) ? props.modelValue : []
    }
    return props.modelValue ? [props.modelValue] : []
  },
  set(value) {
    if (props.multiple || props.maxCount > 1) {
      emit('update:modelValue', value)
    } else {
      emit('update:modelValue', value[0] || '')
    }
  },
})

const fileIcon = computed(() => getFileIconFromAccept(props.accept))

function getFileIconFromAccept(accept) {
  if (accept.includes('video')) return 'video'
  if (accept.includes('audio')) return 'mp3'
  return 'image'
}

function getFileIcon(url) {
  if (props.previewMap[url]?.preview_url) return 'image'
  const ext = url.split('?')[0].split('.').pop().toLowerCase()
  if (['mp4', 'webm', 'mov', 'avi', 'mkv'].includes(ext)) return 'video'
  if (['mp3', 'wav', 'ogg', 'flac', 'aac'].includes(ext)) return 'mp3'
  return 'image'
}

function isImage(url) {
  if (props.previewMap[url]?.preview_url) return true
  const ext = url.split('?')[0].split('.').pop().toLowerCase()
  return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg'].includes(ext)
}

function previewUrl(url) {
  return props.previewMap[url]?.preview_url || url
}

function fileTitle(url) {
  return props.previewMap[url]?.title || fileExt(url)
}

function fileExt(url) {
  return url.split('?')[0].split('.').pop().toUpperCase()
}

function addFileUrl(url) {
  const fileUrl = replaceImg(url)
  if (props.multiple || props.maxCount > 1) {
    if (fileList.value.length >= props.maxCount) {
      ElMessage.warning(`最多只能上传 ${props.maxCount} 个文件`)
      return
    }
    fileList.value = [...fileList.value, fileUrl]
  } else {
    fileList.value = [fileUrl]
  }
  emit('upload-success', fileUrl)
}

function selectMaterial(url) {
  addFileUrl(url)
}

function validateFile(file) {
  if (props.accept && props.accept !== '*') {
    const types = props.accept.split(',').map((t) => t.trim())
    const match = types.some((t) => {
      if (t.endsWith('/*')) return file.type.startsWith(t.replace('/*', '/'))
      return file.type === t || file.name.endsWith(t)
    })
    if (!match) {
      ElMessage.error(`不支持的文件格式，请上传 ${props.accept} 类型的文件`)
      return false
    }
  }
  if (file.size > props.maxSize * 1024 * 1024) {
    ElMessage.error(`文件大小不能超过 ${props.maxSize}MB`)
    return false
  }
  return true
}

async function doUpload(file) {
  const formData = new FormData()
  formData.append('file', file)
  const res = await httpPost('/api/upload', formData)
  return replaceImg(res.data.url)
}

async function handleUpload(options) {
  const file = options.file || options
  if (!validateFile(file)) {
    options.onError?.(new Error('validation failed'))
    return
  }

  uploading.value = true
  uploadProgress.value = 0
  const timer = setInterval(() => {
    if (uploadProgress.value < 90) uploadProgress.value += 10
  }, 100)

  try {
    const url = await doUpload(file)
    clearInterval(timer)
    uploadProgress.value = 100

    addFileUrl(url)
    ElMessage.success('上传成功')
    options.onSuccess?.({})
  } catch (e) {
    clearInterval(timer)
    ElMessage.error('上传失败: ' + (e.message || '网络错误'))
    options.onError?.(e)
  } finally {
    uploading.value = false
    uploadProgress.value = 0
  }
}

function handlePaste(e) {
  const files = Array.from(e.clipboardData?.files || [])
  if (files.length === 0) return
  e.preventDefault()
  e.stopImmediatePropagation()
  uploadFiles(files)
}

function handleWindowPaste(e) {
  if (!rootRef.value) return
  const activeElement = document.activeElement
  if (activeElement && ['INPUT', 'TEXTAREA'].includes(activeElement.tagName)) return
  if (!props.multiple && props.maxCount === 1 && fileList.value.length > 0) return
  handlePaste(e)
}

function uploadFiles(files) {
  const remaining = props.maxCount - fileList.value.length
  const selectedFiles = props.multiple || props.maxCount > 1 ? files.slice(0, Math.max(remaining, 0)) : files.slice(0, 1)
  if (selectedFiles.length === 0) {
    ElMessage.warning(`最多只能上传 ${props.maxCount} 个文件`)
    return
  }
  selectedFiles.forEach((file) => {
    handleUpload({ file, onSuccess: () => {}, onError: () => {} })
  })
}

onMounted(() => window.addEventListener('paste', handleWindowPaste))
onUnmounted(() => window.removeEventListener('paste', handleWindowPaste))

function removeFile(index) {
  const list = [...fileList.value]
  list.splice(index, 1)
  fileList.value = list
}
</script>

<style lang="scss" scoped>
.file-upload {
  width: 100%;
  outline: none;
}

.upload-area {
  :deep(.el-upload) {
    width: 100%;
  }
  :deep(.el-upload-dragger) {
    width: 100%;
    padding: 16px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
  }
}

.upload-text {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.upload-tip {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}

.file-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.file-item {
  width: 80px;
  height: 80px;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  position: relative;
  cursor: pointer;

  &.single {
    width: 100%;
    height: 120px;
  }

  &.add-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background: var(--el-fill-color-light);
    border-style: dashed;

    :deep(.el-upload) {
      width: 100%;
      height: 100%;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
    }
  }
}

.file-thumb {
  width: 100%;
  height: 100%;
}

.file-icon-wrap {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-light);
  gap: 4px;
}

.file-icon {
  font-size: 24px;
  color: var(--el-text-color-secondary);
}

.file-ext {
  font-size: 10px;
  color: var(--el-text-color-placeholder);
}

.file-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s;
}

.file-item:hover .file-overlay {
  opacity: 1;
}

.add-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
}

.material-picker-entry {
  display: flex;
  justify-content: center;
  margin-top: 8px;
}

.material-add-btn {
  :deep(.material-picker),
  :deep(.el-button) {
    width: 100%;
    height: 100%;
  }

  :deep(.el-button) {
    border: 0;
    white-space: normal;
    line-height: 1.3;
  }
}

.upload-progress {
  margin-top: 6px;
}
</style>
