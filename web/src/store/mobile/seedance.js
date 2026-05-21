import { checkSession } from '@/store/cache'
import { showMessageError, showMessageOK } from '@/utils/dialog'
import { httpGet, httpPost } from '@/utils/http'
import { replaceImg } from '@/utils/libs'
import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'

export const useSeedanceStore = defineStore('mobile-seedance', () => {
  const activeMode = ref('text_to_video')
  const loading = ref(false)
  const submitting = ref(false)
  const page = ref(1)
  const pageSize = ref(10)
  const currentList = ref([])
  const listLoading = ref(false)
  const listFinished = ref(false)
  const isLogin = ref(false)
  const userPower = ref(100)
  const currentPrompt = ref('')
  const powerConfig = reactive({})
  const showVideoDialog = ref(false)
  const currentVideoUrl = ref('')

  const modes = [
    { key: 'text_to_video', name: '文生视频' },
    { key: 'image_to_video_first', name: '图生视频' },
    { key: 'image_to_video_dual', name: '首尾帧' },
    { key: 'multimodal_ref', name: '多模态' },
    { key: 'edit_video', name: '编辑' },
    { key: 'extend_video', name: '延长' },
    { key: 'virtual_avatar', name: '虚拟人像' },
  ]

  const ratioOptions = [
    { label: '自适应', value: 'adaptive' },
    { label: '16:9', value: '16:9' },
    { label: '9:16', value: '9:16' },
    { label: '1:1', value: '1:1' },
  ]

  const textToVideoParams = reactive({ model: 'fast', resolution: '720p', ratio: '16:9', duration: 5, generate_audio: true, watermark: false })
  const imageToVideoFirstParams = reactive({ model: 'fast', first_frame_url: '', resolution: '720p', ratio: 'adaptive', duration: 5, generate_audio: false, watermark: false })
  const imageToVideoDualParams = reactive({ model: 'fast', first_frame_url: '', last_frame_url: '', resolution: '720p', ratio: 'adaptive', duration: 5, generate_audio: false, watermark: false })
  const multimodalRefParams = reactive({ model: 'fast', image_urls: [], video_urls: [], audio_urls: [], resolution: '720p', ratio: '16:9', duration: 5, generate_audio: true, watermark: false })
  const editVideoParams = reactive({ model: 'fast', ref_video_url: '', ref_image_url: '', resolution: '720p', ratio: '16:9', duration: 5, generate_audio: true, watermark: false })
  const extendVideoParams = reactive({ model: 'fast', video_urls: [], resolution: '720p', ratio: '16:9', duration: 8, generate_audio: true, watermark: false })
  const virtualAvatarParams = reactive({ model: 'fast', asset_id: '', resolution: '720p', ratio: '16:9', duration: 5, generate_audio: true, watermark: false })

  const currentMode = computed(() => modes.find((m) => m.key === activeMode.value) || modes[0])
  const currentPowerCost = computed(() => powerConfig[activeMode.value] || 10)

  const init = async () => {
    try {
      const powerRes = await httpGet('/api/seedance/power-config')
      if (powerRes.data) Object.assign(powerConfig, powerRes.data)
      const user = await checkSession()
      isLogin.value = true
      userPower.value = user.power
      await fetchData(1)
    } catch (e) {
      console.error('init failed:', e)
    }
  }

  const getModeName = (key) => modes.find((m) => m.key === key)?.name || key
  const getStatusText = (s) => ({ queued: '排队中', running: '生成中', succeeded: '已完成', failed: '失败', expired: '已过期' }[s] || s)

  const fetchData = async (pageNum = 1) => {
    try {
      listLoading.value = true
      page.value = pageNum
      const res = await httpPost('/api/seedance/jobs', { page: pageNum, page_size: pageSize.value })
      const items = res.data?.items || []
      if (items.length < pageSize.value) listFinished.value = true
      if (pageNum === 1) {
        currentList.value = items
      } else {
        currentList.value.push(...items)
      }
    } catch (e) {
      showMessageError('获取列表失败')
    } finally {
      listLoading.value = false
    }
  }

  const submitTask = async () => {
    if (!currentPrompt.value && activeMode.value !== 'image_to_video_first') {
      showMessageError('提示词不能为空')
      return
    }
    try {
      submitting.value = true
      const p = getParams()
      const req = {
        task_type: activeMode.value,
        prompt: currentPrompt.value,
        model: p.model,
        resolution: p.resolution,
        ratio: p.ratio,
        duration: p.duration,
        generate_audio: p.generate_audio,
        watermark: p.watermark,
      }
      // Mode-specific fields
      if (activeMode.value === 'image_to_video_first') req.first_frame_url = imageToVideoFirstParams.first_frame_url
      if (activeMode.value === 'image_to_video_dual') {
        req.first_frame_url = imageToVideoDualParams.first_frame_url
        req.last_frame_url = imageToVideoDualParams.last_frame_url
      }
      if (activeMode.value === 'multimodal_ref') {
        req.image_urls = multimodalRefParams.image_urls
        req.video_urls = multimodalRefParams.video_urls
        req.audio_urls = multimodalRefParams.audio_urls
      }
      if (activeMode.value === 'edit_video') {
        req.ref_video_url = editVideoParams.ref_video_url
        req.ref_image_url = editVideoParams.ref_image_url
      }
      if (activeMode.value === 'extend_video') req.video_urls = extendVideoParams.video_urls
      if (activeMode.value === 'virtual_avatar') req.asset_id = virtualAvatarParams.asset_id

      const res = await httpPost('/api/seedance/task', req)
      if (res.data) {
        showMessageOK('任务提交成功')
        listFinished.value = false
        await fetchData(1)
      }
    } catch (e) {
      showMessageError(e.message || '提交失败')
    } finally {
      submitting.value = false
    }
  }

  function getParams() {
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

  const removeJob = async (item) => {
    try {
      const res = await httpGet('/api/seedance/remove', { id: item.id })
      if (res.data) {
        showMessageOK('删除成功')
        await fetchData(1)
      }
    } catch (e) {
      showMessageError('删除失败')
    }
  }

  const retryTask = async (id) => {
    try {
      const res = await httpGet(`/api/seedance/retry?id=${id}`)
      if (res.data) {
        showMessageOK('重试已提交')
        await fetchData(1)
      }
    } catch (e) {
      showMessageError('重试失败')
    }
  }

  const playVideo = (item) => {
    currentVideoUrl.value = replaceImg(item.video_url)
    showVideoDialog.value = true
  }

  return {
    activeMode, loading, submitting, currentList, listLoading, listFinished,
    isLogin, userPower, currentPrompt, powerConfig, showVideoDialog, currentVideoUrl,
    modes, ratioOptions,
    textToVideoParams, imageToVideoFirstParams, imageToVideoDualParams,
    multimodalRefParams, editVideoParams, extendVideoParams, virtualAvatarParams,
    currentMode, currentPowerCost,
    init, switchMode: (m) => { activeMode.value = m }, getModeName, getStatusText,
    fetchData, submitTask, removeJob, retryTask, playVideo,
  }
})
