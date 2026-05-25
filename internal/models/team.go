package models

import "time"

type Team struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	Members   []TeamMember `json:"members,omitempty"`
}

type TeamMember struct {
	TeamID   int64     `json:"team_id"`
	UserID   int64     `json:"user_id"`
	TeamRole string    `json:"team_role"`
	JoinedAt time.Time `json:"joined_at"`
}
