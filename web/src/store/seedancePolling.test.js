import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import assert from 'node:assert/strict'

const desktopStore = readFileSync(new URL('./seedance.js', import.meta.url), 'utf8')
const mobileStore = readFileSync(new URL('./mobile/seedance.js', import.meta.url), 'utf8')

test('Veo polling refreshes silently on desktop', () => {
  assert.match(desktopStore, /fetchVeoData\(1, true, true\)/)
})

test('Veo polling refreshes silently on mobile', () => {
  assert.match(mobileStore, /fetchVeoData\(1, true, true\)/)
})
