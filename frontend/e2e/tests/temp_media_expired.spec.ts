import { test, expect } from '@playwright/test';

const BASE = 'https://ofuqalmadenah.com';

test('Login and check media expired messages', async ({ page }) => {
  test.setTimeout(60000);
  
  // Login
  await page.goto(`${BASE}/login`);
  await page.waitForLoadState('networkidle');
  
  await page.locator('#email, input[type="email"]').first().fill('admin@whatomate.local');
  await page.locator('#password, input[type="password"]').first().fill('f46EyrhpqSq/apkqu2DmjFOIgS/6/b7i');
  await page.locator('button[type="submit"]').first().click();
  
  await page.waitForURL('**/chat/**', { timeout: 15000 });
  await page.waitForTimeout(2000);
  console.log('Logged in OK');

  // Test 1: Chat 3d755a2e - document
  console.log('\n--- Chat 3d755a2e (Document) ---');
  await page.goto(`${BASE}/chat/3d755a2e-ca6b-418a-95a7-321d1c763a7c`);
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(6000);
  await page.screenshot({ path: '/tmp/test_chat1.png', fullPage: false });
  
  const expired1 = await page.locator(':text("File no longer available")').count();
  const doc1 = await page.locator(':text("[Document]")').count();
  console.log(`  "File no longer available": ${expired1}`);
  console.log(`  "[Document]": ${doc1}`);

  // Test 2: Chat 1f46de7a - images  
  console.log('\n--- Chat 1f46de7a (Images) ---');
  await page.goto(`${BASE}/chat/1f46de7a-abbb-437b-9eab-6c4407ca2350`);
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(6000);
  await page.screenshot({ path: '/tmp/test_chat2.png', fullPage: false });
  
  const expired2 = await page.locator(':text("File no longer available")').count();
  const img2 = await page.locator(':text("[Image]")').count();
  console.log(`  "File no longer available": ${expired2}`);
  console.log(`  "[Image]": ${img2}`);
});
