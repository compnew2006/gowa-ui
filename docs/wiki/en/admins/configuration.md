---
title: Configuration
---

# Configuration

Whatomate is configured via a TOML file with environment variable overrides for all settings.

## Configuration File

Create `config.toml` in the project root or specify with `-config` flag:

```bash
./whatomate serve -config /etc/whatomate/config.toml
```

## Complete Configuration Example

```toml
[app]
name = "Whatomate"
version = "1.0.0"
environment = "production"
debug = false
encryption_key = "your-32-byte-encryption-key-here!!"

[server]
host = "0.0.0.0"
port = 8080
read_timeout = "30s"
write_timeout = "30s"
allowed_origins = ["https://whatomate.example.com"]
max_request_body_size = 52428800  # 50MB

[database]
host = "127.0.0.1"
port = 5432
user = "whatomate"
password = "secure_db_password"
dbname = "whatomate"
ssl_mode = "disable"

[redis]
host = "127.0.0.1"
port = 6379
password = ""
db = 0

[whatsapp]
provider = "meta"  # "meta" or "whatsmeow"
base_url = "https://graph.facebook.com/v17.0"
webhook_verify_token = "your-webhook-verify-token"

[whatsmeow]
queue_depth = 100
rate_limit = 60  # messages per minute

[jwt]
secret = "your-jwt-secret-min-32-characters!!"
access_token_ttl = "15m"
refresh_token_ttl = "7d"

[default_admin]
email = "admin@whatomate.example.com"
password = "Admin@1234"
full_name = "System Administrator"

[storage]
local_path = "./storage"

[rate_limit]
enabled = true
per_user = 1000
per_ip = 100
```

## Configuration Sections

### app

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `name` | string | No | Application display name |
| `version` | string | No | Application version |
| `environment` | string | No | "development" or "production" |
| `debug` | bool | No | Enable debug logging |
| `encryption_key` | string | **Yes** | 32-byte key for AES-256-GCM encryption |

**encryption_key**: Used to encrypt sensitive data (tokens, API keys, secrets). Must be exactly 32 bytes. Generate with:

```bash
openssl rand -base64 32
```

### server

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `host` | string | No | Bind address (default: `0.0.0.0`) |
| `port` | int | No | HTTP port (default: `8080`) |
| `read_timeout` | duration | No | Request read timeout (default: `30s`) |
| `write_timeout` | duration | No | Response write timeout (default: `30s`) |
| `allowed_origins` | []string | No | CORS allowed origins |
| `max_request_body_size` | int | No | Max request body in bytes (default: 50MB) |

### database

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `host` | string | **Yes** | PostgreSQL host |
| `port` | int | No | PostgreSQL port (default: `5432`) |
| `user` | string | **Yes** | Database user |
| `password` | string | **Yes** | Database password |
| `dbname` | string | **Yes** | Database name |
| `ssl_mode` | string | No | SSL mode (default: `disable`) |

### redis

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `host` | string | **Yes** | Redis host |
| `port` | int | No | Redis port (default: `6379`) |
| `password` | string | No | Redis password |
| `db` | int | No | Redis database number (default: `0`) |

### whatsapp

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `provider` | string | **Yes** | "meta" or "whatsmeow" |
| `base_url` | string | No | Meta API base URL |
| `webhook_verify_token` | string | No | Webhook subscription token |

### whatsmeow

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `queue_depth` | int | No | Per-instance queue depth (default: `100`) |
| `rate_limit` | int | No | Messages per minute per instance (default: `60`) |

### jwt

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `secret` | string | **Yes** | JWT signing secret (min 32 characters) |
| `access_token_ttl` | duration | No | Access token lifetime (default: `15m`) |
| `refresh_token_ttl` | duration | No | Refresh token lifetime (default: `7d`) |

### default_admin

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `email` | string | No | Default admin email |
| `password` | string | No | Default admin password |
| `full_name` | string | No | Default admin display name |

The default admin is created during the first migration if it doesn't already exist.

### storage

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `local_path` | string | No | Local file storage path (default: `./storage`) |

### rate_limit

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `enabled` | bool | No | Enable rate limiting (default: `true`) |
| `per_user` | int | No | Requests per window per user (default: `1000`) |
| `per_ip` | int | No | Requests per window per IP (default: `100`) |

## Environment Variable Overrides

All configuration values can be overridden with environment variables using the `WHATOMATE_` prefix and uppercase section/key names:

```bash
# Database
export WHATOMATE_DATABASE_HOST=prod-db.example.com
export WHATOMATE_DATABASE_PASSWORD=super_secret

# Redis
export WHATOMATE_REDIS_HOST=prod-redis.example.com
export WHATOMATE_REDIS_PASSWORD=redis_secret

# JWT
export WHATOMATE_JWT_SECRET=$(openssl rand -base64 64)

# Encryption
export WHATOMATE_APP_ENCRYPTION_KEY=$(openssl rand -base64 32)

# Server
export WHATOMATE_SERVER_PORT=443
export WHATOMATE_SERVER_ALLOWED_ORIGINS='["https://app.example.com"]'

# WhatsApp
export WHATOMATE_WHATSAPP_PROVIDER=whatsmeow
```

Environment variables take precedence over the TOML file.

## Encryption Key Setup

The encryption key is critical for data security. It encrypts:

- WhatsApp account access tokens
- Webhook verify tokens
- SSO client secrets
- Chatbot AI API keys
- Webhook secrets
- Custom action headers

**Important**: If you change the encryption key, you must run the crypto migration to re-encrypt all existing data:

```bash
whatomate crypto-migrate
```

See [Data Migration](data-migration.md) for details.

## JWT Secret Management

The JWT secret signs access and refresh tokens. It can be set via:

1. Environment variable: `WHATOMATE_JWT_SECRET`
2. Config file: `jwt.secret`

**Key rotation**: Changing the JWT secret invalidates all existing tokens. Users will need to log in again. Plan rotation during maintenance windows.

```bash
# Generate a strong JWT secret
openssl rand -base64 64
```

## Configuration Validation

On startup, Whatomate validates required configuration:

- `encryption_key` must be exactly 32 bytes
- `jwt.secret` must be at least 32 characters
- `database.host`, `database.user`, `database.dbname` must be set
- `redis.host` must be set
- `whatsapp.provider` must be "meta" or "whatsmeow"

Invalid configuration causes the server to exit with an error message.

## See Also

- [Deployment](deployment.md) — Production deployment with configuration
- [Security](security.md) — Security-related configuration
- [Data Migration](data-migration.md) — Crypto migration for encryption key changes
- [Troubleshooting](troubleshooting.md) — Configuration-related issues
