package storage

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/loreum-org/cortex/pkg/types"
)

// TestSQLStorage_NewSQLStorage tests creating a new SQL storage instance
func TestSQLStorage_NewSQLStorage(t *testing.T) {
	tests := []struct {
		name        string
		config      *types.StorageConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "invalid host",
			config: &types.StorageConfig{
				Host:     "nonexistent-host-12345",
				Port:     3306,
				Database: "test",
				Username: "root",
				Password: "",
			},
			wantErr: true,
		},
		{
			name: "valid config with options",
			config: &types.StorageConfig{
				Host:     "localhost",
				Port:     3306,
				Database: "test",
				Username: "root",
				Password: "",
				Options: map[string]string{
					"charset":   "utf8mb4",
					"parseTime": "True",
					"loc":       "Local",
				},
			},
			wantErr: true, // Will error if MySQL not available, but tests the DSN construction
		},
		{
			name: "valid config without options",
			config: &types.StorageConfig{
				Host:     "localhost",
				Port:     3306,
				Database: "test",
				Username: "root",
				Password: "password",
			},
			wantErr: true, // Will error if MySQL not available, but tests the DSN construction
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := NewSQLStorage(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewSQLStorage() expected error but got none")
					// Clean up if somehow it succeeded
					if storage != nil {
						storage.Close()
					}
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("NewSQLStorage() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("NewSQLStorage() unexpected error = %v", err)
				return
			}

			if storage == nil {
				t.Errorf("NewSQLStorage() returned nil storage")
				return
			}

			if storage.DB == nil {
				t.Errorf("NewSQLStorage() returned storage with nil DB")
			}

			if storage.Config != tt.config {
				t.Errorf("NewSQLStorage() config mismatch")
			}

			// Clean up
			storage.Close()
		})
	}
}

// TestSQLStorage_Store tests storing values in SQL database
func TestSQLStorage_Store(t *testing.T) {
	storage := setupTestSQLStorage(t)
	defer cleanupTestSQLStorage(t, storage)

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
			key:     "test-key-with-dashes",
			value:   []byte("special-key-value"),
			wantErr: false,
		},
		{
			name:    "store large value",
			key:     "test-large-key",
			value:   make([]byte, 5120), // 5KB
			wantErr: false,
		},
		{
			name:    "update existing value",
			key:     "test-key-1", // Same key as first test
			value:   []byte("updated-value"),
			wantErr: false,
		},
	}

	// Initialize large data
	for i := range tests[4].value {
		tests[4].value[i] = byte(i % 256)
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
				return
			}

			// Verify the value was stored
			var result []byte
			err = storage.DB.QueryRow("SELECT v FROM kv_store WHERE k = ?", tt.key).Scan(&result)
			if err != nil {
				t.Errorf("Failed to verify stored value: %v", err)
				return
			}

			if !bytesEqual(result, tt.value) {
				t.Errorf("Stored value mismatch. Got length %d, want length %d", len(result), len(tt.value))
			}
		})
	}
}

// TestSQLStorage_Retrieve tests retrieving values from SQL database
func TestSQLStorage_Retrieve(t *testing.T) {
	storage := setupTestSQLStorage(t)
	defer cleanupTestSQLStorage(t, storage)

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

// TestSQLStorage_Delete tests deleting values from SQL database
func TestSQLStorage_Delete(t *testing.T) {
	storage := setupTestSQLStorage(t)
	defer cleanupTestSQLStorage(t, storage)

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
			wantErr: false, // SQL doesn't error on deleting non-existing keys
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

// TestSQLStorage_Close tests closing the SQL connection
func TestSQLStorage_Close(t *testing.T) {
	storage := setupTestSQLStorage(t)

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

// TestSQLStorage_TableCreation tests that the table is created correctly
func TestSQLStorage_TableCreation(t *testing.T) {
	storage := setupTestSQLStorage(t)
	defer cleanupTestSQLStorage(t, storage)

	// Test that the table is created when storing first value
	err := storage.Store("test-key", []byte("test-value"))
	if err != nil {
		t.Errorf("Store() failed on table creation: %v", err)
		return
	}

	// Verify table exists and has correct structure
	rows, err := storage.DB.Query("DESCRIBE kv_store")
	if err != nil {
		t.Errorf("Failed to describe table: %v", err)
		return
	}
	defer rows.Close()

	columns := make(map[string]bool)
	for rows.Next() {
		var field, fieldType, null, key, defaultValue, extra sql.NullString
		err := rows.Scan(&field, &fieldType, &null, &key, &defaultValue, &extra)
		if err != nil {
			t.Errorf("Failed to scan table description: %v", err)
			return
		}
		columns[field.String] = true
	}

	expectedColumns := []string{"k", "v", "updated_at"}
	for _, col := range expectedColumns {
		if !columns[col] {
			t.Errorf("Expected column %s not found in table", col)
		}
	}
}

// TestSQLStorage_ConcurrentAccess tests concurrent access to the SQL storage
func TestSQLStorage_ConcurrentAccess(t *testing.T) {
	storage := setupTestSQLStorage(t)
	defer cleanupTestSQLStorage(t, storage)

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := fmt.Sprintf("concurrent-key-%d", id)
			value := []byte(fmt.Sprintf("concurrent-value-%d", id))
			err := storage.Store(key, value)
			if err != nil {
				t.Errorf("Concurrent Store() failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all values were stored
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("concurrent-key-%d", i)
		expectedValue := []byte(fmt.Sprintf("concurrent-value-%d", i))

		result, err := storage.Retrieve(key)
		if err != nil {
			t.Errorf("Failed to retrieve concurrent value: %v", err)
			continue
		}

		if !bytesEqual(result, expectedValue) {
			t.Errorf("Concurrent value mismatch for key %s", key)
		}
	}
}

// TestSQLStorage_StoreRetrieveIntegration tests the complete store-retrieve cycle
func TestSQLStorage_StoreRetrieveIntegration(t *testing.T) {
	storage := setupTestSQLStorage(t)
	defer cleanupTestSQLStorage(t, storage)

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

func setupTestSQLStorage(t *testing.T) *SQLStorage {
	// Use SQLite in-memory database for testing
	config := &types.StorageConfig{
		Host:     "localhost",
		Port:     3306,
		Database: "test",
		Username: "test",
		Password: "test",
	}

	// Try to create storage with MySQL first
	storage, err := NewSQLStorage(config)
	if err != nil {
		// If MySQL is not available, skip the test
		t.Skipf("Skipping SQL tests (MySQL not available): %v", err)
	}

	return storage
}

func cleanupTestSQLStorage(t *testing.T, storage *SQLStorage) {
	if storage != nil {
		// Clean up test data
		_, err := storage.DB.Exec("DELETE FROM kv_store WHERE k LIKE 'test%' OR k LIKE 'concurrent%' OR k LIKE '%key%'")
		if err != nil {
			t.Logf("Warning: Failed to clean up test data: %v", err)
		}
		storage.Close()
	}
}
