<template>
  <div class="page-seedance">
    <!-- 左侧参数面板 -->
    <div class="params-panel">
      <!-- 模式选择 -->
      <div class="mode-buttons">
        <div class="mode-grid">
          <div
            v-for="mode in store.modes"
            :key="mode.key"
            :class="['mode-btn', { active: store.activeMode === mode.key }]"
            @click="store.switchMode(mode.key)"
          >
            <i :class="'iconfont icon-' + mode.icon"></i>
            <span>{{ mode.name }}</span>
          </div>
        </div>
      </div>

      <!-- 参数区域 -->
      <div class="params-container">
        <!-- 模型选择 -->
        <div class="param-line pt">
          <span class="label">模型：</span>
        </div>
        <div class="param-line">
          <el-radio-group v-model="currentModel" size="small">
            <el-radio-button value="fast">快速</el-radio-button>
            <el-radio-button value="standard">标准</el-radio-button>
          </el-radio-group>
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

        <!-- 图生视频-首帧：上传图片 -->
        <template v-if="store.activeMode === 'image_to_video_first'">
          <div class="param-line pt"><span class="label">首帧图片：</span></div>
          <div class="param-line">
            <el-input v-model="store.imageToVideoFirstParams.first_frame_url" placeholder="输入图片URL" />
          </div>
        </template>

        <!-- 首尾帧模式 -->
        <template v-if="store.activeMode === 'image_to_video_dual'">
          <div class="param-line pt"><span class="label">首帧图片：</span></div>
          <div class="param-line">
            <el-input v-model="store.imageToVideoDualParams.first_frame_url" placeholder="输入首帧图片URL" />
          </div>
          <div class="param-line pt"><span class="label">尾帧图片：</span></div>
          <div class="param-line">
            <el-input v-model="store.imageToVideoDualParams.last_frame_url" placeholder="输入尾帧图片URL" />
          </div>
        </template>

        <!-- 多模态参考 -->
        <template v-if="store.activeMode === 'multimodal_ref'">
          <div class="param-line pt"><span class="label">参考图片：</span></div>
          <div class="param-line">
            <div class="upload-row">
              <el-upload :show-file-list="false" accept="image/*" :http-request="(o) => uploadTo(o, store.multimodalRefParams.image_urls)" multiple>
                <el-button size="small">上传图片</el-button>
              </el-upload>
              <el-input v-model="tempUrls.multimodalImage" size="small" placeholder="或输入URL" @keyup.enter="addUrlTo('multimodalImage', store.multimodalRefParams.image_urls)" />
              <el-button size="small" @click="addUrlTo('multimodalImage', store.multimodalRefParams.image_urls)">添加</el-button>
            </div>
            <div class="url-list" v-if="store.multimodalRefParams.image_urls.length">
              <div v-for="(url, i) in store.multimodalRefParams.image_urls" :key="'mi'+i" class="url-item">
                <el-image :src="store.replaceImg(url)" fit="cover" class="url-thumb" />
                <span class="url-name">图片{{ i + 1 }}</span>
                <el-button size="small" text type="danger" @click="store.multimodalRefParams.image_urls.splice(i, 1)">
                  <i class="iconfont icon-delete"></i>
                </el-button>
              </div>
            </div>
          </div>
          <div class="param-line pt"><span class="label">参考视频：</span></div>
          <div class="param-line">
            <div class="upload-row">
              <el-upload :show-file-list="false" accept="video/*" :http-request="(o) => uploadTo(o, store.multimodalRefParams.video_urls)" multiple>
                <el-button size="small">上传视频</el-button>
              </el-upload>
              <el-input v-model="tempUrls.multimodalVideo" size="small" placeholder="或输入URL" @keyup.enter="addUrlTo('multimodalVideo', store.multimodalRefParams.video_urls)" />
              <el-button size="small" @click="addUrlTo('multimodalVideo', store.multimodalRefParams.video_urls)">添加</el-button>
            </div>
            <div class="url-list" v-if="store.multimodalRefParams.video_urls.length">
              <div v-for="(url, i) in store.multimodalRefParams.video_urls" :key="'mv'+i" class="url-item">
                <i class="iconfont icon-video url-icon"></i>
                <span class="url-name">视频{{ i + 1 }}</span>
                <el-button size="small" text type="danger" @click="store.multimodalRefParams.video_urls.splice(i, 1)">
                  <i class="iconfont icon-delete"></i>
                </el-button>
              </div>
            </div>
          </div>
          <div class="param-line pt"><span class="label">参考音频：</span></div>
          <div class="param-line">
            <div class="upload-row">
              <el-upload :show-file-list="false" accept="audio/*" :http-request="(o) => uploadTo(o, store.multimodalRefParams.audio_urls)" multiple>
                <el-button size="small">上传音频</el-button>
              </el-upload>
              <el-input v-model="tempUrls.multimodalAudio" size="small" placeholder="或输入URL" @keyup.enter="addUrlTo('multimodalAudio', store.multimodalRefParams.audio_urls)" />
              <el-button size="small" @click="addUrlTo('multimodalAudio', store.multimodalRefParams.audio_urls)">添加</el-button>
            </div>
            <div class="url-list" v-if="store.multimodalRefParams.audio_urls.length">
              <div v-for="(url, i) in store.multimodalRefParams.audio_urls" :key="'ma'+i" class="url-item">
                <i class="iconfont icon-mp3 url-icon"></i>
                <span class="url-name">音频{{ i + 1 }}</span>
                <el-button size="small" text type="danger" @click="store.multimodalRefParams.audio_urls.splice(i, 1)">
                  <i class="iconfont icon-delete"></i>
                </el-button>
              </div>
            </div>
          </div>
        </template>

        <!-- 编辑视频 -->
        <template v-if="store.activeMode === 'edit_video'">
          <div class="param-line pt"><span class="label">参考视频URL：</span></div>
          <div class="param-line">
            <el-input v-model="store.editVideoParams.ref_video_url" placeholder="输入参考视频URL" />
          </div>
          <div class="param-line pt"><span class="label">参考图片URL：</span></div>
          <div class="param-line">
            <el-input v-model="store.editVideoParams.ref_image_url" placeholder="输入参考图片URL" />
          </div>
        </template>

        <!-- 延长视频 -->
        <template v-if="store.activeMode === 'extend_video'">
          <div class="param-line pt"><span class="label">参考视频：</span></div>
          <div class="param-line">
            <div class="upload-row">
              <el-upload :show-file-list="false" accept="video/*" :http-request="(o) => uploadTo(o, store.extendVideoParams.video_urls)" multiple>
                <el-button size="small">上传视频</el-button>
              </el-upload>
              <el-input v-model="tempUrls.extendVideo" size="small" placeholder="或输入URL" @keyup.enter="addUrlTo('extendVideo', store.extendVideoParams.video_urls)" />
              <el-button size="small" @click="addUrlTo('extendVideo', store.extendVideoParams.video_urls)">添加</el-button>
            </div>
            <div class="url-list" v-if="store.extendVideoParams.video_urls.length">
              <div v-for="(url, i) in store.extendVideoParams.video_urls" :key="'ev'+i" class="url-item">
                <i class="iconfont icon-video url-icon"></i>
                <span class="url-name">视频{{ i + 1 }}</span>
                <el-button size="small" text type="danger" @click="store.extendVideoParams.video_urls.splice(i, 1)">
                  <i class="iconfont icon-delete"></i>
                </el-button>
              </div>
            </div>
          </div>
        </template>

        <!-- 虚拟人像 -->
        <template v-if="store.activeMode === 'virtual_avatar'">
          <div class="param-line pt"><span class="label">虚拟人像 Asset ID：</span></div>
          <div class="param-line">
            <el-input v-model="store.virtualAvatarParams.asset_id" placeholder="asset-xxxxxxxxx-xxxxx" />
          </div>
        </template>

        <!-- 通用参数 -->
        <div class="param-line pt"><span class="label">分辨率：</span></div>
        <div class="param-line">
          <el-select v-model="currentResolution" placeholder="选择分辨率">
            <el-option v-for="opt in store.resolutionOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </div>

        <div class="param-line pt"><span class="label">宽高比：</span></div>
        <div class="param-line">
          <el-select v-model="currentRatio" placeholder="选择宽高比">
            <el-option v-for="opt in store.ratioOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </div>

        <div class="param-line pt"><span class="label">时长：</span></div>
        <div class="param-line">
          <el-select v-model="currentDuration" placeholder="选择时长">
            <el-option v-for="opt in store.durationOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </div>

        <div class="param-line">
          <el-switch v-model="currentGenerateAudio" active-text="生成音频" inactive-text="无声" />
        </div>

        <div class="param-line">
          <el-switch v-model="currentWatermark" active-text="水印" inactive-text="无水印" />
        </div>
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
            <el-image
              v-if="item.cover_url"
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
              <span>失败</span>
            </div>
          </div>
          <div class="task-info">
            <div class="task-meta">
              <el-tag size="small" type="primary">{{ store.getModeName(item.type) }}</el-tag>
              <el-tag
                size="small"
                :type="item.status === 'succeeded' ? 'success' : item.status === 'failed' ? 'danger' : 'warning'"
              >
                {{ store.getStatusText(item.status) }}
              </el-tag>
            </div>
            <div class="task-prompt">{{ store.substr(item.prompt, 60) }}</div>
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
                <i class="iconfont icon-delete"></i>
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
import { computed, onMounted, onUnmounted, reactive } from 'vue'
import { useSeedanceStore } from '@/store/seedance'
import { Loading } from '@element-plus/icons-vue'

const store = useSeedanceStore()

const filterLabels = { all: '全部', processing: '进行中', succeeded: '已完成', failed: '失败' }

// 代理当前模式的参数到响应式
const currentModel = computed({
  get: () => {
    const p = getParams()
    return p?.model || 'fast'
  },
  set: (v) => { getParams().model = v },
})
const currentResolution = computed({
  get: () => getParams()?.resolution || '720p',
  set: (v) => { getParams().resolution = v },
})
const currentRatio = computed({
  get: () => getParams()?.ratio || '16:9',
  set: (v) => { getParams().ratio = v },
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
    default: return store.textToVideoParams
  }
}

const tempUrls = reactive({
  multimodalImage: '',
  multimodalVideo: '',
  multimodalAudio: '',
  extendVideo: '',
})

async function uploadTo(options, targetArray) {
  const url = await store.uploadFile(options.file)
  if (url) {
    targetArray.push(url)
    options.onSuccess({})
  } else {
    options.onError(new Error('上传失败'))
  }
}

function addUrlTo(key, targetArray) {
  const url = tempUrls[key].trim()
  if (url) {
    targetArray.push(url)
    tempUrls[key] = ''
  }
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
  .cover-img {
    width: 100%;
    height: 100%;
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

.upload-row {
  display: flex;
  gap: 6px;
  align-items: center;
  margin-bottom: 6px;
  .el-input {
    flex: 1;
  }
}
.url-list {
  margin-top: 4px;
}
.url-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  margin-bottom: 4px;
}
.url-thumb {
  width: 32px;
  height: 32px;
  border-radius: 4px;
  flex-shrink: 0;
}
.url-icon {
  font-size: 18px;
  color: var(--el-text-color-regular);
}
.url-name {
  flex: 1;
  font-size: 12px;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
