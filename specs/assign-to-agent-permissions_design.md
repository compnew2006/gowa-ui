# Feature: Assign To Agent Permissions And Instance Visibility

## Requirements (EARS Format)
- While a user has chat assignment permission, when they assign a chat, the system shall authorize the action using role permissions.
- While an assignee lacks access to the contact's WhatsApp account, when a chat or transfer is assigned, the system shall reject the assignment.
- While the assign dialog is open, when the contact or transfer has an instance_id, the system shall only show agents allowed to access that instance.

## Architecture
- Frontend: Filter assignable agents by instance access (user settings send restrictions) in ChatView and AgentTransfersView; include instance_id on transfer models.
- Backend: Enforce assignment authorization via chat.assign:write (or contacts:write) and validate assignee instance access for contacts and transfers; expose instance_id in transfer responses.
- Security: Server-side authorization checks for assignment; deny assignment across instance restrictions; avoid relying solely on client filtering.

## Implementation Plan
- [ ] Backend: add instance_id to transfer responses and validate assignee instance access in transfer assignment.
- [ ] Backend: use chat.assign:write (or contacts:write) for contact assignment authorization.
- [ ] Frontend: filter assignable users/agents by allowed instance IDs in chat and transfers UI.
- [ ] Frontend: extend transfer model with instance_id.
- [ ] Tests: add/adjust unit or integration coverage for assignment permission and instance restriction enforcement.
