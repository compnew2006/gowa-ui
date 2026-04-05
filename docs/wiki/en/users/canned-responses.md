---
title: Canned Responses
---

# Canned Responses

Create and manage pre-written responses for quick replies during conversations. Canned responses save time, ensure consistency, and can include media attachments.

## Overview

Canned responses are reusable message templates that agents can insert into conversations with a single click or shortcut. Each response tracks its own usage count so you can identify the most helpful ones.

## Listing Canned Responses

**Endpoint:** `GET /api/canned-responses`

```
GET /api/canned-responses?category=billing&search=refund
```

| Filter | Description |
|--------|-------------|
| **Category** | Filter by response category |
| **Search** | Search by shortcut or content text |

Each response includes:

- Shortcut key for quick typing
- Content (the message text)
- Category for organization
- Usage count (how many times it has been sent)
- Optional media attachment

## Creating a Canned Response

**Endpoint:** `POST /api/canned-responses`

```json
{
  "shortcut": "refund_policy",
  "content": "Our refund policy allows returns within 30 days of purchase. Please provide your order number and we'll process your refund within 5-7 business days.",
  "category": "billing",
  "media_id": null
}
```

| Field | Required | Description |
|-------|----------|-------------|
| **shortcut** | Yes | Quick-typing shortcut (must be unique) |
| **content** | Yes | The message text to send |
| **category** | No | Organizational category |
| **media_id** | No | Optional media attachment ID |

**Validation:**

- Shortcuts must be unique within your organization
- Content cannot be empty
- Media attachments must reference valid, uploaded media

## Updating a Canned Response

**Endpoint:** `PUT /api/canned-responses/{id}`

Modify any field of an existing canned response:

```json
{
  "content": "Updated refund policy text...",
  "category": "billing-updates"
}
```

## Deleting a Canned Response

**Endpoint:** `DELETE /api/canned-responses/{id}`

Remove a canned response permanently. This action cannot be undone.

## Sending a Canned Response

**Endpoint:** `POST /api/canned-responses/{id}/send`

Send a canned response directly to a contact:

```json
{
  "contact_id": "contact-uuid"
}
```

**What happens:**

1. The canned response content and media are loaded
2. A message is built and routed through the standard send pipeline
3. The message is sent via WhatsApp (Meta or WhatsMeow)
4. The usage count is incremented automatically

This is equivalent to typing the content manually, but faster and with consistent formatting.

## Tracking Usage

**Endpoint:** `POST /api/canned-responses/{id}/use`

Manually increment the usage counter. This is called automatically when a canned response is sent, but can also be triggered manually for tracking purposes.

### Usage Statistics

The usage count helps identify:

- **Most-used responses** — Popular responses that agents rely on
- **Unused responses** — Candidates for cleanup or revision
- **Trending responses** — Responses seeing increased usage (may indicate a common customer issue)

## Media Attachments

Canned responses can include media files:

| Media Type | Description |
|------------|-------------|
| **Image** | JPEG, PNG, WebP |
| **Video** | MP4, 3GP |
| **Audio** | OGG, MP3 |
| **Document** | PDF, DOC, DOCX, XLS, XLSX |

### Uploading Media for Canned Responses

**Endpoint:** `POST /api/canned-responses/media`

Upload a media file and receive a media ID to attach to a canned response:

1. Upload the file via multipart form data
2. The file is stored and validated
3. A media ID is returned
4. Use this ID in the `media_id` field when creating or updating a canned response

### Serving Attached Media

When a canned response with media is sent, the media is served through the standard media endpoint:

**Endpoint:** `GET /api/media/{message_id}`

## Organizing with Categories

Use categories to group canned responses by topic:

| Example Categories | Use Case |
|--------------------|----------|
| `billing` | Refund policies, payment methods, invoices |
| `support` | Troubleshooting steps, FAQ answers |
| `sales` | Product information, pricing, promotions |
| `greeting` | Welcome messages, sign-offs |
| `escalation` | Messages for transferring to a manager |

## Best Practices

1. **Keep shortcuts memorable** — Use descriptive names like `refund_policy` rather than `rp1`
2. **Use categories consistently** — Establish a category naming convention for your team
3. **Review usage regularly** — Remove or update responses that are no longer relevant
4. **Include placeholders** — Use template placeholders like `{{contact.name}}` for personalization
5. **Test before sharing** — Send new canned responses to yourself to verify formatting

## See Also

- [Chat & Messaging](chat-messaging.md) — Sending messages manually
- [Templates & Flows](templates-flows.md) — WhatsApp message templates
- [Teams & Roles](teams-roles.md) — Managing team access to canned responses
- [Analytics](analytics.md) — Tracking response usage metrics
