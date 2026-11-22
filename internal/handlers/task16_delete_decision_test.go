package handlers_test

// Task 16 — the delete decision, recorded 2026-11-21 in the execution ledger:
// agents may NOT delete contacts, WhatsApp accounts, or dashboard widgets.
// Deletion stays admin/manager-only (the plan default, adopted by the user).
// These tests pin the decision server-side: an agent holding the system
// 10-permission agent role (no delete permission at all) gets 403 on all
// three delete endpoints, while an admin succeeds. They use synthetic
// fixtures only — the deleted test conversation 966561853319 is never reused.

import (
	"testing"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

// deleteDecisionContact is a synthetic contact for the delete probes.
func deleteDecisionContact(t *testing.T, app *handlers.App, orgID uuid.UUID) *models.Contact {
	t.Helper()
	return testutil.CreateTestContact(t, app.DB, orgID)
}

func TestApp_DeleteContact_AgentForbidden(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createSystemAgent(t, app, org.ID, "Delete Decision Agent")
	contact := deleteDecisionContact(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.DeleteContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))

	// The contact must still exist — the agent delete did nothing.
	var count int64
	app.DB.Model(&models.Contact{}).Where("id = ?", contact.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestApp_DeleteContact_AdminSucceeds(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)
	contact := deleteDecisionContact(t, app, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", contact.ID.String())

	err := app.DeleteContact(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var count int64
	app.DB.Model(&models.Contact{}).Where("id = ?", contact.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestApp_DeleteAccount_AgentForbidden(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	agent := createSystemAgent(t, app, org.ID, "Delete Decision Agent")
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.DeleteAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))

	// The account must still exist — the agent delete did nothing.
	var count int64
	app.DB.Model(&models.WhatsAppAccount{}).Where("id = ?", account.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestApp_DeleteAccount_AdminSucceeds(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)
	account := testutil.CreateTestWhatsAppAccount(t, app.DB, org.ID)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", account.ID.String())

	err := app.DeleteAccount(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var count int64
	app.DB.Model(&models.WhatsAppAccount{}).Where("id = ?", account.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestApp_DeleteWidget_AgentForbidden(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)
	agent := createSystemAgent(t, app, org.ID, "Delete Decision Agent")
	widget := createTestWidget(t, app, org.ID, &admin.ID, "Delete Decision Widget", true, false)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, agent.ID)
	testutil.SetPathParam(req, "id", widget.ID.String())

	err := app.DeleteWidget(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusForbidden, testutil.GetResponseStatusCode(req))

	// The widget must still exist — the agent delete did nothing.
	var count int64
	app.DB.Model(&models.Widget{}).Where("id = ?", widget.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestApp_DeleteWidget_AdminSucceeds(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	org := testutil.CreateTestOrganization(t, app.DB)
	admin := createAdminUser(t, app, org.ID)
	widget := createTestWidget(t, app, org.ID, &admin.ID, "Delete Decision Widget", true, false)

	req := testutil.NewGETRequest(t)
	testutil.SetAuthContext(req, org.ID, admin.ID)
	testutil.SetPathParam(req, "id", widget.ID.String())

	err := app.DeleteWidget(req)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusOK, testutil.GetResponseStatusCode(req))

	var count int64
	app.DB.Model(&models.Widget{}).Where("id = ?", widget.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

// The agent role itself must hold no delete permission — the decision's
// foundation. If a future change grants the agent a delete permission,
// this fails first and names the decision.
func TestApp_DeleteDecision_AgentRoleHoldsNoDeletePermission(t *testing.T) {
	t.Parallel()

	for _, key := range models.SystemRolePermissions()["agent"] {
		assert.NotContains(t, key, ":delete",
			"agent role must hold no delete permission (Task 16 decision)")
	}
}
