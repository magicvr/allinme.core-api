package config

import (
	"fmt"
	"path/filepath"
	"strconv"
)

const databaseFilename = "allinme.db"

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

func Load(lookup func(string) (string, bool)) (Config, error) {
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

func (config Config) IsDevelopment() bool {
	return config.Environment == Development
}
