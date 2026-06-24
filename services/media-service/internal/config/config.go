package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration, loaded from environment variables
// (with a .env file as a convenience for local runs). All values are read once
// at startup; nothing reads os.Getenv outside this package.
type Config struct {
	HTTPAddr      string
	DBURL         string
	PublicKeyPath string
	Issuer        string

	// MinIO / S3-compatible object storage.
	S3Endpoint      string
	S3AccessKey     string
	S3SecretKey     string
	S3Bucket        string
	S3UseSSL        bool
	S3PublicBaseURL string

	// Upload limits and processing.
	MaxUploadBytes int64
	ResizeWorkers  int
}

func Load() Config {
	// Best-effort: in docker the env is set directly and there is no .env file.
	_ = godotenv.Load()

	return Config{
		HTTPAddr:      getenv("MEDIA_HTTP_ADDR", ":8002"),
		DBURL:         getenv("MEDIA_DB_URL", "postgres://media_svc:media_pw@localhost:5432/marketplace?search_path=media&sslmode=disable"),
		PublicKeyPath: getenv("MEDIA_JWT_PUBLIC_KEY_PATH", "../../keys/jwt_public.pem"),
		Issuer:        getenv("MEDIA_JWT_ISSUER", "marketplace-auth"),

		S3Endpoint:      getenv("MEDIA_S3_ENDPOINT", "localhost:9000"),
		S3AccessKey:     getenv("MEDIA_S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:     getenv("MEDIA_S3_SECRET_KEY", "minioadmin"),
		S3Bucket:        getenv("MEDIA_S3_BUCKET", "media"),
		S3UseSSL:        getenvBool("MEDIA_S3_USE_SSL", false),
		S3PublicBaseURL: getenv("MEDIA_S3_PUBLIC_BASE_URL", "http://localhost:9000/media"),

		MaxUploadBytes: getenvInt64("MEDIA_MAX_UPLOAD_BYTES", 10*1024*1024),
		ResizeWorkers:  getenvInt("MEDIA_RESIZE_WORKERS", 4),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("config: invalid int for %s=%q, using default %d", key, v, fallback)
		return fallback
	}
	return n
}

func getenvInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		log.Printf("config: invalid int for %s=%q, using default %d", key, v, fallback)
		return fallback
	}
	return n
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Printf("config: invalid bool for %s=%q, using default %t", key, v, fallback)
		return fallback
	}
	return b
}
