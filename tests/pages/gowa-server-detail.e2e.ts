import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "GOWA server detail error state page exposes its primary features",
  path: "/settings/gowa-servers/e2e-missing-server",
  heading: "GOWA Servers",
  permission: "gowa_instances",
  instructions:
    "Inspect the missing-server error state and its retry/back navigation. Do not create a device, request a QR code, pair a phone, or retry external connections.",
});
