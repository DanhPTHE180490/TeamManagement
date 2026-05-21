package team

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"team-management/internal/models"
	apperrors "team-management/internal/utils"
)

type TeamRepository interface {
	CreateTeam(ctx context.Context, team *models.Team, userID int64, userRole string) (error, int64)
	GetTeamByID(ctx context.Context, id int64) (*models.Team, error)
	GetTeamsByUserID(ctx context.Context, userID int64) ([]*models.Team, error)
	UpdateTeam(ctx context.Context, team *models.Team) error
	DeleteTeam(ctx context.Context, id int64) error
	UserExists(ctx context.Context, userID int64) (bool, error)
	GetTeamRole(ctx context.Context, teamID int64, userID int64) (string, error)
	AddMember(ctx context.Context, teamID int64, userID int64, role string) error
	RemoveMember(ctx context.Context, teamID int64, userID int64) error
	UpdateMemberRole(ctx context.Context, teamID int64, userID int64, newRole string) error
}

type teamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) TeamRepository {
	return &teamRepository{db: db}
}

func (r *teamRepository) CreateTeam(ctx context.Context, team *models.Team, userID int64, userRole string) (error, int64) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Failed to start transaction for team creation: %v", err)
		return apperrors.NewInternalError("failed to start database transaction", err), 0
	}

	teamQuery := "INSERT INTO teams (name) VALUES (?)"
	result, err := tx.ExecContext(ctx, teamQuery, team.Name)
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to insert team: %v", err)
		return apperrors.NewInternalError("failed to create team in database", err), 0
	}

	teamID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to get team ID after insert: %v", err)
		return apperrors.NewInternalError("failed to retrieve team ID", err), 0
	}

	memberQuery := "INSERT INTO team_members (team_id, user_id, team_role) VALUES (?, ?, 'main_manager')"
	_, err = tx.ExecContext(ctx, memberQuery, teamID, userID)
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to add main_manager to team %d: %v", teamID, err)
		return apperrors.NewInternalError("failed to assign team owner", err), 0
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Failed to commit transaction for team creation: %v", err)
		return apperrors.NewInternalError("failed to commit database transaction", err), 0
	}

	return nil, teamID
}

func (r *teamRepository) GetTeamByID(ctx context.Context, id int64) (*models.Team, error) {
	team := &models.Team{}
	query := "SELECT id, name, created_at, updated_at FROM teams WHERE id = ?"
	err := r.db.QueryRowContext(ctx, query, id).Scan(&team.ID, &team.Name, &team.CreatedAt, &team.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Team not found with ID: %d", id)
			return nil, apperrors.NewNotFoundError("team")
		}
		log.Printf("Database error retrieving team %d: %v", id, err)
		return nil, apperrors.NewInternalError("failed to retrieve team", err)
	}

	membersQuery := `
		SELECT user_id, team_role, joined_at
		FROM team_members
		WHERE team_id = ?
		ORDER BY joined_at ASC`
	membersRows, err := r.db.QueryContext(ctx, membersQuery, id)
	if err != nil {
		log.Printf("Database error retrieving members for team %d: %v", id, err)
		return nil, apperrors.NewInternalError("failed to retrieve team members", err)
	}
	defer membersRows.Close()

	team.Members = make([]models.TeamMember, 0)
	for membersRows.Next() {
		member := models.TeamMember{}
		if err := membersRows.Scan(&member.UserID, &member.TeamRole, &member.JoinedAt); err != nil {
			log.Printf("Error scanning member row for team %d: %v", id, err)
			return nil, apperrors.NewInternalError("failed to parse team members", err)
		}
		member.TeamID = id
		team.Members = append(team.Members, member)
	}

	if err = membersRows.Err(); err != nil {
		log.Printf("Error iterating members for team %d: %v", id, err)
		return nil, apperrors.NewInternalError("failed to retrieve team members", err)
	}
	return team, nil
}

func (r *teamRepository) GetTeamsByUserID(ctx context.Context, userID int64) ([]*models.Team, error) {
	query := `
		SELECT t.id, t.name, t.created_at, t.updated_at
		FROM teams t
		JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = ?`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		log.Printf("Database error retrieving teams for user %d: %v", userID, err)
		return nil, apperrors.NewInternalError("failed to retrieve teams", err)
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		team := &models.Team{}
		if err := rows.Scan(&team.ID, &team.Name, &team.CreatedAt, &team.UpdatedAt); err != nil {
			log.Printf("Error scanning team row for user %d: %v", userID, err)
			return nil, apperrors.NewInternalError("failed to parse team data", err)
		}
		teams = append(teams, team)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating teams for user %d: %v", userID, err)
		return nil, apperrors.NewInternalError("failed to retrieve teams", err)
	}

	return teams, nil
}

func (r *teamRepository) UpdateTeam(ctx context.Context, team *models.Team) error {
	query := "UPDATE teams SET name = ? WHERE id = ?"
	result, err := r.db.ExecContext(ctx, query, team.Name, team.ID)
	if err != nil {
		log.Printf("Database error updating team %d: %v", team.ID, err)
		return apperrors.NewInternalError("failed to update team", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected for team %d update: %v", team.ID, err)
		return apperrors.NewInternalError("failed to verify update", err)
	}

	if rowsAffected == 0 {
		log.Printf("No team found with ID %d to update", team.ID)
		return apperrors.NewNotFoundError("team")
	}

	return nil
}

func (r *teamRepository) DeleteTeam(ctx context.Context, id int64) error {
	query := "DELETE FROM teams WHERE id = ?"
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		log.Printf("Database error deleting team %d: %v", id, err)
		return apperrors.NewInternalError("failed to delete team", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected for team %d delete: %v", id, err)
		return apperrors.NewInternalError("failed to verify deletion", err)
	}

	if rowsAffected == 0 {
		log.Printf("No team found with ID %d to delete", id)
		return apperrors.NewNotFoundError("team")
	}

	return nil
}

func (r *teamRepository) UserExists(ctx context.Context, userID int64) (bool, error) {
	var exists int
	query := "SELECT 1 FROM users WHERE id = ?"
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User not found with ID: %d", userID)
			return false, nil
		}
		log.Printf("Database error checking existence of user %d: %v", userID, err)
		return false, apperrors.NewInternalError("failed to check user existence", err)
	}

	return true, nil
}

func (r *teamRepository) GetTeamRole(ctx context.Context, teamID int64, userID int64) (string, error) {
	var role string
	query := "SELECT team_role FROM team_members WHERE team_id = ? AND user_id = ?"
	err := r.db.QueryRowContext(ctx, query, teamID, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User %d is not a member of team %d", userID, teamID)
			return "", apperrors.NewNotFoundError("user role in team")
		}
		log.Printf("Database error retrieving role for user %d in team %d: %v", userID, teamID, err)
		return "", apperrors.NewInternalError("failed to retrieve user role", err)
	}
	return role, nil
}

func (r *teamRepository) AddMember(ctx context.Context, teamID int64, userID int64, role string) error {
	query := "INSERT INTO team_members (team_id, user_id, team_role) VALUES (?, ?, ?)"
	_, err := r.db.ExecContext(ctx, query, teamID, userID, role)
	if err != nil {
		// Handle duplicate key error (user already a member)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "Duplicate entry") {
			log.Printf("User %d is already a member of team %d", userID, teamID)
			return apperrors.NewConflictError("user is already a member of this team", err)
		}
		log.Printf("Database error adding user %d to team %d: %v", userID, teamID, err)
		return apperrors.NewInternalError("failed to add member to team", err)
	}
	return nil
}

func (r *teamRepository) RemoveMember(ctx context.Context, teamID int64, userID int64) error {
	query := "DELETE FROM team_members WHERE team_id = ? AND user_id = ?"
	result, err := r.db.ExecContext(ctx, query, teamID, userID)
	if err != nil {
		log.Printf("Database error removing user %d from team %d: %v", userID, teamID, err)
		return apperrors.NewInternalError("failed to remove member from team", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected for removing user %d from team %d: %v", userID, teamID, err)
		return apperrors.NewInternalError("failed to verify removal", err)
	}

	if rowsAffected == 0 {
		log.Printf("User %d was not a member of team %d", userID, teamID)
		return apperrors.NewNotFoundError("user is not a member of this team")
	}

	return nil
}

func (r *teamRepository) UpdateMemberRole(ctx context.Context, teamID int64, userID int64, newRole string) error {
	query := "UPDATE team_members SET team_role = ? WHERE team_id = ? AND user_id = ?"
	result, err := r.db.ExecContext(ctx, query, newRole, teamID, userID)
	if err != nil {
		log.Printf("Database error updating role for user %d in team %d: %v", userID, teamID, err)
		return apperrors.NewInternalError("failed to update member role", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected for role update for user %d in team %d: %v", userID, teamID, err)
		return apperrors.NewInternalError("failed to verify role update", err)
	}

	if rowsAffected == 0 {
		log.Printf("User %d is not a member of team %d", userID, teamID)
		return apperrors.NewNotFoundError("user is not a member of this team")
	}

	return nil
}
