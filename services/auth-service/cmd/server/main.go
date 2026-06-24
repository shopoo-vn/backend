package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marketplace/auth-service/internal/config"
	"github.com/marketplace/auth-service/internal/db"
	"github.com/marketplace/auth-service/internal/handler"
	"github.com/marketplace/auth-service/internal/repo"
	"github.com/marketplace/auth-service/internal/router"
	"github.com/marketplace/auth-service/internal/service"
	"github.com/marketplace/auth-service/internal/token"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	rdb, err := db.NewRedis(ctx, cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer rdb.Close()

	tm, err := token.NewManager(cfg.PrivateKeyPath, cfg.PublicKeyPath, cfg.Issuer, time.Duration(cfg.AccessTTLMin)*time.Minute)
	if err != nil {
		log.Fatalf("init token manager: %v", err)
	}

	userRepo := repo.NewUserRepo(pool)
	authSvc := service.NewAuthService(userRepo, rdb, tm, time.Duration(cfg.RefreshTTLDays)*24*time.Hour)
	h := handler.New(authSvc)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router.New(h, tm),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("auth-service listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down auth-service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("auth-service stopped")
}
