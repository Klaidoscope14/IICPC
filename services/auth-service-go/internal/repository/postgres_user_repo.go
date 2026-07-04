package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/iicpc/auth-service-go/internal/domain"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var ErrUserNotFound = errors.New("user not found")
var ErrEmailExists = errors.New("email already exists")

type PostgresUserRepository struct {
	db *sqlx.DB
}

func NewPostgresUserRepository(db *sqlx.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		VALUES (:id, :email, :password_hash, :role, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrEmailExists
		}
		return err
	}
	return nil
}

func (r *PostgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *PostgresUserRepository) GetTeamByUserID(ctx context.Context, userID string) (*domain.Team, error) {
	var team domain.Team
	query := `SELECT * FROM teams WHERE $1 = ANY(user_ids)`
	err := r.db.GetContext(ctx, &team, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User has no team yet
		}
		return nil, err
	}
	return &team, nil
}

func (r *PostgresUserRepository) GetTotalTeamsCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM teams`
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostgresUserRepository) CreateTeam(ctx context.Context, team *domain.Team) error {
	query := `
		INSERT INTO teams (contestant_id, team_name, user_ids, metadata, created_at, updated_at)
		VALUES (:contestant_id, :team_name, :user_ids, :metadata, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, team)
	return err
}
