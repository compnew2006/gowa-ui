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

func TestDeleteOrganizationQueries_UseFreshScopedSessions(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)
	currentOrgID := uuid.New()
	targetOrgID := uuid.New()
	userID := uuid.New()
	requestDB := tenant.ScopedDB(db.Session(&gorm.Session{DryRun: true}), currentOrgID)

	var org models.Organization
	lookupQuery := requestDB.Session(&gorm.Session{}).
		Where("id = ?", targetOrgID)
	require.NoError(t, lookupQuery.First(&org).Error)

	var orgCount int64
	countQuery := requestDB.Session(&gorm.Session{}).
		Model(&models.Organization{})
	require.NoError(t, countQuery.Count(&orgCount).Error)

	var currentUser models.User
	currentUserQuery := requestDB.Session(&gorm.Session{}).
		Select("id", "organization_id").
		Where("id = ?", userID)
	require.NoError(t, currentUserQuery.First(&currentUser).Error)

	lookupSQL := strings.ToLower(lookupQuery.Statement.SQL.String())
	countSQL := strings.ToLower(countQuery.Statement.SQL.String())
	currentUserSQL := strings.ToLower(currentUserQuery.Statement.SQL.String())

	require.Contains(t, lookupSQL, "from `organizations`")
	require.Contains(t, lookupSQL, "`organizations`.`id`")
	require.Contains(t, countSQL, "from `organizations`")
	require.NotContains(t, countSQL, "`organizations`.`id`")
	require.Contains(t, currentUserSQL, "from `users`")
	require.NotContains(t, currentUserSQL, "from `organizations`")
}
