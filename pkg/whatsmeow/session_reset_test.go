package whatsmeow

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	waTypes "go.mau.fi/whatsmeow/types"
	"github.com/zerodha/logf"
)

// fakeSessionStore records the phones whose sessions were deleted.
type fakeSessionStore struct {
	deleted []string
	err     error
}

func (f *fakeSessionStore) DeleteAllSessions(_ context.Context, phone string) error {
	f.deleted = append(f.deleted, phone)
	return f.err
}

// fakeLIDStore returns canned PN<->LID mappings.
type fakeLIDStore struct {
	lidForPN waTypes.JID
	pnForLID waTypes.JID
	lidErr   error
	pnErr    error
}

func (f *fakeLIDStore) GetLIDForPN(_ context.Context, _ waTypes.JID) (waTypes.JID, error) {
	return f.lidForPN, f.lidErr
}

func (f *fakeLIDStore) GetPNForLID(_ context.Context, _ waTypes.JID) (waTypes.JID, error) {
	return f.pnForLID, f.pnErr
}

// fakeResetClient adapts fakes to sessionStoreClient. The session/lid fields
// are interfaces so that "unset" produces a genuinely nil interface (not a
// typed-nil pointer), matching how the real *store.Device exposes nil stores.
type fakeResetClient struct {
	sessions sessionStore
	lids     lidStore
	ownUser  string
}

func (f fakeResetClient) storeSessions() (sessionStore, lidStore, string) {
	return f.sessions, f.lids, f.ownUser
}

func newTestAdapter(t *testing.T) *WhatsmeowAdapter {
	t.Helper()
	return &WhatsmeowAdapter{logger: logf.New(logf.Opts{})}
}

func TestRecipientResetTargetsPNResolvesLID(t *testing.T) {
	pn := waTypes.NewJID("966500000000", waTypes.DefaultUserServer)
	lid := waTypes.NewJID("54611129962560", waTypes.HiddenUserServer)
	lids := &fakeLIDStore{lidForPN: lid}

	got := recipientResetTargets(context.Background(), lids, pn)

	// PN always included; LID counterpart appended; both non-AD normalized.
	// Note: whatsmeow assigns agent=1 to LID identities, so the LID's Signal
	// address user is "<user>_1" — the form session rows are keyed by.
	require.Len(t, got, 2)
	assert.Equal(t, "966500000000", got[0].SignalAddressUser())
	assert.Equal(t, "54611129962560_1", got[1].SignalAddressUser())
}

func TestRecipientResetTargetsLIDResolvesPN(t *testing.T) {
	pn := waTypes.NewJID("966500000000", waTypes.DefaultUserServer)
	lid := waTypes.NewJID("54611129962560", waTypes.HiddenUserServer)
	lids := &fakeLIDStore{pnForLID: pn}

	got := recipientResetTargets(context.Background(), lids, lid)

	require.Len(t, got, 2)
	assert.Equal(t, "54611129962560_1", got[0].SignalAddressUser())
	assert.Equal(t, "966500000000", got[1].SignalAddressUser())
}

func TestRecipientResetTargetsNoLIDStore(t *testing.T) {
	pn := waTypes.NewJID("966500000000", waTypes.DefaultUserServer)

	got := recipientResetTargets(context.Background(), nil, pn)

	require.Len(t, got, 1)
	assert.Equal(t, "966500000000", got[0].SignalAddressUser())
}

func TestRecipientResetTargetsDedupesWhenMappingMissing(t *testing.T) {
	// LID resolution returns empty/err -> only the PN target remains.
	pn := waTypes.NewJID("966500000000", waTypes.DefaultUserServer)
	lids := &fakeLIDStore{lidErr: errors.New("not found")}

	got := recipientResetTargets(context.Background(), lids, pn)

	require.Len(t, got, 1)
}

func TestClearRecipientSessionsDeletesPNAndLIDPair(t *testing.T) {
	pn := waTypes.NewJID("966500000000", waTypes.DefaultUserServer)
	lid := waTypes.NewJID("54611129962560", waTypes.HiddenUserServer)
	store := &fakeSessionStore{}
	client := fakeResetClient{
		sessions: store,
		lids:     &fakeLIDStore{lidForPN: lid},
		ownUser:  "myinstance",
	}

	newTestAdapter(t).clearRecipientSessions(context.Background(), client, pn)

	// Both the PN user and its LID counterpart must be cleared. The LID is
	// agent-suffixed (whatsmeow assigns agent=1 to LIDs), matching how session
	// rows are keyed and how DeleteAllSessions wildcards them.
	assert.ElementsMatch(t, []string{"966500000000", "54611129962560_1"}, store.deleted)
}

func TestClearRecipientSessionsSkipsGroupJIDs(t *testing.T) {
	group := waTypes.NewJID("120363000000000000", waTypes.GroupServer)
	store := &fakeSessionStore{}
	client := fakeResetClient{sessions: store}

	newTestAdapter(t).clearRecipientSessions(context.Background(), client, group)

	assert.Empty(t, store.deleted, "group JIDs use sender keys, sessions must not be touched")
}

func TestClearRecipientSessionsNilStoreIsNoOp(t *testing.T) {
	pn := waTypes.NewJID("966500000000", waTypes.DefaultUserServer)
	// sessions == nil -> helper returns immediately, never reaches LID lookup.
	client := fakeResetClient{sessions: nil, lids: nil}

	assert.NotPanics(t, func() {
		newTestAdapter(t).clearRecipientSessions(context.Background(), client, pn)
	})
}

func TestClearRecipientSessionsSwallowsStoreError(t *testing.T) {
	pn := waTypes.NewJID("966500000000", waTypes.DefaultUserServer)
	store := &fakeSessionStore{err: errors.New("disk full")}
	// Provide a real (non-typed-nil) LID store so the reset reaches the
	// DeleteAllSessions call, which then returns the injected error.
	client := fakeResetClient{sessions: store, lids: &fakeLIDStore{}}

	assert.NotPanics(t, func() {
		newTestAdapter(t).clearRecipientSessions(context.Background(), client, pn)
	})
	// The failed call still recorded the attempt; ensure it didn't panic and
	// surfaced exactly one target.
	assert.Len(t, store.deleted, 1)
}

func TestClearRecipientSessionsHandlesBrokenLIDStore(t *testing.T) {
	// A typed-nil pointer behind a non-nil interface must not panic; the reset
	// proceeds for the primary identity only.
	pn := waTypes.NewJID("966500000000", waTypes.DefaultUserServer)
	store := &fakeSessionStore{}
	var brokenLID lidStore = (*fakeLIDStore)(nil)
	client := fakeResetClient{sessions: store, lids: &fakeLIDStore{}}

	assert.NotPanics(t, func() {
		// Directly exercise the panic-safe counterpart lookup.
		got := recipientResetTargets(context.Background(), brokenLID, pn)
		require.Len(t, got, 1, "broken LID store must yield only the primary target")
		// And the full clear path still completes for the primary.
		newTestAdapter(t).clearRecipientSessions(context.Background(), client, pn)
	})
	assert.Len(t, store.deleted, 1)
}

func TestIsSessionDesyncError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"whatsmeow 400 text", errors.New("failed to send text message: server returned error 400"), true},
		{"whatsmeow 400 document", errors.New("failed to send document message: server returned error 400"), true},
		{"400 bad request alt wording", errors.New("400 bad request"), true},
		{"unrelated error", errors.New("instance not connected"), false},
		{"timeout", context.DeadlineExceeded, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isSessionDesyncError(tt.err))
		})
	}
}
