package publishers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/events"
)

// SystemPublisher publishes system status and health events
type SystemPublisher struct {
	eventBus    *events.EventBus
	server      ServerInterface
	ticker      *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
	interval    time.Duration
	components  map[string]ComponentChecker
}

// ComponentChecker defines interface for checking component health
type ComponentChecker interface {
	GetName() string
	IsHealthy() bool
	GetStatus() map[string]interface{}
}

// NewSystemPublisher creates a new system status publisher
func NewSystemPublisher(eventBus *events.EventBus, server ServerInterface, interval time.Duration) *SystemPublisher {
	if interval == 0 {
		interval = 10 * time.Second // Default system status update interval
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SystemPublisher{
		eventBus:   eventBus,
		server:     server,
		interval:   interval,
		ctx:        ctx,
		cancel:     cancel,
		components: make(map[string]ComponentChecker),
	}
}

// RegisterComponent registers a component for health checking
func (sp *SystemPublisher) RegisterComponent(component ComponentChecker) {
	sp.components[component.GetName()] = component
	log.Printf("🔧 System Publisher: Registered component %s", component.GetName())
}

// Start begins publishing system status events
func (sp *SystemPublisher) Start() {
	sp.ticker = time.NewTicker(sp.interval)
	
	go func() {
		defer sp.ticker.Stop()
		
		log.Printf("⚙️ System Publisher: Started (interval: %v)", sp.interval)
		
		for {
			select {
			case <-sp.ticker.C:
				sp.publishSystemStatus()
				
			case <-sp.ctx.Done():
				log.Printf("⚙️ System Publisher: Stopped")
				return
			}
		}
	}()
}

// Stop stops the system publisher
func (sp *SystemPublisher) Stop() {
	if sp.cancel != nil {
		sp.cancel()
	}
}

// publishSystemStatus collects and publishes system status
func (sp *SystemPublisher) publishSystemStatus() {
	// Check all registered components
	componentStatuses := make(map[string]interface{})
	overallHealthy := true
	
	for name, component := range sp.components {
		isHealthy := component.IsHealthy()
		status := component.GetStatus()
		
		componentStatuses[name] = map[string]interface{}{
			"healthy": isHealthy,
			"status":  status,
		}
		
		if !isHealthy {
			overallHealthy = false
		}
	}
	
	// Add server metrics if available
	serverStatus := map[string]interface{}{
		"operational": true,
		"timestamp":   time.Now(),
	}
	
	if sp.server != nil {
		serverStatus["connections"] = sp.server.GetConnectionCount()
		serverStatus["uptime"] = sp.server.GetUptime().Seconds()
	}
	
	// Create system status data
	systemStatus := map[string]interface{}{
		"status":     "operational",
		"healthy":    overallHealthy,
		"timestamp":  time.Now(),
		"server":     serverStatus,
		"components": componentStatuses,
	}
	
	if !overallHealthy {
		systemStatus["status"] = "degraded"
	}
	
	// Create and publish system status event
	statusEvent := events.NewEvent(
		"system.health.status",
		"system_publisher",
		systemStatus,
	)
	
	// Publish event
	if err := sp.eventBus.Publish(statusEvent); err != nil {
		log.Printf("❌ System Publisher: Failed to publish system status: %v", err)
	}
}

// PublishSystemEvent allows manual publishing of system events
func (sp *SystemPublisher) PublishSystemEvent(eventType, description string, data map[string]interface{}) error {
	systemEvent := events.NewEvent(
		eventType,
		"system_publisher",
		data,
	).WithMetadata("description", description)
	
	return sp.eventBus.Publish(systemEvent)
}

// PublishComponentStarted publishes a component started event
func (sp *SystemPublisher) PublishComponentStarted(componentName string, metadata map[string]interface{}) error {
	return sp.PublishSystemEvent(
		events.EventTypeComponentStarted,
		fmt.Sprintf("Component %s started", componentName),
		map[string]interface{}{
			"component": componentName,
			"metadata":  metadata,
			"timestamp": time.Now(),
		},
	)
}

// PublishComponentStopped publishes a component stopped event
func (sp *SystemPublisher) PublishComponentStopped(componentName string, metadata map[string]interface{}) error {
	return sp.PublishSystemEvent(
		events.EventTypeComponentStopped,
		fmt.Sprintf("Component %s stopped", componentName),
		map[string]interface{}{
			"component": componentName,
			"metadata":  metadata,
			"timestamp": time.Now(),
		},
	)
}

// PublishHealthCheck publishes a health check event
func (sp *SystemPublisher) PublishHealthCheck() error {
	return sp.PublishSystemEvent(
		events.EventTypeHealthCheck,
		"Manual health check triggered",
		map[string]interface{}{
			"trigger":   "manual",
			"timestamp": time.Now(),
		},
	)
}

// GetComponentStatus returns the status of all registered components
func (sp *SystemPublisher) GetComponentStatus() map[string]interface{} {
	result := make(map[string]interface{})
	
	for name, component := range sp.components {
		result[name] = map[string]interface{}{
			"healthy": component.IsHealthy(),
			"status":  component.GetStatus(),
		}
	}
	
	return result
}

// GetInterval returns the current publishing interval
func (sp *SystemPublisher) GetInterval() time.Duration {
	return sp.interval
}

// SetInterval updates the publishing interval
func (sp *SystemPublisher) SetInterval(interval time.Duration) {
	if interval <= 0 {
		return
	}
	
	sp.interval = interval
	
	// Restart ticker with new interval if running
	if sp.ticker != nil {
		sp.ticker.Stop()
		sp.ticker = time.NewTicker(interval)
		log.Printf("⚙️ System Publisher: Interval updated to %v", interval)
	}
}

// Default component implementations

// RAGComponentChecker checks RAG system health
type RAGComponentChecker struct {
	ragSystem interface{}
}

func NewRAGComponentChecker(ragSystem interface{}) *RAGComponentChecker {
	return &RAGComponentChecker{ragSystem: ragSystem}
}

func (rc *RAGComponentChecker) GetName() string {
	return "rag_system"
}

func (rc *RAGComponentChecker) IsHealthy() bool {
	if rc.ragSystem == nil {
		return false
	}
	// Add more sophisticated health checks here
	return true
}

func (rc *RAGComponentChecker) GetStatus() map[string]interface{} {
	if rc.ragSystem == nil {
		return map[string]interface{}{
			"available": false,
			"reason":    "RAG system not initialized",
		}
	}
	
	return map[string]interface{}{
		"available": true,
		"status":    "active",
	}
}

// AgentComponentChecker checks agent registry health
type AgentComponentChecker struct {
	agentRegistry interface{}
}

func NewAgentComponentChecker(agentRegistry interface{}) *AgentComponentChecker {
	return &AgentComponentChecker{agentRegistry: agentRegistry}
}

func (ac *AgentComponentChecker) GetName() string {
	return "agent_registry"
}

func (ac *AgentComponentChecker) IsHealthy() bool {
	return ac.agentRegistry != nil
}

func (ac *AgentComponentChecker) GetStatus() map[string]interface{} {
	if ac.agentRegistry == nil {
		return map[string]interface{}{
			"available": false,
			"reason":    "Agent registry not initialized",
		}
	}
	
	return map[string]interface{}{
		"available": true,
		"status":    "active",
	}
}

// EventBusComponentChecker checks event bus health
type EventBusComponentChecker struct {
	eventBus *events.EventBus
}

func NewEventBusComponentChecker(eventBus *events.EventBus) *EventBusComponentChecker {
	return &EventBusComponentChecker{eventBus: eventBus}
}

func (ec *EventBusComponentChecker) GetName() string {
	return "event_bus"
}

func (ec *EventBusComponentChecker) IsHealthy() bool {
	return ec.eventBus != nil
}

func (ec *EventBusComponentChecker) GetStatus() map[string]interface{} {
	if ec.eventBus == nil {
		return map[string]interface{}{
			"available": false,
			"reason":    "Event bus not initialized",
		}
	}
	
	metrics := ec.eventBus.GetMetrics()
	
	return map[string]interface{}{
		"available":    true,
		"metrics":      metrics,
		"event_types":  ec.eventBus.GetEventTypes(),
	}
}