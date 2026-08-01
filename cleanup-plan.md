# Meta Removal — Remaining Cleanup (updated)

## ✅ Already Fixed (no action needed)
- `cmd/gowa-ui/main.go` — GOWA-only, no Calling/TTS/Meta imports or routes
- `internal/handlers/organization.go` — calling + Meta settings removed
- `internal/handlers/messages.go` — voice_call/flow/MessageTypeFlow removed
- `internal/worker/worker.go` — local template rendering, no Meta API calls
- `internal/handlers/app.go` — WhatsApp/CallManager/TTS fields removed
- All 18 test files fixed (registry, accounts_validation, gowa_webhook, etc.)
- `frontend/src/views/settings/FlowsView.vue` — deleted
- `frontend/src/components/flow-builder/` — deleted
- `frontend/src/views/settings/CatalogsView.vue` — deleted
- `frontend/src/views/settings/CatalogDetailView.vue` — deleted
- `catalogsService`/`productsService` — removed from api.ts
- `Catalog`/`CatalogProduct` interfaces — removed from api.ts
- `organizationService.updateSettings` — calling/Meta fields removed
- Router: `/settings/flows`, `/settings/catalogs` routes — removed

---

## Remaining

### 1. Empty directories (trivial)
```
rm -rf internal/calling/ internal/tts/
```

### 2. `canned_responses.go` — dead voice_call/flow code
| Lines | What |
|-------|------|
| 22-33 | `CannedResponseButton` struct — remove `FlowID string`, `TTLMinutes int`, `Screen string` fields |
| 388-463 | Remove `"voice_call"` and `"flow"` cases from `buttonsToAuditString` and `validateCannedResponseButtons` |

### 3. `test/testutil/db.go` — stale TRUNCATE lists
| Lines | What |
|-------|------|
| 134-135 | Remove `"catalog_products"`, `"catalogs"` from `cleanupTables()` |
| 186-187 | Remove `"catalog_products"`, `"catalogs"` from `TruncateTables()` |

### 4. `internal/handlers/gowa_device_security_test.go`
| Line | What |
|------|------|
| 164 | Remove `"provider_type": "gowa"` from account creation map |

---

### 5. Frontend files to delete
```bash
rm frontend/src/types/flow-preview.ts
rm frontend/src/components/chatbot/flow-preview/PreviewButtonGroup.vue
# If flow-preview directory is now empty:
rmdir frontend/src/components/chatbot/flow-preview/
```

### 6. `frontend/src/services/api.ts`

| Item | What |
|------|------|
| Lines 904-908 | Remove `goto_flow`, `whatsapp_flow` from `ChatNodeType` union |
| Lines 395-401 | Remove flow methods from `chatbotService` (listFlows, getFlow, createFlow, updateFlow, deleteFlow, duplicateFlow) — these are Meta Flow methods, not chatbot flows |

### 7. `frontend/src/components/shared/MessageButtonsEditor.vue`

| Line | What |
|------|------|
| 10 | Remove `{ flowsService }` import from `@/services/api` (no longer exported) |
| Type union | Remove `'voice_call' \| 'flow'` from allowed types |
| ~177-197 | Remove voice_call and flow add-button UI blocks |
| ~248-281 | Remove TTL input and flow selector UI blocks |

### 8. `frontend/src/views/chat/ChatView.vue`

| Line | What |
|------|------|
| 108 | Remove `import PreviewButtonGroup from '@/components/chatbot/flow-preview/PreviewButtonGroup.vue'` |
| ~1197-1234 | Remove voice_call/flow filter and send logic |
| ~1887-1913 | Remove `getVoiceCallData()` and `getFlowButtonText()` |
| ~3028-3045 | Remove the template rendering sections for those |

### 9. `frontend/src/views/settings/CannedResponseDetailView.vue`

| Line | What |
|------|------|
| 15-16 | Remove `import PreviewButtonGroup` and `import type { ButtonConfig }` |
| ~124-136 | Remove voice_call validation |
| ~166-182 | Remove flow validation |
| 379 | Change `allowedTypes` to `['reply', 'url']` |

### 10. `frontend/src/views/settings/SettingsView.vue`

| Line | What |
|------|------|
| 41-44 | Remove `meta_app_id`, `meta_config_id`, `meta_app_secret`, `has_meta_app_secret` from reactive state |
| 79-82 | Remove initialization from API response |

### 11. `frontend/src/views/settings/AccountsView.vue`

| Line | What |
|------|------|
| 37 | Remove `provider_type` from `WhatsAppAccount` interface |

### 12. `frontend/src/views/settings/AccountDetailView.vue`

| Line | What |
|------|------|
| 62 | Remove `provider_type` from interface |
| 68 | Remove `webhook_verify_token` from interface |
| 103, 147, 175 | Remove `provider_type: 'gowa'` assignments |

### 13. i18n — `en.json` and `ar.json`

Remove orphaned key groups:
- `flows.*` (~22+ keys at line 1429)
- `catalogs.*` (~30+ keys at line 2230)
- `products.*` (at line 2260)
- `settings.calling.*` / `callingSettings` etc. (lines 644-653)
- `accounts.*` Meta keys (`metaAppId`, `metaAppIdPlaceholder`, `phoneNumberIdHint`, `businessAccountId`, etc.)
- `sidebar.catalogs`, `resources.flow/Flow/catalog/catalogs/Catalog/Catalogs/product/products`

Keep: `sso.facebook`, `sso.facebookDesc`, `chatbot.flows.*` (these are chatbot conversation flows, NOT Meta Flows)

---

## Verification

```bash
rm -rf internal/calling/ internal/tts/
go build ./...
go vet ./...
go test -p 1 ./...
cd frontend && npm run typecheck && npm run lint
```
