package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// EventStore defines the interface for event persistence
type EventStore interface {
	Store(event Event) error
	GetEvents(filter EventFilter) ([]Event, error)
	GetEventsByType(eventType string, limit int) ([]Event, error)
	GetEventsByTimeRange(start, end time.Time) ([]Event, error)
	GetEventsByCorrelationID(correlationID string) ([]Event, error)
	GetEventStream(filter EventFilter, callback func(Event) error) error
	Close() error
	GetMetrics() EventStoreMetrics
}

// EventFilter defines criteria for querying events
type EventFilter struct {
	EventTypes    []string                 `json:"event_types,omitempty"`
	Sources       []string                 `json:"sources,omitempty"`
	Targets       []string                 `json:"targets,omitempty"`
	StartTime     *time.Time               `json:"start_time,omitempty"`
	EndTime       *time.Time               `json:"end_time,omitempty"`
	CorrelationID string                   `json:"correlation_id,omitempty"`
	Metadata      map[string]interface{}   `json:"metadata,omitempty"`
	Limit         int                      `json:"limit,omitempty"`
	Offset        int                      `json:"offset,omitempty"`
}

// EventStoreMetrics contains metrics about the event store
type EventStoreMetrics struct {
	TotalEvents      int64         `json:"total_events"`
	StorageSize      int64         `json:"storage_size_bytes"`
	AverageStoreTime time.Duration `json:"average_store_time"`
	ErrorCount       int64         `json:"error_count"`
	LastStoreTime    time.Time     `json:"last_store_time"`
	EventsByType     map[string]int64 `json:"events_by_type"`
}

// InMemoryEventStore is a simple in-memory implementation of EventStore
type InMemoryEventStore struct {
	events    []Event
	mu        sync.RWMutex
	metrics   EventStoreMetrics
	maxEvents int
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore(maxEvents int) *InMemoryEventStore {
	if maxEvents <= 0 {
		maxEvents = 10000 // Default maximum events
	}
	
	return &InMemoryEventStore{
		events:    make([]Event, 0, maxEvents),
		maxEvents: maxEvents,
		metrics: EventStoreMetrics{
			EventsByType: make(map[string]int64),
		},
	}
}

// Store stores an event in memory
func (s *InMemoryEventStore) Store(event Event) error {
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
	
	// Add to events slice
	s.events = append(s.events, event)
	
	// Maintain maximum events limit (FIFO)
	if len(s.events) > s.maxEvents {
		s.events = s.events[1:]
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
func (s *InMemoryEventStore) GetEvents(filter EventFilter) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []Event
	
	for _, event := range s.events {
		if s.matchesFilter(event, filter) {
			result = append(result, event)
		}
	}
	
	// Apply limit and offset
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}
	
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}
	
	return result, nil
}

// GetEventsByType retrieves events of a specific type
func (s *InMemoryEventStore) GetEventsByType(eventType string, limit int) ([]Event, error) {
	filter := EventFilter{
		EventTypes: []string{eventType},
		Limit:      limit,
	}
	return s.GetEvents(filter)
}

// GetEventsByTimeRange retrieves events within a time range
func (s *InMemoryEventStore) GetEventsByTimeRange(start, end time.Time) ([]Event, error) {
	filter := EventFilter{
		StartTime: &start,
		EndTime:   &end,
	}
	return s.GetEvents(filter)
}

// GetEventsByCorrelationID retrieves events with the same correlation ID
func (s *InMemoryEventStore) GetEventsByCorrelationID(correlationID string) ([]Event, error) {
	filter := EventFilter{
		CorrelationID: correlationID,
	}
	return s.GetEvents(filter)
}

// GetEventStream streams events that match the filter to a callback function
func (s *InMemoryEventStore) GetEventStream(filter EventFilter, callback func(Event) error) error {
	events, err := s.GetEvents(filter)
	if err != nil {
		return err
	}
	
	for _, event := range events {
		if err := callback(event); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}
	}
	
	return nil
}

// Close closes the event store (no-op for in-memory)
func (s *InMemoryEventStore) Close() error {
	return nil
}

// GetMetrics returns current store metrics
func (s *InMemoryEventStore) GetMetrics() EventStoreMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Calculate storage size (approximate)
	storageSize := int64(0)
	for _, event := range s.events {
		if data, err := json.Marshal(event); err == nil {
			storageSize += int64(len(data))
		}
	}
	
	metrics := s.metrics
	metrics.StorageSize = storageSize
	
	return metrics
}

// Helper method to check if an event matches filter criteria
func (s *InMemoryEventStore) matchesFilter(event Event, filter EventFilter) bool {
	// Check event types
	if len(filter.EventTypes) > 0 {
		found := false
		for _, eventType := range filter.EventTypes {
			if event.Type == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check sources
	if len(filter.Sources) > 0 {
		found := false
		for _, source := range filter.Sources {
			if event.Source == source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check targets
	if len(filter.Targets) > 0 {
		found := false
		for _, target := range filter.Targets {
			if event.Target == target {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check time range
	if filter.StartTime != nil && event.Timestamp.Before(*filter.StartTime) {
		return false
	}
	
	if filter.EndTime != nil && event.Timestamp.After(*filter.EndTime) {
		return false
	}
	
	// Check correlation ID
	if filter.CorrelationID != "" && event.CorrelationID != filter.CorrelationID {
		return false
	}
	
	// Check metadata (simple key-value matching)
	if len(filter.Metadata) > 0 {
		for key, value := range filter.Metadata {
			if eventValue, exists := event.Metadata[key]; !exists || eventValue != value {
				return false
			}
		}
	}
	
	return true
}

// EventStoreMiddleware wraps an event store with logging and metrics
type EventStoreMiddleware struct {
	store EventStore
}

// NewEventStoreMiddleware creates middleware around an event store
func NewEventStoreMiddleware(store EventStore) *EventStoreMiddleware {
	return &EventStoreMiddleware{store: store}
}

// Store wraps the store operation with logging
func (m *EventStoreMiddleware) Store(event Event) error {
	start := time.Now()
	err := m.store.Store(event)
	duration := time.Since(start)
	
	if err != nil {
		log.Printf("❌ EventStore: Failed to store event %s (%s): %v (took %v)", event.ID, event.Type, err, duration)
	} else {
		log.Printf("💾 EventStore: Stored event %s (%s) in %v", event.ID, event.Type, duration)
	}
	
	return err
}

// GetEvents wraps the get operation with logging
func (m *EventStoreMiddleware) GetEvents(filter EventFilter) ([]Event, error) {
	start := time.Now()
	events, err := m.store.GetEvents(filter)
	duration := time.Since(start)
	
	if err != nil {
		log.Printf("❌ EventStore: Failed to get events: %v (took %v)", err, duration)
	} else {
		log.Printf("📖 EventStore: Retrieved %d events in %v", len(events), duration)
	}
	
	return events, err
}

// Delegate other methods to the wrapped store
func (m *EventStoreMiddleware) GetEventsByType(eventType string, limit int) ([]Event, error) {
	return m.store.GetEventsByType(eventType, limit)
}

func (m *EventStoreMiddleware) GetEventsByTimeRange(start, end time.Time) ([]Event, error) {
	return m.store.GetEventsByTimeRange(start, end)
}

func (m *EventStoreMiddleware) GetEventsByCorrelationID(correlationID string) ([]Event, error) {
	return m.store.GetEventsByCorrelationID(correlationID)
}

func (m *EventStoreMiddleware) GetEventStream(filter EventFilter, callback func(Event) error) error {
	return m.store.GetEventStream(filter, callback)
}

func (m *EventStoreMiddleware) Close() error {
	return m.store.Close()
}

func (m *EventStoreMiddleware) GetMetrics() EventStoreMetrics {
	return m.store.GetMetrics()
}

// EventReplayService provides event replay functionality
type EventReplayService struct {
	eventStore EventStore
	eventBus   *EventBus
}

// NewEventReplayService creates a new event replay service
func NewEventReplayService(eventStore EventStore, eventBus *EventBus) *EventReplayService {
	return &EventReplayService{
		eventStore: eventStore,
		eventBus:   eventBus,
	}
}

// ReplayEvents replays stored events matching the filter
func (r *EventReplayService) ReplayEvents(ctx context.Context, filter EventFilter) error {
	log.Printf("🔄 Event Replay: Starting replay with filter %+v", filter)
	
	count := 0
	err := r.eventStore.GetEventStream(filter, func(event Event) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Modify event to indicate it's a replay
			replayEvent := event
			replayEvent.ID = generateEventID() // New ID for replay
			if replayEvent.Metadata == nil {
				replayEvent.Metadata = make(map[string]interface{})
			}
			replayEvent.Metadata["replay"] = true
			replayEvent.Metadata["original_id"] = event.ID
			replayEvent.Metadata["original_timestamp"] = event.Timestamp
			replayEvent.Timestamp = time.Now()
			
			if err := r.eventBus.Publish(replayEvent); err != nil {
				log.Printf("❌ Event Replay: Failed to publish replay event %s: %v", event.ID, err)
				return err
			}
			
			count++
			return nil
		}
	})
	
	if err != nil {
		log.Printf("❌ Event Replay: Failed after %d events: %v", count, err)
		return err
	}
	
	log.Printf("✅ Event Replay: Successfully replayed %d events", count)
	return nil
}

// GetReplayableEvents returns events that can be replayed
func (r *EventReplayService) GetReplayableEvents(filter EventFilter) ([]Event, error) {
	return r.eventStore.GetEvents(filter)
}