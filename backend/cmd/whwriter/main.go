package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"whwriter/backend/internal/config"
	"whwriter/backend/internal/container"
)

func main() {
	cfg := config.Load("config.yaml")

	c, err := container.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize container: %v", err)
	}

	sqlDB, _ := c.DB.DB()
	defer sqlDB.Close()

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      c.Engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("whwriter server starting on %s (mode: %s)", cfg.Addr(), cfg.App.Mode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}
