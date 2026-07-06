# Kingmaster → Go + Vue Refactoring Map — Complete

## 1. Plugin Structure (Target)

Each Dashboard feature becomes a plugin under `plugin/<name>/`:
- `plugin.go` — Plugin interface (`Name()`, `Init()`, `Routes()`, `Migrate()`)
- `models.go` — GORM models
- `handlers.go` — Route handlers
- `service.go` — Business logic (when complex)

Vue frontend: `stores/<domain>.ts` + `services/<domain>.ts` + `views/<domain>/`

## 2. Complete Plugin List

| Plugin | Priority | Whatomate | Go Routes | Models | Vue Views |
|--------|----------|-----------|-----------|--------|-----------|
| **dashboard** | P0 | 60% | 4 | DashboardStats | DashboardView |
| **accounts** | P0 | 40% | 6 | Account (unified) | AccountsView |
| **campaign** | P0 | 75% | Extend | Extend | Extend |
| **facebook** | P1 | 80% | Extend | Extend | Extend |
| **whatsapp** | P1 | 90% | Extend | Extend | Extend |
| **content** | P1 | 50% | 5 | ContentTemplate | ContentsView |
| **instagram** | P2 | 0% | 15+ | InstagramAccount, etc. | 8 new |
| **mlm** | P2 | 0% | 6 | MLMReferral, MLMCommission, MLMSetting | 4 new |
| **wallet** | P2 | 0% | 9 | Wallet, Transaction, Withdrawal | 5 new |
| **ecommerce** | P3 | 0% | 15 | Product, Order, Coupon | 10 new |
| **subscription** | P0 | 100% | — | — | — |

## 3. File-by-File: What Exists vs What Needs Building

### Facebook (80% exists)
| PHP File | Go Status | Vue Status | Priority |
|----------|-----------|------------|----------|
| `api/get_accounts_fb.php` | ✅ fb_accounts.go | ✅ AccountsView | P0 |
| `api/data_fb_search.php` | ✅ fb_people_search.go | ✅ PeopleSearchView | P1 |
| `api/get_comment_post.php` | ✅ fb_comments.go | ✅ CommentsView | P1 |
| `api/create_comments_campaign.php` | ✅ fb_comments.go | ✅ Part of comments | P1 |
| `api/serchpag.php` | ✅ fb_page_search.go | ✅ PageSearchView | P1 |
| `api/search_groups.php` | Partial | ✅ GroupSearchView | P1 |
| `api/facebook_analytics.php` | ✅ meta_analytics.go | ✅ Analytics | P1 |
| `api/rate_post.php` | ❌ New | ❌ New | P2 |
| `api/get_msg_pg.php` | ❌ New | ✅ PageMessengersView | P2 |
| `api/send_img_fb.php` | ❌ New | ❌ New | P2 |
| `api/send_group.php` | ❌ New | ✅ AutoShareView | P2 |
| `api/create_post.php` | ❌ New | ❌ New | P3 |
| `api/get_intervals_fb.php` | ❌ New | ❌ New | P2 |
| `api/save_sending_settings.php` | ❌ New | ❌ New | P2 |

### Instagram (0% exists — new plugin needed)
| PHP File | Actual Content | Notes |
|----------|---------------|-------|
| `insta-search-profile.php` | Profile search | Implement |
| `insta-search-hashtag.php` | Hashtag search | Implement |
| `insta-extract-followers.php` | Extract followers | Implement |
| `insta-extract-likes.php` | Extract likes | Implement |
| `insta-extract-comments.php` | Extract comments | Implement |
| `insta-extract-posts.php` | Extract posts | Implement |
| `insta-extract-viewers.php` | Story viewers | Implement |
| `insta-extract-messages.php` | Extract DMs | Implement |
| `insta-auto-post.php` | Auto post | Implement |
| `insta-auto-story.php` | Auto story | Implement |
| `insta-send-message.php` | Send DM / retarget | Implement |
| `kingmaster.info/insta-mention-tool.php` | Mention campaign | Implement (only `.info` is correct) |
| `kingmaster.info/insta-auto-comments.php` | Auto comments | Implement (only in `.info`) |
| **`insta-follow-tool.php`** | ⚠️ **MISNAMED** — group search | Skip — not real IG follow |
| **`insta-unfollow-tool.php`** | ⚠️ **MISNAMED** — group search | Skip — not real IG unfollow |

### Core Infrastructure (100% exists)
| PHP | Whatomate Equivalent | Status |
|-----|---------------------|--------|
| `api/get_notifications.php` | `notifications.go` + WS hub | ✅ |
| `api/mark_notification_read.php` | `notifications.go` | ✅ |
| `api/files_api.php` | `media.go` + `object_storage.go` | ✅ |
| `api/flows.php` | `flows.go` + `chatbot.go` + template engine | ✅ |
| `api/contacts_api.php` | `contacts.go`, `contacts_management.go` | ✅ |
| `api/send_message.php` | `messages.go` | ✅ |
| `api/get_conversations.php` | `messages.go` | ✅ |
| `api/upload_avatar.php` | `media.go` | ✅ |
| `api/change_password.php` | `auth_handlers.go` | ✅ |

## 4. Database Migration Strategy

| PHP Table | GORM Model | Plugin | Strategy |
|-----------|-----------|--------|----------|
| `users` | Existing Whatomate User | core | Replace |
| `accounts` | `Account` | accounts | New unified model |
| `campaigns` | Extend existing | campaigns | Add tool_type |
| `data_fb` (87M) | `FBPeopleData` | facebook | Partitioned by country |
| `fb_page` (1,967) | `FacebookPage` | facebook | Reuse existing |
| `fb_serch` (38M) | `FBPageSearchResult` | facebook | New model |
| `db_camp` (202K) | `FBCampaignData` | campaigns | New model |
| `groups_list` (5K) | `FacebookGroup` | facebook | New model |
| `content` (843) | `ContentTemplate` | content | New model |
| `contacts` (2,720) | Extend existing | contacts | Add platform filter |
| `mlm_*` | New MLM models | mlm | New plugin |
| `wallet*` | New wallet models | wallet | New plugin |
| `products*` | New product models | ecommerce | New plugin |
| `orders*` | New order models | ecommerce | New plugin |
| `coupons` | New coupon models | ecommerce | New plugin |
| `wa_msg*` (2.2M) | Extend existing | whatsapp | Already exists |
| `ig_*` tables | New IG models | instagram | New plugin |
| `wpp_*` (9.6M) | Extend existing | whatsapp | Already exists |
| `sending_settings` | `SendingSettings` | campaigns | New model |
| `notifications` (19K) | Extend existing | notifications | Already exists |

## 5. Key Implementation Notes

### 5.1 Points Economy → License Tiers
Replace points with subscription/license tiers (Free/Pro/Enterprise). Existing Whatomate licensing handles this.

### 5.2 Large Data Migration
- `data_fb` (87M rows) → PostgreSQL PARTITION BY LIST (country)
- `fb_serch` (38M rows) → PARTITION BY RANGE (campaign_id)
- `retarget_rep` (52M rows) → PARTITION BY RANGE (created_at)
- `wpp_events` (7.4M rows) → Keep 30 days, partition by date

### 5.3 Facebook Rate Limiting
Current: file-based → Target: Redis sliding window
- `data_fb_search`: 30 req/min
- `create_campaign`: 20 req/min

### 5.4 Token Security
- Encrypt at rest using `internal/crypto/`
- OAuth flow through existing `fb_oauth.go`
- `api/or.php` security concern: runs cookies/token tools without any auth

### 5.5 Campaign Worker
Current: PHP polling → Target: Redis Streams consumer group (already in Whatomate)

### 5.6 Hardcoded Secrets Found
- `api/whatsapp_proxy.php`: `$WAAPI_KEY = 'VV9D6WL23X'`
- `api/proxy.php`: Proxies to `apis.kingmaster.info`
- `includes/send_otp.php`: `instance_id = '6967AAB9ADA6E'`, `access_token = '6604ac2316788'`
- `kingmaster.info/config/database.php`: Plaintext DB credentials
