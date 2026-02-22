# Research: Whatsmeow Integration

**Feature**: `001-whatsmeow-integration` | **Date**: 2026-02-17

## R-001: whatsmeow Library Integration Pattern

**Decision**: Use `go.mau.fi/whatsmeow` as the WhatsApp Web protocol client, wrapped behind a new adapter at `pkg/whatsmeow/`.

**Rationale**: whatsmeow is the most actively maintained Go implementation of the WhatsApp Web multi-device API. It supports QR code pairing, persistent sessions via SQL stores, and the full message protocol (text, media, reactions, read receipts, group messaging). It's used in production by several large bridge projects (e.g., mautrix-whatsapp).

**Alternatives considered**:
- `rhymen/go-whatsapp`: Deprecated, single-device only.
- Custom implementation: Prohibitive reverse-engineering effort.
- Meta Cloud API only: Remains as legacy option but has cost, friction, and approval requirements.

## R-002: Session Persistence Strategy

**Decision**: Use whatsmeow's built-in PostgreSQL container store (`sqlstore`) for device session persistence.

**Rationale**: whatsmeow provides `sqlstore.NewWithDB()` which accepts an existing `*sql.DB` connection. Since Whatomate already uses PostgreSQL (via GORM), we can share the same database. The store creates its own tables (`whatsmeow_device`, `whatsmeow_identity_keys`, etc.) and manages session data independently. This avoids introducing new external dependencies (Constitution Principle VI: Single-Binary Simplicity).

**Alternatives considered**:
- File-based store: Not suitable for containerized/multi-replica deployments.
- Separate database: Unnecessary operational complexity.
- Custom GORM-based store: whatsmeow's native store is battle-tested; reimplementing is risky.

## R-003: Adapter Interface Design

**Decision**: Define a new `MessageProvider` Go interface in `pkg/provider/` that both the existing Meta client (`pkg/whatsapp/`) and the new whatsmeow adapter (`pkg/whatsmeow/`) can implement.

**Rationale**: The existing `pkg/whatsapp/client.go` is a 654-line concrete struct with no interface. Per Constitution Principle I (Adapter-First) and II (Strangler Pattern), we must NOT modify it. Instead, we extract a common interface and create a new adapter that wraps `whatsmeow.Client`. Handler code references the interface, and the active implementation is selected at startup based on `config.toml`.

**Alternatives considered**:
- Modify existing `pkg/whatsapp/` in-place: Violates Strangler Pattern (Principle II).
- No interface, direct whatsmeow usage: Violates Adapter-First (Principle I); locks to single provider.

## R-004: Instance vs Account Model Design

**Decision**: Create a new `WhatsAppInstance` model in `internal/models/` with a separate `whatsapp_instances` table. The `WhatsAppAccount` model and `whatsapp_accounts` table remain untouched.

**Rationale**: The existing `WhatsAppAccount` is tightly coupled to Meta Cloud API fields (PhoneID, BusinessID, AccessToken, AppSecret, WebhookVerifyToken, APIVersion). The new whatsmeow instance requires completely different fields (JID, session reference, connection status lifecycle). Per Constitution Principle II, we keep the old table for rollback safety and create a new one.

**Alternatives considered**:
- Add columns to `whatsapp_accounts`: Would create a confusing hybrid entity with nullable-everywhere fields.
- Rename `whatsapp_accounts`: Breaking change; violates rollback safety.

## R-005: Contact/Message Foreign Key Strategy

**Decision**: Add a nullable `InstanceID` (UUID FK) column to `contacts` and `messages` tables alongside the existing `whatsapp_account` string column. New records use `InstanceID`; old records retain `whatsapp_account` for backward compatibility.

**Rationale**: The current FK design uses `WhatsAppAccount.Name` (a string) rather than a UUID FK. The new instance model uses proper UUID primary keys. Adding a parallel column allows gradual migration without breaking existing queries. The migration script (US5) will populate `InstanceID` for existing records.

**Alternatives considered**:
- Replace string FK in-place: Breaking change for all existing queries and handlers.
- Dual-write: Complex and error-prone during transition.

## R-006: Rate Limiting & Anti-Ban Strategy

**Decision**: Implement a per-instance message queue with randomized delays (1-3 seconds between messages) and exponential backoff on errors.

**Rationale**: WhatsApp enforces undocumented rate limits. The whatsmeow library does not handle rate limiting internally. Projects using whatsmeow in production recommend human-like sending intervals with randomized jitter. The queue uses Go channels (per-instance goroutine) with configurable delay ranges in `config.toml`.

**Alternatives considered**:
- Fixed delay: Too predictable; higher ban risk.
- No delay (burst sending): High ban risk.
- External queue (Redis/RabbitMQ): Violates Single-Binary Simplicity (Principle VI).

## R-007: Config Endpoint for Provider Detection

**Decision**: Add a `/api/config` endpoint that returns `{ "whatsapp_provider": "whatsmeow" | "meta" }` based on `config.toml` setting.

**Rationale**: Per clarification Q2, the frontend needs a runtime-determined flag to hide Meta-only features. A backend config endpoint is the single source of truth, works regardless of whether instances exist, and supports potential future dual-mode.

**Alternatives considered**:
- Compile-time flag: No runtime flexibility.
- Instance-based inference: Fragile; fails when no instances exist yet.
