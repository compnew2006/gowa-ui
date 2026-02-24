# PLAN.md

## Immediate Next Steps
1. Investigate how multiple accounts for a single contact are currently handled in `ChatView.vue`.
2. Add a `unified_contacts_view` setting (likely in localStorage or user profile).
3. Update `filteredContacts` in `contactsStore` to group by `phone_number` if the setting is enabled.
4. Update `ChatView.vue` to display the account toggle when a unified contact is selected.
5. Create E2E tests for the new features.
