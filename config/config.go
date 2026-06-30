package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultPort               = 8080
	DefaultMaxConcurrentGames = 100
	DefaultWordlistPath       = "words.txt"
	DefaultLogFormat          = LogFormatJSON
	DefaultLogLevel           = LogLevelInfo

	EnvPort               = "NIP_PORT"
	EnvMaxConcurrentGames = "NIP_MAX_CONCURRENT_GAMES"
	EnvWordlistPath       = "NIP_WORDLIST_PATH"
	EnvLogFormat          = "NIP_LOG_FORMAT"
	EnvLogLevel           = "NIP_LOG_LEVEL"

	LogFormatJSON      = "json"
	LogFormatText      = "text"
	LogFormatTextColor = "text-color"

	LogLevelTrace    = "trace"
	LogLevelDebug    = "debug"
	LogLevelInfo     = "info"
	LogLevelWarn     = "warn"
	LogLevelError    = "error"
	LogLevelDisabled = "disabled"
)

type Config struct {
	Port               int
	MaxConcurrentGames int
	WordlistPath       string
	LogFormat          string
	LogLevel           string
}

func Load() (Config, error) {
	port, err := getPositiveIntEnv(EnvPort, DefaultPort)
	if err != nil {
		return Config{}, err
	}
	maxConcurrentGames, err := getPositiveIntEnv(EnvMaxConcurrentGames, DefaultMaxConcurrentGames)
	if err != nil {
		return Config{}, err
	}
	wordlistPath := getEnv(EnvWordlistPath, DefaultWordlistPath)
	if wordlistPath == "" {
		return Config{}, fmt.Errorf("%s must not be empty", EnvWordlistPath)
	}
	logFormat, err := getLogFormatEnv()
	if err != nil {
		return Config{}, err
	}
	logLevel, err := getLogLevelEnv()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:               port,
		MaxConcurrentGames: maxConcurrentGames,
		WordlistPath:       wordlistPath,
		LogFormat:          logFormat,
		LogLevel:           logLevel,
	}, nil
}

func (c Config) Addr() string {
	return net.JoinHostPort("", strconv.Itoa(c.Port))
}

func getEnv(name, fallback string) string {
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return fallback
	}

	return value
}

func getPositiveIntEnv(name string, fallback int) (int, error) {
	value, ok := os.LookupEnv(name)
	if !ok || value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	if parsed < 1 {
		return 0, fmt.Errorf("%s must be greater than 0", name)
	}

	return parsed, nil
}

func getLogFormatEnv() (string, error) {
	value := strings.ToLower(getEnv(EnvLogFormat, DefaultLogFormat))
	switch value {
	case LogFormatJSON, LogFormatText, LogFormatTextColor:
		return value, nil
	default:
		return "", fmt.Errorf("%s must be one of %s, %s, %s", EnvLogFormat, LogFormatJSON, LogFormatText, LogFormatTextColor)
	}
}

func getLogLevelEnv() (string, error) {
	value := strings.ToLower(getEnv(EnvLogLevel, DefaultLogLevel))
	switch value {
	case LogLevelTrace, LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelDisabled:
		return value, nil
	default:
		return "", fmt.Errorf("%s must be one of %s, %s, %s, %s, %s, %s", EnvLogLevel, LogLevelTrace, LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelDisabled)
	}
}
