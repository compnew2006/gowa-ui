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
import {
  MAX_BATCH_FILES,
  clipboardImageName,
  extractClipboardImages,
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

describe('useChatMedia handlePaste', () => {
  function setup(opts: { contact?: { id: string } | null } = {}) {
    const contactsStore = {
      currentContact: opts.contact === undefined ? { id: 'c1' } : opts.contact,
      messages: [],
      addMessage: vi.fn()
    }
    const media = useChatMedia({
      t: (key: string) => key,
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
    expect(toast.error).toHaveBeenCalledWith('chat.tooManyFiles')
  })

  it('rejects an oversized image without opening the dialog', () => {
    const media = setup()
    // Over WhatsApp's 16MB media cap. A real 17MB buffer, not a patched size:
    // the handler renames by copying the File, which re-reads the true size.
    const huge = imageFile('big.png', 'image/png', 17 * 1024 * 1024)

    media.handlePaste(pasteEvent(fakeClipboard({ types: ['Files'], files: [huge] })).event)

    expect(media.isMediaDialogOpen.value).toBe(false)
    expect(media.selectedFiles.value).toHaveLength(0)
    expect(toast.error).toHaveBeenCalledWith(
      'chat.fileTooLarge',
      expect.objectContaining({ description: expect.stringMatching(/^image-/) })
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
