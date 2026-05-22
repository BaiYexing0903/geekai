<template>
  <div class="mobile-seedance">
    <div class="sticky-header">
      <van-icon name="arrow-left" @click="$router.back()" />
      <span class="title">Seedance视频</span>
      <span></span>
    </div>

    <!-- 模式选择 -->
    <div class="mode-tabs">
      <div
        v-for="mode in store.modes"
        :key="mode.key"
        :class="['mode-tab', { active: store.activeMode === mode.key }]"
        @click="store.switchMode(mode.key)"
      >
        {{ mode.name }}
      </div>
    </div>

    <!-- 参数区域 -->
    <div class="form-card">
      <!-- 模型选择 -->
      <div class="form-item">
        <span class="form-label">模型</span>
        <div class="model-btns">
          <div :class="['model-btn', { active: getParams().model === 'fast' }]" @click="getParams().model = 'fast'">快速</div>
          <div :class="['model-btn', { active: getParams().model === 'standard' }]" @click="getParams().model = 'standard'">标准</div>
        </div>
      </div>

      <!-- 提示词 -->
      <div class="form-item" v-if="store.activeMode !== 'image_to_video_first'">
        <van-field v-model="store.currentPrompt" type="textarea" rows="3" placeholder="描述你想生成的视频画面..." maxlength="1000" show-word-limit />
      </div>

      <!-- 首帧图片 -->
      <div v-if="store.activeMode === 'image_to_video_first'" class="form-item">
        <span class="form-label">首帧图片</span>
        <FileUpload v-model="store.imageToVideoFirstParams.first_frame_url" accept="image/*" placeholder="点击上传图片" />
      </div>

      <!-- 首尾帧 -->
      <template v-if="store.activeMode === 'image_to_video_dual'">
        <div class="form-item">
          <span class="form-label">首帧图片</span>
          <FileUpload v-model="store.imageToVideoDualParams.first_frame_url" accept="image/*" placeholder="点击上传图片" />
        </div>
        <div class="form-item">
          <span class="form-label">尾帧图片</span>
          <FileUpload v-model="store.imageToVideoDualParams.last_frame_url" accept="image/*" placeholder="点击上传图片" />
        </div>
      </template>

      <!-- 编辑视频 -->
      <template v-if="store.activeMode === 'edit_video'">
        <div class="form-item">
          <span class="form-label">参考视频</span>
          <FileUpload v-model="store.editVideoParams.ref_video_url" accept="video/*" placeholder="点击上传视频" />
        </div>
        <div class="form-item">
          <span class="form-label">参考图片</span>
          <FileUpload v-model="store.editVideoParams.ref_image_url" accept="image/*" placeholder="点击上传图片" />
        </div>
      </template>

      <!-- 延长视频 -->
      <div v-if="store.activeMode === 'extend_video'" class="form-item">
        <span class="form-label">参考视频</span>
        <FileUpload v-model="store.extendVideoParams.video_urls" accept="video/*" multiple :maxCount="9" placeholder="点击上传视频" />
      </div>

      <!-- 多模态 -->
      <template v-if="store.activeMode === 'multimodal_ref'">
        <div class="form-item"><span class="form-label">参考图片</span><FileUpload v-model="store.multimodalRefParams.image_urls" accept="image/*" multiple :maxCount="9" placeholder="点击上传图片" /></div>
        <div class="form-item"><span class="form-label">参考视频</span><FileUpload v-model="store.multimodalRefParams.video_urls" accept="video/*" multiple :maxCount="9" placeholder="点击上传视频" /></div>
        <div class="form-item"><span class="form-label">参考音频</span><FileUpload v-model="store.multimodalRefParams.audio_urls" accept="audio/*" multiple :maxCount="9" placeholder="点击上传音频" /></div>
      </template>

      <!-- 虚拟人像 -->
      <div v-if="store.activeMode === 'virtual_avatar'" class="form-item">
        <span class="form-label">Asset ID</span>
        <van-field v-model="store.virtualAvatarParams.asset_id" placeholder="asset-xxxxxxxxx-xxxxx" />
      </div>

      <!-- 通用参数 -->
      <div class="form-item">
        <span class="form-label">宽高比</span>
        <div class="ratio-btns">
          <div v-for="r in store.ratioOptions" :key="r.value" :class="['ratio-btn', { active: getParams().ratio === r.value }]" @click="getParams().ratio = r.value">
            {{ r.label }}
          </div>
        </div>
      </div>

      <div class="form-item">
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
              <van-tag type="primary" size="medium">{{ store.getModeName(item.type) }}</van-tag>
              <van-tag :type="item.status === 'succeeded' ? 'success' : item.status === 'failed' ? 'danger' : 'warning'" size="medium">
                {{ store.getStatusText(item.status) }}
              </van-tag>
            </div>
            <div class="work-prompt">{{ item.prompt?.substring(0, 50) }}</div>
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
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useSeedanceStore } from '@/store/mobile/seedance'
import FileUpload from '@/components/FileUpload.vue'

const store = useSeedanceStore()

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

onMounted(() => store.init())
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
.work-actions { display: flex; gap: 12px; }
</style>
