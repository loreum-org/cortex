package consciousness

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/loreum-org/cortex/internal/events"
)

// ConsciousnessEventBridge connects consciousness system to event system
type ConsciousnessEventBridge struct {
	consciousnessSystem *ConsciousnessSystem
	eventBus            *events.EventBus

	// Event streaming
	streamingConnections map[string]*StreamingConnection
	streamingMutex       sync.RWMutex

	// State tracking
	lastBroadcastState *SystemStatus
	stateMutex         sync.RWMutex

	// Configuration
	broadcastInterval time.Duration
	enableStreaming   bool

	// Runtime
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
}

// StreamingConnection represents a streaming connection for consciousness data
type StreamingConnection struct {
	ID           string
	ConnectionID string
	UserID       string
	EventTypes   []string
	Filters      map[string]interface{}
	LastSeen     time.Time
	Channel      chan events.Event
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewConsciousnessEventBridge creates a new consciousness event bridge
func NewConsciousnessEventBridge(
	consciousnessSystem *ConsciousnessSystem,
	eventBus *events.EventBus,
	broadcastInterval time.Duration,
) *ConsciousnessEventBridge {
	if broadcastInterval == 0 {
		broadcastInterval = 1 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ConsciousnessEventBridge{
		consciousnessSystem:  consciousnessSystem,
		eventBus:             eventBus,
		streamingConnections: make(map[string]*StreamingConnection),
		broadcastInterval:    broadcastInterval,
		enableStreaming:      true,
		ctx:                  ctx,
		cancel:               cancel,
	}
}

// Start begins the consciousness event bridge
func (ceb *ConsciousnessEventBridge) Start() error {
	if !ceb.consciousnessSystem.isRunning {
		return fmt.Errorf("consciousness system must be running before starting event bridge")
	}

	// Start periodic broadcasts
	ceb.ticker = time.NewTicker(ceb.broadcastInterval)

	go ceb.broadcastLoop()

	log.Printf("🌉 Consciousness Event Bridge: Started with %v interval", ceb.broadcastInterval)
	return nil
}

// Stop stops the consciousness event bridge
func (ceb *ConsciousnessEventBridge) Stop() error {
	if ceb.ticker != nil {
		ceb.ticker.Stop()
	}

	if ceb.cancel != nil {
		ceb.cancel()
	}

	// Close all streaming connections
	ceb.streamingMutex.Lock()
	for _, conn := range ceb.streamingConnections {
		conn.cancel()
		close(conn.Channel)
	}
	ceb.streamingConnections = make(map[string]*StreamingConnection)
	ceb.streamingMutex.Unlock()

	log.Printf("🌉 Consciousness Event Bridge: Stopped")
	return nil
}

// broadcastLoop handles periodic consciousness state broadcasting
func (ceb *ConsciousnessEventBridge) broadcastLoop() {
	defer ceb.ticker.Stop()

	for {
		select {
		case <-ceb.ticker.C:
			ceb.broadcastConsciousnessState()

		case <-ceb.ctx.Done():
			return
		}
	}
}

// broadcastConsciousnessState broadcasts current consciousness state
func (ceb *ConsciousnessEventBridge) broadcastConsciousnessState() {
	if !ceb.consciousnessSystem.isRunning {
		return
	}

	currentStatus := ceb.consciousnessSystem.GetSystemStatus()

	// Check if state has changed significantly
	if ceb.shouldBroadcast(currentStatus) {
		ceb.stateMutex.Lock()
		ceb.lastBroadcastState = currentStatus
		ceb.stateMutex.Unlock()

		// Create consciousness event
		consciousnessData := events.ConsciousnessData{
			State: map[string]interface{}{
				"is_running":              currentStatus.IsRunning,
				"uptime":                  currentStatus.Uptime.String(),
				"active_domains":          currentStatus.ActiveDomains,
				"collective_intelligence": currentStatus.CollectiveIntelligence,
				"system_health":           currentStatus.SystemHealth,
				"performance_metrics":     currentStatus.PerformanceMetrics,
			},
			NodeID: "consciousness_system",
		}

		// Enhanced consciousness state with domain details
		domainStates := make(map[string]interface{})
		for _, domainType := range currentStatus.ActiveDomains {
			if domainState, err := ceb.consciousnessSystem.GetDomainStatus(domainType); err == nil {
				domainStates[string(domainType)] = map[string]interface{}{
					"type":              domainState.Type,
					"intelligence":      domainState.Intelligence,
					"capabilities":      domainState.Capabilities,
					"active_goals":      domainState.ActiveGoals,
					"recent_insights":   domainState.RecentInsights,
					"metrics":           domainState.Metrics,
					"last_update":       domainState.LastUpdate,
					"version":           domainState.Version,
				}
			}
		}
		consciousnessData.State["domain_states"] = domainStates

		// Create and publish event
		event := events.NewEvent(
			events.EventTypeConsciousnessUpdated,
			"consciousness_bridge",
			consciousnessData,
		).WithMetadata("system_status", currentStatus)

		// Broadcast to event bus
		if err := ceb.eventBus.Publish(event); err != nil {
			log.Printf("❌ Consciousness Bridge: Failed to publish consciousness event: %v", err)
		}

		// Stream to connected clients
		ceb.streamToConnections(event)
	}
}

// shouldBroadcast determines if consciousness state should be broadcast
func (ceb *ConsciousnessEventBridge) shouldBroadcast(currentStatus *SystemStatus) bool {
	ceb.stateMutex.RLock()
	defer ceb.stateMutex.RUnlock()

	if ceb.lastBroadcastState == nil {
		return true
	}

	last := ceb.lastBroadcastState
	current := currentStatus

	// Check for significant changes
	if last.IsRunning != current.IsRunning {
		return true
	}

	if len(last.ActiveDomains) != len(current.ActiveDomains) {
		return true
	}

	// Check health score changes
	if abs(last.SystemHealth.OverallHealth-current.SystemHealth.OverallHealth) > 0.1 {
		return true
	}

	// Check collective intelligence changes
	if last.CollectiveIntelligence != nil && current.CollectiveIntelligence != nil {
		if abs(last.CollectiveIntelligence.OverallLevel-current.CollectiveIntelligence.OverallLevel) > 0.05 {
			return true
		}
	}

	// Check performance metrics changes
	if last.PerformanceMetrics.ProcessedInputs != current.PerformanceMetrics.ProcessedInputs {
		return true
	}

	// Broadcast at least every 30 seconds regardless
	timeSinceLastBroadcast := time.Since(last.LastUpdate)
	if timeSinceLastBroadcast > 30*time.Second {
		return true
	}

	return false
}

// streamToConnections streams events to connected WebSocket clients
func (ceb *ConsciousnessEventBridge) streamToConnections(event events.Event) {
	if !ceb.enableStreaming {
		return
	}

	ceb.streamingMutex.RLock()
	defer ceb.streamingMutex.RUnlock()

	for _, conn := range ceb.streamingConnections {
		if ceb.shouldStreamToConnection(conn, event) {
			select {
			case conn.Channel <- event:
				conn.LastSeen = time.Now()
			default:
				// Channel is full, remove connection
				go ceb.RemoveStreamingConnection(conn.ID)
			}
		}
	}
}

// shouldStreamToConnection determines if event should be streamed to connection
func (ceb *ConsciousnessEventBridge) shouldStreamToConnection(conn *StreamingConnection, event events.Event) bool {
	// Check event type filters
	if len(conn.EventTypes) > 0 {
		found := false
		for _, eventType := range conn.EventTypes {
			if eventType == event.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Apply additional filters
	if len(conn.Filters) > 0 {
		// Filter by domain if specified
		if domainFilter, exists := conn.Filters["domain"]; exists {
			if eventData, ok := event.Data.(events.ConsciousnessData); ok {
				if domainStates, ok := eventData.State["domain_states"].(map[string]interface{}); ok {
					if _, domainExists := domainStates[domainFilter.(string)]; !domainExists {
						return false
					}
				}
			}
		}

		// Filter by health threshold if specified
		if healthThreshold, exists := conn.Filters["min_health"]; exists {
			if eventData, ok := event.Data.(events.ConsciousnessData); ok {
				if healthData, ok := eventData.State["system_health"].(SystemHealth); ok {
					if healthData.OverallHealth < healthThreshold.(float64) {
						return false
					}
				}
			}
		}
	}

	return true
}

// AddStreamingConnection adds a new streaming connection
func (ceb *ConsciousnessEventBridge) AddStreamingConnection(
	connectionID, userID string,
	eventTypes []string,
	filters map[string]interface{},
) *StreamingConnection {

	ctx, cancel := context.WithCancel(ceb.ctx)

	conn := &StreamingConnection{
		ID:           generateStreamingID(),
		ConnectionID: connectionID,
		UserID:       userID,
		EventTypes:   eventTypes,
		Filters:      filters,
		LastSeen:     time.Now(),
		Channel:      make(chan events.Event, 100),
		ctx:          ctx,
		cancel:       cancel,
	}

	ceb.streamingMutex.Lock()
	ceb.streamingConnections[conn.ID] = conn
	ceb.streamingMutex.Unlock()

	log.Printf("🌉 Consciousness Bridge: Added streaming connection %s for user %s", conn.ID, userID)
	return conn
}

// RemoveStreamingConnection removes a streaming connection
func (ceb *ConsciousnessEventBridge) RemoveStreamingConnection(streamingID string) {
	ceb.streamingMutex.Lock()
	if conn, exists := ceb.streamingConnections[streamingID]; exists {
		conn.cancel()
		close(conn.Channel)
		delete(ceb.streamingConnections, streamingID)
		log.Printf("🌉 Consciousness Bridge: Removed streaming connection %s", streamingID)
	}
	ceb.streamingMutex.Unlock()
}

// GetStreamingConnection retrieves a streaming connection by ID
func (ceb *ConsciousnessEventBridge) GetStreamingConnection(streamingID string) (*StreamingConnection, bool) {
	ceb.streamingMutex.RLock()
	defer ceb.streamingMutex.RUnlock()

	conn, exists := ceb.streamingConnections[streamingID]
	return conn, exists
}

// GetActiveStreamingConnections returns all active streaming connections
func (ceb *ConsciousnessEventBridge) GetActiveStreamingConnections() []*StreamingConnection {
	ceb.streamingMutex.RLock()
	defer ceb.streamingMutex.RUnlock()

	connections := make([]*StreamingConnection, 0, len(ceb.streamingConnections))
	for _, conn := range ceb.streamingConnections {
		connections = append(connections, conn)
	}

	return connections
}

// SetBroadcastInterval updates the broadcast interval
func (ceb *ConsciousnessEventBridge) SetBroadcastInterval(interval time.Duration) {
	if interval <= 0 {
		return
	}

	ceb.broadcastInterval = interval

	if ceb.ticker != nil {
		ceb.ticker.Stop()
		ceb.ticker = time.NewTicker(interval)
		log.Printf("🌉 Consciousness Bridge: Broadcast interval updated to %v", interval)
	}
}

// EnableStreaming enables/disables streaming
func (ceb *ConsciousnessEventBridge) EnableStreaming(enabled bool) {
	ceb.enableStreaming = enabled
	log.Printf("🌉 Consciousness Bridge: Streaming %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

// GetBridgeMetrics returns bridge metrics
func (ceb *ConsciousnessEventBridge) GetBridgeMetrics() map[string]interface{} {
	ceb.streamingMutex.RLock()
	defer ceb.streamingMutex.RUnlock()

	return map[string]interface{}{
		"active_streaming_connections": len(ceb.streamingConnections),
		"broadcast_interval":           ceb.broadcastInterval.String(),
		"streaming_enabled":            ceb.enableStreaming,
		"last_broadcast_time": func() string {
			ceb.stateMutex.RLock()
			defer ceb.stateMutex.RUnlock()
			if ceb.lastBroadcastState != nil {
				return ceb.lastBroadcastState.LastUpdate.Format(time.RFC3339)
			}
			return "never"
		}(),
	}
}

// Helper functions

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func generateStreamingID() string {
	return fmt.Sprintf("stream_%d", time.Now().UnixNano())
}
