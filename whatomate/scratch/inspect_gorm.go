package main

import (
	"fmt"
	"log"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/database"
	"github.com/compnew2006/whatomate/internal/models"
	"github.com/compnew2006/whatomate/internal/tenant"
	"github.com/google/uuid"
)

func main() {
	cfg, err := config.Load("config.toml")
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewPostgres(&cfg.Database, true) // Enable raw SQL logs
	if err != nil {
		log.Fatal(err)
	}

	orgID, err := uuid.Parse("0fcc8258-a332-4455-8f7e-f9f9bbc284df")
	if err != nil {
		log.Fatal(err)
	}

	instanceID, err := uuid.Parse("d26fe3c5-2955-43a3-beec-2d449d63340a")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("--- Testing Scoped DB Query ---")
	requestDB := tenant.ScopedDB(db, orgID)

	var existing []models.WhatsAppInstance
	err = requestDB.Model(&models.WhatsAppInstance{}).
		Where("id IN ?", []uuid.UUID{instanceID}).
		Find(&existing).Error

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d instances\n", len(existing))
	for _, inst := range existing {
		fmt.Printf("ID: %s, Name: %s, OrgID: %s\n", inst.ID, inst.Name, inst.OrganizationID)
	}
}
