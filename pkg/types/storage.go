package types

// StorageService defines the interface for storage services
type StorageService interface {
	Store(key string, value []byte) error
	Retrieve(key string) ([]byte, error)
	Delete(key string) error
}

// StorageConfig represents the configuration for a storage service
type StorageConfig struct {
	Type     string            `json:"type"`
	Host     string            `json:"host"`
	Port     int               `json:"port"`
	Database string            `json:"database"`
	Username string            `json:"username"`
	Password string            `json:"-"` // Password not serialized to JSON
	Options  map[string]string `json:"options"`
}

// NewStorageConfig creates a new storage configuration with default values
func NewStorageConfig(storageType string) *StorageConfig {
	config := &StorageConfig{
		Type:    storageType,
		Options: make(map[string]string),
	}

	switch storageType {
	case "mysql":
		config.Host = "localhost"
		config.Port = 3306
		config.Database = "cortex"
		config.Username = "cortex"
	case "redis":
		config.Host = "localhost"
		config.Port = 6379
		config.Database = "0"
	case "vector":
		config.Host = "localhost"
		config.Port = 6333
		config.Database = "cortex"
	}

	return config
}
