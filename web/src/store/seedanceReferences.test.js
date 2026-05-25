import { describe, expect, test } from 'vitest'
import {
  buildSeedanceMentionOptions,
  splitSeedanceReferenceUrls,
  transformSeedancePromptMentions,
} from './seedanceReferences'

describe('splitSeedanceReferenceUrls', () => {
  test('groups reference urls by media type', () => {
    const result = splitSeedanceReferenceUrls([
      'https://cdn.example.com/a.png',
      'https://cdn.example.com/b.MP4?token=1',
      'https://cdn.example.com/c.mp3',
      'https://cdn.example.com/d.webp',
      'https://cdn.example.com/e.wav?x=1',
      'https://cdn.example.com/f.mov',
    ])

    expect(result).toEqual({
      image_urls: [
        'https://cdn.example.com/a.png',
        'https://cdn.example.com/d.webp',
      ],
      video_urls: [
        'https://cdn.example.com/b.MP4?token=1',
        'https://cdn.example.com/f.mov',
      ],
      audio_urls: [
        'https://cdn.example.com/c.mp3',
        'https://cdn.example.com/e.wav?x=1',
      ],
    })
  })
})

describe('buildSeedanceMentionOptions', () => {
  test('returns media mention options in upload order with per-type counters', () => {
    const urls = [
      'https://cdn.example.com/image-a.png',
      'https://cdn.example.com/video-a.mp4?token=1',
      'https://cdn.example.com/audio-a.wav#clip',
      'https://cdn.example.com/image-b.WEBP',
      'https://cdn.example.com/unknown.txt',
    ]

    expect(buildSeedanceMentionOptions(urls)).toEqual([
      {
        label: '@图片1',
        replacement: '第1张图片',
        description: '图片1',
        type: 'image',
        index: 1,
        url: 'https://cdn.example.com/image-a.png',
      },
      {
        label: '@视频1',
        replacement: '第1个视频',
        description: '视频1',
        type: 'video',
        index: 1,
        url: 'https://cdn.example.com/video-a.mp4?token=1',
      },
      {
        label: '@音频1',
        replacement: '第1段音频',
        description: '音频1',
        type: 'audio',
        index: 1,
        url: 'https://cdn.example.com/audio-a.wav#clip',
      },
      {
        label: '@图片2',
        replacement: '第2张图片',
        description: '图片2',
        type: 'image',
        index: 2,
        url: 'https://cdn.example.com/image-b.WEBP',
      },
    ])
  })
})

describe('transformSeedancePromptMentions', () => {
  test('prepends resource instructions and transforms image video and audio mentions', () => {
    const result = transformSeedancePromptMentions(
      '参考 @图片1 的色彩、@视频1 的运镜和 @音频1 的节奏',
      [
        'https://cdn.example.com/source.png',
        'https://cdn.example.com/motion.mp4',
        'https://cdn.example.com/music.mp3',
      ],
    )

    expect(result).toBe([
      '资源说明：',
      '第1张图片对应用户提示词中的“@图片1”。',
      '第1个视频对应用户提示词中的“@视频1”。',
      '第1段音频对应用户提示词中的“@音频1”。',
      '',
      '用户要求：',
      '参考 第1张图片 的色彩、第1个视频 的运镜和 第1段音频 的节奏',
    ].join('\n'))
  })

  test('returns prompt unchanged when no mentions are used', () => {
    const prompt = '生成一段城市夜景视频'

    expect(transformSeedancePromptMentions(prompt, [
      'https://cdn.example.com/source.png',
    ])).toBe(prompt)
  })

  test('returns prompt unchanged when only missing mentions are used', () => {
    const prompt = '参考 @图片2 的构图'

    expect(transformSeedancePromptMentions(prompt, [
      'https://cdn.example.com/source.png',
    ])).toBe(prompt)
  })

  test('replaces all occurrences of a repeated mention', () => {
    const result = transformSeedancePromptMentions(
      '@图片1 作为开头，结尾也呼应 @图片1',
      ['https://cdn.example.com/source.png'],
    )

    expect(result).toBe([
      '资源说明：',
      '第1张图片对应用户提示词中的“@图片1”。',
      '',
      '用户要求：',
      '第1张图片 作为开头，结尾也呼应 第1张图片',
    ].join('\n'))
  })

  test('leaves longer missing image mentions unchanged when shorter mention exists', () => {
    const prompt = '参考@图片10。'

    expect(transformSeedancePromptMentions(prompt, [
      'https://cdn.example.com/source.png',
    ])).toBe(prompt)
  })

  test('replaces two digit image mentions without prefix corruption', () => {
    const result = transformSeedancePromptMentions(
      '参考@图片10，也参考@图片1。',
      [
        'https://cdn.example.com/image-1.png',
        'https://cdn.example.com/image-2.png',
        'https://cdn.example.com/image-3.png',
        'https://cdn.example.com/image-4.png',
        'https://cdn.example.com/image-5.png',
        'https://cdn.example.com/image-6.png',
        'https://cdn.example.com/image-7.png',
        'https://cdn.example.com/image-8.png',
        'https://cdn.example.com/image-9.png',
        'https://cdn.example.com/image-10.png',
      ],
    )

    expect(result).toBe([
      '资源说明：',
      '第1张图片对应用户提示词中的“@图片1”。',
      '第10张图片对应用户提示词中的“@图片10”。',
      '',
      '用户要求：',
      '参考第10张图片，也参考第1张图片。',
    ].join('\n'))
  })
})
