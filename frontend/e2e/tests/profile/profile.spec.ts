import { test, expect } from '@playwright/test'
import { loginAsAdmin } from '../../helpers'
import { ProfilePage } from '../../pages'

test.describe('Profile Page', () => {
  let profilePage: ProfilePage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    profilePage = new ProfilePage(page)
    await profilePage.goto()
  })

  test('should display profile page', async () => {
    await profilePage.expectPageVisible()
  })

  test('should show account information card', async () => {
    await expect(profilePage.accountInfoCard).toBeVisible()
    await expect(profilePage.accountInfoCard).toContainText('Account Information')
  })

  test('should show change password card', async () => {
    await expect(profilePage.changePasswordCard).toBeVisible()
    await expect(profilePage.changePasswordCard).toContainText('Change Password')
  })

  // Rule 3: collapsed three near-duplicate label tests
  // (display user name / email / role) into one.
  test('should display user name, email, and role labels', async () => {
    await expect(profilePage.accountInfoCard).toContainText('Name')
    await expect(profilePage.accountInfoCard).toContainText('Email')
    await expect(profilePage.accountInfoCard).toContainText('Role')
  })
})

test.describe('Password Change Form', () => {
  let profilePage: ProfilePage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    profilePage = new ProfilePage(page)
    await profilePage.goto()
  })

  // removed: form-field-render tests (Rule 7); validation tests cover this surface.
  // (was: have current/new/confirm password field, have change password button)

  test('should show validation error for mismatched passwords', async () => {
    await profilePage.fillPasswordForm('oldpassword', 'newpassword1', 'newpassword2')
    await profilePage.changePasswordButton.click()
    await profilePage.expectToast(/match/i)
  })

  test('should show validation error for short password', async () => {
    await profilePage.fillPasswordForm('oldpassword', '12345', '12345')
    await profilePage.changePasswordButton.click()
    await profilePage.expectToast(/6 characters/i)
  })

  // Rule 3: collapsed three near-duplicate visibility-toggle tests into
  // one data-driven test over the three fields.
  test('should toggle password field visibility for each field', async () => {
    const fields: Array<{
      name: 'current' | 'new' | 'confirm'
      locator: () => import('@playwright/test').Locator
    }> = [
      { name: 'current', locator: () => profilePage.currentPasswordInput },
      { name: 'new', locator: () => profilePage.newPasswordInput },
      { name: 'confirm', locator: () => profilePage.confirmPasswordInput },
    ]

    for (const field of fields) {
      const input = field.locator()
      await expect(input).toHaveAttribute('type', 'password')
      await profilePage.togglePasswordVisibility(field.name)
      await expect(input).toHaveAttribute('type', 'text')
    }
  })

  // Rule 4 + Rule 5: name claimed "clear form after successful password
  // change" but the body never submitted — a misleading no-op. Renamed
  // to reflect what it actually verifies: the form retains entered values.
  test('should retain entered values in the password form', async () => {
    await profilePage.fillPasswordForm('test', 'newpass123', 'newpass123')
    await expect(profilePage.currentPasswordInput).toHaveValue('test')
    await expect(profilePage.newPasswordInput).toHaveValue('newpass123')
    await expect(profilePage.confirmPasswordInput).toHaveValue('newpass123')
  })
})

test.describe('Profile Page Labels', () => {
  let profilePage: ProfilePage

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
    profilePage = new ProfilePage(page)
    await profilePage.goto()
  })

  // removed: form-field-render tests (Rule 7); validation tests cover this surface.
  // (was: Current/New/Confirm New Password label tests)

  test('should show password requirement hint', async () => {
    await expect(profilePage.changePasswordCard.getByText(/6 characters/i)).toBeVisible()
  })
})
