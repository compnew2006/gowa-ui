# Whatsmeow Phone Number Authentication Guide (Go)

This guide covers the complete login lifecycle for a WhatsApp bot using `whatsmeow`, including:

- Device credential storage and lifecycle
- QR login and phone-number pairing code login
- Error handling and reconnect strategy
- Production schema and deployment practices
- Multi-device behavior and external logout handling

## Version Baseline

This repo currently pins:

- `go.mau.fi/whatsmeow v0.0.0-20260216124546-34b971e686b6`

API behavior in this guide is aligned with that version.

## 1. Authentication Modes (What Is Actually Supported)

For WhatsApp MD through `whatsmeow`, you have two pairing paths:

1. QR pairing:
- Create a fresh device store (`Store.ID == nil`)
- Call `GetQRChannel(ctx)` before `Connect()`
- Render QR values from `QRChannelItem{Event: "code", Code: ...}`

2. Phone-number pairing code (not SMS OTP):
- Still starts from an unpaired session and active websocket
- Wait until QR flow is active (first QR event)
- Call `PairPhone(ctx, phoneE164Digits, showPushNotification, clientType, clientDisplayName)`
- Show returned 8-char code (format like `ABCD-EFGH`) to the user to enter in WhatsApp

Important:

- `whatsmeow` does not provide direct SMS OTP login APIs.
- The phone-number path is WhatsApp's linked-device code flow.

## 2. Credential Storage Model (Do This in Production)

Use a two-layer model:

1. Layer A: `whatsmeow` cryptographic/session tables (managed by `sqlstore`)
- This stores keys, sessions, app-state sync data, prekeys, sender keys, etc.
- Do not copy these secrets into your own app tables.

2. Layer B: your app metadata table
- Stores tenancy + routing metadata (instance ID, JID, status, phone label, timestamps)
- Used to decide which device session to load (`GetDevice`) and to drive UI state

### `sqlstore` tables created by `Upgrade()`

From `store/sqlstore/upgrades/00-latest-schema.sql`:

- `whatsmeow_device`
- `whatsmeow_identity_keys`
- `whatsmeow_pre_keys`
- `whatsmeow_sessions`
- `whatsmeow_sender_keys`
- `whatsmeow_app_state_sync_keys`
- `whatsmeow_app_state_version`
- `whatsmeow_app_state_mutation_macs`
- `whatsmeow_contacts`
- `whatsmeow_chat_settings`
- `whatsmeow_message_secrets`
- `whatsmeow_privacy_tokens`
- `whatsmeow_lid_map`
- `whatsmeow_event_buffer`

## 3. Suggested App Metadata Schema

Use one row per bot instance/session owner:

```sql
CREATE TABLE IF NOT EXISTS wa_bot_sessions (
  instance_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  owner_user_id UUID,
  jid TEXT UNIQUE,
  phone_e164 TEXT,
  login_mode TEXT NOT NULL CHECK (login_mode IN ('qr', 'phone_code')),
  status TEXT NOT NULL CHECK (status IN (
    'disconnected', 'pairing', 'connected', 'logged_out', 'banned', 'error', 'conflict'
  )),
  last_disconnect_reason TEXT,
  last_connected_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_wa_bot_sessions_tenant_jid
  ON wa_bot_sessions (tenant_id, jid)
  WHERE jid IS NOT NULL;
```

Rules:

- `jid` is null before pairing succeeds.
- After `PairSuccess`, persist `jid` and switch status to `connected`.
- On external logout, set status `logged_out` and clear `jid` only if your routing requires rebind-by-pairing.

## 4. Correct API Sequence

### 4.1 Startup / resume path

1. Open SQL DB connection
2. Build `sqlstore.Container` and run `Upgrade(ctx)`
3. Load app metadata row for instance
4. If metadata has `jid`, call `container.GetDevice(ctx, parsedJID)`
5. If no device found, fall back to `container.NewDevice()` and mark status `pairing`
6. Create `client := whatsmeow.NewClient(device, logger)`
7. Register event handlers before connect
8. If `client.Store.ID == nil`, create `qrChan := client.GetQRChannel(ctx)` **before** `Connect()`
9. Call `client.Connect()`
10. Wait for `*events.Connected`

### 4.2 New login with QR

1. `GetQRChannel(ctx)`
2. `Connect()`
3. Emit/render each `QRChannelEventCode`
4. Wait `success` terminal event and then `*events.Connected`
5. Persist JID from `*events.PairSuccess` (or `client.Store.ID` after `Connected`)

### 4.3 New login with phone-number pairing code

1. `GetQRChannel(ctx)`
2. `Connect()`
3. Wait first QR channel item (`Event == "code"`)
4. Immediately call `PairPhone(...)`
5. Show returned code to user
6. Wait for `*events.PairSuccess` then `*events.Connected`

Notes:

- `PairPhone` must be called after connection is ready; the source docs explicitly recommend waiting for QR readiness.
- Pairing windows are short; generate pairing code right after connect.

## 5. Working Example (Full Login Flow)

This example shows:

- Metadata row migration
- Existing-session resume by JID
- Fresh pairing (QR or phone code)
- Event-based status updates
- External logout handling

Extra dependencies used by this sample:

```bash
go get github.com/jackc/pgx/v5/stdlib github.com/mdp/qrterminal/v3
```

```go
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	qrterminal "github.com/mdp/qrterminal/v3"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type LoginMode string

const (
	LoginModeQR       LoginMode = "qr"
	LoginModePhone    LoginMode = "phone_code"
	StatusDisconnected          = "disconnected"
	StatusPairing               = "pairing"
	StatusConnected             = "connected"
	StatusLoggedOut             = "logged_out"
	StatusBanned                = "banned"
	StatusConflict              = "conflict"
	StatusError                 = "error"
)

type SessionMeta struct {
	InstanceID           uuid.UUID
	TenantID             uuid.UUID
	JID                  sql.NullString
	PhoneE164            sql.NullString
	LoginMode            string
	Status               string
	LastDisconnectReason sql.NullString
	LastConnectedAt      sql.NullTime
}

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Migrate(ctx context.Context) error {
	const q = `
CREATE TABLE IF NOT EXISTS wa_bot_sessions (
  instance_id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL,
  owner_user_id UUID,
  jid TEXT UNIQUE,
  phone_e164 TEXT,
  login_mode TEXT NOT NULL CHECK (login_mode IN ('qr', 'phone_code')),
  status TEXT NOT NULL CHECK (status IN (
    'disconnected', 'pairing', 'connected', 'logged_out', 'banned', 'error', 'conflict'
  )),
  last_disconnect_reason TEXT,
  last_connected_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`
	_, err := r.db.ExecContext(ctx, q)
	return err
}

func (r *SessionRepo) Upsert(ctx context.Context, m SessionMeta) error {
	const q = `
INSERT INTO wa_bot_sessions (
  instance_id, tenant_id, jid, phone_e164, login_mode, status, last_disconnect_reason, last_connected_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
ON CONFLICT (instance_id) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  jid = EXCLUDED.jid,
  phone_e164 = EXCLUDED.phone_e164,
  login_mode = EXCLUDED.login_mode,
  status = EXCLUDED.status,
  last_disconnect_reason = EXCLUDED.last_disconnect_reason,
  last_connected_at = EXCLUDED.last_connected_at,
  updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, q,
		m.InstanceID,
		m.TenantID,
		nullable(m.JID),
		nullable(m.PhoneE164),
		m.LoginMode,
		m.Status,
		nullable(m.LastDisconnectReason),
		nullableTime(m.LastConnectedAt),
	)
	return err
}

func (r *SessionRepo) Get(ctx context.Context, instanceID uuid.UUID) (SessionMeta, error) {
	const q = `
SELECT instance_id, tenant_id, jid, phone_e164, login_mode, status, last_disconnect_reason, last_connected_at
FROM wa_bot_sessions WHERE instance_id = $1`
	var m SessionMeta
	err := r.db.QueryRowContext(ctx, q, instanceID).Scan(
		&m.InstanceID,
		&m.TenantID,
		&m.JID,
		&m.PhoneE164,
		&m.LoginMode,
		&m.Status,
		&m.LastDisconnectReason,
		&m.LastConnectedAt,
	)
	if err != nil {
		return SessionMeta{}, err
	}
	return m, nil
}

func nullable(v sql.NullString) any {
	if v.Valid {
		return v.String
	}
	return nil
}

func nullableTime(v sql.NullTime) any {
	if v.Valid {
		return v.Time
	}
	return nil
}

func normalizePhoneE164Digits(in string) string {
	b := strings.Builder{}
	for _, r := range in {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func main() {
	var (
		dsn        = flag.String("dsn", os.Getenv("WA_DSN"), "Postgres DSN (pgx)")
		instanceID = flag.String("instance-id", "", "Instance UUID")
		tenantID   = flag.String("tenant-id", "", "Tenant UUID")
		mode       = flag.String("mode", "qr", "qr or phone_code")
		phone      = flag.String("phone", "", "Phone number in E.164 (required for phone_code)")
	)
	flag.Parse()

	if *dsn == "" || *instanceID == "" || *tenantID == "" {
		log.Fatal("dsn, instance-id, and tenant-id are required")
	}

	instUUID, err := uuid.Parse(*instanceID)
	if err != nil {
		log.Fatal(err)
	}
	tenantUUID, err := uuid.Parse(*tenantID)
	if err != nil {
		log.Fatal(err)
	}
	loginMode := LoginMode(*mode)
	if loginMode != LoginModeQR && loginMode != LoginModePhone {
		log.Fatal("mode must be qr or phone_code")
	}
	if loginMode == LoginModePhone && normalizePhoneE164Digits(*phone) == "" {
		log.Fatal("--phone is required for phone_code mode")
	}

	ctx := context.Background()

	db, err := sql.Open("pgx", *dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	repo := NewSessionRepo(db)
	if err := repo.Migrate(ctx); err != nil {
		log.Fatal(err)
	}

	waDBLog := waLog.Stdout("WA-DB", "INFO", true)
	container := sqlstore.NewWithDB(db, "postgres", waDBLog)
	if err := container.Upgrade(ctx); err != nil {
		log.Fatal(err)
	}

	meta, err := repo.Get(ctx, instUUID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Fatal(err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		meta = SessionMeta{
			InstanceID: instUUID,
			TenantID:   tenantUUID,
			LoginMode:  string(loginMode),
			Status:     StatusDisconnected,
		}
	}

	var device *store.Device
	if meta.JID.Valid {
		jid, parseErr := types.ParseJID(meta.JID.String)
		if parseErr != nil {
			log.Printf("invalid stored JID; forcing re-pair: %v", parseErr)
		}
		if parseErr == nil {
			d, getErr := container.GetDevice(ctx, jid)
			if getErr != nil {
				log.Fatal(getErr)
			}
			if d != nil {
				device = d
			}
		}
	}
	if device == nil {
		device = container.NewDevice()
		meta.Status = StatusPairing
		meta.JID = sql.NullString{}
		meta.LoginMode = string(loginMode)
		if loginMode == LoginModePhone {
			meta.PhoneE164 = sql.NullString{String: normalizePhoneE164Digits(*phone), Valid: true}
		}
		if err := repo.Upsert(ctx, meta); err != nil {
			log.Fatal(err)
		}
	}

	waClientLog := waLog.Stdout("WA", "INFO", true)
	client := whatsmeow.NewClient(device, waClientLog)
	client.EnableAutoReconnect = true

	var connectedOnce sync.Once
	connectedCh := make(chan struct{})

	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.PairSuccess:
			meta.JID = sql.NullString{String: v.ID.String(), Valid: true}
			meta.Status = StatusConnected
			now := time.Now().UTC()
			meta.LastConnectedAt = sql.NullTime{Time: now, Valid: true}
			if err := repo.Upsert(context.Background(), meta); err != nil {
				log.Printf("upsert pair success failed: %v", err)
			}
			log.Printf("paired: jid=%s", v.ID)
		case *events.Connected:
			if client.Store.ID != nil {
				meta.JID = sql.NullString{String: client.Store.ID.String(), Valid: true}
			}
			meta.Status = StatusConnected
			now := time.Now().UTC()
			meta.LastConnectedAt = sql.NullTime{Time: now, Valid: true}
			meta.LastDisconnectReason = sql.NullString{}
			if err := repo.Upsert(context.Background(), meta); err != nil {
				log.Printf("upsert connected failed: %v", err)
			}
			connectedOnce.Do(func() { close(connectedCh) })
		case *events.LoggedOut:
			meta.Status = StatusLoggedOut
			meta.LastDisconnectReason = sql.NullString{String: v.Reason.String(), Valid: true}
			if err := repo.Upsert(context.Background(), meta); err != nil {
				log.Printf("upsert logged_out failed: %v", err)
			}
			// External logout already deletes local session inside whatsmeow.
			log.Printf("logged out externally: %s", v.Reason)
		case *events.TemporaryBan:
			meta.Status = StatusBanned
			meta.LastDisconnectReason = sql.NullString{String: v.String(), Valid: true}
			if err := repo.Upsert(context.Background(), meta); err != nil {
				log.Printf("upsert banned failed: %v", err)
			}
		case *events.StreamReplaced:
			meta.Status = StatusConflict
			meta.LastDisconnectReason = sql.NullString{String: "stream replaced (duplicate active client)", Valid: true}
			if err := repo.Upsert(context.Background(), meta); err != nil {
				log.Printf("upsert conflict failed: %v", err)
			}
		case events.PermanentDisconnect:
			log.Printf("permanent disconnect: %s", v.PermanentDisconnectDescription())
		}
	})

	var qrChan <-chan whatsmeow.QRChannelItem
	if client.Store.ID == nil {
		qrChan, err = client.GetQRChannel(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}

	if err := client.Connect(); err != nil {
		meta.Status = StatusError
		meta.LastDisconnectReason = sql.NullString{String: err.Error(), Valid: true}
		_ = repo.Upsert(context.Background(), meta)
		log.Fatal(err)
	}

	if qrChan != nil {
		go func() {
			phoneRequested := false
			for item := range qrChan {
				switch item.Event {
				case whatsmeow.QRChannelEventCode:
					if loginMode == LoginModeQR {
						fmt.Printf("scan QR (valid ~%s):\n", item.Timeout)
						qrterminal.GenerateHalfBlock(item.Code, qrterminal.L, os.Stdout)
						fmt.Println()
						continue
					}

					if loginMode == LoginModePhone && !phoneRequested {
						code, pairErr := client.PairPhone(
							ctx,
							normalizePhoneE164Digits(*phone),
							true,
							whatsmeow.PairClientChrome,
							"Chrome (Linux)",
						)
						if pairErr != nil {
							log.Printf("pair code request failed: %v", pairErr)
							continue
						}
						phoneRequested = true
						fmt.Printf("enter this linking code in WhatsApp: %s\n", code)
					}
				case whatsmeow.QRChannelEventError:
					log.Printf("pairing channel error: %v", item.Error)
				case "err-scanned-without-multidevice":
					log.Printf("phone must enable linked devices, then retry scan/code")
				case "err-client-outdated":
					log.Printf("client version rejected by WhatsApp; update whatsmeow")
				case "timeout":
					log.Printf("pairing timeout; call Connect() again to request fresh QR/codes")
				case "success":
					log.Printf("pairing success; waiting for Connected event")
				}
			}
		}()
	}

	select {
	case <-connectedCh:
		fmt.Println("authenticated and connected")
	case <-time.After(3 * time.Minute):
		log.Fatal("timed out waiting for authenticated connection")
	}

	// Keep process alive for bot runtime.
	select {}
}

```

Implementation note for the example:

- The auth sequence and event handling are production-correct.
- If you prefer fewer dependencies, replace terminal QR rendering with your own UI renderer.

## 6. Reconnect and Failure Strategy

Use both transport-level and event-level handling:

1. Transport reconnect:
- `EnableAutoReconnect` is on by default
- `Disconnected` events trigger auto-reconnect for eligible cases

2. Permanent disconnect class (`events.PermanentDisconnect`):
- Includes `LoggedOut`, `TemporaryBan`, `StreamReplaced`, `ClientOutdated`, and some connect failures
- Treat these as state transitions requiring operator/user action, not blind retry

3. External logout/device deletion:
- `LoggedOut` event is emitted
- `whatsmeow` deletes local device store for logout-type connect failures and device-removed stream errors
- Mark your instance `logged_out` and require fresh pairing

4. Self-initiated unlink:
- `client.Logout(ctx)` unlinks + disconnects + deletes local store
- This does **not** emit `LoggedOut` event (by design)

Recommended retry policy:

- Retry immediately for temporary network failures
- Cap retries and alert on repeated reconnect loops
- Stop automatic reconnect and surface action-required status on permanent disconnects

## 7. Multi-Device Guidance

For SaaS bots with many WhatsApp sessions:

1. One `Client` per linked WhatsApp identity
- Map by `instance_id -> jid`
- Keep each client isolated (goroutine/service object)

2. Prevent duplicate live connections for same JID
- If two workers connect same session, `StreamReplaced` can occur
- Use distributed locking or leader election per `instance_id`

3. On startup
- Load all active app sessions
- For each with valid JID: `GetDevice` + `Connect`
- For each without JID: keep in `disconnected/pairing` until user triggers new login

4. Horizontal scaling
- Keep all workers pointed to the same SQL credential store
- But ensure only one active connector per instance at a time

## 8. What Happens When User Logs Out on Their Phone?

When the user removes your linked device from WhatsApp:

1. Server sends logout/connect-failure signals
2. `whatsmeow` emits `*events.LoggedOut`
3. `whatsmeow` deletes stored local session credentials for that device
4. Your client cannot reconnect with old credentials
5. Your app should:
- Set status to `logged_out`
- Notify user/admin
- Trigger a new QR/phone-code pairing flow

## 9. Production Checklist

- Use PostgreSQL with encrypted disks and restricted DB roles
- Call `container.Upgrade(ctx)` at startup
- Never log private key material or full QR payloads in prod logs
- Persist only routing metadata in app tables; leave crypto material in sqlstore tables
- Health-check each instance (`connected`, reconnect error count, last connected time)
- Alert on `logged_out`, `banned`, `client_outdated`, `stream_replaced`
- Back up DB regularly; treat backups as highly sensitive

## Primary Sources

- `whatsmeow` repository: [https://github.com/tulir/whatsmeow](https://github.com/tulir/whatsmeow)
- Package docs (`Client`, `GetQRChannel`, `PairPhone`, `Logout`): [https://pkg.go.dev/go.mau.fi/whatsmeow](https://pkg.go.dev/go.mau.fi/whatsmeow)
- Events docs (`LoggedOut`, `TemporaryBan`, `PermanentDisconnect`, `ConnectFailureReason`): [https://pkg.go.dev/go.mau.fi/whatsmeow/types/events](https://pkg.go.dev/go.mau.fi/whatsmeow/types/events)
- SQL store docs (`Container`, `GetDevice`, `NewDevice`, `Upgrade`): [https://pkg.go.dev/go.mau.fi/whatsmeow/store/sqlstore](https://pkg.go.dev/go.mau.fi/whatsmeow/store/sqlstore)
- SQL store latest schema (table list): [https://github.com/tulir/whatsmeow/blob/main/store/sqlstore/upgrades/00-latest-schema.sql](https://github.com/tulir/whatsmeow/blob/main/store/sqlstore/upgrades/00-latest-schema.sql)
