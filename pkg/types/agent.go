package types

import (
	"context"
)

// Capability represents an agent's capability
type Capability string

// Metrics represents performance metrics for an agent
type Metrics struct {
	ResponseTime   float64 `json:"response_time"`
	SuccessRate    float64 `json:"success_rate"`
	ErrorRate      float64 `json:"error_rate"`
	RequestsPerMin int     `json:"requests_per_min"`
}

// Query represents a query to be processed by an agent
type Query struct {
	ID        string            `json:"id"`
	Text      string            `json:"text"`
	Type      string            `json:"type"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp int64             `json:"timestamp"`
}

// Response represents a response from an agent
type Response struct {
	QueryID   string            `json:"query_id"`
	Text      string            `json:"text"`
	Data      interface{}       `json:"data"`
	Metadata  map[string]string `json:"metadata"`
	Status    string            `json:"status"`
	Timestamp int64             `json:"timestamp"`
}

// Agent interface defines the methods that all agents must implement
type Agent interface {
	Process(context.Context, *Query) (*Response, error)
	GetCapabilities() []Capability
	GetPerformanceMetrics() Metrics
}

// QueryRouter routes queries to appropriate agents
type QueryRouter struct {
	Agents       map[string]Agent  `json:"agents"`
	RoutingRules map[string]string `json:"routing_rules"`
}

// TaskOrchestrator orchestrates tasks across multiple agents
type TaskOrchestrator struct {
	TaskQueue     chan *Query     `json:"-"`
	RunningTasks  map[string]bool `json:"running_tasks"`
	MaxConcurrent int             `json:"max_concurrent"`
}
