package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"whwriter/internal/api"
	"whwriter/internal/config"
	"whwriter/internal/service"
	"whwriter/internal/store"
)

func main() {
	cfg := config.Load()

	db, err := store.NewDB(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	userStore := store.NewUserStore(db)
	emailSvc := service.NewEmailService()
	authSvc := service.NewAuthService(userStore, emailSvc, cfg)

	router := api.SetupRouter(cfg, userStore, authSvc)

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("whwriter server starting on %s (mode: %s)", cfg.Addr(), cfg.Mode)
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
