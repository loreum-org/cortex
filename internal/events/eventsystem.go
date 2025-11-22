package events

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"time"

// 	"github.com/loreum-org/cortex/internal/agents"
// 	"github.com/loreum-org/cortex/internal/events"
// 	"github.com/loreum-org/cortex/internal/events/handlers"
// 	"github.com/loreum-org/cortex/internal/rag"
// 	"github.com/loreum-org/cortex/internal/services"
// )

// // EventSystem represents the complete event-driven system
// type EventSystem struct {
// 	EventBus          *events.EventBus
// 	RAGHandler        *handlers.RAGEventHandler
// 	AgentHandler      *handlers.AgentEventHandler
// 	WebSocketHandler  *handlers.WebSocketEventHandler
// 	APIHandler        *handlers.APIEventHandler
// 	MetricsMiddleware *events.MetricsMiddleware
// 	LoggingMiddleware *events.LoggingMiddleware
// 	ReplayService     *events.EventReplayService
// }

// // NewEventSystem creates a complete event system with all handlers
// func NewEventSystem(ragSystem *rag.RAGSystem, agentRegistry *agents.AgentRegistry, serviceRegistry *services.ServiceRegistryManager, server interface{}) (*EventSystem, error) {
// 	// Create event bus with custom configuration
// 	config := &events.EventBusConfig{
// 		QueueSize:        2000,  // Larger queue for high throughput
// 		WorkerCount:      15,    // More workers for parallel processing
// 		HandlerTimeout:   45 * time.Second,
// 		ShutdownTimeout:  15 * time.Second,
// 		MaxRetries:       3,
// 		RetryDelay:       2 * time.Second,
// 		EnableMetrics:    true,
// 		LogEvents:        true,
// 		EnableEventStore: true,
// 		MaxStoredEvents:  15000, // Larger store for high-throughput system
// 	}

// 	eventBus := events.NewEventBus(config)

// 	// Create middleware
// 	loggingMiddleware := events.NewLoggingMiddleware(events.LogLevelInfo)
// 	metricsMiddleware := events.NewMetricsMiddleware()
// 	tracingMiddleware := events.NewTracingMiddleware("cortex")
// 	validationMiddleware := events.NewValidationMiddleware([]string{})
// 	rateLimitMiddleware := events.NewRateLimitMiddleware(100, time.Minute) // 100 events per minute per type

// 	// Add middleware to event bus
// 	eventBus.AddMiddleware(validationMiddleware)
// 	eventBus.AddMiddleware(rateLimitMiddleware)
// 	eventBus.AddMiddleware(tracingMiddleware)
// 	eventBus.AddMiddleware(loggingMiddleware)
// 	eventBus.AddMiddleware(metricsMiddleware)

// 	// Create domain handlers
// 	ragHandler := handlers.NewRAGEventHandler(ragSystem, eventBus)
// 	agentHandler := handlers.NewAgentEventHandler(agentRegistry, eventBus)
// 	webSocketHandler := handlers.NewWebSocketEventHandler(eventBus)
// 	apiHandler := handlers.NewAPIEventHandler(eventBus, ragSystem, agentRegistry, serviceRegistry, server)

// 	// Subscribe handlers to event bus
// 	if err := eventBus.Subscribe(ragHandler); err != nil {
// 		return nil, fmt.Errorf("failed to subscribe RAG handler: %w", err)
// 	}

// 	if err := eventBus.Subscribe(agentHandler); err != nil {
// 		return nil, fmt.Errorf("failed to subscribe agent handler: %w", err)
// 	}

// 	if err := eventBus.Subscribe(webSocketHandler); err != nil {
// 		return nil, fmt.Errorf("failed to subscribe WebSocket handler: %w", err)
// 	}

// 	if err := eventBus.Subscribe(apiHandler); err != nil {
// 		return nil, fmt.Errorf("failed to subscribe API handler: %w", err)
// 	}

// 	// Create replay service if event store is available
// 	var replayService *events.EventReplayService
// 	if eventStore := eventBus.GetEventStore(); eventStore != nil {
// 		replayService = events.NewEventReplayService(eventStore, eventBus)
// 		log.Printf("✅ Event replay service initialized")
// 	}

// 	log.Printf("🚀 Event System initialized with %d handlers", 4)

// 	return &EventSystem{
// 		EventBus:          eventBus,
// 		RAGHandler:        ragHandler,
// 		AgentHandler:      agentHandler,
// 		WebSocketHandler:  webSocketHandler,
// 		APIHandler:        apiHandler,
// 		MetricsMiddleware: metricsMiddleware,
// 		LoggingMiddleware: loggingMiddleware,
// 		ReplayService:     replayService,
// 	}, nil
// }

// // PublishSystemStartup publishes system startup events
// func (es *EventSystem) PublishSystemStartup() error {
// 	startupEvents := []events.Event{
// 		events.NewEvent(events.EventTypeComponentStarted, "system", map[string]interface{}{
// 			"component": "event_bus",
// 			"timestamp": time.Now(),
// 		}),
// 		events.NewEvent(events.EventTypeComponentStarted, "rag", map[string]interface{}{
// 			"component": "rag_system",
// 			"timestamp": time.Now(),
// 		}),
// 		events.NewEvent(events.EventTypeComponentStarted, "agent", map[string]interface{}{
// 			"component": "agent_registry",
// 			"timestamp": time.Now(),
// 		}),
// 		events.NewEvent(events.EventTypeComponentStarted, "websocket", map[string]interface{}{
// 			"component": "websocket_manager",
// 			"timestamp": time.Now(),
// 		}),
// 	}

// 	for _, event := range startupEvents {
// 		if err := es.EventBus.Publish(event); err != nil {
// 			log.Printf("❌ Failed to publish startup event: %v", err)
// 		}
// 	}

// 	log.Printf("✅ System startup events published")
// 	return nil
// }

// // StartPeriodicHealthChecks starts periodic health check events
// func (es *EventSystem) StartPeriodicHealthChecks(ctx context.Context, interval time.Duration) {
// 	go func() {
// 		ticker := time.NewTicker(interval)
// 		defer ticker.Stop()

// 		for {
// 			select {
// 			case <-ticker.C:
// 				healthEvent := events.NewEvent(events.EventTypeHealthCheck, "system", map[string]interface{}{
// 					"check_type": "periodic",
// 					"timestamp":  time.Now(),
// 				})

// 				if err := es.EventBus.Publish(healthEvent); err != nil {
// 					log.Printf("❌ Failed to publish health check event: %v", err)
// 				}

// 			case <-ctx.Done():
// 				log.Printf("🔄 Stopping periodic health checks")
// 				return
// 			}
// 		}
// 	}()

// 	log.Printf("🔄 Started periodic health checks every %v", interval)
// }

// // StartMetricsCollection starts periodic metrics collection
// func (es *EventSystem) StartMetricsCollection(ctx context.Context, interval time.Duration) {
// 	go func() {
// 		ticker := time.NewTicker(interval)
// 		defer ticker.Stop()

// 		for {
// 			select {
// 			case <-ticker.C:
// 				// Collect system metrics
// 				busMetrics := es.EventBus.GetMetrics()
// 				middlewareMetrics := es.MetricsMiddleware.GetMetrics()

// 				// Collect handler metrics
// 				ragMetrics := es.RAGHandler.GetMetrics()
// 				agentMetrics := es.AgentHandler.GetMetrics()
// 				wsMetrics := es.WebSocketHandler.GetMetrics()
// 				apiMetrics := es.APIHandler.GetMetrics()

// 				// Combine all metrics
// 				allMetrics := map[string]interface{}{
// 					"event_bus":  busMetrics,
// 					"middleware": middlewareMetrics,
// 					"handlers": map[string]interface{}{
// 						"rag":       ragMetrics,
// 						"agent":     agentMetrics,
// 						"websocket": wsMetrics,
// 						"api":       apiMetrics,
// 					},
// 					"timestamp": time.Now(),
// 				}

// 				// Publish metrics event
// 				metricsEvent := events.NewEvent(events.EventTypeMetricsUpdated, "system", events.MetricsData{
// 					Type:      "system_metrics",
// 					Values:    allMetrics,
// 					Source:    "event_system",
// 					Timestamp: time.Now(),
// 				})

// 				if err := es.EventBus.Publish(metricsEvent); err != nil {
// 					log.Printf("❌ Failed to publish metrics event: %v", err)
// 				}

// 			case <-ctx.Done():
// 				log.Printf("🔄 Stopping metrics collection")
// 				return
// 			}
// 		}
// 	}()

// 	log.Printf("📊 Started metrics collection every %v", interval)
// }

// // ProcessWebSocketMessage processes a WebSocket message through the event system
// func (es *EventSystem) ProcessWebSocketMessage(conn *handlers.WebSocketConnection, msg *handlers.UnifiedMessage) error {
// 	// Update connection last seen
// 	conn.LastSeen = time.Now()

// 	// Process through WebSocket handler
// 	return es.WebSocketHandler.HandleWebSocketMessage(conn, msg)
// }

// // RegisterWebSocketConnection registers a WebSocket connection with the event system
// func (es *EventSystem) RegisterWebSocketConnection(conn *handlers.WebSocketConnection) {
// 	es.WebSocketHandler.RegisterConnection(conn)

// 	// Publish connection event
// 	connEvent := events.NewEvent(events.EventTypeWebSocketConnected, "websocket", events.WebSocketMessageData{
// 		ConnectionID: conn.ID,
// 		UserID:       conn.UserID,
// 		MessageType:  "connection",
// 		Data:         map[string]interface{}{"action": "connected"},
// 	})

// 	es.EventBus.Publish(connEvent)
// }

// // UnregisterWebSocketConnection removes a WebSocket connection from the event system
// func (es *EventSystem) UnregisterWebSocketConnection(connectionID string) {
// 	es.WebSocketHandler.UnregisterConnection(connectionID)

// 	// Publish disconnection event
// 	disconnEvent := events.NewEvent(events.EventTypeWebSocketDisconnected, "websocket", events.WebSocketMessageData{
// 		ConnectionID: connectionID,
// 		MessageType:  "connection",
// 		Data:         map[string]interface{}{"action": "disconnected"},
// 	})

// 	es.EventBus.Publish(disconnEvent)
// }

// // GetSystemStatus returns the overall system status
// func (es *EventSystem) GetSystemStatus() map[string]interface{} {
// 	status := map[string]interface{}{
// 		"event_bus": map[string]interface{}{
// 			"metrics":          es.EventBus.GetMetrics(),
// 			"event_types":      es.EventBus.GetEventTypes(),
// 			"middleware_count": len(es.EventBus.GetMiddleware()),
// 		},
// 		"handlers": map[string]interface{}{
// 			"rag":       es.RAGHandler.GetMetrics(),
// 			"agent":     es.AgentHandler.GetMetrics(),
// 			"websocket": es.WebSocketHandler.GetMetrics(),
// 			"api":       es.APIHandler.GetMetrics(),
// 		},
// 		"middleware_metrics": es.MetricsMiddleware.GetMetrics(),
// 		"timestamp":          time.Now(),
// 	}

// 	// Add event store metrics if available
// 	if eventStore := es.EventBus.GetEventStore(); eventStore != nil {
// 		status["event_store"] = eventStore.GetMetrics()
// 	}

// 	// Add replay service status if available
// 	if es.ReplayService != nil {
// 		status["replay_service"] = map[string]interface{}{
// 			"available": true,
// 		}
// 	} else {
// 		status["replay_service"] = map[string]interface{}{
// 			"available": false,
// 		}
// 	}

// 	return status
// }

// // Shutdown gracefully shuts down the event system
// func (es *EventSystem) Shutdown() error {
// 	log.Printf("🔄 Shutting down event system...")

// 	// Publish shutdown events
// 	shutdownEvent := events.NewEvent(events.EventTypeComponentStopped, "system", map[string]interface{}{
// 		"component": "event_system",
// 		"timestamp": time.Now(),
// 	})

// 	es.EventBus.Publish(shutdownEvent)

// 	// Give time for final events to process
// 	time.Sleep(1 * time.Second)

// 	// Shutdown event bus
// 	return es.EventBus.Shutdown()
// }

// // Event Store and Replay Methods

// // GetStoredEvents retrieves events from the event store
// func (es *EventSystem) GetStoredEvents(filter events.EventFilter) ([]events.Event, error) {
// 	eventStore := es.EventBus.GetEventStore()
// 	if eventStore == nil {
// 		return nil, fmt.Errorf("event store not available")
// 	}

// 	return eventStore.GetEvents(filter)
// }

// // GetEventsByType retrieves events of a specific type from the store
// func (es *EventSystem) GetEventsByType(eventType string, limit int) ([]events.Event, error) {
// 	eventStore := es.EventBus.GetEventStore()
// 	if eventStore == nil {
// 		return nil, fmt.Errorf("event store not available")
// 	}

// 	return eventStore.GetEventsByType(eventType, limit)
// }

// // GetEventsByTimeRange retrieves events within a time range
// func (es *EventSystem) GetEventsByTimeRange(start, end time.Time) ([]events.Event, error) {
// 	eventStore := es.EventBus.GetEventStore()
// 	if eventStore == nil {
// 		return nil, fmt.Errorf("event store not available")
// 	}

// 	return eventStore.GetEventsByTimeRange(start, end)
// }

// // ReplayEvents replays stored events matching the filter
// func (es *EventSystem) ReplayEvents(ctx context.Context, filter events.EventFilter) error {
// 	if es.ReplayService == nil {
// 		return fmt.Errorf("replay service not available")
// 	}

// 	return es.ReplayService.ReplayEvents(ctx, filter)
// }

// // GetEventStoreMetrics returns event store metrics
// func (es *EventSystem) GetEventStoreMetrics() (events.EventStoreMetrics, error) {
// 	eventStore := es.EventBus.GetEventStore()
// 	if eventStore == nil {
// 		return events.EventStoreMetrics{}, fmt.Errorf("event store not available")
// 	}

// 	return eventStore.GetMetrics(), nil
// }