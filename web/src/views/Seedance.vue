<template>
  <div class="page-seedance">
    <!-- 左侧参数面板 -->
    <div class="params-panel">
      <!-- 参数区域 -->
      <div class="params-container">
        <!-- 模型选择 -->
        <div class="param-line pt">
          <span class="label">模型：</span>
        </div>
        <div class="param-line">
          <el-select v-model="store.selectedModel" placeholder="选择模型" @change="onModelChange">
            <el-option v-for="model in store.videoModels" :key="model.value" :label="model.label" :value="model.value" />
          </el-select>
        </div>

        <!-- 提示词（非虚拟人像必须） -->
        <div v-if="store.activeMode !== 'image_to_video_first'" class="param-line pt">
          <span class="label">提示词：</span>
        </div>
        <div v-if="store.activeMode !== 'image_to_video_first'" class="param-line">
          <el-input
            v-model="store.currentPrompt"
            type="textarea"
            :autosize="{ minRows: 3, maxRows: 6 }"
            placeholder="描述你想生成的视频画面..."
            maxlength="1000"
            show-word-limit
          />
        </div>

        <!-- 参考素材 -->
        <div class="param-line pt"><span class="label">参考素材：</span></div>
        <div class="param-line">
          <FileUpload
            v-if="store.isVeo"
            v-model="store.veoParams.images"
            accept="image/*"
            multiple
            :maxCount="2"
            placeholder="上传首帧/尾帧图片，不上传则为文生视频"
            tip="不上传图片：文生视频；上传首帧：图生视频；上传首帧和尾帧：首尾帧视频"
          />
          <FileUpload
            v-else
            v-model="store.multimodalRefParams.reference_urls"
            accept="image/*,video/*,audio/*"
            multiple
            :maxCount="9"
            placeholder="拖拽图片、视频或音频，或点击上传"
            tip="支持图片、视频、音频素材"
          />
        </div>

        <!-- 通用参数 -->
        <div class="param-line pt"><span class="label">分辨率：</span></div>
        <div class="param-line">
          <el-select v-model="currentResolution" placeholder="选择分辨率">
            <el-option v-for="opt in currentResolutionOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </div>

        <div class="param-line pt"><span class="label">宽高比：</span></div>
        <div class="param-line">
          <el-select v-model="currentRatio" placeholder="选择宽高比">
            <el-option v-for="opt in currentRatioOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </div>

        <div v-if="!store.isVeo" class="param-line pt"><span class="label">时长：</span></div>
        <div v-if="!store.isVeo" class="param-line">
          <el-select v-model="currentDuration" placeholder="选择时长">
            <el-option v-for="opt in store.durationOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </div>

        <template v-if="!store.isVeo">
          <div class="param-line">
            <el-switch v-model="currentGenerateAudio" active-text="生成音频" inactive-text="无声" />
          </div>

          <div class="param-line">
            <el-switch v-model="currentWatermark" active-text="水印" inactive-text="无水印" />
          </div>
        </template>
      </div>

      <!-- 提交按钮 -->
      <div class="submit-area">
        <el-button
          type="primary"
          :loading="store.submitting"
          @click="store.submitTask"
          class="submit-btn"
        >
          <span v-if="!store.submitting">生成视频</span>
          <span v-else>提交中...</span>
          <span class="power-cost">（{{ store.currentPowerCost }} 算力）</span>
        </el-button>
      </div>
    </div>

    <!-- 右侧作品列表 -->
    <div class="main-content">
      <div class="works-header">
        <h3>我的作品</h3>
        <div class="filter-btns">
          <el-button
            v-for="f in ['all', 'processing', 'succeeded', 'failed']"
            :key="f"
            :type="store.taskFilter === f ? 'primary' : ''"
            size="small"
            @click="store.taskFilter = f; store.fetchData(1)"
          >
            {{ filterLabels[f] }}
          </el-button>
        </div>
      </div>

      <div class="task-list" v-loading="store.loading">
        <div v-if="store.currentList.length === 0" class="empty-tip">
          <el-empty description="还没有作品，快去创建吧" />
        </div>
        <div
          v-for="item in store.currentList"
          :key="item.id"
          class="task-item"
        >
          <div class="task-cover" @click="item.video_url && store.playVideo(item)">
            <video
              v-if="item.status === 'succeeded' && item.video_url"
              :src="store.replaceImg(item.video_url)"
              class="cover-video"
              muted
              autoplay
              loop
              playsinline
            />
            <el-image
              v-else-if="item.cover_url"
              :src="item.cover_url"
              fit="cover"
              class="cover-img"
            />
            <div v-else class="cover-placeholder">
              <i class="iconfont icon-video"></i>
            </div>
            <div v-if="item.status === 'queued' || item.status === 'running'" class="status-overlay">
              <el-icon class="is-loading"><Loading /></el-icon>
              <span>{{ store.getStatusText(item.status) }}</span>
            </div>
            <div v-if="item.status === 'failed'" class="status-overlay failed">
              <i class="iconfont icon-warning"></i>
              <span>{{ item.err_msg || '生成失败' }}</span>
            </div>
          </div>
          <div class="task-info">
            <div class="task-meta">
              <el-tag
                size="small"
                :type="item.status === 'succeeded' ? 'success' : item.status === 'failed' ? 'danger' : 'warning'"
              >
                {{ store.getStatusText(item.status) }}
              </el-tag>
            </div>
            <div class="task-prompt">{{ store.substr(item.prompt, 60) }}</div>
            <div v-if="item.status === 'failed' && item.err_msg" class="task-error">{{ item.err_msg }}</div>
            <div class="task-actions">
              <el-button v-if="item.video_url" size="small" @click="store.playVideo(item)" text>
                <i class="iconfont icon-play"></i>
              </el-button>
              <el-button v-if="item.video_url" size="small" @click="store.downloadFile(item)" text :loading="item.downloading">
                <i class="iconfont icon-download"></i>
              </el-button>
              <el-button v-if="item.status === 'failed'" size="small" @click="store.retryTask(item.id)" text>
                <i class="iconfont icon-retry"></i>
              </el-button>
              <el-button size="small" @click="store.removeJob(item)" text type="danger">
                <i class="iconfont icon-remove"></i>
                <span>删除</span>
              </el-button>
            </div>
            <div class="task-time">
              {{ new Date(item.created_at * 1000).toLocaleString() }} | {{ item.power }} 算力
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 视频预览 -->
    <el-dialog v-model="store.showDialog" title="视频预览" width="800px" destroy-on-close>
      <video
        v-if="store.currentVideoUrl"
        :src="store.replaceImg(store.currentVideoUrl)"
        controls
        autoplay
        class="preview-video"
      />
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted } from 'vue'
import { useSeedanceStore } from '@/store/seedance'
import { Loading } from '@element-plus/icons-vue'
import FileUpload from '@/components/FileUpload.vue'

const store = useSeedanceStore()

const filterLabels = { all: '全部', processing: '进行中', succeeded: '已完成', failed: '失败' }

const currentResolutionOptions = computed(() => store.isVeo ? store.veoResolutionOptions : store.resolutionOptions)
const currentRatioOptions = computed(() => store.isVeo ? store.veoRatioOptions : store.ratioOptions)

const currentResolution = computed({
  get: () => store.isVeo ? store.veoParams.resolution : getParams()?.resolution || '720p',
  set: (v) => {
    if (store.isVeo) {
      store.veoParams.resolution = v
      return
    }
    getParams().resolution = v
  },
})
const currentRatio = computed({
  get: () => store.isVeo ? store.veoParams.aspect_ratio : getParams()?.ratio || '16:9',
  set: (v) => {
    if (store.isVeo) {
      store.veoParams.aspect_ratio = v
      return
    }
    getParams().ratio = v
  },
})
const currentDuration = computed({
  get: () => getParams()?.duration || 5,
  set: (v) => { getParams().duration = v },
})
const currentGenerateAudio = computed({
  get: () => getParams()?.generate_audio ?? true,
  set: (v) => { getParams().generate_audio = v },
})
const currentWatermark = computed({
  get: () => getParams()?.watermark ?? false,
  set: (v) => { getParams().watermark = v },
})

function getParams() {
  switch (store.activeMode) {
    case 'text_to_video': return store.textToVideoParams
    case 'image_to_video_first': return store.imageToVideoFirstParams
    case 'image_to_video_dual': return store.imageToVideoDualParams
    case 'multimodal_ref': return store.multimodalRefParams
    case 'edit_video': return store.editVideoParams
    case 'extend_video': return store.extendVideoParams
    case 'virtual_avatar': return store.virtualAvatarParams
    default: return store.multimodalRefParams
  }
}

function onModelChange(value) {
  const model = store.videoModels.find((item) => item.value === value)
  if (!model) return
  if (model.provider === 'veo') {
    store.veoParams.model = model.model
    store.fetchData(1)
    return
  }
  store.multimodalRefParams.model = model.model
  store.fetchData(1)
}

onMounted(() => store.init())
onUnmounted(() => store.cleanup())
</script>

<style lang="scss" scoped>
.page-seedance {
  display: flex;
  min-height: 100vh;
  background: var(--el-bg-color-page);
}

.params-panel {
  width: 340px;
  min-width: 340px;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color);
  padding: 16px;
  overflow-y: auto;
  height: 100vh;
  position: sticky;
  top: 0;
}

.mode-buttons {
  margin-bottom: 16px;
}
.mode-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 6px;
}
.mode-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 4px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 12px;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  transition: all 0.2s;
  i {
    font-size: 18px;
    margin-bottom: 4px;
  }
  &:hover {
    background: var(--el-color-primary-light-9);
  }
  &.active {
    background: var(--el-color-primary-light-8);
    color: var(--el-color-primary);
  }
}

.params-container {
  margin-bottom: 16px;
}
.param-line {
  margin-bottom: 10px;
  &.pt {
    padding-top: 8px;
  }
  .label {
    font-size: 13px;
    color: var(--el-text-color-regular);
    font-weight: 500;
  }
  .el-select, .el-input {
    width: 100%;
  }
}

.submit-area {
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.submit-btn {
  width: 100%;
}
.power-cost {
  font-size: 12px;
  margin-left: 4px;
  opacity: 0.8;
}

.main-content {
  flex: 1;
  padding: 16px 20px;
  overflow-y: auto;
}
.works-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  h3 {
    margin: 0;
    font-size: 18px;
  }
}
.filter-btns {
  display: flex;
  gap: 6px;
}

.task-list {
  min-height: 200px;
}
.empty-tip {
  padding: 60px 0;
}
.task-item {
  display: flex;
  gap: 12px;
  padding: 12px;
  background: var(--el-bg-color);
  border-radius: 8px;
  margin-bottom: 10px;
  border: 1px solid var(--el-border-color-lighter);
}
.task-cover {
  width: 160px;
  height: 90px;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  position: relative;
  flex-shrink: 0;
  background: var(--el-fill-color);
  .cover-video,
  .cover-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  .cover-placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    i {
      font-size: 28px;
      color: var(--el-text-color-placeholder);
    }
  }
  .status-overlay {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0,0,0,0.5);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: #fff;
    font-size: 12px;
    gap: 4px;
    &.failed {
      background: rgba(255,0,0,0.3);
    }
  }
}
.task-info {
  flex: 1;
  min-width: 0;
}
.task-meta {
  display: flex;
  gap: 6px;
  margin-bottom: 6px;
}
.task-prompt {
  font-size: 13px;
  color: var(--el-text-color-regular);
  margin-bottom: 8px;
  line-height: 1.4;
}
.task-error {
  font-size: 12px;
  color: var(--el-color-danger);
  margin-bottom: 6px;
  line-height: 1.4;
}
.task-actions {
  display: flex;
  gap: 2px;
  margin-bottom: 4px;
}
.task-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}

.preview-video {
  width: 100%;
  max-height: 70vh;
}

@media (max-width: 768px) {
  .page-seedance {
    flex-direction: column;
  }
  .params-panel {
    width: 100%;
    min-width: auto;
    height: auto;
    position: relative;
  }
}
</style>
