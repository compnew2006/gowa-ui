package audit

import (
	"context"
	"fmt"

	"github.com/compnew2006/whatomate/internal/middleware"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
)

// EventBuilder is a fluent constructor for AuditEvent. Obtain one via NewEvent.
type EventBuilder struct {
	e models.AuditEvent
}

// NewEvent starts a builder for the given action. Category is inferred from
// the Action* constant via actionCategory, so call sites specify only the action.
func NewEvent(action string) *EventBuilder {
	return &EventBuilder{e: models.AuditEvent{
		Action:   action,
		Category: actionCategory(action),
		Source:   SourceUser, // default; override with ActorSystem for system events
		Success:  true,
		Details:  models.JSONB{},
	}}
}

// Category overrides the inferred category (rarely needed).
func (b *EventBuilder) Category(c string) *EventBuilder { b.e.Category = c; return b }

// Org sets the tenant scope from a nullable org ID.
func (b *EventBuilder) Org(id *uuid.UUID) *EventBuilder { b.e.OrganizationID = id; return b }

// OrgValue sets the tenant scope from a non-nil org ID.
func (b *EventBuilder) OrgValue(id uuid.UUID) *EventBuilder {
	b.e.OrganizationID = &id
	return b
}

// ActorFromRequest pulls actor identity + request origin from a fastglue
// request context. Safe to call on requests without a user (no-op for missing fields).
func (b *EventBuilder) ActorFromRequest(r *fastglue.Request) *EventBuilder {
	if r == nil {
		return b
	}
	if uid, ok := middleware.GetUserID(r); ok {
		b.e.ActorUserID = &uid
	}
	if u, ok := middleware.GetUser(r); ok && u != nil {
		b.e.ActorEmail = u.Email
		if u.Role != nil {
			b.e.ActorRole = u.Role.Name
		}
	}
	b.e.IPAddress = clientIP(r)
	b.e.UserAgent = string(r.RequestCtx.UserAgent())
	b.e.Source = SourceUser
	return b
}

// ActorSystem marks a system-originated event (no human actor). componentName
// is echoed into ActorEmail for traceability (e.g. "worker", "scheduler").
func (b *EventBuilder) ActorSystem(componentName string) *EventBuilder {
	b.e.Source = SourceSystem
	b.e.ActorUserID = nil
	b.e.ActorEmail = componentName
	return b
}

// Target sets the action target. id is stringified to handle UUIDs, numeric IDs,
// and JIDs uniformly.
func (b *EventBuilder) Target(typ string, id any) *EventBuilder {
	b.e.TargetType = typ
	if id == nil {
		return b
	}
	s := toString(id)
	b.e.TargetID = &s
	return b
}

// Success sets the outcome.
func (b *EventBuilder) Success(v bool) *EventBuilder { b.e.Success = v; return b }

// Reason sets a short failure/extra reason.
func (b *EventBuilder) Reason(s string) *EventBuilder { b.e.Reason = s; return b }

// Detail merges a key/value into the JSONB Details without clobbering other keys.
func (b *EventBuilder) Detail(k string, v any) *EventBuilder {
	if b.e.Details == nil {
		b.e.Details = models.JSONB{}
	}
	b.e.Details[k] = v
	return b
}

// Build returns the immutable event.
func (b *EventBuilder) Build() models.AuditEvent { return b.e }

// Record builds and records the event in one call. svc may be nil (no-op).
func (b *EventBuilder) Record(ctx context.Context, svc *Service) {
	if svc == nil {
		return
	}
	svc.Record(ctx, b.e)
}

// clientIP extracts the peer IP from a fastglue request. It honors the
// X-Forwarded-For first hop when present (matches the project's real-client-IP
// logging convention). Kept dependency-light rather than importing a handlers helper.
func clientIP(r *fastglue.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.RequestCtx.Request.Header.Peek("X-Forwarded-For"); len(xff) > 0 {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return string(xff[:i])
			}
		}
		return string(xff)
	}
	return r.RequestCtx.RemoteIP().String()
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case uuid.UUID:
		return x.String()
	case *uuid.UUID:
		if x == nil {
			return ""
		}
		return x.String()
	default:
		return fmt.Sprint(v)
	}
}
