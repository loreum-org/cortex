package types

import (
	"time"
)

// ServiceType represents the type of service offered by a node
type ServiceType string

const (
	ServiceTypeAgent  ServiceType = "agent"
	ServiceTypeModel  ServiceType = "model"
	ServiceTypeSensor ServiceType = "sensor"
	ServiceTypeTool   ServiceType = "tool"
)

// ServiceCapability represents specific capabilities within a service type
type ServiceCapability struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  map[string]string `json:"parameters"`
	Version     string            `json:"version"`
}

// ServiceOffering represents a service offered by a node
type ServiceOffering struct {
	ID           string               `json:"id"`
	NodeID       string               `json:"node_id"`
	Type         ServiceType          `json:"type"`
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Capabilities []ServiceCapability  `json:"capabilities"`
	Pricing      *ServicePricing      `json:"pricing"`
	Metadata     map[string]interface{} `json:"metadata"`
	Status       ServiceStatus        `json:"status"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
	LastSeen     time.Time            `json:"last_seen"`
}

// ServicePricing defines pricing for a service
type ServicePricing struct {
	BasePrice      string `json:"base_price"`      // Base price in smallest token unit
	PricePerToken  string `json:"price_per_token"` // Price per token for models
	PricePerCall   string `json:"price_per_call"`  // Price per call for tools/sensors
	PricePerMinute string `json:"price_per_minute"` // Price per minute for long-running tasks
}

// ServiceStatus represents the current status of a service
type ServiceStatus string

const (
	ServiceStatusActive      ServiceStatus = "active"
	ServiceStatusInactive    ServiceStatus = "inactive"
	ServiceStatusMaintenance ServiceStatus = "maintenance"
	ServiceStatusError       ServiceStatus = "error"
)

// ServiceQuery represents a query that requires specific services
type ServiceQuery struct {
	ID               string            `json:"id"`
	UserID           string            `json:"user_id"`
	RequiredServices []ServiceRequirement `json:"required_services"`
	Content          string            `json:"content"`
	Metadata         map[string]interface{} `json:"metadata"`
	MaxPrice         string            `json:"max_price"`
	Timeout          time.Duration     `json:"timeout"`
	CreatedAt        time.Time         `json:"created_at"`
}

// ServiceRequirement specifies what services are needed for a query
type ServiceRequirement struct {
	Type         ServiceType `json:"type"`
	Capabilities []string    `json:"capabilities"`
	MinVersion   string      `json:"min_version,omitempty"`
	Preferences  map[string]interface{} `json:"preferences,omitempty"`
}

// ServiceAttestation represents proof of service execution for payments
type ServiceAttestation struct {
	ID          string    `json:"id"`
	QueryID     string    `json:"query_id"`
	NodeID      string    `json:"node_id"`
	ServiceID   string    `json:"service_id"`
	UserID      string    `json:"user_id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	InputHash   string    `json:"input_hash"`   // Hash of input data
	OutputHash  string    `json:"output_hash"`  // Hash of output data
	Success     bool      `json:"success"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
	TokensUsed  int64     `json:"tokens_used,omitempty"`
	ComputeTime int64     `json:"compute_time_ms"`
	Cost        string    `json:"cost"`
	Signature   string    `json:"signature"`   // Node's signature
	CreatedAt   time.Time `json:"created_at"`
}

// ServiceRegistry maintains the registry of all services in the network
type ServiceRegistry struct {
	Services    map[string]*ServiceOffering `json:"services"`     // service_id -> service
	NodeIndex   map[string][]string         `json:"node_index"`   // node_id -> []service_id
	TypeIndex   map[ServiceType][]string    `json:"type_index"`   // service_type -> []service_id
	UpdatedAt   time.Time                   `json:"updated_at"`
}

// QueryRouter handles routing queries to appropriate nodes
type ServiceQueryRouter struct {
	Registry      *ServiceRegistry `json:"registry"`
	RoutingPolicy RoutingPolicy    `json:"routing_policy"`
}

// RoutingPolicy defines how queries are routed to nodes
type RoutingPolicy struct {
	Strategy          RoutingStrategy `json:"strategy"`
	ReputationWeight  float64         `json:"reputation_weight"`
	PriceWeight       float64         `json:"price_weight"`
	ResponseTimeWeight float64        `json:"response_time_weight"`
	LoadWeight        float64         `json:"load_weight"`
}

// RoutingStrategy defines different strategies for routing
type RoutingStrategy string

const (
	RoutingStrategyRoundRobin     RoutingStrategy = "round_robin"
	RoutingStrategyLowestPrice    RoutingStrategy = "lowest_price"
	RoutingStrategyHighestRep     RoutingStrategy = "highest_reputation"
	RoutingStrategyFastestResponse RoutingStrategy = "fastest_response"
	RoutingStrategyWeighted       RoutingStrategy = "weighted"
)

// ServiceRegistryTransaction represents a transaction for service registration/deregistration
type ServiceRegistryTransaction struct {
	TransactionBase
	Action    ServiceAction    `json:"action"`
	ServiceID string           `json:"service_id"`
	NodeID    string           `json:"node_id"`
	Service   *ServiceOffering `json:"service,omitempty"`
}

// ServiceAction represents actions that can be performed on services
type ServiceAction string

const (
	ServiceActionRegister   ServiceAction = "register"
	ServiceActionDeregister ServiceAction = "deregister"
	ServiceActionUpdate     ServiceAction = "update"
	ServiceActionHeartbeat  ServiceAction = "heartbeat"
)

// TransactionBase provides common fields for all transaction types
type TransactionBase struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp int64     `json:"timestamp"`
	Signature string    `json:"signature"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}