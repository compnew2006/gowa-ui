package whatsmeow

import (
	"context"

	"github.com/compnew2006/whatomate/pkg/provider"
	"go.mau.fi/whatsmeow"
	waTypes "go.mau.fi/whatsmeow/types"
)

// sessionStoreClient is the narrow whatsmeow client surface that
// clearRecipientSessions needs. *whatsmeow.Client satisfies it via its Store
// field; tests supply a fake. Keeping it internal avoids exporting adapter
// internals while making the reset logic unit-testable without a live client.
type sessionStoreClient interface {
	storeSessions() (sessions sessionStore, lids lidStore, ownUser string)
}

// sessionStore mirrors the subset of whatsmeow's store.SessionStore used here.
type sessionStore interface {
	DeleteAllSessions(ctx context.Context, phone string) error
}

// lidStore mirrors the subset of whatsmeow's store.LIDStore used here.
type lidStore interface {
	GetLIDForPN(ctx context.Context, pn waTypes.JID) (waTypes.JID, error)
	GetPNForLID(ctx context.Context, lid waTypes.JID) (waTypes.JID, error)
}

// realClientStore adapts *whatsmeow.Client to sessionStoreClient.
type realClientStore struct{ client *whatsmeow.Client }

func (r realClientStore) storeSessions() (sessionStore, lidStore, string) {
	if r.client == nil || r.client.Store == nil {
		return nil, nil, ""
	}
	ownUser := ""
	if r.client.Store.ID != nil {
		ownUser = r.client.Store.ID.User
	}
	return r.client.Store.Sessions, r.client.Store.LIDs, ownUser
}

// ResetRecipientSession clears the Signal Protocol sessions for a single
// recipient so the next outbound message rebuilds them from a fresh prekey
// exchange instead of reusing a stale/desynced session.
//
// Why this exists: when WhatsApp migrates a recipient from a phone-number JID
// (PN, @s.whatsapp.net) to a Linked Identity (LID, @lid), whatsmeow tries to
// migrate the existing Signal session. If no session exists to migrate
// (recipient reinstalled, changed devices, or the local PN<->LID mapping
// desynced), the server acks the send with a "400 bad request" stanza error,
// surfaced as "failed to send X message: server returned error 400". Retrying
// with the same stale local store fails identically every time, so the send
// queue calls this once before the first retry to force a clean rebuild.
//
// Scope: per-recipient only. It deletes all device sessions for the target
// user and, when resolvable, its PN<->LID counterpart. It never wipes the
// whole instance store, so other chats are unaffected. Group sender keys are
// left in place (SenderKeyStore has no Delete in this whatsmeow version); 1:1
// DM sends, which are the documented failure case, only need the session store.
//
// All errors are logged and swallowed: a reset failure must not strand the
// retry loop. The next send attempt proceeds regardless and may still succeed
// via the on-the-fly prekey path.
//
// This method implements provider.SessionResetter.
func (a *WhatsmeowAdapter) ResetRecipientSession(ctx context.Context, instanceID string, to string) error {
	client, jid, err := a.resolveClientAndJID(ctx, instanceID, to)
	if err != nil {
		// Resolve failures are not fatal for the retry path; the send itself
		// will surface a proper error if the client is truly unavailable.
		a.logger.Warn(
			"session reset: skipping, could not resolve client/JID",
			"instance_id", instanceID,
			"to", to,
			"error", err,
		)
		return nil
	}
	a.clearRecipientSessions(ctx, realClientStore{client: client}, jid)
	return nil
}

// clearRecipientSessions deletes the Signal sessions for jid's user and its
// PN<->LID counterpart (if known). It is safe to call on any JID type; for
// group/broadcast JIDs it is a no-op because those use sender keys, not
// 1:1 sessions, and the 400 failure mode is specific to 1:1 DMs.
func (a *WhatsmeowAdapter) clearRecipientSessions(ctx context.Context, c sessionStoreClient, jid waTypes.JID) {
	sessions, lids, ownUser := c.storeSessions()
	if sessions == nil {
		return
	}
	// Only 1:1 user JIDs carry Signal sessions worth resetting. Groups,
	// broadcasts, and newsletters use sender keys/app-state keys instead.
	if jid.Server != waTypes.DefaultUserServer && jid.Server != waTypes.HiddenUserServer {
		return
	}

	targets := recipientResetTargets(ctx, lids, jid)
	for _, t := range targets {
		phone := t.SignalAddressUser()
		if phone == "" {
			continue
		}
		// DeleteAllSessions matches "<phone>:%", wiping every device session
		// for that user so the next send does a full prekey rebuild.
		if err := sessions.DeleteAllSessions(ctx, phone); err != nil {
			a.logger.Warn(
				"session reset: failed to delete sessions",
				"instance_id", ownUser,
				"recipient_user", phone,
				"error", err,
			)
		} else {
			a.logger.Info(
				"session reset: cleared recipient sessions",
				"instance_id", ownUser,
				"recipient_user", phone,
			)
		}
	}
}

// recipientResetTargets returns the set of users whose sessions should be
// cleared: always the resolved JID's user, plus its PN<->LID counterpart when
// the mapping is known locally. Each returned JID is normalized with ToNonAD
// so companion-device suffixes don't fragment the deletion.
//
// The LID lookup is best-effort: if the LID store is missing, returns an error,
// or panics (e.g. a typed-nil pointer behind a non-nil interface — a known
// footgun for interface fields), the counterpart is simply skipped. A reset
// that clears only the primary identity is still strictly better than none.
func recipientResetTargets(ctx context.Context, lids lidStore, jid waTypes.JID) []waTypes.JID {
	targets := []waTypes.JID{jid.ToNonAD()}
	counterpart := lookupCounterpartSafe(ctx, lids, jid)
	if !counterpart.IsEmpty() {
		targets = append(targets, counterpart.ToNonAD())
	}
	return dedupeResetTargets(targets)
}

// lookupCounterpartSafe resolves the PN<->LID counterpart of jid, swallowing
// errors and panics from a broken/nil LID store. Returns an empty JID when the
// counterpart is unknown or the lookup cannot be performed.
func lookupCounterpartSafe(ctx context.Context, lids lidStore, jid waTypes.JID) (counterpart waTypes.JID) {
	if lids == nil {
		return waTypes.EmptyJID
	}
	// Guard against a typed-nil pointer behind the interface (would panic on
	// the method call). Defer/recover turns any such panic into a no-op.
	defer func() {
		if r := recover(); r != nil {
			counterpart = waTypes.EmptyJID
		}
	}()
	switch jid.Server {
	case waTypes.DefaultUserServer:
		if lid, err := lids.GetLIDForPN(ctx, jid); err == nil {
			return lid
		}
	case waTypes.HiddenUserServer:
		if pn, err := lids.GetPNForLID(ctx, jid); err == nil {
			return pn
		}
	}
	return waTypes.EmptyJID
}

// dedupeResetTargets drops empty/duplicate users so DeleteAllSessions isn't
// invoked twice for the same identity (e.g. when PN and LID resolve to the
// same SignalAddressUser, which can happen for non-migrated recipients).
func dedupeResetTargets(targets []waTypes.JID) []waTypes.JID {
	seen := make(map[string]struct{}, len(targets))
	out := targets[:0]
	for _, t := range targets {
		if t.IsEmpty() {
			continue
		}
		key := t.SignalAddressUser()
		if key == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, t)
	}
	return out
}

// Compile-time assertion that WhatsmeowAdapter satisfies SessionResetter.
var _ provider.SessionResetter = (*WhatsmeowAdapter)(nil)
