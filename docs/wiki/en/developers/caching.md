---
title: Caching System
---

# Caching System

Whatomate uses Redis for caching frequently accessed data to reduce database load and improve response times.

## Cache Architecture

The cache layer sits between handlers and the database. On a cache miss, data is loaded from the database, stored in Redis with a TTL, and returned. On updates, the relevant cache keys are invalidated.

```
Request → Cache Get → Hit? → Return
                    → Miss? → DB Query → Cache Set → Return
Update → DB Write → Cache Invalidate
```

## Cached Data Types

| Data Type | Cache Key Pattern | TTL | Source |
|-----------|------------------|-----|--------|
| WhatsApp Accounts | `account:phone_number_id:{pnid}` | 5 minutes | `whatsapp_accounts` table |
| Role Permissions | `role:permissions:{role_id}` | 10 minutes | `permissions` table |
| Chatbot Settings | `chatbot:settings:{org_id}` | 5 minutes | `chatbot_settings` table |
| Organization Settings | `org:settings:{org_id}` | 5 minutes | `organizations` table |

## Cache Operations

### Get

Check Redis for the cached value. Returns the deserialized value if found.

```go
func GetAccountByPhoneNumberIDCached(phoneNumberID string) (*WhatsAppAccount, error) {
    key := fmt.Sprintf("account:phone_number_id:%s", phoneNumberID)
    cached, err := redisClient.Get(ctx, key).Result()
    if err == nil {
        // Cache hit
        var account WhatsAppAccount
        json.Unmarshal([]byte(cached), &account)
        return &account, nil
    }
    // Cache miss — load from database
    return loadAccountFromDB(phoneNumberID)
}
```

### Miss

On cache miss, load from database and store with TTL:

```go
func cacheAccount(account *WhatsAppAccount, phoneNumberID string) {
    key := fmt.Sprintf("account:phone_number_id:%s", phoneNumberID)
    data, _ := json.Marshal(account)
    redisClient.Set(ctx, key, data, 5*time.Minute)
}
```

### Hit

Cache hits return immediately without database access, reducing latency from ~10ms (DB query) to ~1ms (Redis GET).

### Invalidate

Cache keys are deleted when the underlying data changes:

```go
func InvalidateAccountCache(phoneNumberID string) {
    key := fmt.Sprintf("account:phone_number_id:%s", phoneNumberID)
    redisClient.Del(ctx, key)
}

func InvalidateRolePermissionsCache(roleID uint) {
    key := fmt.Sprintf("role:permissions:%d", roleID)
    redisClient.Del(ctx, key)
}

func InvalidateChatbotSettingsCache(orgID uint) {
    key := fmt.Sprintf("chatbot:settings:%d", orgID)
    redisClient.Del(ctx, key)
}
```

## TTL Settings

| Data Type | TTL | Rationale |
|-----------|-----|-----------|
| Accounts | 5 minutes | Account credentials change infrequently; short TTL ensures timely rotation |
| Role Permissions | 10 minutes | Permissions change rarely; longer TTL reduces DB load during auth checks |
| Chatbot Settings | 5 minutes | Settings may change during configuration; moderate TTL balances freshness and performance |
| Organization Settings | 5 minutes | Org settings change occasionally; moderate TTL |

## Cache Key Patterns

```
account:phone_number_id:{phone_number_id}   → WhatsAppAccount JSON
role:permissions:{role_id}                  → []Permission JSON
chatbot:settings:{org_id}                   → ChatbotSettings JSON
org:settings:{org_id}                       → Organization settings JSON
```

## Cache Invalidation Triggers

| Action | Invalidated Cache |
|--------|-------------------|
| Create/Update/Delete Account | `account:phone_number_id:*` |
| Create/Update/Delete Role | `role:permissions:{role_id}` |
| Update Chatbot Settings | `chatbot:settings:{org_id}` |
| Update Organization Settings | `org:settings:{org_id}` |

## Implementation Details

**Source Files:** `internal/handlers/cache.go`

```go
// GetRolePermissionsCached loads permissions from cache or database
func GetRolePermissionsCached(roleID uint) ([]Permission, error) {
    key := fmt.Sprintf("role:permissions:%d", roleID)

    // Try cache first
    cached, err := RedisClient.Get(ctx, key).Result()
    if err == nil {
        var perms []Permission
        if err := json.Unmarshal([]byte(cached), &perms); err == nil {
            return perms, nil
        }
    }

    // Cache miss — load from DB
    var perms []Permission
    db.Where("role_id = ?", roleID).Find(&perms)

    // Store in cache
    if data, err := json.Marshal(perms); err == nil {
        RedisClient.Set(ctx, key, data, 10*time.Minute)
    }

    return perms, nil
}
```

## Redis Connection

Redis is configured via `config.toml` or environment variables:

```toml
[redis]
host = "127.0.0.1"
port = 6379
password = ""
db = 0
```

The connection is established at startup and reused across all cache operations. Connection pooling is handled by the Redis client library.

## Cache and Auth Flow

During authentication, role permissions are loaded from cache:

```
Login → Load user from DB
      → GetRolePermissionsCached(user.RoleID)
        → Cache hit: return permissions
        → Cache miss: query DB, cache for 10min, return
      → Generate JWT with permissions
```

## Cache and Webhook Processing

During incoming webhook processing, account lookup uses cache:

```
Webhook → Extract phone_number_id from payload
        → GetAccountByPhoneNumberIDCached(pnid)
          → Cache hit: return account (fast)
          → Cache miss: query DB, cache for 5min, return
        → Process message with account context
```

## See Also

- [Configuration System](../admins/configuration.md) — Redis connection configuration
- [Database Models](database-models.md) — Models that are cached
- [Monitoring](../admins/monitoring.md) — Redis health monitoring
