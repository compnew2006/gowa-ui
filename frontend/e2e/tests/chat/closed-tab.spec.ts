import { test, expect, Page, Route } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'

/**
 * Closed tab: closed conversations stop receiving messages, so on a busy org
 * they rank far below the newest page of the general contact list (ordered by
 * last message). The tab used to filter only that loaded page — with no closed
 * chat among the newest 50 it stayed empty and had nothing to scroll. It now
 * pages closed chats from the server (`chat_status=closed`).
 *
 * Route interception only — no backend rows needed beyond the login user.
 */

const REPORTED_ID = '658c6695-c7e4-4ff4-b3f8-bf7a96d54e97'

function contact(id: string, overrides: Record<string, any> = {}) {
  return {
    id,
    phone_number: `+2010${id.replace(/\D/g, '').slice(0, 8).padEnd(8, '0')}`,
    name: `Contact ${id.slice(0, 8)}`,
    profile_name: `Contact ${id.slice(0, 8)}`,
    status: 'active',
    unread_count: 0,
    whatsapp_account: 'account-1',
    chat_status: 'pending',
    created_at: '2026-10-01T09:00:00Z',
    updated_at: '2026-10-01T09:00:00Z',
    ...overrides,
  }
}

// The newest page of the general list: busy, and not a single closed chat.
const newestPage = Array.from({ length: 50 }, (_, i) =>
  contact(`00000000-0000-0000-0000-${String(i).padStart(12, '0')}`, {
    last_message_at: `2026-10-16T11:${String(59 - i).padStart(2, '0')}:00Z`,
  })
)
// 55 closed chats in the server's closed list; the reported one is on page 2.
const closedRows = Array.from({ length: 55 }, (_, i) =>
  contact(i === 52 ? REPORTED_ID : `11111111-0000-0000-0000-${String(i).padStart(12, '0')}`, {
    name: i === 52 ? 'Closed by Ramzy' : `Closed ${i}`,
    chat_status: 'closed',
    last_message_at: `2026-10-15T${String(23 - Math.floor(i / 3)).padStart(2, '0')}:00:00Z`,
  })
)

async function setupRoutes(page: Page, closedRequests: string[]) {
  await page.route('**/api/accounts', async (route: Route) => {
    await route.fulfill({ json: { status: 'success', data: { accounts: [{ name: 'account-1', id: '1' }] } } })
  })
  await page.route('**/api/contacts?*', async (route: Route) => {
    const url = new URL(route.request().url())
    const pageNo = Number(url.searchParams.get('page') || '1')
    if (url.searchParams.get('chat_status') === 'closed') {
      closedRequests.push(url.search)
      const rows = closedRows.slice((pageNo - 1) * 50, pageNo * 50)
      await route.fulfill({
        json: { status: 'success', data: { contacts: rows, total: closedRows.length, page: pageNo, limit: 50 } },
      })
      return
    }
    await route.fulfill({
      json: { status: 'success', data: { contacts: pageNo === 1 ? newestPage : [], total: 23946, page: pageNo, limit: 50 } },
    })
  })
}

test.describe('Closed tab', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
  })

  test('lists closed chats even when none are in the newest page', async ({ page }) => {
    const closedRequests: string[] = []
    await setupRoutes(page, closedRequests)
    await page.goto('/chat', { waitUntil: 'domcontentloaded' })

    await page.locator('#tab-closed').click()
    const rows = page.locator('[data-contact-id^="11111111-"]')
    await expect(rows.first()).toBeVisible()
    await expect(rows).toHaveCount(50)
    // Only closed chats in the Closed tab.
    await expect(page.locator('[data-contact-id^="00000000-0000-0000-0000-"]')).toHaveCount(0)
    expect(closedRequests[0]).toContain('page=1')
  })

  test('scrolling the Closed tab pages in older closed chats', async ({ page }) => {
    const closedRequests: string[] = []
    await setupRoutes(page, closedRequests)
    await page.goto('/chat', { waitUntil: 'domcontentloaded' })

    await page.locator('#tab-closed').click()
    await expect(page.locator(`[data-contact-id="${REPORTED_ID}"]`)).toHaveCount(0)

    // Scroll to the end of the list; the infinite scroll asks for page 2.
    const last = page.locator('[data-contact-id^="11111111-"]').last()
    await expect(last).toBeVisible()
    await last.scrollIntoViewIfNeeded()
    await expect(page.locator(`[data-contact-id="${REPORTED_ID}"]`)).toHaveCount(1)
    expect(closedRequests.some((q) => q.includes('page=2'))).toBe(true)
  })
})
