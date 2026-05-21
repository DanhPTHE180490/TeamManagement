package auth_test

import (
	"context"
	"testing"

	"team-management/internal/auth"
	testutil "team-management/internal/utils"
)

func TestAuthRegisterAndLogin(t *testing.T) {
	db := testutil.InitAndResetDB(t)
	defer db.Close()

	authRepo := auth.NewAuthRepository(db)
	authSvc := auth.NewAuthService(authRepo)

	user, err := authSvc.Register(context.Background(), "itest-user", "itest@example.com", "password123", "member")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	token, err := authSvc.Login(context.Background(), user.Email, "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token == "" {
		t.Fatalf("expected token, got empty string")
	}
}
