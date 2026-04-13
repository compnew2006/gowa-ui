package handlers

import (
	"strings"
	"testing"

	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBuildUsersListBaseQuery_UsesIsolatedStatements(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)
	orgID := uuid.New()
	requestDB := tenant.ScopedDB(db.Session(&gorm.Session{DryRun: true}), orgID)

	countQuery := buildUsersListBaseQuery(requestDB, orgID, "alice")
	var total int64
	require.NoError(t, countQuery.Count(&total).Error)

	var users []models.User
	dataQuery := buildUsersListBaseQuery(requestDB, orgID, "alice")
	require.NoError(t, dataQuery.Order("users.created_at DESC").Find(&users).Error)

	require.Equal(t, 1, strings.Count(countQuery.Statement.SQL.String(), "JOIN user_organizations"))
	require.Equal(t, 1, strings.Count(dataQuery.Statement.SQL.String(), "JOIN user_organizations"))
	require.Equal(t, 1, strings.Count(dataQuery.Statement.SQL.String(), "users.deleted_at IS NULL"))
}
