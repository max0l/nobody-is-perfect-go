package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

const (
	DefaultHost               = "0.0.0.0"
	DefaultPort               = 8080
	DefaultMaxConcurrentGames = 100
	DefaultWordlistPath       = "words.txt"

	EnvHost               = "NIP_HOST"
	EnvPort               = "NIP_PORT"
	EnvMaxConcurrentGames = "NIP_MAX_CONCURRENT_GAMES"
	EnvWordlistPath       = "NIP_WORDLIST_PATH"
)

type Config struct {
	Host               string
	Port               int
	MaxConcurrentGames int
	WordlistPath       string
}

func Load() (Config, error) {
	host := getEnv(EnvHost, DefaultHost)
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

	return Config{
		Host:               host,
		Port:               port,
		MaxConcurrentGames: maxConcurrentGames,
		WordlistPath:       wordlistPath,
	}, nil
}

func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
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
