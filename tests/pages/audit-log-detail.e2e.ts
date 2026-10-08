import { definePageExploration } from "../helpers/page-exploration";

definePageExploration({
  name: "Saved audit log detail page exposes its primary features",
  path: "/settings/audit-logs",
  heading: "",
  fixture: "audit-log-detail",
  permission: "audit_logs",
  instructions:
    "Inspect the audit event, formatted change values, event metadata, and account resource link. Verify the link is present without opening it. This is read-only; do not change any record.",
});
