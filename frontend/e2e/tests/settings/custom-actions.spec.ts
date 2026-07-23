import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { CustomActionsPage } from '../../pages'
import { createTestScope } from '../../framework'

const scope = createTestScope('custom-actions')

test.describe('Custom Actions Management', () => {
  let customActionsPage: CustomActionsPage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    customActionsPage = new CustomActionsPage(page)
    await customActionsPage.goto()
  })

  // TODO(test-guard): custom_actions.go handler does NOT call a.logAudit —
  // known audit gap that mirrors contacts.go (see ARCHITECTURE.md "known
  // gap" section). The CRUD tests below therefore do NOT yet assert
  // verifyAuditLogged, because the backend never writes the audit row and a
  // hard assert would regress these green tests. Once the handler emits
  // audit rows, add to beforeEach: `new ApiHelper(request).login(...)`, then
  // after each mutation resolve the id via GET /api/custom-actions?search=<name>
  // (envelope: data.custom_actions[].id) and assert:
  //   verifyAuditLogged(request, 'custom_action', id, 'created'|'updated'|'deleted')
  // (resource_type 'custom_action' singular, per the canned_response convention).

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

  test('should create a webhook custom action', async () => {
    const actionName = scope.name('webhook')

    await customActionsPage.openCreateDialog()
    await customActionsPage.fillWebhookAction(actionName, 'https://api.example.com/webhook')
    await customActionsPage.submitDialog()

    await customActionsPage.expectToast('created')
    await customActionsPage.expectRowExists(actionName)
    // TODO(test-guard): add verifyAuditLogged(request, 'custom_action', id, 'created') once handler emits audit rows.
  })

  test('should create a URL custom action', async () => {
    const actionName = scope.name('url')

    await customActionsPage.openCreateDialog()
    await customActionsPage.fillUrlAction(actionName, 'https://crm.example.com/contact')
    await customActionsPage.submitDialog()

    await customActionsPage.expectToast('created')
    await customActionsPage.expectRowExists(actionName)
    // TODO(test-guard): add verifyAuditLogged(request, 'custom_action', id, 'created') once handler emits audit rows.
  })

  test('should create a JavaScript custom action', async () => {
    const actionName = scope.name('js')

    await customActionsPage.openCreateDialog()
    await customActionsPage.fillJsAction(actionName, 'return { clipboard: contact.phone_number }')
    await customActionsPage.submitDialog()

    await customActionsPage.expectToast('created')
    await customActionsPage.expectRowExists(actionName)
    // TODO(test-guard): add verifyAuditLogged(request, 'custom_action', id, 'created') once handler emits audit rows.
  })

  test('should edit existing custom action', async () => {
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
    // TODO(test-guard): add verifyAuditLogged(request, 'custom_action', id, 'updated') once handler emits audit rows.
  })

  test('should delete custom action', async () => {
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

    // Delete the action
    await customActionsPage.deleteRow(actionName)
    await customActionsPage.confirmDelete()

    await customActionsPage.expectToast('deleted')
    // TODO(test-guard): resolve id via GET /api/custom-actions?search=<name> before delete, then add
    // verifyAuditLogged(request, 'custom_action', id, 'deleted') once handler emits audit rows.
  })

  test('should toggle custom action status', async () => {
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
    // TODO(test-guard): add verifyAuditLogged(request, 'custom_action', id, 'updated') once handler emits audit rows.
  })

  test('should cancel custom action creation', async () => {
    await customActionsPage.openCreateDialog()
    await customActionsPage.getDialogInput('name').fill('Cancelled Action')
    await customActionsPage.cancelDialog()
    await customActionsPage.expectDialogHidden()
  })
})
