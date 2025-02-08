package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"cadence/internal/data"
	"cadence/internal/speech"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	config := LoadServerConfig()
	logger := InitLogger(config.Environment)

	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		logger.Fatal("database connection failed:", err)
	}

	repo := data.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		logger.Fatal("database migration failed:", err)
	}

	if err := speech.InitVoices(); err != nil {
		logger.Fatal("voice initialization failed:", err)
	}

	server := NewServer(config, repo, logger)
	srv := &http.Server{
		Addr:         config.Port,
		Handler:      server.router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	go func() {
		logger.Printf("server starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server startup failed:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Printf("shutdown signal received: %v", sig)

	ctx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("forced shutdown:", err)
	}

	logger.Println("server stopped")
}