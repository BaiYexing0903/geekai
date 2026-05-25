import { describe, expect, it } from 'vitest'
import { buildUploadedPortrait, normalizePortraitAsset } from './seedanceReferences'

describe('normalizePortraitAsset', () => {
  it('normalizes public portrait assets', () => {
    expect(normalizePortraitAsset({
      asset_id: 'asset-public',
      asset_url: 'asset://asset-public',
      preview_url: 'https://cdn.example.com/public.jpg',
      title: '公共人像',
      metadata: { gender: '女性', age: '25', country: '中国' },
    })).toEqual({
      asset_id: 'asset-public',
      asset_url: 'asset://asset-public',
      preview_url: 'https://cdn.example.com/public.jpg',
      title: '公共人像',
      metadata: { gender: '女性', age: '25', country: '中国' },
    })
  })

  it('normalizes uploaded asset responses', () => {
    expect(normalizePortraitAsset({
      id: 'asset-uploaded',
      url: 'asset://asset-uploaded',
      preview_url: 'https://cdn.example.com/uploaded.jpg',
      name: '我的人像',
    })).toEqual({
      asset_id: 'asset-uploaded',
      asset_url: 'asset://asset-uploaded',
      preview_url: 'https://cdn.example.com/uploaded.jpg',
      title: '我的人像',
      metadata: {},
    })
  })
})

describe('buildUploadedPortrait', () => {
  it('builds the request body for Seedance asset registration', () => {
    expect(buildUploadedPortrait('https://cdn.example.com/me.jpg', '我的人像')).toEqual({
      url: 'https://cdn.example.com/me.jpg',
      name: '我的人像',
      asset_type: 'Image',
    })
  })
})
