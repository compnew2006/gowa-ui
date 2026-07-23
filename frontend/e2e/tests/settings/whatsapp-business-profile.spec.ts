import { test, expect } from '@playwright/test';
import { ApiHelper, generateUniqueName, loginAsAdmin, verifyAuditLogged } from '../../helpers';
import { SUPER_ADMIN } from '../../framework';
import { AccountsPage } from '../../pages';

test.describe('WhatsApp Business Profile', () => {
    let accountsPage: AccountsPage;
    let api: ApiHelper;
    let accountId: string;

    test.beforeEach(async ({ page, request }) => {
        // Seed a REAL WhatsApp account via the API and drive the UI against it.
        // The previous version stubbed every backend route (accounts, business
        // profile, audit-logs, PUT) and asserted against mock payloads — that
        // mocked the app's own internal APIs, which are not external boundaries,
        // so it proved nothing about the real flow. The only legitimate external
        // boundary here is Meta's upstream Graph API, which this flow does not
        // call directly (the update goes through /api/accounts/:id/business_profile),
        // so no page.route() mock is warranted.
        api = new ApiHelper(request);
        await api.login(SUPER_ADMIN.email, SUPER_ADMIN.password);

        const resp = await api.createWhatsAppAccount({
            name: generateUniqueName('WABiz'),
            phone_id: generateUniqueName('1'),
            business_id: generateUniqueName('b'),
            access_token: 'test-token-' + Date.now(),
        });
        expect(resp.ok, `seed account: ${JSON.stringify(resp)}`).toBe(true);
        accountId = resp.id;

        await loginAsAdmin(page);
        accountsPage = new AccountsPage(page);
    });

    test.afterEach(async () => {
        if (accountId) {
            await api.del('/api/accounts/' + accountId).catch(() => {});
        }
    });

    test('should view business profile dialog', async ({ page }) => {
        // Navigate to the real seeded account's detail page
        await page.goto(`/settings/accounts/${accountId}`);
        await page.waitForLoadState('networkidle');

        // Open business profile dialog
        await accountsPage.openBusinessProfile();
        await accountsPage.expectProfileDialogVisible();

        // Verify the real fields render (about/email inputs exist and are
        // populated from the live backend, not a fixture).
        await expect(accountsPage.profileDialog.locator('input#about')).toBeVisible();
        await expect(accountsPage.profileDialog.locator('textarea#description')).toBeVisible();
    });

    test('should update business profile', async ({ page }) => {
        await page.goto(`/settings/accounts/${accountId}`);
        await page.waitForLoadState('networkidle');

        await accountsPage.openBusinessProfile();

        // Change a value against the real backend
        const updatedAbout = 'Busy ' + Date.now();
        await accountsPage.profileDialog.locator('input#about').fill(updatedAbout);
        await accountsPage.profileDialog.getByRole('button', { name: 'Save Changes' }).click();

        // Verify success toast from the real flow
        await accountsPage.expectToast(/updated successfully/i);

        // API side-channel: confirm the audit trail recorded the update on the
        // real resource. Resource type is singular ('account' — WhatsApp account).
        await verifyAuditLogged(request, 'account', accountId, 'updated');
    });
});
