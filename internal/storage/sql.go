package storage

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/loreum-org/cortex/pkg/types"
)

// SQLStorage implements the StorageService interface for SQL databases
type SQLStorage struct {
	DB     *sql.DB
	Config *types.StorageConfig
}

// NewSQLStorage creates a new SQL storage service
func NewSQLStorage(config *types.StorageConfig) (*SQLStorage, error) {
	// Create the DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database)

	// Add options if any
	if len(config.Options) > 0 {
		dsn += "?"
		for key, value := range config.Options {
			dsn += key + "=" + value + "&"
		}
		// Remove trailing "&"
		dsn = dsn[:len(dsn)-1]
	}

	// Connect to the database
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// Check the connection
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return &SQLStorage{
		DB:     db,
		Config: config,
	}, nil
}

// Store stores a value with the given key
func (s *SQLStorage) Store(key string, value []byte) error {
	// Create the table if it doesn't exist
	_, err := s.DB.Exec(`
		CREATE TABLE IF NOT EXISTS kv_store (
			k VARCHAR(255) PRIMARY KEY,
			v BLOB,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Insert or update the value
	_, err = s.DB.Exec(`
		INSERT INTO kv_store (k, v) VALUES (?, ?)
		ON DUPLICATE KEY UPDATE v = ?
	`, key, value, value)

	return err
}

// Retrieve retrieves a value with the given key
func (s *SQLStorage) Retrieve(key string) ([]byte, error) {
	// Check if the table exists
	_, err := s.DB.Exec(`
		CREATE TABLE IF NOT EXISTS kv_store (
			k VARCHAR(255) PRIMARY KEY,
			v BLOB,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, err
	}

	// Query the value
	var value []byte
	err = s.DB.QueryRow(`
		SELECT v FROM kv_store WHERE k = ?
	`, key).Scan(&value)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return value, err
}

// Delete deletes a value with the given key
func (s *SQLStorage) Delete(key string) error {
	// Check if the table exists
	_, err := s.DB.Exec(`
		CREATE TABLE IF NOT EXISTS kv_store (
			k VARCHAR(255) PRIMARY KEY,
			v BLOB,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Delete the value
	_, err = s.DB.Exec(`
		DELETE FROM kv_store WHERE k = ?
	`, key)

	return err
}

// Close closes the database connection
func (s *SQLStorage) Close() error {
	return s.DB.Close()
}
