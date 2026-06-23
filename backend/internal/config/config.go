package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

const defaultHTTPPort = "8080"

type Config struct {
	HTTPPort      string
	DatabaseURL   string
	RedisAddr     string
	JWTSecret     string
	RunMigrations bool
}

func Load() (Config, error) {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = defaultHTTPPort
	}

	if err := validatePort(port); err != nil {
		return Config{}, fmt.Errorf("HTTP_PORT: %w", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		return Config{}, fmt.Errorf("REDIS_ADDR is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}

	runMigrations := true
	if value := os.Getenv("RUN_MIGRATIONS"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("RUN_MIGRATIONS must be a boolean")
		}
		runMigrations = parsed
	}

	return Config{
		HTTPPort:      port,
		DatabaseURL:   databaseURL,
		RedisAddr:     redisAddr,
		JWTSecret:     jwtSecret,
		RunMigrations: runMigrations,
	}, nil
}

func validatePort(port string) error {
	if _, err := net.LookupPort("tcp", port); err != nil {
		return fmt.Errorf("must be a valid TCP port: %w", err)
	}

	return nil
}
