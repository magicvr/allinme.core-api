package config

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

const databaseFilename = "allinme.db"

const (
	MinimumSigningKeyBytes = 32
	MinimumPasswordBytes   = 12
	MaximumPasswordBytes   = 72
)

type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

type Config struct {
	Environment  Environment
	Port         int
	Address      string
	DataDir      string
	DatabasePath string
	WALPath      string
	SHMPath      string
}

type APIConfig struct {
	Config
	JWTSigningKey []byte
}

type DemoSeedConfig struct {
	Config
	DemoAccountPassword string
}

type BootstrapAdminConfig struct {
	Config
	Username string
	Password string
}

func LoadBase(lookup func(string) (string, bool)) (Config, error) {
	environmentValue, environmentSet := lookup("APP_ENV")
	if !environmentSet || environmentValue == "" {
		environmentValue = string(Development)
	}

	environment := Environment(environmentValue)
	if environment != Development && environment != Production {
		return Config{}, fmt.Errorf("APP_ENV must be development or production")
	}
	if environment == Production && !environmentSet {
		return Config{}, fmt.Errorf("APP_ENV must be explicitly configured in production")
	}

	portValue, portSet := lookup("PORT")
	if !portSet || portValue == "" {
		if environment == Production {
			return Config{}, fmt.Errorf("PORT must be explicitly configured in production")
		}
		portValue = "8080"
	}
	port, err := strconv.Atoi(portValue)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be a decimal integer from 1 to 65535")
	}

	dataDir, dataDirSet := lookup("DATA_DIR")
	if !dataDirSet || dataDir == "" {
		if environment == Production {
			return Config{}, fmt.Errorf("DATA_DIR must be explicitly configured in production")
		}
		dataDir = "./data"
	}
	if environment == Production && !filepath.IsAbs(dataDir) {
		return Config{}, fmt.Errorf("DATA_DIR must be absolute in production")
	}

	dataDir, err = filepath.Abs(dataDir)
	if err != nil {
		return Config{}, fmt.Errorf("resolve DATA_DIR: %w", err)
	}
	dataDir = filepath.Clean(dataDir)
	databasePath := filepath.Join(dataDir, databaseFilename)

	return Config{
		Environment:  environment,
		Port:         port,
		Address:      ":" + strconv.Itoa(port),
		DataDir:      dataDir,
		DatabasePath: databasePath,
		WALPath:      databasePath + "-wal",
		SHMPath:      databasePath + "-shm",
	}, nil
}

func Load(lookup func(string) (string, bool)) (Config, error) {
	return LoadBase(lookup)
}

func LoadAPI(lookup func(string) (string, bool)) (APIConfig, error) {
	base, err := LoadBase(lookup)
	if err != nil {
		return APIConfig{}, err
	}
	key, ok := lookup("JWT_SIGNING_KEY")
	if !ok || len([]byte(key)) < MinimumSigningKeyBytes {
		return APIConfig{}, fmt.Errorf("JWT_SIGNING_KEY must be explicitly configured with at least %d bytes", MinimumSigningKeyBytes)
	}
	return APIConfig{Config: base, JWTSigningKey: []byte(key)}, nil
}

func LoadDemoSeed(lookup func(string) (string, bool)) (DemoSeedConfig, error) {
	base, err := LoadBase(lookup)
	if err != nil {
		return DemoSeedConfig{}, err
	}
	if !base.IsDevelopment() {
		return DemoSeedConfig{}, fmt.Errorf("demo seed is only available in development")
	}
	password, ok := lookup("DEMO_ACCOUNT_PASSWORD")
	if !ok {
		return DemoSeedConfig{}, fmt.Errorf("DEMO_ACCOUNT_PASSWORD must be explicitly configured")
	}
	if err := validatePassword("DEMO_ACCOUNT_PASSWORD", password); err != nil {
		return DemoSeedConfig{}, err
	}
	return DemoSeedConfig{Config: base, DemoAccountPassword: password}, nil
}

func LoadBootstrapAdmin(lookup func(string) (string, bool)) (BootstrapAdminConfig, error) {
	base, err := LoadBase(lookup)
	if err != nil {
		return BootstrapAdminConfig{}, err
	}
	if base.Environment != Production {
		return BootstrapAdminConfig{}, fmt.Errorf("bootstrap-admin is only available in production")
	}
	username, ok := lookup("BOOTSTRAP_ADMIN_USERNAME")
	username = strings.ToLower(strings.TrimSpace(username))
	if !ok || username == "" {
		return BootstrapAdminConfig{}, fmt.Errorf("BOOTSTRAP_ADMIN_USERNAME must be explicitly configured and non-empty")
	}
	password, ok := lookup("BOOTSTRAP_ADMIN_PASSWORD")
	if !ok {
		return BootstrapAdminConfig{}, fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD must be explicitly configured")
	}
	if err := validatePassword("BOOTSTRAP_ADMIN_PASSWORD", password); err != nil {
		return BootstrapAdminConfig{}, err
	}
	return BootstrapAdminConfig{Config: base, Username: username, Password: password}, nil
}

func validatePassword(name, password string) error {
	length := len([]byte(password))
	if length < MinimumPasswordBytes || length > MaximumPasswordBytes {
		return fmt.Errorf("%s must be %d to %d bytes", name, MinimumPasswordBytes, MaximumPasswordBytes)
	}
	return nil
}

func (config Config) IsDevelopment() bool {
	return config.Environment == Development
}
