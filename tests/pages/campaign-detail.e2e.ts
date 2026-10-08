import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Saved campaign detail page exposes its primary features",
  path: "/campaigns",
  heading: "",
  fixture: "campaign-detail",
  permission: "campaigns",
  instructions:
    "Read the saved draft campaign summary, template, recipient count, and recipient status. Do not click any action buttons or row controls. In particular, do not use Start, Delete, Add Recipients, or the recipient-row action. Do not edit, delete, add recipients, start, pause, cancel, or send the campaign.",
});
