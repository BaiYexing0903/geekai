import { describe, expect, test } from 'vitest'
import { seedanceModes } from './seedanceModes'

describe('seedanceModes', () => {
  test('only exposes multimodal reference creation', () => {
    expect(seedanceModes).toEqual([
      { key: 'multimodal_ref', name: '多模态', icon: 'api-key', needsImage: false },
    ])
  })
})
