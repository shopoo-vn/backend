package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration, loaded from environment variables
// (with a .env file as a convenience for local runs).
type Config struct {
	HTTPAddr       string
	DBURL          string
	RedisAddr      string
	RedisPassword  string
	PrivateKeyPath string
	PublicKeyPath  string
	AccessTTLMin   int
	RefreshTTLDays int
	Issuer         string
}

func Load() Config {
	// Best-effort: in docker the env is set directly and there is no .env file.
	_ = godotenv.Load()

	return Config{
		HTTPAddr:       getenv("AUTH_HTTP_ADDR", ":8001"),
		DBURL:          getenv("AUTH_DB_URL", "postgres://auth_svc:auth_pw@localhost:5432/marketplace?search_path=auth&sslmode=disable"),
		RedisAddr:      getenv("AUTH_REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getenv("AUTH_REDIS_PASSWORD", ""),
		PrivateKeyPath: getenv("AUTH_PRIVATE_KEY_PATH", "../../keys/jwt_private.pem"),
		PublicKeyPath:  getenv("AUTH_PUBLIC_KEY_PATH", "../../keys/jwt_public.pem"),
		AccessTTLMin:   getenvInt("AUTH_ACCESS_TTL_MIN", 15),
		RefreshTTLDays: getenvInt("AUTH_REFRESH_TTL_DAYS", 7),
		Issuer:         getenv("AUTH_JWT_ISSUER", "marketplace-auth"),
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
