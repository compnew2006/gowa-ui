import { test, expect } from '@playwright/test';

test.describe('Wiki Documentation', () => {
    const wikiUrl = 'http://127.0.0.1:8005';

    test('should load the English homepage and show the logo', async ({ page }) => {
        await page.goto(wikiUrl);
        
        // Wait for the page to load
        await expect(page).toHaveTitle(/Whatomate Documentation/);
        
        // Check for the logo (it should be in the header)
        // MkDocs Material logo class is usually .md-logo or img in .md-header__button
        const logo = page.locator('.md-logo img');
        await expect(logo).toBeVisible();
        
        // Take a screenshot
        await page.screenshot({ path: 'test-results/screenshots/wiki-en.png' });
        
        // Verify a link in the User Guide
        const authLink = page.getByRole('link', { name: 'Authentication & Login' });
        await expect(authLink).toBeVisible();
    });

    test('should switch to Arabic and display correct RTL layout', async ({ page }) => {
        await page.goto(wikiUrl);
        
        // Switch to Arabic
        // MkDocs Material language switcher is usually in the header or footer
        // Using the switcher in the header (extra.alternate in mkdocs.yml)
        // Usually it's in a menu or a link with lang="ar"
        const langSwitcher = page.locator('a[lang="ar"]');
        await langSwitcher.click();
        
        // Wait for navigation
        await page.waitForURL(/.*\/ar\/.*/);
        
        // Check for Arabic title
        await expect(page).toHaveTitle(/توثيقات واتومات/);
        
        // Verify RTL direction
        const html = page.locator('html');
        await expect(html).toHaveAttribute('dir', 'rtl');
        
        // Take a screenshot
        await page.screenshot({ path: 'test-results/screenshots/wiki-ar.png' });
        
        // Verify an Arabic link
        const authLinkAr = page.getByRole('link', { name: 'المصادقة وتسجيل الدخول' });
        await expect(authLinkAr).toBeVisible();
    });
});
