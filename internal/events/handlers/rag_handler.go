package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/events"
	"github.com/loreum-org/cortex/internal/rag"
)

// RAGEventHandler handles RAG system events
type RAGEventHandler struct {
	ragSystem *rag.RAGSystem
	eventBus  *events.EventBus
	name      string
}

// NewRAGEventHandler creates a new RAG event handler
func NewRAGEventHandler(ragSystem *rag.RAGSystem, eventBus *events.EventBus) *RAGEventHandler {
	return &RAGEventHandler{
		ragSystem: ragSystem,
		eventBus:  eventBus,
		name:      "rag_handler",
	}
}

// HandlerName returns the handler name
func (rh *RAGEventHandler) HandlerName() string {
	return rh.name
}

// SubscribedEvents returns the events this handler subscribes to
func (rh *RAGEventHandler) SubscribedEvents() []string {
	return []string{
		events.EventTypeQuerySubmit,
		events.EventTypeComponentStarted,
		events.EventTypeHealthCheck,
	}
}

// Handle processes events
func (rh *RAGEventHandler) Handle(ctx context.Context, event events.Event) error {
	switch event.Type {
	case events.EventTypeQuerySubmit:
		return rh.handleQuerySubmit(ctx, event)
	case events.EventTypeComponentStarted:
		return rh.handleComponentStarted(ctx, event)
	case events.EventTypeHealthCheck:
		return rh.handleHealthCheck(ctx, event)
	default:
		return fmt.Errorf("unsupported event type: %s", event.Type)
	}
}

// handleQuerySubmit processes query submission events
func (rh *RAGEventHandler) handleQuerySubmit(ctx context.Context, event events.Event) error {
	log.Printf("🧠 RAG Handler: Processing query submission %s", event.ID)
	
	// Extract query data
	queryData, ok := event.Data.(events.QueryData)
	if !ok {
		return fmt.Errorf("invalid query data type")
	}
	
	if rh.ragSystem == nil {
		return rh.publishError(event, "RAG system not available")
	}
	
	// Publish query started event
	rh.publishQueryStarted(event, queryData)
	
	// Process query with context timeout
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	start := time.Now()
	
	// Execute RAG query
	result, err := rh.ragSystem.Query(queryCtx, queryData.Text)
	
	duration := time.Since(start)
	
	if err != nil {
		log.Printf("❌ RAG Query failed for %s: %v", event.ID, err)
		return rh.publishQueryError(event, queryData, err, duration)
	}
	
	log.Printf("✅ RAG Query completed for %s in %v", event.ID, duration)
	
	// Publish successful result
	return rh.publishQueryResult(event, queryData, result, duration)
}

// handleComponentStarted handles component startup events
func (rh *RAGEventHandler) handleComponentStarted(ctx context.Context, event events.Event) error {
	// Check if this is a RAG-related component
	if event.Source == "rag" || event.Source == "vector_db" || event.Source == "consciousness" {
		log.Printf("🔄 RAG Handler: Component %s started", event.Source)
		
		// Optionally perform health checks or initialization
		if rh.ragSystem != nil {
			// Could trigger model loading, vector DB verification, etc.
			return rh.performHealthCheck(ctx, event)
		}
	}
	
	return nil
}

// handleHealthCheck processes health check events
func (rh *RAGEventHandler) handleHealthCheck(ctx context.Context, event events.Event) error {
	return rh.performHealthCheck(ctx, event)
}

// performHealthCheck checks RAG system health
func (rh *RAGEventHandler) performHealthCheck(ctx context.Context, event events.Event) error {
	if rh.ragSystem == nil {
		return rh.publishHealthStatus(event, "unhealthy", "RAG system not available")
	}
	
	// Check vector database
	vectorDBHealthy := rh.ragSystem.VectorDB != nil
	
	// Check model manager
	modelManagerHealthy := rh.ragSystem.ModelManager != nil
	
	// Check context manager
	contextManagerHealthy := rh.ragSystem.ContextManager != nil
	
	status := "healthy"
	message := "All RAG components operational"
	
	if !vectorDBHealthy || !modelManagerHealthy || !contextManagerHealthy {
		status = "degraded"
		message = fmt.Sprintf("RAG components status: VectorDB=%v, ModelManager=%v, ContextManager=%v",
			vectorDBHealthy, modelManagerHealthy, contextManagerHealthy)
	}
	
	return rh.publishHealthStatus(event, status, message)
}

// Helper methods for publishing events

func (rh *RAGEventHandler) publishQueryStarted(originalEvent events.Event, queryData events.QueryData) {
	event := events.NewEventWithCorrelation(
		events.EventTypeQueryStarted,
		"rag",
		events.QueryResultData{
			QueryID: queryData.QueryID,
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	rh.eventBus.Publish(event)
}

func (rh *RAGEventHandler) publishQueryResult(originalEvent events.Event, queryData events.QueryData, result string, duration time.Duration) error {
	resultData := events.QueryResultData{
		QueryID:  queryData.QueryID,
		Result:   result,
		Source:   "rag",
		Duration: duration,
		Metadata: map[string]interface{}{
			"use_rag":       queryData.UseRAG,
			"user_id":       queryData.UserID,
			"connection_id": queryData.ConnectionID,
		},
	}
	
	event := events.NewEventWithCorrelation(
		events.EventTypeQueryProcessed,
		"rag",
		resultData,
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return rh.eventBus.Publish(event)
}

func (rh *RAGEventHandler) publishQueryError(originalEvent events.Event, queryData events.QueryData, err error, duration time.Duration) error {
	resultData := events.QueryResultData{
		QueryID:  queryData.QueryID,
		Source:   "rag",
		Duration: duration,
		Error:    err.Error(),
		Metadata: map[string]interface{}{
			"use_rag":       queryData.UseRAG,
			"user_id":       queryData.UserID,
			"connection_id": queryData.ConnectionID,
		},
	}
	
	event := events.NewEventWithCorrelation(
		events.EventTypeQueryFailed,
		"rag",
		resultData,
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return rh.eventBus.Publish(event)
}

func (rh *RAGEventHandler) publishError(originalEvent events.Event, message string) error {
	errorEvent := events.NewEventWithCorrelation(
		events.EventTypeError,
		"rag",
		map[string]interface{}{
			"error":   message,
			"handler": rh.name,
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return rh.eventBus.Publish(errorEvent)
}

func (rh *RAGEventHandler) publishHealthStatus(originalEvent events.Event, status, message string) error {
	healthEvent := events.NewEventWithCorrelation(
		"system.health.status",
		"rag",
		map[string]interface{}{
			"component": "rag",
			"status":    status,
			"message":   message,
			"timestamp": time.Now(),
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return rh.eventBus.Publish(healthEvent)
}

// GetMetrics returns RAG-specific metrics
func (rh *RAGEventHandler) GetMetrics() map[string]interface{} {
	if rh.ragSystem == nil {
		return map[string]interface{}{
			"status": "unavailable",
		}
	}
	
	return map[string]interface{}{
		"status":              "available",
		"vector_db_active":    rh.ragSystem.VectorDB != nil,
		"model_manager_active": rh.ragSystem.ModelManager != nil,
		"context_manager_active": rh.ragSystem.ContextManager != nil,
		"handler_name":        rh.name,
	}
}