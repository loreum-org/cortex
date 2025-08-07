package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/events"
)

// WebSocketEventHandler bridges WebSocket messages to domain events
type WebSocketEventHandler struct {
	eventBus              *events.EventBus
	name                  string
	connections           map[string]*WebSocketConnection
	subscriptions         map[string]map[string]bool            // eventType -> connectionID -> subscribed
	serviceSubscriptions  map[string]map[string]map[string]bool // serviceType -> eventType -> connectionID -> subscribed
	categorySubscriptions map[string]map[string]bool            // category -> connectionID -> subscribed
}

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	ID       string
	UserID   string
	Send     chan interface{}
	LastSeen time.Time
}

// UnifiedMessage represents the unified WebSocket message format
type UnifiedMessage struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Method        string                 `json:"method,omitempty"`
	Data          interface{}            `json:"data"`
	Error         string                 `json:"error,omitempty"`
	Timestamp     string                 `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	// Service categorization
	ServiceType string `json:"service_type,omitempty"`
	ServiceID   string `json:"service_id,omitempty"`
	Category    string `json:"category,omitempty"`
}

// NewWebSocketEventHandler creates a new WebSocket event handler
func NewWebSocketEventHandler(eventBus *events.EventBus) *WebSocketEventHandler {
	return &WebSocketEventHandler{
		eventBus:              eventBus,
		name:                  "websocket_handler",
		connections:           make(map[string]*WebSocketConnection),
		subscriptions:         make(map[string]map[string]bool),
		serviceSubscriptions:  make(map[string]map[string]map[string]bool),
		categorySubscriptions: make(map[string]map[string]bool),
	}
}

// HandlerName returns the handler name
func (wh *WebSocketEventHandler) HandlerName() string {
	return wh.name
}

// SubscribedEvents returns the events this handler subscribes to
func (wh *WebSocketEventHandler) SubscribedEvents() []string {
	return []string{
		// WebSocket events
		events.EventTypeWebSocketMessage,
		events.EventTypeWebSocketConnected,
		events.EventTypeWebSocketDisconnected,

		// Query events
		events.EventTypeQueryProcessed,
		events.EventTypeQueryFailed,
		events.EventTypeQueryStarted,
		events.EventTypeQueryProgress,
		"query.response", // Handle query responses to send back to WebSocket clients

		// Agent events
		events.EventTypeAgentRegistered,
		events.EventTypeAgentDeregistered,

		// Model events
		events.EventTypeModelDownloaded,
		events.EventTypeModelDeleted,
		events.EventTypeModelDownload, // Progress events

		// System events
		events.EventTypeMetricsUpdated,
		events.EventTypeConsciousnessUpdated,
		events.EventTypeError,
		"system.health.status",

		// API Response events
		events.EventTypeModelsData,
		events.EventTypeAgentsData,
		events.EventTypeStatusData,
		events.EventTypeMetricsData,
		events.EventTypeTransactionsData,
		events.EventTypeAccountsData,
		events.EventTypeNodeInfoData,
		events.EventTypePeersData,
		events.EventTypeWalletData,
		events.EventTypeServicesData,
	}
}

// Handle processes events and forwards them to WebSocket connections
func (wh *WebSocketEventHandler) Handle(ctx context.Context, event events.Event) error {
	switch event.Type {
	// WebSocket events
	case events.EventTypeWebSocketMessage:
		return wh.handleWebSocketMessageEvent(ctx, event)
	case events.EventTypeWebSocketConnected:
		return wh.handleWebSocketConnectedEvent(ctx, event)
	case events.EventTypeWebSocketDisconnected:
		return wh.handleWebSocketDisconnectedEvent(ctx, event)

	// Query events
	case events.EventTypeQueryProcessed, events.EventTypeQueryFailed:
		return wh.handleQueryResult(ctx, event)
	case events.EventTypeQueryStarted:
		return wh.handleQueryStarted(ctx, event)
	case events.EventTypeQueryProgress:
		return wh.handleQueryProgress(ctx, event)
	case "query.response":
		return wh.handleQueryResponse(ctx, event)

	// Agent events
	case events.EventTypeAgentRegistered, events.EventTypeAgentDeregistered:
		return wh.handleAgentEvent(ctx, event)

	// Model events
	case events.EventTypeModelDownloaded, events.EventTypeModelDeleted, events.EventTypeModelDownload:
		return wh.handleModelEvent(ctx, event)

	// System events
	case events.EventTypeMetricsUpdated:
		return wh.handleMetricsUpdate(ctx, event)
	case events.EventTypeConsciousnessUpdated:
		return wh.handleConsciousnessUpdate(ctx, event)
	case events.EventTypeError:
		return wh.handleError(ctx, event)
	case "system.health.status":
		return wh.handleHealthStatus(ctx, event)

	// API Response events - forward to requesting connection
	case events.EventTypeModelsData, events.EventTypeAgentsData, events.EventTypeStatusData,
		events.EventTypeMetricsData, events.EventTypeTransactionsData, events.EventTypeAccountsData,
		events.EventTypeNodeInfoData, events.EventTypePeersData, events.EventTypeWalletData,
		events.EventTypeServicesData:
		return wh.handleAPIResponse(ctx, event)

	default:
		return fmt.Errorf("unsupported event type: %s", event.Type)
	}
}

// RegisterConnection registers a new WebSocket connection
func (wh *WebSocketEventHandler) RegisterConnection(conn *WebSocketConnection) {
	wh.connections[conn.ID] = conn
	log.Printf("🔌 WebSocket Handler: Connection %s registered", conn.ID)
}

// UnregisterConnection removes a WebSocket connection
func (wh *WebSocketEventHandler) UnregisterConnection(connectionID string) {
	delete(wh.connections, connectionID)

	// Clean up subscriptions
	for eventType := range wh.subscriptions {
		delete(wh.subscriptions[eventType], connectionID)
		if len(wh.subscriptions[eventType]) == 0 {
			delete(wh.subscriptions, eventType)
		}
	}

	// Clean up service subscriptions
	for serviceType := range wh.serviceSubscriptions {
		for eventType := range wh.serviceSubscriptions[serviceType] {
			delete(wh.serviceSubscriptions[serviceType][eventType], connectionID)
			if len(wh.serviceSubscriptions[serviceType][eventType]) == 0 {
				delete(wh.serviceSubscriptions[serviceType], eventType)
				if len(wh.serviceSubscriptions[serviceType]) == 0 {
					delete(wh.serviceSubscriptions, serviceType)
				}
			}
		}
	}

	// Clean up category subscriptions
	for category := range wh.categorySubscriptions {
		delete(wh.categorySubscriptions[category], connectionID)
		if len(wh.categorySubscriptions[category]) == 0 {
			delete(wh.categorySubscriptions, category)
		}
	}

	log.Printf("🔌 WebSocket Handler: Connection %s unregistered", connectionID)
}

// HandleWebSocketMessage processes incoming WebSocket messages and converts them to domain events
func (wh *WebSocketEventHandler) HandleWebSocketMessage(conn *WebSocketConnection, msg *UnifiedMessage) error {
	log.Printf("🔍 HandleWebSocketMessage: Type=%s, Method=%s, Data=%+v", msg.Type, msg.Method, msg.Data)

	switch msg.Type {
	case "request":
		return wh.handleWebSocketRequest(conn, msg)
	case "query":
		// Handle direct query messages by converting them to request format
		log.Printf("✅ Converting query message to request format")
		msg.Type = "request"
		msg.Method = "query"
		return wh.handleWebSocketRequest(conn, msg)
	case "subscribe":
		return wh.handleWebSocketSubscribe(conn, msg)
	case "unsubscribe":
		return wh.handleWebSocketUnsubscribe(conn, msg)
	case "subscribe_service":
		return wh.handleServiceSubscribe(conn, msg)
	case "unsubscribe_service":
		return wh.handleServiceUnsubscribe(conn, msg)
	case "subscribe_category":
		return wh.handleCategorySubscribe(conn, msg)
	case "unsubscribe_category":
		return wh.handleCategoryUnsubscribe(conn, msg)
	default:
		log.Printf("❌ Unknown message type: %s", msg.Type)
		return wh.sendError(conn, msg.ID, fmt.Sprintf("Unknown message type: %s", msg.Type))
	}
}

// handleWebSocketRequest converts WebSocket requests to domain events
func (wh *WebSocketEventHandler) handleWebSocketRequest(conn *WebSocketConnection, msg *UnifiedMessage) error {
	var domainEvent events.Event

	switch msg.Method {
	// Query operations
	case "query":
		domainEvent = wh.createQueryEvent(conn, msg)

	// Model operations
	case "getModels":
		domainEvent = wh.createGetModelsEvent(conn, msg)
	case "downloadModel":
		domainEvent = wh.createDownloadModelEvent(conn, msg)
	case "deleteModel":
		domainEvent = wh.createDeleteModelEvent(conn, msg)

	// Agent operations
	case "getAgents":
		domainEvent = wh.createGetAgentsEvent(conn, msg)

	// System operations
	case "getStatus":
		domainEvent = wh.createGetStatusEvent(conn, msg)
	case "getMetrics":
		domainEvent = wh.createGetMetricsEvent(conn, msg)

	// Network operations
	case "getNodeInfo":
		domainEvent = wh.createGetNodeInfoEvent(conn, msg)
	case "getPeers":
		domainEvent = wh.createGetPeersEvent(conn, msg)

	// Wallet operations
	case "getWalletBalance":
		domainEvent = wh.createGetWalletBalanceEvent(conn, msg)
	case "walletTransfer":
		domainEvent = wh.createWalletTransferEvent(conn, msg)

	// Transaction operations
	case "getTransactions":
		domainEvent = wh.createGetTransactionsEvent(conn, msg)
	case "getAccounts":
		domainEvent = wh.createGetAccountsEvent(conn, msg)

	// Service operations
	case "getServices":
		domainEvent = wh.createGetServicesEvent(conn, msg)
	case "registerService":
		domainEvent = wh.createServiceRegisterEvent(conn, msg)
	case "deregisterService":
		domainEvent = wh.createServiceDeregisterEvent(conn, msg)

	default:
		return wh.sendError(conn, msg.ID, fmt.Sprintf("Unknown request method: %s", msg.Method))
	}

	// Publish domain event
	return wh.eventBus.Publish(domainEvent)
}

// handleWebSocketSubscribe handles subscription requests
func (wh *WebSocketEventHandler) handleWebSocketSubscribe(conn *WebSocketConnection, msg *UnifiedMessage) error {
	eventType, ok := msg.Data.(string)
	if !ok {
		return wh.sendError(conn, msg.ID, "Invalid subscription data")
	}

	// Map WebSocket subscription types to domain event types
	mappedEventType := wh.mapSubscriptionType(eventType)

	// Register subscription
	if wh.subscriptions[mappedEventType] == nil {
		wh.subscriptions[mappedEventType] = make(map[string]bool)
	}
	wh.subscriptions[mappedEventType][conn.ID] = true

	log.Printf("🔔 WebSocket Handler: Connection %s subscribed to %s", conn.ID, mappedEventType)

	// Send confirmation
	return wh.sendMessage(conn, &UnifiedMessage{
		ID:   msg.ID,
		Type: "response",
		Data: map[string]interface{}{
			"subscribed": true,
			"event_type": eventType,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// handleWebSocketUnsubscribe handles unsubscription requests
func (wh *WebSocketEventHandler) handleWebSocketUnsubscribe(conn *WebSocketConnection, msg *UnifiedMessage) error {
	eventType, ok := msg.Data.(string)
	if !ok {
		return wh.sendError(conn, msg.ID, "Invalid unsubscription data")
	}

	mappedEventType := wh.mapSubscriptionType(eventType)

	// Remove subscription
	if wh.subscriptions[mappedEventType] != nil {
		delete(wh.subscriptions[mappedEventType], conn.ID)
		if len(wh.subscriptions[mappedEventType]) == 0 {
			delete(wh.subscriptions, mappedEventType)
		}
	}

	log.Printf("🔕 WebSocket Handler: Connection %s unsubscribed from %s", conn.ID, mappedEventType)

	// Send confirmation
	return wh.sendMessage(conn, &UnifiedMessage{
		ID:   msg.ID,
		Type: "response",
		Data: map[string]interface{}{
			"unsubscribed": true,
			"event_type":   eventType,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// handleServiceSubscribe handles service-specific subscription requests
func (wh *WebSocketEventHandler) handleServiceSubscribe(conn *WebSocketConnection, msg *UnifiedMessage) error {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return wh.sendError(conn, msg.ID, "Invalid service subscription data")
	}

	serviceType, hasServiceType := data["service_type"].(string)
	eventType, hasEventType := data["event_type"].(string)

	if !hasServiceType {
		return wh.sendError(conn, msg.ID, "service_type is required")
	}

	// If no specific event type, subscribe to all events for this service
	if !hasEventType {
		eventType = "*" // Wildcard for all events
	}

	// Initialize nested maps if needed
	if wh.serviceSubscriptions[serviceType] == nil {
		wh.serviceSubscriptions[serviceType] = make(map[string]map[string]bool)
	}
	if wh.serviceSubscriptions[serviceType][eventType] == nil {
		wh.serviceSubscriptions[serviceType][eventType] = make(map[string]bool)
	}

	wh.serviceSubscriptions[serviceType][eventType][conn.ID] = true

	log.Printf("🔔 WebSocket Handler: Connection %s subscribed to service %s events %s", conn.ID, serviceType, eventType)

	return wh.sendMessage(conn, &UnifiedMessage{
		ID:   msg.ID,
		Type: "response",
		Data: map[string]interface{}{
			"subscribed":   true,
			"service_type": serviceType,
			"event_type":   eventType,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// handleServiceUnsubscribe handles service-specific unsubscription requests
func (wh *WebSocketEventHandler) handleServiceUnsubscribe(conn *WebSocketConnection, msg *UnifiedMessage) error {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return wh.sendError(conn, msg.ID, "Invalid service unsubscription data")
	}

	serviceType, hasServiceType := data["service_type"].(string)
	eventType, hasEventType := data["event_type"].(string)

	if !hasServiceType {
		return wh.sendError(conn, msg.ID, "service_type is required")
	}

	if !hasEventType {
		eventType = "*"
	}

	// Remove subscription
	if wh.serviceSubscriptions[serviceType] != nil &&
		wh.serviceSubscriptions[serviceType][eventType] != nil {
		delete(wh.serviceSubscriptions[serviceType][eventType], conn.ID)

		// Clean up empty maps
		if len(wh.serviceSubscriptions[serviceType][eventType]) == 0 {
			delete(wh.serviceSubscriptions[serviceType], eventType)
			if len(wh.serviceSubscriptions[serviceType]) == 0 {
				delete(wh.serviceSubscriptions, serviceType)
			}
		}
	}

	log.Printf("🔕 WebSocket Handler: Connection %s unsubscribed from service %s events %s", conn.ID, serviceType, eventType)

	return wh.sendMessage(conn, &UnifiedMessage{
		ID:   msg.ID,
		Type: "response",
		Data: map[string]interface{}{
			"unsubscribed": true,
			"service_type": serviceType,
			"event_type":   eventType,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// handleCategorySubscribe handles category-specific subscription requests
func (wh *WebSocketEventHandler) handleCategorySubscribe(conn *WebSocketConnection, msg *UnifiedMessage) error {
	category, ok := msg.Data.(string)
	if !ok {
		return wh.sendError(conn, msg.ID, "Invalid category subscription data")
	}

	if wh.categorySubscriptions[category] == nil {
		wh.categorySubscriptions[category] = make(map[string]bool)
	}
	wh.categorySubscriptions[category][conn.ID] = true

	log.Printf("🔔 WebSocket Handler: Connection %s subscribed to category %s", conn.ID, category)

	return wh.sendMessage(conn, &UnifiedMessage{
		ID:   msg.ID,
		Type: "response",
		Data: map[string]interface{}{
			"subscribed": true,
			"category":   category,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// handleCategoryUnsubscribe handles category-specific unsubscription requests
func (wh *WebSocketEventHandler) handleCategoryUnsubscribe(conn *WebSocketConnection, msg *UnifiedMessage) error {
	category, ok := msg.Data.(string)
	if !ok {
		return wh.sendError(conn, msg.ID, "Invalid category unsubscription data")
	}

	if wh.categorySubscriptions[category] != nil {
		delete(wh.categorySubscriptions[category], conn.ID)
		if len(wh.categorySubscriptions[category]) == 0 {
			delete(wh.categorySubscriptions, category)
		}
	}

	log.Printf("🔕 WebSocket Handler: Connection %s unsubscribed from category %s", conn.ID, category)

	return wh.sendMessage(conn, &UnifiedMessage{
		ID:   msg.ID,
		Type: "response",
		Data: map[string]interface{}{
			"unsubscribed": true,
			"category":     category,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Event handlers for forwarding domain events to WebSocket connections

func (wh *WebSocketEventHandler) handleQueryResult(ctx context.Context, event events.Event) error {
	resultData, ok := event.Data.(events.QueryResultData)
	if !ok {
		return fmt.Errorf("invalid query result data")
	}

	// Find connection that initiated this query
	connectionID := wh.getConnectionIDFromMetadata(event)
	if connectionID == "" {
		return nil // No specific connection, broadcast to subscribers
	}

	conn := wh.connections[connectionID]
	if conn == nil {
		return nil // Connection no longer exists
	}

	messageType := "response"
	if event.Type == events.EventTypeQueryFailed {
		messageType = "error"
	}

	wsMessage := &UnifiedMessage{
		ID:   event.CorrelationID,
		Type: messageType,
		Data: map[string]interface{}{
			"query_id": resultData.QueryID,
			"result":   resultData.Result,
			"source":   resultData.Source,
			"duration": resultData.Duration.String(),
			"metadata": resultData.Metadata,
		},
		Error:     resultData.Error,
		Timestamp: event.Timestamp.Format(time.RFC3339),
	}

	return wh.sendMessage(conn, wsMessage)
}

func (wh *WebSocketEventHandler) handleQueryStarted(ctx context.Context, event events.Event) error {
	return wh.broadcastToSubscribers("queryProgress", event)
}

func (wh *WebSocketEventHandler) handleQueryProgress(ctx context.Context, event events.Event) error {
	return wh.broadcastToSubscribers("queryProgress", event)
}

func (wh *WebSocketEventHandler) handleQueryResponse(ctx context.Context, event events.Event) error {
	log.Printf("🔌 WebSocket Handler: Handling query response event %s", event.ID)

	// Extract response data - handle both QueryResultData and generic data
	var responseData events.QueryResultData
	var ok bool

	if responseData, ok = event.Data.(events.QueryResultData); !ok {
		// Try to extract from generic map data
		if dataMap, isMap := event.Data.(map[string]interface{}); isMap {
			responseData = events.QueryResultData{
				QueryID:  fmt.Sprintf("%v", dataMap["query_id"]),
				Result:   fmt.Sprintf("%v", dataMap["result"]),
				Source:   fmt.Sprintf("%v", dataMap["source"]),
				Error:    fmt.Sprintf("%v", dataMap["error"]),
				Metadata: make(map[string]interface{}),
			}
			if metadata, hasMetadata := dataMap["metadata"].(map[string]interface{}); hasMetadata {
				responseData.Metadata = metadata
			}
		} else {
			log.Printf("⚠️ WebSocket Handler: Invalid query response data type: %T", event.Data)
			return fmt.Errorf("invalid query response data type")
		}
	}

	// Find the connection that made the original query
	connectionID := wh.getConnectionIDFromMetadata(event)
	if connectionID == "" {
		log.Printf("⚠️ WebSocket Handler: No connection ID found in query response metadata")
		// Try to broadcast to all subscribers as fallback
		return wh.broadcastToSubscribers("queryProgress", event)
	}

	conn := wh.connections[connectionID]
	if conn == nil {
		log.Printf("⚠️ WebSocket Handler: Connection %s not found for query response", connectionID)
		// Try to broadcast to all subscribers as fallback
		return wh.broadcastToSubscribers("queryProgress", event)
	}

	// Extract the original query ID from the WebSocket message ID stored in metadata
	originalQueryID := event.CorrelationID
	if originalQueryIDFromMeta, ok := event.Metadata["original_query_id"].(string); ok {
		originalQueryID = originalQueryIDFromMeta
	} else if originalQueryID == "" {
		originalQueryID = responseData.QueryID
	}

	// Create WebSocket response message with original query ID
	responseMessage := &UnifiedMessage{
		ID:   originalQueryID, // Use original query ID so frontend can match it
		Type: "query_chunk",   // Match the expected frontend message type
		Data: map[string]interface{}{
			"chunk": responseData.Result,
			"metadata": map[string]interface{}{
				"source":          responseData.Source,
				"processing_type": "consciousness_processed",
				"query_id":        responseData.QueryID,
				"duration":        responseData.Duration.String(),
				"conversation_id": connectionID,
				"error":           responseData.Error,
			},
		},
		Timestamp: event.Timestamp.Format(time.RFC3339),
	}

	log.Printf("✅ WebSocket Handler: Sending query response to connection %s with ID %s", connectionID, originalQueryID)

	// Send response to the specific connection
	select {
	case conn.Send <- responseMessage:
		log.Printf("✅ WebSocket Handler: Query response sent successfully")
		return nil
	default:
		log.Printf("❌ WebSocket Handler: Failed to send query response - channel full or closed")
		return fmt.Errorf("failed to send response to connection %s", connectionID)
	}
}

func (wh *WebSocketEventHandler) handleAgentEvent(ctx context.Context, event events.Event) error {
	return wh.broadcastToSubscribers("agents", event)
}

func (wh *WebSocketEventHandler) handleModelEvent(ctx context.Context, event events.Event) error {
	return wh.broadcastToSubscribers("models", event)
}

func (wh *WebSocketEventHandler) handleMetricsUpdate(ctx context.Context, event events.Event) error {
	return wh.broadcastToSubscribers("metrics", event)
}

func (wh *WebSocketEventHandler) handleConsciousnessUpdate(ctx context.Context, event events.Event) error {
	// Enhanced consciousness update handling with streaming support
	consciousnessData, ok := event.Data.(events.ConsciousnessData)
	if !ok {
		log.Printf("⚠️ WebSocket Handler: Invalid consciousness data in event %s", event.ID)
		return wh.broadcastToSubscribers("consciousness", event)
	}

	// Create enhanced WebSocket message with consciousness details
	enhancedMessage := &UnifiedMessage{
		ID:   event.ID,
		Type: "consciousness_update",
		Data: map[string]interface{}{
			"state":          consciousnessData.State,
			"node_id":        consciousnessData.NodeID,
			"working_memory": consciousnessData.WorkingMemory,
			"metrics":        consciousnessData.Metrics,
			"timestamp":      event.Timestamp,
		},
		Timestamp: event.Timestamp.Format(time.RFC3339),
		Metadata: map[string]interface{}{
			"event_type":    event.Type,
			"source":        event.Source,
			"consciousness": true,
			"streaming":     true,
		},
	}

	// Send to consciousness subscribers with enhanced data
	return wh.broadcastConsciousnessUpdate(enhancedMessage)
}

func (wh *WebSocketEventHandler) handleError(ctx context.Context, event events.Event) error {
	// Send error to specific connection if correlation ID matches
	connectionID := wh.getConnectionIDFromMetadata(event)
	if connectionID != "" {
		conn := wh.connections[connectionID]
		if conn != nil {
			wsMessage := &UnifiedMessage{
				ID:        event.CorrelationID,
				Type:      "error",
				Error:     fmt.Sprintf("%v", event.Data),
				Timestamp: event.Timestamp.Format(time.RFC3339),
			}
			return wh.sendMessage(conn, wsMessage)
		}
	}

	return nil
}

func (wh *WebSocketEventHandler) handleHealthStatus(ctx context.Context, event events.Event) error {
	return wh.broadcastToSubscribers("systemEvents", event)
}

func (wh *WebSocketEventHandler) handleAPIResponse(ctx context.Context, event events.Event) error {
	// Find connection that made this request
	connectionID := wh.getConnectionIDFromMetadata(event)
	if connectionID == "" {
		return nil // No specific connection, ignore
	}

	conn := wh.connections[connectionID]
	if conn == nil {
		return nil // Connection no longer exists
	}

	wsMessage := &UnifiedMessage{
		ID:        event.CorrelationID,
		Type:      "response",
		Data:      event.Data,
		Timestamp: event.Timestamp.Format(time.RFC3339),
		Metadata: map[string]interface{}{
			"event_type": event.Type,
			"source":     event.Source,
		},
	}

	return wh.sendMessage(conn, wsMessage)
}

// Helper methods

func (wh *WebSocketEventHandler) createQueryEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	data := msg.Data.(map[string]interface{})

	queryData := events.QueryData{
		QueryID:      msg.ID,
		Text:         data["query"].(string),
		UseRAG:       data["use_rag"].(bool),
		UserID:       conn.UserID,
		ConnectionID: conn.ID,
		Parameters:   data,
	}

	// Use correlation ID from message if available, otherwise use message ID
	correlationID := msg.CorrelationID
	if correlationID == "" {
		correlationID = msg.ID
	}

	return events.NewEventWithCorrelation(
		events.EventTypeQuerySubmit,
		"websocket",
		queryData,
		correlationID,
	).WithMetadata("connection_id", conn.ID).WithMetadata("user_id", conn.UserID).WithMetadata("original_query_id", msg.ID).WithCategory(events.CategoryRAG)
}

func (wh *WebSocketEventHandler) createGetAgentsEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	return events.NewEventWithCorrelation(
		"agent.list",
		"websocket",
		map[string]interface{}{
			"request_type": "get_agents",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID).WithCategory(events.CategoryAgent)
}

func (wh *WebSocketEventHandler) createGetModelsEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	return events.NewEventWithCorrelation(
		"model.list",
		"websocket",
		map[string]interface{}{
			"request_type": "get_models",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID).WithCategory(events.CategoryModel)
}

func (wh *WebSocketEventHandler) createDownloadModelEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	data := msg.Data.(map[string]interface{})

	return events.NewEventWithCorrelation(
		events.EventTypeModelDownload,
		"websocket",
		events.ModelData{
			ModelName: data["model_name"].(string),
			Action:    "download",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createDeleteModelEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	data := msg.Data.(map[string]interface{})

	return events.NewEventWithCorrelation(
		events.EventTypeModelDelete,
		"websocket",
		events.ModelData{
			ModelName: data["model_name"].(string),
			Action:    "delete",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createGetStatusEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	return events.NewEventWithCorrelation(
		events.EventTypeGetStatus,
		"websocket",
		map[string]interface{}{
			"request_type": "get_status",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createGetMetricsEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	return events.NewEventWithCorrelation(
		events.EventTypeGetMetrics,
		"websocket",
		map[string]interface{}{
			"request_type": "get_metrics",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createGetNodeInfoEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	return events.NewEventWithCorrelation(
		events.EventTypeGetNodeInfo,
		"websocket",
		map[string]interface{}{
			"request_type": "get_node_info",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createGetPeersEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	return events.NewEventWithCorrelation(
		events.EventTypeGetPeers,
		"websocket",
		map[string]interface{}{
			"request_type": "get_peers",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createGetWalletBalanceEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	data := msg.Data.(map[string]interface{})

	return events.NewEventWithCorrelation(
		events.EventTypeWalletBalance,
		"websocket",
		map[string]interface{}{
			"account": data["account"],
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createWalletTransferEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	data := msg.Data.(map[string]interface{})

	return events.NewEventWithCorrelation(
		events.EventTypeWalletTransfer,
		"websocket",
		data,
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createGetTransactionsEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	return events.NewEventWithCorrelation(
		events.EventTypeGetTransactions,
		"websocket",
		map[string]interface{}{
			"request_type": "get_transactions",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createGetAccountsEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	return events.NewEventWithCorrelation(
		events.EventTypeGetAccounts,
		"websocket",
		map[string]interface{}{
			"request_type": "get_accounts",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createGetServicesEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	return events.NewEventWithCorrelation(
		events.EventTypeGetServices,
		"websocket",
		map[string]interface{}{
			"request_type": "get_services",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createServiceRegisterEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	data := msg.Data.(map[string]interface{})

	return events.NewEventWithCorrelation(
		events.EventTypeServiceRegister,
		"websocket",
		data,
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createServiceDeregisterEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	data := msg.Data.(map[string]interface{})

	return events.NewEventWithCorrelation(
		events.EventTypeServiceDeregister,
		"websocket",
		data,
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) createHealthCheckEvent(conn *WebSocketConnection, msg *UnifiedMessage) events.Event {
	return events.NewEventWithCorrelation(
		events.EventTypeHealthCheck,
		"websocket",
		map[string]interface{}{
			"request_type": "health_check",
		},
		msg.ID,
	).WithMetadata("connection_id", conn.ID)
}

func (wh *WebSocketEventHandler) mapSubscriptionType(wsType string) string {
	switch wsType {
	case "metrics":
		return events.EventTypeMetricsUpdated
	case "consciousness":
		return events.EventTypeConsciousnessUpdated
	case "agents":
		return events.EventTypeAgentRegistered
	case "models":
		return events.EventTypeModelDownloaded
	case "systemEvents":
		return "system.health.status"
	case "queryProgress":
		return events.EventTypeQueryProgress
	default:
		return wsType
	}
}

func (wh *WebSocketEventHandler) broadcastToSubscribers(subscriptionType string, event events.Event) error {
	mappedEventType := wh.mapSubscriptionType(subscriptionType)
	subscribers := wh.subscriptions[mappedEventType]

	// Collect all subscribers (regular + service + category)
	allSubscribers := make(map[string]bool)

	// Add regular subscribers
	for connID := range subscribers {
		allSubscribers[connID] = true
	}

	// Add service-specific subscribers
	if event.ServiceType != "" {
		if serviceEvents := wh.serviceSubscriptions[event.ServiceType]; serviceEvents != nil {
			// Check for wildcard subscription
			if wildcardSubs := serviceEvents["*"]; wildcardSubs != nil {
				for connID := range wildcardSubs {
					allSubscribers[connID] = true
				}
			}
			// Check for specific event type subscription
			if eventSubs := serviceEvents[event.Type]; eventSubs != nil {
				for connID := range eventSubs {
					allSubscribers[connID] = true
				}
			}
		}
	}

	// Add category subscribers
	if categorySubs := wh.categorySubscriptions[string(event.Category)]; categorySubs != nil {
		for connID := range categorySubs {
			allSubscribers[connID] = true
		}
	}

	if len(allSubscribers) == 0 {
		return nil
	}

	wsMessage := &UnifiedMessage{
		ID:          event.ID,
		Type:        "event",
		Data:        event.Data,
		Timestamp:   event.Timestamp.Format(time.RFC3339),
		ServiceType: event.ServiceType,
		ServiceID:   event.ServiceID,
		Category:    string(event.Category),
		Metadata: map[string]interface{}{
			"event_type": event.Type,
			"source":     event.Source,
			"category":   event.Category,
		},
	}

	for connectionID := range allSubscribers {
		conn := wh.connections[connectionID]
		if conn != nil {
			wh.sendMessage(conn, wsMessage)
		}
	}

	return nil
}

func (wh *WebSocketEventHandler) broadcastConsciousnessUpdate(message *UnifiedMessage) error {
	// Get consciousness subscribers
	subscribers := wh.subscriptions[events.EventTypeConsciousnessUpdated]

	if len(subscribers) == 0 {
		return nil
	}

	// Send to all consciousness subscribers
	for connectionID := range subscribers {
		conn := wh.connections[connectionID]
		if conn != nil {
			// Clone message for each connection to avoid race conditions
			connMessage := &UnifiedMessage{
				ID:        message.ID,
				Type:      message.Type,
				Data:      message.Data,
				Timestamp: message.Timestamp,
				Metadata:  message.Metadata,
			}

			// Add connection-specific metadata
			if connMessage.Metadata == nil {
				connMessage.Metadata = make(map[string]interface{})
			}
			connMessage.Metadata["connection_id"] = connectionID

			wh.sendMessage(conn, connMessage)
		}
	}

	log.Printf("🧠 WebSocket Handler: Broadcasted consciousness update to %d subscribers", len(subscribers))
	return nil
}

func (wh *WebSocketEventHandler) sendMessage(conn *WebSocketConnection, msg *UnifiedMessage) error {
	select {
	case conn.Send <- msg:
		return nil
	default:
		return fmt.Errorf("connection send channel is full")
	}
}

func (wh *WebSocketEventHandler) sendError(conn *WebSocketConnection, messageID, errorMsg string) error {
	return wh.sendMessage(conn, &UnifiedMessage{
		ID:        messageID,
		Type:      "error",
		Error:     errorMsg,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

func (wh *WebSocketEventHandler) getConnectionIDFromMetadata(event events.Event) string {
	if event.Metadata != nil {
		if connID, ok := event.Metadata["connection_id"].(string); ok {
			return connID
		}
	}
	return ""
}

// GetMetrics returns WebSocket-specific metrics
func (wh *WebSocketEventHandler) GetMetrics() map[string]interface{} {
	// Count service subscriptions
	serviceSubCount := 0
	for _, eventMap := range wh.serviceSubscriptions {
		for _, connMap := range eventMap {
			serviceSubCount += len(connMap)
		}
	}

	// Count category subscriptions
	categorySubCount := 0
	for _, connMap := range wh.categorySubscriptions {
		categorySubCount += len(connMap)
	}

	return map[string]interface{}{
		"active_connections":     len(wh.connections),
		"active_subscriptions":   len(wh.subscriptions),
		"service_subscriptions":  serviceSubCount,
		"category_subscriptions": categorySubCount,
		"handler_name":           wh.name,
		"subscription_breakdown": map[string]interface{}{
			"by_service":  len(wh.serviceSubscriptions),
			"by_category": len(wh.categorySubscriptions),
			"by_event":    len(wh.subscriptions),
		},
	}
}

// WebSocket event handlers

func (wh *WebSocketEventHandler) handleWebSocketMessageEvent(ctx context.Context, event events.Event) error {
	// Extract WebSocket message data from event
	wsMessageData, ok := event.Data.(events.WebSocketMessageData)
	if !ok {
		log.Printf("⚠️ WebSocket Handler: Invalid message data type in event %s, got %T", event.ID, event.Data)
		return fmt.Errorf("invalid WebSocket message data type")
	}

	log.Printf("🔌 WebSocket Handler: Processing message event %s for connection %s", event.ID, wsMessageData.ConnectionID)

	// Find the connection
	conn := wh.connections[wsMessageData.ConnectionID]
	if conn == nil {
		log.Printf("⚠️ WebSocket Handler: Connection %s not found for message event", wsMessageData.ConnectionID)
		return nil // Connection might have been closed
	}

	// Create UnifiedMessage from the event data
	unifiedMsg := UnifiedMessage{
		ID:        wsMessageData.MessageID,
		Type:      wsMessageData.MessageType,
		Method:    wsMessageData.Method,
		Data:      wsMessageData.Data,
		Timestamp: event.Timestamp.Format(time.RFC3339),
	}

	log.Printf("🔌 WebSocket Handler: Processing %s message: %s", unifiedMsg.Type, unifiedMsg.Method)

	// Pass correlation ID from the original event to preserve EventRouter correlation
	unifiedMsg.CorrelationID = event.CorrelationID

	// Process the WebSocket message
	return wh.HandleWebSocketMessage(conn, &unifiedMsg)
}

func (wh *WebSocketEventHandler) handleWebSocketConnectedEvent(ctx context.Context, event events.Event) error {
	log.Printf("🔌 WebSocket Handler: Connection event - %s", event.Type)

	// Extract connection info from event data
	if connData, ok := event.Data.(events.WebSocketMessageData); ok {
		log.Printf("🔌 WebSocket Handler: Connection %s established", connData.ConnectionID)
	}

	return nil
}

func (wh *WebSocketEventHandler) handleWebSocketDisconnectedEvent(ctx context.Context, event events.Event) error {
	log.Printf("🔌 WebSocket Handler: Disconnection event - %s", event.Type)

	// Extract connection info from event data
	if connData, ok := event.Data.(events.WebSocketMessageData); ok {
		log.Printf("🔌 WebSocket Handler: Connection %s disconnected", connData.ConnectionID)
		// Clean up the connection
		wh.UnregisterConnection(connData.ConnectionID)
	}

	return nil
}
