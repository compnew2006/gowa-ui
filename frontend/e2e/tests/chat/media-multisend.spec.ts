import { test, expect, Page, Route } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

/**
 * Multi-file send: the composer file input accepts several files, the media
 * dialog queues them, and Send posts each file as its own message via
 * POST /api/messages/media (caption on the first only).
 *
 * Uses route interception (same harness as media-album.spec.ts) — no real
 * provider delivery is needed.
 */

const CONTACT_ID = '00000000-0000-0000-0000-0000000000b2'

const CONTACT = {
  id: CONTACT_ID,
  phone_number: '+1234567890',
  name: 'Multisend Receiver',
  profile_name: 'Multisend Receiver',
  status: 'active',
  unread_count: 0,
  whatsapp_account: 'account-1',
  created_at: '2026-09-08T09:00:00Z',
  updated_at: '2026-09-08T09:00:00Z',
}

function makeSentMessage(i: number) {
  return {
    id: crypto.randomUUID(),
    contact_id: CONTACT_ID,
    direction: 'outgoing',
    message_type: 'document',
    content: i === 0 ? { body: 'batch caption' } : {},
    media_url: `/files/doc-${i}.pdf`,
    media_mime_type: 'application/pdf',
    media_filename: `doc-${i}.pdf`,
    status: 'sent',
    whatsapp_account: 'account-1',
    created_at: `2026-09-08T10:00:0${i}Z`,
    updated_at: `2026-09-08T10:00:0${i}Z`,
  }
}

async function setupMockRoutes(page: Page, onMediaPost: () => void) {
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
  let n = 0
  await page.route('**/api/messages/media', async (route: Route) => {
    if (route.request().method() === 'POST') {
      const msg = makeSentMessage(n)
      n += 1
      onMediaPost()
      await route.fulfill({ json: { status: 'success', data: msg } })
    } else {
      await route.continue()
    }
  })
}

const PDF_BYTES = Buffer.from('%PDF-1.4 fake\n')

test.describe('Multi-file send', () => {
  let chatPage: ChatPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatPage = new ChatPage(page)
  })

  test('picking two files queues both and Send posts two messages', async ({ page }) => {
    let posts = 0
    await setupMockRoutes(page, () => { posts += 1 })
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    // Pick two files directly on the hidden input (no native dialog).
    await page.locator('input[type="file"]').setInputFiles([
      { name: 'doc-0.pdf', mimeType: 'application/pdf', buffer: PDF_BYTES },
      { name: 'doc-1.pdf', mimeType: 'application/pdf', buffer: PDF_BYTES },
    ])

    // Queue lists both files; the send button carries the count.
    const queue = page.getByTestId('media-queue')
    await expect(queue).toBeVisible()
    await expect(queue.locator('button', { hasText: 'doc-0.pdf' })).toHaveCount(1)
    await expect(queue.locator('button', { hasText: 'doc-1.pdf' })).toHaveCount(1)

    // Caption rides along; the dialog send button shows the batch count.
    await page.getByPlaceholder(/Add a caption/i).fill('batch caption')
    const sendBatch = page.getByRole('button', { name: /Send 2 files/i })
    await expect(sendBatch).toBeVisible()

    await sendBatch.click()
    // Both files posted sequentially, each as its own message bubble
    // (bubbles carry id="message-{id}"; data-message-id exists only on
    // album tiles, and the captioned first file never joins an album).
    await expect.poll(() => posts, { timeout: 15_000 }).toBe(2)
    await expect(page.locator('[id^="message-"]')).toHaveCount(2)
  })

  test('a batch containing an unsupported file never starts', async ({ page }) => {
    let posts = 0
    await setupMockRoutes(page, () => { posts += 1 })
    await chatPage.goto(CONTACT_ID)
    await page.waitForTimeout(500)

    await page.locator('input[type="file"]').setInputFiles([
      { name: 'doc-0.pdf', mimeType: 'application/pdf', buffer: PDF_BYTES },
      { name: 'evil.exe', mimeType: 'application/x-msdownload', buffer: Buffer.from('MZ') },
    ])

    // Error toast names the offender; the media dialog stays closed.
    await expect(page.getByText('evil.exe').first()).toBeVisible({ timeout: 5_000 })
    await expect(page.getByTestId('media-queue')).toHaveCount(0)
    await page.waitForTimeout(500)
    expect(posts).toBe(0)
  })
})
