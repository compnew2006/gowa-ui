---
title: Contributing Guide
---

# Contributing Guide

This guide covers code style, development patterns, and the PR process for Whatomate.

## Code Style

### Go Conventions

- Follow `gofmt` formatting (run `gofmt -w .` before committing)
- Use `ruff check .` for Python files if applicable
- Package names are lowercase, single-word
- Exported names use PascalCase; unexported use camelCase
- Interface names end with `-er` when possible (`Provider`, `Handler`)
- Error variables are prefixed with `Err` (`ErrNotFound`, `ErrUnauthorized`)

```go
// Good
type MessageProvider interface {
    SendMessage(msg OutgoingMessage) error
}

var ErrAccountNotFound = errors.New("account not found")

// Bad
type message_provider interface {
    send_message(msg outgoingMessageRequest) error
}
```

### File Organization

```
internal/
├── handlers/        # HTTP request handlers
├── middleware/      # HTTP middleware
├── models/          # GORM model definitions
├── config/          # Configuration loading
├── database/        # Database connection and migrations
├── crypto/          # Encryption utilities
├── queue/           # Redis queue system
├── worker/          # Background workers
└── websocket/       # WebSocket hub and messages

pkg/
├── whatsapp/        # Meta WhatsApp client
├── whatsmeow/       # WhatsMeow connection manager
└── provider/        # Provider abstraction interface
```

### Naming Conventions

| Entity | Convention | Example |
|--------|-----------|---------|
| Handlers | `App.ActionName()` | `App.ListContacts()` |
| Middleware | Descriptive noun | `AuthMiddleware`, `CSRFProtection` |
| Models | Singular PascalCase | `User`, `WhatsAppAccount` |
| Config structs | PascalCase | `AppConfig`, `DatabaseConfig` |
| Constants | UPPER_SNAKE_CASE | `MAX_RETRY_COUNT` |
| Error codes | camelCase | `instance_not_found`, `chat_closed` |

## Middleware Chain Order

Middleware is applied in this order for every request:

```
1. CORS Wrapper          (fasthttp level — handles preflight)
2. Security Headers      (X-Content-Type-Options, X-Frame-Options, etc.)
3. Request Logger        (logs method, path, duration, status)
4. Recovery              (panic recovery — returns 500)
5. CSRF Protection       (validates CSRF token for mutating requests)
6. Activity Log          (records significant actions)
7. Auth Middleware       (validates JWT or API key)
8. [Handler]             (business logic)
   ├── Permission Check  (requirePermission at handler level)
   ├── Provider Guard    (provider compatibility check)
   └── Rate Limiting     (per-endpoint rate limits)
```

When adding new middleware, consider where it fits in this chain. Security-related middleware should be early; business logic middleware should be after auth.

## Error Handling Patterns

### Error Envelope

All API errors follow a consistent JSON envelope:

```json
{
  "error": {
    "message": "Human-readable error message",
    "code": "machine_readable_code",
    "field": "field_name_if_validation_error"
  }
}
```

### HTTP Status Codes

| Status | Meaning | When to Use |
|--------|---------|-------------|
| 400 | Bad Request | Validation errors, malformed input |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Authenticated but lacks permission |
| 404 | Not Found | Resource does not exist |
| 409 | Conflict | Duplicate resource, closed chat |
| 413 | Payload Too Large | Request body exceeds limit |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Unexpected server failure |

### Error Handling Strategy

```go
func (app *App) CreateUser(c *fasthttp.RequestCtx) {
    // 1. Validate input early
    var req CreateUserRequest
    if err := json.Unmarshal(c.PostBody(), &req); err != nil {
        sendError(c, 400, "invalid_request", "Invalid JSON body")
        return
    }

    // 2. Check auth (handled by middleware)
    userID := getUserIDFromContext(c)

    // 3. Check permissions
    if !app.requirePermission(c, "users", "write") {
        sendError(c, 403, "permission_denied", "Insufficient permissions")
        return
    }

    // 4. Check resource exists
    if app.isEmailTaken(req.Email) {
        sendError(c, 409, "duplicate_email", "Email already exists")
        return
    }

    // 5. Execute operation
    user, err := app.createUser(req)
    if err != nil {
        sendError(c, 500, "internal_error", "Failed to create user")
        return
    }

    // 6. Return success
    sendJSON(c, 201, user)
}
```

### Reason Codes

Standardized reason codes enable programmatic error handling in the frontend:

```go
// internal/handlers/reason_codes.go
const (
    ReasonInstanceNotFound    = "instance_not_found"
    ReasonInstanceNotConnected = "instance_not_connected"
    ReasonInstanceNotAllowed  = "instance_not_allowed"
    ReasonChatUnclaimed       = "chat_unclaimed"
    ReasonChatClosed          = "chat_closed"
    ReasonRestrictionViolation = "restriction_violation"
)

// Error with reason code
func asInstanceSelectionError(code, message string) error {
    return &InstanceSelectionError{
        ReasonCode: code,
        Message:    message,
    }
}
```

## Adding New Features

### 1. Define the Model

Add the model to `internal/models/`:

```go
type NewFeature struct {
    gorm.Model
    Name           string `gorm:"not null"`
    OrganizationID uint   `gorm:"index"`
    Settings       datatypes.JSON
}
```

### 2. Run Migration

Migrations use GORM AutoMigrate. The model will be auto-created on startup with the `-migrate` flag:

```bash
./whatomate -migrate
```

Or trigger via API (super admin only):

```bash
curl -X POST https://whatomate.example.com/api/admin/migrate \
  -H "Authorization: Bearer <token>"
```

### 3. Add Cache Entry (if applicable)

If the data is frequently accessed, add cache support in `internal/handlers/cache.go`:

```go
func GetNewFeatureCached(id uint) (*NewFeature, error) {
    key := fmt.Sprintf("feature:%d", id)
    // ... cache get/miss/set pattern
}
```

### 4. Create Handlers

Add handlers in `internal/handlers/`:

```go
func (app *App) ListNewFeatures(c *fasthttp.RequestCtx) {
    if !app.requirePermission(c, "features", "read") {
        sendError(c, 403, "permission_denied", "Insufficient permissions")
        return
    }
    // ... query and return
}
```

### 5. Register Routes

Add routes in `cmd/whatomate/main.go`:

```go
api.GET("/features", app.AuthMiddleware(app.ListNewFeatures))
api.POST("/features", app.AuthMiddleware(app.CreateNewFeature))
api.PUT("/features/{id}", app.AuthMiddleware(app.UpdateNewFeature))
api.DELETE("/features/{id}", app.AuthMiddleware(app.DeleteNewFeature))
```

### 6. Add Tests

Create `*_test.go` file alongside handlers:

```go
func TestListNewFeatures(t *testing.T) {
    db := setupTestDB()
    app := setupTestApp(db)

    // Test with valid auth and permission
    // Test with missing permission
    // Test with empty results
}
```

### 7. Add Webhook Events (if applicable)

If the feature should trigger webhooks, add event types and dispatch calls:

```go
app.DispatchWebhook("feature_created", map[string]interface{}{
    "feature_id": feature.ID,
    "name":       feature.Name,
})
```

### 8. Add WebSocket Events (if applicable)

If the feature needs real-time updates:

```go
websocket.BroadcastToOrg(orgID, websocket.WSMessage{
    Type: "feature_updated",
    Payload: feature,
})
```

## Testing Requirements

- All new handlers should have unit tests
- Critical paths should have E2E tests
- Coverage should not decrease (check `coverage.out`)
- Run `go test ./...` before submitting PR

## PR Process

1. Create a feature branch: `git checkout -b feature/description`
2. Make changes following code style conventions
3. Add tests for new functionality
4. Run tests: `go test ./...`
5. Run linter: `gofmt -w .`
6. Commit with descriptive message
7. Push and create pull request
8. Address review feedback
9. Merge after approval

## Branch Naming

| Type | Pattern | Example |
|------|---------|---------|
| Feature | `feature/description` | `feature/campaign-scheduling` |
| Bug Fix | `fix/description` | `fix/webhook-signature` |
| Refactor | `refactor/description` | `refactor/cache-layer` |
| Docs | `docs/description` | `docs/api-reference` |
| Health | `desloppify/description` | `desloppify/code-health` |

## See Also

- [Architecture Overview](architecture.md) — System design and component relationships
- [Database Models](database-models.md) — Existing model patterns
- [Testing Infrastructure](testing.md) — Test setup and patterns
- [API Reference](api-reference.md) — REST API conventions
- [Webhook Integration](webhook-integration.md) — Outbound webhook patterns
