# Diagnostic Report: Group Message "Unsupported Message Type"

## Issue Summary

Messages sent to group `120363404563821424@g.us` containing "Mechanical" are displaying as "[Unsupported message type]".

## Root Cause Analysis

Investigation reveals that the "Unsupported message type" error is triggered by specific message types that are not fully handled by the `extractMessageContentWithMedia` reference implementation in `pkg/whatsmeow`.

### Findings:

1.  **PollUpdateMessage (Voting)**: Testing confirmed that `PollUpdateMessage` (when a user votes on a poll) returns `[Unsupported message type]` instead of a descriptive message or the vote content.
    - **Hypothesis**: The user's "message containing 'Mechanical'" is likely a **vote** on a poll option named "Mechanical". The system receives the update, fails to parse/render it as a readable timeline item, and defaults to "Unsupported".
2.  **SenderKeyDistributionMessage**: This administrative message (used for group encryption) also triggers `[Unsupported message type]`. While this is less likely to be described as "containing 'Mechanical'" by a user, it contributes to the frequency of this error in group chats.
3.  **Standard Text Types OK**: `Conversation` (simple text), `ExtendedTextMessage`, `ButtonsMessage`, `ListMessage`, `InteractiveMessage`, and `TemplateMessage` correctly extract their content.

## Technical Details

- **File**: `pkg/whatsmeow/incoming_media.go` (and `message_persist.go` logic).
- **Function**: `extractMessageContentwithId`.
- **Logic**: The switch case for message types falls through to `default` for types like `PollUpdateMessage` and `SenderKeyDistributionMessage`.
- **Frontend**: `frontend/src/stores/contacts.ts` defines the string `[Unsupported message type]`, but the backend explicitly returns this string in the default case (verified by test).

## Recommendations (For Future Fix)

1.  **Implement Poll Update Handling**: Update `extractMessageContentWithMedia` to handle `PollUpdateMessage`. Since votes are encrypted (`EncPayload`), they might need decryption or simply be ignored/hidden if not supported in the UI.
2.  **Silence Administrative Messages**: Ensure `SenderKeyDistributionMessage` returns an empty string or a special internal type that the frontend ignores.
3.  **Frontend Graceful Fallback**: If a message type is unsupported, the frontend should ideally show a "Update App to view this message" or similar, or just hide it if it's a technical message.

## Reproduction

A reproduction test script has been archived at `.repro_archive/2026-02-18_issue_investigation.go`. It demonstrates that `PollUpdateMessage` triggers the error.
