import { test, expect, Locator, Page, Route } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'

/**
 * Paste an image (Ctrl/Cmd+V) into the composer: the media-send dialog opens
 * with the image queued, nothing is sent until the agent presses Send, a second
 * paste while the dialog is open joins the queue, and ordinary text / rich-text
 * pastes are left alone.
 *
 * Same route-interception harness as media-multisend.spec.ts. The clipboard is
 * simulated by dispatching a real ClipboardEvent carrying a DataTransfer — the
 * OS clipboard itself is not reachable from a headless browser.
 */

const CONTACT_ID = '00000000-0000-0000-0000-0000000000c3'

const CONTACT = {
  id: CONTACT_ID,
  phone_number: '+1234567890',
  name: 'Paste Receiver',
  profile_name: 'Paste Receiver',
  status: 'active',
  chat_status: 'open',
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
    message_type: 'image',
    content: i === 0 ? { body: 'pasted caption' } : {},
    media_url: `/files/pasted-${i}.png`,
    media_mime_type: 'image/png',
    media_filename: `pasted-${i}.png`,
    status: 'sent',
    whatsapp_account: 'account-1',
    created_at: `2026-09-08T10:00:0${i}Z`,
    updated_at: `2026-09-08T10:00:0${i}Z`,
  }
}

interface MediaPost {
  body: string
}

async function setupMockRoutes(page: Page, onMediaPost: (post: MediaPost) => void) {
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
      onMediaPost({ body: route.request().postDataBuffer()?.toString('latin1') ?? '' })
      await route.fulfill({ json: { status: 'success', data: msg } })
    } else {
      await route.continue()
    }
  })
}

interface PasteSpec {
  /** Number of PNG bitmaps on the simulated clipboard. */
  images?: number
  text?: string
  html?: string
}

/**
 * Dispatch a cancelable `paste` ClipboardEvent on `target` and report whether
 * the page claimed it (preventDefault) — i.e. whether it would have suppressed
 * the browser's own paste.
 */
async function pasteInto(target: Locator, spec: PasteSpec): Promise<boolean> {
  return target.evaluate(async (el, s) => {
    const dt = new DataTransfer()
    if (s.text !== undefined) dt.setData('text/plain', s.text)
    if (s.html !== undefined) dt.setData('text/html', s.html)
    for (let i = 0; i < (s.images ?? 0); i++) {
      // A real, decodable PNG so the dialog preview renders.
      const canvas = document.createElement('canvas')
      canvas.width = 320
      canvas.height = 200
      const ctx = canvas.getContext('2d')!
      const gradient = ctx.createLinearGradient(0, 0, 320, 200)
      gradient.addColorStop(0, '#0ea5e9')
      gradient.addColorStop(1, '#10b981')
      ctx.fillStyle = gradient
      ctx.fillRect(0, 0, 320, 200)
      ctx.fillStyle = '#ffffff'
      ctx.font = '28px sans-serif'
      ctx.fillText(`screenshot ${i + 1}`, 24, 108)
      const blob: Blob = await new Promise((resolve) => canvas.toBlob((b) => resolve(b!), 'image/png'))
      // Browsers hand pasted bitmaps over under the generic name "image.png".
      dt.items.add(new File([blob], 'image.png', { type: 'image/png' }))
    }
    const event = new ClipboardEvent('paste', { clipboardData: dt, bubbles: true, cancelable: true })
    el.dispatchEvent(event)
    return event.defaultPrevented
  }, spec)
}

test.describe('Paste image into the composer', () => {
  let chatPage: ChatPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    chatPage = new ChatPage(page)
  })

  test('Ctrl+V of an image opens the media dialog and sends nothing until Send', async ({ page }) => {
    const posts: MediaPost[] = []
    await setupMockRoutes(page, (p) => posts.push(p))
    await chatPage.goto(CONTACT_ID)

    const composer = chatPage.messageInput
    await expect(composer).toBeVisible()
    await composer.focus()

    expect(await pasteInto(composer, { images: 1 })).toBe(true)

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(dialog.getByText('Send Media')).toBeVisible()
    // The clipboard's generic "image.png" is replaced by a distinct name.
    const preview = dialog.locator('img')
    await expect(preview).toBeVisible()
    await expect(preview).toHaveAttribute('alt', /^image-\d{8}-\d{6}\.png$/)
    expect(posts).toHaveLength(0)

    await dialog.getByPlaceholder(/Add a caption/i).fill('pasted caption')
    await dialog.getByRole('button', { name: /^Send$/ }).click()

    await expect.poll(() => posts.length, { timeout: 10_000 }).toBe(1)
    expect(posts[0].body).toMatch(/filename="image-\d{8}-\d{6}\.png"/)
    expect(posts[0].body).toContain('image/png')
    await expect(page.locator('[id^="message-"]')).toHaveCount(1)
    await expect(dialog).toBeHidden()
  })

  test('a second paste while the dialog is open joins the queue and keeps the caption', async ({ page }) => {
    await setupMockRoutes(page, () => {})
    await chatPage.goto(CONTACT_ID)

    await chatPage.messageInput.focus()
    await pasteInto(chatPage.messageInput, { images: 1 })

    const dialog = page.getByRole('dialog')
    const caption = dialog.getByPlaceholder(/Add a caption/i)
    await caption.fill('order 42')
    await caption.focus()
    expect(await pasteInto(caption, { images: 1 })).toBe(true)

    const queue = page.getByTestId('media-queue')
    await expect(queue.locator('button')).toHaveCount(2)
    // Both pastes land within the same second, yet the files stay tellable apart.
    const names = await queue.locator('button span.truncate').allTextContents()
    expect(new Set(names).size).toBe(2)
    await expect(dialog.getByRole('button', { name: /Send 2 files/i })).toBeVisible()
    await expect(caption).toHaveValue('order 42')
  })

  test('Cancel discards the pasted image without sending, and pasting works again', async ({ page }) => {
    const posts: MediaPost[] = []
    await setupMockRoutes(page, (p) => posts.push(p))
    await chatPage.goto(CONTACT_ID)

    await chatPage.messageInput.focus()
    await pasteInto(chatPage.messageInput, { images: 1 })
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()

    await dialog.getByRole('button', { name: /^Cancel$/ }).click()
    await expect(dialog).toBeHidden()

    await chatPage.messageInput.focus()
    await pasteInto(chatPage.messageInput, { images: 1 })
    await expect(dialog).toBeVisible()
    await expect(page.getByTestId('media-queue')).toHaveCount(0)
    expect(posts).toHaveLength(0)
  })

  test('a plain text paste is left to the browser', async ({ page }) => {
    await setupMockRoutes(page, () => {})
    await chatPage.goto(CONTACT_ID)

    await chatPage.messageInput.focus()
    expect(await pasteInto(chatPage.messageInput, { text: 'see you at 5' })).toBe(false)
    await expect(page.getByRole('dialog')).toHaveCount(0)
  })

  test('a rich-text copy that carries a bitmap rendering stays a text paste', async ({ page }) => {
    await setupMockRoutes(page, () => {})
    await chatPage.goto(CONTACT_ID)

    await chatPage.messageInput.focus()
    const claimed = await pasteInto(chatPage.messageInput, {
      images: 1,
      text: 'qty\tprice\n2\t10',
      html: '<table><tr><td>qty</td><td>price</td></tr></table>',
    })
    expect(claimed).toBe(false)
    await expect(page.getByRole('dialog')).toHaveCount(0)
  })
})
