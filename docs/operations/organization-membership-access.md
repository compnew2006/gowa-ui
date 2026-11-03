# Organization membership and access

Authenticated API requests verify that the user is active and has a current
membership in the organization on the access token or API key. Super admins
can access organizations without a membership. A removed member's existing
access token and organization-scoped API key are rejected on their next API
request; their access to other organizations is unaffected.

Organization-scoped permission lookups also check current membership before
using cached permissions. A missing membership or database error fails closed.
Issuing a WebSocket token and authenticating a new WebSocket connection use the
same membership check.

Existing WebSocket connections are not disconnected by this change. In a
multi-instance deployment, membership removal needs a shared revocation and
disconnect mechanism to terminate already authenticated connections across
instances. Until then, do not treat removal as immediate revocation of an open
WebSocket session. This change does not add that mechanism.

Before rollout, check for active users whose home organization has no active
`user_organizations` row. Such users will be rejected until their membership
is repaired. Test super-admin and cross-organization access with the actual
deployment configuration. The additional membership checks query PostgreSQL;
monitor authentication latency and database load after deployment.

The remaining reliability work identified in the engineering review, including
webhook processing errors, pending Redis jobs, and campaign send reconciliation,
is separate and is not addressed by this change.
