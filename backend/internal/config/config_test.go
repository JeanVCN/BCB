package config

import "testing"

func TestLoadUsesDefaultPort(t *testing.T) {
	t.Setenv("HTTP_PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPPort != defaultHTTPPort {
		t.Fatalf("HTTPPort = %q, want %q", cfg.HTTPPort, defaultHTTPPort)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("HTTP_PORT", "not-a-port")

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}
