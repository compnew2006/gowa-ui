package handlers

import (
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/google/uuid"
)

func TestSelectedRenderedOptionMatchesSnapshotNumber(t *testing.T) {
	userID := uuid.New()
	teamID := uuid.New()
	snapshot := models.JSONBArray{
		map[string]any{
			"number":    float64(1),
			"option_id": "participant-1",
			"type":      string(models.AgentSelectionOptionAgent),
			"label":     "Agent One",
			"user_id":   userID.String(),
		},
		map[string]any{
			"number":    float64(2),
			"option_id": "team-1",
			"type":      string(models.AgentSelectionOptionTeam),
			"label":     "Support",
			"team_id":   teamID.String(),
		},
	}

	option, ok := selectedRenderedOption(snapshot, "2")
	if !ok {
		t.Fatal("expected option to be selected")
	}
	if option.Type != models.AgentSelectionOptionTeam {
		t.Fatalf("type = %q, want %q", option.Type, models.AgentSelectionOptionTeam)
	}
	if option.TeamID == nil || *option.TeamID != teamID {
		t.Fatalf("team id = %v, want %s", option.TeamID, teamID)
	}
}

func TestSelectedRenderedOptionRejectsInvalidReply(t *testing.T) {
	snapshot := models.JSONBArray{
		map[string]any{
			"number":    float64(1),
			"option_id": "custom_final",
			"type":      string(models.AgentSelectionOptionCustom),
			"label":     "Visit branch",
		},
	}

	if _, ok := selectedRenderedOption(snapshot, "9"); ok {
		t.Fatal("expected out-of-range reply to be rejected")
	}
	if _, ok := selectedRenderedOption(snapshot, "hello"); ok {
		t.Fatal("expected non-numeric reply to be rejected")
	}
}

func TestSessionHasProcessedInbound(t *testing.T) {
	inboundID := uuid.New()
	session := &models.AgentSelectionSession{
		ProcessedInboundIDs: models.StringArray{inboundID.String()},
	}

	if !sessionHasProcessedInbound(session, inboundID) {
		t.Fatal("expected inbound id to be marked as processed")
	}
	if sessionHasProcessedInbound(session, uuid.New()) {
		t.Fatal("expected unknown inbound id to be unprocessed")
	}
}

func TestNormalizeStringArrayTrimsDeduplicatesAndDropsEmpty(t *testing.T) {
	got := normalizeStringArray([]string{" transfer ", "", "Transfer", "agent"})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(got), got)
	}
	if got[0] != "transfer" || got[1] != "agent" {
		t.Fatalf("got %#v, want [transfer agent]", got)
	}
}

func TestAgentSelectionSettingsAppliesToInstance(t *testing.T) {
	instanceA := uuid.New()
	instanceB := uuid.New()

	global := &models.AgentSelectionSettings{}
	if !agentSelectionSettingsAppliesToInstance(global, &instanceA) {
		t.Fatal("expected unrestricted global settings to apply")
	}

	restricted := &models.AgentSelectionSettings{
		AllowedInstanceIDs: models.StringArray{instanceA.String()},
	}
	if !agentSelectionSettingsAppliesToInstance(restricted, &instanceA) {
		t.Fatal("expected restricted global settings to apply to allowed instance")
	}
	if agentSelectionSettingsAppliesToInstance(restricted, &instanceB) {
		t.Fatal("expected restricted global settings to skip unlisted instance")
	}
	if agentSelectionSettingsAppliesToInstance(restricted, nil) {
		t.Fatal("expected restricted global settings to skip nil instance")
	}

	instanceScoped := &models.AgentSelectionSettings{
		InstanceID: &instanceA,
	}
	if !agentSelectionSettingsAppliesToInstance(instanceScoped, &instanceA) {
		t.Fatal("expected exact instance settings to apply")
	}
	if agentSelectionSettingsAppliesToInstance(instanceScoped, &instanceB) {
		t.Fatal("expected exact instance settings to skip other instances")
	}
}
