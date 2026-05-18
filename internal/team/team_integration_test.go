package team_test

import (
	"testing"

	"team-management/internal/auth"
	"team-management/internal/team"
	"team-management/internal/testutil"
)

func TestTeamCreateAndAddMember(t *testing.T) {
	db := testutil.InitAndResetDB(t)
	defer db.Close()

	authRepo := auth.NewAuthRepository(db)
	authSvc := auth.NewAuthService(authRepo)

	// create manager and member
	mgr, err := authSvc.Register("manager-itest", "mgr-itest@example.com", "password123", "manager")
	if err != nil {
		t.Fatalf("failed to register manager: %v", err)
	}
	member, err := authSvc.Register("member-itest", "member-itest@example.com", "password123", "member")
	if err != nil {
		t.Fatalf("failed to register member: %v", err)
	}

	teamRepo := team.NewTeamRepository(db)
	teamSvc := team.NewTeamService(teamRepo)

	created, err := teamSvc.CreateTeam("Integration Team", int64(mgr.ID), mgr.SystemRole)
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	if created.ID == 0 {
		t.Fatalf("expected created team ID > 0")
	}

	if err := teamSvc.AddMemberToTeam(created.ID, int64(member.ID), int64(mgr.ID)); err != nil {
		t.Fatalf("failed to add member to team: %v", err)
	}

	teams, err := teamSvc.GetTeamsByUserID(int64(member.ID))
	if err != nil {
		t.Fatalf("failed to get teams for member: %v", err)
	}
	if len(teams) == 0 {
		t.Fatalf("expected member to be in at least one team")
	}
}
