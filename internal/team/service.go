package team

import (
	"context"
	"log"
	"team-management/internal/models"
	"team-management/internal/utils"
	"time"

	"team-management/internal/audit"

	"github.com/redis/go-redis/v9"
)

type TeamService interface {
	CreateTeam(ctx context.Context, name string, userID int64, userRole string) (*models.Team, error)
	GetTeamByID(ctx context.Context, id int64) (*models.Team, error)
	GetTeamsByUserID(ctx context.Context, userID int64) ([]*models.Team, error)
	UpdateTeam(ctx context.Context, id int64, name string, requesterID int64) (*models.Team, error)
	DeleteTeam(ctx context.Context, id int64, requesterID int64) error
	AddMemberToTeam(ctx context.Context, teamID int64, targetID int64, requesterID int64) error
	RemoveMemberFromTeam(ctx context.Context, teamID int64, targetID int64, requesterID int64) error
	UpdateMemberRole(ctx context.Context, teamID int64, targetID int64, newRole string, requesterID int64) error
}

type teamService struct {
	repo        TeamRepository
	redisClient *redis.Client
}

func NewTeamService(repo TeamRepository, redisClient *redis.Client) TeamService {
	return &teamService{repo: repo, redisClient: redisClient}
}

func (s *teamService) CreateTeam(ctx context.Context, name string, userID int64, userRole string) (*models.Team, error) {
	// Validate input
	if len(name) == 0 {
		return nil, utils.NewValidationError("team name", "cannot be empty")
	}
	if len(name) > 100 {
		return nil, utils.NewValidationError("team name", "cannot exceed 100 characters")
	}

	// Validate permission
	if userRole == "member" {
		log.Printf("Unauthorized team creation attempt by member user %d", userID)
		return nil, utils.NewForbiddenError("members cannot create teams")
	}

	team := &models.Team{
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err, teamID := s.repo.CreateTeam(ctx, team, userID, userRole)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeInternal) {
			log.Printf("Failed to create team '%s' for user %d: %v", name, userID, err)
			return nil, utils.NewInternalError("Failed to create team", err)
		}
		log.Printf("Database error creating team '%s' for user %d: %v", name, userID, err)
		return nil, utils.NewInternalError("Failed to create team", err)
	}

	team.ID = teamID

	audit.PublishEvent(ctx, s.redisClient, &userID, "TEAM_CREATED", "team", &teamID, map[string]any{"team_name": name})

	log.Printf("Team created successfully: '%s' (ID: %d) by user %d", name, teamID, userID)
	return team, nil
}

func (s *teamService) GetTeamByID(ctx context.Context, id int64) (*models.Team, error) {
	if id <= 0 {
		return nil, utils.NewValidationError("team ID", "must be greater than 0")
	}

	team, err := s.repo.GetTeamByID(ctx, id)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			return nil, err
		}
		log.Printf("Database error retrieving team %d: %v", id, err)
		return nil, utils.NewInternalError("Failed to retrieve team", err)
	}
	return team, nil
}

func (s *teamService) GetTeamsByUserID(ctx context.Context, userID int64) ([]*models.Team, error) {
	if userID <= 0 {
		return nil, utils.NewValidationError("user ID", "must be greater than 0")
	}

	teams, err := s.repo.GetTeamsByUserID(ctx, userID)
	if err != nil {
		log.Printf("Database error retrieving teams for user %d: %v", userID, err)
		return nil, utils.NewInternalError("Failed to retrieve teams", err)
	}
	return teams, nil
}

func (s *teamService) UpdateTeam(ctx context.Context, id int64, name string, requesterID int64) (*models.Team, error) {
	// Validate input
	if id <= 0 {
		return nil, utils.NewValidationError("team ID", "must be greater than 0")
	}
	if len(name) == 0 {
		return nil, utils.NewValidationError("team name", "cannot be empty")
	}
	if len(name) > 100 {
		return nil, utils.NewValidationError("team name", "cannot exceed 100 characters")
	}

	// Check if team exists
	team, err := s.repo.GetTeamByID(ctx, id)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			return nil, utils.NewNotFoundError("team")
		}
		log.Printf("Database error retrieving team %d: %v", id, err)
		return nil, utils.NewInternalError("Failed to retrieve team", err)
	}

	if team == nil {
		return nil, utils.NewNotFoundError("team")
	}

	// Check if requester is a manager of this team
	requesterRole, err := s.repo.GetTeamRole(ctx, id, requesterID)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			log.Printf("User %d is not a member of team %d", requesterID, id)
			return nil, utils.NewForbiddenError("you are not a member of this team")
		}
		log.Printf("Database error checking team role for user %d in team %d: %v", requesterID, id, err)
		return nil, utils.NewInternalError("Failed to verify permissions", err)
	}

	if requesterRole == "member" {
		log.Printf("Team member %d attempted to update team %d", requesterID, id)
		return nil, utils.NewForbiddenError("only managers can update team information")
	}

	team.Name = name
	team.UpdatedAt = time.Now()

	err = s.repo.UpdateTeam(ctx, team)
	if err != nil {
		log.Printf("Database error updating team %d: %v", id, err)
		return nil, utils.NewInternalError("Failed to update team", err)
	}

	audit.PublishEvent(ctx, s.redisClient, &requesterID, "TEAM_UPDATED", "team", &id, map[string]any{"team_name": name})
	log.Printf("Team %d updated successfully by user %d: new name '%s'", id, requesterID, name)
	return team, nil
}

func (s *teamService) DeleteTeam(ctx context.Context, id int64, requesterID int64) error {
	// Validate input
	if id <= 0 {
		return utils.NewValidationError("team ID", "must be greater than 0")
	}

	// Check if team exists
	team, err := s.repo.GetTeamByID(ctx, id)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			return utils.NewNotFoundError("team")
		}
		log.Printf("Database error retrieving team %d: %v", id, err)
		return utils.NewInternalError("Failed to retrieve team", err)
	}

	if team == nil {
		return utils.NewNotFoundError("team")
	}

	// Check if requester is the main_manager of this team (only main manager can delete)
	requesterRole, err := s.repo.GetTeamRole(ctx, id, requesterID)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			log.Printf("User %d is not a member of team %d", requesterID, id)
			return utils.NewForbiddenError("you are not a member of this team")
		}
		log.Printf("Database error checking team role for user %d in team %d: %v", requesterID, id, err)
		return utils.NewInternalError("Failed to verify permissions", err)
	}

	if requesterRole != "main_manager" {
		log.Printf("User %d attempted to delete team %d but is only %s", requesterID, id, requesterRole)
		return utils.NewForbiddenError("only the main manager can delete a team")
	}

	err = s.repo.DeleteTeam(ctx, id)
	if err != nil {
		log.Printf("Database error deleting team %d: %v", id, err)
		return utils.NewInternalError("Failed to delete team", err)
	}

	audit.PublishEvent(ctx, s.redisClient, &requesterID, "TEAM_DELETED", "team", &id, nil)
	log.Printf("Team %d deleted successfully by main manager %d", id, requesterID)
	return nil
}

func (s *teamService) AddMemberToTeam(ctx context.Context, teamID int64, targetID int64, requesterID int64) error {
	// Validate input
	if teamID <= 0 {
		return utils.NewValidationError("team ID", "must be greater than 0")
	}
	if targetID <= 0 {
		return utils.NewValidationError("user ID", "must be greater than 0")
	}
	if targetID == requesterID {
		return utils.NewValidationError("user ID", "cannot add yourself to a team (you should already be a member)")
	}

	// Check if team exists
	team, err := s.repo.GetTeamByID(ctx, teamID)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			return utils.NewNotFoundError("team")
		}
		log.Printf("Database error retrieving team %d: %v", teamID, err)
		return utils.NewInternalError("Failed to retrieve team", err)
	}

	if team == nil {
		return utils.NewNotFoundError("team")
	}

	// Check if requester has permission
	requesterRole, err := s.repo.GetTeamRole(ctx, teamID, requesterID)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			log.Printf("User %d is not a member of team %d", requesterID, teamID)
			return utils.NewForbiddenError("you are not a member of this team")
		}
		log.Printf("Database error checking team role for user %d in team %d: %v", requesterID, teamID, err)
		return utils.NewInternalError("Failed to verify permissions", err)
	}

	if requesterRole == "member" {
		log.Printf("Team member %d attempted to add user %d to team %d", requesterID, targetID, teamID)
		return utils.NewForbiddenError("only managers can add members to a team")
	}

	userExists, err := s.repo.UserExists(ctx, targetID)
	if err != nil {
		log.Printf("Database error checking user existence for user %d: %v", targetID, err)
		return utils.NewInternalError("Failed to verify target user", err)
	}
	if !userExists {
		log.Printf("Manager %d attempted to add non-existent user %d to team %d", requesterID, targetID, teamID)
		return utils.NewNotFoundError("user")
	}

	err = s.repo.AddMember(ctx, teamID, targetID, "member")
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeConflict) {
			log.Printf("User %d is already a member of team %d", targetID, teamID)
			return err
		}
		log.Printf("Database error adding user %d to team %d: %v", targetID, teamID, err)
		return utils.NewInternalError("Failed to add member to team", err)
	}

	log.Printf("User %d added to team %d by manager %d", targetID, teamID, requesterID)
	audit.PublishEvent(ctx, s.redisClient, &requesterID, "TEAM_MEMBER_ADDED", "team", &teamID, map[string]any{"user_id": targetID})
	return nil
}

func (s *teamService) RemoveMemberFromTeam(ctx context.Context, teamID int64, targetID int64, requesterID int64) error {
	// Validate input
	if teamID <= 0 {
		return utils.NewValidationError("team ID", "must be greater than 0")
	}
	if targetID <= 0 {
		return utils.NewValidationError("user ID", "must be greater than 0")
	}

	// Check if team exists
	team, err := s.repo.GetTeamByID(ctx, teamID)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			return utils.NewNotFoundError("team")
		}
		log.Printf("Database error retrieving team %d: %v", teamID, err)
		return utils.NewInternalError("Failed to retrieve team", err)
	}

	if team == nil {
		return utils.NewNotFoundError("team")
	}

	// Check if requester has permission
	requesterRole, err := s.repo.GetTeamRole(ctx, teamID, requesterID)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			log.Printf("User %d is not a member of team %d", requesterID, teamID)
			return utils.NewForbiddenError("you are not a member of this team")
		}
		log.Printf("Database error checking team role for user %d in team %d: %v", requesterID, teamID, err)
		return utils.NewInternalError("Failed to verify permissions", err)
	}

	if requesterRole == "member" {
		log.Printf("Team member %d attempted to remove user %d from team %d", requesterID, targetID, teamID)
		return utils.NewForbiddenError("only managers can remove members from a team")
	}

	err = s.repo.RemoveMember(ctx, teamID, targetID)
	if err != nil {
		log.Printf("Database error removing user %d from team %d: %v", targetID, teamID, err)
		return utils.NewInternalError("Failed to remove member from team", err)
	}

	log.Printf("User %d removed from team %d by manager %d", targetID, teamID, requesterID)
	audit.PublishEvent(ctx, s.redisClient, &requesterID, "TEAM_MEMBER_REMOVED", "team", &teamID, map[string]any{"user_id": targetID})
	return nil
}

func (s *teamService) UpdateMemberRole(ctx context.Context, teamID int64, targetID int64, newRole string, requesterID int64) error {
	// Validate input
	if teamID <= 0 {
		return utils.NewValidationError("team ID", "must be greater than 0")
	}
	if targetID <= 0 {
		return utils.NewValidationError("user ID", "must be greater than 0")
	}
	if newRole != "member" && newRole != "manager" && newRole != "main_manager" {
		return utils.NewValidationError("role", "must be 'member', 'manager', or 'main_manager'")
	}

	// Check if team exists
	team, err := s.repo.GetTeamByID(ctx, teamID)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			return utils.NewNotFoundError("team")
		}
		log.Printf("Database error retrieving team %d: %v", teamID, err)
		return utils.NewInternalError("Failed to retrieve team", err)
	}

	if team == nil {
		return utils.NewNotFoundError("team")
	}

	// Check if requester is the main_manager (only main manager can change roles)
	requesterRole, err := s.repo.GetTeamRole(ctx, teamID, requesterID)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			log.Printf("User %d is not a member of team %d", requesterID, teamID)
			return utils.NewForbiddenError("you are not a member of this team")
		}
		log.Printf("Database error checking team role for user %d in team %d: %v", requesterID, teamID, err)
		return utils.NewInternalError("Failed to verify permissions", err)
	}

	if requesterRole != "main_manager" {
		log.Printf("User %d attempted to update roles in team %d but is only %s", requesterID, teamID, requesterRole)
		return utils.NewForbiddenError("only the main manager can update member roles")
	}

	// Verify target user is a member of the team
	targetRole, err := s.repo.GetTeamRole(ctx, teamID, targetID)
	if err != nil {
		if utils.IsErrorType(err, utils.ErrTypeNotFound) {
			log.Printf("User %d is not a member of team %d", targetID, teamID)
			return utils.NewNotFoundError("user is not a member of this team")
		}
		log.Printf("Database error checking role for user %d in team %d: %v", targetID, teamID, err)
		return utils.NewInternalError("Failed to verify member status", err)
	}

	// Prevent main_manager from being changed
	if targetRole == "main_manager" {
		log.Printf("Attempt to change role of main_manager %d in team %d", targetID, teamID)
		return utils.NewForbiddenError("cannot change the role of the main manager")
	}

	err = s.repo.UpdateMemberRole(ctx, teamID, targetID, newRole)
	if err != nil {
		log.Printf("Database error updating role for user %d in team %d: %v", targetID, teamID, err)
		return utils.NewInternalError("Failed to update member role", err)
	}

	audit.PublishEvent(ctx, s.redisClient, &requesterID, "TEAM_MEMBER_ROLE_UPDATED", "team", &teamID, map[string]any{"user_id": targetID, "new_role": newRole})
	log.Printf("User %d role in team %d changed to '%s' by main manager %d", targetID, teamID, newRole, requesterID)
	return nil
}
