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
        <!-- 创作模式 -->
        <template v-if="!store.isVeo">
          <div class="param-line pt">
            <span class="label">创作模式：</span>
          </div>
          <div class="mode-buttons">
            <div class="mode-grid">
              <div
                v-for="mode in store.modes"
                :key="mode.key"
                :class="['mode-btn', { active: store.activeMode === mode.key }]"
                @click="switchSeedanceMode(mode.key)"
              >
                <i v-if="mode.icon" :class="['iconfont', `icon-${mode.icon}`]"></i>
                <span>{{ mode.name }}</span>
              </div>
            </div>
          </div>
        </template>

        <!-- 提示词（非虚拟人像必须） -->
        <div v-if="store.activeMode !== 'image_to_video_first'" class="param-line pt">
          <span class="label">提示词：</span>
        </div>
        <div v-if="store.activeMode !== 'image_to_video_first'" class="param-line">
          <div class="prompt-box">
            <el-input
              ref="promptInputRef"
              v-model="store.currentPrompt"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 20 }"
              placeholder="描述你想生成的视频画面..."
              maxlength="1000"
              show-word-limit
              @input="onPromptInput"
              @click="rememberPromptCursor"
              @keyup="rememberPromptCursor"
              @select="rememberPromptCursor"
              @blur="onPromptBlur"
            />
            <el-button
              v-if="!store.isVeo && store.activeMode === 'multimodal_ref'"
              class="mention-btn"
              text
              @mousedown.prevent.stop="toggleMentionPicker"
              @click.prevent.stop
            >@</el-button>
            <div v-if="store.activeMode === 'multimodal_ref' && showMentionPicker" class="mention-menu" @mousedown.prevent.stop @click.stop>
              <div v-if="mentionOptions.length === 0" class="mention-empty">还没创建主体</div>
              <button
                v-for="option in mentionOptions"
                :key="option.label"
                type="button"
                class="mention-option"
                @mousedown.prevent.stop
                @click.stop="insertMention(option.label)"
              >
                <span class="mention-preview">
                  <img v-if="option.type === 'image'" :src="option.url" alt="" />
                  <i v-else :class="['iconfont', option.type === 'video' ? 'icon-video' : 'icon-mp3']"></i>
                </span>
                <span class="mention-text">
                  <strong>{{ option.label }}</strong>
                  <small>{{ option.description }}</small>
                </span>
              </button>
            </div>
          </div>
        </div>

        <!-- 参考素材 -->
        <div class="param-line pt"><span class="label">参考素材：</span></div>
        <div class="param-line material-upload-line">
          <div class="material-upload-wrap">
            <FileUpload
              v-if="store.isVeo"
              v-model="store.veoParams.images"
              accept="image/*"
              multiple
              :maxCount="2"
              placeholder="上传首帧/尾帧图片，不上传则为文生视频"
              tip="不上传图片：文生视频；上传首帧：图生视频；上传首帧和尾帧：首尾帧视频"
              @picker-open="pauseVideoPreview" @picker-close="resumeVideoPreview"
            />
            <template v-else-if="store.activeMode === 'image_to_video_dual'">
              <FileUpload
                v-model="store.imageToVideoDualParams.first_frame_url"
                accept="image/*"
                placeholder="上传首帧图片"
                tip="首帧图片，必填"
                @picker-open="pauseVideoPreview" @picker-close="resumeVideoPreview"
              />
              <FileUpload
                v-model="store.imageToVideoDualParams.last_frame_url"
                accept="image/*"
                placeholder="上传尾帧图片"
                tip="尾帧图片，必填"
                @picker-open="pauseVideoPreview" @picker-close="resumeVideoPreview"
              />
            </template>
            <FileUpload
              v-else
              v-model="store.multimodalRefParams.reference_urls"
              accept="image/*,video/*,audio/*"
              multiple
              :maxCount="9"
              placeholder="拖拽图片、视频或音频，或点击上传"
              tip="支持图片、视频、音频素材"
              :previewMap="store.referenceAssetPreviews"
              @picker-open="pauseVideoPreview" @picker-close="resumeVideoPreview"
            />
          </div>
          <el-button
            v-if="!store.isVeo && store.activeMode === 'multimodal_ref'"
            class="portrait-picker-btn"
            :loading="store.portraitLoading"
            @click="store.openPortraitDialog"
          >
            选择虚拟人像
          </el-button>
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

    <el-dialog v-model="store.portraitDialogVisible" title="选择虚拟人像" width="860px" destroy-on-close>
      <el-tabs v-model="store.portraitActiveTab">
        <el-tab-pane label="人像库" name="library">
          <div class="portrait-filters">
            <el-select v-model="store.portraitFilters.gender" clearable placeholder="性别" @change="store.fetchPortraits(1)">
              <el-option label="女性" value="女性" />
              <el-option label="男性" value="男性" />
            </el-select>
            <el-input v-model="store.portraitFilters.country" clearable placeholder="国家，如：中国" @change="store.fetchPortraits(1)" />
            <el-input v-model="store.portraitFilters.occupation" clearable placeholder="职业，如：演员" @change="store.fetchPortraits(1)" />
          </div>
          <div v-loading="store.portraitLoading" class="portrait-grid">
            <button v-for="portrait in store.portraitList" :key="portrait.asset_id" type="button" class="portrait-card" @click="store.selectPortrait(portrait)">
              <img :src="portrait.preview_url" alt="" />
              <strong>{{ portrait.title }}</strong>
              <span>{{ portrait.metadata?.gender }} · {{ portrait.metadata?.age }}岁 · {{ portrait.metadata?.country }}</span>
            </button>
            <el-empty v-if="!store.portraitLoading && store.portraitList.length === 0" description="没有找到虚拟人像" />
          </div>
        </el-tab-pane>
        <el-tab-pane label="上传人像" name="upload">
          <el-upload
            drag
            :show-file-list="false"
            :http-request="uploadPortraitImage"
            accept="image/jpeg,image/png,image/webp,image/bmp,image/tiff,image/gif,image/heic,image/heif"
            class="portrait-upload"
          >
            <el-icon :size="28"><UploadFilled /></el-icon>
            <div class="upload-text">拖拽人像图片到此处，或点击上传</div>
            <div class="upload-tip">支持 jpeg、png、webp、bmp、tiff、gif、heic/heif，单张小于 30MB</div>
          </el-upload>
          <el-alert
            v-if="store.portraitUploadLoading"
            title="正在注册人像素材，请稍候"
            type="info"
            show-icon
            :closable="false"
            class="portrait-upload-alert"
          />
        </el-tab-pane>
      </el-tabs>
      <template #footer>
        <el-pagination
          v-if="store.portraitActiveTab === 'library'"
          layout="prev, pager, next"
          :current-page="store.portraitFilters.page"
          :page-size="store.portraitFilters.page_size"
          :total="store.portraitTotal"
          @current-change="store.fetchPortraits"
        />
      </template>
    </el-dialog>

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
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useSeedanceStore } from '@/store/seedance'
import { Loading, UploadFilled } from '@element-plus/icons-vue'
import { httpPost } from '@/utils/http'
import { replaceImg } from '@/utils/libs'
import FileUpload from '@/components/FileUpload.vue'
import { buildSeedanceMentionOptions } from '@/store/seedanceReferences'

const store = useSeedanceStore()
const promptInputRef = ref(null)
const showMentionPicker = ref(false)
const pausedVideoUrl = ref('')
const promptCursor = ref(0)

const filterLabels = { all: '全部', processing: '进行中', succeeded: '已完成', failed: '失败' }

const currentResolutionOptions = computed(() => store.isVeo ? store.veoResolutionOptions : store.resolutionOptions)
const currentRatioOptions = computed(() => store.isVeo ? store.veoRatioOptions : store.ratioOptions)
const mentionOptions = computed(() => buildSeedanceMentionOptions(store.multimodalRefParams.reference_urls || [], store.referenceAssetPreviews))

async function uploadPortraitImage(options) {
  const file = options.file
  const formData = new FormData()
  formData.append('file', file)
  try {
    const response = await httpPost('/api/upload', formData)
    const imageUrl = replaceImg(response.data.url)
    await store.registerUploadedPortrait(imageUrl, file.name.replace(/\.[^.]+$/, ''))
    options.onSuccess?.({})
  } catch (error) {
    options.onError?.(error)
  }
}

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

function getPromptTextarea() {
  return promptInputRef.value?.textarea
}

function rememberPromptCursor() {
  const textarea = getPromptTextarea()
  if (textarea) promptCursor.value = textarea.selectionStart ?? store.currentPrompt.length
}

function onPromptInput() {
  rememberPromptCursor()
  if (store.activeMode !== 'multimodal_ref') return
  const textarea = getPromptTextarea()
  const cursor = textarea?.selectionStart ?? store.currentPrompt.length
  if ((store.currentPrompt || '')[cursor - 1] === '@') showMentionPicker.value = true
}

function onPromptBlur() {
  rememberPromptCursor()
}

function toggleMentionPicker() {
  rememberPromptCursor()
  if (store.activeMode !== 'multimodal_ref') return
  showMentionPicker.value = true
}

function pauseVideoPreview() {
  if (!store.showDialog) return
  pausedVideoUrl.value = store.currentVideoUrl
  store.showDialog = false
  store.currentVideoUrl = ''
}

function resumeVideoPreview() {
  if (!pausedVideoUrl.value) return
  store.currentVideoUrl = pausedVideoUrl.value
  store.showDialog = true
  pausedVideoUrl.value = ''
}

function switchSeedanceMode(mode) {
  store.switchMode(mode)
  showMentionPicker.value = false
}

async function insertMention(label) {
  const cursor = promptCursor.value
  const prompt = store.currentPrompt || ''
  const start = cursor > 0 && prompt[cursor - 1] === '@' ? cursor - 1 : cursor
  const prefix = prompt.slice(0, start)
  const suffix = prompt.slice(cursor)
  store.currentPrompt = `${prefix}${label}${suffix}`
  showMentionPicker.value = false
  await nextTick()
  const nextCursor = prefix.length + label.length
  const textarea = getPromptTextarea()
  textarea?.focus()
  textarea?.setSelectionRange(nextCursor, nextCursor)
  promptCursor.value = nextCursor
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
  store.imageToVideoDualParams.model = model.model
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
  min-height: 74px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
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
.prompt-box {
  position: relative;
  .mention-btn {
    position: absolute;
    left: 8px;
    bottom: 8px;
    width: 24px;
    height: 24px;
    min-height: 24px;
    padding: 0;
    border-radius: 50%;
    font-weight: 600;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    z-index: 1;
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

.material-upload-line {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.material-upload-wrap {
  flex: 1;
  min-width: 0;
}

.portrait-picker-btn {
  flex: 0 0 auto;
}

.portrait-filters {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin-bottom: 14px;
}

.portrait-grid {
  min-height: 260px;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.portrait-card {
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  background: var(--el-bg-color);
  padding: 8px;
  text-align: left;
  cursor: pointer;

  img {
    width: 100%;
    aspect-ratio: 1;
    object-fit: cover;
    border-radius: 8px;
    margin-bottom: 6px;
  }

  strong,
  span {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  span {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    margin-top: 3px;
  }
}

.portrait-upload {
  width: 100%;
}

.portrait-upload :deep(.el-upload),
.portrait-upload :deep(.el-upload-dragger) {
  width: 100%;
}

.portrait-upload-alert {
  margin-top: 12px;
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

.mention-menu {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 6px);
  z-index: 20;
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 280px;
  overflow-y: auto;
  padding: 6px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color-overlay);
  box-shadow: var(--el-box-shadow-light);
}
.mention-empty {
  padding: 14px 8px;
  text-align: center;
  font-size: 13px;
  color: var(--el-text-color-placeholder);
}
.mention-option {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  border: 0;
  border-radius: 6px;
  padding: 8px 10px;
  background: transparent;
  color: var(--el-text-color-primary);
  cursor: pointer;
  font-size: 13px;
  text-align: left;
  &:hover {
    background: var(--el-fill-color-light);
  }
}
.mention-preview {
  flex-shrink: 0;
  width: 42px;
  height: 42px;
  border-radius: 6px;
  overflow: hidden;
  background: var(--el-fill-color-light);
  display: flex;
  align-items: center;
  justify-content: center;
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  i {
    font-size: 22px;
    color: var(--el-text-color-secondary);
  }
}
.mention-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  strong {
    font-weight: 600;
  }
  small {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
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
