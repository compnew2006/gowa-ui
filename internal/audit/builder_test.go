package audit

import (
	"context"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent_InfersCategory(t *testing.T) {
	assert.Equal(t, CategoryAdmin, NewEvent(ActionUserCreated).e.Category)
	assert.Equal(t, CategoryAuth, NewEvent(ActionLoginSuccess).e.Category)
	assert.Equal(t, CategoryChat, NewEvent(ActionChatClaimed).e.Category)
	assert.Equal(t, CategorySystem, NewEvent(ActionServerStarted).e.Category)
}

func TestBuilder_DefaultsSuccessTrue(t *testing.T) {
	assert.True(t, NewEvent(ActionUserCreated).e.Success)
}

func TestBuilder_ActorSystem_SetsSourceAndNilActor(t *testing.T) {
	evt := NewEvent(ActionWorkerStarted).
		ActorSystem("worker").
		Build()
	assert.Equal(t, SourceSystem, evt.Source)
	assert.Nil(t, evt.ActorUserID)
	assert.Equal(t, "worker", evt.ActorEmail) // component name echoed for traceability
}

func TestBuilder_OrgValue_SetsOrganizationID(t *testing.T) {
	id := uuid.New()
	evt := NewEvent(ActionUserCreated).OrgValue(id).Build()
	require.NotNil(t, evt.OrganizationID)
	assert.Equal(t, id, *evt.OrganizationID)
}

func TestBuilder_Org_Nilable(t *testing.T) {
	evt := NewEvent(ActionUserCreated).Org(nil).Build()
	assert.Nil(t, evt.OrganizationID)
}

func TestBuilder_Target_StringifiesUUIDAndJID(t *testing.T) {
	id := uuid.New()
	evt := NewEvent(ActionChatClaimed).Target("contact", id).Build()
	require.NotNil(t, evt.TargetID)
	assert.Equal(t, id.String(), *evt.TargetID)
	assert.Equal(t, "contact", evt.TargetType)

	evt2 := NewEvent(ActionChatClaimed).Target("group", "120363abc@g.us").Build()
	require.NotNil(t, evt2.TargetID)
	assert.Equal(t, "120363abc@g.us", *evt2.TargetID)
}

func TestBuilder_Target_NilIDNoTargetID(t *testing.T) {
	evt := NewEvent(ActionUserCreated).Target("user", nil).Build()
	assert.Nil(t, evt.TargetID)
	assert.Equal(t, "user", evt.TargetType)
}

func TestBuilder_Detail_MergesWithoutClobbering(t *testing.T) {
	evt := NewEvent(ActionUserCreated).
		Detail("ip", "1.2.3.4").
		Detail("ua", "curl/8").
		Detail("ip", "9.9.9.9"). // overwrite same key
		Build()
	assert.Equal(t, "9.9.9.9", evt.Details["ip"])
	assert.Equal(t, "curl/8", evt.Details["ua"])
}

func TestBuilder_Record_NilService_IsNoOp(t *testing.T) {
	// Must not panic when svc is nil.
	NewEvent(ActionLogout).
		ActorSystem("test").
		Record(context.Background(), nil)
}

func TestBuilder_Record_Persists(t *testing.T) {
	db := testutil.SetupTestDB(t) // skips if TEST_DATABASE_URL unset
	svc := New(db, testutil.NopLogger())

	NewEvent(ActionServerStarted).
		ActorSystem("server").
		Record(context.Background(), svc)

	var got models.AuditEvent
	require.NoError(t, db.Where("action = ?", ActionServerStarted).First(&got).Error)
	assert.Equal(t, CategorySystem, got.Category)
	assert.Equal(t, SourceSystem, got.Source)
	assert.Equal(t, "server", got.ActorEmail)
}
