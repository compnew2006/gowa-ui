import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "New account form exposes its primary features",
  path: "/settings/accounts/new",
  heading: "New Account",
  permission: "accounts",
  instructions:
    "Inspect the new-account form fields for account name, GOWA URL, device/JID, and incoming/outgoing/read-receipt routing. This is the creation route, so focus on the fields and helper text shown here. Do not save, connect, or delete an account.",
  regularUserInstructions:
    "Inspect the new-account form and verify its name, GOWA URL, device/JID, and incoming/outgoing/read-receipt routing controls are read-only or disabled for this role. Do not try to enable them or save.",
});

definePageExploration({
  name: "Saved WhatsApp account detail page exposes its primary features",
  path: "/settings/accounts",
  heading: "",
  fixture: "account-detail",
  permission: "accounts",
  instructions:
    "Inspect the saved account's routing and automation settings, metadata, and activity log. Review the device-status summary and GOWA Gateway link. Do not edit, save, delete, or connect any device.",
  regularUserInstructions:
    "Inspect the saved account detail and verify settings are read-only for this role. Do not edit, save, delete, or connect any device.",
});
