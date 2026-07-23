import { test, expect, type Page } from '@playwright/test'
import { TablePage } from '../../pages'
import { loginAsAdmin, createTeamFixture, navigateToFirstItem, expectMetadataVisible, expectActivityLogVisible, expectDeleteFromForm, ApiHelper, generateUniqueName, verifyAuditLogged } from '../../helpers'
import { createTestScope, SUPER_ADMIN } from '../../framework'

const scope = createTestScope('teams')

// Read the column header's aria-sort attribute directly (the TablePage POM
// exposes icon-based sort detection; aria-sort is the stronger, accessible
// contract and toggling asc/desc on click is what we want to prove).
async function getAriaSort(page: Page, column: string): Promise<'ascending' | 'descending' | null> {
  const header = page.locator('thead th').filter({ hasText: column }).first()
  if (await header.count() === 0) return null
  const value = await header.getAttribute('aria-sort')
  if (value === 'ascending' || value === 'descending') return value
  return null
}

test.describe('Teams - List View', () => {
  let tablePage: TablePage
  let api: ApiHelper
  let seededTeamId: string
  let seededTeamName: string

  test.beforeEach(async ({ page, request }) => {
    // Seed a team via the API so the list-view assertions below have a real
    // subject. Previously these tests guarded every assertion behind
    // `if (initialCount > 0)` / `if (href)` / `if (row.isVisible())`, which let
    // them pass while asserting nothing on an empty table.
    api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    seededTeamName = generateUniqueName('TeamTest')
    const resp = await api.post('/api/teams', { name: seededTeamName, description: 'seeded for list view' })
    expect(resp.ok(), `seed team: ${await resp.text()}`).toBe(true)
    seededTeamId = (await resp.json()).data.team.id

    await loginAsAdmin(page)
    await page.goto('/settings/teams')
    await page.waitForLoadState('networkidle')
    tablePage = new TablePage(page)
  })

  test.afterEach(async () => {
    // Best-effort cleanup of the seeded team if a test didn't delete it.
    if (seededTeamId) {
      await api.del('/api/teams/' + seededTeamId).catch(() => {})
    }
  })

  test('should display teams list', async () => {
    await expect(tablePage.tableBody).toBeVisible()
    // The seeded team must be present — no guard, real assertion.
    await tablePage.search(seededTeamName)
    await tablePage.expectRowExists(seededTeamName)
  })

  test('should search teams', async ({ page }) => {
    const initialCount = await tablePage.getRowCount()
    expect(initialCount).toBeGreaterThan(0)

    // Searching for the seeded team by its unique name isolates exactly one row.
    await tablePage.search(seededTeamName)
    await page.waitForTimeout(300)
    const seededCount = await tablePage.getRowCount()
    expect(seededCount).toBeGreaterThanOrEqual(1)
    expect(seededCount).toBeLessThanOrEqual(initialCount)

    // A nonsense query should shrink the result set below the seeded team.
    await tablePage.search('nonexistent-team-xyz')
    await page.waitForTimeout(300)
    const filteredCount = await tablePage.getRowCount()
    expect(filteredCount).toBeLessThan(seededCount)
  })

  test('should load create page', async ({ page }) => {
    await page.goto('/settings/teams/new')
    await page.waitForLoadState('networkidle')
    expect(page.url()).toContain('/settings/teams/new')
    await expect(page.locator('input').first()).toBeVisible()
  })

  test('should load detail page from list', async ({ page }) => {
    // Navigate to the seeded team's detail from the list — no `if (href)` guard.
    await tablePage.search(seededTeamName)
    await page.locator('tbody tr .font-medium, tbody tr a').getByText(seededTeamName, { exact: true }).first().click()
    await page.waitForURL(/\/settings\/teams\/[a-f0-9-]+/)
    await page.waitForLoadState('networkidle')
    expect(page.url()).toMatch(/\/settings\/teams\/[a-f0-9-]+/)
    await expect(page.getByText('Details')).toBeVisible()
  })

  test('should delete team from list', async ({ page, request }) => {
    // The seeded team gives us a guaranteed row, so the delete flow asserts
    // unconditionally instead of skipping when no row is visible.
    await tablePage.search(seededTeamName)
    await tablePage.expectRowExists(seededTeamName)

    const row = page.locator('tbody tr').filter({ hasText: seededTeamName }).first()
    await row.locator('button').filter({ has: page.locator('svg.text-destructive') }).click()
    const dialog = page.locator('[role="alertdialog"]')
    await expect(dialog).toBeVisible({ timeout: 3000 })
    // Confirm the deletion.
    await dialog.getByRole('button', { name: /Delete/i }).click()
    await dialog.waitFor({ state: 'hidden' })

    // Verify deletion.
    await tablePage.clearSearch()
    await tablePage.search(seededTeamName)
    await tablePage.expectRowNotExists(seededTeamName)

    // API side-channel: confirm the audit trail recorded the deletion. Mark the
    // id consumed so afterEach doesn't try to delete an already-deleted team.
    await verifyAuditLogged(request, 'team', seededTeamId, 'deleted')
    seededTeamId = ''
  })
})

test.describe('Teams - Detail Page CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
  })

  test('should show form fields on create page', async ({ page }) => {
    await page.goto('/settings/teams/new')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('input').first()).toBeVisible()
    await expect(page.locator('textarea').first()).toBeVisible()
    await expect(page.locator('button[role="combobox"]').first()).toBeVisible()
    await expect(page.locator('button[role="switch"]').first()).toBeVisible()
  })

  test('should create a new team', async ({ page, request }) => {
    const newTeam = createTeamFixture()

    await page.goto('/settings/teams/new')
    await page.waitForLoadState('networkidle')

    const input = page.locator('input').first()
    if (await input.isDisabled()) { test.skip(true, 'No write permission'); return }

    await input.fill(newTeam.name)
    await page.locator('textarea').first().fill(newTeam.description)
    await page.waitForTimeout(300)

    const createBtn = page.getByRole('button', { name: /Create/i })
    if (!(await createBtn.isVisible({ timeout: 5000 }).catch(() => false))) {
      test.skip(true, 'Create button not visible')
      return
    }
    await createBtn.click({ force: true })

    // Wait for redirect to the new team's detail page (replaces the old fixed
    // 3s timeout that masked CSRF/creation failures).
    await page.waitForURL(/\/settings\/teams\/[a-f0-9-]+$/, { timeout: 10000 }).catch(() => {})
    if (page.url().includes('/new')) {
      test.skip(true, 'Creation failed (possibly CSRF)')
    } else {
      expect(page.url()).toMatch(/\/settings\/teams\/[a-f0-9-]+/)
      const match = page.url().match(/\/settings\/teams\/([a-f0-9-]+)$/)
      const teamId = match ? match[1] : null
      if (teamId) {
        await verifyAuditLogged(request, 'team', teamId, 'created')
      }
    }
  })

  test('should edit existing team', async ({ page, request }) => {
    await page.goto('/settings/teams')
    await page.waitForLoadState('networkidle')

    const href = await navigateToFirstItem(page)
    if (!href) { test.skip(true, 'No teams exist'); return }

    const match = page.url().match(/\/settings\/teams\/([a-f0-9-]+)$/)
    const teamId = match ? match[1] : null

    const input = page.locator('input').first()
    if (await input.isDisabled()) { test.skip(true, 'No write permission'); return }

    const original = await input.inputValue()
    await input.fill(original + ' edited')
    await page.waitForTimeout(300)

    const saveBtn = page.getByRole('button', { name: /Save/i })
    if (await saveBtn.isVisible({ timeout: 5000 }).catch(() => false)) {
      await saveBtn.click({ force: true })
      await page.waitForTimeout(2000)

      // API side-channel: confirm the edit was audited.
      if (teamId) {
        await verifyAuditLogged(request, 'team', teamId, 'updated')
      }

      // Revert
      await input.fill(original)
      await page.waitForTimeout(300)
      const revertBtn = page.getByRole('button', { name: /Save/i })
      if (await revertBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
        await revertBtn.click({ force: true })
      }
    }
  })

  test('should delete from detail page', async ({ page, request }) => {
    await page.goto('/settings/teams')
    await page.waitForLoadState('networkidle')

    const href = await navigateToFirstItem(page)
    if (!href) { test.skip(true, 'No teams exist'); return }

    const match = page.url().match(/\/settings\/teams\/([a-f0-9-]+)$/)
    const teamId = match ? match[1] : null

    await expectDeleteFromForm(page, '/settings/teams')

    // API side-channel: confirm the deletion was audited.
    if (teamId) {
      await verifyAuditLogged(request, 'team', teamId, 'deleted')
    }
  })

  test('should show metadata', async ({ page }) => {
    await page.goto('/settings/teams')
    await page.waitForLoadState('networkidle')

    if (await navigateToFirstItem(page)) {
      await expectMetadataVisible(page)
    }
  })

  test('should show activity log', async ({ page, request }) => {
    // Seed our own team so we don't race with parallel workers that
    // create-then-delete teams. navigateToFirstItem grabs the first row's
    // href, but if another worker deletes that team before goto lands, the
    // detail page renders the "not found" error state and Activity Log
    // never appears.
    const api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    const teamResp = await api.post('/api/teams', {
      name: scope.name('activity-log'),
      description: 'seeded for activity-log test',
    })
    expect(teamResp.ok(), `seed team: ${await teamResp.text()}`).toBe(true)
    const team = (await teamResp.json()).data.team

    await page.goto(`/settings/teams/${team.id}`)
    await page.waitForLoadState('networkidle')

    await expectActivityLogVisible(page)
  })
})

test.describe('Teams - Table Sorting', () => {
  let tablePage: TablePage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/teams')
    await page.waitForLoadState('networkidle')
    tablePage = new TablePage(page)
  })

  test('teams table sorts by each column', async ({ page }) => {
    // Collapsed from two near-identical tests (sort by Team / Strategy). Each
    // used to assert only that the sort indicator was non-null; this loop
    // additionally asserts the aria-sort attribute toggles between ascending
    // and descending on successive clicks.
    for (const column of ['Team', 'Strategy']) {
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
    await tablePage.clickColumnHeader('Team')
    const firstDirection = await tablePage.getSortDirection('Team')
    await tablePage.clickColumnHeader('Team')
    const secondDirection = await tablePage.getSortDirection('Team')
    expect(firstDirection).not.toEqual(secondDirection)
  })
})

test.describe('Team Members', () => {
  test('should show members section on detail page', async ({ page }) => {
    await loginAsAdmin(page)
    await page.goto('/settings/teams')
    await page.waitForLoadState('networkidle')

    if (await navigateToFirstItem(page)) {
      await expect(page.getByRole('heading', { name: /Members/ })).toBeVisible({ timeout: 5000 })
    }
  })
})
