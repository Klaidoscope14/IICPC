package domain

import "time"

type LoginRequest struct {
	Password string `json:"password" binding:"required"`
}

type HackathonDatesRequest struct {
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
}

type HackathonConfig struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type Team struct {
	ContestantID string   `json:"contestant_id"`
	TeamName     string   `json:"team_name"`
	UserIDs      []string `json:"user_ids"`
	CreatedAt    time.Time `json:"created_at"`
}

type Submission struct {
	ID               string    `json:"id"`
	ContestantID     string    `json:"contestant_id"`
	TeamName         string    `json:"team_name"`
	Language         string    `json:"language"`
	Status           string    `json:"status"`
	Version          int       `json:"version"`
	CreatedAt        time.Time `json:"created_at"`
}

type Stats struct {
	TotalUsers       int `json:"total_users"`
	TotalTeams       int `json:"total_teams"`
	TotalSubmissions int `json:"total_submissions"`
}

type AuthToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
