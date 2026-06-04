import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'

/**
 * Session-expired flow: when the access/refresh token becomes invalid, the
 * frontend must:
 *  1. Redirect to /login (soft, not a hard page reload)
 *  2. Show a "session expired" toast so the user understands what happened
 *  3. Clear the local user/legacy-token localStorage entries
 */
test.describe('Session Expired (401 redirect)', () => {
  test('should show session expired toast and redirect to login when token is invalid', async ({
    page,
    context,
  }) => {
    // 1. Login as admin
    await loginAsAdmin(page)
    await expect(page).not.toHaveURL(/\/login/)

    // 2. Simulate token expiry: drop all auth cookies and any cached local user
    await context.clearCookies()
    await page.evaluate(() => {
      localStorage.removeItem('user')
      localStorage.removeItem('auth_token')
      localStorage.removeItem('refresh_token')
    })

    // 3. Trigger a re-evaluation of auth state. Reloading causes the router
    //    guard to call restoreSession() which calls /api/me. The /api/me call
    //    will get 401, the response interceptor will try /api/auth/refresh,
    //    which also fails, and the session-expired handler should fire.
    await page.goto('/chat').catch(() => {})

    // 4. Should redirect to /login (soft, not a hard reload)
    await expect(page).toHaveURL(/\/login(?:\?.*)?$/, { timeout: 15000 })

    // 5. Should show a "session expired" toast (en + ar copy)
    const sessionExpiredToast = page
      .locator('[data-sonner-toast]')
      .filter({ hasText: /session has expired|انتهت صلاحية/i })
    await expect(sessionExpiredToast).toBeVisible({ timeout: 5000 })

    // 6. The cached user must be cleared from localStorage
    const cachedUser = await page.evaluate(() => localStorage.getItem('user'))
    expect(cachedUser).toBeNull()
  })
})
