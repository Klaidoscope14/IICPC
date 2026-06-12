package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/iicpc/admin-service-go/config"
	"github.com/iicpc/admin-service-go/internal/domain"
	"github.com/iicpc/admin-service-go/internal/repository"
)

type AdminService struct {
	repo   *repository.PostgresAdminRepository
	config *config.Config
}

func NewAdminService(repo *repository.PostgresAdminRepository, cfg *config.Config) *AdminService {
	return &AdminService{
		repo:   repo,
		config: cfg,
	}
}

func (s *AdminService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthToken, error) {
	if req.Password != s.config.AdminPassword {
		return nil, errors.New("invalid admin credentials")
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"user_id": uuid.NewString(), // Admin gets a dummy user ID or we can just use "admin"
		"email":   "admin@iicpc.org",
		"role":    "admin",
		"team_id": "admin",
		"exp":     expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &domain.AuthToken{
		Token:     tokenString,
		ExpiresAt: expirationTime,
	}, nil
}

func (s *AdminService) UpdateHackathonDates(ctx context.Context, req domain.HackathonDatesRequest) error {
	if req.EndDate.Before(req.StartDate) {
		return errors.New("end date cannot be before start date")
	}
	return s.repo.UpdateHackathonDates(ctx, req.StartDate, req.EndDate)
}

func (s *AdminService) GetHackathonConfig(ctx context.Context) (*domain.HackathonConfig, error) {
	return s.repo.GetHackathonConfig(ctx)
}

func (s *AdminService) GetUsers(ctx context.Context) ([]domain.User, error) {
	return s.repo.GetUsers(ctx)
}

func (s *AdminService) GetTeams(ctx context.Context) ([]domain.Team, error) {
	return s.repo.GetTeams(ctx)
}

func (s *AdminService) GetSubmissions(ctx context.Context) ([]domain.Submission, error) {
	return s.repo.GetSubmissions(ctx)
}

func (s *AdminService) GetStats(ctx context.Context) (*domain.Stats, error) {
	return s.repo.GetStats(ctx)
}
