package api

import (
	"context"
	"testing"
	"time"

	"github.com/loreum-org/cortex/internal/events"
	"github.com/loreum-org/cortex/internal/events/handlers"
)

// TestEventCoordination verifies that events flow properly between components
func TestEventCoordination(t *testing.T) {
	// Create event bus
	eventBus := events.NewEventBus(events.DefaultEventBusConfig())
	defer eventBus.Shutdown()

	// Create WebSocket handler
	wsHandler := handlers.NewWebSocketEventHandler(eventBus)

	// Subscribe WebSocket handler to event bus
	err := eventBus.Subscribe(wsHandler)
	if err != nil {
		t.Fatalf("Failed to subscribe WebSocket handler: %v", err)
	}

	// Create a mock WebSocket connection
	mockConn := &handlers.WebSocketConnection{
		ID:     "test-conn-1",
		UserID: "test-user",
		Send:   make(chan interface{}, 10),
	}

	// Register the connection
	wsHandler.RegisterConnection(mockConn)

	// Create a test query event
	queryEvent := events.NewEventWithCorrelation(
		events.EventTypeQuerySubmit,
		"test",
		events.QueryData{
			QueryID:      "test-query-1",
			Text:         "What is the meaning of life?",
			UseRAG:       true,
			UserID:       "test-user",
			ConnectionID: "test-conn-1",
		},
		"test-correlation-1",
	).WithMetadata("connection_id", "test-conn-1")

	// Publish the query event
	err = eventBus.Publish(queryEvent)
	if err != nil {
		t.Fatalf("Failed to publish query event: %v", err)
	}

	// Create a mock response event
	responseEvent := events.NewEventWithCorrelation(
		"query.response",
		"test",
		events.QueryResultData{
			QueryID: "test-query-1",
			Result:  "42",
			Source:  "test",
		},
		"test-correlation-1",
	).WithMetadata("connection_id", "test-conn-1")

	// Publish the response event
	err = eventBus.Publish(responseEvent)
	if err != nil {
		t.Fatalf("Failed to publish response event: %v", err)
	}

	// Wait for message to be processed
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	select {
	case msg := <-mockConn.Send:
		// Verify the message was received
		if msg == nil {
			t.Fatal("Received nil message")
		}
		t.Logf("✅ Successfully received WebSocket message: %+v", msg)
	case <-ctx.Done():
		t.Fatal("Timeout waiting for WebSocket message")
	}

	// Cleanup
	wsHandler.UnregisterConnection("test-conn-1")
}

// TestEventBusMetrics verifies that the event bus provides proper metrics
func TestEventBusMetrics(t *testing.T) {
	eventBus := events.NewEventBus(events.DefaultEventBusConfig())
	defer eventBus.Shutdown()

	// Get initial metrics
	metrics := eventBus.GetMetrics()
	if metrics.StartTime.IsZero() {
		t.Fatal("Expected metrics to be available")
	}

	t.Logf("✅ Event bus metrics: %+v", metrics)

	// Verify metrics structure
	if metrics.EventsPublished < 0 {
		t.Error("Expected non-negative events published count")
	}

	if metrics.EventsProcessed < 0 {
		t.Error("Expected non-negative events processed count")
	}
}

// TestWebSocketHandlerMetrics verifies WebSocket handler metrics
func TestWebSocketHandlerMetrics(t *testing.T) {
	eventBus := events.NewEventBus(events.DefaultEventBusConfig())
	defer eventBus.Shutdown()

	wsHandler := handlers.NewWebSocketEventHandler(eventBus)

	// Get metrics
	metrics := wsHandler.GetMetrics()
	if metrics == nil {
		t.Fatal("Expected WebSocket handler metrics to be available")
	}

	t.Logf("✅ WebSocket handler metrics: %+v", metrics)

	// Verify metrics structure
	if handlerName, ok := metrics["handler_name"].(string); !ok || handlerName != "websocket_handler" {
		t.Error("Expected handler name to be 'websocket_handler'")
	}

	if activeConns, ok := metrics["active_connections"].(int); !ok || activeConns < 0 {
		t.Error("Expected non-negative active connections count")
	}
}
