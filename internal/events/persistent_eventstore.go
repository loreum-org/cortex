package events

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// PersistentEventStore provides persistent event storage using SQLite
type PersistentEventStore struct {
	db         *sql.DB
	dbPath     string
	mu         sync.RWMutex
	metrics    EventStoreMetrics
	maxEvents  int64
	closeOnce  sync.Once
}

// NewPersistentEventStore creates a new persistent event store
func NewPersistentEventStore(dbPath string, maxEvents int64) (*PersistentEventStore, error) {
	if maxEvents <= 0 {
		maxEvents = 100000 // Default maximum events
	}
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Open database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	// Configure database
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Set pragmas for performance
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = 10000",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA mmap_size = 268435456", // 256MB
	}
	
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			log.Printf("⚠️ Failed to set pragma %s: %v", pragma, err)
		}
	}
	
	store := &PersistentEventStore{
		db:        db,
		dbPath:    dbPath,
		maxEvents: maxEvents,
		metrics: EventStoreMetrics{
			EventsByType: make(map[string]int64),
		},
	}
	
	// Initialize schema
	if err := store.initializeSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	
	// Load metrics
	if err := store.loadMetrics(); err != nil {
		log.Printf("⚠️ Failed to load metrics: %v", err)
	}
	
	// Start cleanup routine
	go store.cleanupRoutine()
	
	log.Printf("💾 Persistent EventStore: Initialized at %s (max events: %d)", dbPath, maxEvents)
	return store, nil
}

// initializeSchema creates the necessary tables
func (s *PersistentEventStore) initializeSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		source TEXT NOT NULL,
		target TEXT,
		correlation_id TEXT,
		data TEXT NOT NULL,
		metadata TEXT,
		timestamp DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
	CREATE INDEX IF NOT EXISTS idx_events_source ON events(source);
	CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
	CREATE INDEX IF NOT EXISTS idx_events_correlation_id ON events(correlation_id);
	CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
	
	CREATE TABLE IF NOT EXISTS event_metrics (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	
	_, err := s.db.Exec(schema)
	return err
}

// Store stores an event persistently
func (s *PersistentEventStore) Store(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	start := time.Now()
	
	// Ensure event has ID and timestamp
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	
	// Serialize data and metadata
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		s.metrics.ErrorCount++
		return fmt.Errorf("failed to marshal event data: %w", err)
	}
	
	var metadataJSON []byte
	if event.Metadata != nil {
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			s.metrics.ErrorCount++
			return fmt.Errorf("failed to marshal event metadata: %w", err)
		}
	}
	
	// Insert event
	query := `
	INSERT INTO events (id, type, source, target, correlation_id, data, metadata, timestamp)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err = s.db.Exec(query,
		event.ID,
		event.Type,
		event.Source,
		event.Target,
		event.CorrelationID,
		string(dataJSON),
		string(metadataJSON),
		event.Timestamp,
	)
	
	if err != nil {
		s.metrics.ErrorCount++
		return fmt.Errorf("failed to insert event: %w", err)
	}
	
	// Update metrics
	s.metrics.TotalEvents++
	s.metrics.EventsByType[event.Type]++
	s.metrics.LastStoreTime = time.Now()
	
	// Update average store time
	storeTime := time.Since(start)
	if s.metrics.AverageStoreTime == 0 {
		s.metrics.AverageStoreTime = storeTime
	} else {
		s.metrics.AverageStoreTime = (s.metrics.AverageStoreTime + storeTime) / 2
	}
	
	return nil
}

// GetEvents retrieves events based on filter criteria
func (s *PersistentEventStore) GetEvents(filter EventFilter) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	query := "SELECT id, type, source, target, correlation_id, data, metadata, timestamp FROM events WHERE 1=1"
	args := []interface{}{}
	
	// Build WHERE clause
	if len(filter.EventTypes) > 0 {
		placeholders := make([]string, len(filter.EventTypes))
		for i, eventType := range filter.EventTypes {
			placeholders[i] = "?"
			args = append(args, eventType)
		}
		query += fmt.Sprintf(" AND type IN (%s)", joinPlaceholders(placeholders))
	}
	
	if len(filter.Sources) > 0 {
		placeholders := make([]string, len(filter.Sources))
		for i, source := range filter.Sources {
			placeholders[i] = "?"
			args = append(args, source)
		}
		query += fmt.Sprintf(" AND source IN (%s)", joinPlaceholders(placeholders))
	}
	
	if len(filter.Targets) > 0 {
		placeholders := make([]string, len(filter.Targets))
		for i, target := range filter.Targets {
			placeholders[i] = "?"
			args = append(args, target)
		}
		query += fmt.Sprintf(" AND target IN (%s)", joinPlaceholders(placeholders))
	}
	
	if filter.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, *filter.StartTime)
	}
	
	if filter.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, *filter.EndTime)
	}
	
	if filter.CorrelationID != "" {
		query += " AND correlation_id = ?"
		args = append(args, filter.CorrelationID)
	}
	
	// Order by timestamp
	query += " ORDER BY timestamp DESC"
	
	// Apply limit and offset
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}
	
	// Execute query
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()
	
	var events []Event
	for rows.Next() {
		var event Event
		var dataJSON, metadataJSON sql.NullString
		
		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.Source,
			&event.Target,
			&event.CorrelationID,
			&dataJSON,
			&metadataJSON,
			&event.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		
		// Deserialize data
		if dataJSON.Valid {
			if err := json.Unmarshal([]byte(dataJSON.String), &event.Data); err != nil {
				log.Printf("⚠️ Failed to unmarshal event data for %s: %v", event.ID, err)
				event.Data = map[string]interface{}{"error": "failed to unmarshal data"}
			}
		}
		
		// Deserialize metadata
		if metadataJSON.Valid {
			if err := json.Unmarshal([]byte(metadataJSON.String), &event.Metadata); err != nil {
				log.Printf("⚠️ Failed to unmarshal event metadata for %s: %v", event.ID, err)
				event.Metadata = map[string]interface{}{"error": "failed to unmarshal metadata"}
			}
		}
		
		events = append(events, event)
	}
	
	return events, nil
}

// GetEventsByType retrieves events of a specific type
func (s *PersistentEventStore) GetEventsByType(eventType string, limit int) ([]Event, error) {
	filter := EventFilter{
		EventTypes: []string{eventType},
		Limit:      limit,
	}
	return s.GetEvents(filter)
}

// GetEventsByTimeRange retrieves events within a time range
func (s *PersistentEventStore) GetEventsByTimeRange(start, end time.Time) ([]Event, error) {
	filter := EventFilter{
		StartTime: &start,
		EndTime:   &end,
	}
	return s.GetEvents(filter)
}

// GetEventsByCorrelationID retrieves events with the same correlation ID
func (s *PersistentEventStore) GetEventsByCorrelationID(correlationID string) ([]Event, error) {
	filter := EventFilter{
		CorrelationID: correlationID,
	}
	return s.GetEvents(filter)
}

// GetEventStream streams events that match the filter to a callback function
func (s *PersistentEventStore) GetEventStream(filter EventFilter, callback func(Event) error) error {
	// For SQLite, we'll use a cursor-based approach for large datasets
	batchSize := 1000
	offset := 0
	
	for {
		batchFilter := filter
		batchFilter.Limit = batchSize
		batchFilter.Offset = offset
		
		events, err := s.GetEvents(batchFilter)
		if err != nil {
			return fmt.Errorf("failed to get event batch: %w", err)
		}
		
		if len(events) == 0 {
			break
		}
		
		for _, event := range events {
			if err := callback(event); err != nil {
				return fmt.Errorf("callback error: %w", err)
			}
		}
		
		if len(events) < batchSize {
			break
		}
		
		offset += batchSize
	}
	
	return nil
}

// Close closes the database connection
func (s *PersistentEventStore) Close() error {
	var err error
	s.closeOnce.Do(func() {
		if s.db != nil {
			err = s.db.Close()
			log.Printf("💾 Persistent EventStore: Closed database connection")
		}
	})
	return err
}

// GetMetrics returns current store metrics
func (s *PersistentEventStore) GetMetrics() EventStoreMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Get current storage size
	var storageSize int64
	err := s.db.QueryRow("SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()").Scan(&storageSize)
	if err != nil {
		log.Printf("⚠️ Failed to get storage size: %v", err)
	}
	
	metrics := s.metrics
	metrics.StorageSize = storageSize
	
	return metrics
}

// loadMetrics loads metrics from the database
func (s *PersistentEventStore) loadMetrics() error {
	// Load total events
	var totalEvents int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&totalEvents)
	if err != nil {
		return fmt.Errorf("failed to load total events: %w", err)
	}
	s.metrics.TotalEvents = totalEvents
	
	// Load events by type
	rows, err := s.db.Query("SELECT type, COUNT(*) FROM events GROUP BY type")
	if err != nil {
		return fmt.Errorf("failed to load events by type: %w", err)
	}
	defer rows.Close()
	
	eventsByType := make(map[string]int64)
	for rows.Next() {
		var eventType string
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			return fmt.Errorf("failed to scan event type count: %w", err)
		}
		eventsByType[eventType] = count
	}
	s.metrics.EventsByType = eventsByType
	
	return nil
}

// cleanupRoutine periodically cleans up old events
func (s *PersistentEventStore) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		if err := s.cleanup(); err != nil {
			log.Printf("❌ EventStore cleanup failed: %v", err)
		}
	}
}

// cleanup removes old events to maintain the maximum event limit
func (s *PersistentEventStore) cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if cleanup is needed
	var currentCount int64
	err := s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&currentCount)
	if err != nil {
		return fmt.Errorf("failed to count events: %w", err)
	}
	
	if currentCount <= s.maxEvents {
		return nil
	}
	
	// Delete oldest events
	eventsToDelete := currentCount - s.maxEvents
	
	_, err = s.db.Exec(`
		DELETE FROM events 
		WHERE id IN (
			SELECT id FROM events 
			ORDER BY timestamp ASC 
			LIMIT ?
		)
	`, eventsToDelete)
	
	if err != nil {
		return fmt.Errorf("failed to delete old events: %w", err)
	}
	
	// Vacuum to reclaim space
	_, err = s.db.Exec("VACUUM")
	if err != nil {
		log.Printf("⚠️ Failed to vacuum database: %v", err)
	}
	
	log.Printf("🧹 EventStore: Cleaned up %d old events", eventsToDelete)
	return nil
}

// Helper functions

func joinPlaceholders(placeholders []string) string {
	result := ""
	for i, placeholder := range placeholders {
		if i > 0 {
			result += ","
		}
		result += placeholder
	}
	return result
}

// EventStorageConfig holds configuration for persistent event storage
type EventStorageConfig struct {
	DatabasePath  string `json:"database_path"`
	MaxEvents     int64  `json:"max_events"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// DefaultEventStorageConfig returns default configuration
func DefaultEventStorageConfig() EventStorageConfig {
	return EventStorageConfig{
		DatabasePath:    "./data/events.db",
		MaxEvents:       100000,
		CleanupInterval: 1 * time.Hour,
	}
}