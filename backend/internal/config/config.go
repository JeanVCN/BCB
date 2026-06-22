package config

import (
	"fmt"
	"net"
	"os"
)

const defaultHTTPPort = "8080"

type Config struct {
	HTTPPort string
}

func Load() (Config, error) {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = defaultHTTPPort
	}

	if err := validatePort(port); err != nil {
		return Config{}, fmt.Errorf("HTTP_PORT: %w", err)
	}

	return Config{HTTPPort: port}, nil
}

func validatePort(port string) error {
	if _, err := net.LookupPort("tcp", port); err != nil {
		return fmt.Errorf("must be a valid TCP port: %w", err)
	}

	return nil
}
