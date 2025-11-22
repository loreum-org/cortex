package events

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestEventHandler is a simple test handler
type TestEventHandler struct {
	name           string
	subscribedTo   []string
	receivedEvents []Event
	mu             sync.Mutex
}

func NewTestEventHandler(name string, eventTypes []string) *TestEventHandler {
	return &TestEventHandler{
		name:           name,
		subscribedTo:   eventTypes,
		receivedEvents: make([]Event, 0),
	}
}

func (th *TestEventHandler) HandlerName() string {
	return th.name
}

func (th *TestEventHandler) SubscribedEvents() []string {
	return th.subscribedTo
}

func (th *TestEventHandler) Handle(ctx context.Context, event Event) error {
	th.mu.Lock()
	defer th.mu.Unlock()
	th.receivedEvents = append(th.receivedEvents, event)
	return nil
}

func (th *TestEventHandler) GetReceivedEvents() []Event {
	th.mu.Lock()
	defer th.mu.Unlock()
	result := make([]Event, len(th.receivedEvents))
	copy(result, th.receivedEvents)
	return result
}

func (th *TestEventHandler) GetEventCount() int {
	th.mu.Lock()
	defer th.mu.Unlock()
	return len(th.receivedEvents)
}

func TestEventBusBasicFunctionality(t *testing.T) {
	// Create event bus
	config := DefaultEventBusConfig()
	config.WorkerCount = 2
	config.QueueSize = 10
	eb := NewEventBus(config)
	defer eb.Shutdown()
	
	// Create test handler
	handler := NewTestEventHandler("test_handler", []string{"test.event"})
	
	// Subscribe handler
	err := eb.Subscribe(handler)
	if err != nil {
		t.Fatalf("Failed to subscribe handler: %v", err)
	}
	
	// Create and publish event
	event := NewEvent("test.event", "test", map[string]interface{}{
		"message": "hello world",
	})
	
	err = eb.Publish(event)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}
	
	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	
	// Verify event was received
	receivedEvents := handler.GetReceivedEvents()
	if len(receivedEvents) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(receivedEvents))
	}
	
	if receivedEvents[0].Type != "test.event" {
		t.Errorf("Expected event type 'test.event', got '%s'", receivedEvents[0].Type)
	}
}

func TestEventBusMultipleHandlers(t *testing.T) {
	eb := NewEventBus(DefaultEventBusConfig())
	defer eb.Shutdown()
	
	// Create multiple handlers for the same event type
	handler1 := NewTestEventHandler("handler1", []string{"test.event"})
	handler2 := NewTestEventHandler("handler2", []string{"test.event"})
	handler3 := NewTestEventHandler("handler3", []string{"other.event"})
	
	// Subscribe all handlers
	eb.Subscribe(handler1)
	eb.Subscribe(handler2)
	eb.Subscribe(handler3)
	
	// Publish event
	event := NewEvent("test.event", "test", "test data")
	eb.Publish(event)
	
	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	
	// Verify correct handlers received the event
	if handler1.GetEventCount() != 1 {
		t.Errorf("Handler1 should have received 1 event, got %d", handler1.GetEventCount())
	}
	
	if handler2.GetEventCount() != 1 {
		t.Errorf("Handler2 should have received 1 event, got %d", handler2.GetEventCount())
	}
	
	if handler3.GetEventCount() != 0 {
		t.Errorf("Handler3 should have received 0 events, got %d", handler3.GetEventCount())
	}
}

func TestEventBusUnsubscribe(t *testing.T) {
	eb := NewEventBus(DefaultEventBusConfig())
	defer eb.Shutdown()
	
	handler := NewTestEventHandler("test_handler", []string{"test.event"})
	
	// Subscribe and publish event
	eb.Subscribe(handler)
	eb.Publish(NewEvent("test.event", "test", "data1"))
	time.Sleep(50 * time.Millisecond)
	
	// Unsubscribe and publish another event
	eb.Unsubscribe(handler)
	eb.Publish(NewEvent("test.event", "test", "data2"))
	time.Sleep(50 * time.Millisecond)
	
	// Should only have received the first event
	if handler.GetEventCount() != 1 {
		t.Errorf("Expected 1 event after unsubscribe, got %d", handler.GetEventCount())
	}
}

func TestEventBusMetrics(t *testing.T) {
	eb := NewEventBus(DefaultEventBusConfig())
	defer eb.Shutdown()
	
	handler := NewTestEventHandler("test_handler", []string{"test.event"})
	eb.Subscribe(handler)
	
	// Publish multiple events
	for i := 0; i < 5; i++ {
		eb.Publish(NewEvent("test.event", "test", i))
	}
	
	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	
	// Check metrics
	metrics := eb.GetMetrics()
	
	if metrics.EventsPublished != 5 {
		t.Errorf("Expected 5 published events, got %d", metrics.EventsPublished)
	}
	
	if metrics.EventsProcessed != 5 {
		t.Errorf("Expected 5 processed events, got %d", metrics.EventsProcessed)
	}
	
	if metrics.HandlersRegistered != 1 {
		t.Errorf("Expected 1 registered handler, got %d", metrics.HandlersRegistered)
	}
}

func TestEventBusMiddleware(t *testing.T) {
	eb := NewEventBus(DefaultEventBusConfig())
	defer eb.Shutdown()
	
	// Add logging middleware
	loggingMiddleware := NewLoggingMiddleware(LogLevelDebug)
	eb.AddMiddleware(loggingMiddleware)
	
	// Add metrics middleware
	metricsMiddleware := NewMetricsMiddleware()
	eb.AddMiddleware(metricsMiddleware)
	
	handler := NewTestEventHandler("test_handler", []string{"test.event"})
	eb.Subscribe(handler)
	
	// Publish events
	for i := 0; i < 3; i++ {
		eb.Publish(NewEvent("test.event", "test", i))
	}
	
	// Wait for processing
	time.Sleep(200 * time.Millisecond)
	
	// Check middleware metrics
	middlewareMetrics := metricsMiddleware.GetMetrics()
	
	if len(middlewareMetrics) == 0 {
		t.Error("Expected middleware metrics to be collected")
	}
	
	if metrics, exists := middlewareMetrics["test.event"]; exists {
		if metrics.Count != 3 {
			t.Errorf("Expected 3 events in middleware metrics, got %d", metrics.Count)
		}
	} else {
		t.Error("Expected 'test.event' metrics in middleware")
	}
}

func TestEventBusConcurrency(t *testing.T) {
	config := DefaultEventBusConfig()
	config.WorkerCount = 5
	eb := NewEventBus(config)
	defer eb.Shutdown()
	
	handler := NewTestEventHandler("test_handler", []string{"test.event"})
	eb.Subscribe(handler)
	
	// Publish events concurrently
	var wg sync.WaitGroup
	eventCount := 100
	
	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			event := NewEvent("test.event", "test", i)
			eb.Publish(event)
		}(i)
	}
	
	wg.Wait()
	
	// Wait for all events to be processed
	time.Sleep(500 * time.Millisecond)
	
	// Check that all events were received
	receivedCount := handler.GetEventCount()
	if receivedCount != eventCount {
		t.Errorf("Expected %d events, received %d", eventCount, receivedCount)
	}
}

func TestEventBusShutdown(t *testing.T) {
	eb := NewEventBus(DefaultEventBusConfig())
	
	handler := NewTestEventHandler("test_handler", []string{"test.event"})
	eb.Subscribe(handler)
	
	// Publish some events
	for i := 0; i < 5; i++ {
		eb.Publish(NewEvent("test.event", "test", i))
	}
	
	// Shutdown
	err := eb.Shutdown()
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
	
	// Try to publish after shutdown (should fail or be ignored)
	err = eb.Publish(NewEvent("test.event", "test", "after shutdown"))
	if err == nil {
		t.Error("Expected error when publishing after shutdown")
	}
}

func BenchmarkEventBus(b *testing.B) {
	eb := NewEventBus(DefaultEventBusConfig())
	defer eb.Shutdown()
	
	handler := NewTestEventHandler("bench_handler", []string{"bench.event"})
	eb.Subscribe(handler)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		event := NewEvent("bench.event", "bench", i)
		eb.Publish(event)
	}
}

func BenchmarkEventBusWithMiddleware(b *testing.B) {
	eb := NewEventBus(DefaultEventBusConfig())
	defer eb.Shutdown()
	
	// Add middleware
	eb.AddMiddleware(NewLoggingMiddleware(LogLevelError)) // Only log errors
	eb.AddMiddleware(NewMetricsMiddleware())
	eb.AddMiddleware(NewTracingMiddleware("bench"))
	
	handler := NewTestEventHandler("bench_handler", []string{"bench.event"})
	eb.Subscribe(handler)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		event := NewEvent("bench.event", "bench", i)
		eb.Publish(event)
	}
}