import { test, expect, type Page } from '@playwright/test'
import { ApiHelper, login, loginAsAdmin } from '../../helpers'
import { ChatPage } from '../../pages'
import { createTestScope } from '../../framework'

const cannedScope = createTestScope('chat-canned-preview')
// Seed via API as admin@admin.com (always exists per migrations) and log the
// browser in as the same user — otherwise the API-created contact lives in a
// different org from the UI session and the chat composer never renders.
const ADMIN_USER = { email: 'admin@admin.com', password: 'admin', role: 'admin' as const }

// Scope + helper for the low-level chat-layout specs below. Every contact
// created here gets a unique profile name so search/clear-search tests can
// assert against a known row.
const layoutScope = createTestScope('chat-layout')

/**
 * Resolve a contact row in the contacts sidebar by its rendered name.
 *
 * TODO(test-guard): encapsulate in ChatPage POM. ChatPage.contactList /
 * getContactItem() currently target `.contact-item, [data-testid="contact"]`
 * selectors that do not exist in ChatView.vue — the rows are plain
 * div.cursor-pointer. Until the POM is realigned, match the seeded name text
 * directly against the rendered row.
 */
function contactRowByName(page: Page, name: string) {
  return page.locator('div.cursor-pointer').filter({ hasText: name }).first()
}

test.describe('Chat page layout', () => {
  let chatPage: ChatPage
  let seededName: string

  test.beforeEach(async ({ page, request }) => {
    // Seed a contact via the API so the list is guaranteed non-empty and a
    // unique name is available for search assertions. Uses the same
    // admin@admin.com session the UI logs in as (see Canned Response block).
    const api = new ApiHelper(request)
    await api.login(ADMIN_USER.email, ADMIN_USER.password)
    seededName = layoutScope.name('Chat Test')
    await api.createContact(layoutScope.phone(), seededName)

    await loginAsAdmin(page)
    chatPage = new ChatPage(page)
    await chatPage.goto()
  })

  test('chat view shows the contact list on first load', async () => {
    // The search input is part of the contacts sidebar — its presence proves
    // the chat layout (not a bare body) rendered.
    await expect(chatPage.searchInput).toBeVisible()
  })

  test('contact list area is visible beside the chat area', async ({ page }) => {
    await expect(chatPage.searchInput).toBeVisible()
    // The seeded contact's name should appear in the sidebar list.
    // TODO(test-guard): encapsulate in ChatPage POM
    await expect(contactRowByName(page, seededName)).toBeVisible({ timeout: 10_000 })
  })

  test('contacts sidebar renders the search input', async () => {
    await expect(chatPage.searchInput).toBeVisible()
    await expect(chatPage.searchInput).toHaveAttribute('placeholder', /Search/i)
  })

  test('searching by a contact name filters the list to that row', async ({ page }) => {
    // TODO(test-guard): encapsulate in ChatPage POM
    const seededRow = contactRowByName(page, seededName)
    await expect(seededRow).toBeVisible({ timeout: 10_000 })

    await chatPage.searchContacts(seededName)
    // The matched row stays visible after server-side filtering.
    await expect(seededRow).toBeVisible()
  })

  test('clearing the search restores the full contact list', async ({ page }) => {
    // TODO(test-guard): encapsulate in ChatPage POM
    const seededRow = contactRowByName(page, seededName)
    await expect(seededRow).toBeVisible({ timeout: 10_000 })

    await chatPage.searchContacts('zzzz-no-such-contact')
    // A no-match search hides the seeded row.
    await expect(seededRow).toHaveCount(0)

    // Clearing brings the full list back.
    await chatPage.searchContacts('')
    await expect(seededRow).toBeVisible({ timeout: 10_000 })
  })

  test('contact list shows at least one seeded contact', async ({ page }) => {
    // TODO(test-guard): encapsulate in ChatPage POM
    const seededRow = contactRowByName(page, seededName)
    await expect(seededRow).toBeVisible({ timeout: 10_000 })
  })
})

test.describe('Chat composer', () => {
  let chatPage: ChatPage
  let contactId: string

  test.beforeEach(async ({ page, request }) => {
    const api = new ApiHelper(request)
    await api.login(ADMIN_USER.email, ADMIN_USER.password)
    const contact = await api.createContact(layoutScope.phone(), layoutScope.name('Composer'))
    contactId = contact.id

    await loginAsAdmin(page)
    chatPage = new ChatPage(page)
    // Navigate straight to the seeded contact so the composer mounts without
    // depending on the (stale) POM contact-row locators.
    await chatPage.goto(contactId)
  })

  test('chat composer toolbar renders all action buttons when a contact is selected', async () => {
    // Collapsed from the former per-button "should have X button" no-op tests
    // (Rule 3): one body asserts the whole toolbar renders. The send button
    // only enables once there is text; visibility is what we assert here.
    await expect(chatPage.messageInput).toBeVisible({ timeout: 10_000 })
    await expect(chatPage.sendButton).toBeVisible()
    await expect(chatPage.attachButton).toBeVisible()
    await expect(chatPage.emojiButton).toBeVisible()
    await expect(chatPage.cannedResponsesButton).toBeVisible()
    await expect(chatPage.templatePickerButton).toBeVisible()
  })

  test('typing into the message input updates its value', async () => {
    await expect(chatPage.messageInput).toBeVisible({ timeout: 10_000 })
    const text = `Hello ${Date.now()}`
    await chatPage.messageInput.fill(text)
    await expect(chatPage.messageInput).toHaveValue(text)
  })

  test('clearing the message input empties its value', async () => {
    await expect(chatPage.messageInput).toBeVisible({ timeout: 10_000 })
    await chatPage.messageInput.fill('Temporary text')
    await chatPage.messageInput.fill('')
    await expect(chatPage.messageInput).toHaveValue('')
  })
})

test.describe('Chat messages display', () => {
  let chatPage: ChatPage
  let contactId: string

  test.beforeEach(async ({ page, request }) => {
    const api = new ApiHelper(request)
    await api.login(ADMIN_USER.email, ADMIN_USER.password)
    const contact = await api.createContact(layoutScope.phone(), layoutScope.name('Messages'))
    contactId = contact.id

    await loginAsAdmin(page)
    chatPage = new ChatPage(page)
    await chatPage.goto(contactId)
  })

  // A fresh contact has no messages — the messages area only renders bubbles
  // once a row exists. Seeding an inbound message requires SQL (there is no
  // createMessage API factory; see message-bubbles.spec.ts for the pattern).
  // Marking fixme until a SQL seed is wired in here, rather than asserting a
  // tautology on a possibly-empty list.
  test.fixme('messages area renders bubbles for a conversation with history', async () => {
    // TODO(test-guard): seed a message via SQL — see message-bubbles.spec.ts pattern
    await expect(chatPage.messageList).toBeVisible()
  })

  // removed: tautological assertion ("messageCount >= 0"); fully covered by
  // message-bubbles.spec.ts which seeds messages via SQL and asserts real
  // bubble rendering.
})

test.describe('Canned Response Preview', () => {
  let api: ApiHelper
  let chatPage: ChatPage
  let contact: { id: string; profile_name: string }
  let plain: { id: string; name: string; shortcut: string }
  let withParam: { id: string; name: string; shortcut: string }
  let withContactName: { id: string; name: string; shortcut: string }

  test.beforeEach(async ({ page, request }) => {
    api = new ApiHelper(request)
    await api.login('admin@admin.com', 'admin')

    // Names use the random-suffix form (no second arg) so each test creates
    // distinct rows. The DELETE endpoint soft-deletes, so a deterministic
    // name would collide with the prior test's tombstone within the worker.
    contact = await api.createContact(cannedScope.phone(), cannedScope.name())
    const plainName = cannedScope.name()
    plain = await api.createCannedResponse({
      name: plainName,
      shortcut: plainName.toLowerCase(),
      content: 'Hello! Thanks for contacting us.',
    })
    const orderName = cannedScope.name()
    withParam = await api.createCannedResponse({
      name: orderName,
      shortcut: orderName.toLowerCase(),
      content: 'Your order #{{order_id}} is ready.',
    })
    const greetName = cannedScope.name()
    withContactName = await api.createCannedResponse({
      name: greetName,
      shortcut: greetName.toLowerCase(),
      content: 'Hi {{contact_name}}, how can I help?',
    })

    await login(page, ADMIN_USER)
    chatPage = new ChatPage(page)
    await chatPage.goto(contact.id)
  })

  test.afterEach(async () => {
    for (const id of [plain?.id, withParam?.id, withContactName?.id]) {
      if (id) await api.deleteCannedResponse(id).catch(() => {})
    }
  })

  test('opens preview dialog with no inputs when content has no placeholders', async () => {
    await chatPage.openCannedResponses()
    await chatPage.selectCannedPickerItem(plain.name)

    await chatPage.cannedDialog.waitFor({ state: 'visible' })
    await expect(chatPage.cannedDialog).toContainText(plain.name)
    await expect(chatPage.cannedDialogPreview).toContainText('Hello! Thanks for contacting us.')
    await expect(chatPage.cannedDialogParamInputs).toHaveCount(0)
  })

  test('renders an input for each {{custom}} token and updates preview live', async () => {
    await chatPage.openCannedResponses()
    await chatPage.selectCannedPickerItem(withParam.name)

    await chatPage.cannedDialog.waitFor({ state: 'visible' })
    await expect(chatPage.cannedDialogParamInputs).toHaveCount(1)
    // Until the agent fills the input the preview keeps the literal token.
    await expect(chatPage.cannedDialogPreview).toContainText('{{order_id}}')

    await chatPage.fillCannedParam('order_id', '12345')
    await expect(chatPage.cannedDialogPreview).toContainText('Your order #12345 is ready.')
    await expect(chatPage.cannedDialogPreview).not.toContainText('{{order_id}}')
  })

  test('auto-resolves {{contact_name}} without exposing an input', async () => {
    await chatPage.openCannedResponses()
    await chatPage.selectCannedPickerItem(withContactName.name)

    await chatPage.cannedDialog.waitFor({ state: 'visible' })
    await expect(chatPage.cannedDialogParamInputs).toHaveCount(0)
    await expect(chatPage.cannedDialogPreview).toContainText(`Hi ${contact.profile_name}, how can I help?`)
  })

  test('Send dispatches the resolved body to the messages API', async ({ page }) => {
    let postedBody: { type?: string; content?: { body?: string } } | null = null
    // Intercept the POST so we can assert the resolved body without depending
    // on a real WhatsApp account being configured in the test backend.
    await page.route(`**/api/contacts/${contact.id}/messages`, async (route) => {
      postedBody = JSON.parse(route.request().postData() || '{}')
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'error', message: 'mocked' }),
      })
    })

    await chatPage.openCannedResponses()
    await chatPage.selectCannedPickerItem(withParam.name)
    await chatPage.cannedDialog.waitFor({ state: 'visible' })
    await chatPage.fillCannedParam('order_id', '99')
    await chatPage.sendCannedDialog()

    await expect.poll(() => postedBody, { timeout: 5000 }).not.toBeNull()
    expect(postedBody!.type).toBe('text')
    expect(postedBody!.content?.body).toBe('Your order #99 is ready.')
  })

  test('Cancel closes the dialog and clears any leftover slash command', async () => {
    // Open the picker via slash command, then cancel — the textarea should
    // be empty afterwards (the slash command is dropped on selection).
    await chatPage.messageInput.fill(`/${plain.shortcut}`)
    await chatPage.selectCannedPickerItem(plain.name)
    await chatPage.cannedDialog.waitFor({ state: 'visible' })

    await chatPage.cancelCannedDialog()
    await expect(chatPage.messageInput).toHaveValue('')
  })
})
