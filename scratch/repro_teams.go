package main

import (
	"fmt"
	"log"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/database"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func main() {
	// Setup DB connection
	dbCfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "whatomate",
		Password: "whatomate",
		Name:     "whatomate",
		SSLMode:  "disable",
		LogSQL:   true, // Enable SQL logs to print the queries
	}

	db, err := database.NewPostgres(dbCfg, true)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	orgID := uuid.New()
	teamID := uuid.New()
	memberUserID := uuid.New()

	// Create test org and team and users
	err = db.Transaction(func(tx *gorm.DB) error {
		org := &models.Organization{BaseModel: models.BaseModel{ID: orgID}, Name: "Repro Org", Slug: "repro-org"}
		if err := tx.Create(org).Error; err != nil {
			return err
		}
		team := &models.Team{BaseModel: models.BaseModel{ID: teamID}, OrganizationID: orgID, Name: "Repro Team"}
		if err := tx.Create(team).Error; err != nil {
			return err
		}
		user := &models.User{BaseModel: models.BaseModel{ID: memberUserID}, OrganizationID: orgID, Email: "repro@user.com", FullName: "Repro User"}
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to seed database: %v", err)
	}

	// Clean up after run
	defer func() {
		db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.TeamMember{}, "team_id = ?", teamID)
		db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Team{}, "id = ?", teamID)
		db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.User{}, "id = ?", memberUserID)
		db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Organization{}, "id = ?", orgID)
	}()

	// Simulate the TenantScope scoped DB
	requestDB := tenant.ScopedDB(db, orgID)

	fmt.Println("\n--- Query 1: Verify team exists ---")
	var team models.Team
	if err := requestDB.Session(&gorm.Session{}).Where("id = ? AND organization_id = ?", teamID, orgID).
		Preload("Members").First(&team).Error; err != nil {
		log.Fatalf("failed Query 1: %v", err)
	}

	fmt.Println("\n--- Query 2: Check if already a member ---")
	var existingMember models.TeamMember
	err = requestDB.Session(&gorm.Session{}).Where("team_id = ? AND user_id = ?", teamID, memberUserID).First(&existingMember).Error
	fmt.Printf("Query 2 Result error: %v (expected ErrRecordNotFound)\n", err)

	fmt.Println("\n--- Query 3: Create team member ---")
	member := models.TeamMember{
		TeamID: teamID,
		UserID: memberUserID,
		Role:   models.TeamRoleAgent,
	}
	if err := requestDB.Session(&gorm.Session{}).Create(&member).Error; err != nil {
		fmt.Printf("Query 3 FAILED: %v\n", err)
	} else {
		fmt.Println("Query 3 SUCCESS")
	}
}
