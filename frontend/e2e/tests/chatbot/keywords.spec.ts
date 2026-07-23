import { test, expect, type APIRequestContext } from '@playwright/test'
import { loginAsAdmin, navigateToFirstItem, expectMetadataVisible, expectActivityLogVisible, expectDeleteFromForm, verifyAuditLogged, ApiHelper } from '../../helpers'
import { KeywordsPage } from '../../pages'
import { createTestScope, SUPER_ADMIN } from '../../framework'

const scope = createTestScope('keywords')

// Seed a keyword rule via the API. Returns the new resource's ID, or null on
// failure. Replaces the old UI-based seed which silently skipped when
// permissions or click-timing didn't cooperate. Also asserts the audit log
// recorded the create — per ARCHITECTURE.md every CRUD mutation gets a
// verifyAuditLogged side-channel.
async function seedKeywordRuleViaAPI(request: APIRequestContext): Promise<string | null> {
  const api = new ApiHelper(request)
  await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
  const resp = await api.post('/api/chatbot/keywords', {
    name: scope.name('seed'),
    keywords: [`seedkw-${Date.now()}`],
    match_type: 'contains',
    response_type: 'text',
    response_content: { body: 'E2E seeded response' },
    enabled: true,
  })
  if (!resp.ok()) return null
  const body = await resp.json()
  const id = body.data?.id ?? null
  if (id) {
    await verifyAuditLogged(request, 'keyword_rule', id, 'created')
  }
  return id
}

// Variant that exposes the seeded name so search tests can target it
// deterministically instead of asserting `<= 50` rows on a negative search.
async function seedNamedKeywordRuleViaAPI(request: APIRequestContext, name: string): Promise<string | null> {
  const api = new ApiHelper(request)
  await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
  const resp = await api.post('/api/chatbot/keywords', {
    name,
    keywords: [`seedkw-${Date.now()}`],
    match_type: 'contains',
    response_type: 'text',
    response_content: { body: 'E2E seeded response' },
    enabled: true,
  })
  if (!resp.ok()) return null
  const body = await resp.json()
  const id = body.data?.id ?? null
  if (id) {
    await verifyAuditLogged(request, 'keyword_rule', id, 'created')
  }
  return id
}

test.describe('Keyword Rules - List View', () => {
  let keywordsPage: KeywordsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    keywordsPage = new KeywordsPage(page)
    await keywordsPage.goto()
  })

  test('should display keywords page', async () => {
    await keywordsPage.expectPageVisible()
  })

  test('should have search input', async () => {
    await expect(keywordsPage.searchInput).toBeVisible()
  })

  test('should load create page', async ({ page }) => {
    await page.goto('/chatbot/keywords/new')
    await page.waitForLoadState('networkidle')
    expect(page.url()).toContain('/chatbot/keywords/new')
    await expect(page.locator('input').first()).toBeVisible()
  })

  test('should load detail page from list', async ({ page, request }) => {
    let href = await navigateToFirstItem(page)
    if (!href) {
      const id = await seedKeywordRuleViaAPI(request)
      expect(id, 'API seed must succeed').toBeTruthy()
      await page.goto('/chatbot/keywords')
      await page.waitForLoadState('networkidle')
      href = await navigateToFirstItem(page)
    }
    expect(href).toBeTruthy()
    expect(page.url()).toMatch(/\/chatbot\/keywords\/[a-f0-9-]+/)
  })

  test('should search and filter', async ({ page, request }) => {
    // Positive case: seed a uniquely-named rule, search for that exact
    // name, and assert it is the only visible row.
    const uniqueName = scope.name('search-target')
    const id = await seedNamedKeywordRuleViaAPI(request, uniqueName)
    expect(id, 'API seed must succeed').toBeTruthy()
    await keywordsPage.goto()
    await keywordsPage.search(uniqueName)
    // TODO(test-guard): move to KeywordsPage POM (raw table-row locator)
    const matchingRows = page.locator('tbody tr')
    await expect(matchingRows).toHaveCount(1)
    await expect(matchingRows.first()).toContainText(uniqueName)

    // Negative case: a string that matches nothing must yield zero rows.
    await keywordsPage.search('nonexistent-keyword-xyz')
    await expect(page.locator('tbody tr')).toHaveCount(0)
  })

  test('should show delete confirmation from list', async ({ page, request }) => {
    // Always seed via API — under parallel workers, relying on pre-existing
    // rows is racy because another worker may delete them mid-test.
    const id = await seedKeywordRuleViaAPI(request)
    expect(id, 'API seed must succeed').toBeTruthy()
    await keywordsPage.goto()

    const deleteBtn = page.locator('tbody').getByRole('button', { name: /delete/i }).first()
    await expect(deleteBtn).toBeVisible({ timeout: 10000 })
    await deleteBtn.click()
    await expect(keywordsPage.alertDialog).toBeVisible({ timeout: 5000 })
    await keywordsPage.alertDialog.getByRole('button', { name: /Cancel/i }).click()
  })
})

test.describe('Keyword Rules - Detail Page CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
  })

  test('should show all form fields on create page', async ({ page }) => {
    await page.goto('/chatbot/keywords/new')
    await page.waitForLoadState('networkidle')

    await expect(page.locator('input').first()).toBeVisible()
    const selects = page.locator('button[role="combobox"]')
    expect(await selects.count()).toBeGreaterThanOrEqual(2)
    await expect(page.locator('textarea').first()).toBeVisible()
    await expect(page.locator('button[role="switch"]').first()).toBeVisible()
  })

  test('should create keyword rule', async ({ page, request }) => {
    const id = await seedKeywordRuleViaAPI(request)
    expect(id, 'API seed must succeed').toBeTruthy()
    await page.goto(`/chatbot/keywords/${id}`)
    await page.waitForLoadState('networkidle')
    expect(page.url()).toMatch(/\/chatbot\/keywords\/[a-f0-9-]+/)
  })

  test('should edit existing rule', async ({ page, request }) => {
    // The previous version gated the save in `if (await saveBtn.isVisible())`,
    // so any permission or timing hiccup turned the test into a silent no-op.
    const id = await seedKeywordRuleViaAPI(request)
    expect(id, 'API seed must succeed').toBeTruthy()
    await page.goto(`/chatbot/keywords/${id}`)
    await page.waitForLoadState('networkidle')

    const input = page.locator('input').first()
    expect(await input.isDisabled()).toBe(false)

    const original = await input.inputValue()
    const edited = `${original}, e2e-edit`
    await input.fill(edited)
    await page.waitForTimeout(300)

    const saveBtn = page.getByRole('button', { name: /Save/i })
    await expect(saveBtn).toBeVisible({ timeout: 5000 })
    await saveBtn.click({ force: true })

    // Verify the edit persisted by reloading the page — this is the
    // assertion the previous no-op skipped.
    await page.reload()
    await page.waitForLoadState('networkidle')
    await expect(input).toHaveValue(edited)

    // Revert so the seeded row is left clean.
    await input.fill(original)
    await page.waitForTimeout(300)
    const revertBtn = page.getByRole('button', { name: /Save/i })
    await expect(revertBtn).toBeVisible({ timeout: 5000 })
    await revertBtn.click({ force: true })

    await verifyAuditLogged(request, 'keyword_rule', id!, 'updated', {
      expectedFields: ['name'],
    })
  })

  test('should delete from detail page', async ({ page, request }) => {
    const id = await seedKeywordRuleViaAPI(request)
    expect(id, 'API seed must succeed').toBeTruthy()
    await page.goto(`/chatbot/keywords/${id}`)
    await page.waitForLoadState('networkidle')
    await expectDeleteFromForm(page, '/chatbot/keywords')
  })

  test('should show metadata', async ({ page, request }) => {
    const id = await seedKeywordRuleViaAPI(request)
    expect(id, 'API seed must succeed').toBeTruthy()
    await page.goto(`/chatbot/keywords/${id}`)
    await page.waitForLoadState('networkidle')
    await expectMetadataVisible(page)
  })

  test('should show activity log', async ({ page, request }) => {
    const id = await seedKeywordRuleViaAPI(request)
    expect(id, 'API seed must succeed').toBeTruthy()
    await page.goto(`/chatbot/keywords/${id}`)
    await page.waitForLoadState('networkidle')
    await expectActivityLogVisible(page)
  })
})
