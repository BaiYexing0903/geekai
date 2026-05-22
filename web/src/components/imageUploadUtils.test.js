import { describe, expect, test } from 'vitest'
import { isImageFileTooLarge, normalizeImageModelValue } from './imageUploadUtils'

describe('normalizeImageModelValue', () => {
  test('treats an empty array as empty in single image mode', () => {
    expect(normalizeImageModelValue([], false, 1)).toEqual([])
  })

  test('uses the first non-empty array item in single image mode', () => {
    expect(normalizeImageModelValue(['', 'https://cdn.example.com/a.png'], false, 1)).toEqual(['https://cdn.example.com/a.png'])
  })
})

describe('isImageFileTooLarge', () => {
  test('uses the configured max size in megabytes', () => {
    expect(isImageFileTooLarge({ size: 6 * 1024 * 1024 }, 5)).toBe(true)
    expect(isImageFileTooLarge({ size: 6 * 1024 * 1024 }, 50)).toBe(false)
  })
})
