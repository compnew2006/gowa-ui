import { test, expect, type Page, type Locator } from '@playwright/test'
import { TablePage } from '../../pages'
import { loginAsAdmin, verifyAuditLogged } from '../../helpers'
import { createTestScope } from '../../framework'

const scope = createTestScope('roles')

// Read the column header's aria-sort attribute directly. The TablePage POM
// exposes icon-based sort detection, but asserting aria-sort is the stronger,
// accessible contract (and toggling asc/desc on click is what we want to
// prove). This is a spec-local reader, not a POM edit.
async function getAriaSort(page: Page, column: string): Promise<'ascending' | 'descending' | null> {
  const header = page.locator('thead th').filter({ hasText: column }).first()
  if (await header.count() === 0) return null
  const value = await header.getAttribute('aria-sort')
  if (value === 'ascending' || value === 'descending') return value
  return null
}

// Extract a role id from a /settings/roles/:id detail URL.
function roleIdFromUrl(page: Page): string | null {
  const match = page.url().match(/\/settings\/roles\/([a-f0-9-]+)$/)
  return match ? match[1] : null
}

// Detail-page form helpers
function nameInput(page: Page): Locator {
  return page.getByPlaceholder('e.g., Support Lead')
}

function descriptionInput(page: Page): Locator {
  return page.getByPlaceholder('Describe what this role is for...')
}

function saveButton(page: Page): Locator {
  return page.getByRole('button', { name: /^(Create|Save)$/i }).first()
}

async function gotoCreateRole(page: Page) {
  await page.getByRole('button', { name: /^Add Role$/i }).first().click()
  await page.waitForURL(/\/settings\/roles\/new$/)
  await page.waitForLoadState('networkidle')
}

async function openRoleDetail(tablePage: TablePage, page: Page, rowText: string) {
  await tablePage.search(rowText)
  // Click the exact-name link to avoid colliding with rows whose names share the prefix
  // (e.g. "admin" vs "admin_fe9fb00b" left over from earlier runs).
  await page.locator('tbody tr .font-medium').getByText(rowText, { exact: true }).first().click()
  await page.waitForURL(/\/settings\/roles\/[a-f0-9-]+$/)
  await page.waitForLoadState('networkidle')
}

test.describe('Roles Management', () => {
  let tablePage: TablePage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')

    tablePage = new TablePage(page)
  })

  test('should display roles list', async ({ page }) => {
    // Should show table with roles
    await expect(tablePage.tableBody).toBeVisible()
    // System roles (admin, manager, agent) should exist
    const rowCount = await tablePage.getRowCount()
    expect(rowCount).toBeGreaterThan(0)
  })

  test('should show system roles with badges', async ({ page }) => {
    // System roles should have a "System" badge
    await expect(page.locator('text=System').first()).toBeVisible()
  })

  test('should search roles', async ({ page }) => {
    await tablePage.search('admin')
    await page.waitForTimeout(500)
    // The system "admin" role must be present; match the name span exactly so we
    // don't collide with custom roles whose names contain "admin".
    await expect(
      page.locator('tbody tr .font-medium').getByText('admin', { exact: true }).first()
    ).toBeVisible()
  })

  test('should navigate to create role page', async ({ page }) => {
    await gotoCreateRole(page)
    expect(page.url()).toContain('/settings/roles/new')
    // Name input and permissions heading should be visible
    await expect(nameInput(page)).toBeVisible()
    await expect(page.getByText(/^Permissions$/).first()).toBeVisible()
  })

  test('should create a new custom role', async ({ page, request }) => {
    const roleName = scope.name('test')

    await gotoCreateRole(page)
    await nameInput(page).fill(roleName)
    await descriptionInput(page).fill('A custom test role for E2E testing')

    // Select a permission (first checkbox in the matrix)
    const permissionCheckbox = page.locator('button[role="checkbox"]').first()
    if (await permissionCheckbox.isVisible()) {
      await permissionCheckbox.click()
    }

    await saveButton(page).click()
    await page.waitForLoadState('networkidle')

    // Capture the new role's id from the detail-page URL for the audit assertion.
    const roleId = roleIdFromUrl(page)
    expect(roleId, 'expected to land on the new role detail page').not.toBeNull()

    // Verify role appears in list
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')
    await tablePage.search(roleName)
    await tablePage.expectRowExists(roleName)

    // API side-channel: confirm the audit trail recorded the creation.
    if (roleId) {
      await verifyAuditLogged(request, 'role', roleId, 'created')
    }
  })

  test('should require role name', async ({ page }) => {
    await gotoCreateRole(page)

    // Fill description but leave name empty
    await descriptionInput(page).fill('Role without name')
    await saveButton(page).click()

    // Error toast should appear (we stay on /new)
    const toast = page.locator('[data-sonner-toast]')
    await expect(toast).toBeVisible({ timeout: 5000 })
    expect(page.url()).toContain('/settings/roles/new')
  })

  test('should view system role details (read-only)', async ({ page }) => {
    // Open the system admin role's detail page
    await openRoleDetail(tablePage, page, 'admin')

    // Name input should be disabled for system roles
    const input = nameInput(page)
    await expect(input).toBeDisabled()

    // System badge should be shown
    await expect(page.locator('text=System').first()).toBeVisible()
  })

  test('should edit custom role', async ({ page, request }) => {
    // Create a role via the detail page
    const originalName = scope.name('edit')
    await gotoCreateRole(page)
    await nameInput(page).fill(originalName)
    await descriptionInput(page).fill('Original description')
    await saveButton(page).click()
    await page.waitForLoadState('networkidle')

    // Navigate back to list and open edit for the new role
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')
    await openRoleDetail(tablePage, page, originalName)

    const roleId = roleIdFromUrl(page)

    const updatedName = scope.name('updated')
    await nameInput(page).fill(updatedName)
    await descriptionInput(page).fill('Updated description')
    await page.waitForTimeout(300)
    await saveButton(page).click()
    await page.waitForLoadState('networkidle')

    // Verify update appears in the list
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')
    await tablePage.search(updatedName)
    await tablePage.expectRowExists(updatedName)

    // API side-channel: confirm the audit trail recorded the update.
    if (roleId) {
      await verifyAuditLogged(request, 'role', roleId, 'updated')
    }
  })

  test('should delete custom role', async ({ page, request }) => {
    // Create a role via the detail page
    const roleName = scope.name('delete')
    await gotoCreateRole(page)
    await nameInput(page).fill(roleName)
    await saveButton(page).click()
    await page.waitForLoadState('networkidle')

    const roleId = roleIdFromUrl(page)

    // Navigate back and delete from the list
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')
    await tablePage.search(roleName)
    await tablePage.expectRowExists(roleName)
    await tablePage.deleteRow(roleName)

    // Verify deletion
    await tablePage.clearSearch()
    await tablePage.search(roleName)
    await tablePage.expectRowNotExists(roleName)

    // API side-channel: confirm the audit trail recorded the deletion.
    if (roleId) {
      await verifyAuditLogged(request, 'role', roleId, 'deleted')
    }
  })

  test('should not allow deleting system roles', async ({ page }) => {
    // System roles should not have a delete button
    await tablePage.search('admin')

    // The delete button should be hidden or disabled for system roles
    const deleteButton = page.locator(`tr:has-text("admin") button:has-text("Delete")`)
    await expect(deleteButton).not.toBeVisible().catch(async () => {
      // If visible, it should be disabled
      await expect(deleteButton).toBeDisabled()
    })
  })

  test('should show delete confirmation when deleting custom role', async ({ page }) => {
    // Create a role via the detail page
    const roleName = scope.name('confirm-delete')
    await gotoCreateRole(page)
    await nameInput(page).fill(roleName)
    await saveButton(page).click()
    await page.waitForLoadState('networkidle')

    // Search for the role in the list
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')
    await tablePage.search(roleName)
    await tablePage.expectRowExists(roleName)

    // Click delete and verify confirmation dialog appears
    const deleteButton = page.locator(`tr:has-text("${roleName}") button:has(svg.text-destructive)`)
    await deleteButton.click()

    // Should show confirmation dialog
    const alertDialog = page.locator('[role="alertdialog"]')
    await expect(alertDialog).toBeVisible()
    await expect(alertDialog).toContainText('delete')

    // Cancel to clean up
    await alertDialog.getByRole('button', { name: 'Cancel' }).click()
  })

  test('should toggle default role flag', async ({ page }) => {
    // Create a role and set it as default via the detail page
    const roleName = scope.name('default')

    await gotoCreateRole(page)
    await nameInput(page).fill(roleName)

    // Toggle the default switch (only one switch on the create page)
    const defaultSwitch = page.locator('button[role="switch"]').first()
    await defaultSwitch.click()

    await saveButton(page).click()
    await page.waitForLoadState('networkidle')

    // Verify the role shows default badge in the list
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')
    await tablePage.search(roleName)
    const defaultBadge = page.locator(`tr:has-text("${roleName}") .rounded-full`).filter({ hasText: 'Default' })
    await expect(defaultBadge).toBeVisible()
  })

  test('should abandon role creation via unsaved changes dialog', async ({ page }) => {
    await gotoCreateRole(page)
    await nameInput(page).fill('Cancelled Role')
    await page.waitForTimeout(300)

    // Clicking the back link triggers the unsaved-changes guard
    await page.goto('/settings/roles')

    const leaveDialog = page.locator('[role="alertdialog"]')
    if (await leaveDialog.isVisible({ timeout: 2000 }).catch(() => false)) {
      await leaveDialog.getByRole('button', { name: /Leave/i }).click()
    }

    await page.waitForURL('**/settings/roles', { timeout: 5000 }).catch(() => {})
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')

    // Role should not be created
    await tablePage.search('Cancelled Role')
    await tablePage.expectRowNotExists('Cancelled Role')
  })

  test('should display permission count in role list', async ({ page }) => {
    // Roles should show permission count
    const permissionBadge = page.locator('tr td >> text=/\\d+/').first()
    await expect(permissionBadge).toBeVisible()
  })

  test('should navigate to roles from settings', async ({ page }) => {
    // Go to settings first
    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    // Click on Roles card/link
    await page.locator('text=Roles').click()

    // Should be on roles page
    await expect(page).toHaveURL(/\/settings\/roles/)
    await expect(page.locator('h1:has-text("Roles")')).toBeVisible()
  })
})

test.describe('Roles - Table Sorting', () => {
  let tablePage: TablePage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')
    tablePage = new TablePage(page)
  })

  test('roles table sorts by each column', async ({ page }) => {
    // Collapsed from four near-identical tests (sort by Role / Description /
    // Users / Created). Each used to assert only that the sort indicator was
    // non-null; this loop additionally asserts the aria-sort attribute toggles
    // between ascending and descending on successive clicks.
    for (const column of ['Role', 'Description', 'Users', 'Created']) {
      await tablePage.clickColumnHeader(column)
      const firstSort = await getAriaSort(page, column)
      expect(firstSort, `${column}: expected a sort direction after first click`).not.toBeNull()

      await tablePage.clickColumnHeader(column)
      const secondSort = await getAriaSort(page, column)
      expect(secondSort, `${column}: expected a sort direction after second click`).not.toBeNull()
      expect(secondSort, `${column}: sort should toggle on second click`).not.toEqual(firstSort)
    }
  })

  test('should toggle sort direction', async () => {
    await tablePage.clickColumnHeader('Role')
    const firstDirection = await tablePage.getSortDirection('Role')

    await tablePage.clickColumnHeader('Role')
    const secondDirection = await tablePage.getSortDirection('Role')

    expect(firstDirection).not.toEqual(secondDirection)
  })
})

test.describe('Roles - Permissions Selection', () => {
  test('should display permission groups in accordion', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')

    await gotoCreateRole(page)

    // Should show permission groups (Users, Contacts, Messages, etc.)
    await expect(page.locator('text=Users').first()).toBeVisible()
    await expect(page.locator('text=Contacts').first()).toBeVisible()
  })

  test('should select all permissions in a group', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/roles')
    await page.waitForLoadState('networkidle')

    await gotoCreateRole(page)

    // Click the group checkbox to select all
    const groupCheckbox = page.locator('[data-testid="group-users-checkbox"]').or(
      page.locator('button[role="checkbox"]').first()
    )
    await groupCheckbox.click()

    // Permission count should increase
    const selectedCount = page.locator('text=/\\d+ selected/')
    await expect(selectedCount).toBeVisible()
  })
})
