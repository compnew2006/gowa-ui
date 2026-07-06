import {
  test,
  expect,
  request,
  type APIRequestContext,
  type Browser,
  type BrowserContext,
  type Page,
} from '@playwright/test'
import {
  loginAsAdmin,
  ApiHelper,
  generateUniqueEmail,
  generateUniqueName,
} from '../../helpers'
import { ChatPage } from '../../pages'

const E2E_TEST_PASSWORD = 'Password123!'
const ADMIN_EMAIL = 'admin@test.com'
const BACKEND_BASE_URL = process.env.BASE_URL || 'http://localhost:8080'

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function generatePhoneNumber(): string {
  return `91${Date.now().toString().slice(-10)}`
}

async function loginWithCredentials(
  page: Page,
  email: string,
  password: string,
) {
  await page.goto('/login')
  await page.locator('input[name="email"], input[type="email"]').fill(email)
  await page
    .locator('input[name="password"], input[type="password"]')
    .fill(password)
  await page.locator('button[type="submit"]').click()
  await page.waitForURL((url) => !url.pathname.includes('/login'), {
    timeout: 10000,
  })
}

async function createLoggedInSession(
  browser: Browser,
  email: string,
  password: string,
): Promise<{ context: BrowserContext; page: Page }> {
  const context = await browser.newContext()
  const page = await context.newPage()
  await loginWithCredentials(page, email, password)
  return { context, page }
}

function getSidebarEntry(page: Page, contactName: string) {
  return page
    .getByTestId('chat-sidebar-entry')
    .filter({ hasText: new RegExp(escapeRegExp(contactName), 'i') })
    .first()
}

async function createContactFixture(
  api: ApiHelper,
  prefix: string,
): Promise<{ id: string; name: string }> {
  const name = generateUniqueName(prefix)
  const contact = await api.createContact(generatePhoneNumber(), name)
  return {
    id: contact.id,
    name,
  }
}

test.describe('Chat Soft Delete - Playwright', () => {
  test.describe.configure({ mode: 'serial' })

  let adminApiContext: APIRequestContext
  let adminApi: ApiHelper

  const limitedRoleName = generateUniqueName('E2E Soft Delete Limited Role')
  const limitedUserEmail = generateUniqueEmail('e2e-soft-delete-limited')
  const limitedUserName = generateUniqueName('E2E Soft Delete Limited User')

  const softDeleteRoleName = generateUniqueName('E2E Soft Delete Role')
  const softDeleteUserEmail = generateUniqueEmail('e2e-soft-delete')
  const softDeleteUserName = generateUniqueName('E2E Soft Delete User')

  let limitedRoleId = ''
  let limitedUserId = ''
  let softDeleteRoleId = ''
  let softDeleteUserId = ''

  test.beforeAll(async () => {
    adminApiContext = await request.newContext({
      baseURL: BACKEND_BASE_URL,
    })
    adminApi = new ApiHelper(adminApiContext)
    await adminApi.loginAsAdmin()

    const limitedPermissions = await adminApi.findPermissionKeys([
      { resource: 'chat', action: 'read' },
      { resource: 'contacts', action: 'read' },
    ])
    const limitedRole = await adminApi.createRole({
      name: limitedRoleName,
      description: 'E2E role without soft delete permission',
      permissions: limitedPermissions,
    })
    limitedRoleId = limitedRole.id

    const limitedUser = await adminApi.createUser({
      email: limitedUserEmail,
      password: E2E_TEST_PASSWORD,
      full_name: limitedUserName,
      role_id: limitedRoleId,
    })
    limitedUserId = limitedUser.id

    const softDeletePermissions = await adminApi.findPermissionKeys([
      { resource: 'chat', action: 'read' },
      { resource: 'contacts', action: 'read' },
      { resource: 'contacts', action: 'soft_delete' },
    ])
    const softDeleteRole = await adminApi.createRole({
      name: softDeleteRoleName,
      description: 'E2E role with soft delete permission',
      permissions: softDeletePermissions,
    })
    softDeleteRoleId = softDeleteRole.id

    const softDeleteUser = await adminApi.createUser({
      email: softDeleteUserEmail,
      password: E2E_TEST_PASSWORD,
      full_name: softDeleteUserName,
      role_id: softDeleteRoleId,
    })
    softDeleteUserId = softDeleteUser.id
  })

  test.afterAll(async () => {
    if (limitedUserId) {
      await adminApi.deleteUser(limitedUserId).catch(() => {})
    }
    if (softDeleteUserId) {
      await adminApi.deleteUser(softDeleteUserId).catch(() => {})
    }
    if (limitedRoleId) {
      await adminApi.deleteRole(limitedRoleId).catch(() => {})
    }
    if (softDeleteRoleId) {
      await adminApi.deleteRole(softDeleteRoleId).catch(() => {})
    }

    await adminApiContext?.dispose()
  })

  test('user without contacts:soft_delete does not see hide affordances', async ({
    page,
  }) => {
    const contact = await createContactFixture(adminApi, 'Soft Delete Limited')

    await loginWithCredentials(page, limitedUserEmail, E2E_TEST_PASSWORD)
    const chatPage = new ChatPage(page)
    await chatPage.goto(contact.id)

    const sidebarEntry = getSidebarEntry(page, contact.name)
    await expect(sidebarEntry).toBeVisible()
    await expect(
      sidebarEntry.getByRole('button', { name: /hide chat|confirm hide chat/i }),
    ).toHaveCount(0)

    await page.locator('#info-button').click()
    await expect(page.locator('button[title="Hide chat"]')).toHaveCount(0)
  })

  test('user with contacts:soft_delete must confirm inline before sidebar hide completes', async ({
    page,
    browser,
  }) => {
    const contact = await createContactFixture(adminApi, 'Soft Delete Sidebar')

    await loginWithCredentials(page, softDeleteUserEmail, E2E_TEST_PASSWORD)
    const chatPage = new ChatPage(page)
    await chatPage.goto(contact.id)

    const sidebarEntry = getSidebarEntry(page, contact.name)
    await expect(sidebarEntry).toBeVisible()

    await sidebarEntry.getByRole('button', { name: /hide chat/i }).click()
    const confirmButton = sidebarEntry.getByRole('button', {
      name: /confirm hide chat/i,
    })
    await expect(confirmButton).toBeVisible()

    await confirmButton.click()
    await expect(page.getByText('Chat hidden')).toBeVisible()
    await expect(getSidebarEntry(page, contact.name)).toHaveCount(0)

    const { context: adminContext, page: adminPage } = await createLoggedInSession(
      browser,
      ADMIN_EMAIL,
      E2E_TEST_PASSWORD,
    )

    try {
      await adminPage.goto('/chat')
      await adminPage.getByTestId('notification-bell-button').click()

      const notificationItem = adminPage
        .locator('button')
        .filter({ hasText: new RegExp(escapeRegExp(softDeleteUserName), 'i') })
        .filter({ hasText: new RegExp(escapeRegExp(contact.name), 'i') })

      await expect(notificationItem).toBeVisible({ timeout: 10000 })
      await notificationItem.click()

      await expect(adminPage).toHaveURL(
        new RegExp(`/chat/${escapeRegExp(contact.id)}$`),
      )
    } finally {
      await adminContext.close()
    }
  })

  test('contact info panel flow accepts browser confirmation and hides the active chat', async ({
    page,
  }) => {
    const contact = await createContactFixture(adminApi, 'Soft Delete Panel')

    await loginWithCredentials(page, softDeleteUserEmail, E2E_TEST_PASSWORD)
    const chatPage = new ChatPage(page)
    await chatPage.goto(contact.id)

    await page.locator('#info-button').click()
    const panelSoftDeleteButton = page.locator('button[title="Hide chat"]')
    await expect(panelSoftDeleteButton).toBeVisible()

    const dialogPromise = page.waitForEvent('dialog')
    await panelSoftDeleteButton.click()
    const dialog = await dialogPromise
    expect(dialog.message()).toContain('hide this chat')
    await dialog.accept()

    await expect(page.getByText('Chat hidden')).toBeVisible()
    await expect(page).toHaveURL(/\/chat$/)
  })
})
