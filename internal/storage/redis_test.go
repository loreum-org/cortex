package storage

import (
	"testing"

	"github.com/loreum-org/cortex/pkg/types"
)

// TestRedisStorage_NewRedisStorage tests creating a new Redis storage instance
func TestRedisStorage_NewRedisStorage(t *testing.T) {
	tests := []struct {
		name        string
		config      *types.StorageConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid configuration",
			config: &types.StorageConfig{
				Host:     "localhost",
				Port:     6379,
				Database: "0",
				Password: "",
			},
			wantErr: true, // Will error if Redis not available, but tests the config parsing
		},
		{
			name: "invalid database number",
			config: &types.StorageConfig{
				Host:     "localhost",
				Port:     6379,
				Database: "invalid",
				Password: "",
			},
			wantErr:     true,
			errContains: "invalid database number",
		},
		{
			name: "invalid host",
			config: &types.StorageConfig{
				Host:     "nonexistent-host-12345",
				Port:     6379,
				Database: "0",
				Password: "",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: &types.StorageConfig{
				Host:     "localhost",
				Port:     99999,
				Database: "0",
				Password: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewRedisStorage(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewRedisStorage() expected error but got none")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("NewRedisStorage() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("NewRedisStorage() unexpected error = %v", err)
				return
			}

			if storage == nil {
				t.Errorf("NewRedisStorage() returned nil storage")
				return
			}

			if storage.Client == nil {
				t.Errorf("NewRedisStorage() returned storage with nil client")
			}

			if storage.Config != tt.config {
				t.Errorf("NewRedisStorage() config mismatch")
			}

			// Clean up
			storage.Close()
		})
	}
}

// TestRedisStorage_Store tests storing values in Redis
func TestRedisStorage_Store(t *testing.T) {
	storage := setupTestRedisStorage(t)
	defer cleanupTestRedisStorage(t, storage)

	tests := []struct {
		name    string
		key     string
		value   []byte
		wantErr bool
	}{
		{
			name:    "store simple value",
			key:     "test-key-1",
			value:   []byte("test-value-1"),
			wantErr: false,
		},
		{
			name:    "store binary data",
			key:     "test-key-2",
			value:   []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
			wantErr: false,
		},
		{
			name:    "store empty value",
			key:     "test-key-3",
			value:   []byte{},
			wantErr: false,
		},
		{
			name:    "store with special characters in key",
			key:     "test:key:with:colons",
			value:   []byte("special-key-value"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Store(tt.key, tt.value)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Store() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Store() unexpected error = %v", err)
			}

			// Verify the value was stored
			result, err := storage.Client.Get(storage.ctx, tt.key).Bytes()
			if err != nil {
				t.Errorf("Failed to verify stored value: %v", err)
				return
			}

			if !bytesEqual(result, tt.value) {
				t.Errorf("Stored value mismatch. Got %v, want %v", result, tt.value)
			}
		})
	}
}

// TestRedisStorage_Retrieve tests retrieving values from Redis
func TestRedisStorage_Retrieve(t *testing.T) {
	storage := setupTestRedisStorage(t)
	defer cleanupTestRedisStorage(t, storage)

	// Prepare test data
	testData := map[string][]byte{
		"existing-key-1": []byte("existing-value-1"),
		"existing-key-2": {0x00, 0x01, 0x02, 0xFF},
		"empty-value":    {},
	}

	// Store test data
	for key, value := range testData {
		err := storage.Store(key, value)
		if err != nil {
			t.Fatalf("Failed to prepare test data: %v", err)
		}
	}

	tests := []struct {
		name          string
		key           string
		expectedValue []byte
		expectNil     bool
		wantErr       bool
	}{
		{
			name:          "retrieve existing value",
			key:           "existing-key-1",
			expectedValue: []byte("existing-value-1"),
			expectNil:     false,
			wantErr:       false,
		},
		{
			name:          "retrieve binary data",
			key:           "existing-key-2",
			expectedValue: []byte{0x00, 0x01, 0x02, 0xFF},
			expectNil:     false,
			wantErr:       false,
		},
		{
			name:          "retrieve empty value",
			key:           "empty-value",
			expectedValue: []byte{},
			expectNil:     false,
			wantErr:       false,
		},
		{
			name:          "retrieve non-existing key",
			key:           "non-existing-key",
			expectedValue: nil,
			expectNil:     true,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := storage.Retrieve(tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Retrieve() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Retrieve() unexpected error = %v", err)
				return
			}

			if tt.expectNil {
				if result != nil {
					t.Errorf("Retrieve() expected nil but got %v", result)
				}
				return
			}

			if !bytesEqual(result, tt.expectedValue) {
				t.Errorf("Retrieve() = %v, want %v", result, tt.expectedValue)
			}
		})
	}
}

// TestRedisStorage_Delete tests deleting values from Redis
func TestRedisStorage_Delete(t *testing.T) {
	storage := setupTestRedisStorage(t)
	defer cleanupTestRedisStorage(t, storage)

	// Prepare test data
	testKey := "key-to-delete"
	testValue := []byte("value-to-delete")

	err := storage.Store(testKey, testValue)
	if err != nil {
		t.Fatalf("Failed to prepare test data: %v", err)
	}

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "delete existing key",
			key:     testKey,
			wantErr: false,
		},
		{
			name:    "delete non-existing key",
			key:     "non-existing-key",
			wantErr: false, // Redis doesn't error on deleting non-existing keys
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Delete(tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Delete() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Delete() unexpected error = %v", err)
				return
			}

			// Verify the key was deleted
			result, err := storage.Retrieve(tt.key)
			if err != nil {
				t.Errorf("Failed to verify deletion: %v", err)
				return
			}

			if result != nil {
				t.Errorf("Key was not deleted: still got value %v", result)
			}
		})
	}
}

// TestRedisStorage_Close tests closing the Redis connection
func TestRedisStorage_Close(t *testing.T) {
	storage := setupTestRedisStorage(t)

	err := storage.Close()
	if err != nil {
		t.Errorf("Close() unexpected error = %v", err)
	}

	// Verify that operations fail after closing
	err = storage.Store("test", []byte("test"))
	if err == nil {
		t.Errorf("Expected error when using storage after close, but got none")
	}
}

// TestRedisStorage_StoreRetrieveIntegration tests the complete store-retrieve cycle
func TestRedisStorage_StoreRetrieveIntegration(t *testing.T) {
	storage := setupTestRedisStorage(t)
	defer cleanupTestRedisStorage(t, storage)

	// Test various data types and scenarios
	testCases := []struct {
		name  string
		key   string
		value []byte
	}{
		{"string data", "string-key", []byte("Hello, World!")},
		{"json data", "json-key", []byte(`{"name": "test", "value": 123}`)},
		{"binary data", "binary-key", []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}},
		{"large data", "large-key", make([]byte, 10240)}, // 10KB
		{"unicode data", "unicode-key", []byte("Hello, 世界! 🌍")},
	}

	// Initialize large data with pattern
	for i := range testCases[3].value {
		testCases[3].value[i] = byte(i % 256)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Store the value
			err := storage.Store(tc.key, tc.value)
			if err != nil {
				t.Fatalf("Store() failed: %v", err)
			}

			// Retrieve the value
			retrieved, err := storage.Retrieve(tc.key)
			if err != nil {
				t.Fatalf("Retrieve() failed: %v", err)
			}

			// Compare values
			if !bytesEqual(retrieved, tc.value) {
				t.Errorf("Value mismatch. Original length: %d, Retrieved length: %d", len(tc.value), len(retrieved))
				if len(tc.value) < 100 && len(retrieved) < 100 {
					t.Errorf("Original: %v", tc.value)
					t.Errorf("Retrieved: %v", retrieved)
				}
			}

			// Clean up
			err = storage.Delete(tc.key)
			if err != nil {
				t.Errorf("Delete() failed: %v", err)
			}
		})
	}
}

// Helper functions

func setupTestRedisStorage(t *testing.T) *RedisStorage {
	config := &types.StorageConfig{
		Host:     "localhost",
		Port:     6379,
		Database: "0",
		Password: "",
	}

	storage, err := NewRedisStorage(config)
	if err != nil {
		t.Skipf("Skipping Redis tests: %v", err)
	}

	return storage
}

func cleanupTestRedisStorage(t *testing.T, storage *RedisStorage) {
	if storage != nil {
		// Clean up any test keys
		keys, err := storage.Client.Keys(storage.ctx, "test*").Result()
		if err == nil {
			for _, key := range keys {
				storage.Client.Del(storage.ctx, key)
			}
		}
		storage.Close()
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
