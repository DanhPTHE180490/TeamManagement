package team_test

import (
	"context"
	"testing"

	"team-management/internal/auth"
	"team-management/internal/team"
	testutil "team-management/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestTeamCreateAndAddMember(t *testing.T) {
	db := testutil.InitAndResetDB(t)
	defer db.Close()

	authRepo := auth.NewAuthRepository(db)
	authSvc := auth.NewAuthService(authRepo)

	// create manager and member
	mgr, err := authSvc.Register(context.Background(), "manager-itest", "mgr-itest@example.com", "password123", "manager")
	if err != nil {
		t.Fatalf("failed to register manager: %v", err)
	}
	member, err := authSvc.Register(context.Background(), "member-itest", "member-itest@example.com", "password123", "member")
	if err != nil {
		t.Fatalf("failed to register member: %v", err)
	}

	miniRedis := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})
	defer redisClient.Close()

	teamRepo := team.NewTeamRepository(db, redisClient)
	teamSvc := team.NewTeamService(teamRepo)

	created, err := teamSvc.CreateTeam(context.Background(), "Integration Team", int64(mgr.ID), mgr.SystemRole)
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	if created.ID == 0 {
		t.Fatalf("expected created team ID > 0")
	}

	if err := teamSvc.AddMemberToTeam(context.Background(), created.ID, int64(member.ID), int64(mgr.ID)); err != nil {
		t.Fatalf("failed to add member to team: %v", err)
	}

	teams, err := teamSvc.GetTeamsByUserID(context.Background(), int64(member.ID))
	if err != nil {
		t.Fatalf("failed to get teams for member: %v", err)
	}
	if len(teams) == 0 {
		t.Fatalf("expected member to be in at least one team")
	}
}
