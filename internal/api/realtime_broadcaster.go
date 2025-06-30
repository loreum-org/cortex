package api

import (
	"log"
	"time"
)

// RealtimeBroadcaster handles real-time broadcasting of system events
type RealtimeBroadcaster struct {
	wsManager *WebSocketManager
	server    *Server
	
	// Broadcasting channels
	systemEvents    chan *WebSocketMessage
	ollamaEvents    chan *WebSocketMessage
	queryEvents     chan *WebSocketMessage
	metricsEvents   chan *WebSocketMessage
	
	// Control
	stopChan chan struct{}
	started  bool
}

// NewRealtimeBroadcaster creates a new real-time broadcaster
func NewRealtimeBroadcaster(wsManager *WebSocketManager, server *Server) *RealtimeBroadcaster {
	return &RealtimeBroadcaster{
		wsManager:     wsManager,
		server:        server,
		systemEvents:  make(chan *WebSocketMessage, 100),
		ollamaEvents:  make(chan *WebSocketMessage, 100),
		queryEvents:   make(chan *WebSocketMessage, 100),
		metricsEvents: make(chan *WebSocketMessage, 100),
		stopChan:      make(chan struct{}),
	}
}

// Start begins real-time broadcasting
func (rb *RealtimeBroadcaster) Start() {
	if rb.started {
		return
	}
	
	rb.started = true
	log.Printf("Starting real-time broadcaster")
	
	// Start event processors
	go rb.processSystemEvents()
	go rb.processOllamaEvents()
	go rb.processQueryEvents()
	go rb.processMetricsEvents()
	
	// Start periodic broadcasters
	go rb.broadcastPeriodicMetrics()
	go rb.broadcastConsciousnessState()
	go rb.broadcastOllamaStatus()
}

// Stop stops the real-time broadcaster
func (rb *RealtimeBroadcaster) Stop() {
	if !rb.started {
		return
	}
	
	log.Printf("Stopping real-time broadcaster")
	close(rb.stopChan)
	rb.started = false
}

// BroadcastSystemEvent broadcasts a system event
func (rb *RealtimeBroadcaster) BroadcastSystemEvent(eventType string, data interface{}) {
	if !rb.started {
		return
	}
	
	message := &WebSocketMessage{
		Type: WSMsgTypeNotification,
		Data: map[string]interface{}{
			"event_type": eventType,
			"data":       data,
			"source":     "system",
		},
		Timestamp: time.Now(),
	}
	
	select {
	case rb.systemEvents <- message:
	default:
		log.Printf("System events channel full, dropping event")
	}
}

// BroadcastOllamaEvent broadcasts an Ollama-related event
func (rb *RealtimeBroadcaster) BroadcastOllamaEvent(eventType string, data interface{}) {
	if !rb.started {
		return
	}
	
	message := &WebSocketMessage{
		Type: WSMsgTypeOllamaStatus,
		Data: map[string]interface{}{
			"event_type": eventType,
			"data":       data,
			"source":     "ollama",
		},
		Timestamp: time.Now(),
	}
	
	select {
	case rb.ollamaEvents <- message:
	default:
		log.Printf("Ollama events channel full, dropping event")
	}
}

// BroadcastQueryEvent broadcasts a query-related event
func (rb *RealtimeBroadcaster) BroadcastQueryEvent(queryID, eventType string, data interface{}) {
	if !rb.started {
		return
	}
	
	message := &WebSocketMessage{
		Type: WSMsgTypeNotification,
		ID:   queryID,
		Data: map[string]interface{}{
			"event_type": eventType,
			"data":       data,
			"source":     "query",
		},
		Timestamp: time.Now(),
	}
	
	select {
	case rb.queryEvents <- message:
	default:
		log.Printf("Query events channel full, dropping event")
	}
}

// processSystemEvents processes and broadcasts system events
func (rb *RealtimeBroadcaster) processSystemEvents() {
	for {
		select {
		case event := <-rb.systemEvents:
			rb.wsManager.BroadcastToSubscribers(SubTypeSystemEvents, event)
			
		case <-rb.stopChan:
			return
		}
	}
}

// processOllamaEvents processes and broadcasts Ollama events
func (rb *RealtimeBroadcaster) processOllamaEvents() {
	for {
		select {
		case event := <-rb.ollamaEvents:
			rb.wsManager.BroadcastToSubscribers(SubTypeOllamaStatus, event)
			
		case <-rb.stopChan:
			return
		}
	}
}

// processQueryEvents processes and broadcasts query events
func (rb *RealtimeBroadcaster) processQueryEvents() {
	for {
		select {
		case event := <-rb.queryEvents:
			rb.wsManager.BroadcastToSubscribers(SubTypeQueryResults, event)
			
		case <-rb.stopChan:
			return
		}
	}
}

// processMetricsEvents processes and broadcasts metrics events
func (rb *RealtimeBroadcaster) processMetricsEvents() {
	for {
		select {
		case event := <-rb.metricsEvents:
			rb.wsManager.BroadcastToSubscribers(SubTypeMetrics, event)
			
		case <-rb.stopChan:
			return
		}
	}
}

// broadcastPeriodicMetrics broadcasts system metrics periodically
func (rb *RealtimeBroadcaster) broadcastPeriodicMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rb.broadcastSystemMetrics()
			
		case <-rb.stopChan:
			return
		}
	}
}

// broadcastSystemMetrics broadcasts current system metrics
func (rb *RealtimeBroadcaster) broadcastSystemMetrics() {
	if rb.server == nil || rb.server.Metrics == nil {
		return
	}
	
	rb.server.Metrics.mu.RLock()
	metrics := map[string]interface{}{
		"queries_processed":   rb.server.Metrics.QueriesProcessed,
		"query_successes":     rb.server.Metrics.QuerySuccesses,
		"query_failures":      rb.server.Metrics.QueryFailures,
		"documents_added":     rb.server.Metrics.DocumentsAdded,
		"rag_queries_total":   rb.server.Metrics.RAGQueriesTotal,
		"rag_query_failures":  rb.server.Metrics.RAGQueryFailures,
		"websocket_connections": rb.wsManager.GetConnectionCount(),
	}
	rb.server.Metrics.mu.RUnlock()
	
	message := &WebSocketMessage{
		Type: WSMsgTypeMetrics,
		Data: map[string]interface{}{
			"system_metrics": metrics,
			"timestamp":      time.Now(),
		},
		Timestamp: time.Now(),
	}
	
	rb.wsManager.BroadcastToSubscribers(SubTypeMetrics, message)
}

// broadcastConsciousnessState broadcasts consciousness state updates
func (rb *RealtimeBroadcaster) broadcastConsciousnessState() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if rb.server.RAGSystem != nil && rb.server.RAGSystem.ContextManager != nil {
				consciousnessRuntime := rb.server.RAGSystem.ContextManager.GetConsciousnessRuntime()
				if consciousnessRuntime != nil {
					state := consciousnessRuntime.GetConsciousnessState()
					
					message := &WebSocketMessage{
						Type: WSMsgTypeConsciousness,
						Data: map[string]interface{}{
							"consciousness_state": state,
							"timestamp":           time.Now(),
						},
						Timestamp: time.Now(),
					}
					
					rb.wsManager.BroadcastToSubscribers(SubTypeConsciousness, message)
				}
			}
			
		case <-rb.stopChan:
			return
		}
	}
}

// broadcastOllamaStatus broadcasts Ollama status updates
func (rb *RealtimeBroadcaster) broadcastOllamaStatus() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			if rb.server.RAGSystem != nil {
				status := rb.server.RAGSystem.GetEmbeddedStatus()
				
				message := &WebSocketMessage{
					Type: WSMsgTypeOllamaStatus,
					Data: map[string]interface{}{
						"ollama_status": status,
						"timestamp":     time.Now(),
					},
					Timestamp: time.Now(),
				}
				
				rb.wsManager.BroadcastToSubscribers(SubTypeOllamaStatus, message)
			}
			
		case <-rb.stopChan:
			return
		}
	}
}