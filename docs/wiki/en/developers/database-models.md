---
title: Database Models
---

# Database Models

Whatomate uses GORM for ORM with PostgreSQL as the primary database. All models support soft-delete via GORM's `deleted_at` column.

## Soft-Delete Pattern

All models embed `gorm.Model` which provides `ID`, `CreatedAt`, `UpdatedAt`, and `DeletedAt` fields. Queries automatically exclude soft-deleted records unless `Unscoped()` is called.

```go
// Soft-delete
db.Delete(&user)          // Sets deleted_at = NOW()
db.Unscoped().Delete(&user) // Hard delete

// Query excludes soft-deleted by default
db.Find(&users)            // WHERE deleted_at IS NULL
db.Unscoped().Find(&users) // Includes soft-deleted
```

## Core Models

### User

Represents a user account in the system.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| Email | string | Unique email address |
| Password | string | Bcrypt-hashed password |
| FullName | string | Display name |
| IsActive | bool | Account active flag |
| OrganizationID | uint | Current active organization |
| RoleID | uint | Current role within organization |
| Settings | JSONB | Per-user settings (send restrictions, preferences) |
| DeletedAt | gorm.DeletedAt | Soft-delete timestamp |

```go
type User struct {
    gorm.Model
    Email          string `gorm:"uniqueIndex:idx_users_email_org;not null"`
    Password       string `gorm:"not null"`
    FullName       string
    IsActive       bool `gorm:"default:true"`
    OrganizationID uint
    RoleID         uint
    Settings       datatypes.JSON
    Role           CustomRole
}
```

**Relationships:**
- Belongs to `CustomRole` via `RoleID`
- Belongs to `Organization` via `OrganizationID`
- Has many `UserOrganization` memberships
- Has many `Contact` assignments (assigned_to)
- Has many `ConversationNote` entries

### Organization

Multi-tenant organization container.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| Name | string | Unique organization name |
| Settings | JSONB | Organization-level settings |
| StrictSendingRestrictionsEnabled | bool | Master send restriction toggle |
| OutboundMode | string | "inbound_only" or "mixed" |
| StrictRolloutMode | string | "audit" or "enforce" |
| CampaignDraftOnly | bool | Restrict campaigns to draft |
| DeletedAt | gorm.DeletedAt | Soft-delete timestamp |

```go
type Organization struct {
    gorm.Model
    Name                             string `gorm:"uniqueIndex;not null"`
    Settings                         datatypes.JSON
    StrictSendingRestrictionsEnabled bool
    OutboundMode                     string
    StrictRolloutMode                string
    StrictRolloutEnforceAt           *time.Time
    CampaignDraftOnly                bool
}
```

**Relationships:**
- Has many `User` records
- Has many `CustomRole` definitions
- Has many `WhatsAppAccount` records
- Has many `WhatsAppInstance` records
- Cascade soft-delete on organization deletion

### CustomRole

Custom role definitions within an organization.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| Name | string | Role name (unique within org) |
| OrganizationID | uint | Parent organization |
| IsDefault | bool | Default role for new users |
| IsSystem | bool | System role (admin, agent, manager) |
| DeletedAt | gorm.DeletedAt | Soft-delete timestamp |

```go
type CustomRole struct {
    gorm.Model
    Name           string `gorm:"uniqueIndex:idx_roles_name_org;not null"`
    OrganizationID uint   `gorm:"not null"`
    IsDefault      bool
    IsSystem       bool
    Permissions    []Permission `gorm:"foreignKey:RoleID"`
}
```

**Relationships:**
- Belongs to `Organization`
- Has many `Permission` records
- Has many `User` assignments

### Permission

Resource:action permission pairs attached to roles.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| RoleID | uint | Parent role |
| Resource | string | Resource name (users, contacts, messages, etc.) |
| Action | string | Action (read, write, delete, admin) |

```go
type Permission struct {
    gorm.Model
    RoleID   uint   `gorm:"not null;index"`
    Resource string `gorm:"not null"`
    Action   string `gorm:"not null"`
}
```

## WhatsApp Models

### WhatsAppAccount

Meta WhatsApp Business Cloud API account configuration.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| Name | string | Display name |
| PhoneNumberID | string | Encrypted Meta phone number ID |
| AccessToken | string | Encrypted Meta access token |
| BusinessAccountID | string | Encrypted business account ID |
| WebhookVerifyToken | string | Encrypted webhook verification token |
| OrganizationID | uint | Parent organization |
| Provider | string | "meta" |
| Status | string | Connection status |
| DeletedAt | gorm.DeletedAt | Soft-delete timestamp |

```go
type WhatsAppAccount struct {
    gorm.Model
    Name               string `gorm:"uniqueIndex:idx_accounts_name_org;not null"`
    PhoneNumberID      string // Encrypted with enc3: prefix
    AccessToken        string // Encrypted with enc3: prefix
    BusinessAccountID  string // Encrypted with enc3: prefix
    WebhookVerifyToken string // Encrypted with enc3: prefix
    OrganizationID     uint   `gorm:"not null;index"`
    Provider           string
    Status             string
}
```

**Relationships:**
- Belongs to `Organization`
- Has many `Message` records
- Has many `BulkMessageCampaign` records
- Has many `WhatsAppInstance` associations

### WhatsAppInstance

WhatsMeow direct WhatsApp Web instance.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| Name | string | Instance name (unique within org) |
| OrganizationID | uint | Parent organization |
| IsDefault | bool | Default instance for org |
| AutoReadReceipt | bool | Auto-send read receipts |
| Settings | JSONB | Instance-specific settings |
| Status | string | Connection status (disconnected, connecting, connected, qr, paired) |
| DeletedAt | gorm.DeletedAt | Soft-delete timestamp |

```go
type WhatsAppInstance struct {
    gorm.Model
    Name            string `gorm:"uniqueIndex:idx_instances_name_org;not null"`
    OrganizationID  uint   `gorm:"not null;index"`
    IsDefault       bool
    AutoReadReceipt bool
    Settings        datatypes.JSON
    Status          string
}
```

**Relationships:**
- Belongs to `Organization`
- Has many `Message` records
- Has many `BulkMessageCampaign` records

### Contact

WhatsApp contact / chat conversation.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| PhoneNumber | string | WhatsApp phone number |
| Name | string | Display name |
| ProfileName | string | WhatsApp profile name |
| OrganizationID | uint | Parent organization |
| AssignedUserID | uint | Assigned agent |
| WhatsAppAccountID | uint | Associated Meta account |
| WhatsAppInstanceID | uint | Associated WhatsMeow instance |
| Status | string | "open", "closed", "pending" |
| IsPublic | bool | Public chat visibility |
| ClosedAt | *time.Time | When chat was closed |
| ClosedByUserID | uint | User who closed the chat |
| LastMessageAt | *time.Time | Last message timestamp |
| UnreadCount | int | Unread message count |
| DeletedAt | gorm.DeletedAt | Soft-delete timestamp |

```go
type Contact struct {
    gorm.Model
    PhoneNumber        string     `gorm:"index:idx_contacts_phone_org"`
    Name               string
    ProfileName        string
    OrganizationID     uint       `gorm:"index"`
    AssignedUserID     uint
    WhatsAppAccountID  uint
    WhatsAppInstanceID uint
    Status             string
    IsPublic           bool
    ClosedAt           *time.Time
    ClosedByUserID     uint
    LastMessageAt      *time.Time
    UnreadCount        int
    Tags               []Tag            `gorm:"many2many:contact_tags"`
    AssignedUser       User
    Collaborators      []ContactCollaborator
}
```

**Relationships:**
- Belongs to `Organization`
- Belongs to `User` (assigned)
- Belongs to `WhatsAppAccount` or `WhatsAppInstance`
- Has many `Message` records
- Has many `ConversationNote` records
- Has many `Tag` via `contact_tags` join table
- Has many `ContactCollaborator` records

### Message

WhatsApp message record (inbound and outbound).

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| ContactID | uint | Associated contact |
| WhatsAppMessageID | string | Meta/WhatsApp message ID |
| Direction | string | "inbound" or "outbound" |
| Type | string | "text", "image", "video", "audio", "document", "template", "interactive", "location", "contact" |
| Content | string | Message text content |
| MediaURL | string | Local or remote media URL |
| MediaType | string | MIME type |
| Status | string | "pending", "sent", "delivered", "read", "failed" |
| ActorType | string | "user", "system", "chatbot" |
| ReplyToMessageID | uint | Parent message for replies |
| DeletedAt | gorm.DeletedAt | Soft-delete timestamp |

```go
type Message struct {
    gorm.Model
    ContactID          uint   `gorm:"index"`
    WhatsAppMessageID  string `gorm:"index"`
    Direction          string
    Type               string
    Content            string
    MediaURL           string
    MediaType          string
    Status             string
    ActorType          string
    ReplyToMessageID   uint
    OrganizationID     uint   `gorm:"index"`
    Contact            Contact
    ReplyTo            *Message
    Reactions          []MessageReaction
}
```

## Campaign Models

### BulkMessageCampaign

Campaign definition for bulk message sending.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| Name | string | Campaign name |
| OrganizationID | uint | Parent organization |
| WhatsAppAccountID | uint | Sending account |
| WhatsAppInstanceID | uint | Sending instance |
| TemplateID | uint | Message template |
| BodyContent | string | Template body with placeholders |
| HeaderMediaID | string | Header media handle |
| MinDelaySeconds | int | Minimum delay between messages |
| MaxDelaySeconds | int | Maximum delay between messages |
| Status | string | "draft", "running", "paused", "completed", "cancelled" |
| TotalRecipients | int | Total recipient count |
| SentCount | int | Successfully sent count |
| FailedCount | int | Failed send count |
| ScheduledAt | *time.Time | Scheduled start time |
| StartedAt | *time.Time | Actual start time |
| CompletedAt | *time.Time | Completion time |
| DeletedAt | gorm.DeletedAt | Soft-delete timestamp |

```go
type BulkMessageCampaign struct {
    gorm.Model
    Name               string     `gorm:"not null"`
    OrganizationID     uint       `gorm:"index"`
    WhatsAppAccountID  uint
    WhatsAppInstanceID uint
    TemplateID         uint
    BodyContent        string
    HeaderMediaID      string
    MinDelaySeconds    int        `gorm:"default:20"`
    MaxDelaySeconds    int        `gorm:"default:45"`
    Status             string     `gorm:"default:draft"`
    TotalRecipients    int
    SentCount          int
    FailedCount        int
    ScheduledAt        *time.Time
    StartedAt          *time.Time
    CompletedAt        *time.Time
    Template           Template
    Account            WhatsAppAccount
    Recipients         []BulkMessageRecipient
}
```

### BulkMessageRecipient

Individual recipient within a campaign.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| CampaignID | uint | Parent campaign |
| PhoneNumber | string | Recipient phone number |
| Name | string | Recipient name |
| Params | JSONB | Template parameters |
| Status | string | "pending", "sent", "delivered", "failed", "cancelled" |
| ErrorMessage | string | Failure reason |
| SentAt | *time.Time | When message was sent |

```go
type BulkMessageRecipient struct {
    gorm.Model
    CampaignID   uint           `gorm:"index"`
    PhoneNumber  string         `gorm:"not null"`
    Name         string
    Params       datatypes.JSON
    Status       string         `gorm:"default:pending;index"`
    ErrorMessage string
    SentAt       *time.Time
    Campaign     BulkMessageCampaign
}
```

## Chatbot Models

### Template

WhatsApp message template (local and Meta-synced).

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| Name | string | Template name |
| Category | string | Template category |
| Language | string | Template language code |
| Components | JSONB | Header, body, footer, buttons |
| Status | string | "approved", "pending", "rejected", "draft" |
| OrganizationID | uint | Parent organization |
| MetaTemplateID | string | Meta template ID |
| DeletedAt | gorm.DeletedAt | Soft-delete timestamp |

```go
type Template struct {
    gorm.Model
    Name           string         `gorm:"uniqueIndex:idx_templates_name_org_lang"`
    Category       string
    Language       string         `gorm:"default:en"`
    Components     datatypes.JSON
    Status         string         `gorm:"default:draft"`
    OrganizationID uint           `gorm:"index"`
    MetaTemplateID string
    Campaigns      []BulkMessageCampaign
}
```

### ChatbotSettings

Chatbot automation configuration per organization.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| Enabled | bool | Chatbot enabled flag |
| GreetingMessage | string | Auto-greeting text |
| FallbackMessage | string | No-match fallback text |
| SessionTimeoutMinutes | int | Session expiry |
| BusinessHours | JSONB | Per-day schedule |
| AIEnabled | bool | AI response enabled |
| AIProvider | string | AI provider name |
| AIModel | string | AI model identifier |
| AIAPIKey | string | Encrypted API key |
| AISystemPrompt | string | System prompt |
| AIMaxTokens | int | Max tokens per response |
| SLAResponseMinutes | int | Response SLA threshold |
| SLAResolutionMinutes | int | Resolution SLA threshold |
| SLAEscalationMinutes | int | Escalation SLA threshold |
| SLAAutoCloseHours | int | Auto-close threshold |

```go
type ChatbotSettings struct {
    gorm.Model
    OrganizationID        uint       `gorm:"uniqueIndex;not null"`
    Enabled               bool
    GreetingMessage       string
    FallbackMessage       string
    SessionTimeoutMinutes int        `gorm:"default:30"`
    BusinessHours         datatypes.JSON
    AIEnabled             bool
    AIProvider            string
    AIModel               string
    AIAPIKey              string     // Encrypted
    AISystemPrompt        string
    AIMaxTokens           int
    SLAResponseMinutes    int
    SLAResolutionMinutes  int
    SLAEscalationMinutes  int
    SLAAutoCloseHours     int
    SLAEscalationNotifyIDs datatypes.JSON
}
```

### KeywordRule

Chatbot keyword matching rules.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| Name | string | Rule name |
| Keywords | JSONB | Array of keywords |
| MatchType | string | "exact", "contains", "regex" |
| ResponseType | string | "text", "buttons", "flow" |
| ResponseContent | string | Encrypted response content |
| Priority | int | Match priority (lower = higher priority) |
| Enabled | bool | Rule enabled flag |

```go
type KeywordRule struct {
    gorm.Model
    Name            string         `gorm:"not null"`
    OrganizationID  uint           `gorm:"index"`
    Keywords        datatypes.JSON
    MatchType       string         `gorm:"default:contains"`
    ResponseType    string
    ResponseContent string         // Encrypted
    Priority        int            `gorm:"default:0"`
    Enabled         bool           `gorm:"default:true"`
}
```

### ChatbotFlow

Multi-step conversation flows.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| Name | string | Flow name |
| Description | string | Flow description |
| TriggerKeywords | JSONB | Trigger keyword array |
| Steps | JSONB | Flow step definitions |
| Enabled | bool | Flow enabled flag |

```go
type ChatbotFlow struct {
    gorm.Model
    Name            string         `gorm:"not null"`
    OrganizationID  uint           `gorm:"index"`
    Description     string
    TriggerKeywords datatypes.JSON
    Steps           datatypes.JSON
    Enabled         bool           `gorm:"default:true"`
}
```

### AIContext

Knowledge context for AI responses.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| Name | string | Context name |
| ContentType | string | "static", "dynamic", "url" |
| Content | string | Context content or URL |
| Keywords | JSONB | Trigger keywords |
| Priority | int | Context priority |
| Enabled | bool | Context enabled flag |

```go
type AIContext struct {
    gorm.Model
    Name           string         `gorm:"not null"`
    OrganizationID uint           `gorm:"index"`
    ContentType    string         `gorm:"default:static"`
    Content        string
    Keywords       datatypes.JSON
    Priority       int
    Enabled        bool           `gorm:"default:true"`
}
```

### AgentTransfer

Chatbot-to-human agent transfer requests.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| ContactID | uint | Associated contact |
| Reason | string | Transfer reason |
| Priority | int | Transfer priority |
| Status | string | "pending", "assigned", "completed", "cancelled" |
| AssignedUserID | uint | Assigned agent |
| ResolvedAt | *time.Time | Resolution timestamp |

```go
type AgentTransfer struct {
    gorm.Model
    ContactID      uint       `gorm:"index"`
    Reason         string
    Priority       int        `gorm:"default:0"`
    Status         string     `gorm:"default:pending"`
    AssignedUserID uint
    ResolvedAt     *time.Time
    Contact        Contact
    AssignedUser   User
}
```

## Productivity Models

### CannedResponse

Pre-written response templates.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| Shortcut | string | Quick-insert shortcut |
| Content | string | Response content |
| Category | string | Category label |
| MediaURL | string | Associated media |
| UsageCount | int | Times used |

```go
type CannedResponse struct {
    gorm.Model
    Shortcut       string `gorm:"uniqueIndex:idx_canned_shortcut_org;not null"`
    Content        string `gorm:"not null"`
    Category       string
    MediaURL       string
    UsageCount     int    `gorm:"default:0"`
    OrganizationID uint   `gorm:"index"`
}
```

### Tag

Contact tagging system.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| Name | string | Tag name (unique within org) |
| Color | string | Display color |
| OrganizationID | uint | Parent organization |

```go
type Tag struct {
    gorm.Model
    Name           string `gorm:"uniqueIndex:idx_tags_name_org;not null"`
    Color          string
    OrganizationID uint   `gorm:"index"`
    Contacts       []Contact `gorm:"many2many:contact_tags"`
}
```

### Team

User team groupings.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| Name | string | Team name |
| Description | string | Team description |
| OrganizationID | uint | Parent organization |

```go
type Team struct {
    gorm.Model
    Name           string `gorm:"not null"`
    Description    string
    OrganizationID uint   `gorm:"index"`
    Members        []TeamMember
}
```

### TeamMember

Team membership records.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| TeamID | uint | Parent team |
| UserID | uint | Team member |

```go
type TeamMember struct {
    gorm.Model
    TeamID uint `gorm:"index"`
    UserID uint `gorm:"index"`
    Team   Team
    User   User
}
```

## Integration Models

### Webhook

Outbound webhook endpoint configuration.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| URL | string | Webhook endpoint URL |
| Secret | string | Encrypted HMAC signing secret |
| Events | JSONB | Subscribed event types |
| Enabled | bool | Webhook enabled flag |
| LastTriggeredAt | *time.Time | Last delivery attempt |

```go
type Webhook struct {
    gorm.Model
    URL             string         `gorm:"not null"`
    Secret          string         // Encrypted
    Events          datatypes.JSON
    Enabled         bool           `gorm:"default:true"`
    OrganizationID  uint           `gorm:"index"`
    LastTriggeredAt *time.Time
}
```

### CustomAction

Custom HTTP action definitions for chatbot flows.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| Name | string | Action name |
| URL | string | Target endpoint |
| Method | string | HTTP method |
| Headers | JSONB | Encrypted request headers |
| BodyTemplate | string | Request body template |
| Events | JSONB | Trigger events |

```go
type CustomAction struct {
    gorm.Model
    Name           string         `gorm:"not null"`
    OrganizationID uint           `gorm:"index"`
    URL            string         `gorm:"not null"`
    Method         string         `gorm:"default:POST"`
    Headers        datatypes.JSON // Encrypted
    BodyTemplate   string
    Events         datatypes.JSON
}
```

## Audit & Communication Models

### ConversationNote

Private notes on contacts/chats.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| ContactID | uint | Associated contact |
| UserID | uint | Note author |
| Content | string | Note content |

```go
type ConversationNote struct {
    gorm.Model
    ContactID uint   `gorm:"index"`
    UserID    uint   `gorm:"index"`
    Content   string `gorm:"not null"`
    Contact   Contact
    User      User
}
```

### ActivityLog

Audit trail for significant actions.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| UserID | uint | Acting user |
| Action | string | Action type |
| Resource | string | Resource type |
| ResourceID | uint | Affected resource |
| Details | JSONB | Additional context |

```go
type ActivityLog struct {
    gorm.Model
    OrganizationID uint           `gorm:"index"`
    UserID         uint           `gorm:"index"`
    Action         string         `gorm:"not null"`
    Resource       string
    ResourceID     uint
    Details        datatypes.JSON
    User           User
}
```

### Widget

Custom analytics widget definitions.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| Name | string | Widget name |
| Type | string | Widget type |
| Query | JSONB | Data query configuration |
| Config | JSONB | Display configuration |

```go
type Widget struct {
    gorm.Model
    Name           string         `gorm:"not null"`
    Type           string
    Query          datatypes.JSON
    Config         datatypes.JSON
    OrganizationID uint           `gorm:"index"`
}
```

### LeadRequest

Public lead capture form submissions.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| WidgetID | uint | Source widget |
| Name | string | Lead name |
| Email | string | Lead email |
| Phone | string | Lead phone |
| Message | string | Lead message |
| Status | string | "new", "contacted", "converted", "rejected" |

```go
type LeadRequest struct {
    gorm.Model
    Name           string
    Email          string
    Phone          string
    Message        string
    Status         string         `gorm:"default:new"`
    OrganizationID uint           `gorm:"index"`
    WidgetID       uint
}
```

### Notification

In-app user notifications.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| UserID | uint | Target user |
| OrganizationID | uint | Parent organization |
| Type | string | Notification type |
| Title | string | Notification title |
| Message | string | Notification body |
| IsRead | bool | Read status |
| Data | JSONB | Additional payload |

```go
type Notification struct {
    gorm.Model
    UserID         uint           `gorm:"index"`
    OrganizationID uint           `gorm:"index"`
    Type           string
    Title          string
    Message        string
    IsRead         bool           `gorm:"default:false"`
    Data           datatypes.JSON
}
```

## Join & Association Models

### UserOrganization

User-to-organization membership records.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| UserID | uint | User |
| OrganizationID | uint | Organization |
| RoleID | uint | Role within this organization |

```go
type UserOrganization struct {
    gorm.Model
    UserID         uint `gorm:"uniqueIndex:idx_user_org_user_org"`
    OrganizationID uint `gorm:"uniqueIndex:idx_user_org_user_org"`
    RoleID         uint
    User           User
    Organization   Organization
    Role           CustomRole
}
```

### ContactCollaborator

Contact collaboration access records.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| ContactID | uint | Associated contact |
| UserID | uint | Collaborator user |
| Status | string | "pending", "accepted", "declined" |

```go
type ContactCollaborator struct {
    gorm.Model
    ContactID uint   `gorm:"uniqueIndex:idx_collab_contact_user"`
    UserID    uint   `gorm:"uniqueIndex:idx_collab_contact_user"`
    Status    string `gorm:"default:pending"`
    Contact   Contact
    User      User
}
```

### SSOProvider

SSO provider configurations.

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Primary key |
| OrganizationID | uint | Parent organization |
| Name | string | Provider name (google, azure, etc.) |
| ClientID | string | OAuth client ID |
| ClientSecret | string | Encrypted OAuth client secret |
| AuthorizationURL | string | OAuth authorization endpoint |
| TokenURL | string | OAuth token endpoint |
| UserInfoURL | string | User info endpoint |
| Scopes | JSONB | OAuth scopes |
| Enabled | bool | Provider enabled flag |

```go
type SSOProvider struct {
    gorm.Model
    Name             string         `gorm:"not null"`
    OrganizationID   uint           `gorm:"index"`
    ClientID         string         `gorm:"not null"`
    ClientSecret     string         // Encrypted
    AuthorizationURL string
    TokenURL         string
    UserInfoURL      string
    Scopes           datatypes.JSON
    Enabled          bool           `gorm:"default:true"`
    DisplayName      string
}
```

## Model Relationships Diagram

```
Organization (1) ────< User (N)
Organization (1) ────< CustomRole (N) ────< Permission (N)
Organization (1) ────< WhatsAppAccount (N)
Organization (1) ────< WhatsAppInstance (N)
Organization (1) ────< Contact (N) ────< Message (N)
Organization (1) ────< BulkMessageCampaign (N) ────< BulkMessageRecipient (N)
Organization (1) ────< Template (N)
Organization (1) ────< ChatbotSettings (1)
Organization (1) ────< KeywordRule (N)
Organization (1) ────< ChatbotFlow (N)
Organization (1) ────< AIContext (N)
Organization (1) ────< CannedResponse (N)
Organization (1) ────< Tag (N)
Organization (1) ────< Team (N) ────< TeamMember (N) ────> User
Organization (1) ────< Webhook (N)
Organization (1) ────< CustomAction (N)
Organization (1) ────< ConversationNote (N)
Organization (1) ────< ActivityLog (N)
Organization (1) ────< Widget (N)
Organization (1) ────< LeadRequest (N)
Organization (1) ────< Notification (N)
Organization (1) ────< SSOProvider (N)
User (N) ────< UserOrganization (N) ────> Organization
Contact (1) ────< ContactCollaborator (N) ────> User
Contact (1) ────< AgentTransfer (N)
Contact (N) ────< Tag (N) [many-to-many via contact_tags]
```

## See Also

- [Architecture Overview](architecture.md) — System architecture and component relationships
- [Caching System](caching.md) — Redis cache patterns for model data
- [API Reference](api-reference.md) — REST API endpoints for each model
