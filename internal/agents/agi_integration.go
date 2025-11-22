package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/loreum-org/cortex/internal/agibridge"
)

// Type aliases for AGI bridge types to avoid import cycles
type AGIConnection = agibridge.AGIConnection
type AGIStreamData = agibridge.AGIStreamData
type AGIStreamType = agibridge.AGIStreamType
type AGIStreamHandler = agibridge.AGIStreamHandler
type AGIState = agibridge.AGIState
type ConsciousnessSnapshot = agibridge.ConsciousnessSnapshot
type AttentionFocus = agibridge.AttentionFocus
type ConsciousnessUpdate = agibridge.ConsciousnessUpdate
type ConsciousnessUpdateType = agibridge.ConsciousnessUpdateType
type AgentPerformanceReport = agibridge.AgentPerformanceReport
type AgentInsight = agibridge.AgentInsight
type AgentInsightType = agibridge.AgentInsightType
type InsightImpact = agibridge.InsightImpact
type AGICapability = agibridge.AGICapability
type AGIGuidanceRequest = agibridge.AGIGuidanceRequest
type AGIGuidanceResponse = agibridge.AGIGuidanceResponse

// Re-export constants
const (
	AGIStreamTypeQuery             = agibridge.AGIStreamTypeQuery
	AGIStreamTypeResponse          = agibridge.AGIStreamTypeResponse
	AGIStreamTypeMetrics           = agibridge.AGIStreamTypeMetrics
	AGIStreamTypeInsight           = agibridge.AGIStreamTypeInsight
	AGIStreamTypeError             = agibridge.AGIStreamTypeError
	AGIStreamTypeStateUpdate       = agibridge.AGIStreamTypeStateUpdate
	AGIStreamTypeConsciousnessSync = agibridge.AGIStreamTypeConsciousnessSync
	AGIStreamTypeCapabilityUpdate  = agibridge.AGIStreamTypeCapabilityUpdate
	AGIStreamTypeGuidance          = agibridge.AGIStreamTypeGuidance
	AGIStreamTypeFeedback          = agibridge.AGIStreamTypeFeedback
)

const (
	ConsciousnessUpdateAttention = agibridge.ConsciousnessUpdateAttention
	ConsciousnessUpdateEmotion   = agibridge.ConsciousnessUpdateEmotion
	ConsciousnessUpdateGoal      = agibridge.ConsciousnessUpdateGoal
	ConsciousnessUpdateMemory    = agibridge.ConsciousnessUpdateMemory
	ConsciousnessUpdateInsight   = agibridge.ConsciousnessUpdateInsight
	ConsciousnessUpdateLearning  = agibridge.ConsciousnessUpdateLearning
	ConsciousnessUpdateEnergy    = agibridge.ConsciousnessUpdateEnergy
)

const (
	InsightTypePattern      = agibridge.InsightTypePattern
	InsightTypeCorrelation  = agibridge.InsightTypeCorrelation
	InsightTypeAnomaly      = agibridge.InsightTypeAnomaly
	InsightTypeOptimization = agibridge.InsightTypeOptimization
	InsightTypePrediction   = agibridge.InsightTypePrediction
	InsightTypeKnowledge    = agibridge.InsightTypeKnowledge
	InsightTypeStrategy     = agibridge.InsightTypeStrategy
)

// AGIConnectionManager manages AGI connections for the agent registry
type AGIConnectionManager struct {
	registry       *AgentRegistry
	agiConnections map[string]AGIConnection
	streamHandlers map[string]AGIStreamHandler
	mu             sync.RWMutex

	// Streaming infrastructure
	agiStreamChan chan *AGIStreamData
	stopChan      chan struct{}
	isRunning     bool

	// State synchronization
	lastAGIState      *AGIState
	stateSyncInterval time.Duration
}

// NewAGIConnectionManager creates a new AGI connection manager
func NewAGIConnectionManager(registry *AgentRegistry) *AGIConnectionManager {
	return &AGIConnectionManager{
		registry:          registry,
		agiConnections:    make(map[string]AGIConnection),
		streamHandlers:    make(map[string]AGIStreamHandler),
		agiStreamChan:     make(chan *AGIStreamData, 1000),
		stopChan:          make(chan struct{}),
		stateSyncInterval: 5 * time.Second, // Sync AGI state every 5 seconds
	}
}

// Start starts the AGI connection manager
func (acm *AGIConnectionManager) Start(ctx context.Context) error {
	acm.mu.Lock()
	defer acm.mu.Unlock()

	if acm.isRunning {
		return fmt.Errorf("AGI connection manager is already running")
	}

	acm.isRunning = true

	// Start stream processing
	go acm.processAGIStreams(ctx)

	// Start state synchronization
	go acm.synchronizeAGIState(ctx)

	log.Printf("AGI connection manager started")
	return nil
}

// Stop stops the AGI connection manager
func (acm *AGIConnectionManager) Stop() error {
	acm.mu.Lock()
	defer acm.mu.Unlock()

	if !acm.isRunning {
		return nil
	}

	close(acm.stopChan)
	acm.isRunning = false

	log.Printf("AGI connection manager stopped")
	return nil
}

// RegisterAGIConnection registers an AGI connection for an agent
func (acm *AGIConnectionManager) RegisterAGIConnection(agentID string, connection AGIConnection) error {
	acm.mu.Lock()
	defer acm.mu.Unlock()

	acm.agiConnections[agentID] = connection

	// Set up stream handler for this agent
	handler := func(data *AGIStreamData) error {
		return acm.handleAGIStream(agentID, data)
	}

	acm.streamHandlers[agentID] = handler

	// Subscribe to AGI streams
	if err := connection.SubscribeToAGI(handler); err != nil {
		return fmt.Errorf("failed to subscribe agent %s to AGI: %w", agentID, err)
	}

	log.Printf("Registered AGI connection for agent %s", agentID)
	return nil
}

// UnregisterAGIConnection unregisters an AGI connection
func (acm *AGIConnectionManager) UnregisterAGIConnection(agentID string) error {
	acm.mu.Lock()
	defer acm.mu.Unlock()

	if connection, exists := acm.agiConnections[agentID]; exists {
		connection.UnsubscribeFromAGI()
		delete(acm.agiConnections, agentID)
		delete(acm.streamHandlers, agentID)

		log.Printf("Unregistered AGI connection for agent %s", agentID)
	}

	return nil
}

// StreamToAGI streams data from an agent to AGI
func (acm *AGIConnectionManager) StreamToAGI(agentID string, data *AGIStreamData) error {
	acm.mu.RLock()
	connection, exists := acm.agiConnections[agentID]
	acm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no AGI connection registered for agent %s", agentID)
	}

	data.AgentID = agentID
	data.Timestamp = time.Now()

	return connection.StreamToAGI(data)
}

// BroadcastToAGI broadcasts data from all connected agents to AGI
func (acm *AGIConnectionManager) BroadcastToAGI(data *AGIStreamData) error {
	acm.mu.RLock()
	connections := make(map[string]AGIConnection)
	for id, conn := range acm.agiConnections {
		connections[id] = conn
	}
	acm.mu.RUnlock()

	var lastErr error
	for agentID, connection := range connections {
		streamData := *data // Copy
		streamData.AgentID = agentID
		streamData.Timestamp = time.Now()

		if err := connection.StreamToAGI(&streamData); err != nil {
			log.Printf("Failed to stream to AGI from agent %s: %v", agentID, err)
			lastErr = err
		}
	}

	return lastErr
}

// GetAGIState returns the current AGI state
func (acm *AGIConnectionManager) GetAGIState() *AGIState {
	acm.mu.RLock()
	defer acm.mu.RUnlock()

	if acm.lastAGIState != nil {
		// Return a copy
		stateCopy := *acm.lastAGIState
		return &stateCopy
	}

	return nil
}

// processAGIStreams processes incoming AGI stream data
func (acm *AGIConnectionManager) processAGIStreams(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-acm.stopChan:
			return
		case streamData := <-acm.agiStreamChan:
			acm.handleIncomingAGIStream(streamData)
		}
	}
}

// handleAGIStream handles AGI stream data for a specific agent
func (acm *AGIConnectionManager) handleAGIStream(agentID string, data *AGIStreamData) error {
	// Add to processing queue
	select {
	case acm.agiStreamChan <- data:
		return nil
	default:
		log.Printf("AGI stream buffer full, dropping message from agent %s", agentID)
		return fmt.Errorf("stream buffer full")
	}
}

// handleIncomingAGIStream processes incoming AGI stream data
func (acm *AGIConnectionManager) handleIncomingAGIStream(data *AGIStreamData) {
	switch data.Type {
	case AGIStreamTypeStateUpdate:
		acm.handleAGIStateUpdate(data)
	case AGIStreamTypeGuidance:
		acm.handleAGIGuidance(data)
	case AGIStreamTypeConsciousnessSync:
		acm.handleConsciousnessSync(data)
	case AGIStreamTypeFeedback:
		acm.handleAGIFeedback(data)
	default:
		log.Printf("Unknown AGI stream type: %s", data.Type)
	}
}

// handleAGIStateUpdate handles AGI state updates
func (acm *AGIConnectionManager) handleAGIStateUpdate(data *AGIStreamData) {
	// Parse AGI state from stream data
	stateBytes, err := json.Marshal(data.Data)
	if err != nil {
		log.Printf("Failed to marshal AGI state data: %v", err)
		return
	}

	var agiState AGIState
	if err := json.Unmarshal(stateBytes, &agiState); err != nil {
		log.Printf("Failed to unmarshal AGI state: %v", err)
		return
	}

	acm.mu.Lock()
	acm.lastAGIState = &agiState
	acm.mu.Unlock()

	// Notify all agents of AGI state update
	acm.notifyAgentsOfAGIUpdate(&agiState)
}

// handleAGIGuidance handles guidance from AGI
func (acm *AGIConnectionManager) handleAGIGuidance(data *AGIStreamData) {
	// Extract guidance and route to appropriate agent
	if targetAgent, exists := data.Metadata["target_agent"]; exists {
		if agent, err := acm.registry.GetAgent(targetAgent); err == nil {
			// Create agent event for guidance
			event := &AgentEvent{
				Type:      "agi_guidance",
				AgentID:   targetAgent,
				Timestamp: time.Now(),
				Data:      data.Data,
			}

			agent.OnEvent(event)
		}
	}
}

// handleConsciousnessSync handles consciousness synchronization
func (acm *AGIConnectionManager) handleConsciousnessSync(data *AGIStreamData) {
	// Update consciousness state based on agent feedback
	log.Printf("Processing consciousness sync from agent %s", data.AgentID)

	// This would integrate with the consciousness runtime
	// to update attention, emotions, goals, etc.
}

// handleAGIFeedback handles feedback from AGI
func (acm *AGIConnectionManager) handleAGIFeedback(data *AGIStreamData) {
	// Process feedback and update agent performance metrics
	if targetAgent, exists := data.Metadata["target_agent"]; exists {
		if agent, err := acm.registry.GetAgent(targetAgent); err == nil {
			event := &AgentEvent{
				Type:      "agi_feedback",
				AgentID:   targetAgent,
				Timestamp: time.Now(),
				Data:      data.Data,
			}

			agent.OnEvent(event)
		}
	}
}

// notifyAgentsOfAGIUpdate notifies all agents of AGI state updates
func (acm *AGIConnectionManager) notifyAgentsOfAGIUpdate(state *AGIState) {
	agents := acm.registry.GetAllAgents()

	for _, agentInfo := range agents {
		if agent, err := acm.registry.GetAgent(agentInfo.ID); err == nil {
			event := &AgentEvent{
				Type:      "agi_state_update",
				AgentID:   agentInfo.ID,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"agi_state": state,
				},
			}

			agent.OnEvent(event)
		}
	}
}

// synchronizeAGIState periodically synchronizes AGI state
func (acm *AGIConnectionManager) synchronizeAGIState(ctx context.Context) {
	ticker := time.NewTicker(acm.stateSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-acm.stopChan:
			return
		case <-ticker.C:
			acm.syncAGIStateFromConnections()
		}
	}
}

// syncAGIStateFromConnections synchronizes AGI state from all connections
func (acm *AGIConnectionManager) syncAGIStateFromConnections() {
	acm.mu.RLock()
	connections := make(map[string]AGIConnection)
	for id, conn := range acm.agiConnections {
		connections[id] = conn
	}
	acm.mu.RUnlock()

	// Get AGI state from first available connection
	for _, connection := range connections {
		if state, err := connection.GetAGIState(); err == nil {
			acm.mu.Lock()
			acm.lastAGIState = state
			acm.mu.Unlock()
			break
		}
	}
}
