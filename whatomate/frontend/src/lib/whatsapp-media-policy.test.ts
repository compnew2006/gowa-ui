import { describe, expect, it } from 'vitest'

import {
  getWhatsAppMediaMaxSizeBytes,
  resolveWhatsAppMediaCategory,
  resolveWhatsAppMediaCategoryForFile,
  validateWhatsAppMediaFile,
} from './whatsapp-media-policy'

describe('whatsapp-media-policy', () => {
  it('derives category from explicit MIME type', () => {
    expect(resolveWhatsAppMediaCategory({ mimeType: 'image/jpeg' })).toBe('image')
    expect(resolveWhatsAppMediaCategory({ mimeType: 'video/3gpp' })).toBe('video')
    expect(resolveWhatsAppMediaCategory({ mimeType: 'audio/mpeg' })).toBe('audio')
    expect(resolveWhatsAppMediaCategory({ mimeType: 'application/pdf' })).toBe('document')
  })

  it('falls back to extension when MIME is empty or generic octet-stream', () => {
    expect(resolveWhatsAppMediaCategory({ mimeType: '', filename: 'photo.jpeg' })).toBe('image')
    expect(
      resolveWhatsAppMediaCategory({
        mimeType: 'application/octet-stream',
        filename: 'voice.mp3',
      }),
    ).toBe('audio')
    expect(
      resolveWhatsAppMediaCategory({
        mimeType: 'application/octet-stream; charset=binary',
        filename: 'clip.mp4',
      }),
    ).toBe('video')
  })

  it('treats unknown MIME as document without trusting extension override', () => {
    expect(
      resolveWhatsAppMediaCategory({
        mimeType: 'application/x-custom-binary',
        filename: 'image.jpg',
      }),
    ).toBe('document')
  })

  it('falls back to document when MIME and extension are unknown', () => {
    expect(resolveWhatsAppMediaCategory({ mimeType: '', filename: 'archive.unknown' })).toBe('document')
  })

  it('resolves category directly from file objects', () => {
    expect(
      resolveWhatsAppMediaCategoryForFile({
        name: 'sound.ogg',
        type: 'application/octet-stream',
      } as Pick<File, 'name' | 'type'>),
    ).toBe('audio')
  })

  it('enforces size limits per derived category', () => {
    const imageLimit = getWhatsAppMediaMaxSizeBytes('image')
    const documentLimit = getWhatsAppMediaMaxSizeBytes('document')

    const validImage = validateWhatsAppMediaFile({
      name: 'photo.jpg',
      type: 'image/jpeg',
      size: imageLimit,
    } as Pick<File, 'name' | 'type' | 'size'>)
    expect(validImage.category).toBe('image')
    expect(validImage.isValid).toBe(true)

    const oversizedImage = validateWhatsAppMediaFile({
      name: 'photo.jpg',
      type: 'image/jpeg',
      size: imageLimit + 1,
    } as Pick<File, 'name' | 'type' | 'size'>)
    expect(oversizedImage.category).toBe('image')
    expect(oversizedImage.isValid).toBe(false)
    expect(oversizedImage.errorCode).toBe('FILE_TOO_LARGE')

    const oversizedDocument = validateWhatsAppMediaFile({
      name: 'report.zip',
      type: 'application/zip',
      size: documentLimit + 1,
    } as Pick<File, 'name' | 'type' | 'size'>)
    expect(oversizedDocument.category).toBe('document')
    expect(oversizedDocument.isValid).toBe(false)
    expect(oversizedDocument.maxSizeBytes).toBe(documentLimit)
  })
})
