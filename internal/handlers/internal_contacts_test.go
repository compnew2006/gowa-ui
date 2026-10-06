package handlers

import (
	"fmt"
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
		saudiPhone: uniquePhone(), egyptPhone: uniquePhone()}
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
		testutil.WithPhoneNumber(uniquePhone()), testutil.WithContactAccount(f.saudi.Name))

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
	customerPhone := uniquePhone()
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
		testutil.WithPhoneNumber(uniquePhone()), testutil.WithContactAccount(f.saudi.Name))
	app.maybeSendCloseRatingPrompt(f.org.ID, agent.ID, *customer)
	assert.Equal(t, int64(1), cycles(customer.ID), "customer must get a rating prompt")

	// Closing the conversation with the Egypt account must not.
	internal := testutil.CreateTestContactWith(t, app.DB, f.org.ID,
		testutil.WithPhoneNumber(f.egyptPhone), testutil.WithContactAccount(f.saudi.Name))
	app.maybeSendCloseRatingPrompt(f.org.ID, agent.ID, *internal)
	assert.Equal(t, int64(0), cycles(internal.ID), "internal conversations never get rating prompts")
}
