package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/loreum-org/cortex/internal/ai"
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
	broadcast chan *WebSocketMessage
	register  chan *WebSocketConnection
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
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Message types
const (
	// Client to Server
	WSMsgTypeQuery           = "query"
	WSMsgTypeSubscribe       = "subscribe"
	WSMsgTypeUnsubscribe     = "unsubscribe"
	WSMsgTypePing            = "ping"
	WSMsgTypeGetStatus       = "get_status"
	WSMsgTypeRestartOllama   = "restart_ollama"
	
	// Server to Client  
	WSMsgTypeResponse        = "response"
	WSMsgTypeStatus          = "status"
	WSMsgTypeError           = "error"
	WSMsgTypePong            = "pong"
	WSMsgTypeNotification    = "notification"
	WSMsgTypeMetrics         = "metrics"
	WSMsgTypeConsciousness   = "consciousness"
	WSMsgTypeOllamaStatus    = "ollama_status"
	
	// Streaming
	WSMsgTypeQueryStart      = "query_start"
	WSMsgTypeQueryProgress   = "query_progress"
	WSMsgTypeQueryChunk      = "query_chunk"
	WSMsgTypeQueryComplete   = "query_complete"
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
		Type:      WSMsgTypeStatus,
		Data: map[string]interface{}{
			"connected": true,
			"server_time": time.Now(),
			"connection_id": conn.ID,
		},
		Timestamp: time.Now(),
	}
	
	select {
	case conn.Send <- welcome:
	default:
		close(conn.Send)
		delete(wsm.connections, conn.ID)
	}
}

// unregisterConnection unregisters a WebSocket connection
func (wsm *WebSocketManager) unregisterConnection(conn *WebSocketConnection) {
	wsm.mu.Lock()
	defer wsm.mu.Unlock()
	
	if _, ok := wsm.connections[conn.ID]; ok {
		delete(wsm.connections, conn.ID)
		close(conn.Send)
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
			close(conn.Send)
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
			close(conn.Send)
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
			close(conn.Send)
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
	wsConn := &WebSocketConnection{
		ID:         fmt.Sprintf("ws_%d", time.Now().UnixNano()),
		UserID:     r.Header.Get("X-User-ID"), // Could be from auth
		Conn:       conn,
		Send:       make(chan *WebSocketMessage, 256),
		Manager:    s.WebSocketManager,
		RemoteAddr: r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		StartTime:  time.Now(),
		LastSeen:   time.Now(),
	}
	
	// Register connection
	s.WebSocketManager.register <- wsConn
	
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
		
	default:
		c.sendError(msg.ID, fmt.Sprintf("Unknown message type: %s", msg.Type))
	}
}

// handleQuery processes query messages with streaming support
func (c *WebSocketConnection) handleQuery(msg *WebSocketMessage) {
	queryData, ok := msg.Data.(map[string]interface{})
	if !ok {
		c.sendError(msg.ID, "Invalid query data format")
		return
	}
	
	query, ok := queryData["query"].(string)
	if !ok {
		c.sendError(msg.ID, "Query text is required")
		return
	}
	
	// Send query start notification
	c.Send <- &WebSocketMessage{
		Type: WSMsgTypeQueryStart,
		ID:   msg.ID,
		Data: map[string]interface{}{
			"query": query,
			"status": "processing",
		},
		Timestamp: time.Now(),
	}
	
	// Process query asynchronously with streaming
	go func() {
		startTime := time.Now()
		
		// Try streaming first if available
		err := c.processQueryStreaming(query, msg.ID)
		duration := time.Since(startTime)
		
		if err != nil {
			c.Send <- &WebSocketMessage{
				Type:      WSMsgTypeError,
				ID:        msg.ID,
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
			return
		}
		
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
	
	// Use consciousness processing if available
	if c.Manager.server.RAGSystem.ContextManager != nil {
		consciousnessRuntime := c.Manager.server.RAGSystem.ContextManager.GetConsciousnessRuntime()
		if consciousnessRuntime != nil {
			// Create a proper Query object
			queryObj := &types.Query{
				ID:        fmt.Sprintf("ws_query_%d", time.Now().UnixNano()),
				Text:      query,
				Type:      "user_query",
				Metadata:  map[string]string{"source": "websocket"},
				Timestamp: time.Now().Unix(),
			}
			
			resp, err := consciousnessRuntime.ProcessQuery(context.Background(), queryObj)
			if err == nil && resp != nil {
				return resp.Text, nil
			}
		}
	}
	
	// Fallback to direct RAG query
	return c.Manager.server.RAGSystem.Query(context.Background(), query)
}

// processQueryStreaming handles query processing with streaming response
func (c *WebSocketConnection) processQueryStreaming(query string, queryID string) error {
	if c.Manager.server.RAGSystem == nil {
		return fmt.Errorf("RAG system not available")
	}
	
	// Try to use consciousness processing with streaming
	if c.Manager.server.RAGSystem.ContextManager != nil {
		consciousnessRuntime := c.Manager.server.RAGSystem.ContextManager.GetConsciousnessRuntime()
		if consciousnessRuntime != nil {
			// Try to use the model manager's streaming capability if available
			if c.Manager.server.RAGSystem.ModelManager != nil {
				models := c.Manager.server.RAGSystem.ModelManager.ListModels()
				if len(models) > 0 {
					// Try to get the first available model that supports streaming
					for _, modelInfo := range models {
						if model, err := c.Manager.server.RAGSystem.ModelManager.GetModel(modelInfo.ID); err == nil {
							if streamingModel, ok := model.(ai.StreamingAIModel); ok {
								// Use streaming callback to send chunks
								return streamingModel.GenerateResponseStream(
									context.Background(),
									query,
									ai.GenerateOptions{
										Temperature: 0.7,
										MaxTokens:   2000,
									},
									func(chunk string) error {
										// Send each chunk via WebSocket
										c.Send <- &WebSocketMessage{
											Type: WSMsgTypeQueryChunk,
											ID:   queryID,
											Data: map[string]interface{}{
												"chunk": chunk,
											},
											Timestamp: time.Now(),
										}
										return nil
									},
								)
							}
						}
					}
				}
			}
			
			// Fallback to non-streaming consciousness processing
			queryObj := &types.Query{
				ID:        fmt.Sprintf("ws_query_%d", time.Now().UnixNano()),
				Text:      query,
				Type:      "user_query",
				Metadata:  map[string]string{"source": "websocket"},
				Timestamp: time.Now().Unix(),
			}
			
			resp, err := consciousnessRuntime.ProcessQuery(context.Background(), queryObj)
			if err == nil && resp != nil {
				// Send as a single chunk for non-streaming fallback
				c.Send <- &WebSocketMessage{
					Type: WSMsgTypeQueryChunk,
					ID:   queryID,
					Data: map[string]interface{}{
						"chunk": resp.Text,
					},
					Timestamp: time.Now(),
				}
				return nil
			}
		}
	}
	
	// Final fallback to direct RAG query (non-streaming)
	response, err := c.Manager.server.RAGSystem.Query(context.Background(), query)
	if err != nil {
		return err
	}
	
	// Send as a single chunk for RAG fallback
	c.Send <- &WebSocketMessage{
		Type: WSMsgTypeQueryChunk,
		ID:   queryID,
		Data: map[string]interface{}{
			"chunk": response,
		},
		Timestamp: time.Now(),
	}
	
	return nil
}

// handleSubscribe handles subscription requests
func (c *WebSocketConnection) handleSubscribe(msg *WebSocketMessage) {
	// Implementation for subscription management
	c.Send <- &WebSocketMessage{
		Type: WSMsgTypeStatus,
		ID:   msg.ID,
		Data: map[string]interface{}{
			"subscribed": true,
			"type":      msg.Data,
		},
		Timestamp: time.Now(),
	}
}

// handleUnsubscribe handles unsubscription requests
func (c *WebSocketConnection) handleUnsubscribe(msg *WebSocketMessage) {
	// Implementation for unsubscription management
	c.Send <- &WebSocketMessage{
		Type: WSMsgTypeStatus,
		ID:   msg.ID,
		Data: map[string]interface{}{
			"unsubscribed": true,
			"type":        msg.Data,
		},
		Timestamp: time.Now(),
	}
}

// handlePing responds to ping messages
func (c *WebSocketConnection) handlePing(msg *WebSocketMessage) {
	c.Send <- &WebSocketMessage{
		Type:      WSMsgTypePong,
		ID:        msg.ID,
		Timestamp: time.Now(),
	}
}

// handleGetStatus returns current system status
func (c *WebSocketConnection) handleGetStatus(msg *WebSocketMessage) {
	status := map[string]interface{}{
		"server_time":    time.Now(),
		"connection_id":  c.ID,
		"uptime":        time.Since(c.StartTime).String(),
		"ollama_status": c.Manager.server.RAGSystem.GetEmbeddedStatus(),
	}
	
	c.Send <- &WebSocketMessage{
		Type:      WSMsgTypeStatus,
		ID:        msg.ID,
		Data:      status,
		Timestamp: time.Now(),
	}
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
		
		c.Send <- response
	}()
}

// sendError sends an error response
func (c *WebSocketConnection) sendError(messageID, errorMsg string) {
	c.Send <- &WebSocketMessage{
		Type:      WSMsgTypeError,
		ID:        messageID,
		Error:     errorMsg,
		Timestamp: time.Now(),
	}
}