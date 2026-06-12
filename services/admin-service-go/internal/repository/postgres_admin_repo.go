package repository

import (
	"context"
	"time"

	"github.com/iicpc/admin-service-go/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAdminRepository struct {
	db *pgxpool.Pool
}

func NewPostgresAdminRepository(db *pgxpool.Pool) *PostgresAdminRepository {
	return &PostgresAdminRepository{db: db}
}

func (r *PostgresAdminRepository) UpdateHackathonDates(ctx context.Context, startDate, endDate time.Time) error {
	query := `
		INSERT INTO hackathon_config (id, start_date, end_date, status, updated_at)
		VALUES ('global', $1, $2, 'upcoming', NOW())
		ON CONFLICT (id) DO UPDATE SET 
			start_date = EXCLUDED.start_date, 
			end_date = EXCLUDED.end_date, 
			updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, startDate, endDate)
	return err
}

func (r *PostgresAdminRepository) GetHackathonConfig(ctx context.Context) (*domain.HackathonConfig, error) {
	var conf domain.HackathonConfig
	query := `SELECT id, status, start_date, end_date, updated_at FROM hackathon_config WHERE id = 'global'`
	err := r.db.QueryRow(ctx, query).Scan(&conf.ID, &conf.Status, &conf.StartDate, &conf.EndDate, &conf.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

func (r *PostgresAdminRepository) GetUsers(ctx context.Context) ([]domain.User, error) {
	query := `SELECT id, email, role, created_at FROM users ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *PostgresAdminRepository) GetTeams(ctx context.Context) ([]domain.Team, error) {
	query := `SELECT contestant_id, team_name, user_ids, created_at FROM teams ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []domain.Team
	for rows.Next() {
		var t domain.Team
		if err := rows.Scan(&t.ContestantID, &t.TeamName, &t.UserIDs, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func (r *PostgresAdminRepository) GetSubmissions(ctx context.Context) ([]domain.Submission, error) {
	query := `SELECT id, contestant_id, team_name, language, status, version, created_at FROM submissions ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []domain.Submission
	for rows.Next() {
		var s domain.Submission
		if err := rows.Scan(&s.ID, &s.ContestantID, &s.TeamName, &s.Language, &s.Status, &s.Version, &s.CreatedAt); err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}
	return submissions, nil
}

func (r *PostgresAdminRepository) GetStats(ctx context.Context) (*domain.Stats, error) {
	var stats domain.Stats
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM teams`).Scan(&stats.TotalTeams)
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM submissions`).Scan(&stats.TotalSubmissions)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
