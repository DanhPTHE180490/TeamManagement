package testutil

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	"team-management/internal/database"
)

// InitAndResetDB initializes the DB connection using environment fallbacks
// and resets tables used by integration tests. Callers should `defer db.Close()`.
func InitAndResetDB(t *testing.T) *sql.DB {
	if os.Getenv("DB_DSN") == "" {
		os.Setenv("DB_DSN", "root:password_1234@tcp(127.0.0.1:3307)/microservices_capstone?parseTime=true")
	}

	db := database.InitDB()

	resetSQL := `SET FOREIGN_KEY_CHECKS=0;
TRUNCATE TABLE team_members;
TRUNCATE TABLE teams;
TRUNCATE TABLE folders;
TRUNCATE TABLE notes;
TRUNCATE TABLE folder_shares;
TRUNCATE TABLE note_shares;
DELETE FROM users WHERE email != 'admin@example.com';
SET FOREIGN_KEY_CHECKS=1;`

	for _, query := range strings.Split(resetSQL, ";") {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		if _, err := db.Exec(query); err != nil {
			db.Close()
			t.Fatalf("failed to reset DB: %v", err)
		}
	}
	return db
}
