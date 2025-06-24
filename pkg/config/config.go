package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// Config represents the overall application configuration
type Config struct {
	// Server configuration
	Server ServerConfig `json:"server" yaml:"server"`

	// P2P network configuration
	Network NetworkConfig `json:"network" yaml:"network"`

	// Storage configuration
	Storage StorageConfig `json:"storage" yaml:"storage"`

	// Economic engine configuration
	Economy EconomyConfig `json:"economy" yaml:"economy"`

	// AI model configuration
	AI AIConfig `json:"ai" yaml:"ai"`

	// RAG system configuration
	RAG RAGConfig `json:"rag" yaml:"rag"`

	// Logging configuration
	Logging LoggingConfig `json:"logging" yaml:"logging"`

	// Security configuration
	Security SecurityConfig `json:"security" yaml:"security"`
}

// ServerConfig holds API server configuration
type ServerConfig struct {
	Host         string        `json:"host" yaml:"host"`
	Port         int           `json:"port" yaml:"port"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	EnableCORS   bool          `json:"enable_cors" yaml:"enable_cors"`
	EnableHTTPS  bool          `json:"enable_https" yaml:"enable_https"`
	CertFile     string        `json:"cert_file" yaml:"cert_file"`
	KeyFile      string        `json:"key_file" yaml:"key_file"`
}

// NetworkConfig holds P2P network configuration
type NetworkConfig struct {
	ListenAddresses []string      `json:"listen_addresses" yaml:"listen_addresses"`
	BootstrapPeers  []string      `json:"bootstrap_peers" yaml:"bootstrap_peers"`
	EnableMDNS      bool          `json:"enable_mdns" yaml:"enable_mdns"`
	EnableDHT       bool          `json:"enable_dht" yaml:"enable_dht"`
	MaxPeers        int           `json:"max_peers" yaml:"max_peers"`
	ConnTimeout     time.Duration `json:"conn_timeout" yaml:"conn_timeout"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type     string            `json:"type" yaml:"type"` // "redis", "sql", "memory"
	Host     string            `json:"host" yaml:"host"`
	Port     int               `json:"port" yaml:"port"`
	Database string            `json:"database" yaml:"database"`
	Username string            `json:"username" yaml:"username"`
	Password string            `json:"password" yaml:"password"`
	Options  map[string]string `json:"options" yaml:"options"`

	// Connection pool settings
	MaxOpenConns    int           `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
}

// EconomyConfig holds economic engine configuration
type EconomyConfig struct {
	Enable           bool          `json:"enable" yaml:"enable"`
	NetworkTaxRate   float64       `json:"network_tax_rate" yaml:"network_tax_rate"`
	RewardMultiplier float64       `json:"reward_multiplier" yaml:"reward_multiplier"`
	SlashingRate     float64       `json:"slashing_rate" yaml:"slashing_rate"`
	MinStakeAmount   int64         `json:"min_stake_amount" yaml:"min_stake_amount"`
	TokenDecimals    int           `json:"token_decimals" yaml:"token_decimals"`
	BaseQueryCost    int64         `json:"base_query_cost" yaml:"base_query_cost"`
	TaskTimeout      time.Duration `json:"task_timeout" yaml:"task_timeout"`
}

// AIConfig holds AI model configuration
type AIConfig struct {
	DefaultModel   string          `json:"default_model" yaml:"default_model"`
	ModelProviders []ModelProvider `json:"model_providers" yaml:"model_providers"`
	RequestTimeout time.Duration   `json:"request_timeout" yaml:"request_timeout"`
	MaxRetries     int             `json:"max_retries" yaml:"max_retries"`
	RetryDelay     time.Duration   `json:"retry_delay" yaml:"retry_delay"`
	CacheEnabled   bool            `json:"cache_enabled" yaml:"cache_enabled"`
	CacheTTL       time.Duration   `json:"cache_ttl" yaml:"cache_ttl"`
}

// ModelProvider represents an AI model provider configuration
type ModelProvider struct {
	Name     string         `json:"name" yaml:"name"`
	Type     string         `json:"type" yaml:"type"` // "openai", "anthropic", "local", "mock"
	Endpoint string         `json:"endpoint" yaml:"endpoint"`
	APIKey   string         `json:"api_key" yaml:"api_key"`
	Models   []string       `json:"models" yaml:"models"`
	Config   map[string]any `json:"config" yaml:"config"`
}

// RAGConfig holds RAG system configuration
type RAGConfig struct {
	VectorDimensions    int     `json:"vector_dimensions" yaml:"vector_dimensions"`
	SimilarityThreshold float64 `json:"similarity_threshold" yaml:"similarity_threshold"`
	MaxResults          int     `json:"max_results" yaml:"max_results"`
	IndexType           string  `json:"index_type" yaml:"index_type"` // "memory", "faiss", "milvus"
	PersistPath         string  `json:"persist_path" yaml:"persist_path"`
	EmbeddingModel      string  `json:"embedding_model" yaml:"embedding_model"`
	ChunkSize           int     `json:"chunk_size" yaml:"chunk_size"`
	ChunkOverlap        int     `json:"chunk_overlap" yaml:"chunk_overlap"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `json:"level" yaml:"level"`   // "debug", "info", "warn", "error"
	Format     string `json:"format" yaml:"format"` // "json", "text"
	Output     string `json:"output" yaml:"output"` // "stdout", "file"
	File       string `json:"file" yaml:"file"`
	MaxSize    int    `json:"max_size" yaml:"max_size"` // MB
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`
	MaxAge     int    `json:"max_age" yaml:"max_age"` // days
	Compress   bool   `json:"compress" yaml:"compress"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	EnableAuth     bool          `json:"enable_auth" yaml:"enable_auth"`
	JWTSecret      string        `json:"jwt_secret" yaml:"jwt_secret"`
	JWTExpiry      time.Duration `json:"jwt_expiry" yaml:"jwt_expiry"`
	RateLimit      RateLimit     `json:"rate_limit" yaml:"rate_limit"`
	TrustedProxies []string      `json:"trusted_proxies" yaml:"trusted_proxies"`
	EnableTLS      bool          `json:"enable_tls" yaml:"enable_tls"`
	MinTLSVersion  string        `json:"min_tls_version" yaml:"min_tls_version"`
}

// RateLimit holds rate limiting configuration
type RateLimit struct {
	Enable    bool          `json:"enable" yaml:"enable"`
	Requests  int           `json:"requests" yaml:"requests"`
	Window    time.Duration `json:"window" yaml:"window"`
	BurstSize int           `json:"burst_size" yaml:"burst_size"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			EnableCORS:   true,
			EnableHTTPS:  false,
		},
		Network: NetworkConfig{
			ListenAddresses: []string{
				"/ip4/0.0.0.0/tcp/4001",
				"/ip4/0.0.0.0/udp/4001/quic-v1",
			},
			BootstrapPeers: []string{},
			EnableMDNS:     true,
			EnableDHT:      true,
			MaxPeers:       50,
			ConnTimeout:    30 * time.Second,
		},
		Storage: StorageConfig{
			Type:            "memory",
			Host:            "localhost",
			Port:            6379,
			Database:        "0",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		Economy: EconomyConfig{
			Enable:           true,
			NetworkTaxRate:   0.01,
			RewardMultiplier: 1.0,
			SlashingRate:     0.1,
			MinStakeAmount:   1000,
			TokenDecimals:    18,
			BaseQueryCost:    10,
			TaskTimeout:      30 * time.Second,
		},
		AI: AIConfig{
			DefaultModel:   "mock",
			RequestTimeout: 30 * time.Second,
			MaxRetries:     3,
			RetryDelay:     1 * time.Second,
			CacheEnabled:   true,
			CacheTTL:       1 * time.Hour,
			ModelProviders: []ModelProvider{
				{
					Name:   "mock",
					Type:   "mock",
					Models: []string{"mock-model"},
				},
			},
		},
		RAG: RAGConfig{
			VectorDimensions:    384,
			SimilarityThreshold: 0.7,
			MaxResults:          10,
			IndexType:           "memory",
			EmbeddingModel:      "default",
			ChunkSize:           512,
			ChunkOverlap:        50,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
		Security: SecurityConfig{
			EnableAuth:     false,
			JWTExpiry:      24 * time.Hour,
			TrustedProxies: []string{},
			EnableTLS:      false,
			MinTLSVersion:  "1.2",
			RateLimit: RateLimit{
				Enable:    false,
				Requests:  100,
				Window:    1 * time.Minute,
				BurstSize: 10,
			},
		},
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	if configPath == "" {
		return config, nil
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse based on file extension
	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".json":
		err = json.Unmarshal(data, config)
	case ".yaml", ".yml":
		// Note: In a real implementation, you'd use a YAML library like gopkg.in/yaml.v3
		return nil, fmt.Errorf("YAML support not implemented")
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(config)

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, configPath string) error {
	ext := strings.ToLower(filepath.Ext(configPath))

	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
	case ".yaml", ".yml":
		return fmt.Errorf("YAML support not implemented")
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(config *Config) {
	// Server overrides
	if host := os.Getenv("CORTEX_SERVER_HOST"); host != "" {
		config.Server.Host = host
	}
	if port := os.Getenv("CORTEX_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}

	// Network overrides
	if peers := os.Getenv("CORTEX_BOOTSTRAP_PEERS"); peers != "" {
		config.Network.BootstrapPeers = strings.Split(peers, ",")
	}

	// Storage overrides
	if storageType := os.Getenv("CORTEX_STORAGE_TYPE"); storageType != "" {
		config.Storage.Type = storageType
	}
	if host := os.Getenv("CORTEX_STORAGE_HOST"); host != "" {
		config.Storage.Host = host
	}
	if port := os.Getenv("CORTEX_STORAGE_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Storage.Port = p
		}
	}
	if db := os.Getenv("CORTEX_STORAGE_DATABASE"); db != "" {
		config.Storage.Database = db
	}
	if username := os.Getenv("CORTEX_STORAGE_USERNAME"); username != "" {
		config.Storage.Username = username
	}
	if password := os.Getenv("CORTEX_STORAGE_PASSWORD"); password != "" {
		config.Storage.Password = password
	}

	// Economy overrides
	if enable := os.Getenv("CORTEX_ECONOMY_ENABLE"); enable != "" {
		config.Economy.Enable = strings.ToLower(enable) == "true"
	}

	// AI overrides
	if model := os.Getenv("CORTEX_DEFAULT_MODEL"); model != "" {
		config.AI.DefaultModel = model
	}

	// Security overrides
	if secret := os.Getenv("CORTEX_JWT_SECRET"); secret != "" {
		config.Security.JWTSecret = secret
	}

	// Logging overrides
	if level := os.Getenv("CORTEX_LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
}

// validateConfig validates the configuration for common errors
func validateConfig(config *Config) error {
	// Validate server configuration
	if config.Server.Port < 1 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	// Validate storage configuration
	if config.Storage.Type == "" {
		return fmt.Errorf("storage type cannot be empty")
	}

	validStorageTypes := map[string]bool{
		"memory": true,
		"redis":  true,
		"sql":    true,
	}
	if !validStorageTypes[config.Storage.Type] {
		return fmt.Errorf("invalid storage type: %s", config.Storage.Type)
	}

	// Validate RAG configuration
	if config.RAG.VectorDimensions <= 0 {
		return fmt.Errorf("vector dimensions must be positive: %d", config.RAG.VectorDimensions)
	}

	if config.RAG.SimilarityThreshold < 0 || config.RAG.SimilarityThreshold > 1 {
		return fmt.Errorf("similarity threshold must be between 0 and 1: %f", config.RAG.SimilarityThreshold)
	}

	// Validate logging configuration
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	return nil
}

// ToStorageConfig converts the storage config to the types package format
func (sc *StorageConfig) ToStorageConfig() *types.StorageConfig {
	return &types.StorageConfig{
		Host:     sc.Host,
		Port:     sc.Port,
		Database: sc.Database,
		Username: sc.Username,
		Password: sc.Password,
		Options:  sc.Options,
	}
}

// ToNetworkConfig converts the network config to the types package format
func (nc *NetworkConfig) ToNetworkConfig() *types.NetworkConfig {
	return &types.NetworkConfig{
		ListenAddresses: nc.ListenAddresses,
		BootstrapPeers:  nc.BootstrapPeers,
	}
}
