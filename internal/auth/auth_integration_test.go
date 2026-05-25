package auth_test

import (
	"context"
	"testing"

	"team-management/internal/auth"
	testutil "team-management/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestAuthRegisterAndLogin(t *testing.T) {
	db := testutil.InitAndResetDB(t)
	defer db.Close()

	miniRedis := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})
	defer redisClient.Close()

	authRepo := auth.NewAuthRepository(db)
	authSvc := auth.NewAuthService(authRepo, redisClient)

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
