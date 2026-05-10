package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// RunGracefully starts the HTTP server and blocks until a shutdown signal is received.
// It handles SIGINT/SIGTERM and gracefully drains connections with a 10-second timeout.
func RunGracefully(srv *http.Server, serviceName string) {
	go func() {
		log.Printf("Starting %s on %s", serviceName, srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down %s...", serviceName)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Printf("%s exited cleanly", serviceName)
}
