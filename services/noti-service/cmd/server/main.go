package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marketplace/noti-service/internal/config"
	"github.com/marketplace/noti-service/internal/db"
	"github.com/marketplace/noti-service/internal/event"
	"github.com/marketplace/noti-service/internal/handler"
	"github.com/marketplace/noti-service/internal/push"
	"github.com/marketplace/noti-service/internal/repo"
	"github.com/marketplace/noti-service/internal/router"
	"github.com/marketplace/noti-service/internal/service"
	"github.com/marketplace/noti-service/internal/token"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "noti-service")

	cfg := config.Load()
	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DBURL)
	if err != nil {
		log.Error("connect postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		log.Error("run migrations", "err", err)
		os.Exit(1)
	}

	rdb, err := db.NewRedis(ctx, cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		log.Error("connect redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	verifier, err := token.NewVerifier(cfg.PublicKeyPath, cfg.Issuer)
	if err != nil {
		log.Error("init token verifier", "err", err)
		os.Exit(1)
	}

	sender, err := push.New(ctx, cfg.FCMCredsFile, log)
	if err != nil {
		log.Error("init push sender", "err", err)
		os.Exit(1)
	}

	r := repo.NewRepo(pool)
	notiSvc := service.NewNotiService(r, rdb, sender, log, time.Duration(cfg.ProcessedTTLMin)*time.Minute)
	h := handler.New(notiSvc)

	consumer := event.NewConsumer(cfg.RabbitMQURL, cfg.Exchange, notiSvc, log)
	if err := consumer.Start(ctx); err != nil {
		log.Error("start consumer", "err", err)
		os.Exit(1)
	}
	defer consumer.Close()

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router.New(h, verifier),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("noti-service listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Info("shutting down noti-service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Warn("graceful shutdown failed", "err", err)
	}
	log.Info("noti-service stopped")
}
