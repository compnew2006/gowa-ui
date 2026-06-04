# Facebook admin-reply filter (2026-06-04)

## Bug
Page-admin replies appeared as new incoming comments at `/facebook/comments` and triggered an auto-reply back to the page about the page's own message.

## Root cause
`IgnorePageAdminComments` was only consulted in `syncFacebookPageComments` skip path. Webhook path never read the setting and never identified the page author, so admin's own reply was ingested as `Direction: incoming` and `shouldAutoReplyFacebookComment` triggered public auto-reply.

## Fix
- Model: `FacebookComment.IsAdminReply bool` (default false, indexed). GORM AutoMigrate adds column.
- Helper: `isFacebookPageAdminCommenter(pageID, commenterID string) bool` in `internal/handlers/fb_comments.go` (trim + equal).
- Webhook: `upsertFacebookWebhookComment` tags `IsAdminReply` via `value.commenterID() == pageID`; `shouldAutoReplyFacebookComment` early-returns false; webhook auto-reply call site has `&& !comment.IsAdminReply` guard.
- Sync: `syncFacebookPageComments` no longer skips; tags via `actor.ID` / `edge.From.ID` (with `actorFallbacks[edge.ID]`); removed both `IgnorePageAdminComments` skip blocks.
- Response: `facebookCommentResponse` / `facebookCommentToResponse` expose `is_admin_reply` JSON.
- Frontend: `<Badge variant="secondary">` rendered next to author name in list and detail header. i18n key `facebookComments.adminReply` added to en/ar/es.
- Frontend test: `makeComment` factory includes `is_admin_reply: false`.
- Type: `frontend/src/types/facebookComments.ts` adds `is_admin_reply: boolean`.

## Latent bug surfaced
Webhook auto-reply was passing `uuid.Nil` for `userID`; FK `fk_facebook_comment_replies_user` rejected it. Fixed by passing `account.UserID` (page owner). Production auto-reply was silently failing every time before this change.

## Tests
- `TestApp_ReceiveFacebookCommentsWebhook_AdminReplyTaggedAndNotAutoReplied` (webhook, page-id matches, no auto-reply, IsAdminReply=true).
- `TestApp_ReceiveFacebookCommentsWebhook_NonAdminStillAutoReplies` (webhook, PSID != page-id, auto-reply runs, IsAdminReply=false).
- Both compute HMAC `X-Hub-Signature-256` over `req.RequestCtx.PostBody()` and set header so signature verification passes.
- 18 Facebook tests pass.

## Conventions followed
- `Direction: incoming` preserved for admin replies; `IsAdminReply` is orthogonal.
- Defense in depth: auto-reply blocked at both call site and in `shouldAutoReplyFacebookComment`.
- Tag at ingest (both webhook and sync) so all downstream code paths see the flag.
- No silent dropping — admin replies are saved and shown with a badge.

## Related memory
- `facebook/comments-reply-lookup-2026-06-03` — note that reply/status handlers accept internal `id` UUID or Graph `external_id`; this fix does not regress that.
