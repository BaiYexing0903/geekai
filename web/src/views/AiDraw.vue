<template>
  <div>
    <div class="page-aidraw">
      <div class="inner custom-scroll">
        <div class="sd-box">
          <h2>AI 智能绘画</h2>

          <div class="sd-params">
            <el-form :model="params" label-width="80px" label-position="left">
              <div class="param-line pt-1">
                <el-form-item label="生图模型">
                  <template #default>
                    <div class="form-item-inner">
                      <el-select
                        v-model="selectedModel"
                        style="width: 150px"
                        placeholder="请选择模型"
                        @change="changeModel"
                      >
                        <el-option v-for="v in models" :label="v.name" :value="v" :key="v.value" />
                      </el-select>
                    </div>
                  </template>
                </el-form-item>
              </div>

              <div class="param-line">
                <el-form-item label="生成模式">
                  <template #default>
                    <div class="form-item-inner">
                      <el-radio-group v-model="params.mode" @change="changeMode">
                        <el-radio-button value="text_to_image">文生图</el-radio-button>
                        <el-radio-button value="image_to_image">图生图</el-radio-button>
                      </el-radio-group>
                    </div>
                  </template>
                </el-form-item>
              </div>

              <!-- Gemini 参数 -->
              <template v-if="modelFamily === 'gemini'">
                <div class="param-line">
                  <el-form-item label="宽高比">
                    <template #default>
                      <div class="form-item-inner">
                        <el-select v-model="params.aspect_ratio" style="width: 150px">
                          <el-option v-for="v in aspectRatios" :label="v" :value="v" :key="v" />
                        </el-select>
                      </div>
                    </template>
                  </el-form-item>
                </div>
                <div class="param-line">
                  <el-form-item label="分辨率">
                    <template #default>
                      <div class="form-item-inner">
                        <el-select v-model="params.image_size" style="width: 150px">
                          <el-option v-for="v in imageSizes" :label="v" :value="v" :key="v" />
                        </el-select>
                      </div>
                    </template>
                  </el-form-item>
                </div>
              </template>

              <!-- GPT 参数 -->
              <template v-if="modelFamily === 'gpt'">
                <div class="param-line">
                  <el-form-item label="图片质量">
                    <template #default>
                      <div class="form-item-inner">
                        <el-select v-model="params.quality" style="width: 150px">
                          <el-option
                            v-for="v in qualities"
                            :label="v.name"
                            :value="v.value"
                            :key="v.value"
                          />
                        </el-select>
                      </div>
                    </template>
                  </el-form-item>
                </div>
                <div class="param-line">
                  <el-form-item label="图片尺寸">
                    <template #default>
                      <div class="form-item-inner">
                        <el-select v-model="params.size" style="width: 150px">
                          <el-option v-for="v in gptSizes" :label="v.label" :value="v.value" :key="v.value" />
                        </el-select>
                      </div>
                    </template>
                  </el-form-item>
                </div>
              </template>

              <div class="param-line">
                <el-input
                  v-model="params.prompt"
                  :autosize="{ minRows: 4, maxRows: 6 }"
                  type="textarea"
                  ref="promptRef"
                  maxlength="4096"
                  show-word-limit
                  placeholder="请在此输入绘画提示词，您也可以点击下面的提示词助手生成绘画提示词"
                  v-loading="promptGenerating"
                />
              </div>

              <div class="flex justify-end pt-2 pr-2">
                <el-button @click="generatePrompt" type="primary" :loading="promptGenerating">
                  <span v-if="!promptGenerating">
                    <i class="iconfont icon-chuangzuo"></i>
                    生成专业绘画指令
                  </span>
                  <span v-else>生成中...</span>
                </el-button>
              </div>

              <div class="mt-2 mb-2" v-if="params.mode === 'image_to_image'">
                <label class="text-gray-700 font-semibold">参考图</label>
                <div class="py-2">
                  <ImageUpload v-model="params.images" :max-count="1" />
                </div>
              </div>
            </el-form>
          </div>
          <div class="py-4">
            <button
              class="w-full py-3 bg-gradient-to-r from-blue-500 to-purple-600 text-white rounded-xl disabled:from-gray-400 disabled:to-gray-400 disabled:cursor-not-allowed hover:from-blue-600 hover:to-purple-700 transition-all duration-200 flex items-center justify-center space-x-2 text-base"
              type="button"
              @click="generate"
            >
              <i v-if="isGenerating" class="iconfont icon-loading animate-spin"></i>
              <i v-else class="iconfont icon-chuangzuo"></i>
              <span v-if="isGenerating">创作中...</span>
              <span v-else>立即生成({{ drawPower }}算力)</span>
            </button>
          </div>
        </div>
        <div class="task-list-box pl-6 pr-6 pb-4 pt-4 h-dvh">
          <div class="task-list-inner">
            <div class="job-list-box">
              <h2 class="text-xl">任务列表</h2>
              <task-list :list="runningJobs" />
              <template v-if="finishedJobs.length > 0">
                <h2 class="text-xl">创作记录</h2>
                <div class="finish-job-list mt-3">
                  <div v-if="finishedJobs.length > 0">
                    <Waterfall
                      :list="finishedJobs"
                      :row-key="waterfallOptions.rowKey"
                      :gutter="waterfallOptions.gutter"
                      :has-around-gutter="waterfallOptions.hasAroundGutter"
                      :width="waterfallOptions.width"
                      :breakpoints="waterfallOptions.breakpoints"
                      :img-selector="waterfallOptions.imgSelector"
                      :background-color="waterfallOptions.backgroundColor"
                      :animation-effect="waterfallOptions.animationEffect"
                      :animation-duration="waterfallOptions.animationDuration"
                      :animation-delay="waterfallOptions.animationDelay"
                      :animation-cancel="waterfallOptions.animationCancel"
                      :lazyload="waterfallOptions.lazyload"
                      :load-props="waterfallOptions.loadProps"
                      :cross-origin="waterfallOptions.crossOrigin"
                      :align="waterfallOptions.align"
                      :is-loading="loading"
                      :is-over="isOver"
                      @afterRender="loading = false"
                    >
                      <template #default="{ item, url }">
                        <div
                          class="bg-gray-900 rounded-lg shadow-md overflow-hidden transition-all duration-300 ease-linear hover:shadow-md hover:shadow-purple-800 group"
                        >
                          <div class="overflow-hidden rounded-lg">
                            <LazyImg
                              :url="url"
                              v-if="item.progress === 100"
                              class="cursor-pointer transition-all duration-300 ease-linear group-hover:scale-105"
                              @click="previewImg(item)"
                            />
                            <el-image v-else-if="item.progress === 101">
                              <template #error>
                                <div class="image-slot">
                                  <div class="err-msg-container">
                                    <div class="title">任务失败</div>
                                    <div class="opt">
                                      <el-popover
                                        title="错误详情"
                                        trigger="click"
                                        :width="250"
                                        :content="item['err_msg']"
                                        placement="top"
                                      >
                                        <template #reference>
                                          <el-button type="info">详情</el-button>
                                        </template>
                                      </el-popover>
                                      <el-button type="danger" @click="removeImage(item)">删除</el-button>
                                    </div>
                                  </div>
                                </div>
                              </template>
                            </el-image>
                          </div>
                          <div
                            class="px-4 pt-2 pb-4 border-t border-t-gray-800"
                            v-if="item.progress === 100"
                          >
                            <div
                              class="pt-3 flex justify-center items-center border-t border-t-gray-600 border-opacity-50"
                            >
                              <div class="flex">
                                <el-tooltip content="取消分享" placement="top" v-if="item.publish">
                                  <el-button type="warning" @click="publishImage(item, false)" circle>
                                    <i class="iconfont icon-cancel-share"></i>
                                  </el-button>
                                </el-tooltip>
                                <el-tooltip content="分享" placement="top" v-else>
                                  <el-button type="success" @click="publishImage(item, true)" circle>
                                    <i class="iconfont icon-share-bold"></i>
                                  </el-button>
                                </el-tooltip>

                                <el-tooltip content="复制提示词" placement="top">
                                  <el-button
                                    type="info"
                                    circle
                                    class="copy-prompt"
                                    :data-clipboard-text="item.prompt"
                                  >
                                    <i class="iconfont icon-file"></i>
                                  </el-button>
                                </el-tooltip>
                                <el-tooltip content="删除" placement="top">
                                  <el-button type="danger" :icon="Delete" @click="removeImage(item)" circle />
                                </el-tooltip>
                              </div>
                            </div>
                          </div>
                        </div>
                      </template>
                    </Waterfall>

                    <div class="flex justify-center py-10">
                      <img
                        :src="waterfallOptions.loadProps.loading"
                        class="max-w-[50px] max-h-[50px]"
                        v-if="loading"
                      />
                      <div v-else>
                        <button
                          class="px-5 py-2 rounded-full bg-purple-700 text-md text-white cursor-pointer hover:bg-purple-800 transition-all duration-300"
                          @click="fetchFinishJobs"
                          v-if="!isOver"
                        >
                          加载更多
                        </button>
                        <div class="no-more-data" v-else>
                          <span class="text-gray-500 mr-2">没有更多数据了</span>
                          <i class="iconfont icon-face"></i>
                        </div>
                      </div>
                    </div>
                  </div>
                  <el-empty :image-size="100" :image="nodata" description="暂无记录" v-else />
                </div>
              </template>
            </div>
          </div>
          <back-top :right="30" :bottom="30" />
        </div>
      </div>
    </div>

    <el-image-viewer
      @close="
        () => {
          previewURL = ''
        }
      "
      v-if="previewURL !== ''"
      :url-list="[previewURL]"
    />
  </div>
</template>

<script setup>
import nodata from '@/assets/img/no-data.png'
import BackTop from '@/components/BackTop.vue'
import TaskList from '@/components/TaskList.vue'
import ImageUpload from '@/components/ImageUpload.vue'
import { checkSession } from '@/store/cache'
import { useSharedStore } from '@/store/sharedata'
import { showMessageError, showMessageOK } from '@/utils/dialog'
import { httpGet, httpPost } from '@/utils/http'
import { Delete } from '@element-plus/icons-vue'
import Clipboard from 'clipboard'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, onUnmounted, ref, computed } from 'vue'
import { LazyImg, Waterfall } from 'vue-waterfall-plugin-next'
import 'vue-waterfall-plugin-next/dist/style.css'

const listBoxHeight = ref(0)
const isLogin = ref(false)
const loading = ref(true)
const isOver = ref(false)
const previewURL = ref('')
const store = useSharedStore()
const models = ref([])
const waterfallOptions = store.waterfallOptions

const resizeElement = function () {
  listBoxHeight.value = window.innerHeight - 58
}
resizeElement()
window.onresize = () => resizeElement()

// 参数选项
const aspectRatios = ['1:1', '3:4', '4:3', '9:16', '16:9']
const imageSizes = ['512', '1K', '2K', '4K']
const qualities = [
  { name: '低', value: 'low' },
  { name: '中', value: 'medium' },
  { name: '高', value: 'high' },
]
const gptSizes = [
  { label: '1:1 (1024x1024)', value: '1024x1024' },
  { label: '2:3 (1024x1536)', value: '1024x1536' },
  { label: '3:2 (1536x1024)', value: '1536x1024' },
]

const selectedModel = ref(null)
const modelFamily = computed(() => {
  if (!selectedModel.value) return 'gemini'
  return selectedModel.value.value.includes('gemini') ? 'gemini' : 'gpt'
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

const finishedJobs = ref([])
const runningJobs = ref([])
const allowPulling = ref(true)
const downloadPulling = ref(false)
const tastPullHandler = ref(null)
const downloadPullHandler = ref(null)
const userPower = ref(0)
const drawPower = ref(0)
const clipboard = ref(null)
const userId = ref(0)
const page = ref(1)
const pageSize = ref(15)
const isGenerating = ref(false)
const promptGenerating = ref(false)
const promptRef = ref(null)

// Aidraw 模型过滤关键字
const aidrawKeywords = ['gemini', 'gpt-image']

const isAidrawModel = (model) => {
  const val = (model.value || '').toLowerCase()
  return aidrawKeywords.some((k) => val.includes(k))
}

onMounted(() => {
  initData()
  clipboard.value = new Clipboard('.copy-prompt')
  clipboard.value.on('success', () => showMessageOK('复制成功！'))
  clipboard.value.on('error', () => showMessageError('复制失败！'))

  httpGet('/api/aidraw/models')
    .then((res) => {
      // 只显示 aidraw 相关的模型
      models.value = (res.data || []).filter(isAidrawModel)
      if (models.value.length > 0) {
        selectedModel.value = models.value[0]
        params.value.model_id = selectedModel.value.id
        drawPower.value = selectedModel.value.power
      }
    })
    .catch((e) => {
      showMessageError('获取模型列表失败：' + e.message)
    })
})

onUnmounted(() => {
  clipboard.value?.destroy()
  if (tastPullHandler.value) clearInterval(tastPullHandler.value)
  if (downloadPullHandler.value) clearInterval(downloadPullHandler.value)
})

const initData = () => {
  checkSession()
    .then((user) => {
      userPower.value = user['power']
      userId.value = user.id
      isLogin.value = true
      page.value = 0
      fetchRunningJobs()
      fetchFinishJobs()

      tastPullHandler.value = setInterval(() => {
        if (allowPulling.value) fetchRunningJobs()
      }, 5000)

      downloadPullHandler.value = setInterval(() => {
        if (downloadPulling.value) {
          page.value = 0
          fetchFinishJobs()
        }
      }, 5000)
    })
    .catch(() => {})
}

const fetchRunningJobs = () => {
  if (!isLogin.value) return
  httpGet('/api/aidraw/jobs?finish=false')
    .then((res) => {
      if (res.data.items && res.data.items.length !== runningJobs.value.length) {
        page.value = 0
        fetchFinishJobs()
      }
      if (res.data.items.length > 0) {
        runningJobs.value = res.data.items
      } else {
        allowPulling.value = false
        runningJobs.value = []
      }
    })
    .catch((e) => {
      ElMessage.error('获取任务失败：' + e.message)
    })
}

const fetchFinishJobs = () => {
  if (!isLogin.value) return
  loading.value = true
  page.value = page.value + 1
  httpGet(`/api/aidraw/jobs?finish=true&page=${page.value}&page_size=${pageSize.value}`)
    .then((res) => {
      if (res.data.items.length < pageSize.value) {
        isOver.value = true
        loading.value = false
      }
      const imageList = res.data.items
      let needPulling = false
      for (let i = 0; i < imageList.length; i++) {
        if (imageList[i]['img_url']) {
          imageList[i]['img_thumb'] = imageList[i]['img_url'] + '?imageView2/4/w/300/h/0/q/75'
        } else if (imageList[i].progress === 100) {
          needPulling = true
          imageList[i]['img_thumb'] = waterfallOptions.loadProps.loading
        }
      }
      if (page.value === 1) {
        downloadPulling.value = needPulling
      }
      if (page.value === 1) {
        finishedJobs.value = imageList
      } else {
        finishedJobs.value = finishedJobs.value.concat(imageList)
      }
    })
    .catch((e) => {
      ElMessage.error('获取任务失败：' + e.message)
      loading.value = false
    })
}

const generate = () => {
  if (isGenerating.value) return
  if (params.value.prompt === '') {
    promptRef.value?.focus()
    return ElMessage.error('请输入绘画提示词！')
  }
  if (!isLogin.value) {
    store.setShowLoginDialog(true)
    return
  }

  isGenerating.value = true
  httpPost('/api/aidraw/image', params.value)
    .then(() => {
      ElMessage.success('任务执行成功！')
      userPower.value -= drawPower.value
      runningJobs.value.push({
        prompt: params.value.prompt,
        progress: 0,
      })
      allowPulling.value = true
      isOver.value = false
    })
    .catch((e) => {
      ElMessage.error('任务执行失败：' + e.message)
    })
    .finally(() => {
      isGenerating.value = false
    })
}

const removeImage = (item) => {
  ElMessageBox.confirm('此操作将会删除任务和图片，继续操作码?', '删除提示', {
    confirmButtonText: '确认',
    cancelButtonText: '取消',
    type: 'warning',
  })
    .then(() => {
      httpGet('/api/aidraw/remove', { id: item.id })
        .then(() => {
          ElMessage.success('任务删除成功')
          page.value = 0
          isOver.value = false
          fetchFinishJobs()
        })
        .catch((e) => ElMessage.error('任务删除失败：' + e.message))
    })
    .catch(() => {})
}

const previewImg = (item) => {
  previewURL.value = item.img_url
}

const publishImage = (item, action) => {
  const text = action ? '图片发布' : '取消发布'
  httpGet('/api/aidraw/publish', { id: item.id, action: action })
    .then(() => {
      ElMessage.success(text + '成功')
      item.publish = action
      page.value = 0
      isOver.value = false
    })
    .catch((e) => ElMessage.error(text + '失败：' + e.message))
}

const generatePrompt = () => {
  if (params.value.prompt === '') {
    return showMessageError('请输入原始提示词')
  }
  promptGenerating.value = true
  httpPost('/api/prompt/image', { prompt: params.value.prompt })
    .then((res) => {
      params.value.prompt = res.data
    })
    .catch((e) => showMessageError('生成提示词失败：' + e.message))
    .finally(() => {
      promptGenerating.value = false
    })
}

const changeModel = (model) => {
  drawPower.value = model.power
  params.value.model_id = model.id
}

const changeMode = (mode) => {
  if (mode === 'text_to_image') {
    params.value.images = []
  }
}
</script>

<style lang="scss" scoped>
@use '../assets/css/image-aidraw.scss' as *;
@use '../assets/css/custom-scroll.scss' as *;
</style>
