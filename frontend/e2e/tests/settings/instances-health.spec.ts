import { test, expect, type Page } from '@playwright/test'

interface MockHealthInstance {
  id: string
  name: string
  status: string
  is_default: boolean
  auto_read_receipt: boolean
  organization_id: string
  created_at: string
  updated_at: string
}

async function loginAsSuperAdmin(page: Page) {
  await page.goto('/login')
  await page.locator('input[name="email"], input[type="email"]').fill('admin@admin.com')
  await page.locator('input[name="password"], input[type="password"]').fill('admin')
  await page.locator('button[type="submit"]').click()
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 10000 })
}

function buildInstance(id: string, name: string): MockHealthInstance {
  const now = new Date().toISOString()
  return {
    id,
    name,
    status: 'connected',
    is_default: false,
    auto_read_receipt: true,
    organization_id: 'e2e-org-id',
    created_at: now,
    updated_at: now,
  }
}

test.describe('Instance Health Dashboard', () => {
  test('renders health metrics for instances', async ({ page }) => {
    await loginAsSuperAdmin(page)

    const instance = buildInstance('health-render-id', 'Health Render Primary')

    await page.route('**/api/instances**', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: [instance],
        }),
      })
    })

    await page.route('**/api/instances/*/health', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: {
            uptime_seconds: 3720,
            messages_sent_today: 112,
            messages_received_today: 208,
            messages_failed_today: 1,
            error_rate_percent: 2.5,
            queue_depth: 73,
          },
        }),
      })
    })

    await page.goto('/settings/instances/health')

    await expect(page.getByText('Instance Health Dashboard')).toBeVisible()
    await expect(page.getByText(instance.name)).toBeVisible()
    await expect(page.getByText('112', { exact: true })).toBeVisible()
    await expect(page.getByText('208', { exact: true })).toBeVisible()
    await expect(page.getByText('73', { exact: true })).toBeVisible()
    await expect(page.getByText('2.5%')).toBeVisible()
  })

  test('refresh button updates health metrics', async ({ page }) => {
    await loginAsSuperAdmin(page)

    const instance = buildInstance('health-refresh-id', `Health Refresh ${Date.now()}`)
    let healthRequests = 0

    await page.route('**/api/instances**', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: [instance],
        }),
      })
    })

    await page.route('**/api/instances/*/health', async route => {
      healthRequests += 1
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: {
            uptime_seconds: 60,
            messages_sent_today: 1,
            messages_received_today: 1,
            messages_failed_today: 0,
            error_rate_percent: 0,
            queue_depth: healthRequests * 11,
          },
        }),
      })
    })

    await page.goto('/settings/instances/health')
    await expect.poll(() => healthRequests).toBeGreaterThan(0)
    const beforeRefresh = healthRequests
    await page.getByRole('button', { name: 'Refresh' }).click()
    await expect.poll(() => healthRequests).toBeGreaterThan(beforeRefresh)
    await expect(page.getByText(String(healthRequests * 11), { exact: true })).toBeVisible()
  })

  test('uses selected organization context after org switch and refreshes data', async ({ page }) => {
    await loginAsSuperAdmin(page)

    const orgAInstance = buildInstance('org-a-instance', 'Org A Instance')
    const orgBInstance = buildInstance('org-b-instance', 'Org B Instance')

    await page.route('**/api/instances**', async route => {
      const orgID = route.request().headers()['x-organization-id'] || ''
      const selected = orgID === 'org-b' ? orgBInstance : orgAInstance
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: [selected],
        }),
      })
    })

    await page.route('**/api/instances/*/health', async route => {
      const orgID = route.request().headers()['x-organization-id'] || ''
      const queueDepth = orgID === 'org-b' ? 99 : 55
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: {
            uptime_seconds: 120,
            messages_sent_today: 4,
            messages_received_today: 4,
            messages_failed_today: 0,
            error_rate_percent: 0,
            queue_depth: queueDepth,
          },
        }),
      })
    })

    await page.evaluate(() => {
      localStorage.setItem('selected_organization_id', 'org-a')
    })
    await page.goto('/settings/instances/health')

    await expect(page.getByText('Org A Instance')).toBeVisible()
    await expect(page.getByText('55')).toBeVisible()

    await page.evaluate(() => {
      localStorage.setItem('selected_organization_id', 'org-b')
    })
    await page.reload()

    await expect(page.getByText('Org B Instance')).toBeVisible()
    await expect(page.getByText('99')).toBeVisible()
  })
})
