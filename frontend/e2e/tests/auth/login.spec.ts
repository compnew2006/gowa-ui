import { test, expect } from '@playwright/test'
import { LoginPage } from '../../pages'
import { TEST_USERS, logout } from '../../helpers'

test.describe('Login', () => {
  let loginPage: LoginPage

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page)
    await loginPage.goto()
  })

  // removed: form-render test (Rule 7); covered by login-success/error tests.

  test('should login successfully with valid credentials', async ({ page }) => {
    await loginPage.login(TEST_USERS.admin.email, TEST_USERS.admin.password)
    await loginPage.expectLoginSuccess()
  })

  test('should show error with invalid credentials', async ({ page }) => {
    await loginPage.login('invalid@test.com', 'wrongpassword')
    await loginPage.expectLoginError()
  })

  // Rule 3: merged empty-email and empty-password validation tests into
  // one that submits an empty form (both fields blank) and asserts the
  // page stays on /login.
  test('should show validation error for empty fields', async ({ page }) => {
    // Leave both fields empty and submit.
    await loginPage.submitButton.click()
    // Should stay on login page
    await expect(page).toHaveURL(/\/login/)
  })

  // removed: navigate-to-register test (Rule 7); tests vue-router, not product logic.

  test('should logout successfully', async ({ page }) => {
    // First login
    await loginPage.login(TEST_USERS.admin.email, TEST_USERS.admin.password)
    await loginPage.expectLoginSuccess()

    // Then logout
    await logout(page)
    await expect(page).toHaveURL(/\/login/)
  })
})

test.describe('Authentication Redirect', () => {
  test('should redirect to login when accessing protected route', async ({ page }) => {
    await page.goto('/settings/users')
    await expect(page).toHaveURL(/\/login/)
  })

  test('should redirect to dashboard after login', async ({ page }) => {
    const loginPage = new LoginPage(page)
    await loginPage.goto()
    await loginPage.login(TEST_USERS.admin.email, TEST_USERS.admin.password)
    // Rule 4: was `expect(page).toHaveURL(/\/(dashboard|chat)?/)` — the
    // whole group is optional, so the regex matches ANY URL (including
    // /login). Tightened to mirror the expectLoginSuccess helper in
    // helpers/auth.ts: assert we are no longer on /login.
    await expect(page).not.toHaveURL(/\/login/)
  })
})
