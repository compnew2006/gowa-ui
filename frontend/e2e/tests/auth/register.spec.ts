import { test, expect } from '@playwright/test'
import { ApiHelper } from '../../helpers'
import { createTestScope, SUPER_ADMIN } from '../../framework'

const scope = createTestScope('register')

async function createOrgForRegister(api: ApiHelper, label: string): Promise<string | null> {
  try {
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
  } catch {
    try {
      await api.login('admin@test.com', 'password')
    } catch {
      return null
    }
  }
  try {
    const org = await api.createOrganization(scope.name(label))
    return org.id
  } catch {
    return null
  }
}

test.describe('Register', () => {
  test('should show invitation required message without org param', async ({ page }) => {
    await page.goto('/register')

    await expect(page.locator('input#fullName')).not.toBeVisible()
    await expect(page.locator('input#email')).not.toBeVisible()
    await expect(page.locator('input#password')).not.toBeVisible()

    await expect(page.locator('text=/invitation/i')).toBeVisible()
    await expect(page.getByRole('link', { name: /Sign in/i })).toBeVisible()
  })

  test('should display registration form with org query param', async ({ page, request }) => {
    const api = new ApiHelper(request)
    const orgId = await createOrgForRegister(api, 'display')
    test.skip(!orgId, 'Failed to set up test organization')

    await page.goto(`/register?org=${orgId}`)

    await expect(page.locator('input#fullName')).toBeVisible()
    await expect(page.locator('input#email')).toBeVisible()
    await expect(page.locator('input#password')).toBeVisible()
    await expect(page.locator('input#confirmPassword')).toBeVisible()
    await expect(page.locator('button[type="submit"]')).toBeVisible()
  })

  test('should show error for empty fields', async ({ page, request }) => {
    const api = new ApiHelper(request)
    const orgId = await createOrgForRegister(api, 'empty')
    test.skip(!orgId, 'Failed to set up test organization')

    await page.goto(`/register?org=${orgId}`)
    await page.locator('button[type="submit"]').click()

    const toast = page.locator('[data-sonner-toast]')
    await expect(toast).toBeVisible({ timeout: 5000 })
    await expect(toast).toContainText('fill in all fields')
  })

  test('should show error for mismatched passwords', async ({ page, request }) => {
    const api = new ApiHelper(request)
    const orgId = await createOrgForRegister(api, 'mismatch')
    test.skip(!orgId, 'Failed to set up test organization')

    await page.goto(`/register?org=${orgId}`)
    await page.locator('input#fullName').fill('Test User')
    await page.locator('input#email').fill(scope.email('mismatch'))
    await page.locator('input#password').fill('password123')
    await page.locator('input#confirmPassword').fill('different123')
    await page.locator('button[type="submit"]').click()

    const toast = page.locator('[data-sonner-toast]')
    await expect(toast).toBeVisible({ timeout: 5000 })
    await expect(toast).toContainText('do not match')
  })

  test('should show error for short password', async ({ page, request }) => {
    const api = new ApiHelper(request)
    const orgId = await createOrgForRegister(api, 'short')
    test.skip(!orgId, 'Failed to set up test organization')

    await page.goto(`/register?org=${orgId}`)
    await page.locator('input#fullName').fill('Test User')
    await page.locator('input#email').fill(scope.email('short'))
    await page.locator('input#password').fill('short')
    await page.locator('input#confirmPassword').fill('short')
    await page.locator('button[type="submit"]').click()

    const toast = page.locator('[data-sonner-toast]')
    await expect(toast).toBeVisible({ timeout: 5000 })
    await expect(toast).toContainText('at least 8 characters')
  })

  // removed: navigate-to-login-from-invitation-required test (Rule 7);
  // pure vue-router navigation. The invitation-required gate itself is
  // covered by `should show invitation required message without org param`.

  // removed: navigate-to-login-from-registration-form test (Rule 7);
  // pure vue-router navigation, not product logic.
})
