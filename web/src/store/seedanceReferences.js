const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg']
const videoExts = ['mp4', 'webm', 'mov', 'avi', 'mkv']
const audioExts = ['mp3', 'wav', 'ogg', 'flac', 'aac']

function getUrlExt(url) {
  return url.split('?')[0].split('#')[0].split('.').pop().toLowerCase()
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
