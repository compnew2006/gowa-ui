package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/compnew2006/whatomate/internal/config"
	"github.com/compnew2006/whatomate/internal/database"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, err := config.Load("config.toml")
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewPostgres(&cfg.Database, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("--- Running Database AutoMigrate ---")
	if err := database.AutoMigrate(db); err != nil {
		log.Fatal(err)
	}

	connStr := "postgres://whatomate:whatomate@127.0.0.1:5432/whatomate?sslmode=disable"
	rawDB, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer rawDB.Close()

	fmt.Println("\n--- Verifying idx_agent_selection_participant_user ---")
	var def string
	err = rawDB.QueryRow(`
		SELECT indexdef FROM pg_indexes 
		WHERE tablename = 'agent_selection_participants' AND indexname = 'idx_agent_selection_participant_user'
	`).Scan(&def)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Index Definition: %s\n", def)
	if strings.Contains(strings.ToUpper(def), "DELETED_AT IS NULL") {
		fmt.Println("SUCCESS: idx_agent_selection_participant_user is a partial unique index!")
	} else {
		log.Fatal("FAILURE: idx_agent_selection_participant_user is NOT a partial unique index!")
	}

	fmt.Println("\n--- Verifying idx_saved_contents_org_name ---")
	var def2 string
	err = rawDB.QueryRow(`
		SELECT indexdef FROM pg_indexes 
		WHERE tablename = 'saved_contents' AND indexname = 'idx_saved_contents_org_name'
	`).Scan(&def2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Index Definition: %s\n", def2)
	if strings.Contains(strings.ToUpper(def2), "DELETED_AT IS NULL") {
		fmt.Println("SUCCESS: idx_saved_contents_org_name is a partial unique index!")
	} else {
		log.Fatal("FAILURE: idx_saved_contents_org_name is NOT a partial unique index!")
	}
}
