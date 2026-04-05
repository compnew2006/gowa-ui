---
title: Testing Infrastructure
---

# Testing Infrastructure

Whatomate uses a multi-layered testing approach with Go unit tests for backend logic and Playwright E2E tests for the frontend.

## Test Types

| Type | Location | Framework | Purpose |
|------|----------|-----------|---------|
| Unit Tests | `*_test.go` files | Go `testing` package | Handler logic, utilities, helpers |
| Integration Tests | `*_test.go` files | Go `testing` + test DB | Handler integration with database |
| E2E Tests | `frontend/e2e/` | Playwright (TypeScript) | Full-stack user workflows |

## Unit Tests

### Test File Naming

Test files follow the `*_test.go` convention alongside source files:

```
internal/handlers/
├── auth_handlers.go
├── auth_handlers_test.go          # Auth handler tests
├── send_restriction_policy_helpers_test.go  # Send restriction tests
├── contacts_helpers_test.go       # Contact helper tests
├── organization_delete_test.go    # Organization deletion tests
├── sla_processor_test.go          # SLA processor tests
└── testhelpers_test.go            # Shared test helpers
```

### Running Unit Tests

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/handlers/

# Run with verbose output
go test -v ./internal/handlers/

# Run with coverage
go test -coverprofile=coverage.out ./...

# Run specific test function
go test -run TestLogin ./internal/handlers/
```

### Test Helpers

**`testhelpers_test.go`** provides common test utilities:

```go
// Setup test database connection
func setupTestDB() *gorm.DB { ... }

// Create test user with role
func createTestUser(db *gorm.DB, role string) *models.User { ... }

// Create test organization
func createTestOrg(db *gorm.DB) *models.Organization { ... }

// Mock Redis client
func setupMockRedis() *redis.Client { ... }
```

**`stubs.go`** provides stub implementations:

```go
// StubMessageProvider for testing without real WhatsApp
type StubMessageProvider struct {
    SentMessages []OutgoingMessage
}

func (s *StubMessageProvider) SendMessage(msg OutgoingMessage) error {
    s.SentMessages = append(s.SentMessages, msg)
    return nil
}
```

### Coverage Reports

Coverage files are generated per test run:

```
coverage.out              # Main coverage report
coverage_handlers.out     # Handler package coverage
coverage_worker.out       # Worker package coverage
coverage_crypto.out       # Crypto package coverage
```

View coverage in browser:

```bash
go tool cover -html=coverage.out
```

## E2E Tests

### Location

```
frontend/e2e/
├── auth.spec.ts           # Login, registration, logout
├── contacts.spec.ts       # Contact CRUD, search, filters
├── messaging.spec.ts      # Send/receive messages
├── campaigns.spec.ts      # Campaign creation and management
├── chatbot.spec.ts        # Chatbot configuration
├── instances.spec.ts      # WhatsApp instance management
└── helpers/
    └── ApiHelper.ts       # TypeScript API test helper
```

### Running E2E Tests

```bash
# Install Playwright
cd frontend && npx playwright install

# Run E2E tests
npx playwright test

# Run with UI
npx playwright test --ui

# Run specific test file
npx playwright test auth.spec.ts

# Run with report
npx playwright test --reporter=html
```

### ApiHelper

The TypeScript `ApiHelper` class provides programmatic API access for E2E tests:

```typescript
class ApiHelper {
  private baseUrl: string;
  private authToken: string;

  async login(email: string, password: string): Promise<void> { ... }
  async get(path: string): Promise<Response> { ... }
  async post(path: string, body: any): Promise<Response> { ... }
  async put(path: string, body: any): Promise<Response> { ... }
  async delete(path: string): Promise<Response> { ... }

  // Test-specific helpers
  async createContact(data: ContactData): Promise<Contact> { ... }
  async sendMessage(contactId: number, content: string): Promise<Message> { ... }
  async createCampaign(data: CampaignData): Promise<Campaign> { ... }
}
```

## Test Database

Integration tests use a separate test database:

```toml
# config.test.toml
[database]
host = "127.0.0.1"
port = 5432
user = "whatomate_test"
password = "test_password"
dbname = "whatomate_test"
ssl_mode = "disable"
```

Tests create and clean up their own data using transactions:

```go
func TestCreateUser(t *testing.T) {
    db := setupTestDB()
    defer cleanupTestDB(db)

    tx := db.Begin()
    defer tx.Rollback()

    // Test logic with transaction
    user := createTestUser(tx, "admin")
    assert.NotNil(t, user.ID)
}
```

## Testing Patterns

### Handler Testing

```go
func TestLogin(t *testing.T) {
    // Setup
    db := setupTestDB()
    app := setupTestApp(db)
    createTestUser(db, "admin")

    // Execute
    req := createLoginRequest("admin@test.com", "password123")
    resp := app.Login(req)

    // Assert
    assert.Equal(t, 200, resp.StatusCode())
    assert.Contains(t, string(resp.Body()), "access_token")
}
```

### Mock Provider Testing

```go
func TestSendMessage(t *testing.T) {
    stub := &StubMessageProvider{}
    app := setupTestAppWithProvider(stub)

    app.SendMessage(createMessageRequest())

    assert.Len(t, stub.SentMessages, 1)
    assert.Equal(t, "Hello", stub.SentMessages[0].Content)
}
```

## CI Testing

Tests are run in CI with:

```bash
# Start test dependencies
docker compose -f docker-compose.test.yml up -d

# Run backend tests
go test -coverprofile=coverage.out ./...

# Run frontend E2E tests
cd frontend && npx playwright test

# Upload coverage
go tool cover -func=coverage.out
```

## See Also

- [Contributing Guide](contributing.md) — Code style and PR requirements
- [Architecture Overview](architecture.md) — System components to test
- [Error Handling Patterns](architecture.md#error-handling) — Error patterns to test
