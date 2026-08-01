# دليل التثبيت الشامل — Gowa-UI + GOWA

> دليل كامل (خطوة بخطوة) لتثبيت **Gowa-UI** على أي نظام تشغيل، وتنزيل وتشغيل خادم **GOWA** (خادم واتساب متعدد الأجهزة)، وربطهما معًا لعرض رسائل واتساب داخل Gowa-UI.
>
> جميع الأوامر والقيم مأخوذة من الكود المصدري (Go 1.25، Node 22، PostgreSQL 17، Redis 7) وليست تخمينًا.

---

## جدول المحتويات

1. [نظرة معمارية: كيف يعمل النظام كله](#1-نظرة-معمارية-كيف-يعمل-النظام-كله)
2. [المتطلبات الأساسية (Dependencies)](#2-المتطلبات-الأساسية-dependencies)
3. [الطريقة السريعة: Docker (كل شيء بأمر واحد)](#3-الطريقة-السريعة-docker-كل-شيء-بأمر-واحد)
4. [التثبيت اليدوي الكامل (Windows / macOS / Linux)](#4-التثبيت-اليدوي-الكامل-windows--macos--linux)
5. [تنزيل وتشغيل GOWA (خادم واتساب)](#5-تنزيل-وتشغيل-gowa-خادم-واتساب)
6. [ربط GOWA مع Gowa-UI وعرض الرسائل](#6-ربط-gowa-مع-Gowa-UI-وعرض-الرسائل)
7. [ملف الإعدادات الكامل `config.toml`](#7-ملف-الإعدادات-الكامل-configtoml)
8. [حل المشكلات الشائعة](#8-حل-المشكلات-الشائعة)

---

## 1. نظرة معمارية: كيف يعمل النظام كله

```
┌─────────────┐        REST API + Basic Auth + X-Device-Id        ┌──────────────┐
│  Gowa-UI  │ ◄──────────────────────────────────────────────► │     GOWA     │
│  (Go + Vue) │                                                    │  (خادم واتساب)│
│  :8080/:3000│  ◄── Webhook (POST /api/gowa/webhook/{device}) ── │   :3000      │
└─────────────┘        HMAC-SHA256 signature + replay guard        └──────┬───────┘
     │                                                              ┌──────┴───────┐
     │ PostgreSQL + Redis                                          │  واتساب على  │
     └──────────────────────────────────────────────────────────► │  هاتفك (QR)  │
                                                                  └──────────────┘
```

**النقاط الأساسية:**

- **Gowa-UI** هو تطبيق الويب (Go backend + Vue frontend) الذي تتعامل معه — لإدارة جهات الاتصال، الحملات، الدردشة، التقارير.
- **GOWA** هو **خادم منفصل** (مشروع خارجي باسم *WhatsApp API MultiDevice* v9.0.0) يعمل كجسر بين Gowa-UI وخادم واتساب الرسمي (واتساب متعدد الأجهزة). **لا يُشحن داخل Gowa-UI** — يجب تنزيله وتشغيله بشكل مستقل.
- **الاتصال** ثنائي الاتجاه:
  - Gowa-UI → GOWA: عبر **REST API** (إرسال رسالة، طلب QR، مزامنة السجل) باستخدام Basic Auth ورأس `X-Device-Id`.
  - GOWA → Gowa-UI: عبر **Webhook** — عندما تصل رسالة إلى واتساب، يرسلها GOWA إلى `POST /api/gowa/webhook/{device_id}` مع توقيع HMAC للتحقق.

---

## 2. المتطلبات الأساسية (Dependencies)

### 2.1 نسخ البرامج (مهمة — مطابقة للكود المصدري)

| المكوّن | النسخة المطلوبة | السبب |
|---|---|---|
| **Go** | **1.25.0+** | لغة الـ backend (`go.mod` يحدد `go 1.25.0`) |
| **Node.js** | **22+** | بناء الـ frontend (Dockerfile يحدد `node:22-alpine`) |
| **npm** | 10+ (يأتي مع Node 22) | إدارة حزم الـ frontend |
| **PostgreSQL** | **17+** (14 كحد أدنى) | قاعدة البيانات الرئيسية (`postgres:17-alpine` في docker) |
| **Redis** | **7+** (6 كحد أدنى) | الكاش، الطوابير، حدود المعدل، الجلسات (`redis:7-alpine` في docker) |
| **Git** | أي نسخة حديثة | استنساخ المستودع |
| **Docker** (اختياري) | Docker + Compose | الطريقة السريعة (القسم 3) |
| **Make** | GNU Make | تشغيل أوامر `make` (موجود افتراضيًا على macOS/Linux، ثبّته على Windows عبر `choco install make`) |

### 2.2 حزم Go الرئيسية (تُنزّل تلقائيًا عبر `go mod download`)

هذه تُحمل عند تشغيل `make run-migrate` — لا تحتاج لتثبيتها يدويًا، لكنها مذكورة للشفافية:

| الحزمة | الغرض |
|---|---|
| `github.com/zerodha/fastglue` + `github.com/valyala/fasthttp` | خادم الويب (أسرع من net/http) |
| `gorm.io/gorm` + `gorm.io/driver/postgres` | ORM وطبقة قاعدة البيانات |
| `github.com/redis/go-redis/v9` | عميل Redis |
| `github.com/golang-jwt/jwt/v5` | مصادقة JWT (cookies) |
| `github.com/google/uuid` | مُولّد المعرّفات |
| `github.com/knadh/koanf` | قراءة إعدادات TOML + متغيرات البيئة |
| `golang.org/x/crypto` | تشفير AES-256 لأسرار GOWA |
| `github.com/dop251/goja` | محرّك JavaScript (للأفعال المخصّصة) |
| `github.com/expr-lang/expr` | محرّك القواعد (فلترة الحملات) |
| `github.com/fasthttp/websocket` | WebSocket للوقت الحقيقي |

### 2.3 حزم الـ Frontend الرئيسية (تُنزّل عبر `npm install`)

| الحزمة | الغرض |
|---|---|
| `vue@^3.4` | إطار الواجهة |
| `vue-router@^4.2` | التوجيه (SPA) |
| `pinia@^2.1` | إدارة الحالة |
| `vite@^7.3` | أداة البناء وخادم التطوير |
| `axios@^1.16` | عميل HTTP |
| `reka-ui` | مكوّنات الواجهة (shadcn-style) |
| `chart.js` + `vue-chartjs` | الرسوم البيانية |
| `grid-layout-plus` | لوحة القيادة القابلة للسحب |
| `vee-validate` | التحقق من النماذج |

### 2.4 تثبيت المتطلبات حسب النظام

#### macOS (باستخدام Homebrew)
```bash
# تثبيت Homebrew إن لم يكن موجودًا
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

brew install go node@22 postgresql@17 redis git make
brew services start postgresql@17
brew services start redis
```

#### Ubuntu / Debian
```bash
# Go 1.25 (المستودعات قد تحتوي على نسخة أقدم، نثبّتها من الموقع الرسمي)
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc

# Node 22 (عبر NodeSource)
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt install -y nodejs postgresql-17 redis-server git build-essential
sudo systemctl enable --now postgresql redis-server
```

#### Windows (باستخدام Chocolatey أو Scoop)
```powershell
# عبر Chocolatey (شغّل PowerShell كمسؤول)
Set-ExecutionPolicy Bypass -Scope Process -Force
iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
choco install golang nodejs-lts postgresql17 redis-64 git make

# أو عبر Scoop:
# scoop install go nodejs postgresql redis git make
```
> على Windows، شغّل **PostgreSQL** و**Redis** كخدمات (تُثبّت تلقائيًا عبر Chocolatey). استخدم **WSL2** للحصول على تجربة أقرب إلى Linux إن أمكن.

---

## 3. الطريقة السريعة: Docker (كل شيء بأمر واحد)

الأنسب للبدء السريع — تشغّل PostgreSQL + Redis + Gowa-UI معًا.

```bash
# 1. استنساخ المستودع
git clone https://github.com/shridarpatil/Gowa-UI.git
cd Gowa-UI

# 2. نسخ ملفات الإعدادات وتعديلها
cp config.example.toml docker/config.toml
cp docker/.env.example docker/.env
# عدّل docker/.env لضبط كلمة مرور PostgreSQL والمنطقة الزمنية (TZ)

# 3. تشغيل كل الخدمات
make docker-up

# 4. افتح المتصفح
open http://localhost:8080   # macOS
# أو: start http://localhost:8080   (Windows)
# أو: xdg-open http://localhost:8080   (Linux)
```

**بيانات الدخول الافتراضية:** `admin@admin.com` / `admin`

> ⚠️ **غيّر كلمة مرور المدير فورًا** بعد أول تسجيل دخول من صفحة *Profile*.

> 📝 **ملاحظة:** Docker يشغّل Gowa-UI + قاعدة البيانات + Redis فقط. **GOWA خادم منفصل** يجب تثبيته يدويًا (القسم 5) ثم ربطه (القسم 6).

---

## 4. التثبيت اليدوي الكامل (Windows / macOS / Linux)

استخدم هذه الطريقة إذا أردت تحكمًا كاملًا أو كنت تطوّر الكود.

### الخطوة 1: استنساخ المستودع وإعداد الإعدادات
```bash
git clone https://github.com/shridarpatil/Gowa-UI.git
cd Gowa-UI
cp config.example.toml config.toml
# عدّل config.toml (القسم 7 يشرح كل الحقول)
```

### الخطوة 2: إعداد PostgreSQL
```bash
# إنشاء قاعدة البيانات
createdb Gowa-UI

# أو يدويًا عبر psql (مع مستخدم مخصص):
psql -c "CREATE USER Gowa-UI WITH PASSWORD 'Gowa-UI123';"
psql -c "CREATE DATABASE Gowa-UI OWNER Gowa-UI;"
```
ثم حدّث `config.toml`:
```toml
[database]
host = "localhost"   # استخدم "db" مع Docker
port = 5432
user = "Gowa-UI"
password = "Gowa-UI123"
name = "Gowa-UI"
ssl_mode = "disable"  # "require" في الإنتاج إذا كان PostgreSQL يفرض TLS
```

### الخطوة 3: إعداد Redis
```bash
# macOS:
brew install redis && brew services start redis

# Ubuntu/Debian:
sudo apt install redis-server && sudo systemctl start redis-server

# Windows (Chocolatey): يُثبّت كخدمة تلقائيًا

# أو عبر Docker (أي نظام):
docker run -d --name redis -p 6379:6379 redis:7-alpine
```
تحقّق من عمله: `redis-cli ping` → يجب أن يردّ `PONG`.

### الخطوة 4: تشغيل الـ Backend
```bash
# تنزيل حزم Go
go mod download

# تشغيل التهيئة (Migrations) + الخادم معًا
make run-migrate
```
سيعمل الـ backend على `http://localhost:8080`.

### الخطوة 5: تشغيل الـ Frontend (خادم التطوير)
افتح **طرفية جديدة**:
```bash
cd frontend
npm install      # تثبيت حزم Vue
npm run dev      # خادم التطوير على :3000
```
خادم Vite يوجّه (`proxy`) الطلبات `/api`, `/health`, `/ready`, `/ws` إلى `:8080`.

### الخطوة 6: الوصول للتطبيق
| الخدمة | العنوان |
|---|---|
| الواجهة (Frontend) | http://localhost:3000 |
| الـ API | http://localhost:8080 |

**الدخول:** `admin@admin.com` / `admin`

### بناء نسخة الإنتاج (Binary واحد)
```bash
make build-prod
./Gowa-UI server -migrate
```
هذا يُنتج binary واحدًا (الـ frontend مضمّن بداخله) — لا حاجة لخادم frontend منفصل. الوصول على `http://localhost:8080`.

---

## 5. تنزيل وتشغيل GOWA (خادم واتساب)

GOWA هو **خادم منفصل** (ليس جزءًا من binary الـ Gowa-UI). يتصل بخادم واتساب الرسمي عبر بروتوكول الأجهزة المتعددة، ويوفّر REST API + Webhook.

### 5.1 الخيارات المتاحة لتشغيل GOWA

| الطريقة | الأنسب لـ |
|---|---|
| **Docker** (موصى به) | كل الأنظمة — الأسرع |
| **Binary جاهز** | الإنتاج على خادم بدون Docker |
| **البناء من المصدر (Go)** | التطوير والمساهمة |

### 5.2 الطريقة الموصى بها: Docker

```bash
# سحب وتشغيل صورة GOWA الرسمية
docker run -d \
  --name gowa \
  -p 3000:3000 \
  -v gowa_data:/app/data \
  -e APP_BASIC_AUTH_USER="gowa_admin" \
  -e APP_BASIC_AUTH_PASSWORD="gowa_strong_password" \
  -e APP_WEBHOOK_URL="http://host.docker.internal:8080/api/gowa/webhook" \
  -e APP_WEBHOOK_SECRET="your-hmac-secret-here" \
  -e APP_UI_ENABLED=true \
  --restart unless-stopped \
  gowa/gowa:latest
```

> 🔑 **غيّر جميع القيم الحساسة** (`gowa_admin`, `gowa_strong_password`, `your-hmac-secret-here`) بقيم قوية وفريدة.

**شرح متغيرات البيئة:**

| المتغيّر | الواجب |
|---|---|
| `APP_BASIC_AUTH_USER` / `APP_BASIC_AUTH_PASSWORD` | بيانات Basic Auth — ستستخدمها لربط Gowa-UI (القسم 6). **نفس** القيم التي ستدخلها في واجهة Gowa-UI. |
| `APP_WEBHOOK_URL` | عنوان webhook الخاص بـ Gowa-UI. على نفس الجهاز: `http://host.docker.internal:8080/api/gowa/webhook` (macOS/Windows). على Linux استخدم IP الـ host أو الشبكة المشتركة. |
| `APP_WEBHOOK_SECRET` | سرّ HMAC المشترك — **يجب أن يطابق** ما ستضبطه في Gowa-UI لكل جهاز. يُستخدم لتوقيع الـ webhook (`X-Hub-Signature-256`). |
| `APP_UI_ENABLED` | `true` لعرض لوحة تحكم GOWA على المتصفّح. |
| `-v gowa_data:/app/data` | حجم دائم لحفظ جلسات الأجهزة (حتى لا تحتاج لمسح QR عند كل إعادة تشغيل). |

### 5.3 إذا كان Gowa-UI يعمل بـ Docker أيضًا

ضع GOWA على **نفس الشبكة** حتى يتواصلوا بأسماء الخدمات:
```bash
# أنشئ شبكة مشتركة
docker network create Gowa-UI-net

# وصِل GOWA بها
docker network connect Gowa-UI-net gowa

# وصِل حاوية Gowa-UI بها أيضًا
docker network connect Gowa-UI-net Gowa-UI-app

# ثم APP_WEBHOOK_URL تصبح:
#   http://Gowa-UI-app:8080/api/gowa/webhook
# وماومات سيتصل بـ GOWA عبر:  http://gowa:3000
```

### 5.4 التحقق من أن GOWA يعمل
```bash
curl -u gowa_admin:gowa_strong_password http://localhost:3000/health
```
يجب أن يرجع استجابة `200 OK`. افتح `http://localhost:3000` في المتصفح لرؤية لوحة تحكم GOWA.

---

## 6. ربط GOWA مع Gowa-UI وعرض الرسائل

بعد أن يعمل Gowa-UI (القسم 3 أو 4) و GOWA (القسم 5)، اربطهما عبر واجهة Gowa-UI.

### الخطوة 1: تسجيل دخول المدير إلى Gowa-UI
افتح `http://localhost:3000` (أو `:8080` في الإنتاج) وسجّل الدخول بـ `admin@admin.com` / `admin`.

### الخطوة 2: إضافة خادم GOWA
1. اذهب إلى **Settings → GOWA Servers** (`/settings/gowa-servers`).
2. اضغط **Create Server**.
3. املأ الحقول:

| الحقل | القيمة | مثال |
|---|---|---|
| **Name** | اسم وصفي للخادم | `خادم واتساب الرئيسي` |
| **Base URL** | عنوان REST API الخاص بـ GOWA | `http://localhost:3000` (أو `http://gowa:3000` على شبكة Docker مشتركة) |
| **Username** | نفس `APP_BASIC_AUTH_USER` | `gowa_admin` |
| **Password** | نفس `APP_BASIC_AUTH_PASSWORD` | `gowa_strong_password` |
| **Webhook URL** (اختياري) | عنوان webhook Gowa-UI | `http://your-host:8080/api/gowa/webhook` |

4. اضغط **Save**. سيحاول Gowa-UI **اختبار الاتصال** (probe) بالخادم قبل الحفظ — إن فشل، سيرفض ويعرض الخطأ.

> 🔒 بيانات الاعتماد (Username/Password) تُخزّن **مشفّرة** في قاعدة البيانات (AES-256 باستخدام `app.encryption_key` من `config.toml`).

### الخطوة 3: ربط جهاز واتساب (QR Code)
1. من صفحة خادم GOWA الذي أنشأته، اضغط على **Devices**.
2. اضغط **Generate QR** أو **Pair**.
3. ستظهر صورة QR — افتح **واتساب على هاتفك** ← *الإعدادات* ← *الأجهزة المرتبطة* ← *ربط جهاز* ← امسح الـ QR.
4. بمجرد المسح، يتصل GOWA بخادم واتساب ويصبح الجهاز **متصلًا** (online).

> 💡 **بديل:** إن لم تتمكن من مسح QR، اضغط **Pair Code** للحصول على رمز رقمي تُدخله في هاتفك.

### الخطوة 4: ضبط Webhook Secret لكل جهاز
لكل جهاز، هناك **سرّ webhook** منفصل يحقّق Gowa-UI منه التوقيع:
1. في Gowa-UI: **Settings → Accounts** → اختر الحساب المرتبط بالجهاز.
2. اضبط **Webhook Secret** بنفس قيمة `APP_WEBHOOK_SECRET` التي ضبطتها في GOWA (القسم 5.2).
3. احفظ. الآن كل webhook وارد من GOWA لهذا الجهاز سيتم التحقق منه عبر HMAC-SHA256 (ورفضه إن فشل التحقق أو كان أقدم من 5 دقائق — حماية من إعادة التشغيل).

### الخطوة 5: عرض الرسائل في Gowa-UI
1. اذهب إلى **Chat** في الشريط الجانبي (`/chat`).
2. ستظهر جهات الاتصال ومحادثاتها في القائمة.
3. اضغط على أي محادثة — ستظهر **سجل الرسائل** (الواردة والصادرة، مع المُرسِل، المحتوى، والطابع الزمني).
4. الرسائل الجديدة تظهر **فورًا** عبر WebSocket (بدون إعادة تحميل).

### تدفق رسالة واردة (كيف تصل من الهاتف إلى الشاشة)
```
هاتف المرسِل
    ↓ (شبكة واتساب)
خادم واتساب الرسمي
    ↓ (Multi-Device protocol)
GOWA (يستقبل الرسالة)
    ↓ POST /api/gowa/webhook/{device_id}
    ↓ الهيدر: X-Hub-Signature-256: sha256=<hmac>
    ↓ الجسم: {event, device_id, message, ...}
Gowa-UI (يتحقق HMAC + الـ replay guard)
    ↓ يخزّن في PostgreSQL + ينشر حدث WebSocket
المتصفح (يستقبل عبر WebSocket ويعرض الرسالة) ✅
```

---

## 7. ملف الإعدادات الكامل `config.toml`

هذا هو الهيكل الكامل (من `config.example.toml`). القيم الافتراضية موضوعة للتطوير المحلي:

```toml
[app]
name = "Gowa-UI"
environment = "development"   # development | staging | production
debug = true
encryption_key = ""           # مفتاح AES-256 (32+ حرف) — مطلوب في الإنتاج لتشفير أسرار GOWA

[server]
host = "0.0.0.0"
port = 8080
read_timeout = 30
write_timeout = 30
base_path = ""                # "/subpath" إن كنت خلف nginx proxy_pass
allowed_origins = ""          # قائمة CORS مفصولة بفواصل (فارغ = السماح للجميع في التطوير فقط)

[database]
host = "localhost"            # "db" مع Docker
port = 5432
user = "Gowa-UI"
password = "Gowa-UI"
name = "Gowa-UI"
ssl_mode = "disable"          # "require" في الإنتاج
max_open_conns = 25
max_idle_conns = 5
conn_max_lifetime = 300

[redis]
host = "localhost"            # "redis" مع Docker
port = 6379
username = ""                 # مستخدم Redis ACL (Redis 6+)، فارغ = المستخدم الافتراضي
password = ""
db = 0
tls = false                   # true لـ Upstash / Redis Cloud

[jwt]
secret = "your-super-secret-jwt-key-change-in-production"   # 32+ حرف في الإنتاج
access_expiry_mins = 15
refresh_expiry_days = 1

[storage]
type = "local"
local_path = "./uploads"

[cookie]
domain = ""                   # ".example.com" لمشاركة الكوكيز عبر نطاقات فرعية
secure = false                # يُضبط true تلقائيًا عند environment=production

[rate_limit]
enabled = false               # true لتفعيل حدود المعدل
login_max_attempts = 10
register_max_attempts = 10
refresh_max_attempts = 30
sso_max_attempts = 10
window_seconds = 60
trust_proxy = false           # true خلف reverse proxy موثوق (nginx, Cloudflare)
api_max_requests = 200
api_window_seconds = 60

[default_admin]               # يُستخدم فقط عند الإعداد الأولي (لا مستخدمين موجودين)
email = "admin@admin.com"
password = "admin"
full_name = "Admin"

# === إعدادات GOWA (اختيارية — يمكن ضبطها بالكامل من واجهة Gowa-UI بدلًا من هنا) ===
[gowa]
base_url = ""                 # عنوان GOWA الافتراضي، مثال: "http://gowa:3000"
webhook_path = "/api/gowa/webhook"
username = ""                 # Basic Auth username
password = ""                 # Basic Auth password

# يمكن تعريف عدة خوادم GOWA عبر تكرار [[gowa_instances]]
# [[gowa_instances]]
# name = "الخادم الرئيسي"
# base_url = "http://gowa:3000"
# username = "gowa_admin"
# password = "gowa_strong_password"
# webhook_url = "http://Gowa-UI:8080/api/gowa/webhook"
```

> 💡 **الخلاصة:** إعدادات `[gowa]` في الملف **اختيارية** — الطريقة الموصى بها هي إضافة الخوادم من واجهة Gowa-UI (القسم 6) لأنها تتيح إدارة عدة خوادم وتشفّر البيانات. استخدم الملف فقط للإعداد التلقائي (bootstrap) في الإنتاج.

---

## 8. حل المشكلات الشائعة

| المشكلة | السبب والحل |
|---|---|
| `connection refused` (قاعدة البيانات) | تأكد أن PostgreSQL يعمل: `pg_isready`. مع Docker تأكد أن `host = "db"`. |
| `connection refused` (Redis) | تأكد أن Redis يعمل: `redis-cli ping` يجب أن يردّ `PONG`. مع Docker استخدم `host = "redis"`. |
| `database does not exist` | أنشئها: `createdb Gowa-UI`. |
| `permission denied` (قاعدة البيانات) | تحقّق من `user`/`password` في `config.toml`. |
| `port already in use` | غيّر `port` في `[server]` أو أوقف الخدمة المتضاربة. |
| الواجهة لا تصل إلى الـ API | تأكد أن الـ backend يعمل على `:8080`. خادم Vite يوجّه `/api` تلقائيًا. |
| **فشل إنشاء خادم GOWA في Gowa-UI** | الـ probe فشل — تحقّق أن GOWA يعمل (`curl -u user:pass http://localhost:3000/health`)، وأن `Base URL` و`Username`/`Password` صحيحة ومطابقة لبيانات GOWA. |
| **الـ QR لا يظهر / انتهت صلاحيته** | امسح الـ QR سريعًا (ينتهي خلال ~60 ثانية). أعِد توليده. تأكد أن هاتفك متصل بالإنترنت. |
| **الرسائل لا تصل إلى Gowa-UI** | (1) تحقّق أن الجهاز **متصل** في صفحة GOWA Devices. (2) تحقّق أن `APP_WEBHOOK_URL` في GOWA يشير لعنوان Gowa-UI الصحيح القابل للوصول. (3) تحقّق أن **Webhook Secret** في Gowa-UI يطابق `APP_WEBHOOK_SECRET` في GOWA (توقيع HMAC). (4) راجع سجلات Gowa-UI بحثًا عن `GOWA webhook verification failed`. |
| **`Webhook verification failed` في السجلات** | السرّ غير متطابق، أو الـ signature مفقود. تأكد من تطابق السرّ بالضبط. كل مسارات الرفض ترجع نفس الرسالة (لأمان: لا يمكن للمهاجم تمييز جهاز غير مُعد عن سرّ خاطئ). |
| **GOWA لا يصل إلى Gowa-UI (Docker)** | على شبكة Docker مشتركة، استخدم أسماء الحاويات (`http://Gowa-UI-app:8080`) لا `localhost`. `host.docker.internal` يعمل على macOS/Windows فقط. |
| **فقدان الجلسة بعد إعادة تشغيل GOWA** | تأكد من استخدام حجم دائم (`-v gowa_data:/app/data`) كي تُحفظ جلسات الأجهزة ولا تحتاج لمسح QR مجددًا. |

---

## ملخص سريع (للبدء فورًا)

```bash
# 1. Gowa-UI عبر Docker
git clone https://github.com/shridarpatil/Gowa-UI.git && cd Gowa-UI
cp config.example.toml docker/config.toml && cp docker/.env.example docker/.env
make docker-up

# 2. GOWA عبر Docker
docker run -d --name gowa -p 3000:3000 -v gowa_data:/app/data \
  -e APP_BASIC_AUTH_USER=gowa_admin \
  -e APP_BASIC_AUTH_PASSWORD=gowa_strong_password \
  -e APP_WEBHOOK_URL=http://host.docker.internal:8080/api/gowa/webhook \
  -e APP_WEBHOOK_SECRET=your-hmac-secret \
  -e APP_UI_ENABLED=true gowa/gowa:latest

# 3. في المتصفح: http://localhost:8080 → سجّل الدخول → Settings → GOWA Servers → Create
#    أدخل: http://localhost:3000 / gowa_admin / gowa_strong_password
# 4. امسح QR بهاتفك → ستظهر الرسائل في Chat ✅
```

---

*هذا الدليل مبني على الكود المصدري لـ Gowa-UI (`go.mod`, `Dockerfile`, `config.example.toml`, `internal/config/config.go`, `internal/handlers/gowa_*.go`, `pkg/gowa/client.go`) ومواصفات GOWA OpenAPI (`docs/GOWA openapi.yaml`).*
