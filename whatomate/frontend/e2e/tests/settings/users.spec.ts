import { test, expect, request as playwrightRequest } from '@playwright/test'
import { TablePage, DialogPage } from '../../pages'
import { loginAsAdmin, login, createUserFixture, ApiHelper } from '../../helpers'

// const BASE_URL = process.env.BASE_URL || 'http://localhost:8080'

test.describe('Users Management', () => {
  let tablePage: TablePage
  let dialogPage: DialogPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/users')
    await page.waitForLoadState('networkidle')

    tablePage = new TablePage(page)
    dialogPage = new DialogPage(page)
  })

  test('should display users list', async () => {
    // Should show table with users
    await expect(tablePage.tableBody).toBeVisible()
    // At least the admin user should exist
    const rowCount = await tablePage.getRowCount()
    expect(rowCount).toBeGreaterThan(0)
  })

  test('should search users', async ({ page }) => {
    // Search by specific email to avoid multiple matches
    await tablePage.search('admin@test.com')
    // Should filter results
    await page.waitForTimeout(500)
    await tablePage.expectRowExists('admin@test.com')
  })

  test('should open create user dialog', async ({ page }) => {
    await page.getByRole('button', { name: /^Add User$/i }).click()
    await dialogPage.waitForOpen()
    await expect(dialogPage.dialog).toBeVisible()
  })

  test('should create a new user', async ({ page }) => {
    const newUser = createUserFixture()

    await page.getByRole('button', { name: /^Add User$/i }).click()
    await dialogPage.waitForOpen()

    await dialogPage.fillField('Email', newUser.email)
    await dialogPage.fillField('Name', newUser.fullName)
    await dialogPage.fillField('Password', newUser.password)
    await dialogPage.selectOption('Role', 'Agent')

    await dialogPage.submit()
    await dialogPage.waitForClose()

    // Verify user appears in list
    await tablePage.search(newUser.email)
    await tablePage.expectRowExists(newUser.email)
  })

  test('should show validation error for invalid email', async ({ page }) => {
    await page.getByRole('button', { name: /^Add User$/i }).click()
    await dialogPage.waitForOpen()

    await dialogPage.fillField('Email', 'invalid-email')
    await dialogPage.fillField('Name', 'Test User')
    await dialogPage.fillField('Password', 'password123')

    await dialogPage.submit()

    // Should show validation error and stay open
    await expect(dialogPage.dialog).toBeVisible()
  })

  test('should edit existing user', async ({ page }) => {
    // First create a user to edit
    const user = createUserFixture()

    await page.getByRole('button', { name: /^Add User$/i }).click()
    await dialogPage.waitForOpen()
    await dialogPage.fillField('Email', user.email)
    await dialogPage.fillField('Name', user.fullName)
    await dialogPage.fillField('Password', user.password)
    await dialogPage.selectOption('Role', 'Agent')
    await dialogPage.submit()
    await dialogPage.waitForClose()

    // Now edit the user
    await tablePage.search(user.email)
    await tablePage.editRow(user.email)
    await dialogPage.waitForOpen()

    const updatedName = 'Updated User Name'
    await dialogPage.fillField('Name', updatedName)
    await dialogPage.submit()
    await dialogPage.waitForClose()

    // Verify update
    await tablePage.expectRowExists(updatedName)
  })

  test('should delete user', async ({ page }) => {
    // First create a user to delete
    const user = createUserFixture({ fullName: 'User To Delete' })

    await page.getByRole('button', { name: /^Add User$/i }).click()
    await dialogPage.waitForOpen()
    await dialogPage.fillField('Email', user.email)
    await dialogPage.fillField('Name', user.fullName)
    await dialogPage.fillField('Password', user.password)
    await dialogPage.selectOption('Role', 'Agent')
    await dialogPage.submit()
    await dialogPage.waitForClose()

    // Search for the user
    await tablePage.search(user.email)
    await tablePage.expectRowExists(user.email)

    // Delete the user
    await tablePage.deleteRow(user.email)

    // Verify deletion
    await tablePage.clearSearch()
    await tablePage.search(user.email)
    await tablePage.expectRowNotExists(user.email)
  })

  test('should cancel user creation', async ({ page }) => {
    await page.getByRole('button', { name: /^Add User$/i }).click()
    await dialogPage.waitForOpen()

    await dialogPage.fillField('Email', 'cancelled@test.com')
    await dialogPage.cancel()

    await dialogPage.waitForClose()
    // User should not be created
    await tablePage.search('cancelled@test.com')
    await tablePage.expectRowNotExists('cancelled@test.com')
  })
})

test.describe('Users - Restrictions Controls', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/users')
    await page.waitForLoadState('networkidle')
  })

  test('should persist strict sending restrictions and unclaimed chat access toggles', async ({ page }) => {
    let strictSettingsPayload: any = null
    let sendRestrictionsPayload: any = null

    await page.route('**/api/org/settings', async (route) => {
      if (route.request().method() === 'PUT') {
        strictSettingsPayload = route.request().postDataJSON()
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            data: { settings: strictSettingsPayload || {} },
          }),
        })
        return
      }
      await route.fallback()
    })

    await page.route('**/api/users/*/send-restrictions', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            data: {
              enabled: false,
              include_all_contacts: false,
              authorized_numbers: [],
              allowed_instance_ids: ['aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'],
              prefix_agent_name: true,
              allow_unclaimed_chat_view: false,
              allow_unclaimed_chat_send: false,
            },
          }),
        })
        return
      }

      if (route.request().method() === 'PUT') {
        sendRestrictionsPayload = route.request().postDataJSON()
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'success',
            data: sendRestrictionsPayload,
          }),
        })
        return
      }

      await route.fallback()
    })

    const strictSwitch = page.getByRole('switch').first()
    const strictWasEnabled = (await strictSwitch.getAttribute('aria-checked')) === 'true'
    await strictSwitch.click()
    await page.getByRole('button', { name: /^Save$/i }).first().click()

    await expect.poll(() => strictSettingsPayload).not.toBeNull()
    expect(strictSettingsPayload?.strict_sending_restrictions_enabled).toBe(!strictWasEnabled)

    const manageRestrictionsButton = page
      .locator('table tbody tr')
      .first()
      .locator('td')
      .last()
      .locator('button')
      .first()
    await manageRestrictionsButton.click()

    const restrictionsDialog = page.getByRole('dialog').last()
    await expect(restrictionsDialog.getByText('Allow Unclaimed Chat View')).toBeVisible()
    const dialogSwitches = restrictionsDialog.getByRole('switch')
    const viewSwitch = dialogSwitches.nth(3)
    const sendSwitch = dialogSwitches.nth(4)

    await expect(viewSwitch).toHaveAttribute('aria-checked', 'false')
    await expect(sendSwitch).toHaveAttribute('aria-checked', 'false')

    await sendSwitch.click()
    await expect(viewSwitch).toHaveAttribute('aria-checked', 'true')
    await expect(sendSwitch).toHaveAttribute('aria-checked', 'true')

    await page.getByRole('button', { name: /^Save$/i }).last().click()

    await expect.poll(() => sendRestrictionsPayload).not.toBeNull()
    expect(sendRestrictionsPayload?.allow_unclaimed_chat_view).toBe(true)
    expect(sendRestrictionsPayload?.allow_unclaimed_chat_send).toBe(true)
  })
})

test.describe('Users - Role-based Access', () => {
  test.skip('agent should not access users page', async () => {
    // Skip: Role-based access control may be implemented differently
    // This test should be updated based on actual RBAC implementation
  })
})

test.describe('Users - Copy Invite Link', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/users')
    await page.waitForLoadState('networkidle')
  })

  test('should show copy invite link button', async ({ page }) => {
    const copyButton = page.getByRole('button', { name: /Copy Invite Link/i })
    await expect(copyButton).toBeVisible()
  })

  test('should copy invite link to clipboard', async ({ page, context }) => {
    // Grant clipboard permission
    await context.grantPermissions(['clipboard-read', 'clipboard-write'])

    const copyButton = page.getByRole('button', { name: /Copy Invite Link/i })
    await copyButton.click()

    // Should show success toast
    const toast = page.locator('[data-sonner-toast]')
    await expect(toast).toBeVisible({ timeout: 5000 })

    // Verify clipboard contains a registration URL with org param
    const clipboardText = await page.evaluate(() => navigator.clipboard.readText())
    expect(clipboardText).toContain('/register?org=')
  })
})

test.describe('Users - Add Existing User (Single Org)', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/users')
    await page.waitForLoadState('networkidle')
  })

  test('should hide add existing user button in single-org mode', async ({ page }) => {
    const addExistingButton = page.getByRole('button', { name: /Add Existing User/i })
    await expect(addExistingButton).not.toBeVisible()
  })
})

test.describe('Users - Add Existing User (Multi Org)', () => {
  // let tablePage: TablePage
  let testUserEmail: string
  let testUserId: string
  let testRoleId: string
  let secondOrgId: string
  const testPassword = 'Password123!'

  // Set up: create a user with organizations:assign permission in multiple orgs
  test.beforeAll(async () => {
    const reqContext = await playwrightRequest.newContext()
    const api = new ApiHelper(reqContext)
    await api.login('admin@admin.com', 'adminpassword12')

    // Create a role with organizations:assign + users:read (needed to view the page)
    const permissions = await api.findPermissionKeys([
      { resource: 'users', action: 'read' },
      { resource: 'users', action: 'write' },
      { resource: 'organizations', action: 'assign' },
    ])
    const role = await api.createRole({
      name: `E2E OrgAssign Role ${Date.now()}`,
      description: 'E2E test role with organizations:assign',
      permissions,
    })
    testRoleId = role.id

    // Create a test user with this role
    testUserEmail = `e2e-orgassign-${Date.now()}@test.com`
    const user = await api.createUser({
      email: testUserEmail,
      password: testPassword,
      full_name: 'E2E OrgAssign User',
      role_id: testRoleId,
    })
    testUserId = user.id

    // Create a second org and add the user to it
    const org = await api.createOrganization(`E2E Multi-Org ${Date.now()}`)
    secondOrgId = org.id
    await api.addOrgMember(testUserId, undefined, secondOrgId)

    await reqContext.dispose()
  })

  test.afterAll(async () => {
    const reqContext = await playwrightRequest.newContext()
    const api = new ApiHelper(reqContext)
    await api.login('admin@admin.com', 'adminpassword12')
    try { await api.removeOrgMember(testUserId, secondOrgId) } catch { /* ignore */ }
    try { await api.deleteUser(testUserId) } catch { /* ignore */ }
    try { await api.deleteRole(testRoleId) } catch { /* ignore */ }
    await reqContext.dispose()
  })

  test.beforeEach(async ({ page }) => {
    await login(page, { email: testUserEmail, password: testPassword, role: 'admin' })
    await page.goto('/settings/users')
    await page.waitForLoadState('networkidle')
    // tablePage = new TablePage(page)
  })

  test('should show add existing user button in multi-org mode', async ({ page }) => {
    const addExistingButton = page.getByRole('button', { name: /Add Existing User/i })
    await expect(addExistingButton).toBeVisible()
  })

  test('should open add existing user dialog', async ({ page }) => {
    const addExistingButton = page.getByRole('button', { name: /Add Existing User/i })
    await addExistingButton.click()

    // Dialog should appear
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()

    // Should have email input and role select
    await expect(dialog.locator('input[type="email"]')).toBeVisible()

    // Close dialog
    await dialog.getByRole('button', { name: /Cancel/i }).click()
    await expect(dialog).not.toBeVisible()
  })

  test('should show error for empty email in add existing dialog', async ({ page }) => {
    const addExistingButton = page.getByRole('button', { name: /Add Existing User/i })
    await addExistingButton.click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()

    // Try to submit without email — the submit button should be disabled
    const submitButton = dialog.getByRole('button', { name: /Add Existing User/i })
    await expect(submitButton).toBeDisabled()
  })
})

test.describe('Users - Table Sorting', () => {
  let tablePage: TablePage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/users')
    await page.waitForLoadState('networkidle')
    tablePage = new TablePage(page)
  })

  test('should have default sort by user name ascending', async () => {
    // UsersView defaults to sorting by full_name ascending
    await tablePage.expectSortDirection('User', 'asc')
  })

  test('should toggle sort direction on column click', async () => {
    // User column is already sorted ascending by default
    // First click toggles to descending
    await tablePage.clickColumnHeader('User')
    await tablePage.expectSortDirection('User', 'desc')

    // Second click toggles back to ascending
    await tablePage.clickColumnHeader('User')
    await tablePage.expectSortDirection('User', 'asc')
  })

  test('should sort by created date', async () => {
    await tablePage.clickColumnHeader('Created')
    const direction = await tablePage.getSortDirection('Created')
    expect(direction).not.toBeNull()
  })

  test('should sort by status', async () => {
    await tablePage.clickColumnHeader('Status')
    const direction = await tablePage.getSortDirection('Status')
    expect(direction).not.toBeNull()
  })

  test('should sort by role', async () => {
    await tablePage.clickColumnHeader('Role')
    const direction = await tablePage.getSortDirection('Role')
    expect(direction).not.toBeNull()
  })

  test('should change sort column when clicking different header', async () => {
    // User is already sorted ascending by default
    await tablePage.expectSortDirection('User', 'asc')

    // Click Created - switches to that column with desc direction
    await tablePage.clickColumnHeader('Created')
    await tablePage.expectSortDirection('Created', 'desc')

    // User column should no longer show sort indicator
    const userSort = await tablePage.getSortDirection('User')
    expect(userSort).toBeNull()
  })
})
