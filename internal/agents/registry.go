package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/pkg/types"
)

// AgentType represents the type of agent
type AgentType string

const (
	AgentTypeSolver      AgentType = "solver"
	AgentTypeSensor      AgentType = "sensor"
	AgentTypeAnalyzer    AgentType = "analyzer"
	AgentTypeGenerator   AgentType = "generator"
	AgentTypeCoordinator AgentType = "coordinator"
)

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	AgentStatusActive       AgentStatus = "active"
	AgentStatusInactive     AgentStatus = "inactive"
	AgentStatusInitializing AgentStatus = "initializing"
	AgentStatusError        AgentStatus = "error"
	AgentStatusStopping     AgentStatus = "stopping"
)

// AgentCapability represents a specific capability of an agent
type AgentCapability struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Parameters  map[string]interface{} `json:"parameters"`
	InputTypes  []string               `json:"input_types"`
	OutputTypes []string               `json:"output_types"`
}

// AgentMetrics represents comprehensive performance metrics for an agent
type AgentMetrics struct {
	ResponseTime   float64    `json:"response_time"`
	SuccessRate    float64    `json:"success_rate"`
	ErrorRate      float64    `json:"error_rate"`
	RequestsPerMin int        `json:"requests_per_min"`
	TotalQueries   int64      `json:"total_queries"`
	Uptime         float64    `json:"uptime"`
	LastQueryAt    *time.Time `json:"last_query_at,omitempty"`
	StartTime      time.Time  `json:"start_time"`
	ErrorCount     int64      `json:"error_count"`
	AvgProcessTime float64    `json:"avg_process_time"`
}

// AgentInfo contains metadata about an agent
type AgentInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         AgentType              `json:"type"`
	Version      string                 `json:"version"`
	Status       AgentStatus            `json:"status"`
	Capabilities []AgentCapability      `json:"capabilities"`
	ModelID      string                 `json:"model_id,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// StandardAgent interface defines the standardized methods all agents must implement
type StandardAgent interface {
	// Core processing
	Process(ctx context.Context, query *types.Query) (*types.Response, error)

	// Agent lifecycle
	Initialize(ctx context.Context, config map[string]interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error

	// Information and capabilities
	GetInfo() AgentInfo
	GetCapabilities() []AgentCapability
	GetMetrics() AgentMetrics
	GetStatus() AgentStatus

	// Health and monitoring
	HealthCheck(ctx context.Context) error
	UpdateConfig(config map[string]interface{}) error

	// Event handling
	OnEvent(event *AgentEvent) error
}

// AgentEvent represents events in the agent system
type AgentEvent struct {
	Type      string                 `json:"type"`
	AgentID   string                 `json:"agent_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// AgentRegistry manages all registered agents
type AgentRegistry struct {
	agents  map[string]StandardAgent
	info    map[string]AgentInfo
	metrics map[string]AgentMetrics
	mu      sync.RWMutex

	// Event channels
	eventChan chan *AgentEvent
	stopChan  chan struct{}

	// Monitoring
	monitoringEnabled   bool
	healthCheckInterval time.Duration

	// AGI Integration
	agiConnectionManager  *AGIConnectionManager
	agiIntegrationEnabled bool
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() *AgentRegistry {
	registry := &AgentRegistry{
		agents:                make(map[string]StandardAgent),
		info:                  make(map[string]AgentInfo),
		metrics:               make(map[string]AgentMetrics),
		eventChan:             make(chan *AgentEvent, 1000),
		stopChan:              make(chan struct{}),
		monitoringEnabled:     true,
		healthCheckInterval:   30 * time.Second,
		agiIntegrationEnabled: false,
	}

	// Initialize AGI connection manager
	registry.agiConnectionManager = NewAGIConnectionManager(registry)

	return registry
}

// RegisterAgent registers a new agent with the registry
func (r *AgentRegistry) RegisterAgent(agent StandardAgent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := agent.GetInfo()

	// Validate agent info
	if info.ID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	if _, exists := r.agents[info.ID]; exists {
		return fmt.Errorf("agent with ID %s already registered", info.ID)
	}

	// Initialize agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := agent.Initialize(ctx, info.Config); err != nil {
		return fmt.Errorf("failed to initialize agent %s: %w", info.ID, err)
	}

	// Register agent
	r.agents[info.ID] = agent
	r.info[info.ID] = info
	r.metrics[info.ID] = agent.GetMetrics()

	// Start agent
	if err := agent.Start(ctx); err != nil {
		delete(r.agents, info.ID)
		delete(r.info, info.ID)
		delete(r.metrics, info.ID)
		return fmt.Errorf("failed to start agent %s: %w", info.ID, err)
	}

	// Emit registration event
	r.emitEvent(&AgentEvent{
		Type:      "agent_registered",
		AgentID:   info.ID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"name": info.Name,
			"type": info.Type,
		},
	})

	log.Printf("Agent registered: %s (%s)", info.Name, info.ID)
	return nil
}

// DeregisterAgent removes an agent from the registry
func (r *AgentRegistry) DeregisterAgent(agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Stop agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := agent.Stop(ctx); err != nil {
		log.Printf("Warning: failed to stop agent %s cleanly: %v", agentID, err)
	}

	// Remove from registry
	delete(r.agents, agentID)
	delete(r.info, agentID)
	delete(r.metrics, agentID)

	// Emit deregistration event
	r.emitEvent(&AgentEvent{
		Type:      "agent_deregistered",
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{},
	})

	log.Printf("Agent deregistered: %s", agentID)
	return nil
}

// GetAgent retrieves an agent by ID
func (r *AgentRegistry) GetAgent(agentID string) (StandardAgent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}

	return agent, nil
}

// GetAgentInfo retrieves agent information by ID
func (r *AgentRegistry) GetAgentInfo(agentID string) (AgentInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.info[agentID]
	if !exists {
		return AgentInfo{}, fmt.Errorf("agent %s not found", agentID)
	}

	return info, nil
}

// GetAllAgents returns all registered agents' information
func (r *AgentRegistry) GetAllAgents() []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]AgentInfo, 0, len(r.info))
	for _, info := range r.info {
		agents = append(agents, info)
	}

	return agents
}

// GetAgentsByType returns all agents of a specific type
func (r *AgentRegistry) GetAgentsByType(agentType AgentType) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []AgentInfo
	for _, info := range r.info {
		if info.Type == agentType {
			agents = append(agents, info)
		}
	}

	return agents
}

// GetAgentsByCapability returns all agents that have a specific capability
func (r *AgentRegistry) GetAgentsByCapability(capabilityName string) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var agents []AgentInfo
	for _, info := range r.info {
		for _, cap := range info.Capabilities {
			if cap.Name == capabilityName {
				agents = append(agents, info)
				break
			}
		}
	}

	return agents
}

// RouteQuery routes a query to the most appropriate agent
func (r *AgentRegistry) RouteQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	// Find best agent for the query
	agentID, err := r.findBestAgent(query)
	if err != nil {
		return nil, fmt.Errorf("failed to find suitable agent: %w", err)
	}

	agent, err := r.GetAgent(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent %s: %w", agentID, err)
	}

	// Process query
	startTime := time.Now()
	response, err := agent.Process(ctx, query)
	processingTime := time.Since(startTime)

	// Update metrics
	r.updateAgentMetrics(agentID, processingTime, err == nil)

	// Emit query event
	r.emitEvent(&AgentEvent{
		Type:      "query_processed",
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"query_id":        query.ID,
			"processing_time": processingTime.Seconds(),
			"success":         err == nil,
		},
	})

	return response, err
}

// findBestAgent finds the most suitable agent for a query
func (r *AgentRegistry) findBestAgent(query *types.Query) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Simple routing logic - can be enhanced with more sophisticated algorithms
	// For now, prioritize by type and availability

	var bestAgent string
	var bestScore float64

	for agentID, info := range r.info {
		// Get current status from the agent itself, not from stored info
		agent, exists := r.agents[agentID]
		if !exists {
			continue
		}

		currentStatus := agent.GetStatus()
		if currentStatus != AgentStatusActive {
			log.Printf("Agent %s is not active (status: %s), skipping", agentID, currentStatus)
			continue
		}

		score := r.calculateAgentScore(info, query)
		log.Printf("Agent %s score for query type '%s': %.2f", agentID, query.Type, score)
		if score > bestScore {
			bestScore = score
			bestAgent = agentID
		}
	}

	if bestAgent == "" {
		log.Printf("No suitable agent found for query type: %s (checked %d agents)", query.Type, len(r.info))
		return "", fmt.Errorf("no suitable agent found for query type: %s", query.Type)
	}

	log.Printf("Selected agent %s with score %.2f for query type %s", bestAgent, bestScore, query.Type)
	return bestAgent, nil
}

// calculateAgentScore calculates a score for how well an agent fits a query
func (r *AgentRegistry) calculateAgentScore(info AgentInfo, query *types.Query) float64 {
	score := 0.0

	// Type-based scoring
	switch query.Type {
	case "user_query", "natural_language", "question", "text":
		if info.Type == AgentTypeSolver {
			score += 10.0
		}
	case "analysis", "metrics":
		if info.Type == AgentTypeAnalyzer {
			score += 10.0
		}
	case "monitoring", "sensor_data":
		if info.Type == AgentTypeSensor {
			score += 10.0
		}
	case "generation", "content":
		if info.Type == AgentTypeGenerator {
			score += 10.0
		}
	case "coordination", "workflow":
		if info.Type == AgentTypeCoordinator {
			score += 10.0
		}
	}

	// Capability-based scoring
	for _, cap := range info.Capabilities {
		for _, inputType := range cap.InputTypes {
			if inputType == query.Type {
				score += 5.0
			}
		}
	}

	// Performance-based scoring
	if metrics, exists := r.metrics[info.ID]; exists {
		score += metrics.SuccessRate / 10.0 // 0-10 points based on success rate
		score -= metrics.ResponseTime       // Penalty for slow response
	}

	return score
}

// updateAgentMetrics updates performance metrics for an agent
func (r *AgentRegistry) updateAgentMetrics(agentID string, processingTime time.Duration, success bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	metrics, exists := r.metrics[agentID]
	if !exists {
		return
	}

	// Update metrics
	metrics.TotalQueries++
	now := time.Now()
	metrics.LastQueryAt = &now

	// Update success/error rates
	if success {
		metrics.SuccessRate = (metrics.SuccessRate*float64(metrics.TotalQueries-1) + 100.0) / float64(metrics.TotalQueries)
	} else {
		metrics.ErrorCount++
		metrics.SuccessRate = (metrics.SuccessRate*float64(metrics.TotalQueries-1) + 0.0) / float64(metrics.TotalQueries)
	}

	metrics.ErrorRate = 100.0 - metrics.SuccessRate

	// Update response time
	processingSecs := processingTime.Seconds()
	metrics.AvgProcessTime = (metrics.AvgProcessTime*float64(metrics.TotalQueries-1) + processingSecs) / float64(metrics.TotalQueries)
	metrics.ResponseTime = metrics.AvgProcessTime

	// Update uptime
	uptime := time.Since(metrics.StartTime)
	metrics.Uptime = uptime.Hours()

	r.metrics[agentID] = metrics
}

// StartMonitoring starts the agent monitoring routine
func (r *AgentRegistry) StartMonitoring() {
	if !r.monitoringEnabled {
		return
	}

	go r.monitoringLoop()
	log.Printf("Agent monitoring started")
}

// StopMonitoring stops the agent monitoring routine
func (r *AgentRegistry) StopMonitoring() {
	close(r.stopChan)
	log.Printf("Agent monitoring stopped")
}

// monitoringLoop runs periodic health checks and metrics updates
func (r *AgentRegistry) monitoringLoop() {
	ticker := time.NewTicker(r.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.performHealthChecks()
		case event := <-r.eventChan:
			r.handleEvent(event)
		case <-r.stopChan:
			return
		}
	}
}

// performHealthChecks performs health checks on all agents
func (r *AgentRegistry) performHealthChecks() {
	r.mu.RLock()
	agents := make(map[string]StandardAgent)
	for id, agent := range r.agents {
		agents[id] = agent
	}
	r.mu.RUnlock()

	for agentID, agent := range agents {
		go func(id string, a StandardAgent) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			err := a.HealthCheck(ctx)
			r.updateAgentStatus(id, err)
		}(agentID, agent)
	}
}

// updateAgentStatus updates an agent's status based on health check results
func (r *AgentRegistry) updateAgentStatus(agentID string, healthErr error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, exists := r.info[agentID]
	if !exists {
		return
	}

	oldStatus := info.Status

	if healthErr != nil {
		info.Status = AgentStatusError
		log.Printf("Agent %s health check failed: %v", agentID, healthErr)
	} else if info.Status == AgentStatusError {
		info.Status = AgentStatusActive
		log.Printf("Agent %s recovered from error state", agentID)
	}

	if oldStatus != info.Status {
		info.UpdatedAt = time.Now()
		r.info[agentID] = info

		r.emitEvent(&AgentEvent{
			Type:      "agent_status_changed",
			AgentID:   agentID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"old_status": oldStatus,
				"new_status": info.Status,
			},
		})
	}
}

// emitEvent emits an event to the event channel
func (r *AgentRegistry) emitEvent(event *AgentEvent) {
	select {
	case r.eventChan <- event:
	default:
		log.Printf("Warning: agent event channel full, dropping event")
	}
}

// handleEvent handles agent events
func (r *AgentRegistry) handleEvent(event *AgentEvent) {
	// Log event
	eventData, _ := json.Marshal(event)
	log.Printf("Agent event: %s", string(eventData))

	// Additional event handling can be added here
}

// GetRegistryStats returns overall registry statistics
func (r *AgentRegistry) GetRegistryStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := map[string]interface{}{
		"total_agents":      len(r.agents),
		"active_agents":     0,
		"error_agents":      0,
		"total_queries":     int64(0),
		"avg_success_rate":  0.0,
		"avg_response_time": 0.0,
	}

	var totalSuccessRate, totalResponseTime float64
	activeCount := 0

	for _, info := range r.info {
		switch info.Status {
		case AgentStatusActive:
			activeCount++
		case AgentStatusError:
			stats["error_agents"] = stats["error_agents"].(int) + 1
		}
	}

	for _, metrics := range r.metrics {
		stats["total_queries"] = stats["total_queries"].(int64) + metrics.TotalQueries
		totalSuccessRate += metrics.SuccessRate
		totalResponseTime += metrics.ResponseTime
	}

	stats["active_agents"] = activeCount

	if len(r.metrics) > 0 {
		stats["avg_success_rate"] = totalSuccessRate / float64(len(r.metrics))
		stats["avg_response_time"] = totalResponseTime / float64(len(r.metrics))
	}

	return stats
}

// AGI Integration Methods

// EnableAGIIntegration enables AGI integration for all agents
func (r *AgentRegistry) EnableAGIIntegration(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.agiIntegrationEnabled {
		return fmt.Errorf("AGI integration is already enabled")
	}

	// Start AGI connection manager
	if err := r.agiConnectionManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start AGI connection manager: %w", err)
	}

	r.agiIntegrationEnabled = true

	// Connect all existing agents to AGI
	for agentID := range r.agents {
		if err := r.connectAgentToAGI(agentID); err != nil {
			log.Printf("Warning: Failed to connect agent %s to AGI: %v", agentID, err)
		}
	}

	log.Printf("AGI integration enabled for agent registry")
	return nil
}

// DisableAGIIntegration disables AGI integration
func (r *AgentRegistry) DisableAGIIntegration() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.agiIntegrationEnabled {
		return nil
	}

	// Disconnect all agents from AGI
	for agentID := range r.agents {
		r.disconnectAgentFromAGI(agentID)
	}

	// Stop AGI connection manager
	if err := r.agiConnectionManager.Stop(); err != nil {
		log.Printf("Warning: Failed to stop AGI connection manager: %v", err)
	}

	r.agiIntegrationEnabled = false

	log.Printf("AGI integration disabled for agent registry")
	return nil
}

// SetAGIBridge sets the AGI bridge for integration
func (r *AgentRegistry) SetAGIBridge(bridge interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Type assertion would be done here based on the actual bridge interface
	// For now, this is a placeholder for setting up the bridge connection

	log.Printf("AGI bridge configured for agent registry")
	return nil
}

// connectAgentToAGI connects an agent to the AGI system
func (r *AgentRegistry) connectAgentToAGI(agentID string) error {
	if !r.agiIntegrationEnabled {
		return nil
	}

	// Create AGI connection for the agent
	// This would be implemented based on the actual AGI bridge interface

	log.Printf("Connected agent %s to AGI system", agentID)
	return nil
}

// disconnectAgentFromAGI disconnects an agent from the AGI system
func (r *AgentRegistry) disconnectAgentFromAGI(agentID string) {
	if err := r.agiConnectionManager.UnregisterAGIConnection(agentID); err != nil {
		log.Printf("Warning: Failed to disconnect agent %s from AGI: %v", agentID, err)
	}
}

// StreamToAGI streams data from an agent to AGI
func (r *AgentRegistry) StreamToAGI(agentID string, data *AGIStreamData) error {
	if !r.agiIntegrationEnabled {
		return fmt.Errorf("AGI integration is not enabled")
	}

	return r.agiConnectionManager.StreamToAGI(agentID, data)
}

// BroadcastToAGI broadcasts data from all agents to AGI
func (r *AgentRegistry) BroadcastToAGI(data *AGIStreamData) error {
	if !r.agiIntegrationEnabled {
		return fmt.Errorf("AGI integration is not enabled")
	}

	return r.agiConnectionManager.BroadcastToAGI(data)
}

// GetAGIState returns the current AGI state
func (r *AgentRegistry) GetAGIState() *AGIState {
	if !r.agiIntegrationEnabled {
		return nil
	}

	return r.agiConnectionManager.GetAGIState()
}

// GetAGIConnectionManager returns the AGI connection manager
func (r *AgentRegistry) GetAGIConnectionManager() *AGIConnectionManager {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.agiConnectionManager
}

// IsAGIIntegrationEnabled returns whether AGI integration is enabled
func (r *AgentRegistry) IsAGIIntegrationEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.agiIntegrationEnabled
}

// RegisterCodeReflectionAgent registers a code reflection agent
func (ar *AgentRegistry) RegisterCodeReflectionAgent(ragSystem *rag.RAGSystem, modelManager *ai.ModelManager) error {
	// Create the code reflection agent
	codeAgent := NewCodeReflectionAgent(ragSystem, modelManager)

	// Register the agent
	if err := ar.RegisterAgent(codeAgent); err != nil {
		return fmt.Errorf("failed to register code reflection agent: %w", err)
	}

	log.Printf("Code Reflection Agent registered successfully")
	return nil
}

// Agent Lifecycle Management Methods

// StartAgent starts a specific agent by ID
func (r *AgentRegistry) StartAgent(agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Check current status
	currentStatus := agent.GetStatus()
	if currentStatus == AgentStatusActive {
		return fmt.Errorf("agent %s is already active", agentID)
	}

	// Update status to initializing
	info := r.info[agentID]
	info.Status = AgentStatusInitializing
	info.UpdatedAt = time.Now()
	r.info[agentID] = info

	// Start agent with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := agent.Start(ctx)
	if err != nil {
		// Update status to error
		info.Status = AgentStatusError
		info.UpdatedAt = time.Now()
		r.info[agentID] = info

		// Emit event
		r.emitEvent(&AgentEvent{
			Type:      "agent_start_failed",
			AgentID:   agentID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})

		return fmt.Errorf("failed to start agent %s: %w", agentID, err)
	}

	// Update status to active
	info.Status = AgentStatusActive
	info.UpdatedAt = time.Now()
	r.info[agentID] = info

	// Update metrics
	metrics := r.metrics[agentID]
	metrics.StartTime = time.Now()
	r.metrics[agentID] = metrics

	// Emit event
	r.emitEvent(&AgentEvent{
		Type:      "agent_started",
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"name": info.Name,
			"type": info.Type,
		},
	})

	log.Printf("Agent started: %s (%s)", info.Name, agentID)
	return nil
}

// StopAgent stops a specific agent by ID
func (r *AgentRegistry) StopAgent(agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Check current status
	currentStatus := agent.GetStatus()
	if currentStatus == AgentStatusInactive {
		return fmt.Errorf("agent %s is already inactive", agentID)
	}

	// Update status to stopping
	info := r.info[agentID]
	info.Status = AgentStatusStopping
	info.UpdatedAt = time.Now()
	r.info[agentID] = info

	// Stop agent with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := agent.Stop(ctx)
	if err != nil {
		// Update status to error
		info.Status = AgentStatusError
		info.UpdatedAt = time.Now()
		r.info[agentID] = info

		// Emit event
		r.emitEvent(&AgentEvent{
			Type:      "agent_stop_failed",
			AgentID:   agentID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})

		return fmt.Errorf("failed to stop agent %s: %w", agentID, err)
	}

	// Update status to inactive
	info.Status = AgentStatusInactive
	info.UpdatedAt = time.Now()
	r.info[agentID] = info

	// Emit event
	r.emitEvent(&AgentEvent{
		Type:      "agent_stopped",
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"name": info.Name,
			"type": info.Type,
		},
	})

	log.Printf("Agent stopped: %s (%s)", info.Name, agentID)
	return nil
}

// RestartAgent restarts a specific agent by ID
func (r *AgentRegistry) RestartAgent(agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Update status to initializing
	info := r.info[agentID]
	info.Status = AgentStatusInitializing
	info.UpdatedAt = time.Now()
	r.info[agentID] = info

	// Restart agent with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := agent.Restart(ctx)
	if err != nil {
		// Update status to error
		info.Status = AgentStatusError
		info.UpdatedAt = time.Now()
		r.info[agentID] = info

		// Emit event
		r.emitEvent(&AgentEvent{
			Type:      "agent_restart_failed",
			AgentID:   agentID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})

		return fmt.Errorf("failed to restart agent %s: %w", agentID, err)
	}

	// Update status to active
	info.Status = AgentStatusActive
	info.UpdatedAt = time.Now()
	r.info[agentID] = info

	// Reset metrics
	metrics := r.metrics[agentID]
	metrics.StartTime = time.Now()
	r.metrics[agentID] = metrics

	// Emit event
	r.emitEvent(&AgentEvent{
		Type:      "agent_restarted",
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"name": info.Name,
			"type": info.Type,
		},
	})

	log.Printf("Agent restarted: %s (%s)", info.Name, agentID)
	return nil
}

// StartAllAgents starts all inactive agents
func (r *AgentRegistry) StartAllAgents() error {
	r.mu.RLock()
	inactiveAgents := make([]string, 0)
	for agentID, info := range r.info {
		if info.Status == AgentStatusInactive {
			inactiveAgents = append(inactiveAgents, agentID)
		}
	}
	r.mu.RUnlock()

	var errors []string
	for _, agentID := range inactiveAgents {
		if err := r.StartAgent(agentID); err != nil {
			errors = append(errors, fmt.Sprintf("failed to start agent %s: %v", agentID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to start some agents: %v", errors)
	}

	log.Printf("Started %d agents", len(inactiveAgents))
	return nil
}

// StopAllAgents stops all active agents
func (r *AgentRegistry) StopAllAgents() error {
	r.mu.RLock()
	activeAgents := make([]string, 0)
	for agentID, info := range r.info {
		if info.Status == AgentStatusActive {
			activeAgents = append(activeAgents, agentID)
		}
	}
	r.mu.RUnlock()

	var errors []string
	for _, agentID := range activeAgents {
		if err := r.StopAgent(agentID); err != nil {
			errors = append(errors, fmt.Sprintf("failed to stop agent %s: %v", agentID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some agents: %v", errors)
	}

	log.Printf("Stopped %d agents", len(activeAgents))
	return nil
}

// RestartAllAgents restarts all agents
func (r *AgentRegistry) RestartAllAgents() error {
	r.mu.RLock()
	allAgents := make([]string, 0)
	for agentID := range r.agents {
		allAgents = append(allAgents, agentID)
	}
	r.mu.RUnlock()

	var errors []string
	for _, agentID := range allAgents {
		if err := r.RestartAgent(agentID); err != nil {
			errors = append(errors, fmt.Sprintf("failed to restart agent %s: %v", agentID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to restart some agents: %v", errors)
	}

	log.Printf("Restarted %d agents", len(allAgents))
	return nil
}

// UpdateAgentConfig updates configuration for a specific agent
func (r *AgentRegistry) UpdateAgentConfig(agentID string, config map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}

	// Update agent configuration
	err := agent.UpdateConfig(config)
	if err != nil {
		r.emitEvent(&AgentEvent{
			Type:      "agent_config_update_failed",
			AgentID:   agentID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})
		return fmt.Errorf("failed to update agent config for %s: %w", agentID, err)
	}

	// Update stored info
	info := r.info[agentID]
	info.Config = config
	info.UpdatedAt = time.Now()
	r.info[agentID] = info

	// Emit event
	r.emitEvent(&AgentEvent{
		Type:      "agent_config_updated",
		AgentID:   agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"config": config,
		},
	})

	log.Printf("Agent config updated: %s", agentID)
	return nil
}
