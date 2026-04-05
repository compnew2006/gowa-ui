---
title: Quick Start Guide
---

# Quick Start Guide

Get Whatomate up and running in minutes. This guide walks you through installation, configuration, and first login.

## Prerequisites

Before you begin, ensure you have the following installed:

| Component | Version | Purpose |
|-----------|---------|---------|
| **Go** | 1.21+ | Backend runtime and build tool |
| **PostgreSQL** | 13+ | Primary database |
| **Redis** | 6+ | Caching, queuing, and session management |
| **Node.js** | 18+ (optional) | Frontend development/build |

## Step 1: Clone the Repository

```bash
git clone https://github.com/whatomate/whatomate.git
cd whatomate
```

## Step 2: Configure the Application

Copy the example configuration file and edit it:

```bash
cp config.example.toml config.toml
```

Edit `config.toml` with your settings:

```toml
[app]
name = "Whatomate"
environment = "development"
encryption_key = "your-32-byte-encryption-key-here!!"

[server]
host = "0.0.0.0"
port = 8080
allowed_origins = ["http://localhost:5173"]

[database]
host = "localhost"
port = 5432
user = "whatomate"
password = "your_db_password"
dbname = "whatomate"
ssl_mode = "disable"

[redis]
host = "localhost"
port = 6379
password = ""
db = 0

[whatsapp]
provider = "meta"  # or "whatsmeow"
base_url = "https://graph.facebook.com/v18.0"
webhook_verify_token = "your-verify-token"

[jwt]
secret = "your-jwt-secret-min-32-characters!!"
access_token_ttl = "15m"
refresh_token_ttl = "7d"

[default_admin]
email = "admin@whatomate.local"
password = "Admin@1234"
full_name = "System Administrator"
```

> **Important:** Generate secure values for `encryption_key` and `jwt.secret`:
>
> ```bash
> openssl rand -base64 32  # encryption_key (use first 32 bytes)
> openssl rand -base64 64  # jwt.secret
> ```

## Step 3: Build the Frontend (Optional)

If you want the embedded frontend:

```bash
cd frontend
npm install
npm run build
cd ..
```

This outputs static files to `internal/frontend/dist/`, which are embedded into the Go binary.

For development, you can run the frontend separately on port 5173 and proxy API requests.

## Step 4: Run Database Migrations

```bash
go run cmd/whatomate/main.go -migrate
```

This will:

1. Create all database tables using GORM AutoMigrate
2. Create the default admin user (from `default_admin` config)
3. Create default roles (admin, manager, agent)
4. Create default chatbot settings for the admin's organization

## Step 5: Start the Server

```bash
go run cmd/whatomate/main.go
```

Or with a specific config file:

```bash
go run cmd/whatomate/main.go -config /path/to/config.toml
```

The server will start on the configured port (default: `8080`).

## Step 6: Access the Frontend

- **Embedded frontend:** Open `http://localhost:8080` in your browser
- **Development frontend:** Open `http://localhost:5173` (if running the Vite dev server separately)

## Step 7: First Login

Log in with the default admin credentials from your `config.toml`:

- **Email:** `admin@whatomate.local` (or whatever you configured)
- **Password:** `Admin@1234` (or whatever you configured)

> **Security note:** Change the default admin password immediately after first login.

## Step 8: Post-Setup Tasks

After logging in:

1. **Change the default admin password** — Go to your profile and update the password
2. **Configure WhatsApp provider** — Set up your Meta account or WhatsMeow instance
3. **Create your organization** — If not auto-created during migration
4. **Invite team members** — Send invitation tokens to your team
5. **Set up chatbot** — Configure greeting messages, business hours, and keyword rules

## Docker Quick Start

For the fastest setup, use Docker Compose:

```bash
# Create environment file with secure secrets
cat > .env << EOF
DB_PASSWORD=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 64)
ENCRYPTION_KEY=$(openssl rand -base64 32 | head -c 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
EOF

# Start all services
docker compose up -d

# Run migrations
docker compose exec whatomate ./whatomate -migrate
```

Access the application at `http://localhost:8080`.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Database connection failed | Verify PostgreSQL is running and credentials are correct |
| Redis connection failed | Check Redis is running and accessible |
| Migration errors | Ensure database user has CREATE TABLE permissions |
| Port already in use | Change `server.port` in config.toml |
| Frontend not loading | Build the frontend or run the Vite dev server |

## Next Steps

- Read the [Platform Overview](overview.md) to understand Whatomate's architecture
- Explore the [User Guide](users/index.md) for feature documentation
- Review the [Admin Guide](admins/index.md) for deployment and operations
- Check the [FAQ](faq.md) for common questions

## See Also

- [Configuration Reference](admins/configuration.md) — All configuration options
- [Deployment Guide](admins/deployment.md) — Production deployment
- [Security Guide](admins/security.md) — Security best practices
