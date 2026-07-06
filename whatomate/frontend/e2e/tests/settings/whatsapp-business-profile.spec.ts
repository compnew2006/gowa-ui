import { test, expect } from '@playwright/test';
import { loginAsAdmin } from '../../helpers';
import { AccountsPage } from '../../pages';

test.describe('WhatsApp Business Profile', () => {
    let accountsPage: AccountsPage;

    test.beforeEach(async ({ page }) => {
        await loginAsAdmin(page);
        accountsPage = new AccountsPage(page);

        // Mock the GET /accounts to ensure we have a test subject
        await page.route('**/api/accounts', async route => {
            if (route.request().method() === 'GET') {
                await route.fulfill({
                    status: 200,
                    contentType: 'application/json',
                    body: JSON.stringify({
                        data: {
                            accounts: [{
                                id: 'test-acc-id',
                                name: 'Test Account',
                                phone_id: '123456',
                                business_id: '789012',
                                status: 'active'
                            }]
                        }
                    })
                });
            } else {
                await route.continue();
            }
        });

        // Mock GET profile
        await page.route('**/api/accounts/*/business_profile*', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    data: {
                        about: 'Available',
                        address: '123 Test St',
                        description: 'Test Business',
                        email: 'test@example.com',
                        vertical: 'PROF_SERVICES',
                        websites: ['https://example.com'],
                        profile_picture_url: ''
                    }
                })
            });
        });

        await accountsPage.goto();
    });

    test('should view business profile dialog', async () => {
        await accountsPage.expectPageVisible();
        await accountsPage.openBusinessProfile('Test Account');
        await accountsPage.expectProfileDialogVisible();

        // Verify fields contain mocked data
        await expect(accountsPage.profileDialog.locator('input#about')).toHaveValue('Available');
        await expect(accountsPage.profileDialog.locator('input#email')).toHaveValue('test@example.com');
    });

    test('should update business profile', async ({ page }) => {
        await accountsPage.openBusinessProfile('Test Account');

        // Mock PUT request
        await page.route('**/api/accounts/*/business_profile', async route => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    data: { success: true }
                })
            });
        });

        // Change value
        await accountsPage.profileDialog.locator('input#about').fill('Busy');
        await accountsPage.profileDialog.getByRole('button', { name: 'Save Changes' }).click();

        // Verify success toast
        await accountsPage.expectToast(/updated successfully/i);
    });
});
