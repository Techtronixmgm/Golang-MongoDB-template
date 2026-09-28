package main

import (
	"basic-app/config"
	"basic-app/database"
	mongorepo "basic-app/repository/mongo"
	"basic-app/router"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Config Error: %v", err)
		return
	}

	client, db, err := database.Connect(cfg)
	if err != nil {
		log.Printf("DB Error: %v", err)
		return
	}

	defer func() {
		if err := database.Disconnect(client); err != nil {
			log.Printf("mongo disconnect error: %v", err)
		}
	}()

	// Ensure MongoDB indexes exist
	indexCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mongorepo.EnsureUserIndexes(indexCtx, db); err != nil {
		log.Printf("Failed to create user indexes: %v", err)
		return
	}

	if err := mongorepo.EnsureRefreshTokenIndexes(indexCtx, db); err != nil {
		log.Printf("Failed to create refresh token indexes: %v", err)
		return
	}

	gin.SetMode(cfg.GinMode)

	// middleware.StartCleanup()

	engine := router.NewRouter(db, cfg)

	addr := ":" + cfg.ServerPort

	server := &http.Server{
		Addr:              addr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start HTTP server
	go func() {
		log.Printf("Server listening on http://localhost%s", addr)

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Printf("Server Failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	log.Println("Shutdown signal received")

	// Allow active requests to complete
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
