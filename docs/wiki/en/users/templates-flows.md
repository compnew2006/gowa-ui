---
title: Templates & Flows
---

# Templates & Flows

Manage WhatsApp message templates, interactive Flows, catalogs, and products. Sync your content with Meta's platform for approval and deployment.

> **Note:** Templates, Flows, and Catalogs require the **Meta (Cloud API)** provider. These features are not available when using WhatsMeow.

## Template Management

### List Templates

**Endpoint:** `GET /api/templates`

View all message templates for your organization:

```
GET /api/templates?status=approved&category=marketing&language=en
```

| Filter | Description |
|--------|-------------|
| **Status** | Filter by approval status: `approved`, `pending`, `rejected` |
| **Category** | Filter by category: `marketing`, `utility`, `authentication` |
| **Language** | Filter by language code |

Each template includes usage statistics showing how many times it has been sent.

### Create Template

**Endpoint:** `POST /api/templates`

```json
{
  "name": "order_confirmation",
  "category": "utility",
  "language": "en",
  "components": {
    "header": {
      "type": "text",
      "text": "Order Confirmation"
    },
    "body": {
      "text": "Hi {{1}}, your order {{2}} has been confirmed. Expected delivery: {{3}}."
    },
    "footer": {
      "text": "Thank you for your purchase!"
    },
    "buttons": [
      {
        "type": "quick_reply",
        "text": "Track Order"
      },
      {
        "type": "url",
        "text": "View Details",
        "url": "https://example.com/orders/{{2}}"
      }
    ]
  }
}
```

### Template Components

| Component | Description |
|-----------|-------------|
| **Header** | Optional header with text, image, video, or document |
| **Body** | Required message body with optional placeholders (`{{1}}`, `{{2}}`, etc.) |
| **Footer** | Optional footer text |
| **Buttons** | Quick reply, URL, or phone number buttons |

### Update Template

**Endpoint:** `PUT /api/templates/{id}`

Modify an existing template. Note that changes to approved templates may require re-submission to Meta for review.

### Delete Template

**Endpoint:** `DELETE /api/templates/{id}`

Remove a template from your local records and optionally from Meta.

### Submit Template for Approval

**Endpoint:** `POST /api/templates/{id}/publish`

Submit a template to Meta for review:

1. The template is validated for completeness
2. It is submitted to Meta's review system
3. Status changes to `pending`
4. You will be notified when Meta approves or rejects the template via webhook

### Sync Templates

**Endpoint:** `POST /api/templates/sync`

Synchronize templates between Whatomate and Meta:

1. Fetches all templates from the Meta API
2. Compares with local records
3. Creates new templates found on Meta but not locally
4. Updates existing templates with status changes
5. Removes local templates that no longer exist on Meta
6. Returns a summary of changes

### Upload Template Media

**Endpoint:** `POST /api/templates/upload-media`

Upload media files (images, videos, documents) for use in template headers:

1. The file is uploaded to Meta's media endpoint
2. Meta returns a media handle
3. Use the handle in your template's header component

## WhatsApp Flows

WhatsApp Flows enable structured, interactive experiences within WhatsApp conversations (e.g., appointment booking, feedback forms).

### List Flows

**Endpoint:** `GET /api/flows`

View all WhatsApp Flows for your organization.

### Create Flow

**Endpoint:** `POST /api/flows`

```json
{
  "name": "Appointment Booking",
  "categories": ["appointment_booking"],
  "json_payload": {
    "version": "3.0",
    "screens": [
      {
        "id": "SCREEN_1",
        "title": "Book Appointment",
        "terminal": true,
        "data": {},
        "layout": {
          "type": "SingleColumnLayout",
          "children": [
            {
              "type": "Form",
              "name": "appointment_form",
              "children": [
                {
                  "type": "TextInput",
                  "name": "service",
                  "label": "Service",
                  "required": true
                },
                {
                  "type": "DatePicker",
                  "name": "date",
                  "label": "Preferred Date",
                  "required": true
                }
              ]
            }
          ]
        }
      }
    ]
  }
}
```

### Update Flow

**Endpoint:** `PUT /api/flows/{id}`

Modify an existing Flow's JSON payload or metadata.

### Delete Flow

**Endpoint:** `DELETE /api/flows/{id}`

Remove a Flow from your records.

### Save Flow to Meta

**Endpoint:** `POST /api/flows/{id}/save-to-meta`

Push your Flow to Meta's platform:

1. The Flow JSON is validated
2. Meta's Flow API is called to create or update the Flow
3. The Meta Flow ID is stored locally

### Publish Flow

**Endpoint:** `POST /api/flows/{id}/publish`

Make a Flow available for use in messages and templates.

### Deprecate Flow

**Endpoint:** `POST /api/flows/{id}/deprecate`

Mark a Flow as deprecated. Existing conversations using the Flow will complete, but new conversations cannot start it.

### Duplicate Flow

**Endpoint:** `POST /api/flows/{id}/duplicate`

Create a copy of an existing Flow for modification without affecting the original.

### Sync Flows

**Endpoint:** `POST /api/flows/sync`

Synchronize Flows between Whatomate and Meta, similar to template sync.

## Catalogs & Products

Manage your product catalog for use in WhatsApp messages.

### List Catalogs

**Endpoint:** `GET /api/catalogs`

View all catalogs associated with your WhatsApp Business account.

### Create Catalog

**Endpoint:** `POST /api/catalogs`

Create a new product catalog on Meta.

### Delete Catalog

**Endpoint:** `DELETE /api/catalogs/{id}`

Remove a catalog.

### Sync Catalogs

**Endpoint:** `POST /api/catalogs/sync`

Synchronize catalogs between Whatomate and Meta.

### List Catalog Products

**Endpoint:** `GET /api/catalogs/{id}/products`

View all products within a specific catalog.

### Create Product

**Endpoint:** `POST /api/catalogs/{id}/products`

```json
{
  "name": "Premium Widget",
  "description": "High-quality widget for all your needs",
  "price": "29.99",
  "currency": "USD",
  "image_url": "https://example.com/widget.jpg",
  "url": "https://example.com/products/widget"
}
```

### Update Product

**Endpoint:** `PUT /api/products/{id}`

Modify product details including price, description, and images.

### Delete Product

**Endpoint:** `DELETE /api/products/{id}`

Remove a product from the catalog.

## Meta Sync

### Why Sync?

Meta is the source of truth for template approval status, Flow availability, and catalog data. Regular sync ensures your local records match Meta's platform.

### Sync Behavior

| Resource | Sync Action |
|----------|-------------|
| **Templates** | Fetch from Meta, create/update/delete local records |
| **Flows** | Fetch from Meta, sync Flow JSON and status |
| **Catalogs** | Fetch from Meta, sync products and pricing |

### Manual vs. Automatic Sync

- **Manual sync:** Triggered via the sync endpoints above
- **Automatic sync:** Template status updates are received via Meta webhooks in real time

## Template Placeholders

Templates support dynamic placeholders that are resolved at send time:

| Placeholder | Resolved From |
|-------------|---------------|
| `{{1}}`, `{{2}}`, etc. | Parameters provided when sending |
| `{{contact.name}}` | Contact name |
| `{{contact.phone}}` | Contact phone number |
| `{{user.name}}` | Agent/sender name |
| `{{organization.name}}` | Organization name |

## See Also

- [Campaigns](campaigns.md) — Using templates in bulk campaigns
- [Chat & Messaging](chat-messaging.md) — Sending template messages
- [Chatbot](chatbot.md) — Using templates in automated responses
- [Analytics](analytics.md) — Template usage statistics
