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
	if cfg.APIBaseURL != "http://localhost:8080" {
		t.Fatalf("expected default API base URL, got %q", cfg.APIBaseURL)
	}
}

func TestLoadDerivesAPIBaseURLFromHostAndPort(t *testing.T) {
	t.Setenv(EnvHost, "127.0.0.1")
	t.Setenv(EnvPort, "3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.APIBaseURL != "http://127.0.0.1:3000" {
		t.Fatalf("expected derived API base URL, got %q", cfg.APIBaseURL)
	}
}

func TestLoadUsesExplicitAPIBaseURL(t *testing.T) {
	t.Setenv(EnvAPIBaseURL, "https://example.com/api")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.APIBaseURL != "https://example.com/api" {
		t.Fatalf("expected explicit API base URL, got %q", cfg.APIBaseURL)
	}
}

func TestLoadRejectsInvalidAPIBaseURL(t *testing.T) {
	t.Setenv(EnvAPIBaseURL, "not-a-url")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid API base URL error")
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
