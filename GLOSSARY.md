# Gowa-UI

A WhatsApp customer-messaging workspace where an organization's agents handle conversations with customers across the organization's WhatsApp accounts.

## Language

### Who handles a conversation

**Routing**:
The system choosing which team should handle a conversation.
_Avoid_: Transfer, forwarding, dispatch

**Assignment**:
The system choosing which agent inside a team handles a conversation.
_Avoid_: Allocation, distribution

**Auto-assignment**:
Routing followed by Assignment, both done by the system without a person choosing.
_Avoid_: Auto-routing, smart transfer

**Default team**:
The one team that receives Routing for a WhatsApp account's conversations when no Decision says otherwise.
_Avoid_: Account team, owner team, fallback team

**Unassigned queue**:
The open conversations that have no assignee, which any eligible agent may Claim.
_Avoid_: Pending list, inbox, backlog

**Eligible agent**:
A team member who may receive an Assignment right now: active, able to read and reply to conversations, covering the conversation's WhatsApp account, available, online, and under their team's open-conversation cap.
_Avoid_: Available agent, free agent

**Transfer target**:
A colleague the current assignee may Transfer a conversation to; a looser rule than Eligible agent, and not interchangeable with it.
_Avoid_: Eligible agent, transfer candidate

**Claim**:
An agent taking an unassigned conversation for themself.
_Avoid_: Pick up, grab

**Transfer**:
The current assignee handing an open conversation to a colleague.
_Avoid_: Routing, reassignment, forwarding

### Automated decisions

**Decision**:
A typed answer (a choice, a score, or a yes/no probability) the system obtains about a conversation, together with how confident it is.
_Avoid_: Prediction, AI reply, classification result

**Shadow mode**:
A state where Decisions are recorded but not acted on, so they can be compared with what people actually did.
_Avoid_: Dry run, silent mode, test mode
