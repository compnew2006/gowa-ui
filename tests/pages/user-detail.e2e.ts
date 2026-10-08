import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "User detail error state page exposes its primary features",
  path: "/settings/users/e2e-missing-user",
  heading: "User not found",
  permission: "users",
  instructions:
    "Inspect the missing-user state and recovery link. Do not edit roles, account access, passwords, or user membership.",
});

definePageExploration({
  name: "Existing user detail page exposes its primary features",
  path: "/settings/users",
  heading: "",
  fixture: "user-detail",
  permission: "users",
  instructions:
    "Inspect the existing E2E user's profile, role, account assignments, metadata, and available controls. Do not change roles, passwords, account access, or user membership.",
});
