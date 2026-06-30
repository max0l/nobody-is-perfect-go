package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv(EnvPort, "")
	t.Setenv(EnvMaxConcurrentGames, "")
	t.Setenv(EnvWordlistPath, "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
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
	if cfg.LogFormat != DefaultLogFormat {
		t.Fatalf("expected default log format %q, got %q", DefaultLogFormat, cfg.LogFormat)
	}
	if cfg.LogLevel != DefaultLogLevel {
		t.Fatalf("expected default log level %q, got %q", DefaultLogLevel, cfg.LogLevel)
	}
	if cfg.GinMode != DefaultGinMode {
		t.Fatalf("expected default gin mode %q, got %q", DefaultGinMode, cfg.GinMode)
	}
}

func TestAddrListensOnConfiguredPort(t *testing.T) {
	t.Setenv(EnvPort, "3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Addr() != ":3000" {
		t.Fatalf("expected listen address %q, got %q", ":3000", cfg.Addr())
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

func TestLoadUsesExplicitLogFormat(t *testing.T) {
	t.Setenv(EnvLogFormat, "TEXT-COLOR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LogFormat != LogFormatTextColor {
		t.Fatalf("expected log format %q, got %q", LogFormatTextColor, cfg.LogFormat)
	}
}

func TestLoadAcceptsTextLogFormat(t *testing.T) {
	t.Setenv(EnvLogFormat, LogFormatText)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LogFormat != LogFormatText {
		t.Fatalf("expected log format %q, got %q", LogFormatText, cfg.LogFormat)
	}
}

func TestLoadRejectsInvalidLogFormat(t *testing.T) {
	t.Setenv(EnvLogFormat, "xml")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid log format error")
	}
}

func TestLoadUsesExplicitLogLevel(t *testing.T) {
	t.Setenv(EnvLogLevel, "DEBUG")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LogLevel != LogLevelDebug {
		t.Fatalf("expected log level %q, got %q", LogLevelDebug, cfg.LogLevel)
	}
}

func TestLoadAcceptsDisabledLogLevel(t *testing.T) {
	t.Setenv(EnvLogLevel, LogLevelDisabled)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.LogLevel != LogLevelDisabled {
		t.Fatalf("expected log level %q, got %q", LogLevelDisabled, cfg.LogLevel)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv(EnvLogLevel, "verbose")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid log level error")
	}
}

func TestLoadUsesExplicitGinMode(t *testing.T) {
	t.Setenv(EnvGinMode, "RELEASE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.GinMode != GinModeRelease {
		t.Fatalf("expected gin mode %q, got %q", GinModeRelease, cfg.GinMode)
	}
}

func TestLoadRejectsInvalidGinMode(t *testing.T) {
	t.Setenv(EnvGinMode, "production")

	if _, err := Load(); err == nil {
		t.Fatal("expected invalid gin mode error")
	}
}
