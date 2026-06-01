package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	PostgresURI     string
	SQLitePath      string
	HTTPAddress     string
	ShutdownTimeout time.Duration
	SubscribeWord   string
	UnsubscribeWord string
	DBMaxOpenConns  int
	DBMaxIdleConns  int
	DBConnMaxIdle   time.Duration
	DBConnMaxLife   time.Duration
}

func Load() (Config, error) {
	_ = godotenv.Load()

	dbMaxOpenConns, err := envIntOrDefault("DB_MAX_OPEN_CONNS", 10)
	if err != nil {
		return Config{}, err
	}

	dbMaxIdleConns, err := envIntOrDefault("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return Config{}, err
	}

	dbConnMaxIdle, err := envDurationOrDefault("DB_CONN_MAX_IDLE_TIME", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}

	dbConnMaxLife, err := envDurationOrDefault("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		PostgresURI:     os.Getenv("POSTGRES_URI"),
		SQLitePath:      envOrDefault("SQLITE_PATH", "spark-whatsapp-module.db"),
		HTTPAddress:     envOrDefault("HTTP_ADDRESS", ":8080"),
		ShutdownTimeout: 10 * time.Second,
		SubscribeWord:   envOrDefault("WHATSAPP_SUBSCRIBE_WORD", "subscribe"),
		UnsubscribeWord: envOrDefault("WHATSAPP_UNSUBSCRIBE_WORD", "unsubscribe"),
		DBMaxOpenConns:  dbMaxOpenConns,
		DBMaxIdleConns:  dbMaxIdleConns,
		DBConnMaxIdle:   dbConnMaxIdle,
		DBConnMaxLife:   dbConnMaxLife,
	}

	if cfg.SQLitePath == "" {
		return Config{}, fmt.Errorf("SQLITE_PATH is required")
	}

	if cfg.PostgresURI != "" {
		cfg.PostgresURI = normalizePostgresURI(cfg.PostgresURI)
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envIntOrDefault(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer: %w", key, err)
	}

	return parsed, nil
}

func envDurationOrDefault(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}

	return parsed, nil
}

func normalizePostgresURI(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	query := parsed.Query()
	if query.Get("sslmode") == "" {
		query.Set("sslmode", "disable")
		parsed.RawQuery = query.Encode()
	}

	return parsed.String()
}
