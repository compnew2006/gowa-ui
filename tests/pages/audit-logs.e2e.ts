import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Audit logs page exposes its primary features",
  path: "/settings/audit-logs",
  heading: "Audit Logs",
  fixture: "audit-logs-list",
  permission: "audit_logs",
  instructions:
    "Inspect user/action/resource/date filters and the log list or empty state. Open a read-only detail link if present; do not change records.",
});
