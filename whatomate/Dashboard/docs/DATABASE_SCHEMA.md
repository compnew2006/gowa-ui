# Kingmaster Dashboard — Complete Database Schema (76 Tables)

## Overview

Production database: **MySQL 8 / InnoDB / utf8mb4**
Production row count: **~189M+ total rows**
Database name: `kingmaster`
Schema source: `kingmaster/schema_only.sql` (production dump)

## 1. Largest Tables

| # | Table | Rows | Auto-Increment | Purpose |
|---|-------|------|---------------|---------|
| 1 | `data_fb` | **87,542,089** | 87,542,090 | Facebook people data (local cached DB, not live) |
| 2 | `retarget_rep` | **52,008,804** | 52,008,805 | Facebook retargeting data |
| 3 | `fb_serch` | **38,394,474** | 38,398,226 | Facebook page search results |
| 4 | `wpp_events` | **7,357,840** | 7,357,911 | WhatsApp Web events log |
| 5 | `wpp_messages` | **2,246,870** | 3,051,377 | WhatsApp Web messages |
| 6 | `rb_wa` | **752,006** | 752,007 | WhatsApp retargeting |
| 7 | `filter_wa` | **304,815** | 319,353 | WhatsApp filters |
| 8 | `wa_contacts` | **146,066** | 146,067 | WhatsApp contacts |
| 9 | `wa_msg` | **67,873** | 67,874 | WhatsApp messages |
| 10 | `db_camp` | **202,874** | 202,875 | Campaign extracted data |
| 11 | `ig_dms` | **37,988** | 37,992 | Instagram DMs |
| 12 | `wa_members_gb` | **28,799** | 28,800 | WhatsApp group members |
| 13 | `notifications` | **19,863** | 19,864 | In-app notifications |
| 14 | `campaigns` | **5,011** | 12,903 | Core campaign tracking |
| 15 | `groups_list` | **5,063** | 5,064 | Facebook group directory |
| 16 | `ig_follow` | **5,237** | 5,882 | Instagram follows |

## 2. Schema-Only Tables (kingmaster.info Scattered SQL)

The `kingmaster.info` project has scattered SQL files (`sql/` + `database/`) that define ~20 tables — a subset of the production schema with older column definitions. Key differences:
- Uses `platform` enum instead of `channel` in accounts
- Simpler `users` table (missing `msg_count`, `account_count`, `referrer_id`)
- Multiple schema variants for `packages` (3 designs), `posts` (2 variants)
- Missing all large data tables (data_fb, fb_serch, retarget_rep, wpp_*, etc.)

## 3. Table List (76 Production Tables)

| # | Table | Rows | Purpose |
|---|-------|------|---------|
| 1 | `accounts` | 596 | Multi-platform accounts (8 channels) |
| 2 | `announcements` | 0 | System announcements |
| 3 | `campaigns` | 5,011 | Core campaign table |
| 4 | `commission_wallets` | 475 | MLM commission wallet |
| 5 | `contacts` | 2,720 | Contact lists (FB/IG/WA) |
| 6 | `content` | 843 | Message templates |
| 7 | `content_messages` | 4 | Message presets |
| 8 | `conversations` | 5 | Internal chat conversations |
| 9 | `coupons` | 7 | Discount coupons |
| 10 | `data_fb` | 87,542,089 | Facebook people (local cached) |
| 11 | `db_camp` | 202,874 | Campaign extracted data |
| 12 | `fb_page` | 1,967 | Connected FB pages |
| 13 | `fb_serch` | 38,394,474 | FB page search results |
| 14 | `files` | 644 | Uploaded files |
| 15 | `filter_wa` | 304,815 | WhatsApp filter numbers |
| 16 | `gb_wa` | 1,473 | WhatsApp groups |
| 17 | `groups_list` | 5,063 | FB group directory |
| 18 | `ig_dms` | 37,988 | Instagram DMs |
| 19 | `ig_follow` | 5,237 | Instagram follows |
| 20 | `ig_msg` | 8,996 | Instagram messages |
| 21 | `ig_post` | 3,805 | Instagram posts |
| 22 | `ig_retarget` | 3,105 | Instagram retargeting |
| 23 | `ig_search_hashtags` | 55 | Instagram hashtag search |
| 24 | `ig_search_locations` | 60 | Instagram location search |
| 25 | `ig_search_users` | 253 | Instagram user search |
| 26 | `logs` | 7,069 | Activity logs |
| 27 | `media_files` | 4 | Media library |
| 28 | `messages` | 17 | Internal messages |
| 29 | `messenger_templates` | 4 | Facebook Messenger templates |
| 30 | `mlm_commissions` | 0 | MLM commission records |
| 31 | `mlm_referrals` | 33 | MLM referral relationships |
| 32 | `mlm_settings` | 5 | MLM configuration |
| 33 | `mlm_users` | 1 | MLM user profiles |
| 34 | `notifications` | 19,863 | In-app notifications |
| 35 | `order_status_history` | 35 | Order tracking history |
| 36 | `orders` | 10 | E-commerce orders |
| 37 | `otp_wall` | 50 | OTP verification records |
| 38 | `package_mlm_settings` | 3 | Package MLM settings |
| 39 | `packages` | 5 | Subscription packages |
| 40 | `point_use` | 2 | Points usage log |
| 41 | `points_packages` | 1 | Purchaseable points |
| 42 | `points_pricing` | 4 | Points price tiers |
| 43 | `post_ratings` | 0 | Post ratings |
| 44 | `posts` | 1 | Community posts |
| 45 | `product_colors` | 0 | Product color variants |
| 46 | `product_sizes` | 0 | Product size variants |
| 47 | `products` | 0 | Product catalog |
| 48 | `pvsettigs` | 1 | Point value settings |
| 49 | `rb_wa` | 752,006 | WhatsApp retargeting |
| 50 | `ref` | 4 | Reference data |
| 51 | `representative` | 1 | Sales representatives |
| 52 | `retarget_rep` | 52,008,804 | Facebook retargeting |
| 53 | `sales_customers` | 0 | CRM customers |
| 54 | `sales_target` | 3 | Sales targets |
| 55 | `sending_settings` | 351 | Sending interval settings |
| 56 | `syswalt` | 13 | System wallet records |
| 57 | `tools` | 2 | Tool definitions |
| 58 | `transactions` | 296 | Transaction history |
| 59 | `user_announcement_views` | 0 | Announcement read tracking |
| 60 | `user_coupons` | 0 | Used coupons |
| 61 | `user_subscriptions` | 0 | Subscription history |
| 62 | `users` | 381 | Core user accounts |
| 63 | `users_wallet` | 480 | User wallets |
| 64 | `wa_contacts` | 146,066 | WhatsApp contacts |
| 65 | `wa_conv` | 2 | WhatsApp conversations |
| 66 | `wa_flows` | 70 | Chatbot flow definitions |
| 67 | `wa_members_gb` | 28,799 | WhatsApp group members |
| 68 | `wa_msg` | 67,873 | WhatsApp messages |
| 69 | `wallet_transactions` | 0 | Wallet-specific transactions |
| 70 | `wallets` | 1 | Alternative wallet table |
| 71 | `whatsapp_lists` | 1 | WhatsApp list messages |
| 72 | `whatsapp_polls` | 1 | WhatsApp polls |
| 73 | `withdrawals` | 4 | Withdrawal requests |
| 74 | `wpp_events` | 7,357,840 | WhatsApp Web events log |
| 75 | `wpp_messages` | 2,246,870 | WhatsApp Web messages |
| 76 | `wpp_polls` | 1,024 | WhatsApp Web polls |

## 4. Entity Relationship Diagram

```
users ───┬─── accounts (8 channels: facebook, whatsapp, instagram, telegram, email, sms, tiktok, linkedin)
          ├─── fb_page (page tokens)
          ├─── campaigns (unified execution engine)
          │      ├─── fb_serch (page search results, 38M)
          │      ├─── db_camp (campaign data, 202K)
          │      ├─── gb_wa (WhatsApp groups)
          │      └─── groups_list (FB group directory)
          ├─── data_fb (87M local cached people DB)
          ├─── content / content_messages
          ├─── contacts (platform-filtered lists)
          ├─── sending_settings
          ├─── mlm_referrals ─── mlm_commissions
          ├─── users_wallet ──── transactions
          ├─── wa_msg / wa_conv / wa_contacts
          ├─── ig_dms / ig_follow / ig_msg / ig_post / ig_search_*
          ├─── products ─── orders ─── order_status_history
          ├─── coupons ─── user_coupons
          ├─── notifications / announcements
          └─── files / media_files
```

## 5. Migration Strategy (MySQL → PostgreSQL via GORM)

| Concern | Current (MySQL) | Target (PostgreSQL/GORM) |
|---------|----------------|--------------------------|
| Primary Keys | Auto-increment INT | UUID |
| JSON | LONGTEXT (JSON in string) | JSONB |
| ENUMs | MySQL ENUM type | Go string + validation |
| Timestamps | Manual `created_at/updated_at` | GORM auto-managed |
| Soft Deletes | Manual | `gorm.DeletedAt` |
| Full-Text Search | MySQL FULLTEXT index | PostgreSQL GIN/tsvector |
| Large Tables (87M) | Single table | Partitioning by country/date |
| Tenant Scoping | `user_id` column | `organization_id` UUID |
