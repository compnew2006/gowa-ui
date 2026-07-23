import { test, expect } from '@playwright/test'
import { loginAsAdmin, ApiHelper, verifyAuditLogged } from '../../helpers'
import { CustomActionsPage } from '../../pages'
import { createTestScope, SUPER_ADMIN } from '../../framework'

const scope = createTestScope('custom-actions')

test.describe('Custom Actions Management', () => {
  let customActionsPage: CustomActionsPage

  test.beforeEach(async ({ page, request }) => {
    await loginAsAdmin(page)
    // Authenticate the API request fixture so verifyAuditLogged can hit
    // /api/audit-logs once the handler emits audit rows (see TODO below).
    await new ApiHelper(request).login(SUPER_ADMIN.email, SUPER_ADMIN.password)
    customActionsPage = new CustomActionsPage(page)
    await customActionsPage.goto()
  })

  // TODO(test-guard): custom_actions.go handler does NOT call a.logAudit —
  // known audit gap that mirrors contacts.go (see ARCHITECTURE.md "known
  // gap" section). The CRUD tests below therefore do NOT yet assert
  // verifyAuditLogged, because the backend never writes the audit row and a
  // hard assert would regress these green tests. Once the handler emits
  // audit rows, uncomment the verifyAuditLogged(...) calls marked below
  // (resource_type is 'custom_action' singular, per the canned_response
  // convention). For UI-driven mutations, resolve the created id via a
  // GET /api/custom-actions?search=<name> follow-up before verifying.

  test('should display custom actions page', async () => {
    await customActionsPage.expectPageVisible()
    await expect(customActionsPage.addButton).toBeVisible()
  })

  test('should open create custom action dialog', async () => {
    await customActionsPage.openCreateDialog()
    await customActionsPage.expectDialogVisible()
    await expect(customActionsPage.dialog).toContainText('Custom Action')
  })

  test('should show validation error for empty name', async () => {
    await customActionsPage.openCreateDialog()
    await customActionsPage.submitDialog()
    await customActionsPage.expectToast('required')
  })

  test('should create a webhook custom action', async ({ request }) => {
    const actionName = scope.name('webhook')

    await customActionsPage.openCreateDialog()
    await customActionsPage.fillWebhookAction(actionName, 'https://api.example.com/webhook')
    await customActionsPage.submitDialog()

    await customActionsPage.expectToast('created')
    await customActionsPage.expectRowExists(actionName)
    // TODO(test-guard): once custom_actions.go calls logAudit, resolve the
    // created id via GET /api/custom-actions?search=<actionName> and add:
    //   await verifyAuditLogged(request, 'custom_action', id, 'created')
  })

  test('should create a URL custom action', async ({ request }) => {
    const actionName = scope.name('url')

    await customActionsPage.openCreateDialog()
    await customActionsPage.fillUrlAction(actionName, 'https://crm.example.com/contact')
    await customActionsPage.submitDialog()

    await customActionsPage.expectToast('created')
    await customActionsPage.expectRowExists(actionName)
    // TODO(test-guard): once custom_actions.go calls logAudit, resolve the
    // created id via GET /api/custom-actions?search=<actionName> and add:
    //   await verifyAuditLogged(request, 'custom_action', id, 'created')
  })

  test('should create a JavaScript custom action', async ({ request }) => {
    const actionName = scope.name('js')

    await customActionsPage.openCreateDialog()
    await customActionsPage.fillJsAction(actionName, 'return { clipboard: contact.phone_number }')
    await customActionsPage.submitDialog()

    await customActionsPage.expectToast('created')
    await customActionsPage.expectRowExists(actionName)
    // TODO(test-guard): once custom_actions.go calls logAudit, resolve the
    // created id via GET /api/custom-actions?search=<actionName> and add:
    //   await verifyAuditLogged(request, 'custom_action', id, 'created')
  })

  test('should edit existing custom action', async ({ request }) => {
    // First create an action
    const actionName = scope.name('edit')

    await customActionsPage.openCreateDialog()
    await customActionsPage.fillUrlAction(actionName, 'https://example.com')
    await customActionsPage.submitDialog()

    // Wait for create toast and dismiss it
    await customActionsPage.expectToast('created')
    await customActionsPage.dismissToast('created')

    // Wait for action to appear
    await customActionsPage.expectRowExists(actionName)

    // Edit the action
    await customActionsPage.editRow(actionName)
    await customActionsPage.getDialogInput('url').fill('https://updated.example.com')
    await customActionsPage.submitDialog('Update')

    await customActionsPage.expectToast('updated')
    // TODO(test-guard): once custom_actions.go calls logAudit, resolve the
    // id via GET /api/custom-actions?search=<actionName> and add:
    //   await verifyAuditLogged(request, 'custom_action', id, 'updated')
  })

  test('should delete custom action', async ({ request }) => {
    // First create an action
    const actionName = scope.name('delete')

    await customActionsPage.openCreateDialog()
    await customActionsPage.fillUrlAction(actionName, 'https://todelete.com')
    await customActionsPage.submitDialog()

    // Wait for create toast and dismiss it
    await customActionsPage.expectToast('created')
    await customActionsPage.dismissToast('created')

    // Wait for action to appear
    await customActionsPage.expectRowExists(actionName)

    // Resolve the id before deletion so an audit check can reference it.
    const listResp = await new ApiHelper(request).get(`/api/custom-actions?search=${encodeURIComponent(actionName)}`)
    expect(listResp.ok(), await listResp.text()).toBe(true)
    const id = (await listResp.json()).data?.custom_actions?.[0]?.id

    // Delete the action
    await customActionsPage.deleteRow(actionName)
    await customActionsPage.confirmDelete()

    await customActionsPage.expectToast('deleted')
    // TODO(test-guard): once custom_actions.go calls logAudit, add:
    //   await verifyAuditLogged(request, 'custom_action', id, 'deleted')
  })

  test('should toggle custom action status', async ({ request }) => {
    // First create an action
    const actionName = scope.name('toggle')

    await customActionsPage.openCreateDialog()
    await customActionsPage.fillUrlAction(actionName, 'https://toggle.com')
    await customActionsPage.submitDialog()

    // Wait for toast
    await customActionsPage.expectToast('created')

    // Wait for action to appear
    await customActionsPage.expectRowExists(actionName)

    // Toggle the switch (disabling triggers a confirmation dialog)
    await customActionsPage.toggleRowSwitch(actionName)

    // Wait for the confirm dialog and click confirm
    await customActionsPage.alertDialog.waitFor({ state: 'visible' })
    await customActionsPage.alertDialog.getByRole('button', { name: 'Confirm' }).click()

    await customActionsPage.expectToast(/(enabled|disabled)/i)
    // TODO(test-guard): once custom_actions.go calls logAudit, resolve the
    // id via GET /api/custom-actions?search=<actionName> and add:
    //   await verifyAuditLogged(request, 'custom_action', id, 'updated')
  })

  test('should cancel custom action creation', async () => {
    await customActionsPage.openCreateDialog()
    await customActionsPage.getDialogInput('name').fill('Cancelled Action')
    await customActionsPage.cancelDialog()
    await customActionsPage.expectDialogHidden()
  })
})
