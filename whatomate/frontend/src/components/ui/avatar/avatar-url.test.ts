import { describe, expect, it } from 'vitest'

import { isExpiredWhatsAppAvatarURL, normalizeRenderableAvatarURL } from './avatar-url'

describe('avatar-url helpers', () => {
  it('treats expired WhatsApp avatar URLs as non-renderable', () => {
    const expired = 'https://pps.whatsapp.net/v/t61.24694-24/demo.jpg?oe=00000001'
    expect(normalizeRenderableAvatarURL(expired)).toBe('')
  })

  it('keeps future WhatsApp avatar URLs renderable', () => {
    const unexpired = 'https://pps.whatsapp.net/v/t61.24694-24/demo.jpg?oe=FFFFFFFF'
    expect(normalizeRenderableAvatarURL(unexpired)).toBe(unexpired)
  })

  it('only applies WhatsApp expiry parsing to the WhatsApp host', () => {
    const nonWhatsAppURL = 'https://example.com/avatar.jpg?oe=00000001'
    expect(normalizeRenderableAvatarURL(nonWhatsAppURL)).toBe(nonWhatsAppURL)
  })

  it('supports deterministic expiry checks with explicit now values', () => {
    const url = 'https://pps.whatsapp.net/v/t61.24694-24/demo.jpg?oe=00000020'
    expect(isExpiredWhatsAppAvatarURL(url, 0)).toBe(false)
    expect(isExpiredWhatsAppAvatarURL(url, 1000 * 40)).toBe(true)
  })
})
