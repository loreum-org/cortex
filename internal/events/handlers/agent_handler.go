package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/agents"
	"github.com/loreum-org/cortex/internal/events"
	"github.com/loreum-org/cortex/pkg/types"
)

// AgentEventHandler handles agent registry events
type AgentEventHandler struct {
	agentRegistry *agents.AgentRegistry
	eventBus      *events.EventBus
	name          string
}

// NewAgentEventHandler creates a new agent event handler
func NewAgentEventHandler(agentRegistry *agents.AgentRegistry, eventBus *events.EventBus) *AgentEventHandler {
	return &AgentEventHandler{
		agentRegistry: agentRegistry,
		eventBus:      eventBus,
		name:          "agent_handler",
	}
}

// HandlerName returns the handler name
func (ah *AgentEventHandler) HandlerName() string {
	return ah.name
}

// SubscribedEvents returns the events this handler subscribes to
func (ah *AgentEventHandler) SubscribedEvents() []string {
	return []string{
		events.EventTypeQuerySubmit,
		events.EventTypeAgentCreate,
		events.EventTypeAgentDelete,
		events.EventTypeComponentStarted,
		events.EventTypeHealthCheck,
		"agent.start",
		"agent.stop",
		"agent.restart",
		"agent.get_metrics",
		"agent.update_config",
		"agent.start_all",
		"agent.stop_all",
		"agent.restart_all",
	}
}

// Handle processes events
func (ah *AgentEventHandler) Handle(ctx context.Context, event events.Event) error {
	switch event.Type {
	case events.EventTypeQuerySubmit:
		return ah.handleQuerySubmit(ctx, event)
	case events.EventTypeAgentCreate:
		return ah.handleAgentCreate(ctx, event)
	case events.EventTypeAgentDelete:
		return ah.handleAgentDelete(ctx, event)
	case events.EventTypeComponentStarted:
		return ah.handleComponentStarted(ctx, event)
	case events.EventTypeHealthCheck:
		return ah.handleHealthCheck(ctx, event)
	case "agent.start":
		return ah.handleAgentStart(ctx, event)
	case "agent.stop":
		return ah.handleAgentStop(ctx, event)
	case "agent.restart":
		return ah.handleAgentRestart(ctx, event)
	case "agent.get_metrics":
		return ah.handleGetAgentMetrics(ctx, event)
	case "agent.update_config":
		return ah.handleUpdateAgentConfig(ctx, event)
	case "agent.start_all":
		return ah.handleStartAllAgents(ctx, event)
	case "agent.stop_all":
		return ah.handleStopAllAgents(ctx, event)
	case "agent.restart_all":
		return ah.handleRestartAllAgents(ctx, event)
	default:
		return fmt.Errorf("unsupported event type: %s", event.Type)
	}
}

// handleQuerySubmit routes queries to appropriate agents
func (ah *AgentEventHandler) handleQuerySubmit(ctx context.Context, event events.Event) error {
	log.Printf("🤖 Agent Handler: Processing query submission %s", event.ID)
	
	// Extract query data
	queryData, ok := event.Data.(events.QueryData)
	if !ok {
		return fmt.Errorf("invalid query data type")
	}
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Publish query started event
	ah.publishQueryStarted(event, queryData)
	
	// Create agent query object using types.Query
	agentQuery := &types.Query{
		ID:        queryData.QueryID,
		Text:      queryData.Text,
		Type:      "user_query",
		Metadata:  map[string]string{"user_id": queryData.UserID},
		Timestamp: time.Now().Unix(),
	}
	
	// Process query with context timeout
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	start := time.Now()
	
	// Route query to appropriate agent
	result, err := ah.agentRegistry.RouteQuery(queryCtx, agentQuery)
	
	duration := time.Since(start)
	
	if err != nil {
		log.Printf("❌ Agent Query failed for %s: %v", event.ID, err)
		return ah.publishQueryError(event, queryData, err, duration)
	}
	
	log.Printf("✅ Agent Query completed for %s in %v", event.ID, duration)
	
	// Publish successful result
	return ah.publishQueryResult(event, queryData, result, duration)
}

// handleAgentCreate creates a new agent
func (ah *AgentEventHandler) handleAgentCreate(ctx context.Context, event events.Event) error {
	log.Printf("🔧 Agent Handler: Creating agent %s", event.ID)
	
	// Extract agent data
	agentData, ok := event.Data.(events.AgentData)
	if !ok {
		return fmt.Errorf("invalid agent data type")
	}
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Create a base agent with the provided information
	agentInfo := agents.AgentInfo{
		ID:           agentData.AgentID,
		Name:         agentData.Name,
		Type:         agents.AgentTypeSolver, // Default type
		Status:       agents.AgentStatusInitializing,
		Capabilities: []agents.AgentCapability{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	agent := agents.NewBaseAgent(agentInfo)

	// Register the agent
	err := ah.agentRegistry.RegisterAgent(agent)
	if err != nil {
		log.Printf("❌ Agent creation failed for %s: %v", event.ID, err)
		return ah.publishAgentError(event, agentData, "creation_failed", err)
	}
	
	log.Printf("✅ Agent %s created successfully", agentData.AgentID)
	
	// Publish agent registered event
	return ah.publishAgentRegistered(event, agentData, agent)
}

// handleAgentDelete removes an agent
func (ah *AgentEventHandler) handleAgentDelete(ctx context.Context, event events.Event) error {
	log.Printf("🗑️ Agent Handler: Deleting agent %s", event.ID)
	
	// Extract agent data
	agentData, ok := event.Data.(events.AgentData)
	if !ok {
		return fmt.Errorf("invalid agent data type")
	}
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Deregister the agent
	err := ah.agentRegistry.DeregisterAgent(agentData.AgentID)
	if err != nil {
		log.Printf("❌ Agent deletion failed for %s: %v", event.ID, err)
		return ah.publishAgentError(event, agentData, "deletion_failed", err)
	}
	
	log.Printf("✅ Agent %s deleted successfully", agentData.AgentID)
	
	// Publish agent deregistered event
	return ah.publishAgentDeregistered(event, agentData)
}

// handleComponentStarted handles component startup events
func (ah *AgentEventHandler) handleComponentStarted(ctx context.Context, event events.Event) error {
	if event.Source == "agent" || event.Source == "agent_registry" {
		log.Printf("🔄 Agent Handler: Component %s started", event.Source)
		return ah.performHealthCheck(ctx, event)
	}
	return nil
}

// handleHealthCheck processes health check events
func (ah *AgentEventHandler) handleHealthCheck(ctx context.Context, event events.Event) error {
	return ah.performHealthCheck(ctx, event)
}

// performHealthCheck checks agent registry health
func (ah *AgentEventHandler) performHealthCheck(ctx context.Context, event events.Event) error {
	if ah.agentRegistry == nil {
		return ah.publishHealthStatus(event, "unhealthy", "Agent registry not available")
	}
	
	// Get registry stats
	stats := ah.agentRegistry.GetRegistryStats()
	
	status := "healthy"
	message := fmt.Sprintf("Agent registry operational with %d agents", 
		stats["active_agents"])
	
	if stats["active_agents"].(int) == 0 {
		status = "degraded"
		message = "No active agents available"
	}
	
	return ah.publishHealthStatus(event, status, message)
}

// Helper methods for publishing events

func (ah *AgentEventHandler) publishQueryStarted(originalEvent events.Event, queryData events.QueryData) {
	event := events.NewEventWithCorrelation(
		events.EventTypeQueryStarted,
		"agent",
		events.QueryResultData{
			QueryID: queryData.QueryID,
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	ah.eventBus.Publish(event)
}

func (ah *AgentEventHandler) publishQueryResult(originalEvent events.Event, queryData events.QueryData, result *types.Response, duration time.Duration) error {
	resultData := events.QueryResultData{
		QueryID:  queryData.QueryID,
		Result:   result.Text,
		Source:   "agent",
		Duration: duration,
		Metadata: map[string]interface{}{
			"query_id":      result.QueryID,
			"user_id":       queryData.UserID,
			"connection_id": queryData.ConnectionID,
		},
	}
	
	event := events.NewEventWithCorrelation(
		events.EventTypeQueryProcessed,
		"agent",
		resultData,
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(event)
}

func (ah *AgentEventHandler) publishQueryError(originalEvent events.Event, queryData events.QueryData, err error, duration time.Duration) error {
	resultData := events.QueryResultData{
		QueryID:  queryData.QueryID,
		Source:   "agent",
		Duration: duration,
		Error:    err.Error(),
		Metadata: map[string]interface{}{
			"user_id":       queryData.UserID,
			"connection_id": queryData.ConnectionID,
		},
	}
	
	event := events.NewEventWithCorrelation(
		events.EventTypeQueryFailed,
		"agent",
		resultData,
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(event)
}

func (ah *AgentEventHandler) publishAgentRegistered(originalEvent events.Event, agentData events.AgentData, agent agents.StandardAgent) error {
	event := events.NewEventWithCorrelation(
		events.EventTypeAgentRegistered,
		"agent",
		events.AgentData{
			AgentID:      agent.GetInfo().ID,
			Name:         agent.GetInfo().Name,
			Type:         string(agent.GetInfo().Type),
			Status:       string(agent.GetStatus()),
			Capabilities: []string{}, // Convert from AgentCapability to string if needed
			Metadata: map[string]interface{}{
				"created_at": time.Now(),
			},
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(event)
}

func (ah *AgentEventHandler) publishAgentDeregistered(originalEvent events.Event, agentData events.AgentData) error {
	event := events.NewEventWithCorrelation(
		events.EventTypeAgentDeregistered,
		"agent",
		agentData,
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(event)
}

func (ah *AgentEventHandler) publishAgentError(originalEvent events.Event, agentData events.AgentData, operation string, err error) error {
	errorEvent := events.NewEventWithCorrelation(
		events.EventTypeError,
		"agent",
		map[string]interface{}{
			"agent_id":  agentData.AgentID,
			"operation": operation,
			"error":     err.Error(),
			"handler":   ah.name,
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(errorEvent)
}

func (ah *AgentEventHandler) publishError(originalEvent events.Event, message string) error {
	errorEvent := events.NewEventWithCorrelation(
		events.EventTypeError,
		"agent",
		map[string]interface{}{
			"error":   message,
			"handler": ah.name,
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(errorEvent)
}

func (ah *AgentEventHandler) publishHealthStatus(originalEvent events.Event, status, message string) error {
	healthEvent := events.NewEventWithCorrelation(
		"system.health.status",
		"agent",
		map[string]interface{}{
			"component": "agent_registry",
			"status":    status,
			"message":   message,
			"timestamp": time.Now(),
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(healthEvent)
}

// GetMetrics returns agent-specific metrics
// Agent lifecycle handlers

// handleAgentStart handles agent start events
func (ah *AgentEventHandler) handleAgentStart(ctx context.Context, event events.Event) error {
	log.Printf("🚀 Agent Handler: Starting agent %s", event.ID)
	
	// Extract agent ID from event data
	agentID, ok := event.Data.(string)
	if !ok {
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			if id, exists := dataMap["agent_id"]; exists {
				if agentIDStr, ok := id.(string); ok {
					agentID = agentIDStr
				} else {
					return fmt.Errorf("invalid agent_id type in event data")
				}
			} else {
				return fmt.Errorf("agent_id not found in event data")
			}
		} else {
			return fmt.Errorf("invalid event data type for agent start")
		}
	}
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Start the agent
	err := ah.agentRegistry.StartAgent(agentID)
	if err != nil {
		log.Printf("❌ Agent start failed for %s: %v", agentID, err)
		return ah.publishAgentLifecycleError(event, agentID, "start_failed", err)
	}
	
	log.Printf("✅ Agent %s started successfully", agentID)
	
	// Publish success event
	return ah.publishAgentLifecycleSuccess(event, agentID, "started")
}

// handleAgentStop handles agent stop events
func (ah *AgentEventHandler) handleAgentStop(ctx context.Context, event events.Event) error {
	log.Printf("🛑 Agent Handler: Stopping agent %s", event.ID)
	
	// Extract agent ID from event data
	agentID, ok := event.Data.(string)
	if !ok {
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			if id, exists := dataMap["agent_id"]; exists {
				if agentIDStr, ok := id.(string); ok {
					agentID = agentIDStr
				} else {
					return fmt.Errorf("invalid agent_id type in event data")
				}
			} else {
				return fmt.Errorf("agent_id not found in event data")
			}
		} else {
			return fmt.Errorf("invalid event data type for agent stop")
		}
	}
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Stop the agent
	err := ah.agentRegistry.StopAgent(agentID)
	if err != nil {
		log.Printf("❌ Agent stop failed for %s: %v", agentID, err)
		return ah.publishAgentLifecycleError(event, agentID, "stop_failed", err)
	}
	
	log.Printf("✅ Agent %s stopped successfully", agentID)
	
	// Publish success event
	return ah.publishAgentLifecycleSuccess(event, agentID, "stopped")
}

// handleAgentRestart handles agent restart events
func (ah *AgentEventHandler) handleAgentRestart(ctx context.Context, event events.Event) error {
	log.Printf("🔄 Agent Handler: Restarting agent %s", event.ID)
	
	// Extract agent ID from event data
	agentID, ok := event.Data.(string)
	if !ok {
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			if id, exists := dataMap["agent_id"]; exists {
				if agentIDStr, ok := id.(string); ok {
					agentID = agentIDStr
				} else {
					return fmt.Errorf("invalid agent_id type in event data")
				}
			} else {
				return fmt.Errorf("agent_id not found in event data")
			}
		} else {
			return fmt.Errorf("invalid event data type for agent restart")
		}
	}
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Restart the agent
	err := ah.agentRegistry.RestartAgent(agentID)
	if err != nil {
		log.Printf("❌ Agent restart failed for %s: %v", agentID, err)
		return ah.publishAgentLifecycleError(event, agentID, "restart_failed", err)
	}
	
	log.Printf("✅ Agent %s restarted successfully", agentID)
	
	// Publish success event
	return ah.publishAgentLifecycleSuccess(event, agentID, "restarted")
}

// handleGetAgentMetrics handles agent metrics request events
func (ah *AgentEventHandler) handleGetAgentMetrics(ctx context.Context, event events.Event) error {
	log.Printf("📊 Agent Handler: Getting agent metrics %s", event.ID)
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Get registry stats
	stats := ah.agentRegistry.GetRegistryStats()
	
	// Get all agents info
	allAgents := ah.agentRegistry.GetAllAgents()
	
	// Publish metrics event
	metricsEvent := events.NewEventWithCorrelation(
		"agent.metrics_response",
		"agent",
		map[string]interface{}{
			"registry_stats": stats,
			"agents":         allAgents,
			"timestamp":      time.Now(),
		},
		event.CorrelationID,
	).WithMetadata("original_event_id", event.ID)
	
	return ah.eventBus.Publish(metricsEvent)
}

// handleUpdateAgentConfig handles agent configuration update events
func (ah *AgentEventHandler) handleUpdateAgentConfig(ctx context.Context, event events.Event) error {
	log.Printf("⚙️ Agent Handler: Updating agent config %s", event.ID)
	
	// Extract data from event
	dataMap, ok := event.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid event data type for agent config update")
	}
	
	agentID, ok := dataMap["agent_id"].(string)
	if !ok {
		return fmt.Errorf("agent_id not found or invalid in event data")
	}
	
	config, ok := dataMap["config"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("config not found or invalid in event data")
	}
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Update agent configuration
	err := ah.agentRegistry.UpdateAgentConfig(agentID, config)
	if err != nil {
		log.Printf("❌ Agent config update failed for %s: %v", agentID, err)
		return ah.publishAgentLifecycleError(event, agentID, "config_update_failed", err)
	}
	
	log.Printf("✅ Agent %s config updated successfully", agentID)
	
	// Publish success event
	return ah.publishAgentConfigUpdateSuccess(event, agentID, config)
}

// handleStartAllAgents handles start all agents events
func (ah *AgentEventHandler) handleStartAllAgents(ctx context.Context, event events.Event) error {
	log.Printf("🚀 Agent Handler: Starting all agents %s", event.ID)
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Start all agents
	err := ah.agentRegistry.StartAllAgents()
	if err != nil {
		log.Printf("❌ Start all agents failed: %v", err)
		return ah.publishBulkOperationError(event, "start_all_failed", err)
	}
	
	log.Printf("✅ All agents started successfully")
	
	// Publish success event
	return ah.publishBulkOperationSuccess(event, "start_all_completed")
}

// handleStopAllAgents handles stop all agents events
func (ah *AgentEventHandler) handleStopAllAgents(ctx context.Context, event events.Event) error {
	log.Printf("🛑 Agent Handler: Stopping all agents %s", event.ID)
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Stop all agents
	err := ah.agentRegistry.StopAllAgents()
	if err != nil {
		log.Printf("❌ Stop all agents failed: %v", err)
		return ah.publishBulkOperationError(event, "stop_all_failed", err)
	}
	
	log.Printf("✅ All agents stopped successfully")
	
	// Publish success event
	return ah.publishBulkOperationSuccess(event, "stop_all_completed")
}

// handleRestartAllAgents handles restart all agents events
func (ah *AgentEventHandler) handleRestartAllAgents(ctx context.Context, event events.Event) error {
	log.Printf("🔄 Agent Handler: Restarting all agents %s", event.ID)
	
	if ah.agentRegistry == nil {
		return ah.publishError(event, "Agent registry not available")
	}
	
	// Restart all agents
	err := ah.agentRegistry.RestartAllAgents()
	if err != nil {
		log.Printf("❌ Restart all agents failed: %v", err)
		return ah.publishBulkOperationError(event, "restart_all_failed", err)
	}
	
	log.Printf("✅ All agents restarted successfully")
	
	// Publish success event
	return ah.publishBulkOperationSuccess(event, "restart_all_completed")
}

// Helper methods for publishing lifecycle events

func (ah *AgentEventHandler) publishAgentLifecycleSuccess(originalEvent events.Event, agentID, operation string) error {
	successEvent := events.NewEventWithCorrelation(
		fmt.Sprintf("agent.%s_success", operation),
		"agent",
		map[string]interface{}{
			"agent_id":  agentID,
			"operation": operation,
			"timestamp": time.Now(),
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(successEvent)
}

func (ah *AgentEventHandler) publishAgentLifecycleError(originalEvent events.Event, agentID, operation string, err error) error {
	errorEvent := events.NewEventWithCorrelation(
		fmt.Sprintf("agent.%s_error", operation),
		"agent",
		map[string]interface{}{
			"agent_id":  agentID,
			"operation": operation,
			"error":     err.Error(),
			"timestamp": time.Now(),
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(errorEvent)
}

func (ah *AgentEventHandler) publishAgentConfigUpdateSuccess(originalEvent events.Event, agentID string, config map[string]interface{}) error {
	successEvent := events.NewEventWithCorrelation(
		"agent.config_update_success",
		"agent",
		map[string]interface{}{
			"agent_id":  agentID,
			"config":    config,
			"timestamp": time.Now(),
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(successEvent)
}

func (ah *AgentEventHandler) publishBulkOperationSuccess(originalEvent events.Event, operation string) error {
	successEvent := events.NewEventWithCorrelation(
		fmt.Sprintf("agent.%s_success", operation),
		"agent",
		map[string]interface{}{
			"operation": operation,
			"timestamp": time.Now(),
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(successEvent)
}

func (ah *AgentEventHandler) publishBulkOperationError(originalEvent events.Event, operation string, err error) error {
	errorEvent := events.NewEventWithCorrelation(
		fmt.Sprintf("agent.%s_error", operation),
		"agent",
		map[string]interface{}{
			"operation": operation,
			"error":     err.Error(),
			"timestamp": time.Now(),
		},
		originalEvent.CorrelationID,
	).WithMetadata("original_event_id", originalEvent.ID)
	
	return ah.eventBus.Publish(errorEvent)
}

// GetMetrics returns agent-specific metrics
func (ah *AgentEventHandler) GetMetrics() map[string]interface{} {
	if ah.agentRegistry == nil {
		return map[string]interface{}{
			"status": "unavailable",
		}
	}
	
	stats := ah.agentRegistry.GetRegistryStats()
	
	return map[string]interface{}{
		"status":        "available",
		"active_agents": stats["active_agents"],
		"total_queries": stats["total_queries"],
		"handler_name":  ah.name,
	}
}