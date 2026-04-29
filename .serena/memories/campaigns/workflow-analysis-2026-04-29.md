## Campaigns Workflow Analysis (2026-04-29)

### Files Created
- `docs/campaigns.md` — Full campaign workflow documentation (routes, models, lifecycle, queue, worker, auto-campaign, RBAC, testing)
- `docs/gap.md` — 12 gaps identified with proposed solutions and impact analysis

### Key Findings
1. **GAP-01 (HIGH):** ScheduledAt field exists but no scheduler consumes it — dead feature
2. **GAP-02 (HIGH):** Whatsmeow delivery receipts do NOT update campaign stats (delivered_count, read_count) — Meta webhook does, whatsmeow handleReceipt does not
3. **GAP-03 (MEDIUM):** No batch size limit on recipient import — unbounded JSON arrays
4. **GAP-04 (MEDIUM):** No rate limiting on campaign endpoints
5. **GAP-05 (LOW):** Campaign completes on send, not delivery — no "fully delivered" state
6. **GAP-06 (HIGH):** HandleRecipientJob (core send logic) has zero tests
7. **GAP-07 (MEDIUM):** pauseActiveCampaignsForInstance has zero tests
8. **GAP-08 (MEDIUM):** Auto-campaign worker (525 lines) has only 2 tests
9. **GAP-09 (LOW):** MIME type from client header, no magic byte detection
10. **GAP-10 (LOW):** Cancelled campaigns leave orphaned Redis Stream messages
11. **GAP-11 (MEDIUM):** No end-to-end integration test for full campaign lifecycle
12. **GAP-12 (MEDIUM):** CampaignsView.vue is 3,535 lines — needs component extraction

### Architecture Notes
- 15 HTTP routes under /api/campaigns/ + 1 auto-campaign media route
- Redis Streams queue with tenant-scoped consumer groups
- Worker autoscaling per organization (WorkerScaler)
- Real-time stats via Redis pub/sub → WebSocket broadcast
- Dual provider support: Meta Cloud API + Whatsmeow protocol
- Auto-campaign worker for whatsmeow instances (periodic campaign generation)
- 4-layer tenant isolation: middleware + DB scope + handler + related resources
- RBAC: campaigns:read/write/delete/execute