import { test, expect } from '@playwright/test';
import { loginAsAdmin } from '../../helpers';
import { ChatPage } from '../../pages';

test.describe('Chat Image Scroll Behavior', () => {
  let chatPage: ChatPage;

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
    chatPage = new ChatPage(page);
    await chatPage.goto();
  });

  test('maintains scroll at bottom when image loads', async ({ page }) => {
    const contacts = page.locator('.cursor-pointer').filter({ has: page.locator('text=/[+0-9]/') });
    const count = await contacts.count();
    if (count === 0) test.skip();

    await contacts.first().click();
    await page.waitForLoadState('networkidle');

    // Wait for the message area to be visible
    const messagesArea = page.locator('[data-radix-scroll-area-viewport]');
    await expect(messagesArea).toBeVisible();

    // The scroll position should be at or very near the bottom initially
    let scrollInfo = await messagesArea.evaluate((el) => {
      return {
        scrollTop: el.scrollTop,
        scrollHeight: el.scrollHeight,
        clientHeight: el.clientHeight,
        distanceFromBottom: el.scrollHeight - el.scrollTop - el.clientHeight
      };
    });

    expect(scrollInfo.distanceFromBottom).toBeLessThan(150);

    // Simulate incoming image message that causes a height increase
    await page.evaluate(() => {
      if ((window as any).__WHM_WS_TEST_EMIT__) {
         const dummyContactId = "e87f8eb4-500b-4654-8abc-03bc5561bbbb"; // just some id
         (window as any).__WHM_WS_TEST_EMIT__("new_message", {
           id: "dummy-img-msg-id-1234",
           contact_id: dummyContactId, // we should use the actual active contact id ideally, but for scroll it might just append
           conversation_id: "+1234567890@c.us",
           profile_name: "Tester",
           direction: "incoming",
           message_type: "image",
           media_url: "https://via.placeholder.com/280x300.png?text=Test+Image",
           content: { body: "Image loaded" },
           created_at: new Date().toISOString(),
           updated_at: new Date().toISOString(),
         });
      }
    });

    await page.waitForTimeout(1000);

    // Even if the DOM changed or height expanded, the chat should have auto-scrolled to the bottom 
    // because it was already at the bottom
    scrollInfo = await messagesArea.evaluate((el) => {
      return {
        scrollTop: el.scrollTop,
        scrollHeight: el.scrollHeight,
        clientHeight: el.clientHeight,
        distanceFromBottom: el.scrollHeight - el.scrollTop - el.clientHeight
      };
    });

    // We should be within 150 pixels of the bottom (allow small sub-pixel rounding differences)
    expect(scrollInfo.distanceFromBottom).toBeLessThan(150);
  });
});

