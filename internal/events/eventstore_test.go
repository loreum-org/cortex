package events

import (
	"context"
	"testing"
	"time"
)


func TestInMemoryEventStore(t *testing.T) {
	store := NewInMemoryEventStore(100)
	
	// Test storing events
	event1 := NewEvent("test.event1", "test_source", map[string]interface{}{"key": "value1"})
	event2 := NewEvent("test.event2", "test_source", map[string]interface{}{"key": "value2"})
	event3 := NewEvent("test.event1", "other_source", map[string]interface{}{"key": "value3"})
	
	if err := store.Store(event1); err != nil {
		t.Fatalf("Failed to store event1: %v", err)
	}
	
	if err := store.Store(event2); err != nil {
		t.Fatalf("Failed to store event2: %v", err)
	}
	
	if err := store.Store(event3); err != nil {
		t.Fatalf("Failed to store event3: %v", err)
	}
	
	// Test getting all events
	allEvents, err := store.GetEvents(EventFilter{})
	if err != nil {
		t.Fatalf("Failed to get all events: %v", err)
	}
	
	if len(allEvents) != 3 {
		t.Errorf("Expected 3 events, got %d", len(allEvents))
	}
	
	// Test getting events by type
	typeEvents, err := store.GetEventsByType("test.event1", 10)
	if err != nil {
		t.Fatalf("Failed to get events by type: %v", err)
	}
	
	if len(typeEvents) != 2 {
		t.Errorf("Expected 2 events of type test.event1, got %d", len(typeEvents))
	}
	
	// Test filtering by source
	sourceEvents, err := store.GetEvents(EventFilter{
		Sources: []string{"test_source"},
	})
	if err != nil {
		t.Fatalf("Failed to get events by source: %v", err)
	}
	
	if len(sourceEvents) != 2 {
		t.Errorf("Expected 2 events from test_source, got %d", len(sourceEvents))
	}
	
	// Test metrics
	metrics := store.GetMetrics()
	if metrics.TotalEvents != 3 {
		t.Errorf("Expected 3 total events in metrics, got %d", metrics.TotalEvents)
	}
	
	if metrics.EventsByType["test.event1"] != 2 {
		t.Errorf("Expected 2 test.event1 events in metrics, got %d", metrics.EventsByType["test.event1"])
	}
}

func TestEventStoreTimeRange(t *testing.T) {
	store := NewInMemoryEventStore(100)
	
	// Create events with different timestamps
	now := time.Now()
	event1 := NewEvent("test.event", "source", map[string]interface{}{})
	event1.Timestamp = now.Add(-2 * time.Hour)
	
	event2 := NewEvent("test.event", "source", map[string]interface{}{})
	event2.Timestamp = now.Add(-1 * time.Hour)
	
	event3 := NewEvent("test.event", "source", map[string]interface{}{})
	event3.Timestamp = now
	
	store.Store(event1)
	store.Store(event2)
	store.Store(event3)
	
	// Test time range filtering
	start := now.Add(-90 * time.Minute)
	end := now.Add(-30 * time.Minute)
	
	rangeEvents, err := store.GetEventsByTimeRange(start, end)
	if err != nil {
		t.Fatalf("Failed to get events by time range: %v", err)
	}
	
	if len(rangeEvents) != 1 {
		t.Errorf("Expected 1 event in time range, got %d", len(rangeEvents))
	}
	
	if !rangeEvents[0].Timestamp.Equal(event2.Timestamp) {
		t.Errorf("Got wrong event from time range filter")
	}
}

func TestEventStoreCorrelationID(t *testing.T) {
	store := NewInMemoryEventStore(100)
	
	correlationID := "test-correlation-123"
	
	event1 := NewEvent("test.event1", "source", map[string]interface{}{})
	event1.CorrelationID = correlationID
	
	event2 := NewEvent("test.event2", "source", map[string]interface{}{})
	event2.CorrelationID = correlationID
	
	event3 := NewEvent("test.event3", "source", map[string]interface{}{})
	event3.CorrelationID = "other-correlation"
	
	store.Store(event1)
	store.Store(event2)
	store.Store(event3)
	
	// Test correlation ID filtering
	correlatedEvents, err := store.GetEventsByCorrelationID(correlationID)
	if err != nil {
		t.Fatalf("Failed to get events by correlation ID: %v", err)
	}
	
	if len(correlatedEvents) != 2 {
		t.Errorf("Expected 2 events with correlation ID, got %d", len(correlatedEvents))
	}
}

func TestEventStoreMaxEvents(t *testing.T) {
	store := NewInMemoryEventStore(2) // Limit to 2 events
	
	event1 := NewEvent("test.event1", "source", map[string]interface{}{})
	event2 := NewEvent("test.event2", "source", map[string]interface{}{})
	event3 := NewEvent("test.event3", "source", map[string]interface{}{})
	
	store.Store(event1)
	store.Store(event2)
	store.Store(event3) // This should evict event1
	
	allEvents, err := store.GetEvents(EventFilter{})
	if err != nil {
		t.Fatalf("Failed to get all events: %v", err)
	}
	
	if len(allEvents) != 2 {
		t.Errorf("Expected 2 events (max limit), got %d", len(allEvents))
	}
	
	// Check that event1 was evicted and we have event2 and event3
	hasEvent1 := false
	hasEvent2 := false
	hasEvent3 := false
	
	for _, event := range allEvents {
		if event.Type == "test.event1" {
			hasEvent1 = true
		} else if event.Type == "test.event2" {
			hasEvent2 = true
		} else if event.Type == "test.event3" {
			hasEvent3 = true
		}
	}
	
	if hasEvent1 {
		t.Error("Event1 should have been evicted")
	}
	
	if !hasEvent2 || !hasEvent3 {
		t.Error("Event2 and Event3 should still be present")
	}
}

func TestEventReplayService(t *testing.T) {
	// Create event bus and store
	config := DefaultEventBusConfig()
	config.QueueSize = 10
	config.WorkerCount = 2
	eventBus := NewEventBus(config)
	store := NewInMemoryEventStore(100)
	replayService := NewEventReplayService(store, eventBus)
	
	// EventBus starts automatically, no need to call Start/Stop
	
	// Store some events
	event1 := NewEvent("test.replay1", "source", map[string]interface{}{"data": "value1"})
	event2 := NewEvent("test.replay2", "source", map[string]interface{}{"data": "value2"})
	
	store.Store(event1)
	store.Store(event2)
	
	// Set up subscriber to catch replayed events
	testHandler := NewTestEventHandler("test_handler", []string{"test.replay1", "test.replay2"})
	
	eventBus.Subscribe(testHandler)
	
	// Wait for subscribers to be registered
	time.Sleep(100 * time.Millisecond)
	
	// Replay events
	ctx := context.Background()
	filter := EventFilter{
		EventTypes: []string{"test.replay1", "test.replay2"},
	}
	
	if err := replayService.ReplayEvents(ctx, filter); err != nil {
		t.Fatalf("Failed to replay events: %v", err)
	}
	
	// Wait for events to be processed
	time.Sleep(200 * time.Millisecond)
	
	// Check that events were replayed
	replayedEvents := testHandler.GetReceivedEvents()
	if len(replayedEvents) != 2 {
		t.Errorf("Expected 2 replayed events, got %d", len(replayedEvents))
	}
	
	// Check that replayed events have replay metadata
	for _, event := range replayedEvents {
		if replay, exists := event.Metadata["replay"]; !exists || replay != true {
			t.Error("Replayed event should have replay=true in metadata")
		}
		
		if _, exists := event.Metadata["original_id"]; !exists {
			t.Error("Replayed event should have original_id in metadata")
		}
	}
	
	// Cleanup
	eventBus.Shutdown()
}

func TestEventStoreMiddleware(t *testing.T) {
	store := NewInMemoryEventStore(100)
	middleware := NewEventStoreMiddleware(store)
	
	event := NewEvent("test.middleware", "source", map[string]interface{}{})
	
	// Test store with middleware
	if err := middleware.Store(event); err != nil {
		t.Fatalf("Failed to store event through middleware: %v", err)
	}
	
	// Test get with middleware
	events, err := middleware.GetEvents(EventFilter{})
	if err != nil {
		t.Fatalf("Failed to get events through middleware: %v", err)
	}
	
	if len(events) != 1 {
		t.Errorf("Expected 1 event through middleware, got %d", len(events))
	}
}