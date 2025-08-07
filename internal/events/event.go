package events

import (
	"context"
	"strings"
	"time"
)

// Event represents a domain event in the system
type Event struct {
	// Core event identification
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Source        string                 `json:"source"`
	Target        string                 `json:"target,omitempty"`
	
	// Service categorization
	ServiceType   string                 `json:"service_type,omitempty"`
	ServiceID     string                 `json:"service_id,omitempty"`
	Category      EventCategory          `json:"category"`
	
	// Event payload and metadata
	Data          interface{}            `json:"data"`
	Metadata      map[string]interface{} `json:"metadata"`
	
	// Timing and correlation
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	
	// Event processing
	Version       int                    `json:"version"`
	Priority      Priority               `json:"priority"`
}

// Priority defines event processing priority
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// EventCategory categorizes events by service type
type EventCategory string

const (
	CategoryCore          EventCategory = "core"          // Core system events
	CategoryAgent         EventCategory = "agent"         // Agent-related events
	CategoryModel         EventCategory = "model"         // Model-related events
	CategorySensor        EventCategory = "sensor"        // Sensor-related events
	CategoryTool          EventCategory = "tool"          // Tool-related events
	CategoryRAG           EventCategory = "rag"           // RAG system events
	CategoryP2P           EventCategory = "p2p"           // P2P network events
	CategoryConsensus     EventCategory = "consensus"     // Consensus-related events
	CategoryEconomy       EventCategory = "economy"       // Economic engine events
	CategoryWebSocket     EventCategory = "websocket"     // WebSocket communication events
	CategoryConsciousness EventCategory = "consciousness" // Consciousness system events
	CategoryAPI           EventCategory = "api"           // API-related events
	CategorySystem        EventCategory = "system"        // System infrastructure events
)

// EventHandler defines the interface for handling events
type EventHandler interface {
	// Handle processes an event
	Handle(ctx context.Context, event Event) error
	
	// SubscribedEvents returns the event types this handler subscribes to
	SubscribedEvents() []string
	
	// HandlerName returns a unique identifier for this handler
	HandlerName() string
}

// EventData provides strongly-typed event data structures
type EventData interface{}

// Common Event Types
const (
	// Command Events (Request something to happen)
	EventTypeQuerySubmit       = "query.submit"
	EventTypeQueryCancel       = "query.cancel"
	EventTypeModelDownload     = "model.download"
	EventTypeModelDelete       = "model.delete"
	EventTypeAgentCreate       = "agent.create"
	EventTypeAgentDelete       = "agent.delete"
	EventTypeSubscribeRequest  = "subscription.request"
	EventTypeUnsubscribeRequest = "subscription.cancel"
	
	// API Command Events
	EventTypeGetModels         = "api.get_models"
	EventTypeGetAgents         = "api.get_agents"
	EventTypeGetStatus         = "api.get_status"
	EventTypeGetMetrics        = "api.get_metrics"
	EventTypeGetTransactions   = "api.get_transactions"
	EventTypeGetAccounts       = "api.get_accounts"
	EventTypeGetNodeInfo       = "api.get_node_info"
	EventTypeGetPeers          = "api.get_peers"
	EventTypeWalletBalance     = "api.wallet_balance"
	EventTypeWalletTransfer    = "api.wallet_transfer"
	EventTypeServiceRegister   = "api.service_register"
	EventTypeServiceDeregister = "api.service_deregister"
	EventTypeGetServices       = "api.get_services"
	
	// Domain Events (Something happened)
	EventTypeQueryStarted      = "query.started"
	EventTypeQueryProcessed    = "query.processed"
	EventTypeQueryFailed       = "query.failed"
	EventTypeQueryProgress     = "query.progress"
	EventTypeModelDownloaded   = "model.downloaded"
	EventTypeModelDeleted      = "model.deleted"
	EventTypeAgentRegistered   = "agent.registered"
	EventTypeAgentDeregistered = "agent.deregistered"
	EventTypeMetricsUpdated    = "metrics.updated"
	EventTypeConsciousnessUpdated = "consciousness.updated"
	
	// API Response Events
	EventTypeModelsData        = "api.models_data"
	EventTypeAgentsData        = "api.agents_data"
	EventTypeStatusData        = "api.status_data"
	EventTypeMetricsData       = "api.metrics_data"
	EventTypeTransactionsData  = "api.transactions_data"
	EventTypeAccountsData      = "api.accounts_data"
	EventTypeNodeInfoData      = "api.node_info_data"
	EventTypePeersData         = "api.peers_data"
	EventTypeWalletData        = "api.wallet_data"
	EventTypeServicesData      = "api.services_data"
	
	// Integration Events (Cross-boundary)
	EventTypeWebSocketMessage  = "websocket.message"
	EventTypeWebSocketConnected = "websocket.connected"
	EventTypeWebSocketDisconnected = "websocket.disconnected"
	EventTypeP2PMessage        = "p2p.message"
	EventTypeConsensusUpdate   = "consensus.update"
	
	// System Events (Infrastructure)
	EventTypeComponentStarted  = "system.component.started"
	EventTypeComponentStopped  = "system.component.stopped"
	EventTypeHealthCheck       = "system.health.check"
	EventTypeError             = "system.error"
)

// Event Data Structures

// QueryData represents data for query-related events
type QueryData struct {
	QueryID     string                 `json:"query_id"`
	Text        string                 `json:"text"`
	UseRAG      bool                   `json:"use_rag"`
	UserID      string                 `json:"user_id"`
	ConnectionID string                `json:"connection_id"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// QueryResultData represents query processing results
type QueryResultData struct {
	QueryID     string                 `json:"query_id"`
	Result      string                 `json:"result"`
	Source      string                 `json:"source"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// ModelData represents model-related event data
type ModelData struct {
	ModelName   string `json:"model_name"`
	Action      string `json:"action"`
	Progress    int    `json:"progress,omitempty"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
}

// AgentData represents agent-related event data
type AgentData struct {
	AgentID     string                 `json:"agent_id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	Capabilities []string              `json:"capabilities,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SubscriptionData represents subscription requests
type SubscriptionData struct {
	EventType    string `json:"event_type"`
	ConnectionID string `json:"connection_id"`
	UserID       string `json:"user_id"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
}

// MetricsData represents system metrics
type MetricsData struct {
	Type        string                 `json:"type"`
	Values      map[string]interface{} `json:"values"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ConsciousnessData represents consciousness state updates
type ConsciousnessData struct {
	State       map[string]interface{} `json:"state"`
	Metrics     map[string]interface{} `json:"metrics"`
	WorkingMemory map[string]interface{} `json:"working_memory"`
	NodeID      string                 `json:"node_id"`
}

// WebSocketMessageData represents WebSocket message events
type WebSocketMessageData struct {
	ConnectionID string      `json:"connection_id"`
	MessageID    string      `json:"message_id"`
	MessageType  string      `json:"message_type"`
	Method       string      `json:"method,omitempty"`
	Data         interface{} `json:"data"`
	UserID       string      `json:"user_id"`
}

// Helper functions for creating events

// NewEvent creates a new event with required fields
func NewEvent(eventType, source string, data interface{}) Event {
	return Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    source,
		Category:  determineEventCategory(eventType, source),
		Data:      data,
		Metadata:  make(map[string]interface{}),
		Timestamp: time.Now(),
		Version:   1,
		Priority:  PriorityNormal,
	}
}

// NewEventWithCorrelation creates a new event with correlation ID
func NewEventWithCorrelation(eventType, source string, data interface{}, correlationID string) Event {
	event := NewEvent(eventType, source, data)
	event.CorrelationID = correlationID
	return event
}

// NewServiceEvent creates a new event with service categorization
func NewServiceEvent(eventType, source, serviceType, serviceID string, data interface{}) Event {
	event := NewEvent(eventType, source, data)
	event.ServiceType = serviceType
	event.ServiceID = serviceID
	event.Category = mapServiceTypeToCategory(serviceType)
	return event
}

// NewServiceEventWithCorrelation creates a new service event with correlation ID
func NewServiceEventWithCorrelation(eventType, source, serviceType, serviceID string, data interface{}, correlationID string) Event {
	event := NewServiceEvent(eventType, source, serviceType, serviceID, data)
	event.CorrelationID = correlationID
	return event
}

// WithTarget sets the target for an event
func (e Event) WithTarget(target string) Event {
	e.Target = target
	return e
}

// WithPriority sets the priority for an event
func (e Event) WithPriority(priority Priority) Event {
	e.Priority = priority
	return e
}

// WithMetadata adds metadata to an event
func (e Event) WithMetadata(key string, value interface{}) Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithService sets the service information for an event
func (e Event) WithService(serviceType, serviceID string) Event {
	e.ServiceType = serviceType
	e.ServiceID = serviceID
	e.Category = mapServiceTypeToCategory(serviceType)
	return e
}

// WithCategory sets the event category
func (e Event) WithCategory(category EventCategory) Event {
	e.Category = category
	return e
}

// IsServiceEvent returns true if this event is associated with a service
func (e Event) IsServiceEvent() bool {
	return e.ServiceType != "" || e.ServiceID != ""
}

// GetServiceInfo returns service type and ID
func (e Event) GetServiceInfo() (string, string) {
	return e.ServiceType, e.ServiceID
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return "evt_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// determineEventCategory automatically determines the category based on event type and source
func determineEventCategory(eventType, source string) EventCategory {
	// Check event type patterns
	switch {
	case strings.Contains(eventType, "agent"):
		return CategoryAgent
	case strings.Contains(eventType, "model"):
		return CategoryModel
	case strings.Contains(eventType, "query") || strings.Contains(eventType, "rag"):
		return CategoryRAG
	case strings.Contains(eventType, "websocket"):
		return CategoryWebSocket
	case strings.Contains(eventType, "consciousness"):
		return CategoryConsciousness
	case strings.Contains(eventType, "p2p"):
		return CategoryP2P
	case strings.Contains(eventType, "consensus"):
		return CategoryConsensus
	case strings.Contains(eventType, "api"):
		return CategoryAPI
	case strings.Contains(eventType, "system") || strings.Contains(eventType, "component") || strings.Contains(eventType, "health"):
		return CategorySystem
	case strings.Contains(eventType, "wallet") || strings.Contains(eventType, "transaction") || strings.Contains(eventType, "account"):
		return CategoryEconomy
	case strings.Contains(eventType, "service"):
		return CategoryCore
	}
	
	// Check source patterns
	switch source {
	case "agent", "agent_registry", "agent_manager":
		return CategoryAgent
	case "model", "model_manager", "ollama":
		return CategoryModel
	case "rag", "rag_system":
		return CategoryRAG
	case "websocket", "websocket_manager":
		return CategoryWebSocket
	case "consciousness", "consciousness_system":
		return CategoryConsciousness
	case "p2p", "p2p_node":
		return CategoryP2P
	case "consensus", "consensus_service":
		return CategoryConsensus
	case "api", "api_server":
		return CategoryAPI
	case "system", "event_bus", "health":
		return CategorySystem
	case "economy", "economic_engine":
		return CategoryEconomy
	default:
		return CategoryCore
	}
}

// mapServiceTypeToCategory maps service types to event categories
func mapServiceTypeToCategory(serviceType string) EventCategory {
	switch serviceType {
	case "agent":
		return CategoryAgent
	case "model":
		return CategoryModel
	case "sensor":
		return CategorySensor
	case "tool":
		return CategoryTool
	default:
		return CategoryCore
	}
}

