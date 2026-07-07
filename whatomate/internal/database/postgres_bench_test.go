package database

import (
	"testing"
	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/google/uuid"
)

func setupDBMock(b *testing.B) (*gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		b.Fatalf("failed to open mock sql db: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		b.Fatalf("failed to init gorm: %v", err)
	}

	return gormDB, mock
}

func BenchmarkBackfillAdminChatDeletePermission(b *testing.B) {
	gormDB, mock := setupDBMock(b)

	permID := uuid.New()
	roles := make([]uuid.UUID, 100)
	for i := range roles {
		roles[i] = uuid.New()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.ExpectQuery(`SELECT \* FROM "permissions"`).WillReturnRows(
			sqlmock.NewRows([]string{"id"}).AddRow(permID),
		)

		rows := sqlmock.NewRows([]string{"id", "name", "is_system"})
		for _, roleID := range roles {
			rows.AddRow(roleID, "admin", true)
		}
		mock.ExpectQuery(`SELECT \* FROM "custom_roles"`).WillReturnRows(rows)

		mock.ExpectBegin()
		mock.ExpectExec(`(?i)INSERT INTO "role_permissions"`).WillReturnResult(sqlmock.NewResult(1, 100))
		mock.ExpectCommit()

		err := BackfillAdminChatDeletePermission(gormDB)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
