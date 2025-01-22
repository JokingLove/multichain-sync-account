package dynamic

import (
	"github.com/JokingLove/multichain-sync-account/database"
	"testing"
)

func TestExecuteMigration(t *testing.T) {
	db := database.SetupDb()

	err := db.ExecuteSQLMigration("../../migrations")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateTableFromTemplate(t *testing.T) {
	db := database.SetupDb()

	CreateTableFromTemplate("kevin", db)
}
