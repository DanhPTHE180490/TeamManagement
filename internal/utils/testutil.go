package utils

import (
	"os"
	"strings"
	"syscall"
	"testing"

	"team-management/internal/database"

	"github.com/jmoiron/sqlx"
)

// InitAndResetDB initializes the DB connection using environment fallbacks
// and resets tables used by integration tests. Callers should `defer db.Close()`.
func InitAndResetDB(t *testing.T) *sqlx.DB {
	if os.Getenv("DB_DSN") == "" {
		os.Setenv("DB_DSN", "root:password_1234@tcp(127.0.0.1:3307)/microservices_capstone?parseTime=true")
	}

	lockFile, err := os.OpenFile("/tmp/team-management-itest.lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatalf("failed to open integration test lock file: %v", err)
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		lockFile.Close()
		t.Fatalf("failed to acquire integration test lock: %v", err)
	}
	t.Cleanup(func() {
		_ = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
		_ = lockFile.Close()
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
