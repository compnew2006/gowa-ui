import { test, expect } from '@playwright/test'
import { loginAsAdmin, navigateToFirstItem, expectMetadataVisible, expectActivityLogVisible, expectDeleteFromForm, ApiHelper } from '../../helpers'
import { AccountsPage } from '../../pages'
import { createTestScope, loginAsSuperAdmin, SUPER_ADMIN } from '../../framework'

const scope = createTestScope('accounts')

test.describe('WhatsApp Accounts - List View', () => {
  let accountsPage: AccountsPage

  test.beforeEach(async ({ page, request }) => {
    await loginAsAdmin(page)
    // Seed a WhatsApp account so list/detail tests have a stable row and never
    // silently pass on an empty list. The seeded account is shared across the
    // tests in this describe; tests that mutate it create their own.
    const api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    await api.createWhatsAppAccount({
      name: scope.name('seed').toLowerCase().replace(/\s/g, '-'),
      phone_id: `phone-seed-${Date.now()}`,
      business_id: `biz-seed-${Date.now()}`,
      access_token: 'test-token-e2e',
    })
    accountsPage = new AccountsPage(page)
    await accountsPage.goto()
  })

  test('should display accounts page', async () => {
    await accountsPage.expectPageVisible()
    await expect(accountsPage.addButton).toBeVisible()
  })

  // Merged: previously two near-duplicate tests ('should load create page' +
  // 'should show form fields on create page') that both asserted the create
  // page renders its inputs. Collapsed into one per Rule 4.
  test('should load create page with key form fields', async ({ page }) => {
    await page.goto('/settings/accounts/new')
    await page.waitForLoadState('networkidle')
    expect(page.url()).toContain('/settings/accounts/new')
    await expect(page.locator('input').first()).toBeVisible()
    await expect(page.locator('input[type="password"]').first()).toBeVisible()
  })

  test('should show delete confirmation from list', async ({ page }) => {
    // TODO(test-guard): move to AccountsPage POM
    // Find the destructive (red) delete button in the first data row
    const firstRow = page.locator('tbody tr').first()
    await expect(firstRow).toBeVisible({ timeout: 5000 })

    // TODO(test-guard): move to AccountsPage POM
    const deleteBtn = firstRow.locator('button.text-destructive, button:has(svg.text-destructive)').first()
    await expect(deleteBtn).toBeVisible({ timeout: 5000 })
    await deleteBtn.click()
    await expect(accountsPage.alertDialog).toBeVisible({ timeout: 5000 })
    await accountsPage.cancelDelete()
  })

  test('should load detail page from list', async ({ page }) => {
    const href = await navigateToFirstItem(page)
    expect(href, 'a seeded account row should be navigable').not.toBeNull()
    expect(page.url()).toMatch(/\/settings\/accounts\/[a-f0-9-]+/)
    await expect(page.getByText('Account Details')).toBeVisible()
  })
})

test.describe('WhatsApp Accounts - Detail Page CRUD', () => {
  test.beforeEach(async ({ page, request }) => {
    await loginAsAdmin(page)
    // Seed a WhatsApp account for detail-page tests so they assert
    // unconditionally instead of silently skipping on an empty list.
    const api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    await api.createWhatsAppAccount({
      name: scope.name('detail-seed').toLowerCase().replace(/\s/g, '-'),
      phone_id: `phone-detail-${Date.now()}`,
      business_id: `biz-detail-${Date.now()}`,
      access_token: 'test-token-e2e',
    })
  })

  test('should show validation error for empty required fields', async ({ page }) => {
    await page.goto('/settings/accounts/new')
    await page.waitForLoadState('networkidle')

    const createBtn = page.getByRole('button', { name: /Create/i })
    await expect(createBtn).toBeVisible({ timeout: 5000 })
    // Fill something to trigger hasChanges, then clear to surface validation.
    const input = page.locator('input').first()
    await input.fill('test')
    await input.clear()
    await page.waitForTimeout(300)

    await createBtn.click({ force: true })
    const toast = page.locator('[data-sonner-toast]').first()
    await expect(toast).toBeVisible({ timeout: 5000 })
  })

  test('should show webhook config on existing account', async ({ page }) => {
    await page.goto('/settings/accounts')
    await page.waitForLoadState('networkidle')

    const href = await navigateToFirstItem(page)
    expect(href, 'a seeded account row should be navigable').not.toBeNull()
    await expect(page.getByText('Webhook Configuration')).toBeVisible()
  })

  test('should have test connection button', async ({ page }) => {
    await page.goto('/settings/accounts')
    await page.waitForLoadState('networkidle')

    const href = await navigateToFirstItem(page)
    expect(href, 'a seeded account row should be navigable').not.toBeNull()
    await expect(page.getByRole('button', { name: /Test/i })).toBeVisible()
  })

  test('should have subscribe button', async ({ page }) => {
    await page.goto('/settings/accounts')
    await page.waitForLoadState('networkidle')

    const href = await navigateToFirstItem(page)
    expect(href, 'a seeded account row should be navigable').not.toBeNull()
    await expect(page.getByRole('button', { name: /Subscribe/i })).toBeVisible()
  })

  test('should have business profile button', async ({ page }) => {
    await page.goto('/settings/accounts')
    await page.waitForLoadState('networkidle')

    const href = await navigateToFirstItem(page)
    expect(href, 'a seeded account row should be navigable').not.toBeNull()
    await expect(page.getByRole('button', { name: /Profile/i })).toBeVisible()
  })

  test('should delete from detail page', async ({ page }) => {
    await page.goto('/settings/accounts')
    await page.waitForLoadState('networkidle')

    const href = await navigateToFirstItem(page)
    expect(href, 'a seeded account row should be navigable').not.toBeNull()
    await expectDeleteFromForm(page, '/settings/accounts')
  })

  test('should show metadata', async ({ page }) => {
    await page.goto('/settings/accounts')
    await page.waitForLoadState('networkidle')

    const href = await navigateToFirstItem(page)
    expect(href, 'a seeded account row should be navigable').not.toBeNull()
    await expectMetadataVisible(page)
  })

  test('should show activity log', async ({ page }) => {
    await page.goto('/settings/accounts')
    await page.waitForLoadState('networkidle')

    const href = await navigateToFirstItem(page)
    expect(href, 'a seeded account row should be navigable').not.toBeNull()
    await expectActivityLogVisible(page)
  })

  test('should show setup guide', async ({ page, request }) => {
    // Seed our own account so we don't race with parallel workers that
    // create-then-delete accounts (e.g. audit-trail.spec). navigateToFirstItem
    // grabs the first row's href, but if another worker deletes that account
    // before goto lands, the detail page renders the "not found" error state
    // and Setup Guide never appears.
    const api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    const acc = await api.createWhatsAppAccount({
      name: scope.name('setup-guide').toLowerCase().replace(/\s/g, '-'),
      phone_id: `phone-setup-${Date.now()}`,
      business_id: `biz-setup-${Date.now()}`,
      access_token: 'test-token-e2e',
    })

    await page.goto(`/settings/accounts/${acc.id}`)
    await page.waitForLoadState('networkidle')

    await expect(page.getByText('Setup Guide')).toBeVisible({ timeout: 15000 })
  })

  // Collapsed: previously two near-duplicate page.route()-mocked tests for
  // GREEN and UNKNOWN quality_rating. Data-driven per Rule 3/4. The route mock
  // stays — it's a justified HTTP-boundary mock (Rule 2).
  for (const [qualityRating, expectedLabel] of [
    ['GREEN', 'High'],
    ['UNKNOWN', 'Unknown'],
  ] as const) {
    test(`should show connection details card for ${qualityRating} quality rating (${expectedLabel})`, async ({ page, request }) => {
      // Browser must share identity with the API session below; otherwise
      // /settings/accounts/:id 404s for the wrong org. See framework/auth.ts.
      await loginAsSuperAdmin(page)
      const api = new ApiHelper(request)
      await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
      const acc = await api.createWhatsAppAccount({
        name: scope.name(`conn-${qualityRating}`).toLowerCase().replace(/\s/g, '-'),
        phone_id: `phone-conn-${Date.now()}`,
        business_id: `biz-conn-${Date.now()}`,
        access_token: 'test-token-e2e',
      })

      // Stub the connection test response
      await page.route(`**/api/accounts/${acc.id}/test`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              success: true,
              display_phone_number: '1234567890',
              verified_name: 'Test Verified Company Name',
              quality_rating: qualityRating,
              messaging_limit_tier: 'TIER_250',
              code_verification_status: 'VERIFIED',
              account_mode: 'LIVE',
              is_test_number: false
            }
          })
        })
      })

      await page.goto(`/settings/accounts/${acc.id}`)
      await page.waitForLoadState('networkidle')

      // Click the Test button
      await page.getByRole('button', { name: /Test/i }).click()

      // Assert details card is shown and the rating is translated correctly
      await expect(page.getByText('Details', { exact: true })).toBeVisible()
      await expect(page.getByText('Test Verified Company Name')).toBeVisible()
      await expect(page.getByText(expectedLabel)).toBeVisible()
    })
  }
})
