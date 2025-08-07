package api

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/pkg/types"
)

// WebSocketManager manages WebSocket connections and real-time communication
type WebSocketManager struct {
	// Connection management
	connections map[string]*WebSocketConnection
	mu          sync.RWMutex

	// Server reference for API access
	server *Server

	// Channels for broadcasting
	broadcast  chan *WebSocketMessage
	register   chan *WebSocketConnection
	unregister chan *WebSocketConnection

	// Configuration
	upgrader websocket.Upgrader
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	ID       string
	UserID   string
	Conn     *websocket.Conn
	Send     chan *WebSocketMessage
	Manager  *WebSocketManager
	LastSeen time.Time

	// Connection metadata
	RemoteAddr string
	UserAgent  string
	StartTime  time.Time

	// Conversation tracking for memory
	ConversationID string
	QueryCount     int64
	LastResponse   string
	ResponseBuffer []string

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
	
	// Safe cleanup mechanism
	closeOnce sync.Once
	closed    bool
	mu        sync.RWMutex
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Method    string                 `json:"method,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Message types
const (
	// Client to Server
	WSMsgTypeQuery         = "query"
	WSMsgTypeSubscribe     = "subscribe"
	WSMsgTypeUnsubscribe   = "unsubscribe"
	WSMsgTypePing          = "ping"
	WSMsgTypeGetStatus     = "get_status"
	WSMsgTypeRestartOllama = "restart_ollama"
	// Model management
	WSMsgTypeGetModels     = "get_models"
	WSMsgTypeGetAgents     = "get_agents"
	WSMsgTypeDownloadModel = "download_model"
	WSMsgTypeDeleteModel   = "delete_model"
	WSMsgTypeRequest       = "request"

	// Server to Client
	WSMsgTypeResponse      = "response"
	WSMsgTypeStatus        = "status"
	WSMsgTypeError         = "error"
	WSMsgTypePong          = "pong"
	WSMsgTypeNotification  = "notification"
	WSMsgTypeMetrics       = "metrics"
	WSMsgTypeConsciousness = "consciousness"
	WSMsgTypeOllamaStatus  = "ollama_status"
	// Model management responses
	WSMsgTypeModelsData    = "models_data"
	WSMsgTypeAgentsData    = "agents_data"
	WSMsgTypeModelDownload = "model_download"

	// Streaming
	WSMsgTypeQueryStart    = "query_start"
	WSMsgTypeQueryProgress = "query_progress"
	WSMsgTypeQueryChunk    = "query_chunk"
	WSMsgTypeQueryComplete = "query_complete"
)

// Subscription types
const (
	SubTypeMetrics       = "metrics"
	SubTypeConsciousness = "consciousness"
	SubTypeOllamaStatus  = "ollama_status"
	SubTypeSystemEvents  = "system_events"
	SubTypeQueryResults  = "query_results"
)

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(server *Server) *WebSocketManager {
	manager := &WebSocketManager{
		connections: make(map[string]*WebSocketConnection),
		server:      server,
		broadcast:   make(chan *WebSocketMessage, 1000),
		register:    make(chan *WebSocketConnection),
		unregister:  make(chan *WebSocketConnection),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now - should be configurable in production
				return true
			},
		},
	}

	// Start the manager goroutine
	go manager.run()

	return manager
}

// run handles the main WebSocket manager loop
func (wsm *WebSocketManager) run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case conn := <-wsm.register:
			wsm.registerConnection(conn)

		case conn := <-wsm.unregister:
			wsm.unregisterConnection(conn)

		case message := <-wsm.broadcast:
			wsm.broadcastMessage(message)

		case <-ticker.C:
			wsm.cleanupStaleConnections()
		}
	}
}

// registerConnection registers a new WebSocket connection
func (wsm *WebSocketManager) registerConnection(conn *WebSocketConnection) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	wsm.connections[conn.ID] = conn

	log.Printf("WebSocket connection registered: %s (%s)", conn.ID, conn.RemoteAddr)

	// Send welcome message
	welcome := &WebSocketMessage{
		Type: WSMsgTypeStatus,
		Data: map[string]interface{}{
			"connected":     true,
			"server_time":   time.Now(),
			"connection_id": conn.ID,
		},
		Timestamp: time.Now(),
	}

	select {
	case conn.Send <- welcome:
	default:
		// Use safe cleanup mechanism
		conn.closeOnce.Do(func() {
			conn.mu.Lock()
			conn.closed = true
			conn.mu.Unlock()
			
			if conn.cancel != nil {
				conn.cancel()
			}
			// Don't close channel immediately in registerConnection
		})
		delete(wsm.connections, conn.ID)
	}
}

// unregisterConnection unregisters a WebSocket connection
func (wsm *WebSocketManager) unregisterConnection(conn *WebSocketConnection) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	if _, ok := wsm.connections[conn.ID]; ok {
		delete(wsm.connections, conn.ID)
		
		// Safe cleanup using sync.Once
		conn.closeOnce.Do(func() {
			conn.mu.Lock()
			conn.closed = true
			conn.mu.Unlock()
			
			// Cancel context first to signal goroutines
			if conn.cancel != nil {
				conn.cancel()
			}
			
			// Grace period for goroutines to exit
			go func() {
				time.Sleep(100 * time.Millisecond)
				
				// Safe channel close with panic recovery
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic closing connection %s: %v", conn.ID, r)
					}
				}()
				
				if conn.Send != nil {
					close(conn.Send)
				}
			}()
		})

		log.Printf("WebSocket connection unregistered: %s", conn.ID)
	}
}

// broadcastMessage broadcasts a message to all connections
func (wsm *WebSocketManager) broadcastMessage(message *WebSocketMessage) {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	for id, conn := range wsm.connections {
		select {
		case conn.Send <- message:
		default:
			// Use safe cleanup for failed broadcast
			conn.closeOnce.Do(func() {
				conn.mu.Lock()
				conn.closed = true
				conn.mu.Unlock()
				
				if conn.cancel != nil {
					conn.cancel()
				}
				
				go func() {
					time.Sleep(50 * time.Millisecond)
					if conn.Send != nil {
						close(conn.Send)
					}
				}()
			})
			delete(wsm.connections, id)
		}
	}
}

// SendToConnection sends a message to a specific connection
func (wsm *WebSocketManager) SendToConnection(connID string, message *WebSocketMessage) error {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	conn, ok := wsm.connections[connID]
	if !ok {
		return fmt.Errorf("connection %s not found", connID)
	}
	
	// Check if connection is closed
	if conn.isClosed() {
		return fmt.Errorf("connection %s is closed", connID)
	}

	select {
	case conn.Send <- message:
		return nil
	default:
		return fmt.Errorf("connection %s send buffer full", connID)
	}
}

// BroadcastToSubscribers broadcasts to connections subscribed to a specific type
func (wsm *WebSocketManager) BroadcastToSubscribers(subType string, message *WebSocketMessage) {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	for id, conn := range wsm.connections {
		// Check if connection is subscribed (would need subscription tracking)
		select {
		case conn.Send <- message:
		default:
			// Use safe cleanup for failed broadcast to subscribers
			conn.closeOnce.Do(func() {
				conn.mu.Lock()
				conn.closed = true
				conn.mu.Unlock()
				
				if conn.cancel != nil {
					conn.cancel()
				}
				
				go func() {
					time.Sleep(50 * time.Millisecond)
					if conn.Send != nil {
						close(conn.Send)
					}
				}()
			})
			delete(wsm.connections, id)
		}
	}
}

// cleanupStaleConnections removes connections that haven't been seen recently
func (wsm *WebSocketManager) cleanupStaleConnections() {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)

	for id, conn := range wsm.connections {
		if conn.LastSeen.Before(cutoff) {
			log.Printf("Cleaning up stale WebSocket connection: %s", id)
			
			// Use safe cleanup for stale connections
			conn.closeOnce.Do(func() {
				conn.mu.Lock()
				conn.closed = true
				conn.mu.Unlock()
				
				if conn.cancel != nil {
					conn.cancel()
				}
				
				go func() {
					time.Sleep(100 * time.Millisecond)
					if conn.Send != nil {
						close(conn.Send)
					}
				}()
			})
			
			delete(wsm.connections, id)
		}
	}
}

// GetConnectionCount returns the number of active connections
func (wsm *WebSocketManager) GetConnectionCount() int {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()
	return len(wsm.connections)
}

// GetConnections returns information about all connections
func (wsm *WebSocketManager) GetConnections() []map[string]interface{} {
	wsm.mu.RLock()
	defer wsm.mu.RUnlock()

	connections := make([]map[string]interface{}, 0, len(wsm.connections))

	for _, conn := range wsm.connections {
		connections = append(connections, map[string]interface{}{
			"id":          conn.ID,
			"user_id":     conn.UserID,
			"remote_addr": conn.RemoteAddr,
			"start_time":  conn.StartTime,
			"last_seen":   conn.LastSeen,
			"uptime":      time.Since(conn.StartTime).String(),
		})
	}

	return connections
}

// WebSocket HTTP handler
func (s *Server) websocketHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := s.WebSocketManager.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create connection object
	ctx, cancel := context.WithCancel(context.Background())
	wsConn := &WebSocketConnection{
		ID:             fmt.Sprintf("ws_%d", time.Now().UnixNano()),
		UserID:         r.Header.Get("X-User-ID"), // Could be from auth
		Conn:           conn,
		Send:           make(chan *WebSocketMessage, 256),
		Manager:        s.WebSocketManager,
		RemoteAddr:     r.RemoteAddr,
		UserAgent:      r.UserAgent(),
		StartTime:      time.Now(),
		LastSeen:       time.Now(),
		ConversationID: "", // Will be set from Context Manager
		QueryCount:     0,
		ResponseBuffer: make([]string, 0, 10), // Keep last 10 responses for context
		ctx:            ctx,
		cancel:         cancel,
	}

	// Register connection
	s.WebSocketManager.register <- wsConn

	// Sync conversation ID with Context Manager
	wsConn.syncConversationID()

	// Start goroutines for this connection
	go wsConn.writePump()
	go wsConn.readPump()
}

// writePump pumps messages from the hub to the websocket connection
func (c *WebSocketConnection) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Debug log the message being sent
			if message.Type == WSMsgTypeAgentsData {
				log.Printf("📤 WritePump sending agents message - ID: %s, Type: %s", message.ID, message.Type)
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump pumps messages from the websocket connection to the hub
func (c *WebSocketConnection) readPump() {
	defer func() {
		c.Manager.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512 * 1024) // 512KB
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg WebSocketMessage
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		c.LastSeen = time.Now()
		c.handleMessage(&msg)
	}
}

// handleMessage processes incoming WebSocket messages
func (c *WebSocketConnection) handleMessage(msg *WebSocketMessage) {
	switch msg.Type {
	case WSMsgTypeQuery:
		c.handleQuery(msg)

	case WSMsgTypeSubscribe:
		c.handleSubscribe(msg)

	case WSMsgTypeUnsubscribe:
		c.handleUnsubscribe(msg)

	case WSMsgTypePing:
		c.handlePing(msg)

	case WSMsgTypeGetStatus:
		c.handleGetStatus(msg)

	case WSMsgTypeRestartOllama:
		c.handleRestartOllama(msg)

	case WSMsgTypeGetModels:
		c.handleGetModels(msg)

	case WSMsgTypeGetAgents:
		c.handleGetAgents(msg)

	case WSMsgTypeDownloadModel:
		c.handleDownloadModel(msg)

	case WSMsgTypeDeleteModel:
		c.handleDeleteModel(msg)

	case WSMsgTypeRequest:
		c.handleRequest(msg)

	default:
		c.sendError(msg.ID, fmt.Sprintf("Unknown message type: %s", msg.Type))
	}
}

// handleQuery processes query messages with streaming support
func (c *WebSocketConnection) handleQuery(msg *WebSocketMessage) {
	log.Printf("🔍 handleQuery: Message ID=%s, Type=%s, Data=%+v", msg.ID, msg.Type, msg.Data)
	
	queryData, ok := msg.Data.(map[string]interface{})
	if !ok {
		log.Printf("❌ handleQuery: Invalid query data format, Data type: %T", msg.Data)
		c.sendError(msg.ID, "Invalid query data format")
		return
	}

	log.Printf("🔍 handleQuery: Query data parsed successfully: %+v", queryData)

	query, ok := queryData["query"].(string)
	if !ok {
		log.Printf("❌ handleQuery: Query text not found or invalid type in data: %+v", queryData)
		c.sendError(msg.ID, "Query text is required")
		return
	}

	log.Printf("✅ handleQuery: Query extracted successfully: '%s'", query)

	// Store query in conversation memory
	c.storeConversationEvent("user_query", query, msg.ID)

	// Extract user information for profile updates
	if c.Manager.server.RAGSystem != nil && c.Manager.server.RAGSystem.UserProfileManager != nil && c.UserID != "" {
		c.Manager.server.RAGSystem.UserProfileManager.ExtractUserInfoFromText(c.UserID, query)
	}

	// Send query start notification
	c.Send <- &WebSocketMessage{
		Type: WSMsgTypeQueryStart,
		ID:   msg.ID,
		Data: map[string]interface{}{
			"query":  query,
			"status": "processing",
		},
		Timestamp: time.Now(),
	}

	// Process query asynchronously with streaming
	go func() {
		startTime := time.Now()

		// Create a context with timeout for query processing
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Channel to receive the result
		resultChan := make(chan error, 1)

		// Run query processing in a separate goroutine
		go func() {
			err := c.processQueryStreamingWithContext(ctx, query, msg.ID)
			resultChan <- err
		}()

		// Wait for either result or timeout
		var err error
		select {
		case err = <-resultChan:
			// Query completed
		case <-ctx.Done():
			err = fmt.Errorf("query processing timeout after 30 seconds")
		}

		duration := time.Since(startTime)

		if err != nil {
			// Store error in conversation memory
			c.storeConversationEvent("error", fmt.Sprintf("Query processing failed: %v", err), msg.ID)

			c.Send <- &WebSocketMessage{
				Type:      WSMsgTypeError,
				ID:        msg.ID,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
			return
		}

		// Store successful completion in conversation memory
		c.storeConversationEvent("processing_complete", fmt.Sprintf("Query processed successfully in %v", duration), msg.ID)

		// Send completion notification
		c.Send <- &WebSocketMessage{
			Type: WSMsgTypeQueryComplete,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"query":     query,
				"duration":  duration.String(),
				"timestamp": time.Now(),
			},
			Timestamp: time.Now(),
		}
	}()
}

// processQuery handles the actual query processing
func (c *WebSocketConnection) processQuery(query string) (string, error) {
	if c.Manager.server.RAGSystem == nil {
		return "", fmt.Errorf("RAG system not available")
	}

	// Use AGI consciousness processing if available
	if c.Manager.server.RAGSystem != nil {
		agiSystem := c.Manager.server.RAGSystem.GetAGISystem()
		if agiSystem != nil {
			// Create a proper Query object
			queryObj := &types.Query{
				ID:        fmt.Sprintf("ws_query_%d", time.Now().UnixNano()),
				Text:      query,
				Type:      "user_query",
				Metadata:  map[string]string{"source": "websocket"},
				Timestamp: time.Now().Unix(),
			}

			resp, err := agiSystem.ProcessQueryWithConsciousness(context.Background(), queryObj)
			if err == nil && resp != nil {
				return resp.Text, nil
			}
		}
	}

	// Fallback to direct RAG query
	return c.Manager.server.RAGSystem.Query(context.Background(), query)
}

// processQueryStreamingWithContext handles query processing with streaming response and context
func (c *WebSocketConnection) processQueryStreamingWithContext(ctx context.Context, query string, queryID string) error {
	if c.Manager.server.RAGSystem == nil {
		return fmt.Errorf("RAG system not available")
	}

	// Create a comprehensive query object
	queryObj := &types.Query{
		ID:   fmt.Sprintf("ws_query_%d", time.Now().UnixNano()),
		Text: query,
		Type: "user_query",
		Metadata: map[string]string{
			"source":        "websocket",
			"connection_id": c.ID,
			"user_id":       c.UserID,
		},
		Timestamp: time.Now().Unix(),
	}

	// Send consciousness update about new query
	c.sendConsciousnessUpdate("New user query received via WebSocket", map[string]interface{}{
		"query":            query,
		"connection_id":    c.ID,
		"processing_stage": "received",
	})

	// Skip agent registry and route directly to AGI consciousness
	log.Printf("Routing query directly to AGI consciousness: %s", query)

	// Process query through AGI consciousness with enhanced awareness
	if c.Manager.server.RAGSystem != nil {
		agiSystem := c.Manager.server.RAGSystem.GetAGISystem()
		if agiSystem != nil {
			log.Printf("Processing query through AGI consciousness runtime: %s", query)

			// Send AGI consciousness update about processing
			c.sendConsciousnessUpdate("Query being processed by AGI consciousness", map[string]interface{}{
				"query_id":         queryObj.ID,
				"processing_stage": "agi_consciousness_active",
			})

			// Directly use AGI consciousness processing first (simplified path)
			resp, err := agiSystem.ProcessQueryWithConsciousness(ctx, queryObj)
			if err == nil && resp != nil {
				log.Printf("Consciousness processed query successfully, response length: %d", len(resp.Text))
				
				// Get consciousness state for metadata
				consciousnessState := consciousnessRuntime.GetConsciousnessState()

				// Store response in conversation memory
				c.storeConversationEvent("assistant_response", resp.Text, queryID)

				c.Send <- &WebSocketMessage{
					Type: WSMsgTypeQueryChunk,
					ID:   queryID,
					Data: map[string]interface{}{
						"chunk": resp.Text,
						"metadata": map[string]interface{}{
							"source":              "consciousness_runtime",
							"processing_type":     "consciousness_processed",
							"consciousness_cycle": consciousnessState.CurrentCycle,
							"decision_state":      consciousnessState.DecisionState,
							"attention_focus":     consciousnessState.Attention.CurrentFocus,
							"energy_level":        consciousnessState.EnergyLevel,
							"response_metadata":   resp.Metadata,
							"conversation_id":     c.ConversationID,
							"query_count":         c.QueryCount,
						},
					},
					Timestamp: time.Now(),
				}

				// Send consciousness update about completion
				c.sendConsciousnessUpdate("Query processing completed", map[string]interface{}{
					"query_id":            queryObj.ID,
					"response_status":     resp.Status,
					"processing_complete": true,
				})

				return nil
			} else {
				log.Printf("Consciousness processing failed: %v", err)
			}

		} else {
			log.Printf("Consciousness runtime not available")
		}
	} else {
		log.Printf("Context manager not available")
	}

	// Fallback to enhanced RAG with memory integration when consciousness unavailable
	log.Printf("Consciousness unavailable, using enhanced RAG processing: %s", query)

	// Try to enhance query with RAG context and memory
	enhancedQuery := c.enhanceQueryWithRAGContext(query)
	response, err := c.Manager.server.RAGSystem.Query(ctx, enhancedQuery)
	if err != nil {
		return err
	}

	// Store response in conversation memory
	c.storeConversationEvent("assistant_response", response, queryID)

	// Send with enhanced metadata
	c.Send <- &WebSocketMessage{
		Type: WSMsgTypeQueryChunk,
		ID:   queryID,
		Data: map[string]interface{}{
			"chunk": response,
			"metadata": map[string]interface{}{
				"source":          "enhanced_rag",
				"processing_type": "rag_fallback",
				"enhanced_query":  enhancedQuery != query,
				"conversation_id": c.ConversationID,
				"query_count":     c.QueryCount,
			},
		},
		Timestamp: time.Now(),
	}

	return nil
}

// handleSubscribe handles subscription requests
func (c *WebSocketConnection) handleSubscribe(msg *WebSocketMessage) {
	subscriptionType, ok := msg.Data.(string)
	if !ok {
		c.sendError(msg.ID, "Invalid subscription type")
		return
	}

	// Handle different subscription types
	switch subscriptionType {
	case SubTypeConsciousness:
		// Start sending consciousness updates
		go c.streamConsciousnessUpdates()
	case SubTypeMetrics:
		// Start sending metrics updates
		go c.streamMetricsUpdates()
	case SubTypeSystemEvents:
		// Start sending system events
		go c.streamSystemEvents()
	}

	// Safely send subscription confirmation
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in handleSubscribe: %v", r)
			}
		}()

		select {
		case c.Send <- &WebSocketMessage{
			Type: WSMsgTypeStatus,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"subscribed": true,
				"type":       subscriptionType,
			},
			Timestamp: time.Now(),
		}:
		default:
			// Channel is full or closed
		}
	}()
}

// handleUnsubscribe handles unsubscription requests
func (c *WebSocketConnection) handleUnsubscribe(msg *WebSocketMessage) {
	// Implementation for unsubscription management
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in handleUnsubscribe: %v", r)
			}
		}()

		select {
		case c.Send <- &WebSocketMessage{
			Type: WSMsgTypeStatus,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"unsubscribed": true,
				"type":         msg.Data,
			},
			Timestamp: time.Now(),
		}:
		default:
			// Channel is full or closed
		}
	}()
}

// handlePing responds to ping messages
func (c *WebSocketConnection) handlePing(msg *WebSocketMessage) {
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in handlePing: %v", r)
			}
		}()

		select {
		case c.Send <- &WebSocketMessage{
			Type:      WSMsgTypePong,
			ID:        msg.ID,
			Timestamp: time.Now(),
		}:
		default:
			// Channel is full or closed
		}
	}()
}

// handleGetStatus returns current system status
func (c *WebSocketConnection) handleGetStatus(msg *WebSocketMessage) {
	status := map[string]interface{}{
		"server_time":   time.Now(),
		"connection_id": c.ID,
		"uptime":        time.Since(c.StartTime).String(),
		"ollama_status": c.Manager.server.RAGSystem.GetEmbeddedStatus(),
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in handleGetStatus: %v", r)
			}
		}()

		select {
		case c.Send <- &WebSocketMessage{
			Type:      WSMsgTypeStatus,
			ID:        msg.ID,
			Data:      status,
			Timestamp: time.Now(),
		}:
		default:
			// Channel is full or closed
		}
	}()
}

// handleRestartOllama restarts the embedded Ollama server
func (c *WebSocketConnection) handleRestartOllama(msg *WebSocketMessage) {
	if c.Manager.server.RAGSystem == nil {
		c.sendError(msg.ID, "RAG system not available")
		return
	}

	go func() {
		err := c.Manager.server.RAGSystem.RestartEmbeddedOllama(context.Background())

		response := &WebSocketMessage{
			Type:      WSMsgTypeOllamaStatus,
			ID:        msg.ID,
			Timestamp: time.Now(),
		}

		if err != nil {
			response.Error = err.Error()
		} else {
			response.Data = map[string]interface{}{
				"status":  "restarted",
				"message": "Embedded Ollama restart initiated",
			}
		}

		// Safely send response
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in handleRestartOllama: %v", r)
				}
			}()

			select {
			case c.Send <- response:
				// Message sent successfully
			default:
				// Channel is full or closed
			}
		}()
	}()
}

// handleGetModels handles requests for model information
func (c *WebSocketConnection) handleGetModels(msg *WebSocketMessage) {
	go func() {
		// Get installed models from the API handlers
		installedModels, err := c.Manager.server.getInstalledOllamaModels()
		if err != nil {
			log.Printf("Error getting installed models: %v", err)
			// Return empty list rather than error to avoid breaking the UI
			installedModels = []OllamaModel{}
		}

		// Get available models (simplified list for WebSocket)
		availableModels := []map[string]interface{}{
			{"name": "llama3.2:latest", "description": "Latest Llama 3.2 model", "size": "7.8GB", "tags": []string{"llama", "text"}},
			{"name": "llama3.2:3b", "description": "Llama 3.2 3B parameter model", "size": "2.0GB", "tags": []string{"llama", "small"}},
			{"name": "llama3.2:1b", "description": "Llama 3.2 1B parameter model", "size": "1.3GB", "tags": []string{"llama", "tiny"}},
			{"name": "cogito:latest", "description": "Cogito reasoning model", "size": "4.9GB", "tags": []string{"reasoning", "logic"}},
			{"name": "nomic-embed-text:latest", "description": "Text embedding model", "size": "274MB", "tags": []string{"embedding", "text"}},
		}

		response := &WebSocketMessage{
			Type: WSMsgTypeModelsData,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"installed": installedModels,
				"available": availableModels,
				"timestamp": time.Now(),
			},
			Timestamp: time.Now(),
		}

		// Safely send response
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in handleGetModels: %v", r)
				}
			}()

			select {
			case c.Send <- response:
				// Message sent successfully
			default:
				// Channel is full or closed
			}
		}()
	}()
}

// handleGetAgents handles requests for agent information
func (c *WebSocketConnection) handleGetAgents(msg *WebSocketMessage) {
	log.Printf("🔄 handleGetAgents called with message ID: %s", msg.ID)

	if c.Manager.server.AgentRegistry == nil {
		c.sendError(msg.ID, "Agent registry not available")
		return
	}

	go func() {
		agents := c.Manager.server.AgentRegistry.GetAllAgents()
		log.Printf("📊 Found %d agents in registry", len(agents))

		// Convert to WebSocket response format
		agentData := make([]map[string]interface{}, 0, len(agents))
		for _, agentInfo := range agents {
			agent, err := c.Manager.server.AgentRegistry.GetAgent(agentInfo.ID)
			if err != nil {
				log.Printf("Failed to get agent %s: %v", agentInfo.ID, err)
				continue // Skip agents that can't be retrieved
			}

			agentData = append(agentData, map[string]interface{}{
				"info":    agentInfo,
				"metrics": agent.GetMetrics(),
				"status":  agent.GetStatus(),
			})
		}

		response := &WebSocketMessage{
			Type: WSMsgTypeAgentsData,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"agents":    agentData,
				"timestamp": time.Now(),
			},
			Timestamp: time.Now(),
		}

		log.Printf("📤 Sending agents response with ID: %s, Type: %s, Data length: %d", response.ID, response.Type, len(agentData))

		// Safely send response
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in handleGetAgents: %v", r)
				}
			}()

			select {
			case c.Send <- response:
				// Message sent successfully
			default:
				// Channel is full or closed
			}
		}()
	}()
}

// handleDownloadModel handles model download requests
func (c *WebSocketConnection) handleDownloadModel(msg *WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		c.sendError(msg.ID, "Invalid download model data format")
		return
	}

	modelName, ok := data["model_name"].(string)
	if !ok {
		c.sendError(msg.ID, "Model name is required")
		return
	}

	go func() {
		// Send start notification
		c.Send <- &WebSocketMessage{
			Type: WSMsgTypeModelDownload,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"status":     "started",
				"model_name": modelName,
				"message":    fmt.Sprintf("Download started for model: %s", modelName),
				"timestamp":  time.Now(),
			},
			Timestamp: time.Now(),
		}

		// Use actual Ollama CLI to download the model
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		cmd := exec.CommandContext(ctx, "ollama", "pull", modelName)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeModelDownload,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"status":     "failed",
					"model_name": modelName,
					"message":    fmt.Sprintf("Failed to start download: %v", err),
					"timestamp":  time.Now(),
				},
				Timestamp: time.Now(),
			}
			return
		}

		if err := cmd.Start(); err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeModelDownload,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"status":     "failed",
					"model_name": modelName,
					"message":    fmt.Sprintf("Failed to start download: %v", err),
					"timestamp":  time.Now(),
				},
				Timestamp: time.Now(),
			}
			return
		}

		// Read download progress from ollama output
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			// Parse ollama progress output (basic parsing)
			if strings.Contains(line, "pulling") || strings.Contains(line, "downloading") {
				c.Send <- &WebSocketMessage{
					Type: WSMsgTypeModelDownload,
					ID:   msg.ID,
					Data: map[string]interface{}{
						"status":     "downloading",
						"model_name": modelName,
						"message":    line,
						"timestamp":  time.Now(),
					},
					Timestamp: time.Now(),
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeModelDownload,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"status":     "failed",
					"model_name": modelName,
					"message":    fmt.Sprintf("Download failed: %v", err),
					"timestamp":  time.Now(),
				},
				Timestamp: time.Now(),
			}
			return
		}

		// Send completion
		c.Send <- &WebSocketMessage{
			Type: WSMsgTypeModelDownload,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"status":     "completed",
				"model_name": modelName,
				"message":    fmt.Sprintf("Model %s downloaded successfully", modelName),
				"timestamp":  time.Now(),
			},
			Timestamp: time.Now(),
		}
	}()
}

// handleDeleteModel handles model deletion requests
func (c *WebSocketConnection) handleDeleteModel(msg *WebSocketMessage) {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		c.sendError(msg.ID, "Invalid delete model data format")
		return
	}

	modelName, ok := data["model_name"].(string)
	if !ok {
		c.sendError(msg.ID, "Model name is required")
		return
	}

	go func() {
		// Use actual Ollama CLI to delete the model
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "ollama", "rm", modelName)
		output, err := cmd.CombinedOutput()

		if err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeModelsData,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"status":     "failed",
					"model_name": modelName,
					"message":    fmt.Sprintf("Failed to delete model %s: %v, output: %s", modelName, err, string(output)),
					"timestamp":  time.Now(),
				},
				Timestamp: time.Now(),
			}
			return
		}

		// Send success response
		response := &WebSocketMessage{
			Type: WSMsgTypeModelsData,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"status":     "deleted",
				"model_name": modelName,
				"message":    fmt.Sprintf("Model %s deleted successfully", modelName),
				"timestamp":  time.Now(),
			},
			Timestamp: time.Now(),
		}

		// Safely send response
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in handleDeleteModel: %v", r)
				}
			}()

			select {
			case c.Send <- response:
				// Message sent successfully
			default:
				// Channel is full or closed
			}
		}()
	}()
}

// sendError sends an error response
func (c *WebSocketConnection) sendError(messageID, errorMsg string) {
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in sendError: %v", r)
			}
		}()

		select {
		case c.Send <- &WebSocketMessage{
			Type:      WSMsgTypeError,
			ID:        messageID,
			Error:     errorMsg,
			Timestamp: time.Now(),
		}:
		default:
			// Channel is full or closed
		}
	}()
}

// sendConsciousnessUpdate sends consciousness updates via WebSocket
func (c *WebSocketConnection) sendConsciousnessUpdate(message string, data map[string]interface{}) {
	// Send consciousness update to client if they're subscribed
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in sendConsciousnessUpdate: %v", r)
			}
		}()

		select {
		case c.Send <- &WebSocketMessage{
			Type: WSMsgTypeConsciousness,
			Data: map[string]interface{}{
				"message":       message,
				"details":       data,
				"timestamp":     time.Now(),
				"connection_id": c.ID,
			},
			Timestamp: time.Now(),
		}:
		default:
			// Channel is full or closed
		}
	}()
}

// enhanceQueryWithConsciousness enhances a query with consciousness context
func (c *WebSocketConnection) enhanceQueryWithConsciousness(query string, state *rag.ConsciousnessState) string {
	if state == nil {
		return query
	}

	// Build enhanced query with consciousness context
	enhanced := fmt.Sprintf(`[Consciousness Context - Cycle: %d, Focus: %s, Energy: %.2f, State: %s]

Current attention: %s
Active thoughts: %v
Energy level: %.2f
Decision state: %s

User query: %s

Please respond with awareness of the current consciousness state and provide a thoughtful, contextual response.`,
		state.CurrentCycle,
		state.Attention.CurrentFocus,
		state.EnergyLevel,
		state.DecisionState,
		state.Attention.CurrentFocus,
		c.extractThoughtContents(state.ActiveThoughts),
		state.EnergyLevel,
		state.DecisionState,
		query)

	return enhanced
}

// enhanceQueryWithRAGContext enhances a query with RAG and memory context
func (c *WebSocketConnection) enhanceQueryWithRAGContext(query string) string {
	if c.Manager.server.RAGSystem == nil {
		return query
	}

	// First try conversation memory enhancement
	conversationEnhanced := c.enhanceWithConversationMemory(query)
	if conversationEnhanced != query {
		return conversationEnhanced
	}

	// Fallback to general RAG context
	contextQuery := fmt.Sprintf("Recent conversation context for user query: %s", query)

	// Get relevant documents from RAG system
	context, err := c.Manager.server.RAGSystem.Query(context.Background(), contextQuery)
	if err != nil || context == "" {
		return query
	}

	// Build enhanced query with memory context
	enhanced := fmt.Sprintf(`[Memory Context]
%s

[Current Query]
%s

Please respond considering the above context and provide a coherent, contextual response.`,
		context, query)

	return enhanced
}

// extractThoughtContents extracts content from thoughts for display
func (c *WebSocketConnection) extractThoughtContents(thoughts []rag.Thought) []string {
	if len(thoughts) == 0 {
		return []string{"No active thoughts"}
	}

	contents := make([]string, len(thoughts))
	for i, thought := range thoughts {
		contents[i] = thought.Content
	}
	return contents
}

// extractAGIThoughtContents extracts content from AGI thoughts for display
func (c *WebSocketConnection) extractAGIThoughtContents(thoughts []rag.Thought) []string {
	if len(thoughts) == 0 {
		return []string{"No active thoughts"}
	}

	contents := make([]string, len(thoughts))
	for i, thought := range thoughts {
		contents[i] = thought.Content
	}
	return contents
}

// extractAGIGoalDescriptions extracts goal descriptions from AGI goals
func (c *WebSocketConnection) extractAGIGoalDescriptions(goals []rag.Goal) []string {
	if len(goals) == 0 {
		return []string{"No active goals"}
	}

	descriptions := make([]string, len(goals))
	for i, goal := range goals {
		descriptions[i] = goal.Description
	}
	return descriptions
}

// calculateDomainExpertise calculates average domain expertise
func (c *WebSocketConnection) calculateDomainExpertise(domains map[string]*rag.Domain) float64 {
	if len(domains) == 0 {
		return 0.0
	}

	var totalExpertise float64
	for _, domain := range domains {
		if domain != nil {
			totalExpertise += domain.Expertise
		}
	}

	return totalExpertise / float64(len(domains))
}

// storeConversationEvent stores conversation events in RAG system for memory
func (c *WebSocketConnection) storeConversationEvent(eventType, content, messageID string) {
	if c.Manager.server.RAGSystem == nil {
		return
	}

	// Create conversation document for RAG system
	conversationDoc := fmt.Sprintf(`
Conversation Event - %s
Time: %s
Connection: %s
User: %s
Message ID: %s
Event Type: %s
Content: %s
Conversation ID: %s
Query Count: %d
Recent Context: %v
`,
		time.Now().Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
		c.ID,
		c.UserID,
		messageID,
		eventType,
		content,
		c.ConversationID,
		c.QueryCount,
		c.ResponseBuffer,
	)

	// Store in RAG system for future memory recall
	metadata := map[string]interface{}{
		"conversation_id": c.ConversationID,
		"event_type":      eventType,
		"connection_id":   c.ID,
		"user_id":         c.UserID,
		"message_id":      messageID,
		"timestamp":       time.Now().Unix(),
	}
	if err := c.Manager.server.RAGSystem.AddDocument(context.Background(), conversationDoc, metadata); err != nil {
		log.Printf("Failed to store conversation event in RAG: %v", err)
	}

	// Also store in Context Manager activity buffer for immediate access
	if c.Manager.server.RAGSystem.ContextManager != nil {
		c.Manager.server.RAGSystem.ContextManager.TrackAction(
			context.Background(),
			eventType,
			content,
			map[string]interface{}{
				"message_id":    messageID,
				"user_id":       c.UserID,
				"connection_id": c.ID,
			},
		)
	}

	// Update response buffer for context
	if eventType == "assistant_response" || eventType == "user_query" {
		c.ResponseBuffer = append(c.ResponseBuffer, fmt.Sprintf("[%s] %s", eventType, content))
		if len(c.ResponseBuffer) > 10 {
			c.ResponseBuffer = c.ResponseBuffer[1:] // Keep only last 10
		}
	}

	// Increment query count for user queries
	if eventType == "user_query" {
		c.QueryCount++
	}
}

// getConversationContext retrieves recent conversation context from memory
func (c *WebSocketConnection) getConversationContext() string {
	if c.Manager.server.RAGSystem == nil || len(c.ResponseBuffer) == 0 {
		return ""
	}

	// Build context from recent conversation
	context := fmt.Sprintf("Recent conversation context for %s:\n", c.ConversationID)
	for i, response := range c.ResponseBuffer {
		context += fmt.Sprintf("%d. %s\n", i+1, response)
	}

	return context
}

// enhanceWithConversationMemory enhances query with conversation memory
func (c *WebSocketConnection) enhanceWithConversationMemory(query string) string {
	if c.Manager.server.RAGSystem == nil || c.Manager.server.RAGSystem.ContextManager == nil {
		return query
	}

	// Get conversation history from Context Manager
	contextManager := c.Manager.server.RAGSystem.ContextManager

	// Get recent conversation history (up to 10 events)
	history, err := contextManager.GetRecentConversationHistory(context.Background(), 10)
	if err != nil {
		log.Printf("Failed to retrieve conversation history: %v", err)
		return query
	}

	// Format conversation history
	var conversationHistory string
	if len(history) > 0 {
		conversationHistory = "Recent conversation:\n"
		for i, event := range history {
			if i >= 5 { // Limit to last 5 events to keep prompt manageable
				break
			}
			conversationHistory += fmt.Sprintf("- %s: %s\n", event.Type, event.Content)
		}
	} else {
		conversationHistory = "No recent conversation history found."
	}

	// Get user information from conversation context
	userInfo := c.extractUserInformation(history)

	// Get local context
	localContext := c.getConversationContext()

	// Build enhanced query
	enhanced := fmt.Sprintf(`[Conversation Memory]
Conversation ID: %s
Query Count: %d
User Information: %s
Local Context:
%s

%s

[Current Query]
%s

Please respond considering the conversation history and maintain contextual continuity. Remember the user's name and any personal details mentioned.`,
		c.ConversationID,
		c.QueryCount,
		userInfo,
		localContext,
		conversationHistory,
		query)

	return enhanced
}

// streamConsciousnessUpdates streams real-time consciousness state updates
func (c *WebSocketConnection) streamConsciousnessUpdates() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in streamConsciousnessUpdates: %v", r)
		}
	}()

	ticker := time.NewTicker(2 * time.Second) // Update every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Context cancelled, stop streaming
			log.Printf("streamConsciousnessUpdates stopped for connection %s: context cancelled", c.ID)
			return
		case <-ticker.C:
			// Check if connection is still alive first
			if time.Since(c.LastSeen) > 10*time.Minute {
				return
			}

			// Check if connection is still registered with manager
			c.Manager.mu.RLock()
			_, exists := c.Manager.connections[c.ID]
			c.Manager.mu.RUnlock()
			if !exists {
				return
			}

			if c.Manager.server.RAGSystem == nil {
				continue
			}

			agiSystem := c.Manager.server.RAGSystem.GetAGISystem()
			if agiSystem == nil {
				continue
			}

			// Get current AGI consciousness state
			stateMap := agiSystem.GetConsciousnessState()
			if stateMap == nil {
				continue
			}

			// Get AGI state and metrics
			agiState := agiSystem.GetCurrentAGIState()
			if agiState == nil {
				continue
			}
			metrics := map[string]interface{}{
				"cycle_count":        stateMap["cycle_count"],
				"intelligence_level": agiState.IntelligenceLevel,
				"is_running":         stateMap["is_running"],
				"avg_cycle_time":     stateMap["avg_cycle_time"],
			}

			// Create the AGI consciousness message
			msg := &WebSocketMessage{
				Type: WSMsgTypeConsciousness,
				Data: map[string]interface{}{
					"agi_consciousness_state": map[string]interface{}{
						"cycle_count":        stateMap["current_cycle"],
						"energy_level":       stateMap["energy_level"],
						"focus_level":        stateMap["focus_level"],
						"alertness_level":    stateMap["alertness_level"],
						"decision_state":     stateMap["decision_state"],
						"processing_load":    stateMap["processing_load"],
						"intelligence_level": stateMap["intelligence_level"],
						"attention":          stateMap["attention"],
						"active_thoughts": c.extractAGIThoughtContents(agiState.ActiveThoughts),
						"active_goals":    c.extractAGIGoalDescriptions(agiState.Goals),
						"emotions":        agiState.Emotions,
						"domains":         stateMap["domains"],
					},
					"metrics": metrics,
					"agi_awareness": map[string]interface{}{
						"is_conscious":           agiState.CurrentCycle > 0,
						"intelligence_level":     agiState.IntelligenceLevel,
						"processing_capacity":    1.0 - agiState.ProcessingLoad,
						"domain_expertise":      c.calculateDomainExpertise(agiState.KnowledgeDomains),
						"learning_progress":     agiState.LearningState.LearningProgress,
					},
					"conversation_context": map[string]interface{}{
						"conversation_id":  c.ConversationID,
						"query_count":      c.QueryCount,
						"connection_age":   time.Since(c.StartTime),
						"last_interaction": time.Since(c.LastSeen),
					},
				},
				Timestamp: time.Now(),
			}

			// Safely send consciousness update with error recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic sending consciousness update: %v", r)
					}
				}()

				select {
				case c.Send <- msg:
					// Message sent successfully
				default:
					// Channel is full or closed, stop streaming
					return
				}
			}()
		}
	}
}

// streamMetricsUpdates streams system metrics updates
func (c *WebSocketConnection) streamMetricsUpdates() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in streamMetricsUpdates: %v", r)
		}
	}()

	ticker := time.NewTicker(5 * time.Second) // Update every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Context cancelled, stop streaming
			log.Printf("streamMetricsUpdates stopped for connection %s: context cancelled", c.ID)
			return
		case <-ticker.C:
			// Check if connection is still alive first
			if time.Since(c.LastSeen) > 10*time.Minute {
				return
			}

			// Check if connection is still registered with manager
			c.Manager.mu.RLock()
			_, exists := c.Manager.connections[c.ID]
			c.Manager.mu.RUnlock()
			if !exists {
				return
			}

			// Get system metrics
			metrics := map[string]interface{}{
				"connections": c.Manager.GetConnectionCount(),
				"uptime":      time.Since(c.StartTime),
			}

			// Add RAG system metrics if available
			if c.Manager.server.RAGSystem != nil {
				ragMetrics := map[string]interface{}{
					"vector_db_active":       c.Manager.server.RAGSystem.VectorDB != nil,
					"model_manager_active":   c.Manager.server.RAGSystem.ModelManager != nil,
					"context_manager_active": c.Manager.server.RAGSystem.ContextManager != nil,
				}

				// Add embedded manager status if available
				if c.Manager.server.RAGSystem.EmbeddedManager != nil {
					ragMetrics["embedded_status"] = c.Manager.server.RAGSystem.GetEmbeddedStatus()
				}

				metrics["rag_system"] = ragMetrics
			}

			// Add agent registry metrics if available
			if c.Manager.server.AgentRegistry != nil {
				metrics["agent_registry"] = c.Manager.server.AgentRegistry.GetRegistryStats()
			}

			// Safely send message with error recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic sending metrics: %v", r)
					}
				}()

				select {
				case c.Send <- &WebSocketMessage{
					Type:      WSMsgTypeMetrics,
					Data:      metrics,
					Timestamp: time.Now(),
				}:
					// Message sent successfully
				default:
					// Channel is full or closed, stop streaming
					return
				}
			}()
		}
	}
}

// streamSystemEvents streams system events
func (c *WebSocketConnection) streamSystemEvents() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in streamSystemEvents: %v", r)
		}
	}()

	// This would connect to a system event bus if available
	// For now, we'll just send periodic system status updates
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Context cancelled, stop streaming
			log.Printf("streamSystemEvents stopped for connection %s: context cancelled", c.ID)
			return
		case <-ticker.C:
			// Check if connection is still alive first
			if time.Since(c.LastSeen) > 10*time.Minute {
				return
			}

			// Check if connection is still registered with manager
			c.Manager.mu.RLock()
			_, exists := c.Manager.connections[c.ID]
			c.Manager.mu.RUnlock()
			if !exists {
				return
			}

			systemStatus := map[string]interface{}{
				"status":    "operational",
				"timestamp": time.Now(),
				"components": map[string]interface{}{
					"rag_system":        c.Manager.server.RAGSystem != nil,
					"consciousness":     c.Manager.server.RAGSystem != nil && c.Manager.server.RAGSystem.ContextManager != nil,
					"agent_registry":    c.Manager.server.AgentRegistry != nil,
					"websocket_manager": true,
				},
			}

			// Safely send system event with error recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Recovered from panic sending system event: %v", r)
					}
				}()

				// Use safe send method
				systemMsg := &WebSocketMessage{
					Type: WSMsgTypeNotification,
					Data: map[string]interface{}{
						"event_type":    "system_status",
						"system_status": systemStatus,
					},
					Timestamp: time.Now(),
				}
				
				if !c.safeSend(systemMsg) {
					// Connection closed or failed, stop streaming
					return
				}
			}()
		}
	}
}

// Helper methods for consciousness analysis

func (c *WebSocketConnection) extractGoalDescriptions(goals []rag.ConsciousGoal) []string {
	descriptions := make([]string, len(goals))
	for i, goal := range goals {
		descriptions[i] = fmt.Sprintf("%s (%.1f%% complete, priority: %.2f)",
			goal.Description, goal.Progress*100, goal.Priority)
	}
	return descriptions
}

func (c *WebSocketConnection) extractContextKeys(context map[string]interface{}) []string {
	keys := make([]string, 0, len(context))
	for key := range context {
		keys = append(keys, key)
	}
	return keys
}

func (c *WebSocketConnection) calculateAttentionCoherence(attention *rag.AttentionState) float64 {
	if attention == nil {
		return 0.0
	}

	// Calculate coherence based on focus strength and distraction level
	coherence := attention.FocusStrength * (1.0 - attention.DistractionLevel)
	if coherence < 0 {
		coherence = 0
	}
	return coherence
}

func (c *WebSocketConnection) calculateEmotionalStability(emotions map[string]float64) float64 {
	if len(emotions) == 0 {
		return 1.0 // Neutral is stable
	}

	// Calculate variance in emotional states as inverse of stability
	var sum, variance float64
	for _, value := range emotions {
		sum += value
	}
	mean := sum / float64(len(emotions))

	for _, value := range emotions {
		variance += (value - mean) * (value - mean)
	}
	variance /= float64(len(emotions))

	// Convert variance to stability (lower variance = higher stability)
	stability := 1.0 / (1.0 + variance)
	return stability
}

func (c *WebSocketConnection) calculateMemoryIntegration(memory *rag.WorkingMemory) float64 {
	if memory == nil {
		return 0.0
	}

	// Calculate integration based on active context, beliefs, and recent inputs
	contextScore := float64(len(memory.CurrentContext)) / 10.0 // Normalized to 10 items
	beliefScore := float64(len(memory.ActiveBeliefs)) / 5.0    // Normalized to 5 beliefs
	inputScore := float64(len(memory.RecentInputs)) / 10.0     // Normalized to 10 inputs

	// Average the scores and cap at 1.0
	integration := (contextScore + beliefScore + inputScore) / 3.0
	if integration > 1.0 {
		integration = 1.0
	}

	return integration
}

// syncConversationID synchronizes the WebSocket conversation ID with the Context Manager and implements session continuity
func (c *WebSocketConnection) syncConversationID() {
	if c.Manager.server.RAGSystem == nil || c.Manager.server.RAGSystem.ContextManager == nil {
		// Fallback to generating our own ID if Context Manager not available
		c.ConversationID = fmt.Sprintf("conv_%s_%d", c.ID, time.Now().Unix())
		log.Printf("WebSocket using fallback conversation ID: %s", c.ConversationID)
		return
	}

	contextManager := c.Manager.server.RAGSystem.ContextManager
	ragSystem := c.Manager.server.RAGSystem

	// Since there's only one user per node, use the node ID as the user ID
	nodeUserID := fmt.Sprintf("node_user_%s", contextManager.GetNodeID())

	// Get or create the single user profile for this node
	if ragSystem.UserProfileManager != nil {
		userProfile := ragSystem.UserProfileManager.GetOrCreateUserProfile(nodeUserID)
		c.UserID = nodeUserID

		// Use user-specific persistent conversation ID for continuity
		c.ConversationID = contextManager.GetUserPersistentConversationID(nodeUserID)
		log.Printf("WebSocket using node user profile: %s, conversation: %s",
			userProfile.Name, c.ConversationID)

		// Add this conversation to user's profile
		ragSystem.UserProfileManager.AddConversationToProfile(nodeUserID, c.ConversationID)
		return
	}

	// Get the conversation ID from Context Manager for new sessions
	conversationID := contextManager.GetConversationID()

	if conversationID != "" {
		c.ConversationID = conversationID
		log.Printf("WebSocket synced with Context Manager conversation ID: %s", c.ConversationID)
	} else {
		// Context Manager doesn't have a conversation ID yet, create one using its format
		nodeID := contextManager.GetNodeID()
		c.ConversationID = fmt.Sprintf("conv_%s_%d", nodeID, time.Now().Unix())
		log.Printf("WebSocket created new conversation ID: %s", c.ConversationID)
	}

	// Set user ID for fallback case (when UserProfileManager not available)
	if c.UserID == "" {
		c.UserID = fmt.Sprintf("node_user_%s", contextManager.GetNodeID())
	}
}

// extractUserInformation extracts user information from conversation history
func (c *WebSocketConnection) extractUserInformation(history []rag.ActivityEvent) string {
	userInfo := make([]string, 0)

	// Look for user introductions or names in conversation
	for _, event := range history {
		content := strings.ToLower(event.Content)

		// Look for common patterns where users introduce themselves
		patterns := []string{
			"my name is",
			"i'm ",
			"i am ",
			"call me ",
			"name's ",
		}

		for _, pattern := range patterns {
			if idx := strings.Index(content, pattern); idx != -1 {
				// Extract the name (simple heuristic)
				nameStart := idx + len(pattern)
				words := strings.Fields(content[nameStart:])
				if len(words) > 0 {
					name := words[0]
					// Clean up common punctuation
					name = strings.Trim(name, ".,!?;:")
					if len(name) > 1 && name != "a" && name != "the" {
						userInfo = append(userInfo, fmt.Sprintf("User's name: %s", name))
					}
				}
			}
		}

		// Look for other personal information
		if strings.Contains(content, "i like") || strings.Contains(content, "i love") {
			userInfo = append(userInfo, fmt.Sprintf("Preference: %s", event.Content))
		}

		if strings.Contains(content, "i work") || strings.Contains(content, "my job") {
			userInfo = append(userInfo, fmt.Sprintf("Work context: %s", event.Content))
		}
	}

	if len(userInfo) == 0 {
		return "No specific user information identified yet."
	}

	return strings.Join(userInfo, "; ")
}

// isClosed safely checks if the connection is closed
func (c *WebSocketConnection) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// safeSend safely sends a message to the connection with proper error handling
func (c *WebSocketConnection) safeSend(msg *WebSocketMessage) bool {
	if c.isClosed() {
		return false
	}
	
	select {
	case c.Send <- msg:
		return true
	default:
		// Channel full or closed, trigger cleanup
		c.closeOnce.Do(func() {
			c.mu.Lock()
			c.closed = true
			c.mu.Unlock()
			
			if c.cancel != nil {
				c.cancel()
			}
		})
		return false
	}
}

// Note: Removed complex user fingerprinting since there's only one user per node

// handleRequest processes generic request messages with method routing
func (c *WebSocketConnection) handleRequest(msg *WebSocketMessage) {
	// Extract method from message data
	requestData, ok := msg.Data.(map[string]interface{})
	if !ok {
		// Check if method is in the message itself
		if msg.Method != "" {
			c.routeRequestByMethod(msg, msg.Method)
			return
		}
		c.sendError(msg.ID, "Invalid request data format")
		return
	}

	method, ok := requestData["method"].(string)
	if !ok {
		// Check for method in message root
		if msg.Method != "" {
			c.routeRequestByMethod(msg, msg.Method)
			return
		}
		c.sendError(msg.ID, "Request method is required")
		return
	}

	c.routeRequestByMethod(msg, method)
}

// routeRequestByMethod routes request messages based on method name
func (c *WebSocketConnection) routeRequestByMethod(msg *WebSocketMessage, method string) {
	switch method {
	case "getModels":
		c.handleGetModels(msg)
	case "getAgents":
		c.handleGetAgents(msg)
	case "downloadModel":
		c.handleDownloadModel(msg)
	case "deleteModel":
		c.handleDeleteModel(msg)
	case "triggerCodeAnalysis":
		c.handleTriggerCodeAnalysis(msg)
	case "getCodeQuality":
		c.handleGetCodeQuality(msg)
	case "getCodeMetrics":
		c.handleGetCodeMetrics(msg)
	default:
		c.sendError(msg.ID, fmt.Sprintf("Unknown request method: %s", method))
	}
}

// handleTriggerCodeAnalysis handles code analysis trigger requests
func (c *WebSocketConnection) handleTriggerCodeAnalysis(msg *WebSocketMessage) {
	// Route to agent management functionality
	if c.Manager.server.AgentRegistry == nil {
		c.sendError(msg.ID, "Agent registry not available")
		return
	}

	// Extract analysis parameters
	requestData, ok := msg.Data.(map[string]interface{})
	if !ok {
		requestData = make(map[string]interface{})
	}

	analysisType, _ := requestData["analysis_type"].(string)
	if analysisType == "" {
		analysisType = "full"
	}

	targetPath, _ := requestData["target_path"].(string)
	if targetPath == "" {
		targetPath = "."
	}

	// Call the agent management logic (similar to HTTP handler)
	go func() {
		agent, err := c.Manager.server.AgentRegistry.GetAgent("code-reflection-agent")
		if err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeResponse,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"error": fmt.Sprintf("Code reflection agent not found: %v", err),
				},
				Timestamp: time.Now(),
			}
			return
		}

		// Create query for the agent
		ctx := context.Background()
		query := &types.Query{
			ID:   fmt.Sprintf("analysis_%d", time.Now().UnixNano()),
			Text: fmt.Sprintf("trigger %s analysis for %s", analysisType, targetPath),
			Type: "trigger_analysis",
			Metadata: map[string]string{
				"analysis_type": analysisType,
				"target_path":   targetPath,
			},
		}

		_, err = agent.Process(ctx, query)
		if err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeResponse,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"error": fmt.Sprintf("Failed to trigger analysis: %v", err),
				},
				Timestamp: time.Now(),
			}
			return
		}

		c.Send <- &WebSocketMessage{
			Type: WSMsgTypeResponse,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"status":        "triggered",
				"analysis_type": analysisType,
				"target_path":   targetPath,
				"triggered_at":  time.Now(),
			},
			Timestamp: time.Now(),
		}
	}()
}

// handleGetCodeQuality handles code quality issues requests
func (c *WebSocketConnection) handleGetCodeQuality(msg *WebSocketMessage) {
	if c.Manager.server.AgentRegistry == nil {
		c.sendError(msg.ID, "Agent registry not available")
		return
	}

	go func() {
		agent, err := c.Manager.server.AgentRegistry.GetAgent("code-reflection-agent")
		if err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeResponse,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"error": fmt.Sprintf("Code reflection agent not found: %v", err),
				},
				Timestamp: time.Now(),
			}
			return
		}

		ctx := context.Background()
		query := &types.Query{
			ID:   fmt.Sprintf("get_issues_%d", time.Now().UnixNano()),
			Text: "get code issues",
			Type: "get_issues",
		}

		response, err := agent.Process(ctx, query)
		if err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeResponse,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"error": fmt.Sprintf("Failed to get issues: %v", err),
				},
				Timestamp: time.Now(),
			}
			return
		}

		c.Send <- &WebSocketMessage{
			Type: WSMsgTypeResponse,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"issues": map[string]interface{}{
					"agent_response": response.Text,
					"retrieved_at":   time.Now(),
					"status":         response.Status,
				},
			},
			Timestamp: time.Now(),
		}
	}()
}

// handleGetCodeMetrics handles code quality metrics requests
func (c *WebSocketConnection) handleGetCodeMetrics(msg *WebSocketMessage) {
	if c.Manager.server.AgentRegistry == nil {
		c.sendError(msg.ID, "Agent registry not available")
		return
	}

	go func() {
		agent, err := c.Manager.server.AgentRegistry.GetAgent("code-reflection-agent")
		if err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeResponse,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"error": fmt.Sprintf("Code reflection agent not found: %v", err),
				},
				Timestamp: time.Now(),
			}
			return
		}

		ctx := context.Background()
		query := &types.Query{
			ID:   fmt.Sprintf("get_metrics_%d", time.Now().UnixNano()),
			Text: "get code quality metrics",
			Type: "get_metrics",
		}

		response, err := agent.Process(ctx, query)
		if err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeResponse,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"error": fmt.Sprintf("Failed to get metrics: %v", err),
				},
				Timestamp: time.Now(),
			}
			return
		}

		c.Send <- &WebSocketMessage{
			Type: WSMsgTypeResponse,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"metrics": map[string]interface{}{
					"agent_response": response.Text,
					"agent_status":   agent.GetStatus(),
					"agent_metrics":  agent.GetMetrics(),
					"retrieved_at":   time.Now(),
					"status":         response.Status,
				},
			},
			Timestamp: time.Now(),
		}
	}()
}
