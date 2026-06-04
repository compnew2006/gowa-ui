# تقرير التدقيق الأمني الشامل — منصة Whatomate

**التاريخ:** يونيو 2026  
**الإصدار:** 1.0  
**النطاق:** البنية التحتية الخلفية (Go backend) — المصادقة، التفويض، حماية البيانات، التحقق من المدخلات، أمان Webhook، WebSocket، رفع الملفات، وإدارة المفاتيح.

---

## ملخص تنفيذي

تم فحص 20+ ملفًا أساسيًا في البنية الأمنية للمنصة. المنصة تُظهر نضجًا أمنيًا جيدًا في عدة مجالات (تشفير AES-256-GCM مع Argon2id، تحقق HMAC-SHA256 للويبهوكات، حماية SSRF، rate limiting مغلق عند الفشل). ومع ذلك، تم رصد **17 نتيجة** تتراوح بين حرجة ومنخفضة تستدعي المعالجة.

---

## جدول النتائج

| # | الشدة | العنوان | الملف |
|---|-------|---------|-------|
| 1 | عالية | سياسة كلمات المرور لا تتطلب أحرفًا خاصة | `password_policy.go` |
| 2 | عالية | Webhook URL يسمح بمخطط HTTP (غير مشفر) | `webhooks.go` |
| 3 | متوسطة | RBAC على مستوى المعالج فقط — لا تفرض middleware مركزية | `main.go:1382-1394` |
| 4 | متوسطة | تحقق prefix بسيط من المسار قد يتم تجاوزه | `main.go:1342-1348` |
| 5 | متوسطة | Webhook secret يُخزن نصًا عاديًا في قاعدة البيانات | `webhooks.go:235` |
| 6 | متوسطة | سر SSO (client_secret) يُمرر عبر JSON بدون تشفير إضافي | `sso_handlers.go` |
| 7 | متوسطة | WebSocket مصادقة رسالة JWT مع مهلة 5 ثوانٍ فقط | `websocket.go` |
| 8 | منخفضة | Webhook headers مخصصة بدون فلترة | `webhooks.go:291-296` |
| 9 | منخفضة | لا يوجد حد أقصى لعدد API Keys لكل منظمة | `apikeys.go` |
| 10 | منخفضة | Webhook URL لا يتحقق من المنفذ | `webhooks.go` |
| 11 | إيجابي | تشفير AES-256-GCM مع Argon2id KDF | `crypto.go` |
| 12 | إيجابي | حماية SSRF مزدوجة (تحقق URL + Runtime Dialer) | `webhooks.go`, `sso_security.go` |
| 13 | إيجابي | Rate limiter مغلق عند فشل Redis | `ratelimit.go` |
| 14 | إيجابي | CSRF double-submit cookie مع تخطي ذكي للـ header auth | `csrf.go` |
| 15 | إيجابي | رفض symlink + directory traversal في خدمة الملفات | `media.go:466-495` |
| 16 | إيجابي | API key hash بـ bcrypt | `apikeys.go:145` |
| 17 | إيجابي | تحقق إنتاجي من تكوين أمني | `security_validation.go` |

---

## تفاصيل النتائج

### 🔴 1. سياسة كلمات المرور لا تتطلب أحرفًا خاصة — **عالية**

**الملف:** `internal/handlers/password_policy.go`

**الوصف:** سياسة كلمات المرور تتطلب 12–128 حرفًا مع أحرف كبيرة وصغيرة وأرقام، لكنها **لا تتطلب أحرفًا خاصة**. هذا يقلل من تعقيد كلمات المرور بشكل ملحوظ.

**الكود:**
```go
func ValidatePassword(password string) error {
    if len(password) < 12 || len(password) > 128 {
        return fmt.Errorf("password must be between 12 and 128 characters")
    }
    var hasUpper, hasLower, hasDigit bool
    for _, ch := range password {
        switch {
        case unicode.IsUpper(ch):
            hasUpper = true
        case unicode.IsLower(ch):
            hasLower = true
        case unicode.IsDigit(ch):
            hasDigit = true
        }
    }
    if !hasUpper || !hasLower || !hasDigit {
        return fmt.Errorf("password must contain uppercase, lowercase, and digit")
    }
    return nil
}
```

**التوصية:** إضافة شرط الأحرف الخاصة:
```go
var hasSpecial bool
for _, ch := range password {
    if unicode.IsSymbol(ch) || unicode.IsPunct(ch) {
        hasSpecial = true
    }
}
if !hasSpecial {
    return fmt.Errorf("password must contain at least one special character")
}
```

---

### 🔴 2. Webhook URL يسمح بمخطط HTTP — **عالية**

**الملف:** `internal/handlers/webhooks.go:27-29`

**الوصف:** دالة `validateWebhookURL` تقبل مخطط `http://` بالإضافة إلى `https://`. هذا يعني أن بيانات الويبهوك (التي قد تحتوي على رسائل العملاء) ستُرسل غير مشفرة عبر الشبكة.

**الكود:**
```go
if u.Scheme != "https" && u.Scheme != "http" {
    return fmt.Errorf("URL scheme must be http or https")
}
```

**التوصية:** في بيئة الإنتاج، ارفض مخطط HTTP:
```go
if u.Scheme != "https" {
    return fmt.Errorf("URL scheme must be https in production")
}
```

---

### 🟡 3. RBAC على مستوى المعالج فقط — **متوسطة**

**الملف:** `cmd/whatomate/main.go:1382-1394`

**الوصف:** middleware الرول (RBAC) في `main.go` لا يفعل شيئًا فعليًا — التعليق يقول "Route-level permission checks are now handled at the handler level". هذا يعني أن الأذونات تعتمد كليًا على `requirePermission` داخل كل معالج. إذا نسي مطور إضافة هذا الاستدعاء، ستكون النقطة غير محمية.

**الكود:**
```go
g.Before(func(r *fastglue.Request) *fastglue.Request {
    method := string(r.RequestCtx.Method())
    if method == "OPTIONS" {
        return r
    }
    // Route-level permission checks are now handled at the handler level
    // using the granular permission system (HasPermission checks)
    return r
})
```

**التوصية:** ضع خريطة route→permission مركزية كطبقة دفاع ثانية، أو أنشئ أداة تحليل ثابتة (lint) تتحقق أن كل معالج يستدعي `requirePermission`.

---

### 🟡 4. تحقق prefix بسيط من المسار — **متوسطة**

**الملف:** `cmd/whatomate/main.go:1342-1348`

**الوصف:** المسارات العامة (public) تُحدد بفحص prefix النصي: `path[:13] == "/api/auth/sso"` و `path[:28] == "/api/custom-actions/redirect"`. هذا عرضة للأخطاء إذا أُضيفت مسارات مشابهة، أو إذا تم تمرير مسار أقصر من الطول المحدد (التحقق `len(path) >= 13` يمنع panic لكن التطابق الدقيق قد يفشل).

**الكود:**
```go
if len(path) >= 13 && path[:13] == "/api/auth/sso" {
    return r
}
if len(path) >= 28 && path[:28] == "/api/custom-actions/redirect" {
    return r
}
```

**التوصية:** استخدم `strings.HasPrefix(path, "/api/auth/sso")` بدلاً من التحقق اليدوي، أو أنشئ set من المسارات العامة المصرح بها.

---

### 🟡 5. Webhook secret يُخزن نصًا عاديًا — **متوسطة**

**الملف:** `internal/handlers/webhooks.go:235`

**الوصف:** `webhook.Secret` يُخزن مباشرة في قاعدة البيانات بدون تشفير. إذا تم اختراق قاعدة البيانات، يمكن للمهاجم استخدام الأسرار لتزوير طلبات ويبهوك.

**الكود:**
```go
webhook := models.Webhook{
    OrganizationID: orgID,
    Secret:         secret, // stored as plaintext
}
```

**التوصية:** استخدم `internal/crypto` لتشفير السر قبل التخزين:
```go
encrypted, _ := crypto.Encrypt(secret)
webhook.Secret = encrypted
```

---

### 🟡 6. سر SSO يُمرر عبر JSON — **متوسطة**

**الملف:** `internal/handlers/sso_handlers.go`

**الوصف:** SSO `client_secret` يُستقبل ويعاد كجزء من JSON response في تحديث إعدادات SSO. يجب التأكد من تشفيره في قاعدة البيانات وأن API لا يعيد القرة النصية في GET.

**التوصية:** 
1. تشفير client_secret قبل التخزين باستخدام `crypto.Encrypt()`
2. إرجاع `has_secret: true` بدلاً من القيمة في GET requests
3. قبول السر فقط في PUT/POST وعدم إرجاعه أبدًا

---

### 🟡 7. WebSocket مصادقة مع مهلة 5 ثوانٍ — **متوسطة**

**الملف:** `internal/handlers/websocket.go`

**الوصف:** اتصال WebSocket يُقبل بدون مصادقة مبدئية، ثم يجب أن يرسل العميل JWT خلال 5 ثوانٍ. خلال هذه الفترة، يكون الاتصال مفتوحًا ولكن لا يمكنه إرسال/استقبال رسائل.

**التوصية:** 
1. قلل المهلة إلى 3 ثوانٍ
2. أضف rate limiting على محاولات اتصال WebSocket لكل IP
3. سجّل أي اتصال يفشل في المصادقة

---

### 🟢 8. Webhook headers مخصصة بدون فلترة — **منخفضة**

**الملف:** `internal/handlers/webhooks.go:291-296`

**الوصف:** المستخدم يمكنه تعيين أي headers مخصصة تُرسل مع طلبات الويبهوك الصادرة، بما في ذلك `Host`, `Authorization`, أو headers حساسة أخرى.

**التوصية:** أنشئ deny-list من headers المحجوبة:
```go
var blockedHeaders = map[string]bool{
    "host": true, "authorization": true, "cookie": true,
}
```

---

### 🟢 9. لا حد أقصى لـ API Keys لكل منظمة — **منخفضة**

**الملف:** `internal/handlers/apikeys.go`

**الوصف:** يمكن لأي منظمة إنشاء عدد غير محدود من مفاتيح API. هذا قد يُستخدم لاستنزاف موارد bcrypt عند التحقق (كل طلب API key غير صالح يتطلب bcrypt compare).

**التوصية:** أضف حدًا أقصى (مثلاً 10 مفاتيح لكل منظمة):
```go
var count int64
requestDB.Model(&models.APIKey{}).Where("organization_id = ?", orgID).Count(&count)
if count >= 10 {
    return r.SendErrorEnvelope(fasthttp.StatusConflict, "Maximum API keys reached", nil, "")
}
```

---

### 🟢 10. Webhook URL لا يتحقق من المنفذ — **منخفضة**

**الملف:** `internal/handlers/webhooks.go`

**الوصف:** `validateWebhookURL` لا يتحقق من أن المنفذ ليس منفذًا داخليًا (مثل `:6379` لـ Redis أو `:5432` لـ PostgreSQL). رغم أن `SSRFSafeDialer` يمنع الاتصال بعناوين خاصة، فإن التحقق المسبق يوفر طبقة إضافية.

**التوصية:** أضف تحققًا من المنفذ:
```go
port := u.Port()
if port != "" {
    p, _ := strconv.Atoi(port)
    if p < 1024 || p > 65535 {
        return fmt.Errorf("port must be between 1024 and 65535")
    }
}
```

---

## نقاط القوة الأمنية ✅

### 11. تشفير AES-256-GCM مع Argon2id KDF

**الملف:** `internal/crypto/crypto.go`

تشفير بيانات حساسة بـ AES-256-GCM مع اشتقاق مفتاح باستخدام Argon2id (v3). النظام يدعم فك تشفير الإصدارات القديمة (v1/v2) مع إعادة تشفير تلقائي.

### 12. حماية SSRF مزدوجة

**الملفات:** `internal/handlers/webhooks.go`, `internal/handlers/sso_security.go`

حماية على مستويين:
- **تحقق هيكلي:** `validateWebhookURL` يرفض أسماء النطاقات الداخلية وIPs الخاصة
- **Runtime:** `SSRFSafeDialer` يتحقق من IPs بعد DNS resolution لمنع DNS rebinding

### 13. Rate Limiter مغلق عند الفشل

**الملف:** `internal/middleware/ratelimit.go`

إذا فشل Redis، يُرفض الطلب (fail-closed) بدلاً من السماح به. هذا يمنع المهاجم من استغلال تعطل Redis.

### 14. CSRF Double-Submit Cookie

**الملف:** `internal/middleware/csrf.go`

نمط double-submit (cookie `whm_csrf` + header `X-CSRF-Token`) مع تخطي ذكي للمصادقة عبر headers (Bearer/API key).

### 15. حماية مسار الملفات

**الملف:** `internal/handlers/media.go:466-495`

خدمة الملفات تتضمن:
- `filepath.Clean` + `filepath.Abs` لمنع directory traversal
- `filepath.EvalSymlinks` لمنع symlink attacks
- فحص prefix لضمان أن الملف ضمن الدليل المسموح
- رفض symlink صريح

### 16. API Key Hash بـ bcrypt

**الملف:** `internal/handlers/apikeys.go:145`

مفاتيح API تُخزن bcrypt-hashed. المفتاح الكامل يُعاد مرة واحدة فقط عند الإنشاء.

### 17. تحقق إنتاجي من التكوين

**الملف:** `internal/config/security_validation.go`

في بيئة الإنتاج، يُرفض:
- كلمات مرور قاعدة بيانات غير آمنة (مثل `postgres:postgres`)
- رموز تحقق ويبهوك ضعيفة

---

## المسارات غير المصادقة

المسارات التالية لا تتطلب مصادقة (حسب `main.go:1334-1348`):

| المسار | السبب |
|--------|-------|
| `/health`, `/ready` | فحص صحة الخدمة |
| `/api/license/bootstrap` | تفعيل الترخيص |
| `/api/auth/login` | تسجيل الدخول |
| `/api/auth/register` | إنشاء حساب |
| `/api/auth/refresh` | تجديد رمز الوصول |
| `/api/auth/logout` | تسجيل الخروج |
| `/api/webhook` | استقبال ويبهوكات WhatsApp |
| `/api/facebook/comments/webhook` | استقبال ويبهوكات Facebook |
| `/api/auth/sso/*` | تدفق OAuth2/SSO |
| `/api/custom-actions/redirect/*` | إعادة توجيه برمز لمرة واحدة |
| `/ws` | WebSocket (مصادقة برسالة JWT) |

**التوصية:** تأكد من أن `/api/license/bootstrap` و `/api/license/activate` لديهما rate limiting صارم لمنع brute force.

---

## التوصيات ذات الأولوية القصوى

1. **إضافة شرط الأحرف الخاصة لكلمات المرور** (#1) — تغيير بسيط بأثر أمني كبير
2. **رفض HTTP في Webhook URLs إنتاجيًا** (#2) — يحمي بيانات العملاء أثناء النقل
3. **تشفير Webhook secrets في قاعدة البيانات** (#5) — استخدم `crypto.Encrypt()` الموجود
4. **تشفير SSO client_secret في قاعدة البيانات** (#6) — نفس الآلية
5. **إضافة طبقة RBAC مركزية** (#3) — حماية دفاعية ضد نسيان `requirePermission`
