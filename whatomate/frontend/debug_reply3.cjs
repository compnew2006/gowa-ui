const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

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
    
    await page.waitForTimeout(2000);
    const isLogin = await page.locator('input[type="email"]').count() > 0;
    
    if (isLogin) {
      console.log("On login page, logging in...");
      await page.fill('input[type="email"]', 'admin@test.com');
      await page.fill('input[type="password"]', 'Password123!');
      await Promise.all([
        page.waitForURL('**/chat**', { timeout: 15000 }),
        page.click('button[type="submit"]')
      ]);
    }

    console.log("Navigating to chat e1a0e916-a27d-44ea-8d60-50fa58350026...");
    await page.goto('http://localhost:8080/chat/e1a0e916-a27d-44ea-8d60-50fa58350026');

    await page.waitForSelector('.message-bubble', { timeout: 10000 });
    console.log("Messages loaded!");

    const messageBubbles = await page.locator('.message-bubble').all();
    if (messageBubbles.length === 0) {
      console.log("No messages found in chat.");
      process.exit(1);
    }
    
    const lastMessage = messageBubbles[messageBubbles.length - 1];
    await lastMessage.hover();

    const replyButton = lastMessage.locator('button:has(.lucide-reply), button[title*="Reply"], button[title*="reply"], button:has-text("Reply")');
    
    if (await replyButton.count() > 0) {
      console.log("Clicking reply button...");
      await replyButton.first().click({ force: true });
    } else {
      // Maybe we need to click the dropdown trigger first?
      console.log("Reply button not found directly, clicking dropdown trigger...");
      const dropdownTrigger = lastMessage.locator('button:has(.lucide-more-vertical)');
      if (await dropdownTrigger.count() > 0) {
         await dropdownTrigger.first().click();
         await page.waitForTimeout(500);
         const menuReply = page.locator('[role="menuitem"]:has-text("Reply")');
         if (await menuReply.count() > 0) {
             await menuReply.first().click();
             console.log("Clicked reply from menu!");
         } else {
             console.log("Could not find reply in menu.");
             process.exit(1);
         }
      } else {
         console.log("Dropdown trigger not found!");
         process.exit(1);
      }
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
