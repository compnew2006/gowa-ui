/* eslint-disable */
import { chromium } from 'playwright';

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  // Log all network requests
  page.on('request', request => {
    if (request.url().includes('/messages')) {
      console.log('>>', request.method(), request.url(), request.postData());
    }
  });

  page.on('response', async response => {
    if (response.url().includes('/messages') && response.request().method() === 'POST') {
      console.log('<<', response.status(), await response.text());
    }
  });

  try {
    console.log("Navigating to app...");
    await page.goto('http://localhost:8080/');
    
    // Check if we are on login page
    await page.waitForTimeout(2000);
    const isLogin = await page.locator('input[type="email"]').count() > 0;
    
    if (isLogin) {
      console.log("On login page, logging in...");
      await page.fill('input[type="email"]', 'admin@test.com');
      await page.fill('input[type="password"]', 'Password123!');
      await Promise.all([
        page.waitForNavigation({ waitUntil: 'networkidle' }),
        page.click('button[type="submit"]')
      ]);
    }

    console.log("Navigating to chat e1a0e916-a27d-44ea-8d60-50fa58350026...");
    await page.goto('http://localhost:8080/chat/e1a0e916-a27d-44ea-8d60-50fa58350026');

    // Wait for messages to load (wait for message bubbles)
    await page.waitForSelector('.message-bubble', { timeout: 10000 });
    console.log("Messages loaded!");

    const messageBubbles = await page.locator('.message-bubble').all();
    if (messageBubbles.length === 0) {
      console.log("No messages found in chat.");
      process.exit(1);
    }
    
    const lastMessage = messageBubbles[messageBubbles.length - 1];
    await lastMessage.hover();

    const replyButton = lastMessage.locator('button:has(.lucide-reply), button[title*="Reply"], button[title*="reply"]');
    if (await replyButton.count() > 0) {
      console.log("Clicking reply button...");
      await replyButton.first().click();
    } else {
      console.log("Reply button not found!");
      process.exit(1);
    }

    console.log("Typing message...");
    const input = page.locator('textarea, input[placeholder*="Type"], [contenteditable="true"]');
    await input.first().fill('This is an automated reply test from debug script');

    console.log("Clicking send...");
    const sendButton = page.locator('button:has(.lucide-send), button[title*="Send"], button[aria-label="Send message"]');
    await sendButton.first().click();

    await page.waitForTimeout(3000);
    console.log("Done");
  } catch (err) {
    console.error("Error:", err);
  } finally {
    await browser.close();
  }
})();
