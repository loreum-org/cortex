package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketEventBridge bridges WebSocket connections and the event system
type WebSocketEventBridge struct {
	eventRouter    *EventRouter
	connections    map[string]*WebSocketConnection
	connectionsMu  sync.RWMutex
	messageHandler MessageHandler

	// Configuration
	config *BridgeConfig
}

// WebSocketConnection represents a WebSocket connection with event integration
type WebSocketConnection struct {
	ID           string
	UserID       string
	Conn         *websocket.Conn
	SendChan     chan WebSocketMessage
	EventChan    chan Event
	Context      context.Context
	Cancel       context.CancelFunc
	Metadata     map[string]interface{}
	CreatedAt    time.Time
	LastActivity time.Time

	// Event subscription
	Subscription *SubscriptionManager

	// Connection state
	Active bool
	mu     sync.RWMutex
}

// WebSocketMessage represents a WebSocket message with event integration
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Method    string                 `json:"method,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`

	// Event correlation
	CorrelationID string `json:"correlation_id,omitempty"`
	EventType     string `json:"event_type,omitempty"`
}

// MessageHandler handles different types of WebSocket messages
type MessageHandler interface {
	HandleMessage(ctx context.Context, conn *WebSocketConnection, msg WebSocketMessage) error
	GetSupportedMessageTypes() []string
}

// BridgeConfig configures the WebSocket event bridge
type BridgeConfig struct {
	// Connection limits
	MaxConnections int
	MaxMessageSize int64

	// Timeouts
	WriteTimeout time.Duration
	ReadTimeout  time.Duration
	PongTimeout  time.Duration
	PingInterval time.Duration

	// Event processing
	EventChannelBuffer   int
	MessageChannelBuffer int

	// Subscriptions
	DefaultEventFilters   map[string]SubscriptionFilter
	AutoSubscribeToEvents []string
}

// DefaultBridgeConfig returns a default bridge configuration
func DefaultBridgeConfig() *BridgeConfig {
	return &BridgeConfig{
		MaxConnections:       1000,
		MaxMessageSize:       32768, // 32KB
		WriteTimeout:         10 * time.Second,
		ReadTimeout:          60 * time.Second,
		PongTimeout:          60 * time.Second,
		PingInterval:         54 * time.Second,
		EventChannelBuffer:   100,
		MessageChannelBuffer: 100,
		DefaultEventFilters: map[string]SubscriptionFilter{
			"system": {
				Categories: []EventCategory{CategorySystem, CategoryAPI},
			},
			"metrics": {
				EventTypes: []string{EventTypeMetricsUpdated},
			},
			"consciousness": {
				EventTypes: []string{EventTypeConsciousnessUpdated},
			},
		},
		AutoSubscribeToEvents: []string{
			EventTypeMetricsUpdated,
			EventTypeConsciousnessUpdated,
			EventTypeComponentStarted,
			EventTypeComponentStopped,
		},
	}
}

// NewWebSocketEventBridge creates a new WebSocket event bridge
func NewWebSocketEventBridge(eventRouter *EventRouter, config *BridgeConfig) *WebSocketEventBridge {
	if config == nil {
		config = DefaultBridgeConfig()
	}

	bridge := &WebSocketEventBridge{
		eventRouter: eventRouter,
		connections: make(map[string]*WebSocketConnection),
		config:      config,
	}

	// Set default message handler
	bridge.messageHandler = &DefaultMessageHandler{bridge: bridge}

	return bridge
}

// SetMessageHandler sets a custom message handler
func (b *WebSocketEventBridge) SetMessageHandler(handler MessageHandler) {
	b.messageHandler = handler
}

// HandleWebSocketConnection handles a new WebSocket connection
func (b *WebSocketEventBridge) HandleWebSocketConnection(conn *websocket.Conn, userID string) (*WebSocketConnection, error) {
	// Check connection limits
	b.connectionsMu.RLock()
	if len(b.connections) >= b.config.MaxConnections {
		b.connectionsMu.RUnlock()
		return nil, fmt.Errorf("maximum connections exceeded")
	}
	b.connectionsMu.RUnlock()

	// Create connection context
	ctx, cancel := context.WithCancel(context.Background())
	connectionID := fmt.Sprintf("ws_%d_%s", time.Now().UnixNano(), randomString(8))

	// Create WebSocket connection wrapper
	wsConn := &WebSocketConnection{
		ID:           connectionID,
		UserID:       userID,
		Conn:         conn,
		SendChan:     make(chan WebSocketMessage, b.config.MessageChannelBuffer),
		EventChan:    make(chan Event, b.config.EventChannelBuffer),
		Context:      ctx,
		Cancel:       cancel,
		Metadata:     make(map[string]interface{}),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Active:       true,
	}

	// Store connection
	b.connectionsMu.Lock()
	b.connections[connectionID] = wsConn
	b.connectionsMu.Unlock()

	// Create event subscription for the connection
	subscription, err := b.eventRouter.SubscribeConnection(connectionID, userID, b.config.DefaultEventFilters)
	if err != nil {
		log.Printf("[WebSocketEventBridge] Failed to create subscription for connection %s: %v", connectionID, err)
		// Continue without subscription
	} else {
		wsConn.Subscription = subscription
	}

	// Start connection handlers
	go b.handleConnectionReads(wsConn)
	go b.handleConnectionWrites(wsConn)
	go b.handleConnectionEvents(wsConn)

	// Publish connection event
	connectionEvent := NewEvent(EventTypeWebSocketConnected, "websocket_bridge", WebSocketMessageData{
		ConnectionID: connectionID,
		UserID:       userID,
		Data:         map[string]interface{}{"remote_addr": conn.RemoteAddr().String()},
	})
	b.eventRouter.RouteToSubscribers(ctx, connectionEvent)

	log.Printf("[WebSocketEventBridge] New connection established: %s (user: %s)", connectionID, userID)
	return wsConn, nil
}

// CloseConnection closes a WebSocket connection
func (b *WebSocketEventBridge) CloseConnection(connectionID string) error {
	b.connectionsMu.Lock()
	defer b.connectionsMu.Unlock()

	conn, exists := b.connections[connectionID]
	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}

	// Cancel connection context
	conn.Cancel()

	// Close WebSocket connection
	conn.Conn.Close()

	// Mark as inactive
	conn.Active = false

	// Remove subscription
	if conn.Subscription != nil {
		b.eventRouter.UnsubscribeConnection(connectionID)
	}

	// Remove from connections map
	delete(b.connections, connectionID)

	// Publish disconnection event
	disconnectionEvent := NewEvent(EventTypeWebSocketDisconnected, "websocket_bridge", WebSocketMessageData{
		ConnectionID: connectionID,
		UserID:       conn.UserID,
		Data:         map[string]interface{}{"duration": time.Since(conn.CreatedAt).Seconds()},
	})
	b.eventRouter.RouteToSubscribers(context.Background(), disconnectionEvent)

	log.Printf("[WebSocketEventBridge] Connection closed: %s", connectionID)
	return nil
}

// BroadcastEvent broadcasts an event to all connected WebSocket clients
func (b *WebSocketEventBridge) BroadcastEvent(ctx context.Context, event Event) error {
	message := b.eventToWebSocketMessage(event)

	b.connectionsMu.RLock()
	defer b.connectionsMu.RUnlock()

	for _, conn := range b.connections {
		if !conn.Active {
			continue
		}

		select {
		case conn.SendChan <- message:
			// Message queued successfully
		default:
			log.Printf("[WebSocketEventBridge] Send channel full for connection %s", conn.ID)
		}
	}

	return nil
}

// SendEventToConnection sends an event to a specific connection
func (b *WebSocketEventBridge) SendEventToConnection(connectionID string, event Event) error {
	b.connectionsMu.RLock()
	conn, exists := b.connections[connectionID]
	b.connectionsMu.RUnlock()

	if !exists || !conn.Active {
		return fmt.Errorf("connection %s not found or inactive", connectionID)
	}

	message := b.eventToWebSocketMessage(event)

	select {
	case conn.SendChan <- message:
		return nil
	default:
		return fmt.Errorf("send channel full for connection %s", connectionID)
	}
}

// GetConnectionCount returns the number of active connections
func (b *WebSocketEventBridge) GetConnectionCount() int {
	b.connectionsMu.RLock()
	defer b.connectionsMu.RUnlock()
	return len(b.connections)
}

// GetConnection returns a connection by ID
func (b *WebSocketEventBridge) GetConnection(connectionID string) (*WebSocketConnection, bool) {
	b.connectionsMu.RLock()
	defer b.connectionsMu.RUnlock()
	conn, exists := b.connections[connectionID]
	return conn, exists
}

// Private methods

func (b *WebSocketEventBridge) handleConnectionReads(conn *WebSocketConnection) {
	defer func() {
		b.CloseConnection(conn.ID)
	}()

	conn.Conn.SetReadLimit(b.config.MaxMessageSize)
	conn.Conn.SetReadDeadline(time.Now().Add(b.config.ReadTimeout))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(b.config.PongTimeout))
		return nil
	})

	for {
		select {
		case <-conn.Context.Done():
			return
		default:
			var msg WebSocketMessage
			err := conn.Conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[WebSocketEventBridge] Unexpected close error for connection %s: %v", conn.ID, err)
				}
				return
			}

			// Update last activity
			conn.LastActivity = time.Now()

			// Handle the message
			if err := b.handleIncomingMessage(conn, msg); err != nil {
				log.Printf("[WebSocketEventBridge] Error handling message from connection %s: %v", conn.ID, err)

				// Send error response
				errorMsg := WebSocketMessage{
					Type:      "error",
					ID:        msg.ID,
					Error:     err.Error(),
					Timestamp: time.Now(),
				}

				select {
				case conn.SendChan <- errorMsg:
				default:
					log.Printf("[WebSocketEventBridge] Failed to send error message to connection %s", conn.ID)
				}
			}
		}
	}
}

func (b *WebSocketEventBridge) handleConnectionWrites(conn *WebSocketConnection) {
	ticker := time.NewTicker(b.config.PingInterval)
	defer func() {
		ticker.Stop()
		conn.Conn.Close()
	}()

	for {
		select {
		case <-conn.Context.Done():
			return
		case message := <-conn.SendChan:
			conn.Conn.SetWriteDeadline(time.Now().Add(b.config.WriteTimeout))
			if err := conn.Conn.WriteJSON(message); err != nil {
				log.Printf("[WebSocketEventBridge] Write error for connection %s: %v", conn.ID, err)
				return
			}
		case <-ticker.C:
			conn.Conn.SetWriteDeadline(time.Now().Add(b.config.WriteTimeout))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[WebSocketEventBridge] Ping error for connection %s: %v", conn.ID, err)
				return
			}
		}
	}
}

func (b *WebSocketEventBridge) handleConnectionEvents(conn *WebSocketConnection) {
	if conn.Subscription == nil {
		return
	}

	for {
		select {
		case <-conn.Context.Done():
			return
		case event := <-conn.Subscription.ResponseChan:
			// Convert event to WebSocket message and send
			message := b.eventToWebSocketMessage(event)

			select {
			case conn.SendChan <- message:
				// Event sent successfully
			default:
				log.Printf("[WebSocketEventBridge] Failed to send event to connection %s", conn.ID)
			}
		}
	}
}

func (b *WebSocketEventBridge) handleIncomingMessage(conn *WebSocketConnection, msg WebSocketMessage) error {
	// Convert WebSocket message to event
	event := b.webSocketMessageToEvent(conn, msg)

	// Handle different message types
	switch msg.Type {
	case "request":
		// This is a request that expects a response
		return b.handleRequestMessage(conn, msg, event)
	case "query":
		// Legacy query format - treat as request with method=query
		if msg.Method == "" {
			msg.Method = "query"
		}
		return b.handleRequestMessage(conn, msg, event)
	case "command":
		// This is a command that may not expect a response
		return b.handleCommandMessage(conn, msg, event)
	case "subscribe":
		// Handle subscription requests
		return b.handleSubscribeMessage(conn, msg)
	case "unsubscribe":
		// Handle unsubscription requests
		return b.handleUnsubscribeMessage(conn, msg)
	default:
		// Use the message handler for other types
		if b.messageHandler != nil {
			return b.messageHandler.HandleMessage(conn.Context, conn, msg)
		}

		// Default: publish as event
		return b.eventRouter.eventBus.Publish(event)
	}
}

func (b *WebSocketEventBridge) handleRequestMessage(conn *WebSocketConnection, msg WebSocketMessage, event Event) error {
	// Route request with correlation tracking
	correlationCtx, err := b.eventRouter.RouteRequestWithCorrelation(conn.Context, event, conn.ID, conn.UserID)
	if err != nil {
		return fmt.Errorf("failed to route request: %w", err)
	}

	// Wait for response in a goroutine to avoid blocking
	go func() {
		responseEvent, err := b.eventRouter.WaitForResponse(conn.Context, correlationCtx.ID, 30*time.Second)
		if err != nil {
			// Send error response
			errorMsg := WebSocketMessage{
				Type:          "error",
				ID:            msg.ID,
				Error:         err.Error(),
				Timestamp:     time.Now(),
				CorrelationID: correlationCtx.ID,
			}

			select {
			case conn.SendChan <- errorMsg:
			default:
				log.Printf("[WebSocketEventBridge] Failed to send timeout error to connection %s", conn.ID)
			}
			return
		}

		// Convert response event to WebSocket message
		responseMsg := b.eventToWebSocketMessage(responseEvent)
		responseMsg.ID = msg.ID // Keep original message ID for correlation

		select {
		case conn.SendChan <- responseMsg:
		default:
			log.Printf("[WebSocketEventBridge] Failed to send response to connection %s", conn.ID)
		}
	}()

	return nil
}

func (b *WebSocketEventBridge) handleCommandMessage(conn *WebSocketConnection, msg WebSocketMessage, event Event) error {
	// Commands are fire-and-forget
	return b.eventRouter.eventBus.Publish(event)
}

func (b *WebSocketEventBridge) handleSubscribeMessage(conn *WebSocketConnection, msg WebSocketMessage) error {
	// Handle both string and map subscription data formats
	var eventTypes []string

	switch data := msg.Data.(type) {
	case string:
		// Single subscription type as string (legacy format)
		eventTypes = []string{data}
	case map[string]interface{}:
		// Map format with event_types array
		if eventTypesData, exists := data["event_types"]; exists {
			if eventTypesList, ok := eventTypesData.([]interface{}); ok {
				for _, et := range eventTypesList {
					if etStr, ok := et.(string); ok {
						eventTypes = append(eventTypes, etStr)
					}
				}
			}
		} else {
			return fmt.Errorf("missing event_types in subscription data")
		}
	case []interface{}:
		// Array of event types
		for _, et := range data {
			if etStr, ok := et.(string); ok {
				eventTypes = append(eventTypes, etStr)
			}
		}
	default:
		return fmt.Errorf("invalid subscription data format, expected string, array, or map")
	}

	if len(eventTypes) == 0 {
		return fmt.Errorf("no valid event types provided")
	}

	// Update connection subscription
	if conn.Subscription != nil {
		// Add new filters to existing subscription
		filter := SubscriptionFilter{EventTypes: eventTypes}
		conn.Subscription.EventFilters["custom"] = filter
	}

	// Send acknowledgment
	ackMsg := WebSocketMessage{
		Type:      "subscription_ack",
		ID:        msg.ID,
		Data:      map[string]interface{}{"status": "subscribed"},
		Timestamp: time.Now(),
	}

	select {
	case conn.SendChan <- ackMsg:
		return nil
	default:
		return fmt.Errorf("failed to send subscription acknowledgment")
	}
}

func (b *WebSocketEventBridge) handleUnsubscribeMessage(conn *WebSocketConnection, msg WebSocketMessage) error {
	// Handle both string and map unsubscribe data formats
	var filterNames []string

	switch data := msg.Data.(type) {
	case string:
		// Single filter name as string (legacy format)
		filterNames = []string{data}
	case map[string]interface{}:
		// Map format - check for filter_name or event_types
		if filterName, exists := data["filter_name"]; exists {
			if filterNameStr, ok := filterName.(string); ok {
				filterNames = []string{filterNameStr}
			}
		} else if eventTypes, exists := data["event_types"]; exists {
			// Unsubscribe from specific event types
			if eventTypesList, ok := eventTypes.([]interface{}); ok {
				for _, et := range eventTypesList {
					if etStr, ok := et.(string); ok {
						filterNames = append(filterNames, etStr)
					}
				}
			}
		} else {
			// Default: remove all custom filters
			filterNames = []string{"custom"}
		}
	case []interface{}:
		// Array of filter names or event types
		for _, fn := range data {
			if fnStr, ok := fn.(string); ok {
				filterNames = append(filterNames, fnStr)
			}
		}
	default:
		return fmt.Errorf("invalid unsubscribe data format, expected string, array, or map")
	}

	// Remove specific filters if specified
	if conn.Subscription != nil {
		for _, filterName := range filterNames {
			delete(conn.Subscription.EventFilters, filterName)
		}
	}

	// Send acknowledgment
	ackMsg := WebSocketMessage{
		Type:      "unsubscription_ack",
		ID:        msg.ID,
		Data:      map[string]interface{}{"status": "unsubscribed"},
		Timestamp: time.Now(),
	}

	select {
	case conn.SendChan <- ackMsg:
		return nil
	default:
		return fmt.Errorf("failed to send unsubscription acknowledgment")
	}
}

func (b *WebSocketEventBridge) webSocketMessageToEvent(conn *WebSocketConnection, msg WebSocketMessage) Event {
	// Determine event type based on message
	eventType := EventTypeWebSocketMessage
	if msg.Method != "" {
		// Map method to specific event type
		eventType = b.mapMethodToEventType(msg.Method)
	} else if msg.EventType != "" {
		eventType = msg.EventType
	}

	// Create appropriate event data based on event type
	var eventData interface{}

	if eventType == EventTypeQuerySubmit && msg.Method == "query" {
		// For query events, create QueryData directly
		if queryDataMap, ok := msg.Data.(map[string]interface{}); ok {
			queryText, hasQuery := queryDataMap["query"].(string)
			useRAG, hasRAG := queryDataMap["use_rag"].(bool)

			if hasQuery && hasRAG {
				eventData = QueryData{
					QueryID:      msg.ID,
					Text:         queryText,
					UseRAG:       useRAG,
					UserID:       conn.UserID,
					ConnectionID: conn.ID,
					Parameters:   queryDataMap,
				}
			} else {
				// Fallback to WebSocketMessageData if query data is invalid
				eventData = WebSocketMessageData{
					ConnectionID: conn.ID,
					MessageID:    msg.ID,
					MessageType:  msg.Type,
					Method:       msg.Method,
					Data:         msg.Data,
					UserID:       conn.UserID,
				}
			}
		} else {
			// Fallback to WebSocketMessageData if data is not a map
			eventData = WebSocketMessageData{
				ConnectionID: conn.ID,
				MessageID:    msg.ID,
				MessageType:  msg.Type,
				Method:       msg.Method,
				Data:         msg.Data,
				UserID:       conn.UserID,
			}
		}
	} else {
		// For non-query events, use WebSocketMessageData
		eventData = WebSocketMessageData{
			ConnectionID: conn.ID,
			MessageID:    msg.ID,
			MessageType:  msg.Type,
			Method:       msg.Method,
			Data:         msg.Data,
			UserID:       conn.UserID,
		}
	}

	// Create event
	event := NewEvent(eventType, "websocket_bridge", eventData)
	event.CorrelationID = msg.CorrelationID
	event.Target = "api_server"

	// Add connection metadata
	event = event.WithMetadata("connection_id", conn.ID).WithMetadata("user_id", conn.UserID)
	if len(msg.Metadata) > 0 {
		for key, value := range msg.Metadata {
			event = event.WithMetadata(key, value)
		}
	}

	return event
}

func (b *WebSocketEventBridge) eventToWebSocketMessage(event Event) WebSocketMessage {
	// Determine message type based on event
	msgType := "event"
	if event.Type == EventTypeModelsData {
		msgType = "models_data"
	} else if event.Type == EventTypeAgentsData {
		msgType = "agents_data"
	} else if event.Type == EventTypeMetricsUpdated {
		msgType = "metrics"
	} else if event.Type == EventTypeConsciousnessUpdated {
		msgType = "consciousness"
	}

	return WebSocketMessage{
		Type:          msgType,
		ID:            event.ID,
		Data:          event.Data,
		Timestamp:     event.Timestamp,
		Metadata:      event.Metadata,
		CorrelationID: event.CorrelationID,
		EventType:     event.Type,
	}
}

func (b *WebSocketEventBridge) mapMethodToEventType(method string) string {
	methodToEventType := map[string]string{
		"getModels":     EventTypeGetModels,
		"getAgents":     EventTypeGetAgents,
		"getStatus":     EventTypeGetStatus,
		"getMetrics":    EventTypeGetMetrics,
		"query":         EventTypeQuerySubmit, // Handle legacy query format
		"submitQuery":   EventTypeQuerySubmit,
		"downloadModel": EventTypeModelDownload,
		"deleteModel":   EventTypeModelDelete,
	}

	if eventType, exists := methodToEventType[method]; exists {
		return eventType
	}

	return EventTypeWebSocketMessage
}

// DefaultMessageHandler provides default message handling
type DefaultMessageHandler struct {
	bridge *WebSocketEventBridge
}

func (h *DefaultMessageHandler) HandleMessage(ctx context.Context, conn *WebSocketConnection, msg WebSocketMessage) error {
	// Default handling: convert to event and publish
	event := h.bridge.webSocketMessageToEvent(conn, msg)
	return h.bridge.eventRouter.eventBus.Publish(event)
}

func (h *DefaultMessageHandler) GetSupportedMessageTypes() []string {
	return []string{"query", "ping", "get_status", "restart_ollama"}
}
