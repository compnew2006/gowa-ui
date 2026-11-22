package handlers_test

import (
	"testing"
	"time"

	"github.com/compnew2006/gowa-ui/internal/handlers"
	"github.com/compnew2006/gowa-ui/internal/models"
	"github.com/compnew2006/gowa-ui/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// createTestAgent creates a test agent user with agent role in the database.
func createTestAgent(t *testing.T, app *handlers.App, orgID uuid.UUID) *models.User {
	t.Helper()

	role := testutil.CreateAgentRole(t, app.DB, orgID)
	return testutil.CreateTestUser(t, app.DB, orgID,
		testutil.WithRoleID(&role.ID),
		testutil.WithFullName("Test Agent"),
	)
}

// --- Task 4 helpers ---

// systemAgentRole returns the org's seeded-style system "agent" role
// (name "agent", is_system=true — the same lookup production uses in
// auth.go), creating it with the agent permission set when absent and
// granting any extra permission keys requested.
func systemAgentRole(t *testing.T, app *handlers.App, orgID uuid.UUID, extraPermKeys ...string) *models.CustomRole {
	t.Helper()

	var role models.CustomRole
	if err := app.DB.Where("organization_id = ? AND name = ? AND is_system = ?", orgID, "agent", true).First(&role).Error; err == nil {
		grantRolePermissions(t, app, &role, extraPermKeys...)
		return &role
	}

	all := testutil.GetOrCreateTestPermissions(t, app.DB)
	byKey := make(map[string]models.Permission, len(all))
	for _, p := range all {
		byKey[p.Resource+":"+p.Action] = p
	}
	var perms []models.Permission
	for _, key := range append(models.SystemRolePermissions()["agent"], extraPermKeys...) {
		if p, ok := byKey[key]; ok {
			perms = append(perms, p)
		}
	}
	return testutil.CreateTestRoleExact(t, app.DB, orgID, "agent", true, false, perms)
}

// grantRolePermissions grants any of the given permission keys the role does
// not already hold.
func grantRolePermissions(t *testing.T, app *handlers.App, role *models.CustomRole, keys ...string) {
	t.Helper()
	if len(keys) == 0 {
		return
	}
	all := testutil.GetOrCreateTestPermissions(t, app.DB)
	byKey := make(map[string]models.Permission, len(all))
	for _, p := range all {
		byKey[p.Resource+":"+p.Action] = p
	}
	var held []models.Permission
	require.NoError(t, app.DB.Model(role).Association("Permissions").Find(&held))
	heldKeys := make(map[string]bool, len(held))
	for _, p := range held {
		heldKeys[p.Resource+":"+p.Action] = true
	}
	var missing []models.Permission
	for _, key := range keys {
		if p, ok := byKey[key]; ok && !heldKeys[key] {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		require.NoError(t, app.DB.Model(role).Association("Permissions").Append(missing))
	}
}

// createSystemAgent creates a user holding the org's system agent role.
func createSystemAgent(t *testing.T, app *handlers.App, orgID uuid.UUID, name string, extraPermKeys ...string) *models.User {
	t.Helper()

	role := systemAgentRole(t, app, orgID, extraPermKeys...)
	return testutil.CreateTestUser(t, app.DB, orgID,
		testutil.WithFullName(name),
		testutil.WithRoleID(&role.ID),
	)
}

// createAvailabilityLog seeds one availability-log interval for a user in an
// org. A zero end leaves the log open (ended_at NULL).
func createAvailabilityLog(t *testing.T, app *handlers.App, orgID, userID uuid.UUID, isAvailable bool, start, end time.Time) *models.UserAvailabilityLog {
	t.Helper()

	log := &models.UserAvailabilityLog{
		UserID:         userID,
		OrganizationID: orgID,
		IsAvailable:    isAvailable,
		StartedAt:      start,
	}
	if !end.IsZero() {
		log.EndedAt = &end
	}
	require.NoError(t, app.DB.Create(log).Error)
	return log
}
