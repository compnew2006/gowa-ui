package handlers

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/compnew2006/whatomate/test/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type testLogWriter struct {
	buffer *bytes.Buffer
}

func (w testLogWriter) Printf(format string, args ...interface{}) {
	fmt.Fprintf(w.buffer, format, args...)
}

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

func TestDeleteUserQueries_UseFreshScopedSessions(t *testing.T) {
	t.Parallel()

	db := testutil.SetupTestDB(t)
	orgID := uuid.New()
	userID := uuid.New()
	roleID := uuid.New()
	requestDB := tenant.ScopedDB(db.Session(&gorm.Session{DryRun: true}), orgID)

	var user models.User
	lookupQuery := requestDB.Session(&gorm.Session{}).
		Select("users.*").
		Joins("JOIN user_organizations ON user_organizations.user_id = users.id AND user_organizations.organization_id = ? AND user_organizations.deleted_at IS NULL", orgID).
		Where("users.id = ? AND users.deleted_at IS NULL", userID).
		Preload("Role")
	require.NoError(t, lookupQuery.First(&user).Error)

	var userOrg models.UserOrganization
	userOrgQuery := requestDB.Session(&gorm.Session{}).
		Where("user_id = ? AND organization_id = ?", userID, orgID).
		Preload("Role")
	require.NoError(t, userOrgQuery.First(&userOrg).Error)

	var adminRole models.CustomRole
	adminRoleQuery := requestDB.Session(&gorm.Session{}).
		Where("organization_id = ? AND name = ? AND is_system = ?", orgID, "admin", true)
	require.NoError(t, adminRoleQuery.First(&adminRole).Error)

	var adminCount int64
	adminCountQuery := requestDB.Session(&gorm.Session{}).
		Model(&models.UserOrganization{}).
		Where("organization_id = ? AND role_id = ? AND deleted_at IS NULL", orgID, roleID)
	require.NoError(t, adminCountQuery.Count(&adminCount).Error)

	deleteUserQuery := requestDB.Session(&gorm.Session{}).
		Where("id = ?", userID)
	require.NoError(t, deleteUserQuery.Delete(&models.User{}).Error)

	deleteMembershipsQuery := requestDB.Session(&gorm.Session{}).
		Where("user_id = ?", userID)
	require.NoError(t, deleteMembershipsQuery.Delete(&models.UserOrganization{}).Error)

	require.Equal(t, 1, strings.Count(lookupQuery.Statement.SQL.String(), "JOIN user_organizations"))
	require.Equal(t, 0, strings.Count(adminCountQuery.Statement.SQL.String(), "JOIN user_organizations"))
	require.Equal(t, 0, strings.Count(deleteUserQuery.Statement.SQL.String(), "JOIN user_organizations"))
	require.Equal(t, 0, strings.Count(deleteMembershipsQuery.Statement.SQL.String(), "JOIN user_organizations"))
}

func TestCurrentUserSettingsWriteUsesFreshScopedSession(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	logBuffer := &bytes.Buffer{}
	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.New(testLogWriter{buffer: logBuffer}, logger.Config{LogLevel: logger.Info}),
	})
	require.NoError(t, err)

	orgID := uuid.New()
	userID := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*users.*").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = scopedUserWriteDB(db, userID, orgID).Update("settings", models.JSONB{
		"theme_mode": "dark",
	}).Error
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	sql := strings.ToLower(logBuffer.String())
	require.Contains(t, sql, `update "users"`)
	require.NotContains(t, sql, `from "users"`)
}
