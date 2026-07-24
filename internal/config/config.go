package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime configuration for the API server.
// Values are loaded from environment variables with sensible defaults for local demo use.
type Config struct {
	App  AppConfig
	HTTP HTTPConfig
	Log  LogConfig
	DB   DBConfig
}

type AppConfig struct {
	Name    string
	Env     string
	Version string
}

type HTTPConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type LogConfig struct {
	Level string
}

// DBConfig holds persistence settings. MVP default is SQLite; drivers stay behind ports.
type DBConfig struct {
	// Driver is reserved for future multi-driver selection (sqlite | postgres | ...).
	Driver     string
	SQLitePath string
}

// Load reads configuration from the environment.
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Name:    getenv("APP_NAME", "allinme.core-api"),
			Env:     getenv("APP_ENV", "development"),
			Version: getenv("APP_VERSION", "0.1.0"),
		},
		HTTP: HTTPConfig{
			Addr:         getenv("HTTP_ADDR", ":8080"),
			ReadTimeout:  durationEnv("HTTP_READ_TIMEOUT", 5*time.Second),
			WriteTimeout: durationEnv("HTTP_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  durationEnv("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		Log: LogConfig{
			Level: getenv("LOG_LEVEL", "info"),
		},
		DB: DBConfig{
			Driver:     getenv("DB_DRIVER", "sqlite"),
			SQLitePath: getenv("SQLITE_PATH", "data/demo.db"),
		},
	}
	return cfg, nil
}

// LogLevel maps the configured string to slog.Level.
func (c *Config) LogLevel() slog.Level {
	switch strings.ToLower(c.Log.Level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	// Accept Go duration strings (e.g. "5s") or plain seconds as integers.
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if sec, err := strconv.Atoi(v); err == nil {
		return time.Duration(sec) * time.Second
	}
	return fallback
}
