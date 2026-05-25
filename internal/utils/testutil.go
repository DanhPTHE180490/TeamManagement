package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"team-management/internal/database"

	"github.com/gofrs/flock"
	"github.com/jmoiron/sqlx"
)

// InitAndResetDB initializes the DB connection using environment fallbacks
// and resets tables used by integration tests. Callers should `defer db.Close()`.
func InitAndResetDB(t *testing.T) *sqlx.DB {
	if os.Getenv("DB_DSN") == "" {
		os.Setenv("DB_DSN", "root:password_1234@tcp(127.0.0.1:3307)/microservices_capstone?parseTime=true")
	}

	// Use os.TempDir() for cross-platform temp directory resolution
	lockPath := filepath.Join(os.TempDir(), "team-management-itest.lock")
	fileLock := flock.New(lockPath)

	// Block until the lock is successfully acquired
	if err := fileLock.Lock(); err != nil {
		t.Fatalf("failed to acquire integration test lock: %v", err)
	}

	t.Cleanup(func() {
		if err := fileLock.Unlock(); err != nil {
			t.Logf("failed to release integration test lock: %v", err)
		}
	})

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
