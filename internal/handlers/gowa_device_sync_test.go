package handlers

import (
	"testing"

	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/pkg/gowa"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestPlanGowaDeviceRepair covers the startup self-heal matching: stored ids
// that still exist must be kept untouched, renamed devices are re-linked via
// JID, and unmatchable rows return "" so the caller only logs.
func TestPlanGowaDeviceRepair(t *testing.T) {
	orgID := uuid.New()
	acctID := uuid.New()
	devices := []gowa.DeviceInfo{
		{ID: "0026", JID: "966554840026@s.whatsapp.net"},
		{ID: "تصميم عسير -4625", JID: "966594374625@s.whatsapp.net"},
		{ID: "امين -4210", JID: " 966531521631@s.whatsapp.net "},
	}
	mk := func(deviceID, jid string) models.WhatsAppAccount {
		acct := models.WhatsAppAccount{
			OrganizationID: orgID,
			Name:           "Arkan",
			GowaDeviceID:   deviceID,
			GowaJID:        jid,
		}
		acct.ID = acctID
		return acct
	}

	tests := []struct {
		name string
		acct models.WhatsAppAccount
		want string
	}{
		{"stored id still exists — untouched", mk("0026", "966554840026@s.whatsapp.net"), "0026"},
		{"renamed device re-linked via JID", mk("Arkan-4625", "966594374625@s.whatsapp.net"), "تصميم عسير -4625"},
		{"JID match trims engine whitespace", mk("Arkan-4210", "966531521631@s.whatsapp.net"), "امين -4210"},
		{"stale id and no JID — no plan", mk("Arkan-6178", ""), ""},
		{"stale id and JID unknown to engine — no plan", mk("Arkan-9999", "966500000000@s.whatsapp.net"), ""},
		{"empty stored id with JID match fills it", mk("", "966594374625@s.whatsapp.net"), "تصميم عسير -4625"},
		{"empty stored id and no JID — no plan", mk("", ""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := planGowaDeviceRepair(devices, tt.acct)
			assert.Equal(t, tt.want, got)
		})
	}
}
