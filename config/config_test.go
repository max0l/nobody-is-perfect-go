package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv(EnvHost, "")
	t.Setenv(EnvPort, "")
	t.Setenv(EnvMaxConcurrentGames, "")
	t.Setenv(EnvWordlistPath, "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Host != DefaultHost {
		t.Fatalf("expected default host %q, got %q", DefaultHost, cfg.Host)
	}
	if cfg.Port != DefaultPort {
		t.Fatalf("expected default port %d, got %d", DefaultPort, cfg.Port)
	}
	if cfg.MaxConcurrentGames != DefaultMaxConcurrentGames {
		t.Fatalf("expected default max games %d, got %d", DefaultMaxConcurrentGames, cfg.MaxConcurrentGames)
	}
	if cfg.WordlistPath != DefaultWordlistPath {
		t.Fatalf("expected default wordlist %q, got %q", DefaultWordlistPath, cfg.WordlistPath)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv(EnvPort, "invalid")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid port error")
	}
}

func TestLoadRejectsInvalidMaxConcurrentGames(t *testing.T) {
	t.Setenv(EnvMaxConcurrentGames, "0")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid max concurrent games error")
	}
}
