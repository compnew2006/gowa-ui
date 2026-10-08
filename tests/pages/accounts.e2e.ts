import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "WhatsApp accounts page exposes its primary features",
  fixture: "accounts-list",
  path: "/settings/accounts",
  heading: "WhatsApp Accounts",
  permission: "accounts",
  instructions:
    "Inspect the connection-status popover and GOWA Gateway card, then inspect the accounts table or its empty state. If rows exist, review sortable columns and open one account's edit page, then return. If no rows exist, verify Add WhatsApp Number links to GOWA Gateway. This page has no list filters or media-retention controls; those settings are on account details. Do not connect a device or delete anything.",
  regularUserInstructions:
    "Inspect the connection-status popover, GOWA Gateway card, and accounts table or empty state. If rows exist, review the read-only columns and open one account's details, then return. Verify this role has no add, link-device, or delete controls. This page has no list filters or media-retention controls. Do not change settings or connect a device.",
});
