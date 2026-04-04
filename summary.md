## Session Summary

### Objective
- Remove the `Clear all` action from the notification bell and delete its related frontend logic.
- Make the status count in the `Statuses` header render as a true circular badge.
- Remove raw instance UUID display from `/settings/contacts`.

### Skills Used
- `vue-expert`
- `playwright-expert`

### Competencies Applied
- Vue 3 Composition API refactoring
- Tailwind utility-based UI adjustment
- Focused Playwright regression coverage
- Live browser verification with MCP browser tooling

### Changes Made
- Removed the `Clear all` button from [`frontend/src/components/NotificationBell.vue`](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend/src/components/NotificationBell.vue) along with the unused bulk-clear state and handler.
- Updated the status count badge in [`frontend/src/components/chat/status/StatusStoriesBar.vue`](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend/src/components/chat/status/StatusStoriesBar.vue) to stay circular while showing the real count instead of truncating to `9+`.
- Added a targeted regression check in [`frontend/e2e/tests/chat/statuses.spec.ts`](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend/e2e/tests/chat/statuses.spec.ts) for the removed `Clear all` action and circular badge dimensions.
- Removed raw instance UUID rendering from [`frontend/src/views/settings/ContactsView.vue`](/Users/noiemany/Downloads/whatomate_GOWA/whatomate/frontend/src/views/settings/ContactsView.vue) and changed the label fallback so unknown mappings no longer leak the UUID.

### Validation
- Passed: `npx eslint src/components/NotificationBell.vue src/components/chat/status/StatusStoriesBar.vue src/views/settings/ContactsView.vue e2e/tests/chat/statuses.spec.ts`
- Passed: `npx playwright test e2e/tests/chat/statuses.spec.ts`
- Passed via Chrome DevTools MCP with mocked API boot:
  - notification popover shows `Mark all as read` and does not contain `Clear all`
  - status badge measured `21 x 21` in the live DOM and showed `12` for a mocked 12-group state
  - contacts page showed `Primary Inbox` and did not contain `5cdb3701-8f23-4673-ab42-5492b226ab41`

### Notes
- `npm run typecheck` is currently failing on pre-existing repo issues outside this task scope.
