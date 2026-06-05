package main

import (
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Team struct {
	ID             uint
	OrganizationID uint
	Name           string
}

type TeamMember struct {
	ID     uint
	TeamID uint
	UserID uint
}

func main() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DryRun: true,
	})
	if err != nil {
		panic(err)
	}

	// Create a scoped DB session (similar to tenant.ScopedDB)
	scopedDB := db.Session(&gorm.Session{}).Scopes(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("organization_id = ?", 42)
	})

	// Mutate the parent scopedDB by setting Table or Model or Where in-place (clone = 0)
	// Wait, db.Session returns a cloned db, but what if we do operations directly on it?
	// In GORM, if clone == 0 (which it is for scopedDB since it is created via Session()),
	// any method called on it mutates it.
	
	// Let's simulate a mutation on the requestDB (e.g. from a previous query in the middleware or handler)
	// For example, in a previous handler or query:
	scopedDB.Model(&Team{}) // This sets Statement.Model to Team in-place on scopedDB!

	// Now try calling query on scopedDB clone for TeamMember
	tx1 := scopedDB.Session(&gorm.Session{})
	
	// Print Table and Model
	fmt.Printf("tx1 Model: %v\n", tx1.Statement.Model)
	fmt.Printf("tx1 Table: %v\n", tx1.Statement.Table)

	member := TeamMember{TeamID: 1, UserID: 2}
	stmt1 := tx1.Create(&member).Statement
	fmt.Printf("Create SQL: %v\n", stmt1.SQL.String())
}
