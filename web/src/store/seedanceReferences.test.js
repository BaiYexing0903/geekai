import { describe, expect, test } from 'vitest'
import { splitSeedanceReferenceUrls } from './seedanceReferences'

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
