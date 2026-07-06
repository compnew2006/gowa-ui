# Fix Prompt: Facebook/Instagram API Identity Consistency

## الملف المستهدف
`api/or.php` (وكل ملف PHP فيه طلبات curl لـ facebook.com أو instagram.com أو b-graph.facebook.com)

---

## المشكلة

الملف فيه 6+ دوال كل واحدة بتنتحل شخصية مختلفة (تطبيق مختلف، جهاز مختلف، نظام تشغيل مختلف) مع إنهم بيستخدموا نفس الكوكيز ونفس الحساب. Meta بتعمل cross-request analysis وبتشوف التناقضات دي فوراً.

---

## القاعدة الأساسية

**كل حساب = هوية واحدة ثابتة لا تتغير أبداً:**
- تطبيق واحد (مثلاً Facebook Android App — FB4A)
- جهاز واحد (مثلاً GalaxyS22)
- نظام تشغيل واحد (مثلاً Android 13)
- إصدار واحد (مثلاً FBAV/456.1.0.45.107)
- locale واحد (مثلاً ar_EG)
- operator واحد (مثلاً Vodafone مصر = MCC-MNC: 602-02, HNI: 60202)
- device_id فريد (UUID ثابت لنفس الحساب)

---

## التعديلات المطلوبة

### 1. أنشئ ملف `includes/fb_identity.php`

أنشئ ملف جديد فيه كل ثوابت الهوية. كل دالة في `or.php` يجب أن تسأل هذا الملف عن الهوية بدلاً من تعريفها بنفسها.

```php
<?php
// includes/fb_identity.php

/**
 *identity system for Facebook/Instagram curl requests
 * Every account gets ONE consistent identity across ALL requests
 */

// Pool of realistic device identities — each one is a complete, consistent persona
const FB_DEVICE_POOL = [
    // Samsung Galaxy S23 — Android 14 — Facebook App
    [
        'ua'                => '[FBAN/FB4A;FBAV/456.1.0.45.107;FBBV/45678912;FBDM={density=3.0,width=1080,height=2340};FBLC/ar_EG;FBCR/Vodafone;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/14;FBOP/1;FBCA/arm64-v8a;]',
        'device_model'      => 'SM-S911B',
        'device_name'       => 'Galaxy S23',
        'android_version'   => '14',
        'fb_app_version'    => '456.1.0.45.107',
        'fb_app_id'         => '256002347743983',
        'mcc_mnc'           => '60202',
        'carrier'           => 'Vodafone',
        'locale'            => 'ar_EG',
        'country_code'      => 'EG',
        'screen_density'    => '3.0',
        'screen_width'      => '1080',
        'screen_height'     => '2340',
        'cpu_abi'           => 'arm64-v8a',
    ],
    // Samsung Galaxy A54 — Android 13 — Facebook App
    [
        'ua'                => '[FBAN/FB4A;FBAV/452.0.0.30.118;FBBV/45234567;FBDM={density=2.625,width=1080,height=2340};FBLC/ar_EG;FBCR/Orange;FBPN/com.facebook.katana;FBDV/SM-A546B;FBSV/13;FBOP/1;FBCA/arm64-v8a;]',
        'device_model'      => 'SM-A546B',
        'device_name'       => 'Galaxy A54',
        'android_version'   => '13',
        'fb_app_version'    => '452.0.0.30.118',
        'fb_app_id'         => '256002347743983',
        'mcc_mnc'           => '60202',
        'carrier'           => 'Orange',
        'locale'            => 'ar_EG',
        'country_code'      => 'EG',
        'screen_density'    => '2.625',
        'screen_width'      => '1080',
        'screen_height'     => '2340',
        'cpu_abi'           => 'arm64-v8a',
    ],
    // Xiaomi Redmi Note 13 Pro — Android 14 — Facebook App
    [
        'ua'                => '[FBAN/FB4A;FBAV/455.1.0.44.109;FBBV/44987654;FBDM={density=2.75,width=1080,height=2400};FBLC/ar_EG;FBCR/Etisalat;FBPN/com.facebook.katana;FBDV/23090RAAEG;FBSV/14;FBOP/1;FBCA/arm64-v8a;]',
        'device_model'      => '23090RAAEG',
        'device_name'       => 'Redmi Note 13 Pro',
        'android_version'   => '14',
        'fb_app_version'    => '455.1.0.44.109',
        'fb_app_id'         => '256002347743983',
        'mcc_mnc'           => '60203',
        'carrier'           => 'Etisalat',
        'locale'            => 'ar_EG',
        'country_code'      => 'EG',
        'screen_density'    => '2.75',
        'screen_width'      => '1080',
        'screen_height'     => '2400',
        'cpu_abi'           => 'arm64-v8a',
    ],
    // Oppo Reno 10 — Android 13 — Facebook App
    [
        'ua'                => '[FBAN/FB4A;FBAV/453.0.0.28.109;FBBV/44000012;FBDM={density=2.75,width=1080,height=2412};FBLC/ar_EG;FBCR/WE;FBPN/com.facebook.katana;FBDV/CPH2521;FBSV/13;FBOP/1;FBCA/arm64-v8a;]',
        'device_model'      => 'CPH2521',
        'device_name'       => 'Reno 10',
        'android_version'   => '13',
        'fb_app_version'    => '453.0.0.28.109',
        'fb_app_id'         => '256002347743983',
        'mcc_mnc'           => '60204',
        'carrier'           => 'WE',
        'locale'            => 'ar_EG',
        'country_code'      => 'EG',
        'screen_density'    => '2.75',
        'screen_width'      => '1080',
        'screen_height'     => '2412',
        'cpu_abi'           => 'arm64-v8a',
    ],
    // Huawei Nova 11 — Android 12 — Facebook App
    [
        'ua'                => '[FBAN/FB4A;FBAV/448.0.0.32.118;FBBV/43219876;FBDM={density=2.75,width=1080,height=2388};FBLC/ar_EG;FBCR/Mobinil;FBPN/com.facebook.katana;FBDV/FOD-AL10;FBSV/12;FBOP/1;FBCA/arm64-v8a;]',
        'device_model'      => 'FOD-AL10',
        'device_name'       => 'Nova 11',
        'android_version'   => '12',
        'fb_app_version'    => '448.0.0.32.118',
        'fb_app_id'         => '256002347743983',
        'mcc_mnc'           => '60201',
        'carrier'           => 'Mobinil',
        'locale'            => 'ar_EG',
        'country_code'      => 'EG',
        'screen_density'    => '2.75',
        'screen_width'      => '1080',
        'screen_height'     => '2388',
        'cpu_abi'           => 'arm64-v8a',
    ],
    // Samsung Galaxy S24 Ultra — Android 14 — Facebook App
    [
        'ua'                => '[FBAN/FB4A;FBAV/457.0.0.42.110;FBBV/46123456;FBDM={density=3.5,width=1440,height=3120};FBLC/ar_EG;FBCR/Vodafone;FBPN/com.facebook.katana;FBDV/SM-S928B;FBSV/14;FBOP/1;FBCA/arm64-v8a;]',
        'device_model'      => 'SM-S928B',
        'device_name'       => 'Galaxy S24 Ultra',
        'android_version'   => '14',
        'fb_app_version'    => '457.0.0.42.110',
        'fb_app_id'         => '256002347743983',
        'mcc_mnc'           => '60202',
        'carrier'           => 'Vodafone',
        'locale'            => 'ar_EG',
        'country_code'      => 'EG',
        'screen_density'    => '3.5',
        'screen_width'      => '1440',
        'screen_height'     => '3120',
        'cpu_abi'           => 'arm64-v8a',
    ],
    // Realme 11 Pro — Android 13 — Facebook App
    [
        'ua'                => '[FBAN/FB4A;FBAV/451.1.0.38.112;FBBV/43876543;FBDM={density=2.75,width=1080,height=2412};FBLC/ar_EG;FBCR/Etisalat;FBPN/com.facebook.katana;FBDV/RMX3771;FBSV/13;FBOP/1;FBCA/arm64-v8a;]',
        'device_model'      => 'RMX3771',
        'device_name'       => 'Realme 11 Pro',
        'android_version'   => '13',
        'fb_app_version'    => '451.1.0.38.112',
        'fb_app_id'         => '256002347743983',
        'mcc_mnc'           => '60203',
        'carrier'           => 'Etisalat',
        'locale'            => 'ar_EG',
        'country_code'      => 'EG',
        'screen_density'    => '2.75',
        'screen_width'      => '1080',
        'screen_height'     => '2412',
        'cpu_abi'           => 'arm64-v8a',
    ],
    // Poco X6 Pro — Android 14 — Facebook App
    [
        'ua'                => '[FBAN/FB4A;FBAV/454.0.0.35.107;FBBV/44567890;FBDM={density=2.75,width=1080,height=2400};FBLC/ar_EG;FBCR/Orange;FBPN/com.facebook.katana;FBDV/2311DRK48G;FBSV=14;FBOP/1;FBCA/arm64-v8a;]',
        'device_model'      => '2311DRK48G',
        'device_name'       => 'Poco X6 Pro',
        'android_version'   => '14',
        'fb_app_version'    => '454.0.0.35.107',
        'fb_app_id'         => '256002347743983',
        'mcc_mnc'           => '60202',
        'carrier'           => 'Orange',
        'locale'            => 'ar_EG',
        'country_code'      => 'EG',
        'screen_density'    => '2.75',
        'screen_width'      => '1080',
        'screen_height'     => '2400',
        'cpu_abi'           => 'arm64-v8a',
    ],
];

/**
 * Get the identity assigned to a specific account.
 * Each account always gets the SAME identity (deterministic based on account_id).
 */
function getAccountIdentity(string $account_id): array {
    $index = abs(crc32($account_id)) % count(FB_DEVICE_POOL);
    return FB_DEVICE_POOL[$index];
}

/**
 * Generate a unique, stable device_id for an account.
 * Same account always gets the same device_id.
 */
function getDeviceId(string $account_id): string {
    $hash = hash('sha256', 'device_id_salt_' . $account_id . '_fixed');
    return sprintf(
        '%s-%s-%s-%s-%s',
        substr($hash, 0, 8),
        substr($hash, 8, 4),
        substr($hash, 12, 4),
        substr($hash, 16, 4),
        substr($hash, 20, 12)
    );
}

/**
 * Build the complete set of headers for a Facebook Mobile API request.
 * ALL headers are derived from the account's identity — always consistent.
 */
function getFBMobileHeaders(string $account_id, ?string $access_token = null): array {
    $identity = getAccountIdentity($account_id);

    $headers = [
        'User-Agent'              => $identity['ua'],
        'Content-Type'            => 'application/x-www-form-urlencoded',
        'Accept'                  => '*/*',
        'Accept-Encoding'         => 'gzip, deflate',
        'Accept-Language'         => $identity['locale'] . ',en-US;q=0.9',
        'Authorization'           => $access_token
                                        ? 'OAuth ' . $access_token
                                        : 'OAuth null',
        'X-FB-Connection-Type'    => 'WIFI',
        'X-FB-HTTP-Engine'        => 'Liger',
        'X-FB-Device-IP'          => 'Auto',
        'X-FB-Friendly-Name'      => 'unknown',
        'X-FB-NETWORK-BANDWIDTH-GBPS' => '-1.000',
        'X-FB-NETWORK-TYPE'       => '0',
        'X-FB-QPL-BANDWIDTH-GBPS' => '-1.000',
        'X-FB-SIM-HNI'            => $identity['mcc_mnc'],
        'X-FB-SIM-MCC-MNC'        => substr($identity['mcc_mnc'], 0, 3) . '-' . substr($identity['mcc_mnc'], 3),
        'X-FB-SIM-Operator'       => $identity['mcc_mnc'],
        'Connection'              => 'keep-alive',
    ];

    return $headers;
}

/**
 * Format headers array for curl_setopt (from key-value array to indexed array of "Key: Value" strings)
 */
function formatCurlHeaders(array $headers): array {
    $result = [];
    foreach ($headers as $key => $value) {
        $result[] = $key . ': ' . $value;
    }
    return $result;
}
```

---

### 2. عدّل كل دالة في `or.php`

القاعدة: **أي دالة فيها `curl_init` لأي domain تابع لـ Facebook أو Instagram — يجب أن تستخدم `getAccountIdentity()` و `getFBMobileHeaders()` و `getDeviceId()` بدلاً من القيم المكتوبة يدوياً.**

#### 2.1 الدالة `rb()`

**المشكلة الحالية:**
```php
// UA ثابت مكتوب يدوياً — Orca Messenger
"User-Agent: Dalvik/2.1.0 (Linux; U; Android 9; ASUS_I005DA Build/PI) [FBAN/Orca-Android;FBAV/391.2.0.20.404;...]"
// فقط 3 هيدرز
```

**المطلوب:**
```php
function rb($account_id) {
    $identity = getAccountIdentity($account_id);
    $deviceId = getDeviceId($account_id);
    $headers = getFBMobileHeaders($account_id);
    // ... استخدم $headers و $identity['ua'] و $deviceId في كل مكان
}
```

- احذف الـ UA الثابت `Dalvik/...Orca-Android`
- استبدله بـ `$identity['ua']`
- أضف كل الـ X-FB headers من `getFBMobileHeaders()`
- أضف `$deviceId` في الـ POST body بدل القيمة الثابتة

#### 2.2 الدالة `fb_auth_login()`

**المشكلة الحالية:**
```php
"device_id" => "787c0621-0a03-4e95-8f9d-7a779445f62e",  // ثابت!
"User-Agent: Dalvik/2.1.0 ... Orca-Android ..."          // Orca!
"Authorization: OAuth null"                                // دايماً null!
```

**المطلوب:**
- `$account_id` يجب أن يُمرر كـ parameter للدالة
- `device_id` = `getDeviceId($account_id)` — فريد لكل حساب
- `User-Agent` = `$identity['ua']` — مطابق للجهاز المزعوم
- `Authorization` = نفس الـ logic بس إذا عندك access_token استخدمه
- أضف كل X-FB headers
- استخدم `CURLOPT_HTTPHEADER` مع `formatCurlHeaders(getFBMobileHeaders($account_id))`

#### 2.3 الدالة `getliteNew()`

**المشكلة الحالية:**
```php
"User-Agent: [FBAN/FB4A;FBAV/400.0.0.0.0;...GalaxyS22...]"  // FB4A + GalaxyS22
// لكن باقي الدوال بتستخدم Orca + ASUS
"device_id" => "787c0621-..."                                 // نفس device_id الثابت
```

**المطلوب:**
- `$account_id` يُمرر كـ parameter
- كل القيم تأتي من `getAccountIdentity($account_id)`:
  - `app_id` → من الـ identity
  - `api_key` → من الـ identity
  - `device_id` → `getDeviceId($account_id)`
  - `locale` → `$identity['locale']`
  - `client_country_code` → `$identity['country_code']`
  - UA → `$identity['ua']`
- أضف X-FB headers الكاملة

#### 2.4 الدالة `getFbDtsg()`

**المشكلة الحالية:**
```php
'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 ... Chrome/117.0.0.0'
// ده Desktop Chrome على Windows!
// بس باقي الدوال بتقول Android
```

**المطلوب:**
- استبدل الـ UA بـ الـ UA اللي جاي من `getAccountIdentity($account_id)`
- يعني لو الهوية بتقول FB4A Android — ده Android App مش Chrome Desktop
- محتاج تبني Android WebView UA مش Desktop Chrome UA:
  ```
  Mozilla/5.0 (Linux; Android {version}; {device_model}) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.6422.113 Mobile Safari/537.36
  ```
  يعني من الـ identity: `Android {android_version}; {device_model}`
- احذف كل الـ `sec-ch-ua` و `sec-ch-ua-platform` headers اللي بتقول Windows
- أضف `Sec-Fetch-Dest: document`, `Sec-Fetch-Mode: navigate` المناسبة لـ mobile webview
- أو بشكل أفضل: حل الـ fb_dtsg من داخل Facebook App API مباشرة بدون ما تفتح صفحة ويب

#### 2.5 الدالة `run()`

**المشكلة الحالية:**
```php
'sec-ch-ua: "Chromium";v="117", "Not;A=Brand";v="8"',
'sec-ch-ua-platform: "Windows"',
'user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 ... Chrome/117.0.0.0'
// ده Chrome Desktop 117 على Windows — قديم ومتناقض
```

**المطلوب:**
- نفس تعديل `getFbDtsg()` — استخدم Android WebView UA مش Desktop Chrome
- أو استخدم GraphQL API من الـ Mobile endpoint (`b-graph.facebook.com`) بدل `www.facebook.com/api/graphql/`

#### 2.6 الدالة `native_sso_approve()`

**المشكلة الحالية:**
```php
"User-Agent: Mozilla/5.0 (Linux; Android 9; ASUS_I005DA Build/PI) AppleWebKit/537.36 ... Chrome/68.0.3440.70 Mobile Safari/537.36"
// Chrome 68! — ده إصدار من 2018 — مش موجود في الطبيعة
```

**المطلوب:**
- استخدم UA اللي جاي من `getAccountIdentity($account_id)` + Android WebView pattern
- إصدار Chrome يجب أن يكون حديث (125+)

#### 2.7 الدالة `get_one_login()`

**المشكلة الحالية:**
```php
curl_setopt($ch, CURLOPT_USERAGENT, 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 ... Chrome/126.0.0.0');
// ده Chrome Desktop 126 على Windows
// بس باقي الدوال بتقول Android
```

**المطلوب:**
- استخدم نفس UA identity

#### 2.8 الدالة `getfb_dtsg()`

**المشكلة الحالية:**
```php
curl_setopt($ch, CURLOPT_USERAGENT, $user_agent); // $_SERVER['HTTP_USER_AGENT']
// بيستخدم UA الزائر الفعلي — ممكن يكون أي حاجة
```

**المطلوب:**
- ممنوع تستخدم `$_SERVER['HTTP_USER_AGENT']` — ده UA بتاع مستخدم الموقع مش بتاع فيسبوك
- استخدم UA من `getAccountIdentity($account_id)`

#### 2.9 الدالة `get_page()`

**المشكلة الحالية:**
```php
$ch = curl_init("https://102.132.103.8/v2.9/me?fields=...");
curl_setopt($ch, CURLOPT_HTTPHEADER, ['Host: graph.facebook.com']);
curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
// يتصل بـ IP مباشر + Host spoofing + SSL verification disabled
```

**المطلوب:**
- استبدل `https://102.132.103.8` بـ `https://graph.facebook.com` مباشرة
- شيل `CURLOPT_SSL_VERIFYPEER => false` — أو على الأقل خليه `true`
- شيل الـ Host header spoofing

---

### 3. أضف توقيت بين الطلبات

أنشئ `includes/fb_rate_limit.php`:

```php
<?php
// Track request count per account per day
// Enforce delay between requests

function fbRequestGate(string $account_id): void {
    $conn = getDB();

    // Check daily limit
    $stmt = $conn->prepare("
        SELECT COUNT(*) as cnt 
        FROM fb_request_log 
        WHERE account_id = ? AND DATE(created_at) = CURDATE()
    ");
    $stmt->execute([$account_id]);
    $count = (int)$stmt->fetchColumn();

    $limit = fbDailyLimit($account_id);
    if ($count >= $limit) {
        throw new Exception("Daily request limit reached for account $account_id");
    }

    // Enforce minimum delay between requests (2-8 seconds random)
    $lastRequest = fbGetLastRequestTime($account_id);
    if ($lastRequest) {
        $elapsed = time() - $lastRequest;
        $minDelay = rand(2, 8);
        if ($elapsed < $minDelay) {
            sleep($minDelay - $elapsed);
        }
    }

    // Log this request
    fbLogRequest($account_id);
}

function fbDailyLimit(string $account_id): int {
    // Could be based on account age, package, etc.
    return 800;
}
```

---

### 4. أضف proxy لكل حساب

أنشئ migration:
```sql
ALTER TABLE accounts ADD COLUMN proxy_url VARCHAR(500) NULL;
```

وفي كل طلب curl:
```php
if ($proxy = getAccountProxy($account_id)) {
    curl_setopt($ch, CURLOPT_PROXY, $proxy);
    curl_setopt($ch, CURLOPT_PROXYTYPE, CURLPROXY_HTTP);
}
```

---

### 5. استبدال curl بـ curl-impersonate (اختياري بس مهم)

بدل `curl_init()` العادي استخدم binary مخصص:
```php
// Option A: shell_exec with curl-impersonate
$cmd = "/usr/local/bin/curl_chrome116 -s ...";

// Option B: PHP extension (if available)
// see https://github.com/nicehash/php-curl-impersonate
```

---

## Checklist — تأكد إن كل ده اتعمل

- [ ] ملف `includes/fb_identity.php` اتعمل وكل الـ identities متسقة
- [ ] `rb()` بياخد `$account_id` parameter وبيستخدم identity system
- [ ] `fb_auth_login()` بياخد `$account_id` وبيستخدم identity system
- [ ] `getliteNew()` بياخد `$account_id` وبيستخدم identity system
- [ ] `getFbDtsg()` مش بيستخدم Desktop UA — بيستخدم Android WebView UA من identity
- [ ] `run()` مش بيستخدم Desktop UA — بيستخدم Android WebView UA من identity
- [ ] `native_sso_approve()` بيستخدم UA من identity (مش Chrome 68 القديم)
- [ ] `get_one_login()` بيستخدم UA من identity
- [ ] `getfb_dtsg()` مش بيستخدم `$_SERVER['HTTP_USER_AGENT']`
- [ ] `get_page()` بتتصل بـ `graph.facebook.com` مش IP مباشر
- [ ] كل الـ `device_id` الثابتة اتبدلت بـ `getDeviceId($account_id)`
- [ ] كل الـ `Authorization: OAuth null` اتبدلت بـ token فعلي لو موجود
- [ ] X-FB headers موجودة في كل طلب
- [ ] Rate limiting موجود
- [ ] Proxy لكل حساب موجود

## ممنوع تعمله

- متغيرش الـ business logic — نفس الخطوات نفس الترتيب
- متغيرش قاعدة البيانات
- متغيرش أي ملف تاني غير `or.php` و `includes/` (ملفات جديدة بس)
- متحذفش أي دالة — بس عدّلها
- متضيفش dependencies جديدة من غير ما تسأل
