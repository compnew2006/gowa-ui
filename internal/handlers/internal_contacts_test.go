package handlers

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/config"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountPhoneFromJID(t *testing.T) {
	tests := []struct {
		name string
		jid  string
		want string
	}{
		{name: "plain user JID", jid: "966554840026@s.whatsapp.net", want: "966554840026"},
		{name: "device suffix", jid: "966554840026:12@s.whatsapp.net", want: "966554840026"},
		{name: "bare phone", jid: "966554840026", want: "966554840026"},
		{name: "empty", jid: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, accountPhoneFromJID(tt.jid))
		})
	}
}

var internalPhoneSeq atomic.Uint64

// internalTestPhone returns a phone that is unique across calls. uniquePhone()
// alone can repeat within one clock tick, and these tests need several
// distinct org numbers back to back.
func internalTestPhone() string {
	return fmt.Sprintf("9667%08d%03d", time.Now().UnixNano()/1000%1e8, internalPhoneSeq.Add(1)%1000)
}

// internalChatFixture is an org with two connected accounts (Saudi, Egypt)
// plus a regular customer.
type internalChatFixture struct {
	org        *models.Organization
	saudi      *models.WhatsAppAccount
	egypt      *models.WhatsAppAccount
	saudiPhone string
	egyptPhone string
}

func newInternalChatFixture(t *testing.T, app *App) internalChatFixture {
	t.Helper()
	org, saudi := createProcessorTestOrg(t, app)
	egypt := testutil.CreateTestWhatsAppAccountWith(t, app.DB, org.ID,
		testutil.WithAccountName("egypt-"+uuid.New().String()[:8]))
	f := internalChatFixture{org: org, saudi: saudi, egypt: egypt,
		saudiPhone: internalTestPhone(), egyptPhone: internalTestPhone()}
	require.NoError(t, app.DB.Model(saudi).Update("gowa_jid", f.saudiPhone+"@s.whatsapp.net").Error)
	// Device-suffixed JID: the phone must still be recognised.
	require.NoError(t, app.DB.Model(egypt).Update("gowa_jid", f.egyptPhone+":7@s.whatsapp.net").Error)
	return f
}

func TestBuildContactResponses_MarksInternalContacts(t *testing.T) {
	app := newProcessorTestApp(t)
	f := newInternalChatFixture(t, app)
	viewer := testutil.CreateTestUser(t, app.DB, f.org.ID)

	internal := testutil.CreateTestContactWith(t, app.DB, f.org.ID,
		testutil.WithPhoneNumber(f.egyptPhone), testutil.WithContactAccount(f.saudi.Name))
	customer := testutil.CreateTestContactWith(t, app.DB, f.org.ID,
		testutil.WithPhoneNumber(internalTestPhone()), testutil.WithContactAccount(f.saudi.Name))

	for _, mask := range []bool{false, true} {
		t.Run(fmt.Sprintf("list mask=%v", mask), func(t *testing.T) {
			resps := app.buildContactResponsesMasked([]models.Contact{*internal, *customer}, f.org.ID, viewer.ID, mask)
			require.Len(t, resps, 2)
			assert.True(t, resps[0].IsInternal, "a contact whose phone is an org account is internal")
			assert.Equal(t, f.egypt.Name, resps[0].InternalAccountName)
			assert.False(t, resps[1].IsInternal, "a regular customer is not internal")
			assert.Empty(t, resps[1].InternalAccountName)
		})
	}

	single := app.buildContactResponseMasked(internal, f.org.ID, viewer.ID, true)
	assert.True(t, single.IsInternal)
	assert.Equal(t, f.egypt.Name, single.InternalAccountName)

	// Another org's number is not internal here.
	other := testutil.CreateTestOrganization(t, app.DB)
	assert.False(t, app.isInternalPhone(other.ID, f.egyptPhone))
}

// alwaysClosedBusinessHours returns a business-hours block whose one-hour
// window starts two hours from now, so the current time is always outside it.
func alwaysClosedBusinessHours(awayMessage string) models.JSONB {
	now := time.Now().UTC()
	start := now.Add(2 * time.Hour)
	end := now.Add(3 * time.Hour)
	return models.JSONB{"business_hours": map[string]any{
		"enabled":      true,
		"start_time":   start.Format("15:04"),
		"end_time":     end.Format("15:04"),
		"away_message": awayMessage,
	}}
}

func TestMaybeSendAwayReply_SkipsInternalNumbers(t *testing.T) {
	app := newProcessorTestApp(t)
	app.Config = &config.Config{}
	f := newInternalChatFixture(t, app)
	f.saudi.Settings = alwaysClosedBusinessHours("We are closed")

	outgoingTo := func(phone string) int64 {
		var n int64
		app.DB.Model(&models.Message{}).
			Joins("JOIN contacts ON contacts.id = messages.contact_id").
			Where("contacts.organization_id = ? AND contacts.phone_number = ? AND messages.direction = ?",
				f.org.ID, phone, models.DirectionOutgoing).
			Count(&n)
		return n
	}

	// Positive control: a customer writing outside hours gets the away reply.
	customerPhone := internalTestPhone()
	app.maybeSendAwayReply(f.saudi, customerPhone, "Customer")
	assert.Equal(t, int64(1), outgoingTo(customerPhone), "customer must get the away reply")

	// The Egypt account writing to the Saudi account must not.
	app.maybeSendAwayReply(f.saudi, f.egyptPhone, "Egypt branch")
	assert.Equal(t, int64(0), outgoingTo(f.egyptPhone), "internal numbers never get away replies")
}

func TestMaybeSendCloseRatingPrompt_SkipsInternalNumbers(t *testing.T) {
	app := newProcessorTestApp(t)
	app.Config = &config.Config{}
	f := newInternalChatFixture(t, app)
	agent := testutil.CreateTestUser(t, app.DB, f.org.ID)
	require.NoError(t, app.DB.Model(f.saudi).Update("settings",
		models.JSONB{"close_rating": map[string]any{"enabled": true}}).Error)

	cycles := func(contactID uuid.UUID) int64 {
		var n int64
		app.DB.Model(&models.ChatClosureRating{}).Where("contact_id = ?", contactID).Count(&n)
		return n
	}

	// Positive control: closing a customer chat opens a rating cycle.
	customer := testutil.CreateTestContactWith(t, app.DB, f.org.ID,
		testutil.WithPhoneNumber(internalTestPhone()), testutil.WithContactAccount(f.saudi.Name))
	app.maybeSendCloseRatingPrompt(f.org.ID, agent.ID, *customer)
	assert.Equal(t, int64(1), cycles(customer.ID), "customer must get a rating prompt")

	// Closing the conversation with the Egypt account must not.
	internal := testutil.CreateTestContactWith(t, app.DB, f.org.ID,
		testutil.WithPhoneNumber(f.egyptPhone), testutil.WithContactAccount(f.saudi.Name))
	app.maybeSendCloseRatingPrompt(f.org.ID, agent.ID, *internal)
	assert.Equal(t, int64(0), cycles(internal.ID), "internal conversations never get rating prompts")
}

// orgContactsReader is a contacts:read user with no account assignment, so
// scopeAssignedContact gives them every conversation in the org.
func orgContactsReader(t *testing.T, app *App, orgID uuid.UUID) *models.User {
	t.Helper()
	role := testutil.CreateTestRoleWithKeys(t, app.DB, orgID, "reader-"+uuid.New().String()[:8], []string{"contacts:read"})
	return testutil.CreateTestUser(t, app.DB, orgID, testutil.WithRoleID(&role.ID))
}

// saveInternalCopy stores one account's copy of a message on the contact
// whose phone is the other account's number.
func saveInternalCopy(t *testing.T, app *App, orgID uuid.UUID, account string, contactID uuid.UUID,
	dir models.Direction, status models.MessageStatus, body string, at time.Time) {
	t.Helper()
	require.NoError(t, app.DB.Create(&models.Message{
		BaseModel:       models.BaseModel{ID: uuid.New(), CreatedAt: at},
		OrganizationID:  orgID,
		WhatsAppAccount: account,
		ContactID:       contactID,
		Direction:       dir,
		MessageType:     models.MessageTypeText,
		Content:         body,
		Status:          status,
	}).Error)
}

func TestInternalConversations_MergesBothSidesIntoOnePair(t *testing.T) {
	app := newProcessorTestApp(t)
	f := newInternalChatFixture(t, app)
	admin := orgContactsReader(t, app, f.org.ID)
	cairo := testutil.CreateTestWhatsAppAccountWith(t, app.DB, f.org.ID,
		testutil.WithAccountName("cairo-"+uuid.New().String()[:8]))
	cairoPhone := internalTestPhone()
	require.NoError(t, app.DB.Model(cairo).Update("gowa_jid", cairoPhone+"@s.whatsapp.net").Error)

	// One contact per org number; each holds the OTHER accounts' copies.
	saudiC := testutil.CreateTestContactWith(t, app.DB, f.org.ID, testutil.WithPhoneNumber(f.saudiPhone))
	egyptC := testutil.CreateTestContactWith(t, app.DB, f.org.ID, testutil.WithPhoneNumber(f.egyptPhone))
	cairoC := testutil.CreateTestContactWith(t, app.DB, f.org.ID, testutil.WithPhoneNumber(cairoPhone))
	customer := testutil.CreateTestContactWith(t, app.DB, f.org.ID, testutil.WithPhoneNumber(internalTestPhone()))

	t0 := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	// Saudi → Egypt "hi": sender copy on egyptC, recipient copy (unread) on saudiC.
	saveInternalCopy(t, app, f.org.ID, f.saudi.Name, egyptC.ID, models.DirectionOutgoing, models.MessageStatusSent, "hi", t0)
	saveInternalCopy(t, app, f.org.ID, f.egypt.Name, saudiC.ID, models.DirectionIncoming, models.MessageStatusDelivered, "hi", t0)
	// Egypt → Saudi "reply" (latest): unread on the Saudi side.
	saveInternalCopy(t, app, f.org.ID, f.egypt.Name, saudiC.ID, models.DirectionOutgoing, models.MessageStatusSent, "reply", t0.Add(time.Minute))
	saveInternalCopy(t, app, f.org.ID, f.saudi.Name, egyptC.ID, models.DirectionIncoming, models.MessageStatusDelivered, "reply", t0.Add(time.Minute))
	// Saudi → Cairo, only the sender copy synced so far (older than the Egypt pair).
	saveInternalCopy(t, app, f.org.ID, f.saudi.Name, cairoC.ID, models.DirectionOutgoing, models.MessageStatusSent, "cairo?", t0.Add(-time.Minute))
	// Noise that must not form pairs: a customer chat and a self-chat.
	saveInternalCopy(t, app, f.org.ID, f.saudi.Name, customer.ID, models.DirectionIncoming, models.MessageStatusDelivered, "customer", t0)
	saveInternalCopy(t, app, f.org.ID, f.saudi.Name, saudiC.ID, models.DirectionOutgoing, models.MessageStatusSent, "note to self", t0)

	convs, err := app.internalConversations(f.org.ID, admin.ID)
	require.NoError(t, err)
	require.Len(t, convs, 2, "one entry per account pair: saudi↔egypt and saudi↔cairo")

	egyptPair := convs[0]
	assert.ElementsMatch(t, []string{f.saudi.Name, f.egypt.Name}, egyptPair.Accounts[:], "most recent pair first")
	require.Len(t, egyptPair.Sides, 2, "both accounts' copies merge into one conversation")
	assert.Equal(t, "reply", egyptPair.LastMessagePreview)
	assert.Equal(t, 2, egyptPair.UnreadCount, "unread incoming copies from both sides add up")
	for _, side := range egyptPair.Sides {
		assert.Equal(t, 1, side.UnreadCount)
		if side.Account == f.saudi.Name {
			assert.Equal(t, egyptC.ID, side.ContactID, "saudi's copies live on the egypt-number contact")
			assert.Equal(t, f.egypt.Name, side.PeerAccount)
		} else {
			assert.Equal(t, saudiC.ID, side.ContactID, "egypt's copies live on the saudi-number contact")
			assert.Equal(t, f.saudi.Name, side.PeerAccount)
		}
	}

	cairoPair := convs[1]
	assert.ElementsMatch(t, []string{f.saudi.Name, cairo.Name}, cairoPair.Accounts[:])
	require.Len(t, cairoPair.Sides, 1, "a pair with only one synced side still lists once")
	assert.Equal(t, cairoC.ID, cairoPair.Sides[0].ContactID)
	assert.Equal(t, 0, cairoPair.UnreadCount)
}

func TestInternalConversations_NeedsTwoConnectedAccounts(t *testing.T) {
	app := newProcessorTestApp(t)
	org, account := createProcessorTestOrg(t, app)
	admin := orgContactsReader(t, app, org.ID)
	require.NoError(t, app.DB.Model(account).Update("gowa_jid", internalTestPhone()+"@s.whatsapp.net").Error)

	convs, err := app.internalConversations(org.ID, admin.ID)
	require.NoError(t, err)
	assert.Empty(t, convs)
	assert.NotNil(t, convs, "an empty list serialises as [] for the frontend")
}

func TestInternalConversations_FollowsContactVisibility(t *testing.T) {
	app := newProcessorTestApp(t)
	f := newInternalChatFixture(t, app)
	saudiC := testutil.CreateTestContactWith(t, app.DB, f.org.ID, testutil.WithPhoneNumber(f.saudiPhone))
	egyptC := testutil.CreateTestContactWith(t, app.DB, f.org.ID, testutil.WithPhoneNumber(f.egyptPhone))
	at := time.Now().UTC().Truncate(time.Second)
	saveInternalCopy(t, app, f.org.ID, f.saudi.Name, egyptC.ID, models.DirectionOutgoing, models.MessageStatusSent, "hi", at)
	saveInternalCopy(t, app, f.org.ID, f.egypt.Name, saudiC.ID, models.DirectionIncoming, models.MessageStatusDelivered, "hi", at)

	// An agent without contacts:read sees only what they are involved in:
	// assigned to the Saudi side, they get the pair with that side alone.
	role := testutil.CreateTestRoleWithKeys(t, app.DB, f.org.ID, "agent-"+uuid.New().String()[:8], []string{"chat:read"})
	agent := testutil.CreateTestUser(t, app.DB, f.org.ID, testutil.WithRoleID(&role.ID))
	require.NoError(t, app.DB.Model(egyptC).Update("assigned_user_id", agent.ID).Error)

	convs, err := app.internalConversations(f.org.ID, agent.ID)
	require.NoError(t, err)
	require.Len(t, convs, 1)
	require.Len(t, convs[0].Sides, 1, "the side the agent cannot open must not be listed")
	assert.Equal(t, egyptC.ID, convs[0].Sides[0].ContactID)
	assert.Equal(t, f.saudi.Name, convs[0].Sides[0].Account)
}
