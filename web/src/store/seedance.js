import { checkSession } from '@/store/cache'
import { useSharedStore } from '@/store/sharedata'
import { showMessageError, showMessageOK } from '@/utils/dialog'
import { httpDownload, httpGet, httpPost } from '@/utils/http'
import { replaceImg, substr } from '@/utils/libs'
import { ElMessageBox } from 'element-plus'
import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import { seedanceModes } from './seedanceModes'
import { splitSeedanceReferenceUrls, transformSeedancePromptMentions } from './seedanceReferences'

export const useSeedanceStore = defineStore('seedance', () => {
  const activeMode = ref('multimodal_ref')
  const loading = ref(false)
  const submitting = ref(false)
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const taskFilter = ref('all')
  const currentList = ref([])
  const isOver = ref(false)
  const isLogin = ref(false)
  const userPower = ref(100)
  const currentPrompt = ref('')
  const selectedModel = ref('seedance-fast')
  const powerConfig = reactive({})
  const showDialog = ref(false)
  const currentVideoUrl = ref('')

  const shareStore = useSharedStore()

  const modes = seedanceModes

  const videoModels = [
    { label: 'Seedance 2.0 Fast', value: 'seedance-fast', provider: 'seedance', model: 'fast' },
    { label: 'Seedance 2.0', value: 'seedance-standard', provider: 'seedance', model: 'standard' },
    { label: 'Veo 3.1', value: 'veo3.1-4k', provider: 'veo', model: 'veo3.1-4k' },
    { label: 'Veo 3.1 Fast', value: 'veo_3_1-fast-4K', provider: 'veo', model: 'veo_3_1-fast-4K' },
  ]

  const currentModelConfig = computed(() => videoModels.find((m) => m.value === selectedModel.value) || videoModels[0])
  const isVeo = computed(() => currentModelConfig.value.provider === 'veo')

  const resolutionOptions = [
    { label: '480p', value: '480p' },
    { label: '720p', value: '720p' },
    { label: '1080p', value: '1080p' },
  ]

  const ratioOptions = [
    { label: '自适应', value: 'adaptive' },
    { label: '16:9 (横版)', value: '16:9' },
    { label: '9:16 (竖版)', value: '9:16' },
    { label: '1:1 (正方形)', value: '1:1' },
    { label: '4:3', value: '4:3' },
    { label: '3:4', value: '3:4' },
    { label: '21:9', value: '21:9' },
  ]

  const durationOptions = [
    { label: '5秒', value: 5 },
    { label: '8秒', value: 8 },
    { label: '10秒', value: 10 },
    { label: '自动', value: -1 },
  ]

  const veoResolutionOptions = [
    { label: '720p', value: '720p' },
    { label: '1080p', value: '1080p' },
    { label: '4k', value: '4k' },
  ]

  const veoRatioOptions = [
    { label: '16:9 (横版)', value: '16:9' },
    { label: '9:16 (竖版)', value: '9:16' },
  ]

  const textToVideoParams = reactive({
    model: 'fast',
    resolution: '720p',
    ratio: '16:9',
    duration: 5,
    generate_audio: true,
    watermark: false,
  })

  const imageToVideoFirstParams = reactive({
    model: 'fast',
    first_frame_url: '',
    resolution: '720p',
    ratio: 'adaptive',
    duration: 5,
    generate_audio: false,
    watermark: false,
  })

  const imageToVideoDualParams = reactive({
    model: 'fast',
    first_frame_url: '',
    last_frame_url: '',
    resolution: '720p',
    ratio: 'adaptive',
    duration: 5,
    generate_audio: false,
    watermark: false,
  })

  const multimodalRefParams = reactive({
    model: 'fast',
    reference_urls: [],
    resolution: '720p',
    ratio: '16:9',
    duration: 5,
    generate_audio: true,
    watermark: false,
  })

  const veoParams = reactive({
    model: 'veo3.1-4k',
    images: [],
    resolution: '4k',
    aspect_ratio: '16:9',
  })

  const editVideoParams = reactive({
    model: 'fast',
    ref_video_url: '',
    ref_image_url: '',
    resolution: '720p',
    ratio: '16:9',
    duration: 5,
    generate_audio: true,
    watermark: false,
  })

  const extendVideoParams = reactive({
    model: 'fast',
    video_urls: [],
    resolution: '720p',
    ratio: '16:9',
    duration: 8,
    generate_audio: true,
    watermark: false,
  })

  const virtualAvatarParams = reactive({
    model: 'fast',
    asset_id: '',
    resolution: '720p',
    ratio: '16:9',
    duration: 5,
    generate_audio: true,
    watermark: false,
  })

  const currentMode = computed(() => modes.find((m) => m.key === activeMode.value) || modes[0])
  const currentPowerCost = computed(() => {
    if (isVeo.value) {
      const key = `${veoParams.model}_${veoParams.resolution}`
      return powerConfig.veo_powers?.[key] || 0
    }
    const p = getStoreParams()
    const model = p?.model || 'fast'
    const resolution = p?.resolution || '720p'
    const duration = p?.duration || 5
    const effectiveDuration = duration <= 0 ? 5 : duration
    const priceMap = model === 'standard' ? powerConfig.vip_price : powerConfig.fast_price
    const perSecond = priceMap?.[resolution] || priceMap?.['720p'] || 1
    return perSecond * effectiveDuration
  })

  function getStoreParams() {
    switch (activeMode.value) {
      case 'text_to_video': return textToVideoParams
      case 'image_to_video_first': return imageToVideoFirstParams
      case 'image_to_video_dual': return imageToVideoDualParams
      case 'multimodal_ref': return multimodalRefParams
      case 'edit_video': return editVideoParams
      case 'extend_video': return extendVideoParams
      case 'virtual_avatar': return virtualAvatarParams
      default: return multimodalRefParams
    }
  }

  const init = async () => {
    try {
      const powerRes = await httpGet('/api/seedance/power-config')
      if (powerRes.data) {
        Object.assign(powerConfig, powerRes.data)
      }
      const systemRes = await httpGet('/api/config/get?key=system')
      if (systemRes.data?.veo_powers) {
        powerConfig.veo_powers = systemRes.data.veo_powers
      }
      const user = await checkSession()
      isLogin.value = true
      userPower.value = user.power
      await fetchData(1)
      startPolling()
    } catch (error) {
      console.error('初始化失败:', error)
    }
  }

  const switchMode = (mode) => {
    activeMode.value = mode
  }

  const getModeName = (key) => {
    const mode = modes.find((m) => m.key === key)
    return mode ? mode.name : key
  }

  const getStatusText = (status) => {
    const map = {
      queued: '排队中',
      running: '生成中',
      succeeded: '已完成',
      failed: '失败',
      expired: '已过期',
    }
    return map[status] || status
  }

  const fetchData = async (pageNum = 1) => {
    if (isVeo.value) {
      await fetchVeoData(pageNum)
      return
    }
    try {
      loading.value = true
      page.value = pageNum
      const response = await httpPost('/api/seedance/jobs', {
        page: pageNum,
        page_size: pageSize.value,
        filter: taskFilter.value,
      })
      const data = response.data
      if (!data.items || data.items.length === 0) {
        isOver.value = true
        if (pageNum === 1) currentList.value = []
        return
      }
      total.value = data.total || 0
      if (data.items.length < pageSize.value) isOver.value = true
      if (pageNum === 1) {
        currentList.value = data.items
      } else {
        currentList.value = currentList.value.concat(data.items)
      }
    } catch (error) {
      showMessageError('获取任务列表失败:' + error.message)
    } finally {
      loading.value = false
    }
  }

  const fetchVeoData = async (pageNum = 1, mergeExisting = false, silent = false) => {
    try {
      if (!silent) loading.value = true
      page.value = pageNum
      const response = await httpGet('/api/video/list', {
        type: 'veo',
        page: pageNum,
        page_size: pageSize.value,
      })
      const data = response.data
      const items = (data.items || []).map((item) => ({
        ...item,
        status: getVeoStatus(item.progress),
        video_url: item.video_url || item.water_url,
        cover_url: item.cover_url,
      }))
      total.value = data.total || 0
      isOver.value = items.length < pageSize.value
      if (mergeExisting && pageNum === 1) {
        currentList.value.forEach((item) => {
          const found = items.find((i) => i.id === item.id)
          if (found) Object.assign(item, found)
        })
        return
      }
      if (pageNum === 1) {
        currentList.value = items
      } else {
        currentList.value = currentList.value.concat(items)
      }
    } catch (error) {
      showMessageError('获取任务列表失败:' + error.message)
    } finally {
      if (!silent) loading.value = false
    }
  }

  const getVeoStatus = (progress) => {
    if (progress === 100) return 'succeeded'
    if (progress < 0) return 'failed'
    return 'running'
  }

  let pollHandler = null
  const startPolling = () => {
    if (pollHandler) clearInterval(pollHandler)
    pollHandler = setInterval(async () => {
      if (isVeo.value) {
        await fetchVeoData(1, true, true)
        const todoList = currentList.value.filter((i) => i.status === 'queued' || i.status === 'running')
        if (todoList.length === 0) stopPolling()
        return
      }
      const response = await httpPost('/api/seedance/jobs', { page: 1, page_size: 20 })
      const data = response.data
      if (!data.items || data.items.length === 0) {
        stopPolling()
        return
      }
      const todoList = data.items.filter((i) => i.status === 'queued' || i.status === 'running')
      currentList.value.forEach((item) => {
        const found = data.items.find((i) => i.id === item.id)
        if (found) Object.assign(item, found)
      })
      if (todoList.length === 0) stopPolling()
    }, 5000)
  }

  const stopPolling = () => {
    if (pollHandler) {
      clearInterval(pollHandler)
      pollHandler = null
    }
  }

  const submitTask = async () => {
    if (!isLogin.value) {
      shareStore.setShowLoginDialog(true)
      return
    }
    if (userPower.value < currentPowerCost.value) {
      showMessageError('算力不足')
      return
    }
    if (!currentPrompt.value) {
      showMessageError('提示词不能为空')
      return
    }
    if (isVeo.value && veoParams.images.length > 2) {
      showMessageError('最多支持两张参考图片')
      return
    }

    try {
      submitting.value = true
      if (isVeo.value) {
        await submitVeoTask()
        return
      }
      const referenceUrls = activeMode.value === 'multimodal_ref' ? multimodalRefParams.reference_urls || [] : []
      const requestData = {
        task_type: activeMode.value,
        prompt: activeMode.value === 'multimodal_ref'
          ? transformSeedancePromptMentions(currentPrompt.value, referenceUrls)
          : currentPrompt.value,
        model: multimodalRefParams.model,
        resolution: multimodalRefParams.resolution,
        ratio: multimodalRefParams.ratio,
        duration: multimodalRefParams.duration,
        generate_audio: multimodalRefParams.generate_audio,
        watermark: multimodalRefParams.watermark,
      }
      if (activeMode.value === 'multimodal_ref') {
        Object.assign(requestData, splitSeedanceReferenceUrls(referenceUrls))
      }

      const response = await httpPost('/api/seedance/task', requestData)
      if (response.data) {
        showMessageOK('任务提交成功')
        isOver.value = false
        await fetchData(1)
        startPolling()
      }
    } catch (error) {
      showMessageError(error.message || '提交任务失败')
    } finally {
      submitting.value = false
    }
  }

  const submitVeoTask = async () => {
    const response = await httpPost('/api/video/veo/create', {
      model: veoParams.model,
      prompt: currentPrompt.value,
      images: veoParams.images,
      aspect_ratio: veoParams.aspect_ratio,
      resolution: veoParams.resolution,
    })
    if (response.code === 0) {
      showMessageOK('任务提交成功')
      isOver.value = false
      await fetchData(1)
      startPolling()
    }
  }

  const downloadFile = async (item) => {
    const url = replaceImg(item.video_url)
    const downloadURL = `/api/download?url=${url}`
    const urlObj = new URL(url)
    const fileName = urlObj.pathname.split('/').pop()
    item.downloading = true
    try {
      const response = await httpDownload(downloadURL)
      const blob = new Blob([response.data])
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = fileName
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(link.href)
    } catch (error) {
      showMessageError('下载失败')
    } finally {
      item.downloading = false
    }
  }

  const retryTask = async (taskId) => {
    try {
      const response = await httpGet(`/api/seedance/retry?id=${taskId}`)
      if (response.data) {
        showMessageOK('重试任务已提交')
        isOver.value = false
        await fetchData(1)
        startPolling()
      }
    } catch (error) {
      showMessageError(error.message || '重试失败')
    }
  }

  const removeJob = async (item) => {
    try {
      await ElMessageBox.confirm('确定要删除这个任务吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      })
      const response = isVeo.value
        ? await httpGet('/api/video/remove', { id: item.id })
        : await httpGet('/api/seedance/remove', { id: item.id })
      if (response.data) {
        showMessageOK('删除成功')
        await fetchData(1)
      }
    } catch (error) {
      if (error !== 'cancel') {
        showMessageError(error.message || '删除失败')
      }
    }
  }

  const playVideo = (item) => {
    currentVideoUrl.value = item.video_url
    showDialog.value = true
  }

  const cleanup = () => {
    stopPolling()
  }

  return {
    activeMode, loading, submitting, page, pageSize, total, currentList, isOver,
    isLogin, userPower, currentPrompt, selectedModel, powerConfig, showDialog, currentVideoUrl,
    modes, videoModels, currentModelConfig, isVeo, resolutionOptions, ratioOptions, durationOptions,
    veoResolutionOptions, veoRatioOptions,
    textToVideoParams, imageToVideoFirstParams, imageToVideoDualParams,
    multimodalRefParams, veoParams, editVideoParams, extendVideoParams, virtualAvatarParams,
    currentMode, currentPowerCost,
    init, switchMode, getModeName, getStatusText, fetchData, submitTask,
    downloadFile, retryTask, removeJob, playVideo, cleanup,
    substr, replaceImg,
  }
})
