## Chat & Assignment Gaps Report — 2026-05-08

Comprehensive analysis of /chat and assignment systems. 87 gaps identified (8 CRITICAL, 25 HIGH, 35 MEDIUM, 19 LOW).

### Top Critical Issues
1. SEC-001: WS token replay via Sec-WebSocket-Protocol header
2. C1: WS broadcast silently drops messages (buffer=256)
3. FE-01: ChatView.vue is 6,193 lines (mega-component)
4. FE-02: 79% chat i18n missing in Spanish
5. MF-01: No DLQ for inbound messages

### Key Architecture Issues
- processIncomingMessageFull is 1000+ lines
- N+1 in ListContacts (per-contact hydration)
- Missing composite DB indexes on contacts/messages
- No cursor-based pagination
- No caching layer for chat data

### Key Assignment Issues
- Transfer expiration cleanup never runs
- Race condition in CreateAgentTransfer (no transaction)
- Round-robin LastAssignedAt not updated on self-pick
- No skill-based routing
- No push notifications for queue items

### Report Location
Full report: docs/chat-assign-gaps-report.md

### Agents Used
5 specialized agents: backend analyst, frontend analyst, assignment analyst, security reviewer, workflow tracer.