package handlers_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shridarpatil/whatomate/internal/handlers"
	"github.com/shridarpatil/whatomate/internal/models"
	"github.com/shridarpatil/whatomate/test/testutil"
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
