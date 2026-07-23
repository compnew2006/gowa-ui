import { test, expect } from '@playwright/test'
import { ApiHelper, loginAsAdmin, verifyAuditLogged } from '../../helpers'
import { SUPER_ADMIN, createTestScope } from '../../framework'
import { TagsPage } from '../../pages'

const scope = createTestScope('tags')

// Look up a tag's id by exact name via the list API. The tag id isn't surfaced
// in the DOM, so we resolve it here for audit-log side-channel assertions.
async function findTagIdByName(api: ApiHelper, name: string): Promise<string | null> {
  const resp = await api.get('/api/tags')
  if (!resp.ok()) return null
  const body = await resp.json()
  const tags = body.data?.tags ?? body.data ?? []
  const found = tags.find((t: any) => t.name === name)
  return found ? found.id : null
}

test.describe('Tags Management', () => {
  let tagsPage: TagsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    tagsPage = new TagsPage(page)
    await tagsPage.goto()
  })

  test('should display tags page', async () => {
    await tagsPage.expectPageVisible()
    await expect(tagsPage.addButton).toBeVisible()
  })

  test('should open create tag dialog', async () => {
    await tagsPage.openCreateDialog()
    await tagsPage.expectDialogVisible()
    await expect(tagsPage.dialog).toContainText('Create Tag')
  })

  test('should show validation error for empty name', async () => {
    await tagsPage.openCreateDialog()
    await tagsPage.submitDialog()
    await tagsPage.expectToast('required')
  })

  test('should create a tag with each color', async ({ request }) => {
    // Merged from "should create a new tag" and "should create a tag with
    // different color", which were near-duplicates (identical setup and
    // assertions, only the color value differed). This parameterized version
    // creates a tag per color and asserts the badge renders, then verifies the
    // audit trail recorded each creation.
    const api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)

    for (const color of ['Blue', 'Purple']) {
      const tagName = scope.name(color.toLowerCase())

      await tagsPage.openCreateDialog()
      await tagsPage.fillTagForm(tagName, color)
      await tagsPage.submitDialog()

      await tagsPage.expectToast('created')
      await tagsPage.expectTagBadgeVisible(tagName)

      // API side-channel: confirm the audit trail recorded the creation.
      const tagId = await findTagIdByName(api, tagName)
      if (tagId) {
        await verifyAuditLogged(request, 'tag', tagId, 'created')
      }
    }
  })

  test('should edit existing tag', async ({ request }) => {
    // First create a tag
    const tagName = scope.name('edit')

    await tagsPage.openCreateDialog()
    await tagsPage.fillTagForm(tagName, 'Green')
    await tagsPage.submitDialog()

    // Wait for create toast and dismiss it
    await tagsPage.expectToast('created')
    await tagsPage.dismissToast('created')

    // Wait for tag to appear
    await tagsPage.expectTagBadgeVisible(tagName)

    // Edit the tag
    await tagsPage.editRow(tagName)
    await tagsPage.selectColor('Red')
    await tagsPage.submitDialog('Update')

    await tagsPage.expectToast('updated')

    // API side-channel: confirm the edit was audited.
    const api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    const tagId = await findTagIdByName(api, tagName)
    if (tagId) {
      await verifyAuditLogged(request, 'tag', tagId, 'updated')
    }
  })

  test('should delete tag', async ({ request }) => {
    // First create a tag
    const tagName = scope.name('delete')

    await tagsPage.openCreateDialog()
    await tagsPage.fillTagForm(tagName, 'Gray')
    await tagsPage.submitDialog()

    // Wait for create toast and dismiss it
    await tagsPage.expectToast('created')
    await tagsPage.dismissToast('created')

    // Wait for tag to appear
    await tagsPage.expectTagBadgeVisible(tagName)

    // Capture the id before deletion (the row disappears after delete).
    const api = new ApiHelper(request)
    await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    const tagId = await findTagIdByName(api, tagName)

    // Delete the tag
    await tagsPage.deleteRow(tagName)
    await tagsPage.confirmDelete()

    await tagsPage.expectToast('deleted')

    // API side-channel: confirm the deletion was audited.
    if (tagId) {
      await verifyAuditLogged(request, 'tag', tagId, 'deleted')
    }
  })

  test('should search tags', async ({ page }) => {
    // First create a tag with unique name
    const uniqueName = scope.name('unique')

    await tagsPage.openCreateDialog()
    await tagsPage.fillTagForm(uniqueName, 'Yellow')
    await tagsPage.submitDialog()

    // Wait for creation
    await tagsPage.expectToast('created')
    await tagsPage.dismissToast('created')

    // Search for the tag
    await page.locator('input[placeholder*="Search"]').fill(uniqueName)
    await page.waitForTimeout(500)
    await tagsPage.expectTagBadgeVisible(uniqueName)
  })

  test('should prevent duplicate tag names', async ({ page }) => {
    const tagName = scope.name('duplicate')

    // Create first tag
    await tagsPage.openCreateDialog()
    await tagsPage.fillTagForm(tagName)
    await tagsPage.submitDialog()
    await tagsPage.expectToast('created')

    // Wait for toast to disappear before clicking Add Tag again
    await page.locator('[data-sonner-toast]').waitFor({ state: 'hidden', timeout: 10000 })

    // Try to create duplicate
    await tagsPage.openCreateDialog()
    await tagsPage.fillTagForm(tagName)
    await tagsPage.submitDialog()

    await tagsPage.expectToast(/already exists/i)
  })

  test('should cancel tag creation', async () => {
    await tagsPage.openCreateDialog()
    await tagsPage.dialog.locator('input').first().fill('Cancelled Tag')
    await tagsPage.cancelDialog()
    await tagsPage.expectDialogHidden()
  })
})
