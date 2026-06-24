package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration, loaded from environment variables
// (with a .env file as a convenience for local runs). All keys are prefixed
// NOTI_ per the marketplace per-service convention.
type Config struct {
	HTTPAddr        string
	DBURL           string
	RedisAddr       string
	RedisPassword   string
	RabbitMQURL     string
	Exchange        string
	PublicKeyPath   string
	Issuer          string
	FCMCredsFile    string
	ProcessedTTLMin int
}

func Load() Config {
	// Best-effort: in docker the env is set directly and there is no .env file.
	_ = godotenv.Load()

	return Config{
		HTTPAddr:        getenv("NOTI_HTTP_ADDR", ":8003"),
		DBURL:           getenv("NOTI_DB_URL", "postgres://noti_svc:noti_pw@localhost:5432/marketplace?search_path=noti&sslmode=disable"),
		RedisAddr:       getenv("NOTI_REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getenv("NOTI_REDIS_PASSWORD", ""),
		RabbitMQURL:     getenv("NOTI_RABBITMQ_URL", "amqp://guest:guest@localhost:5672"),
		Exchange:        getenv("NOTI_RABBITMQ_EXCHANGE", "marketplace.events"),
		PublicKeyPath:   getenv("NOTI_JWT_PUBLIC_KEY_PATH", "../../keys/jwt_public.pem"),
		Issuer:          getenv("NOTI_JWT_ISSUER", "marketplace-auth"),
		FCMCredsFile:    getenv("NOTI_FCM_CREDENTIALS_FILE", ""),
		ProcessedTTLMin: getenvInt("NOTI_PROCESSED_TTL_MIN", 1440),
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
