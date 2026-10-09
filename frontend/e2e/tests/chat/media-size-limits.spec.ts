import { test, expect, Page, Route } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

/**
 * Upload size limits follow the engine (GOWA), per message type — image 20MB,
 * video 100MB, audio/documents 50MB — not a flat 16MB for every media file.
 * A video above the old 16MB cap is accepted and posted whole; a file over its
 * limit is refused with a message that names the file, its size and the limit.
 *
 * Same route-interception harness as media-multisend.spec.ts; real in-browser
 * File objects of real size are handed to the hidden file input.
 */

const CONTACT_ID = '00000000-0000-0000-0000-0000000000d4'

const CONTACT = {
  id: CONTACT_ID,
  phone_number: '+1234567890',
  name: 'Limits Receiver',
  profile_name: 'Limits Receiver',
  status: 'active',
  chat_status: 'open',
  unread_count: 0,
  whatsapp_account: 'account-1',
  created_at: '2026-09-08T09:00:00Z',
  updated_at: '2026-09-08T09:00:00Z',
}

async function setupMockRoutes(page: Page, onMediaPost: (bytes: number, contentType: string) => void) {
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
        json: { status: 'success', data: { messages: [], total: 0, page: 1, limit: 50, has_more: false } },
      })
    } else {
      await route.continue()
    }
  })
  await page.route(`**/api/contacts/${CONTACT_ID}/session`, async (route: Route) => {
    await route.fulfill({ json: { status: 'success', data: null } })
  })
  await page.route('**/api/messages/media', async (route: Route) => {
    if (route.request().method() === 'POST') {
      const req = route.request()
      onMediaPost(req.postDataBuffer()?.length ?? 0, req.headers()['content-type'] ?? '')
      await route.fulfill({
        json: {
          status: 'success',
          data: {
            id: crypto.randomUUID(),
            contact_id: CONTACT_ID,
            direction: 'outgoing',
            message_type: 'video',
            content: {},
            media_url: '/files/clip.mp4',
            media_mime_type: 'video/mp4',
            media_filename: 'clip.mp4',
            status: 'sent',
            whatsapp_account: 'account-1',
            created_at: '2026-09-08T10:00:00Z',
            updated_at: '2026-09-08T10:00:00Z',
          },
        },
      })
    } else {
      await route.continue()
    }
  })
}

test.describe('Upload size limits', () => {
  let chatPage: ChatPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatPage = new ChatPage(page)
  })

  test('a 30MB video, over the old 16MB cap, is accepted and posted whole', async ({ page }) => {
    const posts: { bytes: number; contentType: string }[] = []
    await setupMockRoutes(page, (bytes, contentType) => posts.push({ bytes, contentType }))
    await chatPage.goto(CONTACT_ID)

    const size = 30 * 1_000_000
    await page.locator('input[type="file"]').setInputFiles({
      name: 'clip.mp4',
      mimeType: 'video/mp4',
      buffer: Buffer.alloc(size, 1),
    })

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(dialog.getByText('clip.mp4').first()).toBeVisible()

    await dialog.getByRole('button', { name: /^Send$/ }).click()

    await expect.poll(() => posts.length, { timeout: 30_000 }).toBe(1)
    expect(posts[0].contentType).toContain('multipart/form-data')
    expect(posts[0].bytes).toBeGreaterThanOrEqual(size)
    await expect(dialog).toBeHidden({ timeout: 15_000 })
  })

  test('an image over 20MB is refused with the file, its size and the limit', async ({ page }) => {
    const posts: number[] = []
    await setupMockRoutes(page, (bytes) => posts.push(bytes))
    await chatPage.goto(CONTACT_ID)

    await page.locator('input[type="file"]').setInputFiles({
      name: 'huge.png',
      mimeType: 'image/png',
      buffer: Buffer.alloc(20_000_001, 1),
    })

    await expect(page.getByText('File too large').first()).toBeVisible({ timeout: 5_000 })
    await expect(page.getByText('huge.png: 20.1 MB (limit 20 MB)')).toBeVisible()
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(posts).toHaveLength(0)
  })
})
