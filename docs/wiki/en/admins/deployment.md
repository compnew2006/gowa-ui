---
title: Deployment
---

# Deployment

Whatomate is a single-binary application with an embedded frontend, making deployment straightforward.

## Build Process

### Prerequisites

- Go 1.21+
- Node.js 18+ (for frontend build)
- npm or pnpm

### Build Frontend

```bash
cd frontend
npm install
npm run build
```

This outputs static files to `internal/frontend/dist/`, which are embedded into the Go binary via `//go:embed`.

### Build Backend

```bash
# Standard build
go build -o whatomate ./cmd/whatomate/

# Production build with optimizations
go build -ldflags="-s -w" -o whatomate ./cmd/whatomate/

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o whatomate ./cmd/whatomate/
```

### Embedded Frontend

The frontend is embedded at build time using Go's `embed` package:

```go
// internal/frontend/embed.go
//go:embed dist/*
var EmbeddedFS embed.FS
```

This means the binary is self-contained — no separate frontend server is needed.

## Docker Deployment

### Dockerfile

```dockerfile
# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist/ ./internal/frontend/dist/
RUN go build -ldflags="-s -w" -o whatomate ./cmd/whatomate/

# Stage 3: Runtime
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/whatomate .
COPY config.toml .
EXPOSE 8080
CMD ["./whatomate", "serve"]
```

### Docker Compose

```yaml
version: "3.8"

services:
  whatomate:
    build: .
    ports:
      - "8080:8080"
    environment:
      - WHATOMATE_DATABASE_HOST=postgres
      - WHATOMATE_DATABASE_PASSWORD=${DB_PASSWORD}
      - WHATOMATE_REDIS_HOST=redis
      - WHATOMATE_JWT_SECRET=${JWT_SECRET}
      - WHATOMATE_APP_ENCRYPTION_KEY=${ENCRYPTION_KEY}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: whatomate
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: whatomate
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U whatomate"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redisdata:/data
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

volumes:
  pgdata:
  redisdata:
```

### Running with Docker Compose

```bash
# Create environment file
cat > .env << EOF
DB_PASSWORD=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 64)
ENCRYPTION_KEY=$(openssl rand -base64 32 | head -c 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
EOF

# Start services
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f whatomate
```

## Environment Setup

### Production Checklist

- [ ] Set strong `encryption_key` (32 bytes, via env var)
- [ ] Set strong `jwt.secret` (32+ chars, via env var)
- [ ] Set strong database password
- [ ] Set strong Redis password
- [ ] Configure `allowed_origins` for CORS
- [ ] Set `environment = "production"` and `debug = false`
- [ ] Run database migrations: `whatomate -migrate`
- [ ] Run crypto migration if upgrading: `whatomate crypto-migrate`
- [ ] Configure SSL/TLS termination (reverse proxy)
- [ ] Set up health check monitoring
- [ ] Configure log rotation
- [ ] Set up database backups
- [ ] Set up Redis persistence
- [ ] Verify webhook endpoints are accessible
- [ ] Test WhatsApp provider connectivity

### Reverse Proxy (Nginx)

```nginx
server {
    listen 443 ssl http2;
    server_name whatomate.example.com;

    ssl_certificate /etc/ssl/certs/whatomate.crt;
    ssl_certificate_key /etc/ssl/private/whatomate.key;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400s;  # For WebSocket
    }
}
```

## Health and Readiness Endpoints

### Health Check

```bash
curl http://localhost:8080/health
```

Response:

```json
{
  "status": "ok",
  "service": "whatomate"
}
```

### Readiness Check

```bash
curl http://localhost:8080/ready
```

Response (ready):

```json
{
  "status": "ready"
}
```

Response (not ready):

```json
{
  "status": "not ready",
  "error": "database connection failed"
}
```

The readiness check verifies:
- Database connectivity (ping)
- Redis connectivity (ping)

Returns HTTP 500 if any dependency is unavailable.

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Command-Line Flags

| Flag | Description |
|------|-------------|
| `-config` | Path to config file |
| `-migrate` | Run database migrations and exit |
| `-crypto-migrate` | Run crypto migration (re-encrypt legacy data) |
| `-dry-run` | Dry-run mode (for migrations) |
| `-batch-size` | Batch size for crypto migration |
| `-include-enc2` | Include enc2 format in crypto migration |
| `-version` | Print version and exit |

## See Also

- [Configuration](configuration.md) — All configuration options
- [Monitoring](monitoring.md) — Health checks and monitoring
- [Security](security.md) — Production security checklist
- [Troubleshooting](troubleshooting.md) — Common deployment issues
