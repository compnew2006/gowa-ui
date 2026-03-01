import { test, expect, type Page } from '@playwright/test'

interface MockClosedChat {
  id: string
  name: string
  profile_name: string
  phone_number: string
  status: 'closed'
  closed_at: string
  updated_at: string
  closed_by_user_id: string
  closed_by_name: string
  assigned_user_id: string
}

interface MockAgent {
  id: string
  full_name: string
  email: string
  organization_id: string
  is_active: boolean
  created_at: string
  updated_at: string
}

const AUTH_USER = {
  id: 'e2e-user-id',
  email: 'admin@test.com',
  full_name: 'E2E Admin',
  organization_id: 'e2e-org-id',
  role: {
    id: 'e2e-role-id',
    name: 'admin',
    is_system: false,
    permissions: [
      { id: 'perm-chat-read', resource: 'chat', action: 'read' },
      { id: 'perm-settings-read', resource: 'settings.general', action: 'read' }
    ]
  },
  is_super_admin: true
}

const MOCK_AGENTS: MockAgent[] = [
  {
    id: 'agent-alpha',
    full_name: 'Closer Alpha',
    email: 'alpha@test.com',
    organization_id: AUTH_USER.organization_id,
    is_active: true,
    created_at: new Date(Date.UTC(2026, 0, 1)).toISOString(),
    updated_at: new Date(Date.UTC(2026, 0, 1)).toISOString()
  },
  {
    id: 'agent-bravo',
    full_name: 'Closer Bravo',
    email: 'bravo@test.com',
    organization_id: AUTH_USER.organization_id,
    is_active: true,
    created_at: new Date(Date.UTC(2026, 0, 1)).toISOString(),
    updated_at: new Date(Date.UTC(2026, 0, 1)).toISOString()
  }
]

async function seedAuthenticatedUser(page: Page) {
  await page.addInitScript((user) => {
    window.localStorage.setItem('user', JSON.stringify(user))
  }, AUTH_USER)
}

async function mockUsersAPI(page: Page, agents: MockAgent[] = MOCK_AGENTS) {
  await page.route('**/api/users**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (request.method() !== 'GET' || !url.pathname.endsWith('/api/users')) {
      await route.fallback()
      return
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: {
          users: agents,
          total: agents.length,
          page: 1,
          limit: agents.length
        }
      })
    })
  })
}

function buildClosedChats(total: number): MockClosedChat[] {
  const chats: MockClosedChat[] = []
  for (let i = 1; i <= total; i++) {
    const closedAt = new Date(Date.UTC(2026, 0, i)).toISOString()
    const closedByName = i % 2 === 0 ? 'Closer Alpha' : 'Closer Bravo'
    const closerID = i % 2 === 0 ? 'agent-alpha' : 'agent-bravo'
    chats.push({
      id: `closed-chat-${i}`,
      name: `Contact ${i}`,
      profile_name: `Contact ${i}`,
      phone_number: `1555000${String(i).padStart(4, '0')}`,
      status: 'closed',
      closed_at: closedAt,
      updated_at: closedAt,
      closed_by_user_id: closerID,
      closed_by_name: closedByName,
      assigned_user_id: closerID
    })
  }
  return chats
}

async function mockClosedChatsAPI(
  page: Page,
  sourceChats: MockClosedChat[],
  options?: { failSearches?: string[] }
) {
  const failSearches = new Set((options?.failSearches || []).map(value => value.trim().toLowerCase()))

  await page.route('**/api/chats**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())

    if (request.method() !== 'GET' || !url.pathname.endsWith('/api/chats')) {
      await route.fallback()
      return
    }

    const status = (url.searchParams.get('status') || '').trim().toLowerCase()
    if (status !== 'closed') {
      await route.fallback()
      return
    }

    const search = (url.searchParams.get('search') || '').trim().toLowerCase()
    const closedBy = (url.searchParams.get('closed_by') || '').trim().toLowerCase()
    const closedFrom = (url.searchParams.get('closed_from') || '').trim()
    const closedTo = (url.searchParams.get('closed_to') || '').trim()
    const pageNumber = Math.max(1, Number(url.searchParams.get('page') || '1'))
    const limit = Math.max(1, Number(url.searchParams.get('limit') || '25'))

    if (search && failSearches.has(search)) {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'error',
          message: 'Mock closed chats failure'
        })
      })
      return
    }

    let filtered = [...sourceChats]

    if (search) {
      filtered = filtered.filter((chat) =>
        chat.name.toLowerCase().includes(search) ||
        chat.profile_name.toLowerCase().includes(search) ||
        chat.phone_number.toLowerCase().includes(search) ||
        chat.closed_by_name.toLowerCase().includes(search)
      )
    }

    if (closedBy) {
      filtered = filtered.filter((chat) =>
        chat.closed_by_name.toLowerCase().includes(closedBy) ||
        chat.closed_by_user_id.toLowerCase().includes(closedBy)
      )
    }

    if (closedFrom) {
      filtered = filtered.filter((chat) => chat.closed_at.slice(0, 10) >= closedFrom)
    }

    if (closedTo) {
      filtered = filtered.filter((chat) => chat.closed_at.slice(0, 10) <= closedTo)
    }

    const total = filtered.length
    const offset = (pageNumber - 1) * limit
    const contacts = filtered.slice(offset, offset + limit)

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'success',
        data: {
          contacts,
          total,
          page: pageNumber,
          limit
        }
      })
    })
  })
}

test.describe('Closed Chats Settings', () => {
  test('supports page navigation, page size options, and scrollable table', async ({ page }) => {
    await seedAuthenticatedUser(page)
    await mockUsersAPI(page)
    await mockClosedChatsAPI(page, buildClosedChats(130))

    await page.goto('/settings/closed-chats')

    await expect(page.getByTestId('closed-chats-page-label')).toHaveText('Page 1 of 6')
    await expect(page.locator('tbody tr')).toHaveCount(25)

    await page.getByTestId('closed-chats-next-page').click()
    await expect(page.getByTestId('closed-chats-page-label')).toHaveText('Page 2 of 6')
    await expect(page.getByText(/^Contact 26$/)).toBeVisible()

    await page.getByTestId('closed-chats-page-size').click()
    await page.getByRole('option', { name: '100' }).click()
    await expect(page.getByTestId('closed-chats-page-label')).toHaveText('Page 1 of 2')
    await expect(page.locator('tbody tr')).toHaveCount(100)

    const scrollInfo = await page.getByTestId('closed-chats-scroll').evaluate((element) => ({
      scrollHeight: element.scrollHeight,
      clientHeight: element.clientHeight
    }))
    expect(scrollInfo.scrollHeight).toBeGreaterThan(scrollInfo.clientHeight)
  })

  test('filters closed chats by agent and closed date range', async ({ page }) => {
    await seedAuthenticatedUser(page)
    await mockUsersAPI(page)
    await mockClosedChatsAPI(page, buildClosedChats(130))

    await page.goto('/settings/closed-chats')

    await page.getByTestId('closed-chats-agent-filter').click()
    await page.getByTestId('closed-chats-agent-search').fill('agent-alpha')
    await page.getByTestId('closed-chats-agent-option-agent-alpha').click()
    await page.waitForTimeout(450)
    await expect(page.locator('tbody tr')).toHaveCount(25)
    await expect(page.locator('tbody')).toContainText('Closer Alpha')
    await expect(page.locator('tbody')).not.toContainText('Closer Bravo')

    await page.getByTestId('closed-chats-date-from').fill('2026-01-20')
    await page.getByTestId('closed-chats-date-to').fill('2026-01-20')
    await page.waitForTimeout(450)

    await expect(page.locator('tbody tr')).toHaveCount(1)
    await expect(page.getByText(/^Contact 20$/)).toBeVisible()
    await expect(page.getByTestId('closed-chats-page-label')).toHaveText('Page 1 of 1')
  })

  test('filters closed chats by phone search and clears stale rows on request failure', async ({ page }) => {
    await seedAuthenticatedUser(page)
    await mockUsersAPI(page)
    const chats = buildClosedChats(40)
    chats[5].phone_number = '565534614'
    await mockClosedChatsAPI(page, chats, { failSearches: ['565534614'] })

    await page.goto('/settings/closed-chats')
    await expect(page.locator('tbody tr')).toHaveCount(25)
    await expect(page.getByText(/^Contact 1$/)).toBeVisible()

    await page.getByTestId('closed-chats-search').fill('565534614')
    await page.waitForTimeout(450)

    await expect(page.locator('tbody tr')).toHaveCount(1)
    await expect(page.locator('tbody')).toContainText('No closed chats found.')
    await expect(page.getByTestId('closed-chats-page-label')).toHaveText('Page 1 of 1')
  })
})
