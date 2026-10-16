import { describe, expect, it, vi } from 'vitest'
import type { Message } from '@/stores/contacts'
import {
  albumFileType,
  buildRenderItems,
  loadFullAlbum,
  type AlbumGroup,
  type MessageRenderItem
} from './useChatAlbums'

let seq = 0
function msg(overrides: Partial<Message> = {}): Message {
  seq++
  return {
    id: `m-${seq}`,
    contact_id: 'c-1',
    direction: 'incoming',
    message_type: 'image',
    content: {},
    media_url: '/files/x',
    media_mime_type: 'image/jpeg',
    status: 'delivered',
    whatsapp_account: 'account-1',
    created_at: `2026-09-08T10:00:${String(seq % 60).padStart(2, '0')}Z`,
    updated_at: '2026-09-08T10:00:00Z',
    ...overrides
  }
}

/** A burst of files one second apart, starting at 10:00:00. */
function burst(specs: Partial<Message>[]): Message[] {
  return specs.map((spec, i) =>
    msg({ created_at: `2026-09-08T10:00:${String(i).padStart(2, '0')}Z`, ...spec })
  )
}

const pdf = (name = 'scan.pdf'): Partial<Message> => ({
  message_type: 'document',
  media_filename: name,
  media_mime_type: 'application/pdf'
})
const jpeg: Partial<Message> = { message_type: 'image', media_mime_type: 'image/jpeg' }

const render = () => true
const albums = (items: MessageRenderItem[]) =>
  items.filter((i): i is AlbumGroup => i.kind === 'album')

describe('albumFileType', () => {
  it('prefers the stored filename extension, normalising spelling variants', () => {
    expect(albumFileType({ message_type: 'document', media_filename: 'Invoice.PDF' })).toBe('pdf')
    expect(albumFileType({ message_type: 'image', media_filename: 'photo.jpg' })).toBe('jpeg')
    expect(albumFileType({ message_type: 'image', media_filename: 'photo.jpeg' })).toBe('jpeg')
    // A generic MIME (GOWA documents) never hides the real extension.
    expect(
      albumFileType({ message_type: 'document', media_filename: 'cv.docx', media_mime_type: 'application/octet-stream' })
    ).toBe('docx')
  })

  it('falls back to the MIME type, then the message type', () => {
    expect(albumFileType({ message_type: 'image', media_mime_type: 'image/jpeg' })).toBe('jpeg')
    expect(albumFileType({ message_type: 'image', media_mime_type: 'image/png' })).toBe('png')
    expect(albumFileType({ message_type: 'document', media_mime_type: 'application/pdf' })).toBe('pdf')
    expect(albumFileType({ message_type: 'sticker' })).toBe('sticker')
  })
})

describe('buildRenderItems grouping modes', () => {
  it("'all' keeps a mixed burst in ONE album (default behaviour)", () => {
    const msgs = burst([pdf(), jpeg, pdf(), jpeg])
    const groups = albums(buildRenderItems(msgs, render))
    expect(groups).toHaveLength(1)
    expect(groups[0].messages).toHaveLength(4)
    expect(groups[0].fileType).toBeUndefined()
    expect(groups[0].runStartIndex).toBe(0)
  })

  it("'by_type' splits an interleaved burst into one album per file type", () => {
    const msgs = burst([pdf('a.pdf'), jpeg, pdf('b.pdf'), jpeg, pdf('c.pdf')])
    const items = buildRenderItems(msgs, render, undefined, 'by_type')
    const groups = albums(items)

    expect(groups.map((g) => g.fileType)).toEqual(['pdf', 'jpeg'])
    expect(groups[0].messages.map((m) => m.id)).toEqual([msgs[0].id, msgs[2].id, msgs[4].id])
    expect(groups[1].messages.map((m) => m.id)).toEqual([msgs[1].id, msgs[3].id])
    // Each album renders at its own first member; both share the burst start.
    expect(groups.map((g) => g.firstIndex)).toEqual([0, 1])
    expect(groups.map((g) => g.runStartIndex)).toEqual([0, 0])
    // Items stay ordered by position.
    const firsts = items.map((i) => i.firstIndex)
    expect(firsts).toEqual([...firsts].sort((a, b) => a - b))
  })

  it("'by_type' renders a type with a single file in the burst on its own", () => {
    const msgs = burst([pdf(), pdf(), jpeg])
    const items = buildRenderItems(msgs, render, undefined, 'by_type')
    expect(albums(items)).toHaveLength(1)
    expect(albums(items)[0].messages).toHaveLength(2)
    expect(items.find((i) => i.kind === 'message')).toMatchObject({ message: { id: msgs[2].id }, firstIndex: 2 })
  })

  it("'by_type' never merges across bursts", () => {
    const first = burst([pdf(), pdf()])
    const text = msg({ message_type: 'text', media_url: undefined, created_at: '2026-09-08T10:00:05Z' })
    const second = [
      msg({ ...pdf(), created_at: '2026-09-08T10:00:06Z' }),
      msg({ ...pdf(), created_at: '2026-09-08T10:00:07Z' })
    ]
    const groups = albums(buildRenderItems([...first, text, ...second], render, undefined, 'by_type'))
    expect(groups).toHaveLength(2)
    expect(groups.map((g) => g.runStartIndex)).toEqual([0, 3])
  })

  it('reports where the burst starts so a truncated album can be detected', () => {
    const before = msg({ message_type: 'text', media_url: undefined, created_at: '2026-09-08T09:59:00Z' })
    const msgs = [before, ...burst([jpeg, jpeg, jpeg])]
    expect(albums(buildRenderItems(msgs, render))[0].runStartIndex).toBe(1)
  })
})

describe('loadFullAlbum', () => {
  /** Simulates a chat whose newest page cut a 55-file burst at 45 files. */
  function pagedChat(total: number, pageSize: number) {
    const all = Array.from({ length: total }, (_, i) =>
      msg({
        ...pdf(`scan-${i}.pdf`),
        created_at: `2026-09-08T10:${String(Math.floor(i / 60)).padStart(2, '0')}:${String(i % 60).padStart(2, '0')}Z`
      })
    )
    let loaded = all.slice(-pageSize)
    const find = (id: string) =>
      albums(buildRenderItems(loaded, render)).find((a) => a.messages.some((m) => m.id === id))
    return {
      all,
      get loaded() {
        return loaded
      },
      deps: {
        findAlbum: find,
        hasMore: () => loaded.length < all.length,
        loadOlder: vi.fn(async () => {
          loaded = all.slice(-(loaded.length + pageSize))
        }),
        messageCount: () => loaded.length
      },
      find
    }
  }

  it('pages in older history until the whole burst is loaded', async () => {
    const chat = pagedChat(55, 45)
    const partial = chat.find(chat.loaded[chat.loaded.length - 1].id)!
    expect(partial.messages).toHaveLength(45)
    expect(partial.runStartIndex).toBe(0)

    const full = await loadFullAlbum(partial, chat.deps)
    expect(full.messages).toHaveLength(55)
    expect(full.messages.map((m) => m.id)).toEqual(chat.all.map((m) => m.id))
  })

  it('does not load anything when the burst starts inside the loaded window', async () => {
    const chat = pagedChat(10, 10)
    const before = msg({ message_type: 'text', media_url: undefined, created_at: '2026-09-08T09:00:00Z' })
    const album = albums(buildRenderItems([before, ...chat.loaded], render))[0]
    const loadOlder = vi.fn(async () => {})
    const result = await loadFullAlbum(album, { ...chat.deps, hasMore: () => true, loadOlder })
    expect(loadOlder).not.toHaveBeenCalled()
    expect(result).toBe(album)
  })

  it('stops when a load adds nothing or the page budget is spent', async () => {
    const chat = pagedChat(55, 45)
    const partial = chat.find(chat.loaded[chat.loaded.length - 1].id)!

    const stuck = vi.fn(async () => {})
    await loadFullAlbum(partial, { ...chat.deps, loadOlder: stuck })
    expect(stuck).toHaveBeenCalledTimes(1)

    const grow = vi.fn(async () => {})
    let count = 45
    await loadFullAlbum(partial, {
      ...chat.deps,
      hasMore: () => true,
      messageCount: () => count++,
      loadOlder: grow,
      maxPages: 3
    })
    expect(grow).toHaveBeenCalledTimes(3)
  })
})
