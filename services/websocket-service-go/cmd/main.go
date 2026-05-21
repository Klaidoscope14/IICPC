package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/pkg/logging"
	"github.com/iicpc/pkg/middleware"
	"github.com/iicpc/pkg/server"
	"github.com/iicpc/websocket-service-go/internal/consumer"
	"github.com/iicpc/websocket-service-go/internal/handler"
	"github.com/iicpc/websocket-service-go/internal/ws"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	brokers := []string{"localhost:9092"}
	if b := os.Getenv("REDPANDA_BROKERS"); b != "" {
		brokers = []string{b}
	}

	logger := logging.NewLogger("websocket-service")

	// Initialize WebSocket Hub
	hub := ws.NewHub(logger)
	go hub.Run()

	// Initialize Event Consumer
	eventConsumer, err := consumer.NewEventConsumer(brokers, hub, logger)
	if err != nil {
		logger.Warn("Failed to initialize event consumer", "error", err)
	} else {
		go func() {
			if err := eventConsumer.Start(context.Background()); err != nil {
				logger.Error("Event consumer error", "error", err)
			}
		}()
		defer eventConsumer.Close()
	}

	// Initialize HTTP Router
	router := gin.Default()
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestLogger(logger))

	// Register WebSocket routes
	wsHandler := handler.NewWebSocketHandler(hub, logger)
	wsHandler.RegisterRoutes(router)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		status := "healthy"
		if eventConsumer == nil {
			status = "degraded (no consumer)"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  status,
			"service": "websocket-service",
		})
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second, // WebSocket write wait is managed internally
		IdleTimeout:  120 * time.Second,
	}

	fmt.Printf("WebSocket Service listening on :%s\n", port)
	server.RunGracefully(srv, "websocket-service")
}
