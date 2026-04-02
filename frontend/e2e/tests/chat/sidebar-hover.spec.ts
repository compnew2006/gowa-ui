import { test, expect, type Page } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'

const SIDEBAR_PINNED_STORAGE_KEY = 'layout.sidebarPinnedOpen'

async function prepareChatShell(page: Page) {
  await page.goto('/login')
  await page.evaluate((storageKey) => localStorage.removeItem(storageKey), SIDEBAR_PINNED_STORAGE_KEY)
  await loginAsAdmin(page)
  await page.goto('/chat')
  await page.waitForLoadState('networkidle')
}

test.describe('Chat Sidebar Hover Behavior', () => {
  test('desktop sidebar expands on hover without changing the reserved main offset', async ({ page }) => {
    await prepareChatShell(page)

    const sidebar = page.getByTestId('app-sidebar')
    const main = page.locator('main')

    await expect(sidebar).toHaveAttribute('data-sidebar-state', 'collapsed')

    const paddingBefore = await main.evaluate((element) => getComputedStyle(element).paddingLeft)

    await sidebar.hover()

    await expect(sidebar).toHaveAttribute('data-sidebar-state', 'expanded')
    await expect(sidebar.getByRole('menuitem', { name: 'Chat', exact: true })).toBeVisible()

    const paddingAfter = await main.evaluate((element) => getComputedStyle(element).paddingLeft)
    expect(paddingAfter).toBe(paddingBefore)
  })

  test('desktop sidebar pin toggle keeps the rail open until it is explicitly unpinned', async ({ page }) => {
    await prepareChatShell(page)

    const sidebar = page.getByTestId('app-sidebar')
    const pinToggle = page.getByTestId('sidebar-pin-toggle')

    await sidebar.hover()
    await expect(sidebar).toHaveAttribute('data-sidebar-state', 'expanded')

    await pinToggle.click()
    await expect(sidebar).toHaveAttribute('data-sidebar-pinned', 'true')

    await page.locator('main').hover()
    await expect(sidebar).toHaveAttribute('data-sidebar-state', 'expanded')

    await pinToggle.click()
    await expect(sidebar).toHaveAttribute('data-sidebar-pinned', 'false')

    await page.evaluate(() => {
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur()
      }
    })

    await page.locator('main').hover()
    await expect(sidebar).toHaveAttribute('data-sidebar-state', 'collapsed')
  })

  test('mobile menu still opens and closes the chat sidebar drawer', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await prepareChatShell(page)

    const sidebar = page.getByTestId('app-sidebar')
    const menuToggle = page.getByRole('button', { name: /toggle menu/i })

    await expect(sidebar).toHaveAttribute('data-sidebar-state', 'collapsed')

    await menuToggle.click()
    await expect(sidebar).toHaveAttribute('data-sidebar-state', 'expanded')

    await page.getByTestId('mobile-menu-overlay').click()
    await expect(sidebar).toHaveAttribute('data-sidebar-state', 'collapsed')
  })
})
