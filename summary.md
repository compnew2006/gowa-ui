# Whatomate close-rating WhatsMeow send fix

## Task

Investigate why closing chat `34b44787-b6cb-4925-9095-af2b3ebce445` on `https://ofuqalmadenah.com` showed the Chat Close Rating message but failed to send with `failed to send text message: server returned error 400`, then fix and deploy the green version.

## Approach and key decisions

- Reproduced the issue in the internal browser on the target chat. The two old close-rating bubbles at 09:17 PM and 09:33 PM were failed with the WhatsMeow 400 error.
- Confirmed in production PostgreSQL that the failed rating rows were real provider-send failures, not UI-only failures.
- Compared successful and failed close-rating content. The prompt contains multilingual text, emoji, and a Google review link.
- Found that plain WhatsMeow text sends used the legacy `Conversation` protobuf field, while replies/status text already used `ExtendedTextMessage`.
- Changed plain WhatsMeow `SendText` to build an `ExtendedTextMessage` payload so rich text prompts use the safer WhatsApp text envelope.

## Files changed

- `pkg/whatsmeow/adapter_send.go`
  - `SendText` now sends `buildTextMessage(text)`.
  - `buildTextMessage` constructs an `ExtendedTextMessage`.
- `pkg/whatsmeow/adapter_send_test.go`
  - Added regression coverage proving close-rating-style text is preserved and no bare `Conversation` payload is used.
- `summary.md`
  - Updated this session record.

## Tests and verification

- Baseline before change: `go test ./pkg/whatsmeow` passed.
- Focused verification after change:
  - `go test ./pkg/whatsmeow` passed.
  - `go test ./internal/handlers ./pkg/whatsmeow` passed.
  - `go test ./...` passed.
- Frontend production build:
  - `cd frontend && npm run build` passed.
- Green deployment verification:
  - Active binary: `/opt/whatomate/bin/whatomate.green.20260522_190123`
  - Version: `Whatomate green-20260522_190123-ba27e62-close-rating-text`
  - Local login smoke passed on ports `18123`, `18124`, `18125`, `18126`.
  - License bootstrap on all four ports reported `enabled=True`, `status=active`, `locked=False`.
- Internal browser verification:
  - Reopened and closed chat `34b44787-b6cb-4925-9095-af2b3ebce445`.
  - New close-rating bubble at 10:12 PM rendered with no new `server returned error 400`.
  - Database row `837f9278-f749-440c-a454-0b4b2f1d564b` has status `delivered`, has a WhatsApp message ID, and no error message.
  - Journal log shows `Message sent` for `837f9278-f749-440c-a454-0b4b2f1d564b`.

## Deployment details

- Branch: `agent/fix-close-rating-whatsmeow-text`
- Commit: `ba27e62 Fix whatsmeow text payload for close ratings`
- Backup created before replacing green:
  - `/root/whatomate_backup_before_close_rating_20260522_190123`
- Blue binary was left untouched.
- Active symlink was corrected to the green slot:
  - `/opt/whatomate/bin/whatomate -> /opt/whatomate/bin/whatomate.green -> /opt/whatomate/bin/whatomate.green.20260522_190123`

## Known limitations

- The two old failed close-rating rows remain visible as historical failed messages. They were not edited or deleted.
- The failed-row Retry button does not resend while the chat is closed; the verified path was the actual reopen-and-close lifecycle.
- Existing unrelated local memory DB changes remain in the worktree.
