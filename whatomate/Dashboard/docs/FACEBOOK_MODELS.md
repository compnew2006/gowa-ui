# Kingmaster Facebook Data Models — Complete Reference

## 1. All Facebook-Related Tables

| # | Table | Rows | Purpose | Target Go GORM Model |
|---|-------|------|---------|---------------------|
| 1 | `accounts` | 596 | Multi-platform accounts (facebook channel) | `FacebookAccount` or shared `Account` |
| 2 | `fb_page` | 1,967 | Connected Facebook pages with tokens | `FacebookPage` |
| 3 | `fb_serch` | 38,394,474 | Facebook page search results | `FBPageSearchResult` |
| 4 | `data_fb` | 87,542,089 | Facebook people data (local cached DB) | `FBPeopleData` |
| 5 | `db_camp` | 202,874 | Campaign extracted people data | `FBCampaignData` |
| 6 | `groups_list` | 5,063 | Facebook group directory | `FacebookGroup` |
| 7 | `campaigns` | 5,011 | All platform campaigns (FB subset) | `FacebookCampaign` |
| 8 | `content` | 843 | Message templates (cross-platform) | `ContentTemplate` |
| 9 | `contacts` | 2,720 | Contact lists (platform='facebook') | `ContactList` |
| 10 | `sending_settings` | 351 | Sending intervals (platform='facebook') | `SendingSettings` |
| 11 | `posts` | 1 | Facebook posts / system updates | `FBPost` |
| 12 | `post_ratings` | 0 | Post ratings | `FBPostRating` |
| 13 | `messenger_templates` | 4 | Facebook Messenger templates | `FBMessengerTemplate` |

## 2. Detailed Model Mappings

### 2.1 Facebook Account → `FacebookAccount`
| PHP (accounts table) | Go GORM Field | Type | Notes |
|---------------------|---------------|------|-------|
| `id` int AI | `ID` | uuid.UUID | PK UUID |
| `user_id` | `OrganizationID` | uuid.UUID | Tenant-scoped |
| `name` | `Name` | string | |
| `account_uid` | `AccountUID` | string | Facebook user ID |
| `channel`='facebook' | `Platform` | string | 'facebook' constant |
| `status` | `Status` | FacebookAccountStatus | active/inactive/closed/expired/revoked |
| `method` | `Method` | FacebookAccountMethod | cookies/credentials/oauth |
| `cookies_text` | `CookiesText` | string (encrypted) | AES-256 encrypted |
| — | `AccessToken` | string (encrypted) | OAuth user token |
| — | `PageTokens` | JSONB | Page ID → token mapping |
| — | `TokenExpiresAt` | *time.Time | Token expiry tracking |
| `data` JSON | `Data` | JSONB | Additional metadata |

### 2.2 Facebook Page → `FacebookPage`
| PHP (fb_page table) | Go GORM Field | Type |
|---------------------|---------------|------|
| `id` int AI | `ID` | uuid.UUID |
| `user_id` | `OrganizationID` | uuid.UUID |
| `facebook_id` | `FacebookID` | string |
| `id_page` | `PageID` | string (indexed) |
| `name` | `Name` | string |
| `token` | — | Stored in `Account.PageTokens` (encrypted) |

### 2.3 Facebook Campaign → `FacebookCampaign`
| PHP (campaigns table) | Go GORM Field | Type |
|---------------------|---------------|------|
| `id` int AI | `ID` | uuid.UUID |
| `user_id` | `OrganizationID` | uuid.UUID |
| `campaign_id` | — | Use `ID` as unique identifier |
| `name` | `Name` | string |
| `status` | `Status` | CampaignStatus string (pending/running/paused/stopped/finished) |
| `tool` | `Tool` | string |
| `type_tools` (Extract/Send/Reply) | `Type` | string |
| `token` (JSON account IDs) | `AccountIDs` | []uuid.UUID |
| `pram1` | `Parameter` | *string |
| `range` | `Range` | int |
| `interval` | `IntervalID` | *uint |
| `content_id` | `ContentID` | *uint |
| `contact` | `ContactListID` | *uint |
| `speed` (slow/medium/fast) | `Speed` | string |
| `count` | `TotalCount` | int64 |
| `true_count` | `SuccessCount` | int64 |
| `false_count` | `FailureCount` | int64 |
| `page_url` | `TargetURL` | *string |

### 2.4 Facebook People Data → `FBPeopleData`
| PHP (data_fb table) | Go GORM Field | Type |
|---------------------|---------------|------|
| `id` int AI | `ID` | uint (keep int for 87M rows) |
| `fb_id` | `FacebookID` | string (indexed) |
| `name` | `Name` | string |
| `mobile_phone` | `MobilePhone` | string (indexed) |
| `gender` | `Gender` | string |
| `birthday` | `Birthday` | string |
| `location` | `Location` | string |
| `relationship` | `Relationship` | string |
| `email` | `Email` | string |
| `work` | `Work` | string |
| `education` | `Education` | string |

### 2.5 Page Search Result → `FBPageSearchResult`
| PHP (fb_serch table) | Go GORM Field | Type |
|---------------------|---------------|------|
| `id` int AI | `ID` | uint |
| `campaign_id` | `CampaignID` | string (indexed) |
| `page_id` | `PageID` | string |
| `name` | `Name` | string |
| `followers_count` (stored as string!) | `FollowersCount` | int64 |

### 2.6 Campaign Data → `FBCampaignData`
| PHP (db_camp table) | Go GORM Field | Type |
|---------------------|---------------|------|
| `id` int AI | `ID` | uint |
| `campaign_id` | `CampaignID` | string (indexed) |
| `fb_id` | `FacebookID` | string |
| `name` | `Name` | text |
| `phone` | `Phone` | string |
| `gender` | `Gender` | string |
| `birthday` | `Birthday` | string |
| `location` | `Location` | text |
| `relashan` | `Relation` | text |
| `email` | `Email` | text |
| `work` | `Work` | text |
| `educ` | `Education` | text |

### 2.7 Facebook Group → `FacebookGroup`
| PHP (groups_list table) | Go GORM Field | Type |
|------------------------|---------------|------|
| `id` int AI | `ID` | uuid.UUID |
| `groupId` | `GroupID` | string (indexed) |
| `groupName` | `Name` | string (full-text indexed) |
| `groupLink` | `Link` | *string |
| `country` | `Country` | *string |
| `Language` | `Language` | *string |
| `groupDesc` | `Description` | *string |
| `GroupImage` | `ImageURL` | *string |
| `categoryName` | `Category` | *string |

## 3. Non-Facebook Models (Target)

| Model | Domain | Tables |
|-------|--------|--------|
| `User` | Core | users |
| `WhatsAppAccount` | WhatsApp | accounts (channel='whatsapp') |
| `InstagramAccount` | Instagram | accounts (channel='instagram') |
| `Package` | Subscriptions | packages |
| `PackageFeature` | Subscriptions | (new, normalized) |
| `ContentTemplate` | Content | content, content_messages |
| `ContactList` | Contacts | contacts |
| `ContactListEntry` | Contacts | (new, normalized) |
| `SendingSettings` | Sending | sending_settings |
| `CampaignResult` | Campaigns | db_camp, fb_serch, gb_wa, ig_* |
| `Product` | E-commerce | products |
| `ProductColor` | E-commerce | product_colors |
| `ProductSize` | E-commerce | product_sizes |
| `Order` | E-commerce | orders |
| `OrderStatusHistory` | E-commerce | order_status_history |
| `Coupon` | E-commerce | coupons |
| `Wallet` | Finance | wallets, users_wallet |
| `Transaction` | Finance | transactions, wallet_transactions |
| `Withdrawal` | Finance | withdrawals |
| `MLMReferral` | MLM | mlm_referrals |
| `MLMCommission` | MLM | mlm_commissions |
| `MLMSetting` | MLM | mlm_settings |
| `CommissionWallet` | MLM | commission_wallets |
| `PointsPackage` | Points | points_packages |
| `PointUsage` | Points | point_use |
| `Notification` | System | notifications |
| `Announcement` | System | announcements |
| `ActivityLog` | System | logs |
| `File` | System | files |
| `InternalMessage` | System | messages |
| `Conversation` | System | conversations |
| `ChatbotFlow` | Chatbot | wa_flows |
| `MessengerTemplate` | Facebook | messenger_templates |
| `SalesTarget` | CRM | sales_target |
| `GroupDirectory` | Facebook | groups_list |

## 4. What Already Exists in Whatomate

### Models in `internal/models/`:
| File | Model | Status |
|------|-------|--------|
| `fb_account.go` | `FacebookAccount`, `FacebookOAuthState` | ✅ |
| `fb_comment.go` | `FacebookComment`, `FacebookCommentReply`, `FacebookCommentSettings`, `FacebookPageCommentSettings` | ✅ |
| `fb_page_search.go` | `FBPageSearch` | ✅ |
| `fb_people_search.go` | `FBPeopleSearch` | ✅ |

### Handlers in `internal/handlers/`:
| File | Purpose |
|------|---------|
| `fb_accounts.go` | Facebook account CRUD |
| `fb_oauth.go` | OAuth flow + page management |
| `fb_comments.go` | Comment sync/reply |
| `fb_page_search.go` | Page search |
| `fb_people_search.go` | People search |

### Vue Views in `frontend/src/views/`:
| View | Route |
|------|-------|
| `FacebookHubView.vue` | `/facebook` |
| `FacebookAccountsView.vue` | `/facebook/accounts` |
| `FacebookCommentsView.vue` | `/facebook/comments` |
| `PageSearchView.vue` | `/facebook/page-search` |
| `GroupSearchView.vue` | `/facebook/group-search` |
| `PeopleSearchView.vue` | `/facebook/people-search` |
| `PageMessengersView.vue` | `/facebook/page-messengers` |
| `RetargetingView.vue` | `/facebook/retargeting` |
| `AutoShareView.vue` | `/facebook/auto-share` |
| `ExtractLikesView.vue` | `/facebook/extract-likes` |
| `ExtractDataView.vue` | `/facebook/extract-data` |

## 5. Kingmaster-Only Features (Not Yet in Whatomate)

| Feature | Priority | Complexity | Notes |
|---------|----------|------------|-------|
| Campaign System (unified) | High | High | Different design from Whatomate |
| Sending Intervals/Speed | High | Medium | Per-platform config |
| Content Management | Medium | Low | Message templates |
| Post Creation | Medium | Low | Create FB posts |
| Post Rating | Medium | Low | Star ratings |
| Group Directory | Medium | Medium | Searchable curated groups |
| Page Messenger Extraction | Low | Medium | Similar to comments model |
| Data Extraction (87M rows) | Low | Very High | Partitioning needed |
| Points Economy | Low | High | Replace with license tiers |
| MLM Commission System | Medium | High | 4-level referral |
| E-commerce (Products/Orders) | Low | High | New plugin needed |
| Wallet System | Low | Medium | New plugin needed |
| Internal Messaging | Low | Low | |
| Chatbot Flow Builder | Low | Medium | Already exists in Whatomate |
| WhatsApp Polls/Lists | Low | Low | |
| Multi-language (i18n) | Low | Low | vue-i18n ready |
