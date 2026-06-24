package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marketplace/media-service/internal/config"
	"github.com/marketplace/media-service/internal/db"
	"github.com/marketplace/media-service/internal/handler"
	"github.com/marketplace/media-service/internal/imageproc"
	"github.com/marketplace/media-service/internal/repo"
	"github.com/marketplace/media-service/internal/router"
	"github.com/marketplace/media-service/internal/service"
	"github.com/marketplace/media-service/internal/storage"
	"github.com/marketplace/media-service/internal/token"
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

	store, err := storage.New(ctx, storage.Config{
		Endpoint:      cfg.S3Endpoint,
		AccessKey:     cfg.S3AccessKey,
		SecretKey:     cfg.S3SecretKey,
		Bucket:        cfg.S3Bucket,
		UseSSL:        cfg.S3UseSSL,
		PublicBaseURL: cfg.S3PublicBaseURL,
	})
	if err != nil {
		log.Fatalf("init object storage: %v", err)
	}

	verifier, err := token.NewVerifier(cfg.PublicKeyPath, cfg.Issuer)
	if err != nil {
		log.Fatalf("init token verifier: %v", err)
	}

	mediaRepo := repo.NewMediaRepo(pool)
	proc := imageproc.NewProcessor(cfg.ResizeWorkers)
	mediaSvc := service.NewMediaService(mediaRepo, store, proc)
	h := handler.New(mediaSvc, cfg.MaxUploadBytes)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router.New(h, verifier),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		log.Printf("media-service listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down media-service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("media-service stopped")
}
