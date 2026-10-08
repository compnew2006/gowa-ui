import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Saved contact detail page exposes its primary features",
  path: "/settings/contacts",
  heading: "",
  fixture: "contact-detail",
  permission: "contacts.manage",
  instructions:
    "Inspect the saved contact profile, metadata, conversation state, and available return navigation. Do not change contact data or open a customer conversation.",
});
