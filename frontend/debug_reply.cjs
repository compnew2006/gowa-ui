const { chromium } = require('playwright');

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
    console.log("Navigating to login...");
    await page.goto('http://localhost:8080/login');
    await page.fill('input[type="email"]', 'admin@test.com');
    await page.fill('input[type="password"]', 'Password123!');
    await page.click('button[type="submit"]');

    await page.waitForURL('**/chat**');
    console.log("Logged in!");

    console.log("Navigating to chat e1a0e916-a27d-44ea-8d60-50fa58350026...");
    await page.goto('http://localhost:8080/chat/e1a0e916-a27d-44ea-8d60-50fa58350026');

    // Wait for messages to load
    await page.waitForTimeout(2000);

    // Find the first message bubble and hover it to reveal the reply button
    console.log("Looking for reply button...");
    const messageBubbles = await page.locator('.message-bubble').all();
    if (messageBubbles.length === 0) {
      console.log("No messages found in chat.");
      process.exit(1);
    }
    
    // Get the last message
    const lastMessage = messageBubbles[messageBubbles.length - 1];
    await lastMessage.hover();

    // Click the reply button (assuming it has an icon or title related to reply)
    // Looking for a button with lucide-reply or title="Reply"
    const replyButton = lastMessage.locator('button:has(.lucide-reply), button[title*="Reply"], button[title*="reply"]');
    if (await replyButton.count() > 0) {
      console.log("Clicking reply button...");
      await replyButton.first().click();
    } else {
      console.log("Reply button not found!");
      // dump html of message bubble
      console.log(await lastMessage.innerHTML());
      process.exit(1);
    }

    // Type a message
    console.log("Typing message...");
    const input = page.locator('textarea, input[placeholder*="Type"], [contenteditable="true"]');
    await input.first().fill('This is an automated reply test');

    // Click send
    console.log("Clicking send...");
    const sendButton = page.locator('button:has(.lucide-send), button[title*="Send"]');
    await sendButton.first().click();

    // Wait a bit to let request finish
    await page.waitForTimeout(2000);
    console.log("Done");
  } catch (err) {
    console.error("Error:", err);
  } finally {
    await browser.close();
  }
})();
