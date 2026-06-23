package config

import (
	"fmt"
	"net"
	"os"
)

const defaultHTTPPort = "8080"

type Config struct {
	HTTPPort    string
	DatabaseURL string
	RedisAddr   string
	JWTSecret   string
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

	return Config{
		HTTPPort:    port,
		DatabaseURL: databaseURL,
		RedisAddr:   redisAddr,
		JWTSecret:   jwtSecret,
	}, nil
}

func validatePort(port string) error {
	if _, err := net.LookupPort("tcp", port); err != nil {
		return fmt.Errorf("must be a valid TCP port: %w", err)
	}

	return nil
}
