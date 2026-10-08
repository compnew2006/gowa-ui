import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "SSO settings page exposes its primary features",
  path: "/settings/sso",
  heading: "Single Sign-On (SSO)",
  permission: "settings.sso",
  instructions:
    "Inspect configured provider controls and safe explanatory states. Do not reveal client secrets, enable/disable providers, or save authentication settings.",
});
