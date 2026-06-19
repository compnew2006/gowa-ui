# Kingmaster API Endpoints — Complete Reference (132 Endpoints)

## 1. Accounts & Users (17 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 1 | `api/accounts.php` | GET, POST | `user_id` (session) | Account CRUD + Facebook token exchange (`changeCookiesFb()`, `changeToken()`) |
| 2 | `api/accounts_api.php` | GET, POST, PUT, DELETE | `{id}`, `channel`, `status` | Full accounts CRUD; auto-creates table |
| 3 | `api/get_accounts.php` | GET | `user_id` (session) | List user's Facebook accounts |
| 4 | `api/get_accounts_fb.php` | GET | — | List Facebook-only accounts |
| 5 | `api/get_accounts_ig.php` | GET | — | List Instagram-only accounts |
| 6 | `api/get_accounts_wa.php` | GET | — | List WhatsApp-only accounts |
| 7 | `api/add_or_update_account.php` | POST | `channel`, `method`, `name`, `account_uid`, `cookies_text`, `data` | Add or update account with package limit check |
| 8 | `api/get_name_account.php` | GET | — | Get named/connected accounts (FB pages) |
| 9 | `api/get_page_accounts.php` | GET | — | Get page accounts |
| 10 | `api/get_users.php` | GET | `user_id` (session) | List other users |
| 11 | `api/get_all_users.php` | GET | — | List all MLM users |
| 12 | `api/users_api.php` | GET, POST, PUT, DELETE, PATCH | `{id}` | Admin user CRUD |
| 13 | `api/add_user.php` | POST | `referral_code`, `username`, `email`, `full_name`, `phone`, `package_id` | Register user under referral (MLM) |
| 14 | `api/change_password.php` | POST | `current_password`, `new_password` | Change password with logging |
| 15 | `api/forgot_password.php` | POST | — | Generate & send new password via WhatsApp |
| 16 | `api/reset_password_login.php` | POST | — | Reset password during login |
| 17 | `api/upload_avatar.php` | POST (file) | `file` (image) | Upload user avatar |

## 2. Packages & Subscriptions (7 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 18 | `api/add_package.php` | POST | `name`, `description`, `price`, `features`, `platforms` | Create subscription package |
| 19 | `api/get_packages.php` | GET | — | List all packages |
| 20 | `api/update_package.php` | POST | `id`, `name`, `price`, etc. | Update package |
| 21 | `api/delete_package.php` | POST | `id` | Delete package |
| 22 | `api/packages_api.php` | GET, POST, PUT, DELETE | `{id}` | Full CRUD for packages |
| 23 | `api/process_package_purchase.php` | POST | `package_id`, `payment_method` | Purchase package (handles payments, MLM, wallet) |
| 24 | `api/update_timezone.php` | POST | `timezone` | Update user timezone |

## 3. Points System (8 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 25 | `api/points.php` | GET, POST | `action` | Get/add/deduct points |
| 26 | `api/add_points_package.php` | POST | `points_count`, `price` | Create purchasable points package |
| 27 | `api/get_points_packages.php` | GET | — | List points packages |
| 28 | `api/get_all_points_packages.php` | GET | — | All points packages (admin) |
| 29 | `api/update_points_package.php` | POST | `id`, `points_count`, `price` | Update points package |
| 30 | `api/delete_points_package.php` | POST | `id` | Delete points package |
| 31 | `api/purchase_points.php` | POST | `package_id`, `payment_method` | Purchase points |
| 32 | `api/points_settings_api.php` | GET, POST, PUT, DELETE | — | Points settings CRUD |

## 4. Wallet & Transfers (5 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 33 | `api/wallet_api.php` | POST | `action` (get_wallet/get_transactions/transfer) | Wallet operations |
| 34 | `api/wallet_otp_api.php` | POST | — | Wallet OTP verification |
| 35 | `api/transfer.php` | POST | `receiver_id`, `transfer_type`, `amount`, `password` | Transfer points/money |
| 36 | `api/get_transactions.php` | GET | — | Transaction history |
| 37 | `api/get_wallet_balance.php` | GET | `user_id` | Get wallet balance |

## 5. Campaigns (8 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 38 | `api/campaigns_api.php` | GET, POST, PUT, DELETE | `{id}` | Full campaign CRUD; auto-creates table |
| 39 | `api/create_campaign.php` | POST | `name`, `accounts`, `tools` (11 types), `platform`, `page_url`, `range`, `interval_id`, `speed` | Create campaign — multi-tool dispatcher for 11 tool types across WhatsApp/Facebook/Instagram |
| 40 | `api/create_comments_campaign.php` | POST | `name`, `accounts`, `content_id`, `can_like` | Create Facebook comment reply campaign |
| 41 | `api/get_campaigns.php` | POST (JSON) | `tool` | Get campaigns filtered by tool |
| 42 | `api/manage_campaign.php` | POST | `action` (change_status/delete), `campaign_id` | Change status or delete |
| 43 | `api/get_intervals.php` | GET | — | Get Facebook intervals |
| 44 | `api/get_intervals_fb.php` | GET | — | Facebook intervals |
| 45 | `api/get_intervals_ig.php` | GET | — | Instagram intervals |
| 46 | `api/get_intervals_wa.php` | GET | — | WhatsApp intervals |

## 6. E-Commerce & Products (11 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 47 | `api/products.php` | GET, POST | `search`, `category` | Product CRUD |
| 48 | `api/add_product.php` | POST | product fields + image | Add product with image |
| 49 | `api/get_products.php` | GET | `admin`, `category` | List products with colors/sizes |
| 50 | `api/manage_product.php` | POST | `action`, `product_id` | Update/delete product |
| 51 | `api/update_product.php` | POST | `id`, fields | Update product |
| 52 | `api/delete_product.php` | POST | `id` | Delete product |
| 53 | `api/get_orders.php` | GET | `status`, `search` | List orders with stats |
| 54 | `api/orders.php` | GET, POST | `search`, `status` | User's orders |
| 55 | `api/track_order.php` | POST | `search` | Track order status |
| 56 | `api/update_order_status.php` | POST | `order_id`, `new_status` | Admin update order status |
| 57 | `api/manage_withdrawal.php` | POST | — | Manage withdrawal requests |
| 58 | `api/get_all_withdrawals.php` | GET | — | All withdrawals (admin) |
| 59 | `api/get_withdrawals.php` | GET | — | User's withdrawals |

## 7. Coupons (7 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 60 | `api/add_coupon.php` | POST | `code`, `discount_type`, `discount_value`, `uses_limit` | Create coupon |
| 61 | `api/get_coupons.php` | GET | — | List coupons |
| 62 | `api/coupon_api.php` | POST | `action` (redeem), `code` | Redeem coupon |
| 63 | `api/coupons_api.php` | GET, POST, PUT, DELETE | `{id}` | Coupons full CRUD |
| 64 | `api/update_coupon.php` | POST | `id`, fields | Update coupon |
| 65 | `api/delete_coupon.php` | POST | `id` | Delete coupon |
| 66 | `api/validate_coupon.php` | POST | `coupon_code`, `package_id` | Validate & calculate discount |

## 8. MLM & Commissions (7 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 67 | `api/calculate_commission.php` | POST | `from_user_id`, `sale_amount` | Calculate & distribute MLM commissions |
| 68 | `api/get_mlm_settings.php` | GET | — | Get MLM settings |
| 69 | `api/save_mlm_settings.php` | POST | `settings` array | Save MLM commission rates |
| 70 | `api/get_mlm_tree.php` | GET | `user_id`, `max_levels` | Recursive MLM tree via `buildTree()` |
| 71 | `api/get_package_mlm_settings.php` | GET | — | Package-specific MLM settings |
| 72 | `api/save_package_mlm_settings.php` | POST | — | Save package MLM settings |
| 73 | `api/verify_referral.php` | POST | — | Verify referral code |

## 9. WhatsApp Tools (12 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 74 | `api/wa_msg.php` | GET, POST | `campaign_id`, `q`, `page`, `per_page` | WhatsApp messages with pagination |
| 75 | `api/wa_contacts.php` | GET, POST | `campaign_id`, `q`, `page`, `per_page` | WhatsApp contacts |
| 76 | `api/wa_filter.php` | GET, POST | `campaign_id`, `q`, `page`, `per_page` | WhatsApp number filtering |
| 77 | `api/wa_groups.php` | GET, POST | `campaign_id`, `q`, `page`, `per_page` | WhatsApp groups |
| 78 | `api/wa_rep.php` | GET, POST | `campaign_id` (required), `q`, `page`, `per_page` | WhatsApp **retarget** reply viewer (queries `rb_wa` table). Name "rep" = retarget reply |
| 79 | `api/whatsapp_lists_api.php` | GET, POST, PUT, DELETE | — | WhatsApp list message links CRUD |
| 80 | `api/whatsapp_polls_api.php` | GET, POST, PUT, DELETE | — | WhatsApp polls CRUD |
| 81 | `api/whatsapp_proxy.php` | GET, POST | `action` (create_instance) | WhatsApp proxy; has hardcoded `$WAAPI_KEY = 'VV9D6WL23X'` |
| 82 | `api/verify_whatsapp.php` | POST | — | Verify WhatsApp number (proxies to `apis.kingmaster.info`) |
| 83 | `api/get_wa_chats.php` | GET | `session_id` | Grouped WhatsApp chats |
| 84 | `api/get_wa_conversations.php` | GET | — | WhatsApp conversations |
| 85 | `api/get_wa_sessions.php` | GET | — | WhatsApp sessions |
| 86 | `api/get_mmbers_wa.php` | GET, POST | `campaign_id`, `q`, `page`, `per_page` | WhatsApp group members viewer (typo: "mmbers") |
| 87 | `api/check_blacklist.php` | GET, POST | `phone`, `settings_id` | Check if phone is blacklisted |

## 10. Facebook Tools (11 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 88 | `api/data_fb_search.php` | POST | `action` (count/extract), `type`, `country`, `maritalStatus`, `limit`, `work[]`, `education[]`, `location[]`, `uids[]` | ⚠️ Queries **local cached DB** (`data_fb` table), NOT Facebook live. Filters pre-fetched people data by demographics. Rate limited 30/min |
| 89 | `api/create_post.php` | POST | `user_id`, `title`, `content` | Create Facebook post |
| 90 | `api/get_posts.php` | GET, POST | — | List posts |
| 91 | `api/send_img_fb.php` | GET | `token`, `id`, `imageUrl` | Send image via Facebook Messenger |
| 92 | `api/rate_post.php` | POST | `post_id`, `user_id`, `rating` (1-5) | Rate a post |
| 93 | `api/templates_api.php` | GET, POST, PUT, DELETE | `id`, `q`, `type`, `channel` | Messenger templates CRUD |
| 94 | `api/templates_send.php` | POST | `template_id`, `recipient_id`, `page_access_token` | Send Facebook Messenger template |
| 95 | `api/get_content.php` | GET | — | List content templates |
| 96 | `api/content_api.php` | POST, GET | `action` | Content CRUD |
| 97 | `api/content_messages_api.php` | GET | — | Get message presets |
| 98 | `api/search_groups.php` | GET | `q` | Search Facebook group directory |
| 99 | `api/serchpag.php` | GET, POST | `campaign_id`, `q`, `page`, `per_page` | ⚠️ Typo in name. Paginated viewer for `fb_serch` table (Facebook page search results) |

## 11. Instagram Tools (1 endpoint)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 100 | `api/ig_basic_info.php` | GET, POST | `campaign_id` (required), `table` (whitelist: ig_msg/ig_post/ig_retarget/ig_dms/ig_follow/ig_search_users/ig_search_hashtags/ig_search_locations), `q`, `page`, `per_page` | ⚠️ **Misleading name** — NOT basic info. Multi-table paginated search engine for 8 Instagram extraction tables with dynamic column search |

## 12. Contacts (5 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 101 | `api/contacts_api.php` | POST, GET | `action` (get_all/add/update/delete/get_sending_status/update_sending_progress/reset_sending) | Full contact management |
| 102 | `api/contacts_add.php` | POST | `name`, `data[]` (array of {identifier, name}) | Add contact list with entries (max 5,000) |
| 103 | `api/get_count_contacts.php` | GET | `id` | Count contacts in a list |
| 104 | `api/contacts_lists_fb.php` | GET | — | Facebook contact lists |
| 105 | `api/contacts_lists_wa.php` | GET | — | WhatsApp contact lists |

## 13. Sending Settings (4 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 106 | `api/save_sending_settings.php` | POST | `platform`, `settingsName`, `intervalFrom`, `intervalTo`, `protectionEnabled`, `msgCount`, `blacklist` | Save interval settings with blacklist |
| 107 | `api/get_sending_settings.php` | GET | — | Get sending settings |
| 108 | `api/update_sending_settings.php` | POST | `id`, fields | Update sending settings |
| 109 | `api/delete_sending_settings.php` | POST | `id` | Delete sending settings |

## 14. Files & Media (6 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 110 | `api/files_api.php` | POST, GET | `action` (get_all/upload/update/delete/get_storage) | File CRUD |
| 111 | `api/files_download.php` | POST | `ids[]` | Bulk download files as ZIP |
| 112 | `api/media_api.php` | GET | — | List media files |
| 113 | `api/upload_bot_image.php` | POST (file) | `file` (jpg/png/gif/webp, max 8MB) | Upload chatbot image |
| 114 | `api/upload_bot_pdf.php` | POST (file) | `file` (PDF) | Upload chatbot PDF |
| 115 | `api/upload_message_image.php` | POST (file) | `file` (image) | Upload message image |

## 15. Analytics & Admin (4 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 116 | `api/analytics_api.php` | GET, POST | `action` (overview/geo/packages/export_all_users/filter_subs/avg_age/birthdays/birthday_alerts/export_country_users/search_users/export_search_users/birthdays_stats) | Comprehensive analytics + CSV exports |
| 117 | `api/get_admin_statistics.php` | GET | — | Admin dashboard stats |
| 118 | `api/sales_target_api.php` | GET, POST, PUT, DELETE | `action` | Sales targets + customer CRUD |
| 119 | `api/tools_api.php` | GET, POST, PUT, DELETE | `{id}`, `q`, `status`, `visible` | Tools CRUD |

## 16. Other Utilities (12 endpoints)

| # | File | Method | Parameters | Purpose |
|---|------|--------|------------|---------|
| 120 | `api/proxy.php` | GET, POST | anything | Reverse proxy to `apis.kingmaster.info` with hardcoded `$WAAPI_KEY = 'VV9D6WL23X'` |
| 121 | `api/db_local.php` | — | — | Local DB connection config (PDO bridge) |
| 122 | `api/save_data.php` | GET | `camp_id`, `tool` | Export campaign data to CSV/XLSX (uses PhpSpreadsheet) |
| 123 | `api/get_single_setting.php` | GET | — | Get single setting |
| 124 | `api/get_account_limit.php` | GET | — | Get account limit by package |
| 125 | `api/get_conversations.php` | GET | — | Internal messaging conversations |
| 126 | `api/get_messages.php` | GET | `conversation_id` | Get messages for conversation |
| 127 | `api/send_message.php` | POST | `receiver_id`, `message`, `image_path` | Send internal message |
| 128 | `api/get_notifications.php` | GET | — | Last 5 notifications + unread count |
| 129 | `api/mark_notification_read.php` | POST | `notification_id` | Mark notification as read |
| 130 | `api/flows.php` | GET, POST | `action`, `id` | Chatbot flow builder CRUD |
| 131 | `api/or.php` | GET | (no auth, no session) | ⚠️ **Obfuscated name**. Facebook session/cookie manipulation — `changeCookiesFb()`, `changeToken()`, `getFbDtsg()`. Runs without authentication (security concern) |
| 132 | `api/ss.php` | GET, POST | `campaign_id`, `q`, `page`, `per_page` | ⚠️ **Obfuscated name**. Paginated viewer for `db_camp` table (campaign extracted data: phone, fb_id) |

## API Response Pattern

### Current (PHP)
```json
// Success
{ "success": true, "data": {...}, "message": "تم بنجاح" }

// Error
{ "success": false, "message": "خطأ في البيانات", "code": 400 }

// Paginated
{ "success": true, "data": [...], "total": 100, "page": 1, "per_page": 25, "total_pages": 4 }
```

### Target (Go — Whatomate conventions)
```json
// Success
{ "data": {...}, "message": "Operation successful" }

// Error
{ "error": { "message": "Insufficient permissions", "code": "FORBIDDEN" } }

// Paginated
{ "data": [...], "total": 100, "page": 1, "perPage": 25, "totalPages": 4 }
```

## Frontend Page Templates (api_get_*.php proxy pattern)

These are PHP files in `root/` that proxy specific Facebook API operations:

| File | Proxied Function |
|------|-----------------|
| `api_get_comment.php` | Proxy: get Facebook post comments |
| `api_get_like_fb.php` | Proxy: get Facebook post likes |
| `api_get_mmbers.php` | Proxy: get Facebook group members |
| `api_get_msg.php` | Proxy: get Facebook page messages (uses `getFacebookConversations()`) |
| `api_get_serch_gb_fb.php` | Proxy: search Facebook groups |
| `api_get_serch_pg_fb.php` | Proxy: search Facebook pages |
| `api_post_gb.php` | Proxy: post content to Facebook group |
| `api_send_gb_img.php` | Proxy: send image to Facebook group |
| `api_send_txt_page.php` | Proxy: send text to Facebook page |
| `api_serch_pepols_fb.php` | Proxy: search Facebook people via GraphQL |
