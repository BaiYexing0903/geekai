const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg']
const videoExts = ['mp4', 'webm', 'mov', 'avi', 'mkv']
const audioExts = ['mp3', 'wav', 'ogg', 'flac', 'aac']

const mentionConfig = {
  image: { label: '图片', replacementUnit: '张图片' },
  video: { label: '视频', replacementUnit: '个视频' },
  audio: { label: '音频', replacementUnit: '段音频' },
}

function getResourceFileName(url) {
  return decodeURIComponent(url.split('?')[0].split('#')[0].split('/').pop() || '')
}

function getUrlExt(url) {
  return getResourceFileName(url).split('.').pop().toLowerCase()
}

export function splitSeedanceReferenceUrls(urls) {
  return urls.reduce((result, url) => {
    const ext = getUrlExt(url)
    if (imageExts.includes(ext)) result.image_urls.push(url)
    if (videoExts.includes(ext)) result.video_urls.push(url)
    if (audioExts.includes(ext)) result.audio_urls.push(url)
    return result
  }, { image_urls: [], video_urls: [], audio_urls: [] })
}

function getMediaType(ext) {
  if (imageExts.includes(ext)) return 'image'
  if (videoExts.includes(ext)) return 'video'
  if (audioExts.includes(ext)) return 'audio'
  return ''
}

export function buildSeedanceMentionOptions(urls) {
  const counters = { image: 0, video: 0, audio: 0 }

  return urls.reduce((options, url) => {
    const type = getMediaType(getUrlExt(url))
    if (!type) return options

    counters[type] += 1
    const index = counters[type]
    const config = mentionConfig[type]

    options.push({
      label: `@${config.label}${index}`,
      replacement: `第${index}${config.replacementUnit}`,
      description: `${config.label}${index} · ${getResourceFileName(url)}`,
      type,
      index,
      url,
    })

    return options
  }, [])
}

export function transformSeedancePromptMentions(prompt, urls) {
  const mentionOptions = buildSeedanceMentionOptions(urls)
  if (!mentionOptions.length) return prompt

  const optionMap = new Map(mentionOptions.map(option => [option.label, option]))
  const usedLabels = new Set()
  const transformedPrompt = prompt.replace(/@(图片|视频|音频)\d+/g, (match) => {
    const option = optionMap.get(match)
    if (!option) return match

    usedLabels.add(option.label)
    return option.replacement
  })

  if (!usedLabels.size) return prompt

  const resourceInstructions = mentionOptions
    .filter(option => usedLabels.has(option.label))
    .map(option => `${option.replacement}对应用户提示词中的“${option.label}”。`)

  return [
    '资源说明：',
    ...resourceInstructions,
    '',
    '用户要求：',
    transformedPrompt,
  ].join('\n')
}
