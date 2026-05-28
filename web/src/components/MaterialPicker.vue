<template>
  <div class="material-picker">
    <el-button size="small" plain @click="openPicker">从素材库选择</el-button>
    <el-dialog v-model="visible" title="选择素材" width="720px" :close-on-click-modal="true">
      <div v-loading="loading" class="material-picker-body">
        <el-empty v-if="!loading && materials.length === 0" description="暂无可复用素材" />
        <div v-else class="material-grid">
          <button
            v-for="item in materials"
            :key="item.id || item.url"
            type="button"
            class="material-item"
            @click="selectMaterial(item)"
          >
            <el-image v-if="isImage(item.ext || item.url)" :src="displayUrl(item.url)" fit="cover" />
            <div v-else-if="isVideo(item.ext || item.url)" class="material-video">
              <video :src="displayUrl(item.url)" muted preload="metadata" />
              <span>视频素材</span>
            </div>
            <div v-else-if="isAudio(item.ext || item.url)" class="material-file">
              <span>音频素材</span>
            </div>
            <div v-else class="material-file">
              <span>{{ fileExt(item.ext || item.url) || '文件' }}</span>
            </div>
            <span class="material-name">{{ item.name || item.url }}</span>
          </button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { httpPost } from '@/utils/http'
import { replaceImg } from '@/utils/libs'
import { ElMessage } from 'element-plus'
import { computed, ref } from 'vue'

const props = defineProps({
  accept: {
    type: String,
    default: '',
  },
  limit: {
    type: Number,
    default: 30,
  },
})

const emit = defineEmits(['select'])
const visible = ref(false)
const loading = ref(false)
const items = ref([])

const materials = computed(() => {
  const types = acceptTypes(props.accept)
  if (types.length === 0) return items.value

  return items.value.filter((item) => {
    const value = item.ext || item.url
    return types.some((type) => matchType(value, type))
  })
})

const acceptTypes = (accept) => {
  const normalized = (accept || '').trim().toLowerCase()
  if (!normalized || normalized === '*') return []

  return normalized
    .split(',')
    .map((type) => type.trim())
    .map((type) => {
      if (type === 'image' || type.startsWith('image/')) return 'image'
      if (type === 'video' || type.startsWith('video/')) return 'video'
      if (type === 'audio' || type.startsWith('audio/')) return 'audio'
      if (type.startsWith('.')) return type.slice(1)
      return type
    })
    .filter(Boolean)
}

const matchType = (value, type) => {
  if (type === 'image') return isImage(value)
  if (type === 'video') return isVideo(value)
  if (type === 'audio') return isAudio(value)
  return fileExt(value) === type
}

const openPicker = async () => {
  visible.value = true
  loading.value = true
  try {
    const res = await httpPost('/api/upload/recent', { limit: props.limit })
    items.value = res.data || []
  } catch (error) {
    ElMessage.error('获取素材失败: ' + (error.message || '网络错误'))
  } finally {
    loading.value = false
  }
}

const selectMaterial = (item) => {
  visible.value = false
  emit('select', item.url)
}

const displayUrl = (url) => replaceImg(url)
const fileExt = (value) => {
  const ext = (value || '').split('?')[0].split('.').pop()
  return ext ? ext.toLowerCase() : ''
}
const isImage = (value) => ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp'].includes(fileExt(value))
const isVideo = (value) => ['mp4', 'webm', 'mov', 'm4v'].includes(fileExt(value))
const isAudio = (value) => ['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a'].includes(fileExt(value))
</script>

<style lang="scss" scoped>
.material-picker {
  display: inline-flex;
}

.material-picker-body {
  min-height: 220px;
}

.material-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 12px;
  max-height: 60vh;
  overflow-y: auto;
}

.material-item {
  border: 1px solid #dcdfe6;
  border-radius: 8px;
  background: var(--el-bg-color);
  padding: 6px;
  cursor: pointer;
  text-align: left;

  &:hover {
    border-color: var(--el-color-primary);
  }

  .el-image,
  video,
  .material-file {
    width: 100%;
    height: 96px;
    border-radius: 6px;
    overflow: hidden;
    background: var(--el-fill-color-light);
  }
}

.material-video,
.material-file {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.material-video video {
  object-fit: cover;
}

.material-video span {
  position: absolute;
  left: 6px;
  bottom: 6px;
  padding: 2px 6px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
}

.material-name {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
