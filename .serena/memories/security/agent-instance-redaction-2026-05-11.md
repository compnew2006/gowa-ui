# Agent Instance Info Redaction Fix — 2026-05-11

## Bug
Agent-role users could see `instance_id` and `whatsapp_account` on public chats assigned to other agents via `/api/chats` (ListContacts) and `/api/contacts/:id` (GetContact). The SQL filter (`applyAgentVisibleChatAccessFilter`) allows agents to see public chats even when assigned to other agents, but the response previously included instance metadata for all visible chats.

## Fix
Added response-scrubbing step after ContactResponse construction in both `ListContacts` and `GetContact` handlers. When the requesting user is an agent-role user:
- Instance fields (`InstanceID`, `WhatsAppAccount`) are nil'd out on contacts where the agent is NOT the assignee AND NOT a collaborator AND the contact has an assignee (i.e., it's a public chat assigned to another agent).
- Instance fields remain visible on: own assignments, collaborator chats, and unassigned/pending chats.

## Files Changed
- `internal/handlers/chat_access_policy.go` — added `shouldRedactInstanceInfoForAgent(contact, userID, isCollaborator)` helper
- `internal/handlers/contacts.go` — `ListContacts` and `GetContact` now save `agentScope` flag and scrub instance fields before returning

## Pattern: Response-Scrubbing for Agent Scope
When adding field-level redaction for agent-role users:
1. Save `shouldRestrictChatVisibilityToAgentScope()` result to a local `agentScope` bool
2. After constructing response structs, check `agentScope && shouldRedactInstanceInfoForAgent(...)` 
3. Nil out sensitive fields on matching response items
4. Non-agent users (admin, super admin) are never affected — the `agentScope` flag gates all redaction

## Key Authorization Functions
- `shouldRestrictChatVisibilityToAgentScope(userID, orgID)` — returns true for agent-role users
- `applyAgentVisibleChatAccessFilter(query, userID)` — SQL filter for chat visibility
- `shouldRedactInstanceInfoForAgent(contact, userID, isCollaborator)` — field-level redaction decision
- `isContactAssignedToUser(contact, userID)` — checks if user is the assignee
- `listCollaboratorContactIDs(orgID, userID)` — batch collaborator check
