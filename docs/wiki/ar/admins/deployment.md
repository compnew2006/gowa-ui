---
title: النشر
rtl: true
lang: ar
---

<div dir="rtl">النشر</div>

Whatomate هو تطبيق بملف تنفيذي واحد مع واجهة أمامية مدمجة، مما يجعل النشر مباشراً.

## عملية البناء

### المتطلبات الأساسية

- Go 1.21+
- Node.js 18+ (لبناء الواجهة الأمامية)
- npm أو pnpm

### بناء الواجهة الأمامية

```bash
cd frontend
npm install
npm run build
```

يُنتج هذا ملفات ثابتة في `internal/frontend/dist/`، والتي يتم تضمينها في الملف التنفيذي لـ Go عبر `//go:embed`.

### بناء الخلفية

```bash
# البناء القياسي
go build -o whatomate ./cmd/whatomate/

# بناء الإنتاج مع التحسينات
go build -ldflags="-s -w" -o whatomate ./cmd/whatomate/

# التجميع المتقاطع لـ Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o whatomate ./cmd/whatomate/
```

### الواجهة الأمامية المدمجة

يتم تضمين الواجهة الأمامية عند البناء باستخدام حزمة `embed` في Go:

```go
// internal/frontend/embed.go
//go:embed dist/*
var EmbeddedFS embed.FS
```

هذا يعني أن الملف التنفيذي مكتفي ذاتياً — لا حاجة لخادم واجهة أمامية منفصل.

## نشر Docker

### Dockerfile

```dockerfile
# المرحلة 1: بناء الواجهة الأمامية
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# المرحلة 2: بناء الخلفية
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist/ ./internal/frontend/dist/
RUN go build -ldflags="-s -w" -o whatomate ./cmd/whatomate/

# المرحلة 3: وقت التشغيل
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

### التشغيل باستخدام Docker Compose

```bash
# إنشاء ملف البيئة
cat > .env << EOF
DB_PASSWORD=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 64)
ENCRYPTION_KEY=$(openssl rand -base64 32 | head -c 32)
REDIS_PASSWORD=$(openssl rand -base64 32)
EOF

# بدء الخدمات
docker compose up -d

# التحقق من الحالة
docker compose ps

# عرض السجلات
docker compose logs -f whatomate
```

## إعداد البيئة

### قائمة التحقق للإنتاج

- [ ] تعيين `encryption_key` قوي (32 بايت، عبر متغير بيئة)
- [ ] تعيين `jwt.secret` قوي (32+ حرفاً، عبر متغير بيئة)
- [ ] تعيين كلمة مرور قوية لقاعدة البيانات
- [ ] تعيين كلمة مرور قوية لـ Redis
- [ ] تكوين `allowed_origins` لـ CORS
- [ ] تعيين `environment = "production"` و`debug = false`
- [ ] تشغيل ترحيلات قاعدة البيانات: `whatomate -migrate`
- [ ] تشغيل ترحيل التشفير عند الترقية: `whatomate crypto-migrate`
- [ ] تكوين إنهاء SSL/TLS (وكيل عكسي)
- [ ] إعداد مراقبة فحص الصحة
- [ ] تكوين تدوير السجلات
- [ ] إعداد النسخ الاحتياطي لقاعدة البيانات
- [ ] إعداد استمرارية Redis
- [ ] التحقق من إمكانية الوصول لنقاط نهاية Webhook
- [ ] اختبار اتصال مزود WhatsApp

### الوكيل العكسي (Nginx)

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
        proxy_read_timeout 86400s;  # لـ WebSocket
    }
}
```

## نقاط نهاية الصحة والجاهزية

### فحص الصحة

```bash
curl http://localhost:8080/health
```

الاستجابة:

```json
{
  "status": "ok",
  "service": "whatomate"
}
```

### فحص الجاهزية

```bash
curl http://localhost:8080/ready
```

الاستجابة (جاهز):

```json
{
  "status": "ready"
}
```

الاستجابة (غير جاهز):

```json
{
  "status": "not ready",
  "error": "database connection failed"
}
```

يتحقق فحص الجاهزية من:
- اتصال قاعدة البيانات (ping)
- اتصال Redis (ping)

يعيد HTTP 500 إذا كان أي تابع غير متاح.

### مسبارات Kubernetes

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

## أعلام سطر الأوامر

| العلم | الوصف |
|------|-------------|
| `-config` | مسار ملف الإعدادات |
| `-migrate` | تشغيل ترحيلات قاعدة البيانات والخروج |
| `-crypto-migrate` | تشغيل ترحيل التشفير (إعادة تشفير البيانات القديمة) |
| `-dry-run` | وضع التشغيل التجريبي (للترحيلات) |
| `-batch-size` | حجم الدفعة لترحيل التشفير |
| `-include-enc2` | تضمين تنسيق enc2 في ترحيل التشفير |
| `-version` | طباعة الإصدار والخروج |

## انظر أيضاً

- [الإعدادات](configuration.md) — جميع خيارات الإعدادات
- [المراقبة](monitoring.md) — فحوصات الصحة والمراقبة
- [الأمان](security.md) — قائمة التحقق الأمنية للإنتاج
- [استكشاف الأخطاء](troubleshooting.md) — مشاكل النشر الشائعة
