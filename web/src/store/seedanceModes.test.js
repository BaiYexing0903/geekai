import { describe, expect, test } from 'vitest'
import { seedanceModes } from './seedanceModes'

describe('seedanceModes', () => {
  test('exposes multimodal reference and dual-frame creation only', () => {
    expect(seedanceModes).toEqual([
      { key: 'multimodal_ref', name: '多模态', icon: 'api-key', needsImage: false },
      { key: 'image_to_video_dual', name: '首尾帧', icon: 'image', needsImage: true },
    ])
  })
})
