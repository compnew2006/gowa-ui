# Kingmaster → Go+Vue Plugin Refactoring — Complete Feature Map

This document maps **every feature** from both Dashboard projects to its minimum Go backend + Vue frontend implementation as a Whatomate plugin.

## Plugin Template

```go
// plugin/<name>/plugin.go
package <name>
func init() { core.RegisterPlugin(&Plugin{}) }

type Plugin struct {
    app *handlers.App; db *gorm.DB; rdb *redis.Client; log *slog.Logger
}
func (p *Plugin) Name() string       { return "<name>" }
func (p *Plugin) Init(...) error     { /* store deps */ }
func (p *Plugin) Routes(g *fastglue.Fastglue) { /* register */ }
func (p *Plugin) Migrate(db *gorm.DB) error   { /* AutoMigrate */ }
```

**Vue pattern:** `stores/<domain>.ts` + `services/<domain>.ts` + `views/<domain>/` + `router/index.ts`

---

## 1. 👤 User Authentication & Access Control

**Current:** `root/login.php`, `root/register.php`, `root/verify_otp.php`, `root/forgot-password.html`, `includes/send_otp.php`

**Whatomate status:** ✅ 100% exists — JWT auth, registration, password reset, RBAC, SSO.

**New:** OTP via WhatsApp — add `POST /api/auth/send-otp` + `POST /api/auth/verify-otp` to `auth_handlers.go`.

---

## 2. 📊 Dashboard & Home

**Current:** `root/index.php`, `includes/functions.php` (dashboard queries), `js/script.js` (charts)

**Min Go Backend (New Plugin: `plugin/dashboard/`):**
| Route | Method | Purpose |
|-------|--------|---------|
| `GET /api/dashboard/stats` | GET | Weekly + platform stats + counts |
| `GET /api/dashboard/points-history` | GET | Points per month (12-value array) |
| `GET /api/dashboard/updates` | GET | System updates/announcements |
| `GET /api/dashboard/referrals` | GET | Recent referrals |

**Vue:** `DashboardView.vue` — stats, charts (Chart.js), referral feed, announcements. Store: `useDashboardStore()`.

---

## 3. 👥 Multi-Platform Account Management

**Current:** 10 API files in `api/` (accounts, get_accounts, etc.), `js/accounts.js`

**Whatomate status:** ✅ Partial — `FacebookAccount` model + `fb_accounts.go` exist.

**Min Go Backend (New Plugin: `plugin/accounts/`):**
| Route | Method | Purpose |
|-------|--------|---------|
| `GET /api/accounts` | GET | List with channel filter |
| `POST /api/accounts` | POST | Add (cookies/data/oauth) |
| `PUT /api/accounts/:id` | PUT | Update |
| `DELETE /api/accounts/:id` | DELETE | Delete |
| `GET /api/accounts/limit` | GET | Package limit check |

**Model:** `Account{ID, OrgID, Name, AccountUID, Channel, Status, Method, CookiesText(encrypted), Data(JSONB)}`

**Vue:** `AccountsView.vue`, `AccountCreateView.vue`, `AccountDetailView.vue`. Store: `useAccountsStore()`.

---

## 4. 🎯 Facebook Tools (Complete Suite)

**Whatomate status:** ✅ **80% exists** — see `FACEBOOK_MODELS.md` for full cross-reference.

### Already exists:
- `FacebookAccount` model + handler ✅
- `FacebookComment` + comment reply + settings ✅
- `FBPageSearch` model + handler ✅
- `FBPeopleSearch` model + handler ✅
- Vue views for accounts, comments, page search, group search, people search, page messengers, retargeting, auto-share, likes extraction, data extraction ✅

### Needs adding:
| Route | Method | Purpose |
|-------|--------|---------|
| `POST /api/facebook/groups/search` | POST | Group search (Vue view exists) |
| `POST /api/facebook/posts/like` | POST | Like campaign |
| `GET /api/facebook/messages/conversations` | GET | Page conversations |
| `POST /api/facebook/messages/send` | POST | Send via page |
| `POST /api/facebook/groups/:id/share` | POST | Auto-share to groups |
| `POST /api/facebook/posts/:id/rate` | POST | Rate post (1-5) |
| `GET /api/facebook/posts/:id/ratings` | GET | Get ratings |
| `GET /api/facebook/content` | GET | Content templates CRUD |
| `GET /api/facebook/settings` | GET | Sending settings CRUD |
| `GET /api/facebook/intervals` | GET | List FB intervals |

**⚠️ Note:** `data_fb_search.php` queries a **local cached DB** (`data_fb` table), not live Facebook. This is a 87M-row pre-fetched data warehouse.

---

## 5. 📱 WhatsApp Tools

**Whatomate status:** ✅ **~90% exists** — Campaign system, message sending, contacts, group extraction, member extraction, message extraction, number filtering, chat UI, WebSocket updates, polls, chatbot flows all already exist.

**Add:** Fine-grained tool type mapping for all campaign types (Extract Contacts WA, Extract Messages WA, Wa Filter, Wa Sender, etc.).

---

## 6. 📸 Instagram Tools

**Whatomate status:** ❌ **0% exists** — New plugin needed.

### Features:
| # | Feature | PHP File | Notes |
|---|---------|----------|-------|
| 1 | Profile Search | `insta-search-profile.php` | ✅ Correct |
| 2 | Hashtag Search | `insta-search-hashtag.php` | ✅ |
| 3 | Location Search | `insta-search-location.php` | ✅ |
| 4 | Extract Followers | `insta-extract-followers.php` | ✅ |
| 5 | Extract Likes | `insta-extract-likes.php` | ✅ |
| 6 | Extract Comments | `insta-extract-comments.php` | ✅ |
| 7 | Extract Posts | `insta-extract-posts.php` | ✅ |
| 8 | Story Viewers | `insta-extract-viewers.php` | ✅ |
| 9 | Extract DMs | `insta-extract-messages.php` | ✅ |
| 10 | Auto-Post | `insta-auto-post.php` | ✅ |
| 11 | Auto-Story | `insta-auto-story.php` | ✅ |
| 12 | **Follow Tool** | `insta-follow-tool.php` | ⚠️ **MISNAMED** — actually a group search tool |
| 13 | **Unfollow Tool** | `insta-unfollow-tool.php` | ⚠️ **MISNAMED** — byte-identical to follow, also group search |
| 14 | **Mention Tool** | `kingmaster.info/insta-mention-tool.php` | ⚠️ Only correct in `.info`; `root/` has group search |
| 15 | Send DM / Retarget | `insta-send-message.php` | ✅ |
| 16 | Auto Comments | `kingmaster.info/insta-auto-comments.php` | ✅ (only in `.info`) |

### Min Go Backend (New Plugin: `plugin/instagram/`):
| Route | Method | Purpose |
|-------|--------|---------|
| `GET /api/instagram/accounts` | GET | List IG accounts |
| `POST /api/instagram/accounts` | POST | Add IG account |
| `DELETE /api/instagram/accounts/:id` | DELETE | Remove |
| `POST /api/instagram/profiles/search` | POST | Search profiles |
| `POST /api/instagram/hashtags/search` | POST | Search hashtags |
| `POST /api/instagram/locations/search` | POST | Search locations |
| `POST /api/instagram/followers/extract` | POST | Extract followers |
| `POST /api/instagram/likes/extract` | POST | Extract likes |
| `POST /api/instagram/comments/extract` | POST | Extract comments |
| `POST /api/instagram/posts/extract` | POST | Extract posts |
| `POST /api/instagram/viewers/extract` | POST | Story viewers |
| `POST /api/instagram/auto-post` | POST | Auto post campaign |
| `POST /api/instagram/auto-story` | POST | Auto story campaign |
| `POST /api/instagram/messages/send` | POST | Send DM |
| `POST /api/instagram/retargeting` | POST | Retargeting campaign |
| `POST /api/instagram/mention` | POST | Mention campaign (post_id, accounts, contacts, mentions_count, content_id, interval_id) |

**Vue:** 8 new views — InstagramHub, InstagramAccounts, InstagramSearch, InstagramExtract, InstagramAutoPost, InstagramMessaging, InstagramRetarget.

---

## 7-19. Remaining Features

| # | Feature | Whatomate | New Plugin? |
|---|---------|-----------|-------------|
| 7 | Campaign System | ✅ 75% exists | Extend existing |
| 8 | MLM System | ❌ 0% | `plugin/mlm/` |
| 9 | Wallet/Finance | ❌ 0% | `plugin/wallet/` |
| 10 | Points Economy | — | Replace with license tiers |
| 11 | Subscriptions | ✅ 100% | Use license system |
| 12 | E-Commerce | ❌ 0% | `plugin/ecommerce/` |
| 13 | Content Management | ~50% | `plugin/content/` |
| 14 | Contact Management | ✅ 100% | Reuse |
| 15 | Notifications | ✅ 100% | Reuse |
| 16 | File Management | ✅ 100% | Reuse |
| 17 | Chatbot/Flows | ✅ 100% | Reuse |
| 18 | Analytics | ✅ 60% | Extend |
| 19 | i18n | ✅ 100% | Reuse |
| 20 | Admin Dashboard | ✅ 80% | Extend |
| 21 | Proxy | — | Replace with fasthttp reverse proxy |

## Priority Summary

| Priority | Plugin | Reuse % | Effort |
|----------|--------|---------|--------|
| 🔴 P0 | Dashboard | 60% | 0.5 day |
| 🔴 P0 | Accounts | 40% | 1 day |
| 🔴 P0 | Campaigns | 75% | 1 day |
| 🟡 P1 | Facebook | 80% | 2 days |
| 🟡 P1 | WhatsApp | 90% | 0.5 day |
| 🟡 P1 | Content | 50% | 0.5 day |
| 🟠 P2 | Instagram | 0% | 3-5 days |
| 🟠 P2 | MLM | 0% | 2-3 days |
| 🟠 P2 | Wallet | 0% | 1-2 days |
| 🔵 P3 | E-Commerce | 0% | 3-5 days |
