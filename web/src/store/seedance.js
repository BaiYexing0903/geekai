import { checkSession } from '@/store/cache'
import { useSharedStore } from '@/store/sharedata'
import { showMessageError, showMessageOK } from '@/utils/dialog'
import { httpDownload, httpGet, httpPost } from '@/utils/http'
import { replaceImg, substr } from '@/utils/libs'
import { ElMessageBox } from 'element-plus'
import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'

export const useSeedanceStore = defineStore('seedance', () => {
  const activeMode = ref('text_to_video')
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
  const powerConfig = reactive({})
  const showDialog = ref(false)
  const currentVideoUrl = ref('')

  const shareStore = useSharedStore()

  const modes = [
    { key: 'text_to_video', name: '文生视频', icon: 'video', needsImage: false },
    { key: 'image_to_video_first', name: '图生视频', icon: 'image', needsImage: true },
    { key: 'image_to_video_dual', name: '首尾帧', icon: 'image', needsImage: true },
    { key: 'multimodal_ref', name: '多模态', icon: 'api-key', needsImage: false },
    { key: 'edit_video', name: '编辑视频', icon: 'edit', needsImage: true },
    { key: 'extend_video', name: '延长视频', icon: 'extend', needsImage: true },
    { key: 'virtual_avatar', name: '虚拟人像', icon: 'user', needsImage: false },
  ]

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
    image_urls: [],
    video_urls: [],
    audio_urls: [],
    resolution: '720p',
    ratio: '16:9',
    duration: 5,
    generate_audio: true,
    watermark: false,
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
      default: return textToVideoParams
    }
  }

  const init = async () => {
    try {
      const powerRes = await httpGet('/api/seedance/power-config')
      if (powerRes.data) {
        Object.assign(powerConfig, powerRes.data)
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
    try {
      loading.value = true
      page.value = pageNum
      const response = await httpPost('/api/seedance/jobs', {
        page: pageNum,
        page_size: pageSize.value,
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

  let pollHandler = null
  const startPolling = () => {
    if (pollHandler) clearInterval(pollHandler)
    pollHandler = setInterval(async () => {
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
    if (activeMode.value !== 'virtual_avatar' && !currentPrompt.value && activeMode.value !== 'image_to_video_first') {
      showMessageError('提示词不能为空')
      return
    }

    try {
      submitting.value = true
      let requestData = {
        task_type: activeMode.value,
        prompt: currentPrompt.value,
      }

      switch (activeMode.value) {
        case 'text_to_video':
          Object.assign(requestData, {
            model: textToVideoParams.model,
            resolution: textToVideoParams.resolution,
            ratio: textToVideoParams.ratio,
            duration: textToVideoParams.duration,
            generate_audio: textToVideoParams.generate_audio,
            watermark: textToVideoParams.watermark,
          })
          break
        case 'image_to_video_first':
          Object.assign(requestData, {
            model: imageToVideoFirstParams.model,
            first_frame_url: imageToVideoFirstParams.first_frame_url,
            resolution: imageToVideoFirstParams.resolution,
            ratio: imageToVideoFirstParams.ratio,
            duration: imageToVideoFirstParams.duration,
            generate_audio: imageToVideoFirstParams.generate_audio,
            watermark: imageToVideoFirstParams.watermark,
          })
          break
        case 'image_to_video_dual':
          Object.assign(requestData, {
            model: imageToVideoDualParams.model,
            first_frame_url: imageToVideoDualParams.first_frame_url,
            last_frame_url: imageToVideoDualParams.last_frame_url,
            resolution: imageToVideoDualParams.resolution,
            ratio: imageToVideoDualParams.ratio,
            duration: imageToVideoDualParams.duration,
            generate_audio: imageToVideoDualParams.generate_audio,
            watermark: imageToVideoDualParams.watermark,
          })
          break
        case 'multimodal_ref':
          Object.assign(requestData, {
            model: multimodalRefParams.model,
            image_urls: multimodalRefParams.image_urls,
            video_urls: multimodalRefParams.video_urls,
            audio_urls: multimodalRefParams.audio_urls,
            resolution: multimodalRefParams.resolution,
            ratio: multimodalRefParams.ratio,
            duration: multimodalRefParams.duration,
            generate_audio: multimodalRefParams.generate_audio,
            watermark: multimodalRefParams.watermark,
          })
          break
        case 'edit_video':
          Object.assign(requestData, {
            model: editVideoParams.model,
            ref_video_url: editVideoParams.ref_video_url,
            ref_image_url: editVideoParams.ref_image_url,
            resolution: editVideoParams.resolution,
            ratio: editVideoParams.ratio,
            duration: editVideoParams.duration,
            generate_audio: editVideoParams.generate_audio,
            watermark: editVideoParams.watermark,
          })
          break
        case 'extend_video':
          Object.assign(requestData, {
            model: extendVideoParams.model,
            video_urls: extendVideoParams.video_urls,
            resolution: extendVideoParams.resolution,
            ratio: extendVideoParams.ratio,
            duration: extendVideoParams.duration,
            generate_audio: extendVideoParams.generate_audio,
            watermark: extendVideoParams.watermark,
          })
          break
        case 'virtual_avatar':
          Object.assign(requestData, {
            model: virtualAvatarParams.model,
            asset_id: virtualAvatarParams.asset_id,
            resolution: virtualAvatarParams.resolution,
            ratio: virtualAvatarParams.ratio,
            duration: virtualAvatarParams.duration,
            generate_audio: virtualAvatarParams.generate_audio,
            watermark: virtualAvatarParams.watermark,
          })
          break
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
      const response = await httpGet('/api/seedance/remove', { id: item.id })
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
    isLogin, userPower, currentPrompt, powerConfig, showDialog, currentVideoUrl,
    modes, resolutionOptions, ratioOptions, durationOptions,
    textToVideoParams, imageToVideoFirstParams, imageToVideoDualParams,
    multimodalRefParams, editVideoParams, extendVideoParams, virtualAvatarParams,
    currentMode, currentPowerCost,
    init, switchMode, getModeName, getStatusText, fetchData, submitTask,
    downloadFile, retryTask, removeJob, playVideo, cleanup,
    substr, replaceImg,
  }
})
