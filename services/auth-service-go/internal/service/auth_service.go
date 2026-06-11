package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/iicpc/auth-service-go/internal/domain"
	"github.com/iicpc/auth-service-go/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo      *repository.PostgresUserRepository
	jwtSecret string
}

func NewAuthService(repo *repository.PostgresUserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

func (s *AuthService) Register(ctx context.Context, req domain.RegistrationRequest) (*domain.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "contestant",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthToken, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	team, err := s.repo.GetTeamByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	teamID := ""
	if team != nil {
		teamID = team.ContestantID
	}

	// Generate JWT Token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"team_id": teamID,
		"exp":     expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &domain.AuthToken{
		Token:     tokenString,
		ExpiresAt: expirationTime,
	}, nil
}

func (s *AuthService) GetProfile(ctx context.Context, userID string) (*domain.User, *domain.Team, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	team, err := s.repo.GetTeamByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	return user, team, nil
}
