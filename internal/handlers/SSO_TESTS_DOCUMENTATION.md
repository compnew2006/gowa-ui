# SSO Handlers Test Documentation

## Overview
This document describes the comprehensive test suite for SSO (Single Sign-On) handlers located in `/internal/handlers/sso_handlers_test.go`.

## Test File Location
`/Users/noiemany/Downloads/whatomate_GOWA/whatomate/internal/handlers/sso_handlers_test.go`

## Test Coverage Summary

### 1. GetPublicSSOProviders Handler Tests

#### Test Cases:
- **TestApp_GetPublicSSOProviders_Success**: Verifies that enabled SSO providers are returned correctly with proper display names and deduplication
- **TestApp_GetPublicSSOProviders_NoProviders**: Tests the response when no SSO providers are configured
- **TestApp_GetPublicSSOProviders_Deduplication**: Ensures that the same provider from multiple organizations is deduplicated in the response

**What's Tested:**
- Public endpoint (no authentication required)
- Provider deduplication across organizations
- Display name mapping (google -> Google, microsoft -> Microsoft, etc.)
- Filtering of disabled providers
- Proper JSON response structure

### 2. InitSSO Handler Tests (OAuth Flow Initiation)

#### Test Cases:
- **TestApp_InitSSO_Success**: Validates successful OAuth flow initiation with state storage in Redis
- **TestApp_InitSSO_InvalidProvider**: Tests rejection of unsupported SSO providers
- **TestApp_InitSSO_ProviderNotConfigured**: Verifies error when provider is not configured
- **TestApp_InitSSO_ProviderDisabled**: Tests error handling when provider is disabled
- **TestApp_InitSSO_CustomProvider**: Validates custom OAuth2 provider configuration

**What's Tested:**
- Provider validation (google, microsoft, github, facebook, custom)
- OAuth state generation and storage in Redis
- State expiration handling (5-minute TTL)
- Redirect URL generation to provider's authorization endpoint
- Callback URL construction with scheme, host, and base path
- Error responses for invalid configurations

### 3. CallbackSSO Handler Tests (OAuth Callback Processing)

#### Success Scenarios:
- **TestApp_CallbackSSO_Success**: End-to-end test of successful SSO login with user auto-creation
- **TestApp_CallbackSSO_ExistingUserUpdatesSSOInfo**: Tests updating existing user's SSO information
- **TestApp_CallbackSSO_ExistingUserUpdatesSSOInfo**: Verifies that existing users get SSO info updated

#### Error Scenarios:
- **TestApp_CallbackSSO_OAuthError**: Handles OAuth provider errors (access_denied, etc.)
- **TestApp_CallbackSSO_MissingParameters**: Validates required code and state parameters
- **TestApp_CallbackSSO_InvalidState**: Tests invalid or missing state from Redis
- **TestApp_CallbackSSO_ExpiredState**: Verifies rejection of expired state tokens
- **TestApp_CallbackSSO_EmailDomainRestriction**: Tests email domain validation
- **TestApp_CallbackSSO_AutoCreateDisabled**: Tests when user auto-creation is disabled
- **TestApp_CallbackSSO_InvalidEmail**: Tests handling of invalid email format from provider
- **TestApp_CallbackSSO_MissingEmail**: Tests handling of missing email from provider
- **TestApp_CallbackSSO_InactiveUser**: Verifies that inactive users cannot login via SSO

**What's Tested:**
- OAuth code exchange for access token
- User info fetching from various providers (Google, Microsoft, GitHub, Facebook, Custom)
- State validation and replay attack prevention
- Email domain restriction validation
- User auto-creation with role assignment
- Existing user SSO info updates
- Inactive user detection
- JWT token generation (access and refresh tokens)
- Auth cookie setting (whm_access, whm_refresh, whm_csrf)
- Error redirects with proper error messages
- Role lookup for auto-created users
- UserOrganization entry creation

### 4. GetSSOSettings Handler Tests (Admin Only)

#### Test Cases:
- **TestApp_GetSSOSettings_Success**: Retrieves SSO settings for an organization
- **TestApp_GetSSOSettings_Unauthorized**: Tests permission checking for non-admin users
- **TestApp_GetSSOSettings_NoProviders**: Tests response when no providers configured

**What's Tested:**
- Authorization requirement (settings:sso:read permission)
- Secret masking (ClientSecret never exposed)
- Multiple provider support per organization
- Proper JSON response structure
- Error handling for unauthorized access

### 5. UpdateSSOProvider Handler Tests (Admin Only)

#### Test Cases:
- **TestApp_UpdateSSOProvider_CreateNew**: Creates a new SSO provider configuration
- **TestApp_UpdateSSOProvider_UpdateExisting**: Updates existing provider without changing secret if not provided
- **TestApp_UpdateSSOProvider_InvalidProvider**: Validates provider type
- **TestApp_UpdateSSOProvider_CustomProviderMissingURLs**: Ensures custom providers have required URLs
- **TestApp_UpdateSSOProvider_CustomProviderSuccess**: Tests custom provider creation
- **TestApp_UpdateSSOProvider_Unauthorized**: Tests permission checking

**What's Tested:**
- Authorization requirement (settings:sso:write permission)
- Provider validation (google, microsoft, github, facebook, custom)
- Custom provider URL validation (auth_url, token_url, user_info_url)
- Client secret encryption
- Secret update logic (only updates if new secret provided)
- Default role assignment (defaults to "agent")
- Allowed domains configuration
- Auto-create user flag
- Enable/disable functionality

### 6. DeleteSSOProvider Handler Tests (Admin Only)

#### Test Cases:
- **TestApp_DeleteSSOProvider_Success**: Successfully deletes an SSO provider
- **TestApp_DeleteSSOProvider_NotFound**: Tests deletion of non-existent provider
- **TestApp_DeleteSSOProvider_Unauthorized**: Tests permission checking

**What's Tested:**
- Authorization requirement (settings:sso:write permission)
- Provider deletion from database
- Proper error handling for non-existent providers
- Success response with confirmation message

## Helper Functions

### Test Helpers:
1. **createTestSSOProvider**: Creates a test SSO provider with encrypted secret
2. **createSSOAdminRole**: Creates a role with SSO permissions
3. **createSSOAdminRequest**: Creates an authenticated request for SSO operations
4. **createMockOAuthServer**: Creates a mock HTTP server for OAuth provider simulation
5. **extractStateFromAuthURL**: Extracts state parameter from OAuth authorization URL

## Mock OAuth Server

The test suite includes a mock OAuth server that simulates:
- **Token Endpoint**: `/oauth/token` - Returns mock access tokens
- **User Info Endpoint**: `/oauth/userinfo` - Returns mock user information

This allows testing the complete OAuth flow without external dependencies.

## Testing Approach

### 1. Unit Testing
Each handler function is tested in isolation with:
- Mocked dependencies (database, Redis, HTTP client)
- Controlled input scenarios
- Verified output and side effects

### 2. Integration Testing
The callback handler tests include integration testing of:
- Database operations (user creation, updates)
- Redis operations (state storage and retrieval)
- HTTP client operations (OAuth token exchange, user info fetching)
- JWT token generation
- Cookie setting

### 3. Security Testing
Security aspects tested include:
- State validation and replay attack prevention
- Secret masking in API responses
- Authorization and permission checking
- Email domain validation
- Redirect URL sanitization
- Inactive user detection

### 4. Error Handling
Comprehensive error scenarios tested:
- Invalid providers
- Missing or expired state
- OAuth provider errors
- Invalid user information
- Database errors
- Permission errors
- Configuration errors

## Test Data Management

### Database:
- Uses `testutil.SetupTestDB(t)` for isolated test database
- Each test creates its own organizations, roles, and users
- Proper cleanup through test framework

### Redis:
- Uses `testutil.SetupTestRedis(t)` for isolated Redis instance
- State tokens stored with proper TTL
- Tests verify state deletion after use (replay prevention)

### HTTP Mocking:
- Uses `httptest.NewServer` for OAuth provider mocking
- Simulates various provider responses
- Tests error conditions (HTTP errors, invalid responses)

## Running the Tests

```bash
# Run all SSO handler tests
go test -v ./internal/handlers -run TestApp_CallbackSSO

# Run specific test
go test -v ./internal/handlers -run TestApp_GetPublicSSOProviders_Success

# Run all tests in the file
go test -v ./internal/handlers/sso_handlers_test.go
```

## Dependencies

- **github.com/stretchr/testify**: Assertion and test helpers
- **github.com/compnew2006/whatomate/test/testutil**: Test utilities
- **github.com/google/uuid**: UUID generation and parsing
- **github.com/valyala/fasthttp**: HTTP server framework
- **golang.org/x/oauth2**: OAuth2 client implementation
- **gorm.io/gorm**: Database ORM

## Coverage Metrics

The test suite covers:
- **Handler Functions**: 100% (all 6 handlers)
- **Success Paths**: Comprehensive coverage
- **Error Paths**: Comprehensive coverage
- **Security Scenarios**: Comprehensive coverage
- **Edge Cases**: Well covered

## Notes

1. **Test Isolation**: Each test is independent and can run in any order
2. **Mock Dependencies**: All external dependencies are mocked for reliability
3. **Realistic Data**: Tests use realistic data structures matching production
4. **Error Messages**: Tests verify proper error messages are returned
5. **HTTP Status Codes**: Tests verify correct status codes for all scenarios
6. **Side Effects**: Tests verify database and Redis state changes
7. **Security**: Tests verify security measures (state validation, authorization, etc.)

## Future Enhancements

Potential additions to the test suite:
1. Performance tests for high-volume SSO logins
2. Concurrent request testing
3. Rate limiting tests
4. Additional provider-specific tests
5. Token refresh flow tests
6. Session management tests
