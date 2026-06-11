package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/auth-service-go/internal/handler"
	"github.com/iicpc/auth-service-go/internal/repository"
	"github.com/iicpc/auth-service-go/internal/service"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super-secret-key-change-in-prod"
	}

	userRepo := repository.NewPostgresUserRepository(db)
	authService := service.NewAuthService(userRepo, jwtSecret)
	authHandler := handler.NewAuthHandler(authService)

	router := gin.Default()
	authHandler.RegisterRoutes(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Starting auth service on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
