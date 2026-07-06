# Kingmaster Dashboard — Complete Functions Catalog

## 1. Core Database Functions (`config/database.php`)

### Environment & Config
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `configValue()` | `($key, $default = '')` | string | Read env var with fallback chain (`getenv` → `$_ENV` → `$_SERVER` → default) |
| `isProductionEnv()` | `()` | bool | `APP_ENV === 'production'` |

### JSON Response Helpers (production version only)
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `respondJson()` | `($payload, $statusCode = 200)` | void (exit) | JSON response + security headers |
| `respondError()` | `($message, $statusCode = 400)` | void (exit) | Error JSON response |
| `readJsonBody()` | `($maxBytes = 1048576)` | array | Parse JSON body with size validation |
| `applySecurityHeaders()` | `()` | void | CSP, XSS, X-Frame, Referrer, Permissions |
| `applyCorsHeaders()` | `($methods = 'GET, POST, OPTIONS')` | void | CORS from allowed origins config |

### Auth & Security (production version only)
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `startSecureSession()` | `()` | void | Secure session with HttpOnly/SameSite |
| `csrfToken()` | `()` | string | Get/generate CSRF token |
| `csrfInput()` | `()` | string | HTML hidden CSRF input |
| `verifyCsrfToken()` | `($token = null)` | void (exit on fail) | Validate CSRF token |
| `clientIpAddress()` | `()` | string | Validated client IP |
| `enforceRateLimit()` | `($scope, $limit, $windowSeconds)` | void (exit on limit) | File-based rate limiting |
| `requireAuthenticatedUser()` | `()` | string user_id | Session auth guard |
| `requireAdminUser()` | `()` | string user_id | Admin role guard (checks `is_admin`/`user_type`) |

### Input Sanitization
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `safeBaseName()` | `($name, $fallback = 'file')` | string | Sanitize filename |
| `cleanText()` | `($value, $maxLength = 255)` | string | Strip tags/special chars, truncate |
| `sanitizeInput()` | `($data)` | string/array | htmlspecialchars (XSS prevention) |
| `requireJsonArray()` | `($data, $message)` | array | Validate array input |

### Database Helpers (Database class — both projects)
| Method | Signature | Return | Description |
|--------|-----------|--------|-------------|
| `Database->connect()` | `()` | PDO | MySQL PDO singleton (utf8mb4, InnoDB) |
| `Database->disconnect()` | `()` | void | Nullify PDO |
| `Database::getInstance()` | `()` | PDO | Static singleton accessor |

### Database Query Helpers (both projects)
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `getDB()` | `()` | PDO | Alias for `Database::getInstance()` |
| `executeQuery()` | `($query, $params = [])` | PDOStatement|false | Prepared statement execution |
| `fetchRow()` | `($query, $params = [])` | array|false | Fetch single row |
| `fetchAll()` | `($query, $params = [])` | array|false | Fetch all rows |
| `getRowCount()` | `($query, $params = [])` | int | Count affected rows |
| `getLastInsertId()` | `()` | string | Last inserted ID |

### Password/Token Utilities (both projects)
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `hashPassword()` | `($password)` | string | ARGON2ID or BCRYPT cost=12 |
| `verifyPassword()` | `($password, $hash)` | bool | password_verify |
| `generateSecureToken()` | `($length = 32)` | string | random_bytes token |
| `isValidEmail()` | `($email)` | bool | filter_var validation |

## 2. Business Logic Functions (`includes/functions.php`)

### User Data
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `getUserByUserId()` | `($user_id)` | array|false | Full user profile query (18 fields) |
| `getUserIsAdmin()` | `($user_id)` | string|null | Check admin status column |
| `getUserData()` | `($user_id)` | array|null | Enriched user profile: name, avatar, plan, features, days remaining, points, referrals |
| `get_Exp()` | `($id)` | array|false | Raw user record from `users` |

### Dashboard Statistics
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `getWeeklyStats()` | `($user_id)` | array | Campaign `true_count` grouped by weekday (last 7 days) |
| `getPlatformStats()` | `($user_id)` | array | Campaign count grouped by platform |
| `getCampaignCount()` | `($user_id)` | int | Total campaigns for user |
| `getMonthlyPoints()` | `($user_id)` | array | 12-value array: points consumed per month (for Chart.js) |
| `getExtractTrueCount()` | `($user_id)` | int | Sum of `true_count` where `type_tools='Extract'` |
| `getLastPosts()` | `($limit = 4)` | array | Latest system posts (New Feature / System Update / Maintenance) |

### Package
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `getPackageName()` | `($id)` | string|null | Package name by ID |
| `getContactCount()` | `($id)` | int | Contact count from `contacts` table |

### Referral/MLM
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `getReferralCountByReferrerId()` | `($referrer_id)` | int | Direct referral count |
| `getReferralsByReferrerId()` | `($referrer_id)` | array | Last 4 referrals with user info |

### Activity & Utilities
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `getActivityLog()` | `($user_id)` | array | Last 5 log entries with icon/color mapping |
| `generateTimezoneList()` | `()` | array | All PHP timezones with GMT offset + city |
| `getcommission_walletsById()` | `($id)` | array|false | Commission wallet record |

### Syswalt (System Wallet)
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `insertSyswalt()` | `($price, $typs, $created_at)` | string | Insert syswalt record |
| `getAllSyswalt()` | `($user_id, $filter_date, $filter_year, $filter_month, $offset, $per_page)` | array | Paginated syswalt with totals |

### Announcements CRUD
| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `getAllAnnouncements()` | `($active_only = true)` | array | List announcements |
| `getAnnouncementById()` | `($id)` | array|false | Single announcement |
| `createAnnouncement()` | `($title, $message, $is_active = 1)` | string | Create announcement |
| `updateAnnouncement()` | `($id, $title, $message, $is_active)` | bool | Update announcement |
| `deleteAnnouncement()` | `($id)` | bool | Delete announcement |
| `getUnreadAnnouncements()` | `($user_id)` | array | Unread announcements for user |
| `markAnnouncementAsViewed()` | `($user_id, $announcement_id)` | bool | Mark announcement viewed |

## 3. MLM Functions (`includes/mlm_functions.php`)

| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `getReferralChain()` | `($conn PDO, $user_id)` | array | Walk up 4 levels of referrers |
| `distributeMLMCommissions()` | `($conn, $buyer_id, $package_id, $final_amount)` | array | Distribute commissions across chain (updates `commission_wallets`, inserts `mlm_commissions`) |
| `registerReferral()` | `($conn, $user_id, $referral_code)` | array | Register new referral (inserts `mlm_referrals`, updates `users.referrer_id`) |
| `getMLMStats()` | `($conn, $user_id)` | array | MLM statistics: direct referrals, total commissions, level breakdown |

**Constants:** `MLM_COMMISSION_RATES` = [1=>15.0, 2=>10.0, 3=>5.0, 4=>2.5]

## 4. Notification Functions (`includes/notification_helper.php`)

| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `addNotification()` | `($user_id, $title, $message, $type = 'info')` | bool | Insert notification record |
| `addNotificationToMultipleUsers()` | `($user_ids, $title, $message, $type = 'info')` | bool | Broadcast to multiple users |

**Types:** `info`, `success`, `warning`, `error`

## 5. WhatsApp OTP Functions (`includes/send_otp.php`)

| Function | Signature | Return | Description |
|----------|-----------|--------|-------------|
| `sendOTP()` | `($phone, $otp)` | array{success, message} | Send OTP via WhatsApp API (`king-master.pro/api/send`) |
| `sendWhatsAppMessage()` | `($phone, $message)` | array{success, message} | Send custom WhatsApp message via API |

## 6. Inline API Helper Functions (in `api/*.php`)

| Function | Defined In | Description |
|----------|-----------|-------------|
| `respond()` | Multiple API files | Local JSON response helper |
| `createAccountsTableIfNotExists()` | `api/accounts_api.php` | Auto-create accounts table |
| `createCampaignsTableIfNotExists()` | `api/campaigns_api.php` | Auto-create campaigns table |
| `generateUid12()` | `api/campaigns_api.php` | Generate unique 12-digit numeric campaign UID |
| `uidExists()` | `api/campaigns_api.php` | Check campaign UID uniqueness |
| `addLikeGroup()` | `api/data_fb_search.php` | Build SQL WHERE clause for multi-value filter fields (work, education, location). Used by local cached DB query |
| `buildTree()` | `api/get_mlm_tree.php` | Recursive MLM tree builder |
| `getWallet()` | `api/wallet_api.php` | Get wallet data |
| `getTransactions()` | `api/wallet_api.php` | Get wallet transactions |
| `transfer()` | `api/wallet_api.php` | Execute wallet transfer |
| `normalizeContactData()` | `api/contacts_api.php` | Normalize contact data structure |
| `changeCookiesFb()` | `api/or.php`, `api/accounts.php` | Parse Facebook cookie string to array. ⚠️ `api/or.php` runs without auth (security concern) |
| `changeToken()` | `api/or.php`, `api/accounts.php` | Exchange Facebook access tokens via fb API. ⚠️ `api/or.php` runs without auth |
| `getFbDtsg()` | `api/or.php` | Fetch Facebook `fb_dtsg` CSRF token. ⚠️ Runs without auth |
| `sendImage()` | `api/send_img_fb.php` | Send image via Facebook Messenger Graph API |
| `getFacebookConversations()` | `root/api_get_msg.php` | Fetch Facebook page conversations |
| `searchFacebookGraphQLPepelo()` | `root/api_serch_pepols_fb.php` | Facebook GraphQL people search |
| `getAllFiles()` | `api/files_api.php` | List all user files |
| `uploadFile()` | `api/files_api.php` | Handle file upload |
| `getStorageInfo()` | `api/files_api.php` | Get storage usage info |
| `buildListUrl()`, `buildPollUrl()` | `api/whatsapp_lists_api.php`, `api/whatsapp_polls_api.php` | Build WhatsApp interactive message URLs |
| `overview()` | `api/analytics_api.php` | Analytics overview |
| `geo()` | `api/analytics_api.php` | Geographic analytics |
| `packages_chart()` | `api/analytics_api.php` | Package distribution chart data |
| `export_all_users_csv()` | `api/analytics_api.php` | Export users to CSV |
| `filter_subs()` | `api/analytics_api.php` | Filter subscribers |
| `addLog()` | `api/change_password.php` | Add activity log entry |

## 7. Frontend JavaScript Functions

### `js/script.js` — Main Dashboard
| Function | Description |
|----------|-------------|
| `toggleSidebar()` | Toggle left sidebar |
| `toggleMobileSidebar()` | Toggle mobile sidebar |
| `toggleTheme()` | Dark/light theme switch |
| `changeLanguage(lang)` | Switch i18n language |
| `setDirection(dir)` | RTL/LTR direction |
| `loadTranslations(lang)` | Load translation strings |
| `fetchDashboardData()` | AJAX dashboard data fetch |
| `initCharts()` | Initialize Chart.js charts (balance, points, tools) |
| CSRF Fetch Guard | Auto-inject CSRF token into fetch headers |

### `js/accounts.js` — Account Management
| `loadAccounts()`, `renderAccounts()`, `createAccountCard()` | Account CRUD |
| `getPlatformInfo()`, `getStatusLabel()` | Platform labels |
| `applyFilters()`, `editAccount()`, `verifyAccount()` | Account actions |
| `reconnectAccount()`, `deleteAccount()`, `loadAccountLimit()` | Account actions |

### `js/wallet.js` — Wallet
| `loadWallet()`, `loadTransactions()` | Wallet data |
| `renderTransactions()`, `transferMoney()` | Transactions |
| `updateMaxAmount()` | Transfer limits |

### `js/products.js` — Product Catalog
| `debounce()`, `loadProducts()`, `renderProducts()` | Product listing |
| `createProductCard()`, `getStockStatus()`, `getProductIcon()` | Product display |
| `openProductDetails()`, `showError()` | Product actions |

### `js/content.js` — Content Templates
| `loadContents()`, `renderContents()` | Content listing |
| `openCreateModal()`, `closeContentModal()` | Modal management |
| `toggleEmojiPicker()`, `insertEmoji()` | Emoji picker |
| `updateCharCount()`, `saveContent()` | Content editing |

### `js/files.js` — File Manager
| `loadData()`, `updateStorageBar()` | File data |
| `openUploadModal()`, `handleFileSelect()`, `uploadFile()` | Upload flow |
| `renderFiles()`, `deleteFile()`, `formatFileSize()` | File display |

### `js/i18n.js` + `js/translations.js` — Internationalization
- `translations` object (ar/en/fr) — Full i18n dictionary

### `js/timezones.js` + `js/country-detection.js` — Utilities
| `getWorldTimezones()`, `populateTimezoneSelect()` | Timezone selection |
| `detectTimezoneFromPhone()`, `searchTimezone()` | Timezone detection |
| `extractCountryCodeFromPhone()`, `detectCountryFromPhone()` | Country detection |

## 8. Layout Include Files

| File | Content |
|------|---------|
| `includes/head.php` | HTML `<head>` with CSS, meta, title |
| `includes/footer.php` | Page footer with JS includes |
| `includes/navbar_top.php` | Top nav bar |
| `includes/navbar_actions.php` | Nav action buttons |
| `includes/navbar_extra_actions.php` | Extra nav actions |
| `includes/sidebar_left.php` | Left sidebar |
| `includes/sidebar_leftx.php` | Extended left sidebar |
| `includes/sidebar_right.php` | Right sidebar |
| `includes/sidebar_rightx.php` | Extended right sidebar |
| `includes/admin_*.php` (7 files) | Admin-specific layouts |
