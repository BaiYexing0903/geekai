import { checkSession } from '@/store/cache'
import { showMessageError, showMessageOK } from '@/utils/dialog'
import { httpGet, httpPost } from '@/utils/http'
import { replaceImg } from '@/utils/libs'
import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import { seedanceModes } from './seedanceModes'
import { splitSeedanceReferenceUrls } from '../seedanceReferences'

export const useSeedanceStore = defineStore('mobile-seedance', () => {
  const activeMode = ref('multimodal_ref')
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
  const selectedModel = ref('seedance-fast')
  const powerConfig = reactive({})
  const showVideoDialog = ref(false)
  const currentVideoUrl = ref('')

  const modes = seedanceModes

  const videoModels = [
    { label: 'Seedance Fast', value: 'seedance-fast', provider: 'seedance', model: 'fast' },
    { label: 'Seedance 2.0', value: 'seedance-standard', provider: 'seedance', model: 'standard' },
    { label: 'Veo 3.1', value: 'veo3.1-4k', provider: 'veo', model: 'veo3.1-4k' },
    { label: 'Veo Fast', value: 'veo_3_1-fast-4K', provider: 'veo', model: 'veo_3_1-fast-4K' },
  ]
  const currentModelConfig = computed(() => videoModels.find((m) => m.value === selectedModel.value) || videoModels[0])
  const isVeo = computed(() => currentModelConfig.value.provider === 'veo')

  const ratioOptions = [
    { label: '自适应', value: 'adaptive' },
    { label: '16:9', value: '16:9' },
    { label: '9:16', value: '9:16' },
    { label: '1:1', value: '1:1' },
    { label: '4:3', value: '4:3' },
    { label: '3:4', value: '3:4' },
    { label: '21:9', value: '21:9' },
  ]
  const veoRatioOptions = [
    { label: '16:9', value: '16:9' },
    { label: '9:16', value: '9:16' },
  ]
  const veoResolutionOptions = [
    { label: '720p', value: '720p' },
    { label: '1080p', value: '1080p' },
    { label: '4k', value: '4k' },
  ]

  const textToVideoParams = reactive({ model: 'fast', resolution: '720p', ratio: '16:9', duration: 5, generate_audio: true, watermark: false })
  const imageToVideoFirstParams = reactive({ model: 'fast', first_frame_url: '', resolution: '720p', ratio: 'adaptive', duration: 5, generate_audio: false, watermark: false })
  const imageToVideoDualParams = reactive({ model: 'fast', first_frame_url: '', last_frame_url: '', resolution: '720p', ratio: 'adaptive', duration: 5, generate_audio: false, watermark: false })
  const multimodalRefParams = reactive({ model: 'fast', reference_urls: [], resolution: '720p', ratio: '16:9', duration: 5, generate_audio: true, watermark: false })
  const editVideoParams = reactive({ model: 'fast', ref_video_url: '', ref_image_url: '', resolution: '720p', ratio: '16:9', duration: 5, generate_audio: true, watermark: false })
  const extendVideoParams = reactive({ model: 'fast', video_urls: [], resolution: '720p', ratio: '16:9', duration: 8, generate_audio: true, watermark: false })
  const virtualAvatarParams = reactive({ model: 'fast', asset_id: '', resolution: '720p', ratio: '16:9', duration: 5, generate_audio: true, watermark: false })
  const veoParams = reactive({ model: 'veo3.1-4k', images: [], resolution: '4k', aspect_ratio: '16:9' })

  const currentMode = computed(() => modes.find((m) => m.key === activeMode.value) || modes[0])
  const currentPowerCost = computed(() => {
    if (isVeo.value) {
      const key = `${veoParams.model}_${veoParams.resolution}`
      return powerConfig.veo_powers?.[key] || 0
    }
    const p = getParams()
    const model = p?.model || 'fast'
    const resolution = p?.resolution || '720p'
    const duration = p?.duration || 5
    const effectiveDuration = duration <= 0 ? 5 : duration
    const priceMap = model === 'standard' ? powerConfig.vip_price : powerConfig.fast_price
    const perSecond = priceMap?.[resolution] || priceMap?.['720p'] || 1
    return perSecond * effectiveDuration
  })

  const init = async () => {
    try {
      const powerRes = await httpGet('/api/seedance/power-config')
      if (powerRes.data) Object.assign(powerConfig, powerRes.data)
      const systemRes = await httpGet('/api/config/get?key=system')
      if (systemRes.data?.veo_powers) powerConfig.veo_powers = systemRes.data.veo_powers
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
    if (isVeo.value) {
      await fetchVeoData(pageNum)
      return
    }
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

  const fetchVeoData = async (pageNum = 1, mergeExisting = false) => {
    try {
      listLoading.value = true
      page.value = pageNum
      const res = await httpGet('/api/video/list', { type: 'veo', page: pageNum, page_size: pageSize.value })
      const items = (res.data?.items || []).map((item) => ({
        ...item,
        status: item.progress === 100 ? 'succeeded' : item.progress < 0 ? 'failed' : 'running',
        video_url: item.video_url || item.water_url,
      }))
      if (items.length < pageSize.value) listFinished.value = true
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
        currentList.value.push(...items)
      }
    } catch (e) {
      showMessageError('获取列表失败')
    } finally {
      listLoading.value = false
    }
  }

  const submitTask = async () => {
    if (!currentPrompt.value) {
      showMessageError('提示词不能为空')
      return
    }
    try {
      submitting.value = true
      if (isVeo.value) {
        const res = await httpPost('/api/video/veo/create', {
          model: veoParams.model,
          prompt: currentPrompt.value,
          images: veoParams.images,
          aspect_ratio: veoParams.aspect_ratio,
          resolution: veoParams.resolution,
        })
        if (res.data) {
          showMessageOK('任务提交成功')
          listFinished.value = false
          await fetchData(1)
        }
        return
      }
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
        Object.assign(req, splitSeedanceReferenceUrls(multimodalRefParams.reference_urls || []))
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
      default: return multimodalRefParams
    }
  }

  const removeJob = async (item) => {
    try {
      const res = isVeo.value
        ? await httpGet('/api/video/remove', { id: item.id })
        : await httpGet('/api/seedance/remove', { id: item.id })
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
    isLogin, userPower, currentPrompt, selectedModel, powerConfig, showVideoDialog, currentVideoUrl,
    modes, videoModels, currentModelConfig, isVeo, ratioOptions, veoRatioOptions, veoResolutionOptions,
    textToVideoParams, imageToVideoFirstParams, imageToVideoDualParams,
    multimodalRefParams, veoParams, editVideoParams, extendVideoParams, virtualAvatarParams,
    currentMode, currentPowerCost,
    init, switchMode: (m) => { activeMode.value = m }, getModeName, getStatusText,
    fetchData, submitTask, removeJob, retryTask, playVideo,
  }
})
