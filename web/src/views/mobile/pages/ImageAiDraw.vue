<template>
  <div class="mobile-sd">
    <van-form>
      <van-cell-group class="px-3 pt-3 pb-4">
        <div>
          <van-field
            v-model="selectedModelText"
            is-link
            label="生图模型"
            placeholder="选择生图模型"
            @click="showModelPicker = true"
          />
          <van-popup v-model:show="showModelPicker" position="bottom" teleport="#app">
            <van-picker :columns="models" @cancel="showModelPicker = false" @confirm="modelConfirm" />
          </van-popup>
        </div>

        <div>
          <van-field label="生成模式">
            <template #input>
              <van-radio-group v-model="params.mode" direction="horizontal" @change="changeMode">
                <van-radio name="text_to_image">文生图</van-radio>
                <van-radio name="image_to_image">图生图</van-radio>
              </van-radio-group>
            </template>
          </van-field>
        </div>

        <!-- Gemini 参数 -->
        <template v-if="modelFamily === 'gemini'">
          <div>
            <van-field
              v-model="aspectRatioText"
              is-link
              label="宽高比"
              placeholder="选择宽高比"
              @click="showAspectRatioPicker = true"
            />
            <van-popup v-model:show="showAspectRatioPicker" position="bottom" teleport="#app">
              <van-picker
                :columns="aspectRatios"
                @cancel="showAspectRatioPicker = false"
                @confirm="aspectRatioConfirm"
              />
            </van-popup>
          </div>
          <div>
            <van-field
              v-model="imageSizeText"
              is-link
              label="分辨率"
              placeholder="选择分辨率"
              @click="showImageSizePicker = true"
            />
            <van-popup v-model:show="showImageSizePicker" position="bottom" teleport="#app">
              <van-picker
                :columns="imageSizes"
                @cancel="showImageSizePicker = false"
                @confirm="imageSizeConfirm"
              />
            </van-popup>
          </div>
        </template>

        <!-- GPT 参数 -->
        <template v-if="modelFamily === 'gpt'">
          <div>
            <van-field
              v-model="qualityText"
              is-link
              label="图片质量"
              placeholder="选择图片质量"
              @click="showQualityPicker = true"
            />
            <van-popup v-model:show="showQualityPicker" position="bottom" teleport="#app">
              <van-picker :columns="qualities" @cancel="showQualityPicker = false" @confirm="qualityConfirm" />
            </van-popup>
          </div>
          <div>
            <van-field
              v-model="sizeText"
              is-link
              label="图片尺寸"
              placeholder="选择图片尺寸"
              @click="showSizePicker = true"
            />
            <van-popup v-model:show="showSizePicker" position="bottom" teleport="#app">
              <van-picker :columns="gptSizes" @cancel="showSizePicker = false" @confirm="sizeConfirm" />
            </van-popup>
          </div>
        </template>

        <van-field
          v-model="params.prompt"
          rows="3"
          autosize
          maxlength="4096"
          type="textarea"
          placeholder="请在此输入绘画提示词"
        />

        <!-- 图生图上传 -->
        <div v-if="params.mode === 'image_to_image'" class="p-3">
          <label class="text-sm font-semibold mb-2 block">参考图</label>
          <van-uploader
            v-model="uploadImages"
            :max-count="1"
            :after-read="afterRead"
            @delete="onDeleteImage"
          />
        </div>

        <div class="sticky bottom-4 bg-[var(--van-cell-group-background)] rounded-xl p-4 shadow-sm">
          <button
            @click="generate"
            :disabled="loading"
            type="button"
            class="w-full py-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white font-semibold rounded-xl disabled:from-gray-400 disabled:to-gray-400 disabled:cursor-not-allowed hover:from-blue-600 hover:to-purple-700 transition-all duration-200 flex items-center justify-center space-x-2"
          >
            <i v-if="loading" class="iconfont icon-loading animate-spin"></i>
            <i v-else class="iconfont icon-chuangzuo"></i>
            <span>{{ loading ? '创作中...' : '立即生成' }}({{ drawPower }}算力)</span>
          </button>
        </div>
      </van-cell-group>
    </van-form>

    <h3 class="m-3">任务列表</h3>
    <div class="running-job-list pt-3 pb-3">
      <van-empty
        v-if="runningJobs.length === 0"
        image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
        image-size="80"
        description="暂无记录"
      />
      <van-grid :gutter="10" :column-num="3" v-else>
        <van-grid-item v-for="item in runningJobs" :key="item.id">
          <div v-if="item.progress > 0">
            <van-image src="/images/img-holder.png"></van-image>
            <div class="progress">
              <van-circle
                v-model:current-rate="item.progress"
                :rate="item.progress"
                :speed="100"
                :text="item.progress + '%'"
                :stroke-width="60"
                size="90px"
              />
            </div>
          </div>
          <div v-else class="task-in-queue">
            <span class="icon"><i class="iconfont icon-quick-start"></i></span>
            <span class="text">排队中</span>
          </div>
        </van-grid-item>
      </van-grid>
    </div>

    <h3 class="m-3">创作记录</h3>
    <div class="finish-job-list">
      <van-empty
        v-if="finishedJobs.length === 0"
        image="https://fastly.jsdelivr.net/npm/@vant/assets/custom-empty-image.png"
        image-size="80"
        description="暂无记录"
      />
      <van-list
        v-else
        v-model:error="error"
        v-model:loading="loading"
        :finished="finished"
        error-text="请求失败，点击重新加载"
        finished-text="没有更多了"
        @load="onLoad"
      >
        <van-grid :gutter="10" :column-num="2">
          <van-grid-item v-for="item in finishedJobs" :key="item.id">
            <div class="failed" v-if="item.progress === 101">
              <div class="title">任务失败</div>
              <div class="opt">
                <van-button size="small" @click="showErrMsg(item)">详情</van-button>
                <van-button type="danger" @click="removeImage($event, item)" size="small">删除</van-button>
              </div>
            </div>
            <div class="job-item" v-else>
              <van-image :src="item['img_url']" lazy-load @click="imageView(item)" fit="cover">
                <template v-slot:loading>
                  <van-loading type="spinner" size="20" />
                </template>
              </van-image>
              <div class="remove">
                <el-button type="danger" :icon="Delete" @click="removeImage($event, item)" circle />
                <el-button type="warning" v-if="item.publish" @click="publishImage($event, item, false)" circle>
                  <i class="iconfont icon-cancel-share"></i>
                </el-button>
                <el-button type="success" v-else @click="publishImage($event, item, true)" circle>
                  <i class="iconfont icon-share-bold"></i>
                </el-button>
                <el-button type="primary" @click="showPrompt(item)" circle>
                  <i class="iconfont icon-prompt"></i>
                </el-button>
              </div>
            </div>
          </van-grid-item>
        </van-grid>
      </van-list>
    </div>
    <button
      style="display: none"
      class="copy-prompt-aidraw"
      :data-clipboard-text="prompt"
      id="copy-btn-aidraw"
    >
      复制
    </button>
  </div>
</template>

<script setup>
import { checkSession } from '@/store/cache'
import { useSharedStore } from '@/store/sharedata'
import { httpGet, httpPost } from '@/utils/http'
import { showLoginDialog } from '@/utils/libs'
import { Delete } from '@element-plus/icons-vue'
import Clipboard from 'clipboard'
import {
  showConfirmDialog,
  showDialog,
  showFailToast,
  showImagePreview,
  showNotify,
  showSuccessToast,
  showToast,
} from 'vant'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'

const isLogin = ref(false)
const router = useRouter()
const store = useSharedStore()
const clipboard = ref(null)
const prompt = ref('')
const drawPower = ref(0)

// 参数选项
const aspectRatios = ['1:1', '3:4', '4:3', '9:16', '16:9']
const imageSizes = ['512', '1K', '2K', '4K']
const qualities = [
  { text: '低', value: 'low' },
  { text: '中', value: 'medium' },
  { text: '高', value: 'high' },
]
const gptSizes = [
  { text: '1:1 (1024x1024)', value: '1024x1024' },
  { text: '2:3 (1024x1536)', value: '1024x1536' },
  { text: '3:2 (1536x1024)', value: '1536x1024' },
]

const showModelPicker = ref(false)
const showAspectRatioPicker = ref(false)
const showImageSizePicker = ref(false)
const showQualityPicker = ref(false)
const showSizePicker = ref(false)

const selectedModelText = ref('')
const aspectRatioText = ref('1:1')
const imageSizeText = ref('1K')
const qualityText = ref('中')
const sizeText = ref('1:1 (1024x1024)')

const models = ref([])
const currentModel = ref(null)
const modelFamily = computed(() => {
  if (!currentModel.value) return 'gemini'
  return currentModel.value.value.includes('gemini') ? 'gemini' : 'gpt'
})

const params = ref({
  mode: 'text_to_image',
  prompt: '',
  aspect_ratio: '1:1',
  image_size: '1K',
  quality: 'medium',
  size: '1024x1024',
  images: [],
  model_id: 0,
})

const uploadImages = ref([])

const runningJobs = ref([])
const finishedJobs = ref([])
const allowPulling = ref(true)
const tastPullHandler = ref(null)
const loading = ref(false)
const finished = ref(false)
const error = ref(false)
const page = ref(0)
const pageSize = ref(10)

const aidrawKeywords = ['gemini', 'gpt-image']
const isAidrawModel = (model) => {
  const val = (model.value || '').toLowerCase()
  return aidrawKeywords.some((k) => val.includes(k))
}

onMounted(() => {
  initData()
  clipboard.value = new Clipboard('.copy-prompt-aidraw')
  clipboard.value.on('success', () => showNotify({ type: 'success', message: '复制成功', duration: 1000 }))
  clipboard.value.on('error', () => showNotify({ type: 'danger', message: '复制失败', duration: 2000 }))

  httpGet('/api/aidraw/models')
    .then((res) => {
      const filtered = (res.data || []).filter(isAidrawModel)
      for (const m of filtered) {
        models.value.push({ text: m.name, value: m.value, id: m.id, power: m.power })
      }
      if (models.value.length > 0) {
        currentModel.value = models.value[0]
        selectedModelText.value = models.value[0].text
        params.value.model_id = models.value[0].id
        drawPower.value = models.value[0].power
      }
    })
    .catch((e) => {
      showNotify({ type: 'danger', message: '获取模型列表失败：' + e.message })
    })
})

onUnmounted(() => {
  clipboard.value?.destroy()
  if (tastPullHandler.value) clearInterval(tastPullHandler.value)
})

const initData = () => {
  checkSession()
    .then((user) => {
      isLogin.value = true
      fetchRunningJobs()
      fetchFinishJobs(1)
      tastPullHandler.value = setInterval(() => {
        if (allowPulling.value) fetchRunningJobs()
      }, 5000)
    })
    .catch(() => {
      loading.value = false
    })
}

const fetchRunningJobs = () => {
  httpGet('/api/aidraw/jobs?finish=0')
    .then((res) => {
      if (runningJobs.value.length !== res.data.items.length) {
        fetchFinishJobs(1)
      }
      if (res.data.items.length === 0) {
        allowPulling.value = false
      }
      runningJobs.value = res.data.items
    })
    .catch((e) => showNotify({ type: 'danger', message: '获取任务失败：' + e.message }))
}

const fetchFinishJobs = (p) => {
  loading.value = true
  httpGet(`/api/aidraw/jobs?finish=1&page=${p}&page_size=${pageSize.value}`)
    .then((res) => {
      const jobs = res.data.items
      if (jobs.length < pageSize.value) finished.value = true
      if (p === 1) {
        finishedJobs.value = jobs
      } else {
        finishedJobs.value = finishedJobs.value.concat(jobs)
      }
      loading.value = false
    })
    .catch((e) => {
      loading.value = false
      showNotify({ type: 'danger', message: '获取任务失败：' + e.message })
    })
}

const onLoad = () => {
  page.value += 1
  fetchFinishJobs(page.value)
}

const generate = () => {
  if (!isLogin.value) return showLoginDialog(router)
  if (params.value.prompt === '') return showToast('请输入绘画提示词！')

  httpPost('/api/aidraw/image', params.value)
    .then(() => {
      showSuccessToast('绘画任务推送成功')
      allowPulling.value = true
      runningJobs.value.push({ progress: 0 })
    })
    .catch((e) => showFailToast('任务推送失败：' + e.message))
}

const showPrompt = (item) => {
  prompt.value = item.prompt
  showConfirmDialog({ title: '绘画提示词', message: item.prompt, confirmButtonText: '复制', cancelButtonText: '关闭' })
    .then(() => document.querySelector('#copy-btn-aidraw').click())
    .catch(() => {})
}

const showErrMsg = (item) => {
  showDialog({ title: '错误详情', message: item['err_msg'] })
}

const removeImage = (event, item) => {
  event.stopPropagation()
  showConfirmDialog({ title: '删除确认', message: '此操作将会删除任务和图片，继续操作码?' })
    .then(() => {
      httpGet('/api/aidraw/remove', { id: item.id })
        .then(() => {
          showSuccessToast('任务删除成功')
          fetchFinishJobs(1)
        })
        .catch((e) => showFailToast('任务删除失败：' + e.message))
    })
    .catch(() => showToast('您取消了操作'))
}

const publishImage = (event, item, action) => {
  event.stopPropagation()
  const text = action ? '图片发布' : '取消发布'
  httpGet('/api/aidraw/publish', { id: item.id, action })
    .then(() => {
      showSuccessToast(text + '成功')
      item.publish = action
    })
    .catch((e) => showFailToast(text + '失败：' + e.message))
}

const imageView = (item) => showImagePreview([item['img_url']])

const modelConfirm = (item) => {
  const opt = item.selectedOptions[0]
  currentModel.value = opt
  selectedModelText.value = opt.text
  params.value.model_id = opt.id
  drawPower.value = opt.power
  showModelPicker.value = false
}

const aspectRatioConfirm = (item) => {
  params.value.aspect_ratio = item.selectedOptions[0].text
  aspectRatioText.value = item.selectedOptions[0].text
  showAspectRatioPicker.value = false
}

const imageSizeConfirm = (item) => {
  params.value.image_size = item.selectedOptions[0].text
  imageSizeText.value = item.selectedOptions[0].text
  showImageSizePicker.value = false
}

const qualityConfirm = (item) => {
  params.value.quality = item.selectedOptions[0].value
  qualityText.value = item.selectedOptions[0].text
  showQualityPicker.value = false
}

const sizeConfirm = (item) => {
  params.value.size = item.selectedOptions[0].value
  sizeText.value = item.selectedOptions[0].text
  showSizePicker.value = false
}

const changeMode = () => {
  if (params.value.mode === 'text_to_image') {
    params.value.images = []
    uploadImages.value = []
  }
}

const afterRead = (file) => {
  params.value.images = [file.content]
}

const onDeleteImage = () => {
  params.value.images = []
}
</script>

<style lang="scss">
@use '@/assets/css/mobile/image-sd.scss' as *;
</style>
