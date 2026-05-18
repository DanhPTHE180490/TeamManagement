package auth_test

import (
	"testing"

	"team-management/internal/auth"
	"team-management/internal/testutil"
)

func TestAuthRegisterAndLogin(t *testing.T) {
	db := testutil.InitAndResetDB(t)
	defer db.Close()

	authRepo := auth.NewAuthRepository(db)
	authSvc := auth.NewAuthService(authRepo)

	user, err := authSvc.Register("itest-user", "itest@example.com", "password123", "member")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	token, err := authSvc.Login(user.Email, "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token == "" {
		t.Fatalf("expected token, got empty string")
	}
}
