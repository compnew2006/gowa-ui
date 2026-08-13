import { test, expect } from '@playwright/test'
import { loginAsAdmin, navigateToFirstItem, expectMetadataVisible, expectActivityLogVisible, expectDeleteFromForm, ApiHelper } from '../../helpers'
import { AccountsPage } from '../../pages'
import { createTestScope, SUPER_ADMIN } from '../../framework'

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
      // scope.name() (no suffix) appends a random suffix: workers are reused
      // across tests and the module-level scope runId is identical for both,
      // so a fixed suffix would collide with the previous test's row.
      name: scope.name().toLowerCase().replace(/\s/g, '-'),
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
    await accountsPage.expectGatewayCardVisible()
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
      name: scope.name().toLowerCase().replace(/\s/g, '-'),
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

  // Each of these detail-page tests seeds its OWN account and navigates to it
  // directly, so parallel workers that create-then-delete rows (delete test,
  // audit-trail.spec) can never invalidate a first-row navigation.
  async function seedAndOpenOwnAccount(page: any, request: any, label: string) {
    const api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    const acc = await api.createWhatsAppAccount({
      name: scope.name(label).toLowerCase().replace(/\s/g, '-'),
      phone_id: `phone-${label}-${Date.now()}`,
      business_id: `biz-${label}-${Date.now()}`,
      access_token: 'test-token-e2e',
    })
    await page.goto(`/settings/accounts/${acc.id}`)
    await page.waitForLoadState('networkidle')
    return acc
  }

  test('should show webhook config on existing account', async ({ page, request }) => {
    await seedAndOpenOwnAccount(page, request, 'webhook-config')
    await expect(page.getByText('Webhook Configuration')).toBeVisible()
  })

  test('should delete from detail page', async ({ page, request }) => {
    await seedAndOpenOwnAccount(page, request, 'delete-detail')
    await expectDeleteFromForm(page, '/settings/accounts')
  })

  test('should show metadata', async ({ page, request }) => {
    await seedAndOpenOwnAccount(page, request, 'meta-info')
    await expectMetadataVisible(page)
  })

  test('should show activity log', async ({ page, request }) => {
    await seedAndOpenOwnAccount(page, request, 'activity-log')
    await expectActivityLogVisible(page)
  })

  test('should link to GOWA Gateway from detail page', async ({ page, request }) => {
    // Device lifecycle (pairing/connect) is owned by the GOWA Gateway page.
    // A plain account without gowa_base_url/gowa_device_id falls back to the
    // generic gateway link.
    const api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    const acc = await api.createWhatsAppAccount({
      name: scope.name('gateway-link').toLowerCase().replace(/\s/g, '-'),
      phone_id: `phone-gateway-${Date.now()}`,
      business_id: `biz-gateway-${Date.now()}`,
      access_token: 'test-token-e2e',
    })

    await page.goto(`/settings/accounts/${acc.id}`)
    await page.waitForLoadState('networkidle')

    const gatewayLink = page.locator('a').filter({ hasText: 'GOWA Gateway' }).first()
    await expect(gatewayLink).toBeVisible({ timeout: 15000 })
    await gatewayLink.click()
    await expect(page).toHaveURL(/\/settings\/gowa-servers/)
  })
})
