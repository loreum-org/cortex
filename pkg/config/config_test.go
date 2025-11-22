package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Errorf("DefaultConfig() returned nil")
		return
	}

	// Test server defaults
	if config.Server.Host != "0.0.0.0" {
		t.Errorf("Default server host = %s, want 0.0.0.0", config.Server.Host)
	}

	if config.Server.Port != 8080 {
		t.Errorf("Default server port = %d, want 8080", config.Server.Port)
	}

	// Test network defaults
	if len(config.Network.ListenAddresses) == 0 {
		t.Errorf("Default network listen addresses should not be empty")
	}

	// Test storage defaults
	if config.Storage.Type != "memory" {
		t.Errorf("Default storage type = %s, want memory", config.Storage.Type)
	}

	// Test economy defaults
	if !config.Economy.Enable {
		t.Errorf("Default economy enable = %t, want true", config.Economy.Enable)
	}

	// Test AI defaults
	if config.AI.DefaultModel != "mock" {
		t.Errorf("Default AI model = %s, want mock", config.AI.DefaultModel)
	}

	// Test RAG defaults
	if config.RAG.VectorDimensions != 384 {
		t.Errorf("Default RAG vector dimensions = %d, want 384", config.RAG.VectorDimensions)
	}

	// Test logging defaults
	if config.Logging.Level != "info" {
		t.Errorf("Default logging level = %s, want info", config.Logging.Level)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent-file.json")
	if err == nil {
		t.Errorf("LoadConfig() expected error for nonexistent file")
	}
}

func TestLoadConfig_EmptyPath(t *testing.T) {
	config, err := LoadConfig("")
	if err != nil {
		t.Errorf("LoadConfig() unexpected error for empty path: %v", err)
	}

	if config == nil {
		t.Errorf("LoadConfig() returned nil config for empty path")
	}

	// Should return default config
	defaultConfig := DefaultConfig()
	if !reflect.DeepEqual(config, defaultConfig) {
		t.Errorf("LoadConfig() with empty path should return default config")
	}
}

func TestLoadConfig_JSONFile(t *testing.T) {
	// Create a temporary JSON config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.json")

	testConfig := DefaultConfig()
	testConfig.Server.Host = "localhost"
	testConfig.Server.Port = 9090
	testConfig.Storage.Type = "redis"
	testConfig.Storage.Host = "redis-host"
	testConfig.Storage.Port = 6380
	testConfig.Logging.Level = "debug"

	// Write test config to file
	data, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Load the config
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Errorf("LoadConfig() unexpected error: %v", err)
		return
	}

	// Verify loaded values (note: LoadConfig merges with defaults)
	if loadedConfig.Server.Host != "localhost" {
		t.Errorf("Loaded server host = %s, want localhost", loadedConfig.Server.Host)
	}

	if loadedConfig.Server.Port != 9090 {
		t.Errorf("Loaded server port = %d, want 9090", loadedConfig.Server.Port)
	}

	if loadedConfig.Storage.Type != "redis" {
		t.Errorf("Loaded storage type = %s, want redis", loadedConfig.Storage.Type)
	}

	if loadedConfig.Logging.Level != "debug" {
		t.Errorf("Loaded logging level = %s, want debug", loadedConfig.Logging.Level)
	}
}

func TestLoadConfig_UnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.xml")

	err := os.WriteFile(configPath, []byte("<config></config>"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Errorf("LoadConfig() expected error for unsupported format")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.json")

	err := os.WriteFile(configPath, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Errorf("LoadConfig() expected error for invalid JSON")
	}
}

func TestSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "saved-config.json")

	config := DefaultConfig()
	config.Server.Host = "test-host"
	config.Server.Port = 9999

	err := SaveConfig(config, configPath)
	if err != nil {
		t.Errorf("SaveConfig() unexpected error: %v", err)
		return
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("SaveConfig() did not create config file")
		return
	}

	// Load and verify content
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Errorf("Failed to load saved config: %v", err)
		return
	}

	if loadedConfig.Server.Host != "test-host" {
		t.Errorf("Saved/loaded server host = %s, want test-host", loadedConfig.Server.Host)
	}

	if loadedConfig.Server.Port != 9999 {
		t.Errorf("Saved/loaded server port = %d, want 9999", loadedConfig.Server.Port)
	}
}

func TestSaveConfig_UnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.xml")

	config := DefaultConfig()
	err := SaveConfig(config, configPath)
	if err == nil {
		t.Errorf("SaveConfig() expected error for unsupported format")
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"CORTEX_SERVER_HOST":  os.Getenv("CORTEX_SERVER_HOST"),
		"CORTEX_SERVER_PORT":  os.Getenv("CORTEX_SERVER_PORT"),
		"CORTEX_STORAGE_TYPE": os.Getenv("CORTEX_STORAGE_TYPE"),
		"CORTEX_LOG_LEVEL":    os.Getenv("CORTEX_LOG_LEVEL"),
	}

	// Set test environment variables
	os.Setenv("CORTEX_SERVER_HOST", "env-host")
	os.Setenv("CORTEX_SERVER_PORT", "8888")
	os.Setenv("CORTEX_STORAGE_TYPE", "sql")
	os.Setenv("CORTEX_LOG_LEVEL", "error")

	defer func() {
		// Restore original environment
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	config := DefaultConfig()
	applyEnvOverrides(config)

	if config.Server.Host != "env-host" {
		t.Errorf("Env override server host = %s, want env-host", config.Server.Host)
	}

	if config.Server.Port != 8888 {
		t.Errorf("Env override server port = %d, want 8888", config.Server.Port)
	}

	if config.Storage.Type != "sql" {
		t.Errorf("Env override storage type = %s, want sql", config.Storage.Type)
	}

	if config.Logging.Level != "error" {
		t.Errorf("Env override log level = %s, want error", config.Logging.Level)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*Config)
		wantErr   bool
	}{
		{
			name:      "valid default config",
			setupFunc: func(c *Config) {}, // no changes
			wantErr:   false,
		},
		{
			name: "invalid server port - too low",
			setupFunc: func(c *Config) {
				c.Server.Port = 0
			},
			wantErr: true,
		},
		{
			name: "invalid server port - too high",
			setupFunc: func(c *Config) {
				c.Server.Port = 70000
			},
			wantErr: true,
		},
		{
			name: "empty storage type",
			setupFunc: func(c *Config) {
				c.Storage.Type = ""
			},
			wantErr: true,
		},
		{
			name: "invalid storage type",
			setupFunc: func(c *Config) {
				c.Storage.Type = "invalid"
			},
			wantErr: true,
		},
		{
			name: "invalid vector dimensions",
			setupFunc: func(c *Config) {
				c.RAG.VectorDimensions = -1
			},
			wantErr: true,
		},
		{
			name: "invalid similarity threshold - too low",
			setupFunc: func(c *Config) {
				c.RAG.SimilarityThreshold = -0.1
			},
			wantErr: true,
		},
		{
			name: "invalid similarity threshold - too high",
			setupFunc: func(c *Config) {
				c.RAG.SimilarityThreshold = 1.1
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			setupFunc: func(c *Config) {
				c.Logging.Level = "invalid"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			tt.setupFunc(config)

			err := validateConfig(config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateConfig() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("validateConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestToStorageConfig(t *testing.T) {
	config := &StorageConfig{
		Host:     "test-host",
		Port:     1234,
		Database: "test-db",
		Username: "test-user",
		Password: "test-pass",
		Options:  map[string]string{"option1": "value1"},
	}

	typesConfig := config.ToStorageConfig()

	if typesConfig.Host != config.Host {
		t.Errorf("ToStorageConfig() host = %s, want %s", typesConfig.Host, config.Host)
	}

	if typesConfig.Port != config.Port {
		t.Errorf("ToStorageConfig() port = %d, want %d", typesConfig.Port, config.Port)
	}

	if typesConfig.Database != config.Database {
		t.Errorf("ToStorageConfig() database = %s, want %s", typesConfig.Database, config.Database)
	}
}

func TestToNetworkConfig(t *testing.T) {
	config := &NetworkConfig{
		ListenAddresses: []string{"/ip4/0.0.0.0/tcp/4001"},
		BootstrapPeers:  []string{"peer1", "peer2"},
	}

	typesConfig := config.ToNetworkConfig()

	if !reflect.DeepEqual(typesConfig.ListenAddresses, config.ListenAddresses) {
		t.Errorf("ToNetworkConfig() listen addresses mismatch")
	}

	if !reflect.DeepEqual(typesConfig.BootstrapPeers, config.BootstrapPeers) {
		t.Errorf("ToNetworkConfig() bootstrap peers mismatch")
	}
}
