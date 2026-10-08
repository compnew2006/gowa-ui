import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "GOWA servers page exposes its primary features",
  path: "/settings/gowa-servers",
  heading: "GOWA Servers",
  fixture: "gowa-servers",
  permission: "gowa_instances",
  instructions:
    "Inspect the server list columns and empty state, then open the add-server form. Review name, base URL, username, password, webhook URL, and Active controls; cancel without submitting. This page has no search or status filter. Do not probe or contact an external server.",
});
