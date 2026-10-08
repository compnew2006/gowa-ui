import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Agent analytics page exposes its primary features",
  path: "/analytics/agents",
  heading: "Agent Analytics",
  permission: "analytics.agents",
  instructions:
    "Inspect the agent filter and date-range presets/custom range controls, then review the visible performance metrics and empty/error states. Do not apply a persistent change.",
});
