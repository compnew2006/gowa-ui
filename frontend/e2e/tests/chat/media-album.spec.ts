import { test, expect, Page, Route } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

/**
 * WhatsApp-style media albums: consecutive caption-less FILE messages (image,
 * document, sticker — audio and video NEVER join) from one sender arrive as
 * individual rows (GOWA has no album payload) and must collapse into ONE grid
 * bubble. Tapping a tile opens that specific file. Grouping rules (see
 * useChatAlbums): same direction + account + sender, no caption, not a reply,
 * ≤2 MINUTES between each two adjacent members, same day, ≥2 members.
 *
 * Uses route interception (same harness as account-tabs.spec.ts) — no real
 * media or backend rows are needed.
 */

const CONTACT_ID = '00000000-0000-0000-0000-0000000000a1'

function makeMessage(overrides: Record<string, any> = {}) {
  return {
    id: crypto.randomUUID(),
    contact_id: CONTACT_ID,
    direction: 'incoming',
    message_type: 'image',
    content: {},
    media_url: '/files/img.jpg',
    media_mime_type: 'image/jpeg',
    status: 'delivered',
    whatsapp_account: 'account-1',
    created_at: '2026-09-08T10:00:00Z',
    updated_at: '2026-09-08T10:00:00Z',
    ...overrides,
  }
}

/** A burst of caption-less files sent together — one album (one member per
 * `startSeconds` step; keep steps well under the 2-minute gap). */
function albumBurst(count = 4, startSeconds = 0, type = 'image'): any[] {
  return Array.from({ length: count }, (_, i) =>
    makeMessage({
      message_type: type,
      media_filename: type === 'document' ? `doc-${i}.pdf` : undefined,
      created_at: `2026-09-08T10:00:${String(startSeconds + i).padStart(2, '0')}Z`,
    })
  )
}

const CONTACT = {
  id: CONTACT_ID,
  phone_number: '+1234567890',
  name: 'Album Sender',
  profile_name: 'Album Sender',
  status: 'active',
  unread_count: 0,
  whatsapp_account: 'account-1',
  created_at: '2026-09-08T09:00:00Z',
  updated_at: '2026-09-08T09:00:00Z',
}

// 1×1 transparent PNG so tile <img> loads instead of falling to the
// broken-media retry button.
const TINY_PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
  'base64'
)

async function setupMockRoutes(page: Page, messages: any[]) {
  await page.route('**/api/accounts', async (route: Route) => {
    await route.fulfill({ json: { status: 'success', data: { accounts: [{ name: 'account-1', id: '1' }] } } })
  })
  await page.route('**/api/contacts?*', async (route: Route) => {
    await route.fulfill({
      json: { status: 'success', data: { contacts: [CONTACT], total: 1, page: 1, limit: 50 } },
    })
  })
  await page.route(`**/api/contacts/${CONTACT_ID}`, async (route: Route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ json: { status: 'success', data: CONTACT } })
    } else {
      await route.continue()
    }
  })
  await page.route(`**/api/contacts/${CONTACT_ID}/messages*`, async (route: Route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        json: { status: 'success', data: { messages, total: messages.length, page: 1, limit: 50, has_more: false } },
      })
    } else {
      await route.continue()
    }
  })
  await page.route(`**/api/contacts/${CONTACT_ID}/session`, async (route: Route) => {
    await route.fulfill({ json: { status: 'success', data: null } })
  })
  // Per-message media endpoint (tiles + previews point here): tiny valid PNG.
  await page.route('**/api/media/**', async (route: Route) => {
    await route.fulfill({ body: TINY_PNG, contentType: 'image/png' })
  })
}

test.describe('Media Albums', () => {
  let chatPage: ChatPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatPage = new ChatPage(page)
  })

  test('consecutive burst collapses into one album bubble with per-photo anchors', async ({ page }) => {
    const burst = albumBurst(4)
    await setupMockRoutes(page, burst)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    // ONE album grid bubble, four tiles.
    const album = page.locator('[data-album]')
    await expect(album).toHaveCount(1)
    await expect(album.locator('[data-message-id]')).toHaveCount(4)

    // Every member keeps its scroll anchor (#message-{id}) — the first on the
    // album row, continuations as zero-height anchors inside it.
    for (const m of burst) {
      await expect(page.locator(`#message-${m.id}`)).toHaveCount(1)
    }
  })

  test('a captioned photo never joins an album', async ({ page }) => {
    // img, captioned-img, img → three separate single bubbles, no album.
    const msgs = [
      makeMessage({ created_at: '2026-09-08T10:00:00Z' }),
      makeMessage({ content: { body: 'with caption' }, created_at: '2026-09-08T10:00:01Z' }),
      makeMessage({ created_at: '2026-09-08T10:00:02Z' }),
    ]
    await setupMockRoutes(page, msgs)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    await expect(page.locator('[data-album]')).toHaveCount(0)
    for (const m of msgs) {
      await expect(page.locator(`#message-${m.id} .chat-bubble`)).toHaveCount(1)
    }
  })

  test('photos up to two minutes apart still group', async ({ page }) => {
    const msgs = [
      makeMessage({ created_at: '2026-09-08T10:00:00Z' }),
      makeMessage({ created_at: '2026-09-08T10:01:30Z' }), // 90s later
      makeMessage({ created_at: '2026-09-08T10:03:00Z' }), // 90s later again
    ]
    await setupMockRoutes(page, msgs)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    await expect(page.locator('[data-album]')).toHaveCount(1)
    await expect(page.locator('[data-album] [data-message-id]')).toHaveCount(3)
  })

  test('a gap larger than two minutes breaks the run', async ({ page }) => {
    const msgs = [
      makeMessage({ created_at: '2026-09-08T10:00:00Z' }),
      makeMessage({ created_at: '2026-09-08T10:02:01Z' }), // 2m01s later
    ]
    await setupMockRoutes(page, msgs)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    await expect(page.locator('[data-album]')).toHaveCount(0)
  })

  test('documents group into an album too', async ({ page }) => {
    const burst = albumBurst(3, 0, 'document')
    await setupMockRoutes(page, burst)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    await expect(page.locator('[data-album]')).toHaveCount(1)
    await expect(page.locator('[data-album] [data-message-id]')).toHaveCount(3)
    // Document tiles surface the filename on the icon card.
    await expect(page.locator('[data-album] [data-message-id]').first()).toContainText('doc-0.pdf')
  })

  test('a mixed image + document burst forms one album', async ({ page }) => {
    const msgs = [
      makeMessage({ created_at: '2026-09-08T10:00:00Z' }),
      makeMessage({ message_type: 'document', media_filename: 'invoice.pdf', created_at: '2026-09-08T10:00:05Z' }),
      makeMessage({ created_at: '2026-09-08T10:00:10Z' }),
    ]
    await setupMockRoutes(page, msgs)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    await expect(page.locator('[data-album]')).toHaveCount(1)
    await expect(page.locator('[data-album] [data-message-id]')).toHaveCount(3)
  })

  test('videos never join an album', async ({ page }) => {
    const msgs = [
      makeMessage({ message_type: 'video', media_mime_type: 'video/mp4', created_at: '2026-09-08T10:00:00Z' }),
      makeMessage({ message_type: 'video', media_mime_type: 'video/mp4', created_at: '2026-09-08T10:00:02Z' }),
      makeMessage({ message_type: 'audio', created_at: '2026-09-08T10:00:04Z' }),
    ]
    await setupMockRoutes(page, msgs)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    await expect(page.locator('[data-album]')).toHaveCount(0)
    for (const m of msgs) {
      await expect(page.locator(`#message-${m.id} .chat-bubble`)).toHaveCount(1)
    }
  })

  test('a direction change breaks the run', async ({ page }) => {
    const msgs = [
      makeMessage({ created_at: '2026-09-08T10:00:00Z' }),
      makeMessage({ direction: 'outgoing', created_at: '2026-09-08T10:00:01Z' }),
      makeMessage({ created_at: '2026-09-08T10:00:02Z' }),
    ]
    await setupMockRoutes(page, msgs)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    await expect(page.locator('[data-album]')).toHaveCount(0)
  })

  test('runs longer than four photos show a +N overflow tile', async ({ page }) => {
    const burst = albumBurst(6)
    await setupMockRoutes(page, burst)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    await expect(page.locator('[data-album]')).toHaveCount(1)
    // Only 4 tiles rendered, the last one carries the +2 overlay.
    await expect(page.locator('[data-album] [data-message-id]')).toHaveCount(4)
    await expect(page.locator('[data-album] [data-message-id]').nth(3)).toContainText('+2')
  })

  test('tapping a tile opens that specific photo', async ({ page }) => {
    const burst = albumBurst(3)
    await setupMockRoutes(page, burst)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    const popupPromise = page.waitForEvent('popup')
    await page.locator('[data-album] [data-message-id]').nth(1).click()
    const popup = await popupPromise
    expect(popup.url()).toContain(`/api/media/${burst[1].id}`)
  })

  test('album ZIP button downloads every member in one request', async ({ page }) => {
    const burst = albumBurst(3)
    await setupMockRoutes(page, burst)
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    await page.route('**/api/media/zip*', async (route: Route) => {
      await route.fulfill({ body: TINY_PNG, contentType: 'application/zip' })
    })

    const zipRequest = page.waitForRequest((req) => req.url().includes('/api/media/zip'))
    // The hover actions column carries the ZIP button (stable data attribute —
    // its visible title is locale-dependent).
    await page.locator('[data-album-zip]').click()
    const req = await zipRequest
    const ids = decodeURIComponent(new URL(req.url()).searchParams.get('ids') || '').split(',')
    expect(ids.sort()).toEqual(burst.map((m) => m.id).sort())
  })
})
