<template>
  <div class="mobile-seedance">
    <div class="sticky-header">
      <van-icon name="arrow-left" @click="$router.back()" />
      <span class="title">Seedance视频</span>
      <span></span>
    </div>

    <!-- 参数区域 -->
    <div class="form-card">
      <!-- 模型选择 -->
      <div class="form-item">
        <span class="form-label">模型</span>
        <van-field
          v-model="selectedModelLabel"
          is-link
          readonly
          placeholder="请选择模型"
          @click="showModelPicker = true"
        />
        <van-popup v-model:show="showModelPicker" round position="bottom">
          <van-picker
            :columns="modelColumns"
            @cancel="showModelPicker = false"
            @confirm="onModelConfirm"
          />
        </van-popup>
      </div>
      <div v-if="!store.isVeo" class="form-item">
        <span class="form-label">创作模式</span>
        <div class="mode-tabs">
          <div
            v-for="mode in store.modes"
            :key="mode.key"
            :class="['mode-tab', { active: store.activeMode === mode.key }]"
            @click="switchSeedanceMode(mode.key)"
          >
            {{ mode.name }}
          </div>
        </div>
      </div>

      <!-- 提示词 -->
      <div class="form-item" v-if="store.activeMode !== 'image_to_video_first'">
        <div class="prompt-box">
          <van-field
            ref="promptFieldRef"
            v-model="store.currentPrompt"
            type="textarea"
            rows="3"
            placeholder="描述你想生成的视频画面..."
            maxlength="1000"
            show-word-limit
            @input="onPromptInput"
            @click="rememberPromptCursor"
            @keyup="rememberPromptCursor"
            @select="rememberPromptCursor"
            @blur="onPromptBlur"
          />
          <button v-if="!store.isVeo && store.activeMode === 'multimodal_ref'" type="button" class="mention-btn" @mousedown.prevent.stop="toggleMentionPicker" @click.prevent.stop>@</button>
        </div>
      </div>

      <div class="form-item">
        <span class="form-label">参考素材</span>
        <div class="material-upload-wrap">
          <FileUpload
            v-if="store.isVeo"
            v-model="store.veoParams.images"
            accept="image/*"
            multiple
            :maxCount="2"
            placeholder="上传首帧/尾帧图片，不上传则为文生视频"
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
            placeholder="点击上传图片、视频或音频"
            :previewMap="store.referenceAssetPreviews"
            @picker-open="pauseVideoPreview" @picker-close="resumeVideoPreview"
          />
        </div>
        <van-button
          v-if="!store.isVeo && store.activeMode === 'multimodal_ref'"
          plain
          type="primary"
          class="portrait-picker-btn"
          :loading="store.portraitLoading"
          @click="store.openPortraitDialog"
        >
          选择虚拟人像
        </van-button>
      </div>

      <!-- 通用参数 -->
      <div class="form-item">
        <span class="form-label">宽高比</span>
        <div class="ratio-btns">
          <div v-for="r in currentRatioOptions" :key="r.value" :class="['ratio-btn', { active: currentRatio === r.value }]" @click="currentRatio = r.value">
            {{ r.label }}
          </div>
        </div>
      </div>

      <div class="form-item" v-if="store.isVeo">
        <span class="form-label">分辨率</span>
        <div class="ratio-btns">
          <div v-for="r in store.veoResolutionOptions" :key="r.value" :class="['ratio-btn', { active: store.veoParams.resolution === r.value }]" @click="setVeoResolution(r.value)">
            {{ r.label }}
          </div>
        </div>
      </div>

      <div class="form-item" v-if="!store.isVeo">
        <van-field name="switch" label="生成音频">
          <template #input><van-switch v-model="getParams().generate_audio" size="20px" /></template>
        </van-field>
      </div>
    </div>

    <!-- 提交 -->
    <div class="submit-area">
      <van-button type="primary" block :loading="store.submitting" @click="store.submitTask">
        生成视频（{{ store.currentPowerCost }}算力）
      </van-button>
    </div>

    <!-- 作品列表 -->
    <div class="works-section">
      <h3>我的作品</h3>
      <van-list v-model:loading="store.listLoading" :finished="store.listFinished" @load="store.fetchData(store.page + 1)">
        <div v-for="item in store.currentList" :key="item.id" class="work-item">
          <div class="work-cover" @click="item.video_url && store.playVideo(item)">
            <img v-if="item.cover_url" :src="item.cover_url" />
            <div v-else class="cover-icon"><van-icon name="video-o" size="28" /></div>
            <div v-if="item.status === 'queued' || item.status === 'running'" class="work-loading">
              <van-loading size="20" color="#fff" />
            </div>
          </div>
          <div class="work-info">
            <div class="work-tags">
              <van-tag :type="item.status === 'succeeded' ? 'success' : item.status === 'failed' ? 'danger' : 'warning'" size="medium">
                {{ store.getStatusText(item.status) }}
              </van-tag>
            </div>
            <div class="work-prompt">{{ item.prompt?.substring(0, 50) }}</div>
            <div v-if="item.status === 'failed' && item.err_msg" class="work-error">{{ item.err_msg }}</div>
            <div class="work-actions">
              <van-icon v-if="item.video_url" name="play-circle-o" size="22" @click="store.playVideo(item)" />
              <van-icon v-if="item.status === 'failed'" name="replay" size="22" @click="store.retryTask(item.id)" />
              <van-icon name="delete-o" size="22" @click="store.removeJob(item)" />
            </div>
          </div>
        </div>
      </van-list>
      <van-empty v-if="store.currentList.length === 0 && !store.listLoading" description="还没有作品" />
    </div>

    <!-- 视频预览 -->
    <van-dialog v-model:show="store.showVideoDialog" title="视频预览" :show-confirm-button="false" close-on-click-overlay>
      <video v-if="store.currentVideoUrl" :src="store.currentVideoUrl" controls autoplay style="width: 100%" />
    </van-dialog>

    <van-popup v-model:show="store.portraitDialogVisible" round position="bottom" :style="{ height: '82%' }">
      <div class="portrait-sheet">
        <div class="portrait-title">选择虚拟人像</div>
        <van-tabs v-model:active="store.portraitActiveTab">
          <van-tab title="人像库" name="library">
            <div class="portrait-filters">
              <van-field v-model="store.portraitFilters.gender" placeholder="性别：女性/男性" @blur="store.fetchPortraits(1)" />
              <van-field v-model="store.portraitFilters.country" placeholder="国家，如：中国" @blur="store.fetchPortraits(1)" />
              <van-field v-model="store.portraitFilters.occupation" placeholder="职业，如：演员" @blur="store.fetchPortraits(1)" />
            </div>
            <van-loading v-if="store.portraitLoading" />
            <div v-else class="portrait-grid">
              <button v-for="portrait in store.portraitList" :key="portrait.asset_id" type="button" class="portrait-card" @click="store.selectPortrait(portrait)">
                <img :src="portrait.preview_url" alt="" />
                <strong>{{ portrait.title }}</strong>
                <span>{{ portrait.metadata?.gender }} · {{ portrait.metadata?.age }}岁 · {{ portrait.metadata?.country }}</span>
              </button>
              <van-empty v-if="store.portraitList.length === 0" description="没有找到虚拟人像" />
            </div>
          </van-tab>
          <van-tab title="上传人像" name="upload">
            <div class="portrait-upload-mobile">
              <van-uploader
                :after-read="uploadPortraitImage"
                accept="image/jpeg,image/png,image/webp,image/bmp,image/tiff,image/gif,image/heic,image/heif"
                :max-size="30 * 1024 * 1024"
              />
              <p>上传本地人像图片，系统会自动注册为 Seedance 素材。</p>
              <van-loading v-if="store.portraitUploadLoading">正在注册人像素材</van-loading>
            </div>
          </van-tab>
        </van-tabs>
      </div>
    </van-popup>

    <van-popup v-if="!store.isVeo && store.activeMode === 'multimodal_ref'" v-model:show="showMentionPicker" round position="bottom">
      <div class="mention-sheet">
        <div class="mention-title">选择参考素材</div>
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
            <van-icon v-else :name="option.type === 'video' ? 'video-o' : 'music-o'" size="22" />
          </span>
          <span class="mention-text">
            <strong>{{ option.label }}</strong>
            <small>{{ option.description }}</small>
          </span>
        </button>
      </div>
    </van-popup>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useSeedanceStore } from '@/store/mobile/seedance'
import FileUpload from '@/components/FileUpload.vue'
import { buildSeedanceMentionOptions } from '@/store/seedanceReferences'
import { httpPost } from '@/utils/http'
import { replaceImg } from '@/utils/libs'

const store = useSeedanceStore()
const showModelPicker = ref(false)
const showMentionPicker = ref(false)
const pausedVideoUrl = ref('')
const promptFieldRef = ref(null)
const promptCursor = ref(0)

const modelColumns = computed(() => store.videoModels.map((model) => ({ text: model.label, value: model.value })))
const selectedModelLabel = computed(() => store.currentModelConfig.label)

const currentRatioOptions = computed(() => store.isVeo ? store.veoRatioOptions : store.ratioOptions)
const mentionOptions = computed(() => buildSeedanceMentionOptions(store.multimodalRefParams.reference_urls || [], store.referenceAssetPreviews))

async function uploadPortraitImage(file) {
  const uploadFile = file.file || file
  const formData = new FormData()
  formData.append('file', uploadFile)
  const response = await httpPost('/api/upload', formData)
  const imageUrl = replaceImg(response.data.url)
  await store.registerUploadedPortrait(imageUrl, uploadFile.name.replace(/\.[^.]+$/, ''))
}
const currentRatio = computed({
  get: () => store.isVeo ? store.veoParams.aspect_ratio : getParams().ratio,
  set: (value) => {
    if (store.isVeo) {
      store.veoParams.aspect_ratio = value
    } else {
      getParams().ratio = value
    }
  },
})

function getPromptTextarea() {
  return promptFieldRef.value?.$el?.querySelector('textarea')
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
  if (!store.showVideoDialog) return
  pausedVideoUrl.value = store.currentVideoUrl
  store.showVideoDialog = false
  store.currentVideoUrl = ''
}

function resumeVideoPreview() {
  if (!pausedVideoUrl.value) return
  store.currentVideoUrl = pausedVideoUrl.value
  store.showVideoDialog = true
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

function onModelConfirm(option) {
  const value = option?.value ?? option?.selectedOptions?.[0]?.value
  const model = store.videoModels.find((item) => item.value === value)
  if (model) selectModel(model)
  showModelPicker.value = false
}

function selectModel(model) {
  store.selectedModel = model.value
  if (model.provider === 'veo') {
    store.veoParams.model = model.model
  } else {
    store.multimodalRefParams.model = model.model
    store.imageToVideoDualParams.model = model.model
  }
  store.fetchData(1)
}

function setVeoResolution(value) {
  store.veoParams.resolution = value
}

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

onMounted(() => store.init())
onUnmounted(() => store.cleanup())
</script>

<style scoped>
.mobile-seedance {
  min-height: 100vh;
  background: #f5f5f5;
  padding-bottom: 30px;
}
.sticky-header {
  position: sticky;
  top: 0;
  z-index: 100;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  border-bottom: 1px solid #eee;
}
.title { font-size: 16px; font-weight: 600; }

.mode-tabs {
  display: flex;
  overflow-x: auto;
  gap: 6px;
  padding: 12px 16px;
  background: #fff;
  border-bottom: 1px solid #eee;
}
.mode-tab {
  flex-shrink: 0;
  padding: 6px 14px;
  border-radius: 16px;
  font-size: 13px;
  background: #f0f0f0;
  color: #666;
}
.mode-tab.active { background: var(--van-primary-color); color: #fff; }

.form-card {
  margin: 12px 16px;
  background: #fff;
  border-radius: 12px;
  padding: 12px;
}
.form-item { margin-bottom: 12px; }
.form-label { font-size: 13px; color: #666; margin-bottom: 6px; display: block; }
.prompt-box {
  position: relative;
}
.mention-btn {
  position: absolute;
  left: 8px;
  bottom: 8px;
  z-index: 1;
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 50%;
  background: #ecf5ff;
  color: var(--van-primary-color);
  font-size: 15px;
  font-weight: 600;
  line-height: 24px;
  text-align: center;
}
.mention-sheet {
  padding: 14px 16px 18px;
}
.mention-title {
  margin-bottom: 10px;
  text-align: center;
  font-size: 15px;
  font-weight: 600;
}
.mention-empty {
  padding: 18px 0;
  text-align: center;
  font-size: 13px;
  color: #999;
}
.mention-option {
  display: flex;
  align-items: center;
  width: 100%;
  border: 0;
  border-radius: 8px;
  padding: 10px;
  background: #f7f8fa;
  color: #323233;
  font-size: 14px;
  text-align: left;
  margin-bottom: 8px;
  gap: 10px;
}
.mention-preview {
  flex-shrink: 0;
  width: 46px;
  height: 46px;
  border-radius: 8px;
  overflow: hidden;
  background: #ebedf0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #969799;
}
.mention-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.material-upload-wrap {
  margin-bottom: 8px;
}

.portrait-picker-btn {
  width: auto;
}
.portrait-sheet {
  padding: 14px;
  height: 100%;
  overflow-y: auto;
}
.portrait-title {
  font-weight: 600;
  margin-bottom: 10px;
}
.portrait-filters {
  display: grid;
  gap: 8px;
  margin-bottom: 12px;
}
.portrait-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 10px;
}
.portrait-card {
  border: 1px solid #eee;
  border-radius: 10px;
  background: #fff;
  padding: 8px;
  text-align: left;
}
.portrait-card img {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  border-radius: 8px;
}
.portrait-card strong,
.portrait-card span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.portrait-card span {
  color: #888;
  font-size: 12px;
  margin-top: 3px;
}
.portrait-upload-mobile {
  padding: 16px;
  text-align: center;
  color: var(--van-text-color-2);
}

.portrait-upload-mobile p {
  margin: 12px 0;
  font-size: 13px;
}
.mention-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.mention-text strong {
  font-weight: 600;
}
.mention-text small {
  color: #969799;
  font-size: 12px;
}

.model-btns, .ratio-btns { display: flex; gap: 8px; }
.model-btn, .ratio-btn {
  flex: 1;
  text-align: center;
  padding: 8px;
  border-radius: 8px;
  background: #f5f5f5;
  font-size: 13px;
}
.model-btn.active, .ratio-btn.active { background: var(--van-primary-color-light); color: var(--van-primary-color); }

.submit-area { padding: 0 16px; margin-bottom: 16px; }

.works-section { padding: 0 16px; }
.works-section h3 { font-size: 16px; margin-bottom: 12px; }

.work-item {
  display: flex;
  gap: 10px;
  background: #fff;
  border-radius: 10px;
  padding: 10px;
  margin-bottom: 10px;
}
.work-cover {
  width: 100px;
  height: 56px;
  border-radius: 6px;
  overflow: hidden;
  flex-shrink: 0;
  background: #eee;
  position: relative;
  img { width: 100%; height: 100%; object-fit: cover; }
  .cover-icon { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; }
  .work-loading { position: absolute; inset: 0; background: rgba(0,0,0,0.4); display: flex; align-items: center; justify-content: center; }
}
.work-info { flex: 1; min-width: 0; }
.work-tags { display: flex; gap: 4px; margin-bottom: 4px; }
.work-prompt { font-size: 12px; color: #666; margin-bottom: 6px; }
.work-error { font-size: 12px; color: #ee0a24; margin-bottom: 6px; line-height: 1.4; }
.work-actions { display: flex; gap: 12px; }
</style>
