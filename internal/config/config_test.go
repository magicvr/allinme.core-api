package config_test

import (
	"path/filepath"
	"testing"

	"github.com/magicvr/allinme.core-api/internal/config"
)

func TestLoad(t *testing.T) {
	absoluteDataDir := filepath.Join(t.TempDir(), "data")
	tests := []struct {
		name      string
		env       map[string]string
		wantEnv   config.Environment
		wantPort  int
		wantError bool
	}{
		{name: "development defaults", env: map[string]string{}, wantEnv: config.Development, wantPort: 8080},
		{name: "development explicit", env: map[string]string{"APP_ENV": "development", "PORT": "9000", "DATA_DIR": "runtime"}, wantEnv: config.Development, wantPort: 9000},
		{name: "production", env: map[string]string{"APP_ENV": "production", "PORT": "443", "DATA_DIR": absoluteDataDir}, wantEnv: config.Production, wantPort: 443},
		{name: "unknown environment", env: map[string]string{"APP_ENV": "staging"}, wantError: true},
		{name: "production missing port", env: map[string]string{"APP_ENV": "production", "DATA_DIR": absoluteDataDir}, wantError: true},
		{name: "production missing data dir", env: map[string]string{"APP_ENV": "production", "PORT": "8080"}, wantError: true},
		{name: "production relative data dir", env: map[string]string{"APP_ENV": "production", "PORT": "8080", "DATA_DIR": "data"}, wantError: true},
		{name: "non decimal port", env: map[string]string{"PORT": "8O80"}, wantError: true},
		{name: "zero port", env: map[string]string{"PORT": "0"}, wantError: true},
		{name: "port too high", env: map[string]string{"PORT": "65536"}, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			loaded, err := config.Load(mapLookup(test.env))
			if test.wantError {
				if err == nil {
					t.Fatal("Load() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if loaded.Environment != test.wantEnv {
				t.Errorf("Environment = %q, want %q", loaded.Environment, test.wantEnv)
			}
			if loaded.Port != test.wantPort {
				t.Errorf("Port = %d, want %d", loaded.Port, test.wantPort)
			}
			if loaded.Address != ":"+stringPort(test.wantPort) {
				t.Errorf("Address = %q", loaded.Address)
			}
			if !filepath.IsAbs(loaded.DataDir) {
				t.Errorf("DataDir = %q, want absolute path", loaded.DataDir)
			}
			wantDatabase := filepath.Join(loaded.DataDir, "allinme.db")
			if loaded.DatabasePath != wantDatabase || loaded.WALPath != wantDatabase+"-wal" || loaded.SHMPath != wantDatabase+"-shm" {
				t.Errorf("database paths = %q, %q, %q", loaded.DatabasePath, loaded.WALPath, loaded.SHMPath)
			}
		})
	}
}

func TestLoadAPIRequiresSigningKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantError bool
	}{
		{name: "missing", wantError: true},
		{name: "31 bytes", key: "1234567890123456789012345678901", wantError: true},
		{name: "32 bytes", key: "12345678901234567890123456789012"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := map[string]string{}
			if test.key != "" {
				values["JWT_SIGNING_KEY"] = test.key
			}
			loaded, err := config.LoadAPI(mapLookup(values))
			if (err != nil) != test.wantError {
				t.Fatalf("LoadAPI() error = %v, wantError %v", err, test.wantError)
			}
			if err == nil && len(loaded.JWTSigningKey) != 32 {
				t.Errorf("JWTSigningKey length = %d", len(loaded.JWTSigningKey))
			}
		})
	}
}

func TestLoadDemoSeedPasswordByteBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantError bool
	}{
		{name: "11 bytes", password: "12345678901", wantError: true},
		{name: "12 bytes", password: "123456789012"},
		{name: "72 bytes", password: string(make([]byte, 72))},
		{name: "73 bytes", password: string(make([]byte, 73)), wantError: true},
		{name: "multibyte 12 bytes", password: "密码密码密码密码"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := config.LoadDemoSeed(mapLookup(map[string]string{"DEMO_ACCOUNT_PASSWORD": test.password}))
			if (err != nil) != test.wantError {
				t.Fatalf("LoadDemoSeed() error = %v, wantError %v", err, test.wantError)
			}
		})
	}
}

func TestLoadBootstrapAdmin(t *testing.T) {
	absoluteDataDir := filepath.Join(t.TempDir(), "data")
	base := map[string]string{
		"APP_ENV": "production", "PORT": "8080", "DATA_DIR": absoluteDataDir,
		"BOOTSTRAP_ADMIN_USERNAME": " Admin ", "BOOTSTRAP_ADMIN_PASSWORD": "123456789012",
	}
	loaded, err := config.LoadBootstrapAdmin(mapLookup(base))
	if err != nil {
		t.Fatalf("LoadBootstrapAdmin() error = %v", err)
	}
	if loaded.Username != "admin" {
		t.Errorf("Username = %q, want admin", loaded.Username)
	}

	development := map[string]string{"BOOTSTRAP_ADMIN_USERNAME": "admin", "BOOTSTRAP_ADMIN_PASSWORD": "123456789012"}
	if _, err := config.LoadBootstrapAdmin(mapLookup(development)); err == nil {
		t.Fatal("LoadBootstrapAdmin() error = nil in development")
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func stringPort(port int) string {
	if port == 443 {
		return "443"
	}
	if port == 9000 {
		return "9000"
	}
	return "8080"
}
