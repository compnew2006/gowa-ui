# P0-7 Fix: Duplicate Send Race Condition

## Task
Fix the P0-7 Duplicate Send Race vulnerability from FIX_PLAN_AR.md — campaign workers sleeping locally while holding unacknowledged Redis Stream messages caused duplicate message sends via XClaim re-delivery.

## Root Cause Analysis

### The Race
1. Worker reads job from Redis Stream via `XReadGroup`
2. `HandleRecipientJob` acquires a 2-minute Redis lock, validates status=Pending
3. `applyCampaignSendDelay` calls `time.Sleep()` for potentially 5-15 minutes
4. During sleep, the message is **NOT ACK'd** in Redis Stream
5. After `ClaimMinIdleTime` (5 minutes), the PEL self-heal cycle marks the message as "stale"
6. A second worker claims it via `XClaim`
7. The 2-minute lock has expired by now, so the second worker acquires it
8. Second worker finds status still Pending (first worker is asleep, hasn't sent yet)
9. **Both workers wake up and send the same message** → duplicate delivery

### Why It Matters
- User annoyance from duplicate messages
- Double billing for API calls
- WhatsApp account ban risk from automated duplicate behavior

## Approach & Key Decisions

### Phase 1: Acknowledge Immediately, Schedule Later
Instead of sleeping while holding the unACK'd message, the fix splits the send into two phases:
- **Phase A (HandleRecipientJob)**: Validate → compute delay → schedule in ZSET → return (message gets ACK'd immediately)
- **Phase B (executeRecipientSend)**: Picked up by poller when `sendAt <= now`

### Phase 2: Redis Sorted Set Scheduling
Jobs with delays are stored in `whatomate:scheduled_sends` ZSET with `score = sendAt.UnixMilli()`. A background poller (`runScheduledSendsPoller`) checks every second for ready jobs.

### Phase 3: Atomic DB Status Transition
Added `MessageStatusSending` as intermediate state. The transition `pending → sending` uses `WHERE status = 'pending'` with `RowsAffected` check — if 0 rows affected, another worker already grabbed it.

### Phase 4: Conditional Status Updates
`updateRecipientStatusConditional` only updates if status matches expected values (`pending` or `sending`), preventing a late worker from overwriting a completed status.

### Phase 5: Extended Lock TTL
Recipient lock TTL increased from 2 minutes to 30 minutes to cover the maximum expected delay window.

### Phase 6: Code Review Fixes (P1-1 through P2)

#### P1-1: checkCampaignCompletion counts 'sending' as unfinished
Changed `checkCampaignCompletion` to query `status IN ('pending', 'sending')` instead of only `status = 'pending'`. Without this fix, a campaign where all remaining recipients were in `sending` state would be prematurely marked as completed.

#### P1-2: Group sends routed through group-specific path
`executeRecipientSend` now branches on `RecipientType == "group"` and delegates to `executeGroupRecipientSend`, which handles group JID validation, group membership verification, group-specific template resolution, and creates Message records without ContactID. Previously, delayed group sends went through the contact path and would fail.

#### P1-3: Crash recovery hardening
Three fixes:
- **Order swap**: ZSET scheduling now happens *before* the `pending → sending` status transition. If crash occurs between ZAdd and status change, the recipient stays `pending` and the ZSET entry will be processed normally (executeRecipientSend accepts both `pending` and `sending`).
- **ZRem after execution**: `pollScheduledSends` only removes the ZSET entry *after* `executeRecipientSend` returns nil (success or terminal state). If execution returns an error or the worker crashes, the ZSET entry remains and the next poll cycle retries. Idempotency (status check + claim lock) prevents duplicate sends on retry.
- **Recovery loop**: `runSendingRecoveryLoop` runs every 5 minutes, finding recipients stuck in `sending` for over 30 minutes. Before resetting to `pending`, it checks the scheduled-send lock (`whatomate:scheduled_send_lock:<id>`) — if the lock exists, the recipient is actively being processed and is skipped.

#### P2: Batch size limit on ZRangeByScore
Added `Count: scheduledSendsBatchSize` (50) to the `ZRangeByScore` query in `pollScheduledSends` to prevent unbounded bursts when many jobs become ready simultaneously.

## Files Modified/Created

| File | Action | Description |
|------|--------|-------------|
| `internal/worker/scheduled_sends.go` | **Created** | ZSET scheduling system: `scheduledSend` struct, `scheduleRecipientSend`, `pollScheduledSends` (ZRem after execution), `claimScheduledSend`, `isScheduledSendLockHeld`, `runSendingRecoveryLoop`, `recoverStuckSendingRecipients` |
| `internal/worker/worker.go` | Modified | Split `HandleRecipientJob` into validate+schedule (ZAdd before status transition); added `executeRecipientSend` (with group routing), `executeGroupRecipientSend`, `computeCampaignDelayDuration`, `transitionRecipientToSending`, `updateRecipientStatusConditional`; fixed `checkCampaignCompletion` to count `sending`; updated `Run` to start poller and recovery loop |
| `internal/worker/worker_group.go` | Modified | Group sends use same scheduling pattern (ZAdd before status transition) |
| `internal/worker/idempotency.go` | Modified | Extended `recipientLockTTL` from 2min to 30min |
| `internal/models/constants.go` | Modified | Added `MessageStatusSending MessageStatus = "sending"` |
| `internal/worker/scheduled_sends_test.go` | **Created** | Regression tests for ZSET poller, claim lock, lock-held guard, corrupt entry cleanup, full E2E with DB |

## Dependencies
No new external dependencies. Uses existing `github.com/redis/go-redis/v9` ZSET operations and Lua scripting.

## Tests
- **New**: `internal/worker/scheduled_sends_test.go` — regression tests for the ZSET poller
  - `TestClaimScheduledSend_AcquiresLock` — lock acquired on first call, denied on second
  - `TestClaimScheduledSend_NilRedis` — nil Redis falls back to `true`
  - `TestPollScheduledSends_DoesNotRemoveIfLockHeldByAnother` — ZSET entry survives when another worker holds the lock
  - `TestPollScheduledSends_RetriesOnDBError` — ZSET entry survives when recipient not found in DB (skips without TEST_DATABASE_URL)
  - `TestPollScheduledSends_RemovesCorruptEntry` — malformed JSON is removed from ZSET
  - `TestPollScheduledSends_FullE2EWithDB` — full poller → provider send → recipient status → ZSET cleanup (requires TEST_DATABASE_URL)
  - `TestIsScheduledSendLockHeld` — detects held vs absent locks
- All existing tests pass: `TestCampaignDelayRedisKey_*`, `TestHandleRecipientJob*` (PASS/SKIP as expected - DB tests skip without TEST_DATABASE_URL)
- `go vet ./internal/...` — clean
- `go build ./cmd/... ./internal/... ./pkg/...` — clean

## Verification
```bash
go build ./cmd/... ./internal/... ./pkg/...
go vet ./internal/worker/...
go test -v ./internal/worker/...
```

## Backward Compatibility
- If Redis is nil (no Redis configured), falls back to original `sleepWithContext` behavior
- If scheduling fails, falls back to immediate send
- `MessageStatusSending` is a new state; any code checking `status == "pending"` that should also accept `"sending"` may need updating (the new `updateRecipientStatusConditional` already handles this)

## Known Limitations
- No migration needed — `MessageStatusSending` is just a string value in the existing status column
- ZSET does not persist across Redis restarts unless Redis has AOF/RDB persistence enabled
- The 1-second poll interval means sends may be delayed up to 1 second beyond their scheduled time
- `computeCampaignDelayDuration` replaces `applyCampaignSendDelay` for the scheduling path but `applyCampaignSendDelay` is kept for backward compatibility with direct (non-scheduled) sends
- Recovery loop resets stuck `sending` recipients to `pending` only if no active scheduled-send lock exists — recipients with active locks are skipped to avoid interrupting in-progress sends
- If executeRecipientSend returns a non-nil error (DB failure, context cancel), the ZSET entry is left for retry — the next poll cycle will attempt again, and idempotency prevents duplicates
