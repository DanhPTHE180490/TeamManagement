package team

import (
	"database/sql"
	"log"
	"strings"
	"team-management/internal/errors"
	"team-management/internal/models"
)

type TeamRepository interface {
	CreateTeam(team *models.Team, userID int64, userRole string) (error, int64)
	GetTeamByID(id int64) (*models.Team, error)
	GetTeamsByUserID(userID int64) ([]*models.Team, error)
	UpdateTeam(team *models.Team) error
	DeleteTeam(id int64) error
	UserExists(userID int64) (bool, error)
	GetTeamRole(teamID int64, userID int64) (string, error)
	AddMember(teamID int64, userID int64, role string) error
	RemoveMember(teamID int64, userID int64) error
	UpdateMemberRole(teamID int64, userID int64, newRole string) error
}

type teamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) TeamRepository {
	return &teamRepository{db: db}
}

func (r *teamRepository) CreateTeam(team *models.Team, userID int64, userRole string) (error, int64) {
	tx, err := r.db.Begin()
	if err != nil {
		log.Printf("Failed to start transaction for team creation: %v", err)
		return errors.NewInternalError("failed to start database transaction", err), 0
	}

	teamQuery := "INSERT INTO teams (name) VALUES (?)"
	result, err := tx.Exec(teamQuery, team.Name)
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to insert team: %v", err)
		return errors.NewInternalError("failed to create team in database", err), 0
	}

	teamID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to get team ID after insert: %v", err)
		return errors.NewInternalError("failed to retrieve team ID", err), 0
	}

	memberQuery := "INSERT INTO team_members (team_id, user_id, team_role) VALUES (?, ?, 'main_manager')"
	_, err = tx.Exec(memberQuery, teamID, userID)
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to add main_manager to team %d: %v", teamID, err)
		return errors.NewInternalError("failed to assign team owner", err), 0
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Failed to commit transaction for team creation: %v", err)
		return errors.NewInternalError("failed to commit database transaction", err), 0
	}

	return nil, teamID
}

func (r *teamRepository) GetTeamByID(id int64) (*models.Team, error) {
	team := &models.Team{}
	query := "SELECT id, name FROM teams WHERE id = ?"
	err := r.db.QueryRow(query, id).Scan(&team.ID, &team.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Team not found with ID: %d", id)
			return nil, errors.NewNotFoundError("team")
		}
		log.Printf("Database error retrieving team %d: %v", id, err)
		return nil, errors.NewInternalError("failed to retrieve team", err)
	}

	membersQuery := `
		SELECT user_id, team_role, joined_at
		FROM team_members
		WHERE team_id = ?
		ORDER BY joined_at ASC`
	membersRows, err := r.db.Query(membersQuery, id)
	if err != nil {
		log.Printf("Database error retrieving members for team %d: %v", id, err)
		return nil, errors.NewInternalError("failed to retrieve team members", err)
	}
	defer membersRows.Close()

	team.Members = make([]models.TeamMember, 0)
	for membersRows.Next() {
		member := models.TeamMember{}
		if err := membersRows.Scan(&member.UserID, &member.TeamRole, &member.JoinedAt); err != nil {
			log.Printf("Error scanning member row for team %d: %v", id, err)
			return nil, errors.NewInternalError("failed to parse team members", err)
		}
		member.TeamID = id
		team.Members = append(team.Members, member)
	}

	if err = membersRows.Err(); err != nil {
		log.Printf("Error iterating members for team %d: %v", id, err)
		return nil, errors.NewInternalError("failed to retrieve team members", err)
	}
	return team, nil
}

func (r *teamRepository) GetTeamsByUserID(userID int64) ([]*models.Team, error) {
	query := `
		SELECT t.id, t.name, t.created_at, t.updated_at
		FROM teams t
		JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = ?`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		log.Printf("Database error retrieving teams for user %d: %v", userID, err)
		return nil, errors.NewInternalError("failed to retrieve teams", err)
	}
	defer rows.Close()

	var teams []*models.Team
	for rows.Next() {
		team := &models.Team{}
		if err := rows.Scan(&team.ID, &team.Name, &team.CreatedAt, &team.UpdatedAt); err != nil {
			log.Printf("Error scanning team row for user %d: %v", userID, err)
			return nil, errors.NewInternalError("failed to parse team data", err)
		}
		teams = append(teams, team)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating teams for user %d: %v", userID, err)
		return nil, errors.NewInternalError("failed to retrieve teams", err)
	}

	return teams, nil
}

func (r *teamRepository) UpdateTeam(team *models.Team) error {
	query := "UPDATE teams SET name = ? WHERE id = ?"
	result, err := r.db.Exec(query, team.Name, team.ID)
	if err != nil {
		log.Printf("Database error updating team %d: %v", team.ID, err)
		return errors.NewInternalError("failed to update team", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected for team %d update: %v", team.ID, err)
		return errors.NewInternalError("failed to verify update", err)
	}

	if rowsAffected == 0 {
		log.Printf("No team found with ID %d to update", team.ID)
		return errors.NewNotFoundError("team")
	}

	return nil
}

func (r *teamRepository) DeleteTeam(id int64) error {
	query := "DELETE FROM teams WHERE id = ?"
	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Printf("Database error deleting team %d: %v", id, err)
		return errors.NewInternalError("failed to delete team", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected for team %d delete: %v", id, err)
		return errors.NewInternalError("failed to verify deletion", err)
	}

	if rowsAffected == 0 {
		log.Printf("No team found with ID %d to delete", id)
		return errors.NewNotFoundError("team")
	}

	return nil
}

func (r *teamRepository) UserExists(userID int64) (bool, error) {
	var exists int
	query := "SELECT 1 FROM users WHERE id = ?"
	err := r.db.QueryRow(query, userID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User not found with ID: %d", userID)
			return false, nil
		}
		log.Printf("Database error checking existence of user %d: %v", userID, err)
		return false, errors.NewInternalError("failed to check user existence", err)
	}

	return true, nil
}

func (r *teamRepository) GetTeamRole(teamID int64, userID int64) (string, error) {
	var role string
	query := "SELECT team_role FROM team_members WHERE team_id = ? AND user_id = ?"
	err := r.db.QueryRow(query, teamID, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("User %d is not a member of team %d", userID, teamID)
			return "", errors.NewNotFoundError("user role in team")
		}
		log.Printf("Database error retrieving role for user %d in team %d: %v", userID, teamID, err)
		return "", errors.NewInternalError("failed to retrieve user role", err)
	}
	return role, nil
}

func (r *teamRepository) AddMember(teamID int64, userID int64, role string) error {
	query := "INSERT INTO team_members (team_id, user_id, team_role) VALUES (?, ?, ?)"
	_, err := r.db.Exec(query, teamID, userID, role)
	if err != nil {
		// Handle duplicate key error (user already a member)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "Duplicate entry") {
			log.Printf("User %d is already a member of team %d", userID, teamID)
			return errors.NewConflictError("user is already a member of this team", err)
		}
		log.Printf("Database error adding user %d to team %d: %v", userID, teamID, err)
		return errors.NewInternalError("failed to add member to team", err)
	}
	return nil
}

func (r *teamRepository) RemoveMember(teamID int64, userID int64) error {
	query := "DELETE FROM team_members WHERE team_id = ? AND user_id = ?"
	result, err := r.db.Exec(query, teamID, userID)
	if err != nil {
		log.Printf("Database error removing user %d from team %d: %v", userID, teamID, err)
		return errors.NewInternalError("failed to remove member from team", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected for removing user %d from team %d: %v", userID, teamID, err)
		return errors.NewInternalError("failed to verify removal", err)
	}

	if rowsAffected == 0 {
		log.Printf("User %d was not a member of team %d", userID, teamID)
		return errors.NewNotFoundError("user is not a member of this team")
	}

	return nil
}

func (r *teamRepository) UpdateMemberRole(teamID int64, userID int64, newRole string) error {
	query := "UPDATE team_members SET team_role = ? WHERE team_id = ? AND user_id = ?"
	result, err := r.db.Exec(query, newRole, teamID, userID)
	if err != nil {
		log.Printf("Database error updating role for user %d in team %d: %v", userID, teamID, err)
		return errors.NewInternalError("failed to update member role", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected for role update for user %d in team %d: %v", userID, teamID, err)
		return errors.NewInternalError("failed to verify role update", err)
	}

	if rowsAffected == 0 {
		log.Printf("User %d is not a member of team %d", userID, teamID)
		return errors.NewNotFoundError("user is not a member of this team")
	}

	return nil
}
