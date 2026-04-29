Settings route (/settings and /settings/*) analysis completed 2026-04-29.

18 sub-routes documented in docs/settings.md.
15 gaps found and documented in docs/gap.md (SETTINGS-GAP-01 through SETTINGS-GAP-15).

Key findings:
- 3 HIGH severity gaps: shared isSubmitting state, no loading indicator, inconsistent permission guards
- 10+ settings views have no component-level permission checks (route guards exist but action buttons visible to all)
- PendingChatsView and AssignedChatsView have hardcoded 200-item limit with no pagination
- Backend organization member management routes (GET current, list members, update/remove member) have no frontend API service methods
- SSOSettingsView makes inline API calls instead of using a service module
- SettingsView.vue is 1000+ lines and should be extracted into composables
- Mixed storage strategy: some settings in backend API, some in localStorage, some in configStore
