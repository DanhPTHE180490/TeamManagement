package models

import "time"

type Team struct {
	ID        int64        `json:"id" db:"id"`
	Name      string       `json:"name" db:"name"`
	CreatedAt time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt time.Time    `json:"updated_at" db:"updated_at"`
	Members   []TeamMember `json:"members,omitempty" db:"-"`
}

type TeamMember struct {
	TeamID   int64     `json:"team_id" db:"team_id"`
	UserID   int64     `json:"user_id" db:"user_id"`
	TeamRole string    `json:"team_role" db:"team_role"`
	JoinedAt time.Time `json:"joined_at" db:"joined_at"`
}
