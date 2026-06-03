import { describe, expect, it, vi } from 'vitest'
import { buildUploadedPortrait, normalizePortraitAsset, waitForUploadedPortraitActive } from './seedanceReferences'

describe('waitForUploadedPortraitActive', () => {
  it('polls Seedance asset status until the uploaded portrait is active', async () => {
    vi.useFakeTimers()
    const fetchAsset = vi.fn().mockResolvedValue({
      id: 'asset-uploaded',
      status: 'Active',
      asset_type: 'Video',
    })
    const waiting = waitForUploadedPortraitActive(fetchAsset, {
      asset_id: 'asset-uploaded',
      asset_url: 'asset://asset-uploaded',
      preview_url: 'https://cdn.example.com/me.mp4',
      title: '我的视频人像',
      metadata: {},
      status: 'Processing',
      asset_type: 'Video',
    }, { maxAttempts: 2, interval: 1000 })

    await vi.advanceTimersByTimeAsync(1000)
    await expect(waiting).resolves.toEqual({
      asset_id: 'asset-uploaded',
      asset_url: 'asset://asset-uploaded',
      preview_url: 'https://cdn.example.com/me.mp4',
      title: '我的视频人像',
      metadata: {},
      status: 'Active',
      asset_type: 'Video',
    })
    expect(fetchAsset).toHaveBeenCalledOnce()
    vi.useRealTimers()
  })
})

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

  it('normalizes Seedance API asset fields without using the real image URL as asset URL', () => {
    expect(normalizePortraitAsset({
      Id: 'asset-uploaded-api',
      URL: 'https://cdn.example.com/real-person.jpg',
      Name: '接口人像',
    })).toEqual({
      asset_id: 'asset-uploaded-api',
      asset_url: 'asset://asset-uploaded-api',
      preview_url: 'https://cdn.example.com/real-person.jpg',
      title: '接口人像',
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

  it('builds the request body for Seedance video asset registration', () => {
    expect(buildUploadedPortrait('https://cdn.example.com/me.mp4', '我的视频人像')).toEqual({
      url: 'https://cdn.example.com/me.mp4',
      name: '我的视频人像',
      asset_type: 'Video',
    })
  })

  it('keeps explicit asset type when registering a portrait asset', () => {
    expect(buildUploadedPortrait('https://cdn.example.com/me.jpg', '我的人像', 'Video')).toEqual({
      url: 'https://cdn.example.com/me.jpg',
      name: '我的人像',
      asset_type: 'Video',
    })
  })
})
