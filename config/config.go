package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultHost               = "0.0.0.0"
	DefaultPort               = 8080
	DefaultMaxConcurrentGames = 100
	DefaultWordlistPath       = "words.txt"
	DefaultLogFormat          = LogFormatJSON

	EnvHost               = "NIP_HOST"
	EnvPort               = "NIP_PORT"
	EnvMaxConcurrentGames = "NIP_MAX_CONCURRENT_GAMES"
	EnvWordlistPath       = "NIP_WORDLIST_PATH"
	EnvAPIBaseURL         = "NIP_API_BASE_URL"
	EnvLogFormat          = "NIP_LOG_FORMAT"

	LogFormatJSON      = "json"
	LogFormatText      = "text"
	LogFormatTextColor = "text-color"
)

type Config struct {
	Host               string
	Port               int
	MaxConcurrentGames int
	WordlistPath       string
	APIBaseURL         string
	LogFormat          string
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
	apiBaseURL := getEnv(EnvAPIBaseURL, defaultAPIBaseURL(host, port))
	if err := validateURL(EnvAPIBaseURL, apiBaseURL); err != nil {
		return Config{}, err
	}
	logFormat, err := getLogFormatEnv()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Host:               host,
		Port:               port,
		MaxConcurrentGames: maxConcurrentGames,
		WordlistPath:       wordlistPath,
		APIBaseURL:         apiBaseURL,
		LogFormat:          logFormat,
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

func getLogFormatEnv() (string, error) {
	value := strings.ToLower(getEnv(EnvLogFormat, DefaultLogFormat))
	switch value {
	case LogFormatJSON, LogFormatText, LogFormatTextColor:
		return value, nil
	default:
		return "", fmt.Errorf("%s must be one of %s, %s, %s", EnvLogFormat, LogFormatJSON, LogFormatText, LogFormatTextColor)
	}
}

func defaultAPIBaseURL(host string, port int) string {
	apiHost := host
	if apiHost == "" || apiHost == "0.0.0.0" || apiHost == "::" {
		apiHost = "localhost"
	}

	return "http://" + net.JoinHostPort(apiHost, strconv.Itoa(port))
}

func validateURL(name, value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("%s must be a valid URL: %w", name, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must include scheme and host", name)
	}

	return nil
}
