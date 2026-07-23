# Quickstart: Verifying the RBAC Gap Fixes

**Feature**: `002-rbac-gaps-gowa`
**Date**: 2026-07-12

This guide verifies each fix locally against a running whatomate instance with PostgreSQL + Redis (per constitution Principle 15: integration tests with real services).

---

## Prerequisites

```bash
# From the whatomate repo root
make test            # Run existing Go integration tests (should pass before changes)
go build ./cmd/whatomate   # Ensure it compiles
```

You need a running PostgreSQL + Redis. The test suite uses `test/testutil/` which spins these up via Docker.

---

## Verification 1: GOWA webhook fail-close (Story 1, FR-001/002)

```bash
# Start the server with a GOWA instance configured
./whatomate server &

# Test 1a: Missing signature header → 403
curl -s -o /dev/null -w "%{http_code}" \
  -X POST http://localhost:8080/api/gowa/webhook \
  -H "Content-Type: application/json" \
  -d '{"event":"message","device_id":"test@s.whatsapp.net","timestamp":'$(date +%s)',"message":{"id":"x","from":"1","type":"text","text":"hi"}}'
# Expected: 403 (was: 200 — the vulnerability)

# Test 1b: Tampered body → 403
SECRET="test-secret"
BODY='{"event":"message","device_id":"test@s.whatsapp.net","timestamp":'$(date +%s)'}'
SIG=$(echo -n "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')
curl -s -o /dev/null -w "%{http_code}" \
  -X POST http://localhost:8080/api/gowa/webhook \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=$SIG" \
  -d '{"event":"message","device_id":"test@s.whatsapp.net","timestamp":'$(date +%s)',"message":{"id":"TAMPERED","from":"1","type":"text","text":"forged"}}'
# Expected: 403 (body doesn't match signature)

# Test 1c: Stale timestamp (> 5 min) → silently dropped (200, no writes)
OLD_TS=$(($(date +%s) - 400))  # 6 minutes ago
BODY='{"event":"connection","device_id":"test@s.whatsapp.net","timestamp":'$OLD_TS'}'
SIG=$(echo -n "$BODY" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')
curl -s -o /dev/null -w "%{http_code}" \
  -X POST http://localhost:8080/api/gowa/webhook \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=$SIG" \
  -d "$BODY"
# Expected: 200 (silently dropped, no DB writes — check logs for "Stale GOWA webhook rejected")
```

---

## Verification 2: Device handler permission gating (Story 2, FR-006/007/008)

```bash
# Log in as an agent (lowest privilege)
AGENT_COOKIE=$(curl -s -c - -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"agent@test.com","password":"pass"}' | grep whm_access | awk '{print $NF}')

# Test 2a: Agent cannot list GOWA instances → 403
curl -s -o /dev/null -w "%{http_code}" \
  -b "whm_access=$AGENT_COOKIE" \
  http://localhost:8080/api/gowa/instances
# Expected: 403

# Test 2b: Agent cannot create a device → 403
curl -s -o /dev/null -w "%{http_code}" \
  -b "whm_access=$AGENT_COOKIE" \
  -X POST http://localhost:8080/api/gowa/create-device \
  -H "Content-Type: application/json" \
  -d '{"base_url":"http://gowa:8080","device_name":"test"}'
# Expected: 403

# Test 2c: Manager CAN list instances → 200
MANAGER_COOKIE=$(curl -s -c - -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"manager@test.com","password":"pass"}' | grep whm_access | awk '{print $NF}')
curl -s -o /dev/null -w "%{http_code}" \
  -b "whm_access=$MANAGER_COOKIE" \
  http://localhost:8080/api/gowa/instances
# Expected: 200
```

---

## Verification 3: Cross-org device provisioning refusal (Story 2, FR-009)

```bash
# As manager of org A, attempt to provision on org B's instance
curl -s -w "\n%{http_code}" \
  -b "whm_access=$MANAGER_COOKIE" \
  -X POST http://localhost:8080/api/gowa/create-device \
  -H "Content-Type: application/json" \
  -d '{"base_url":"http://org-b-gowa:8080","device_name":"cross-org"}'
# Expected: 400 "Unknown GOWA instance for your organization"
```

---

## Verification 4: Media export permission tiering (Story 5, FR-013)

```bash
# Agent (lacks contacts:export) cannot download ZIP → 403
curl -s -o /dev/null -w "%{http_code}" \
  -b "whm_access=$AGENT_COOKIE" \
  "http://localhost:8080/api/media/zip?ids=msg-uuid-1,msg-uuid-2"
# Expected: 403 (was: 200 — the gap)

# Manager (has contacts:export) can download ZIP → 200
curl -s -o /dev/null -w "%{http_code}" \
  -b "whm_access=$MANAGER_COOKIE" \
  "http://localhost:8080/api/media/zip?ids=msg-uuid-1,msg-uuid-2"
# Expected: 200 (application/zip)
```

---

## Verification 5: Re-download cooldown (Story 5, FR-014)

```bash
# First re-download → 200
curl -s -o /dev/null -w "%{http_code}" \
  -b "whm_access=$MANAGER_COOKIE" \
  -X POST http://localhost:8080/api/media/msg-uuid-1/redownload
# Expected: 200

# Immediate second re-download → 429 (cooldown)
curl -s -o /dev/null -w "%{http_code}" \
  -b "whm_access=$MANAGER_COOKIE" \
  -X POST http://localhost:8080/api/media/msg-uuid-1/redownload
# Expected: 429

# After 60 seconds → 200 again
sleep 61
curl -s -o /dev/null -w "%{http_code}" \
  -b "whm_access=$MANAGER_COOKIE" \
  -X POST http://localhost:8080/api/media/msg-uuid-1/redownload
# Expected: 200
```

---

## Verification 6: Automated tests (Story 6, FR-016)

```bash
# Run the new security-path tests
go test -race -run "TestGowaDevice_.*Permission\|TestGowaWebhook_.*Signature\|TestGowaWebhook_.*Replay\|TestMediaZip_.*Export\|TestMediaRedownload_.*Cooldown" ./internal/handlers/...

# Run the full suite to check for regressions
make test
```

**Expected test cases** (new):
- `TestGowaDevice_AgentDenied` — agent gets 403 on all 5 device endpoints
- `TestGowaDevice_CrossOrgProvisioning` — manager from org A cannot provision on org B instance
- `TestGowaWebhook_MissingSignature` — 403, no writes
- `TestGowaWebhook_TamperedBody` — 403, no writes
- `TestGowaWebhook_EmptySecret` — 403, no writes
- `TestGowaWebhook_StaleTimestamp` — 200, no writes (replay dropped)
- `TestGowaWebhook_CrossOrgMessageMutation` — revoked/edit/reaction on another org's message → ignored
- `TestMediaZip_ExportPermission` — agent without `contacts:export` gets 403
- `TestMediaRedownload_Cooldown` — second request within 60s gets 429
- `TestGowaWebhookSecret_AutoGenerated` — account created without secret has one after creation

---

## Verification 7: Permission catalog (Story 3, FR-010/011)

```bash
# As admin, list permissions — verify devices:read and devices:write exist
curl -s -b "whm_access=$ADMIN_COOKIE" http://localhost:8080/api/permissions | jq '.data[] | select(.resource == "devices")'
# Expected:
# {"resource":"devices","action":"read","description":"View GOWA device status and instances"}
# {"resource":"devices","action":"write","description":"Pair and provision GOWA devices"}

# As admin, check manager role has devices permissions
curl -s -b "whm_access=$ADMIN_COOKIE" http://localhost:8080/api/roles | jq '.data[] | select(.name=="manager") | .permissions[] | select(.resource=="devices")'
# Expected: both read and write present

# Verify agent role does NOT have devices permissions
curl -s -b "whm_access=$ADMIN_COOKIE" http://localhost:8080/api/roles | jq '.data[] | select(.name=="agent") | .permissions[] | select(.resource=="devices")'
# Expected: empty (agent has no devices permissions)
```
