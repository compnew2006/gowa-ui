import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

vi.mock('@/services/api', () => ({
  api: { post: vi.fn() },
  getRequestHeaders: vi.fn(() => ({}))
}))
vi.mock('@/lib/media', () => ({
  mediaDisplayName: vi.fn(),
  mediaUrl: vi.fn(),
  saveBlob: vi.fn()
}))
vi.mock('vue-sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn() }
}))

import { toast } from 'vue-sonner'
import { api } from '@/services/api'
import {
  MAX_BATCH_FILES,
  MAX_UPLOAD_BYTES,
  MEDIA_UPLOAD_TIMEOUT_MS,
  clipboardImageName,
  extractClipboardImages,
  uploadLimitBytes,
  useChatMedia
} from './useChatMedia'

// ─── Clipboard fakes (jsdom has no DataTransfer / ClipboardEvent) ───

function imageFile(name = 'image.png', type = 'image/png', size = 8): File {
  return new File([new Uint8Array(size)], name, { type })
}

interface FakeClipboardInit {
  types?: string[]
  files?: File[]
  items?: { kind: string; type: string; getAsFile: () => File | null }[]
  text?: string
}

function fakeClipboard(init: FakeClipboardInit): DataTransfer {
  return {
    types: init.types ?? [],
    files: init.files ?? [],
    items: init.items ?? [],
    getData: (format: string) => (format === 'text/plain' ? (init.text ?? '') : '')
  } as unknown as DataTransfer
}

function pasteEvent(data: DataTransfer | null) {
  const preventDefault = vi.fn()
  return {
    event: { clipboardData: data, preventDefault } as unknown as ClipboardEvent,
    preventDefault
  }
}

describe('extractClipboardImages', () => {
  it('returns [] when there is no clipboard data', () => {
    expect(extractClipboardImages(null)).toEqual([])
    expect(extractClipboardImages(undefined)).toEqual([])
  })

  it('takes a screenshot (files only)', () => {
    const shot = imageFile()
    expect(extractClipboardImages(fakeClipboard({ types: ['Files'], files: [shot] }))).toEqual([shot])
  })

  it('takes an image file copied in a file manager even though the name rides along as text', () => {
    const photo = imageFile('photo.jpg', 'image/jpeg')
    const data = fakeClipboard({ types: ['text/plain', 'Files'], text: 'photo.jpg', files: [photo] })
    expect(extractClipboardImages(data)).toEqual([photo])
  })

  it('takes "Copy image" from a web page (html + image, no plain text)', () => {
    const img = imageFile()
    const data = fakeClipboard({ types: ['text/html', 'Files'], files: [img] })
    expect(extractClipboardImages(data)).toEqual([img])
  })

  it('leaves rich text copies (spreadsheet cells, formatted text) as a text paste', () => {
    const rendering = imageFile()
    const data = fakeClipboard({
      types: ['text/plain', 'text/html', 'Files'],
      text: 'a\tb\nc\td',
      files: [rendering]
    })
    expect(extractClipboardImages(data)).toEqual([])
  })

  it('ignores a whitespace-only text/plain when deciding it was a rich copy', () => {
    const img = imageFile()
    const data = fakeClipboard({ types: ['text/plain', 'text/html', 'Files'], text: '  \n', files: [img] })
    expect(extractClipboardImages(data)).toEqual([img])
  })

  it('falls back to clipboardData.items when files is empty', () => {
    const img = imageFile()
    const data = fakeClipboard({
      types: ['Files'],
      items: [
        { kind: 'string', type: 'text/plain', getAsFile: () => null },
        { kind: 'file', type: 'image/png', getAsFile: () => img },
        { kind: 'file', type: 'image/png', getAsFile: () => null }
      ]
    })
    expect(extractClipboardImages(data)).toEqual([img])
  })

  it('ignores non-image files', () => {
    const pdf = new File(['%PDF'], 'a.pdf', { type: 'application/pdf' })
    expect(extractClipboardImages(fakeClipboard({ types: ['Files'], files: [pdf] }))).toEqual([])
  })

  it('returns [] for a plain text paste', () => {
    expect(extractClipboardImages(fakeClipboard({ types: ['text/plain'], text: 'hello' }))).toEqual([])
  })
})

describe('clipboardImageName', () => {
  const now = new Date(2026, 9, 9, 14, 3, 7) // local time: 2026-10-09 14:03:07

  it('builds a sortable timestamped name with the extension of the MIME type', () => {
    expect(clipboardImageName(imageFile('image.png', 'image/png'), 0, now)).toBe('image-20261009-140307.png')
    expect(clipboardImageName(imageFile('x', 'image/jpeg'), 0, now)).toBe('image-20261009-140307.jpg')
    expect(clipboardImageName(imageFile('x', 'image/webp'), 0, now)).toBe('image-20261009-140307.webp')
  })

  it('disambiguates several images pasted at once', () => {
    expect(clipboardImageName(imageFile(), 1, now)).toBe('image-20261009-140307-2.png')
    expect(clipboardImageName(imageFile(), 2, now)).toBe('image-20261009-140307-3.png')
  })

  it('falls back to png for an unknown image type', () => {
    expect(clipboardImageName(imageFile('x', 'image/heic'), 0, now)).toBe('image-20261009-140307.png')
  })
})

function setup(opts: { contact?: { id: string } | null } = {}) {
  const contactsStore = {
    currentContact: opts.contact === undefined ? { id: 'c1' } : opts.contact,
    messages: [],
    addMessage: vi.fn()
  }
  const media = useChatMedia({
    // Echo key and params so assertions can see what would be rendered.
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key} ${JSON.stringify(params)}` : key,
    contactsStore,
    selectedAccount: { value: null },
    scrollToBottom: vi.fn(),
    sendStatusMedia: vi.fn(),
    isStatusContact: () => false,
    mediaExport: {
      redownloading: { value: new Set<string>() },
      redownload: vi.fn()
    },
    fileInputRef: ref(null)
  })
  return media
}

describe('useChatMedia handlePaste', () => {
  beforeEach(() => {
    URL.createObjectURL = vi.fn(() => 'blob:preview')
    URL.revokeObjectURL = vi.fn()
  })

  it('opens the media dialog with the pasted image queued and claims the event', () => {
    const media = setup()
    const shot = imageFile()
    const { event, preventDefault } = pasteEvent(fakeClipboard({ types: ['Files'], files: [shot] }))

    media.handlePaste(event)

    expect(preventDefault).toHaveBeenCalledTimes(1)
    expect(media.isMediaDialogOpen.value).toBe(true)
    expect(media.selectedFiles.value).toHaveLength(1)
    expect(media.selectedFiles.value[0].name).toMatch(/^image-\d{8}-\d{6}\.png$/)
    expect(media.selectedFiles.value[0].type).toBe('image/png')
    expect(media.filePreviewUrl.value).toBe('blob:preview')
    expect(media.mediaCaption.value).toBe('')
  })

  it('does not touch a text paste', () => {
    const media = setup()
    const { event, preventDefault } = pasteEvent(fakeClipboard({ types: ['text/plain'], text: 'hi' }))

    media.handlePaste(event)

    expect(preventDefault).not.toHaveBeenCalled()
    expect(media.isMediaDialogOpen.value).toBe(false)
    expect(media.selectedFiles.value).toHaveLength(0)
  })

  it('does not touch a rich text copy that carries an image rendering', () => {
    const media = setup()
    const { event, preventDefault } = pasteEvent(
      fakeClipboard({ types: ['text/plain', 'text/html', 'Files'], text: 'x', files: [imageFile()] })
    )

    media.handlePaste(event)

    expect(preventDefault).not.toHaveBeenCalled()
    expect(media.isMediaDialogOpen.value).toBe(false)
  })

  it('adds to the queue, keeping the caption, when the dialog is already open', () => {
    const media = setup()
    media.handlePaste(pasteEvent(fakeClipboard({ types: ['Files'], files: [imageFile()] })).event)
    media.mediaCaption.value = 'order #42'

    media.handlePaste(pasteEvent(fakeClipboard({ types: ['Files'], files: [imageFile()] })).event)

    expect(media.selectedFiles.value).toHaveLength(2)
    expect(media.mediaCaption.value).toBe('order #42')
    expect(media.activeFileIndex.value).toBe(1)
    expect(URL.revokeObjectURL).not.toHaveBeenCalled()
  })

  it('never gives two queued images the same name, even within the same second', () => {
    vi.useFakeTimers({ now: new Date(2026, 9, 9, 14, 3, 7) })
    try {
      const media = setup()
      const paste = () =>
        media.handlePaste(pasteEvent(fakeClipboard({ types: ['Files'], files: [imageFile()] })).event)

      paste()
      paste()
      paste()
      expect(media.selectedFiles.value.map((f) => f.name)).toEqual([
        'image-20261009-140307.png',
        'image-20261009-140307-2.png',
        'image-20261009-140307-3.png'
      ])

      // Dropping the first and pasting again must not collide with the survivors.
      media.removeFile(0)
      paste()
      const names = media.selectedFiles.value.map((f) => f.name)
      expect(new Set(names).size).toBe(names.length)
      expect(names).toHaveLength(3)
    } finally {
      vi.useRealTimers()
    }
  })

  it('refuses to grow the queue past the batch limit', () => {
    const media = setup()
    const many = Array.from({ length: MAX_BATCH_FILES }, () => imageFile())
    media.handlePaste(pasteEvent(fakeClipboard({ types: ['Files'], files: many })).event)
    expect(media.selectedFiles.value).toHaveLength(MAX_BATCH_FILES)

    media.handlePaste(pasteEvent(fakeClipboard({ types: ['Files'], files: [imageFile()] })).event)

    expect(media.selectedFiles.value).toHaveLength(MAX_BATCH_FILES)
    expect(toast.error).toHaveBeenCalledWith('chat.tooManyFiles {"max":10}')
  })

  it('rejects an image over the image limit without opening the dialog', () => {
    const media = setup()
    // A real buffer, not a patched size: the handler renames by copying the
    // File, which re-reads the true size.
    const huge = imageFile('big.png', 'image/png', MAX_UPLOAD_BYTES.image + 1)

    media.handlePaste(pasteEvent(fakeClipboard({ types: ['Files'], files: [huge] })).event)

    expect(media.isMediaDialogOpen.value).toBe(false)
    expect(media.selectedFiles.value).toHaveLength(0)
    expect(toast.error).toHaveBeenCalledWith(
      'chat.fileTooLarge',
      expect.objectContaining({
        description: expect.stringMatching(/^chat\.fileSizeOverLimit .*"name":"image-.*\.png".*"max":"20"/)
      })
    )
  })

  it('leaves the queue alone while a batch is uploading', () => {
    const media = setup()
    media.handlePaste(pasteEvent(fakeClipboard({ types: ['Files'], files: [imageFile()] })).event)
    media.isUploadingMedia.value = true

    const { event, preventDefault } = pasteEvent(fakeClipboard({ types: ['Files'], files: [imageFile()] }))
    media.handlePaste(event)

    expect(preventDefault).toHaveBeenCalled()
    expect(media.selectedFiles.value).toHaveLength(1)
  })

  it('queues nothing when no conversation is open', () => {
    const media = setup({ contact: null })

    media.handlePaste(pasteEvent(fakeClipboard({ types: ['Files'], files: [imageFile()] })).event)

    expect(media.isMediaDialogOpen.value).toBe(false)
    expect(media.selectedFiles.value).toHaveLength(0)
  })

  it('keeps the file picker path working (replace mode)', () => {
    const media = setup()
    const input = { files: [imageFile('a.png'), imageFile('b.png')], value: 'x' }

    media.handleFileSelect({ target: input } as unknown as Event)

    expect(media.selectedFiles.value.map((f) => f.name)).toEqual(['a.png', 'b.png'])
    expect(media.isMediaDialogOpen.value).toBe(true)
    expect(input.value).toBe('')
  })
})

// ─── Upload limits ───

/** A File that reports `size` without allocating it (handleFileSelect keeps the objects as picked). */
function sizedFile(name: string, type: string, size: number): File {
  const file = new File([], name, { type })
  Object.defineProperty(file, 'size', { value: size })
  return file
}

function pick(media: ReturnType<typeof setup>, ...files: File[]) {
  media.handleFileSelect({ target: { files, value: 'x' } } as unknown as Event)
}

describe('upload limits', () => {
  beforeEach(() => {
    URL.createObjectURL = vi.fn(() => 'blob:preview')
    URL.revokeObjectURL = vi.fn()
  })

  // Mirrors GOWA's own caps (src/config/settings.go), counted in decimal MB.
  it.each([
    ['image', 'image/png', 20_000_000],
    ['video', 'video/mp4', 100_000_000],
    ['audio', 'audio/mpeg', 50_000_000],
    ['pdf', 'application/pdf', 50_000_000],
    ['zip', 'application/zip', 50_000_000]
  ])('accepts a %s of exactly the limit and rejects one byte more', (_label, type, limit) => {
    const media = setup()

    pick(media, sizedFile('at-limit', type, limit))
    expect(media.selectedFiles.value).toHaveLength(1)
    expect(toast.error).not.toHaveBeenCalled()

    const over = setup()
    pick(over, sizedFile('over-limit', type, limit + 1))
    expect(over.selectedFiles.value).toHaveLength(0)
    expect(over.isMediaDialogOpen.value).toBe(false)
    expect(toast.error).toHaveBeenCalledWith('chat.fileTooLarge', expect.anything())
  })

  it('no longer stops videos and audio at 16MB', () => {
    const media = setup()
    pick(media, sizedFile('clip.mp4', 'video/mp4', 30 * 1_000_000), sizedFile('talk.mp3', 'audio/mpeg', 25 * 1_000_000))

    expect(media.selectedFiles.value.map((f) => f.name)).toEqual(['clip.mp4', 'talk.mp3'])
    expect(toast.error).not.toHaveBeenCalled()
  })

  it('names the file, its size and the limit that applies to it', () => {
    const media = setup()
    pick(media, sizedFile('big.mp4', 'video/mp4', 101_234_567), sizedFile('huge.png', 'image/png', 35_234_000))

    expect(toast.error).toHaveBeenCalledWith('chat.fileTooLarge', {
      description:
        'chat.fileSizeOverLimit {"name":"big.mp4","size":"101.3","max":"100"}, ' +
        'chat.fileSizeOverLimit {"name":"huge.png","size":"35.3","max":"20"}'
    })
  })

  it('never reads an oversized file as equal to its limit', () => {
    const media = setup()
    pick(media, sizedFile('x.png', 'image/png', 20_000_001))

    expect(toast.error).toHaveBeenCalledWith('chat.fileTooLarge', {
      description: 'chat.fileSizeOverLimit {"name":"x.png","size":"20.1","max":"20"}'
    })
  })

  it('still rejects unsupported types regardless of size', () => {
    const media = setup()
    pick(media, sizedFile('evil.exe', 'application/x-msdownload', 10))

    expect(media.selectedFiles.value).toHaveLength(0)
    expect(toast.error).toHaveBeenCalledWith('chat.unsupportedFileType', { description: 'evil.exe' })
  })

  it('keeps every limit inside the server request body cap (110MB)', () => {
    // cmd/gowa-ui/wiring.go MaxRequestBodySize; multipart framing adds a little.
    const serverBodyLimit = 110 * 1024 * 1024
    for (const limit of Object.values(MAX_UPLOAD_BYTES)) {
      expect(limit + 1_000_000).toBeLessThan(serverBodyLimit)
    }
  })

  it('picks the limit from the message type the file will be sent as', () => {
    expect(uploadLimitBytes('video/quicktime')).toBe(MAX_UPLOAD_BYTES.video)
    expect(uploadLimitBytes('image/webp')).toBe(MAX_UPLOAD_BYTES.image)
    expect(uploadLimitBytes('')).toBe(MAX_UPLOAD_BYTES.document)
  })

  it('uploads with a timeout long enough for a large file, not the 30s default', async () => {
    vi.mocked(api.post).mockResolvedValue({ data: {} })
    const media = setup()
    pick(media, sizedFile('clip.mp4', 'video/mp4', 60 * 1_000_000))

    await media.sendMediaMessage()

    expect(api.post).toHaveBeenCalledTimes(1)
    expect(api.post).toHaveBeenCalledWith(
      '/messages/media',
      expect.any(FormData),
      expect.objectContaining({ timeout: MEDIA_UPLOAD_TIMEOUT_MS })
    )
    expect(MEDIA_UPLOAD_TIMEOUT_MS).toBeGreaterThan(15 * 60 * 1000) // above the gateway's media deadline
  })
})
