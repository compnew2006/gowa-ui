## 2026-02-18 22:50 Issue: "Unsupported message type" Investigation

- **The Trap:** Assuming "Unsupported message type" meant the message content itself (text "Mechanical") was invalid.
- **The Reality:** Standard text messages work fine. The error is triggered by `PollUpdateMessage` (e.g., voting for "Mechanical") or `SenderKeyDistributionMessage` falling through to default handling.
- **The Fix:** (Recommended) Implement `PollUpdateMessage` handling and silence `SenderKeyDistributionMessage`.
- **The Law:** Always verify complex interactions (like polls/voting) when users report specific content triggering errors.
