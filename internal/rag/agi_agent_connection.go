package rag

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/loreum-org/cortex/internal/agibridge"
)

// AGIAgentConnection represents a connection between an agent and the AGI system
type AGIAgentConnection struct {
	agentID        string
	bridge         *AGIAgentBridge
	streamHandlers []agibridge.AGIStreamHandler
	isSubscribed   bool
	mu             sync.RWMutex

	// Metrics
	messagesSent     int64
	messagesReceived int64
	lastActivity     time.Time
	connectionTime   time.Time
}

// StreamToAGI streams data from agent to AGI
func (conn *AGIAgentConnection) StreamToAGI(data *agibridge.AGIStreamData) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if data == nil {
		return fmt.Errorf("stream data cannot be nil")
	}

	// Set agent ID if not set
	if data.AgentID == "" {
		data.AgentID = conn.agentID
	}

	// Set timestamp if not set
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	// Send to bridge for processing
	select {
	case conn.bridge.streamChan <- data:
		conn.messagesSent++
		conn.lastActivity = time.Now()
		return nil
	default:
		return fmt.Errorf("AGI stream channel full")
	}
}

// SubscribeToAGI subscribes to AGI stream data
func (conn *AGIAgentConnection) SubscribeToAGI(handler agibridge.AGIStreamHandler) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if handler == nil {
		return fmt.Errorf("stream handler cannot be nil")
	}

	conn.streamHandlers = append(conn.streamHandlers, handler)
	conn.isSubscribed = true

	log.Printf("Agent %s subscribed to AGI streams", conn.agentID)
	return nil
}

// UnsubscribeFromAGI unsubscribes from AGI stream data
func (conn *AGIAgentConnection) UnsubscribeFromAGI() error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.streamHandlers = conn.streamHandlers[:0]
	conn.isSubscribed = false

	log.Printf("Agent %s unsubscribed from AGI streams", conn.agentID)
	return nil
}

// GetAGIState returns the current AGI state
func (conn *AGIAgentConnection) GetAGIState() (*agibridge.AGIState, error) {
	return conn.bridge.GetCurrentAGIState(), nil
}

// UpdateAGIConsciousness updates AGI consciousness state
func (conn *AGIAgentConnection) UpdateAGIConsciousness(update *agibridge.ConsciousnessUpdate) error {
	if update == nil {
		return fmt.Errorf("consciousness update cannot be nil")
	}

	// Set agent ID if not set
	if update.AgentID == "" {
		update.AgentID = conn.agentID
	}

	// Set timestamp if not set
	if update.Timestamp.IsZero() {
		update.Timestamp = time.Now()
	}

	// Send to bridge for processing
	select {
	case conn.bridge.consciousnessUpdateChan <- update:
		conn.mu.Lock()
		conn.lastActivity = time.Now()
		conn.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("consciousness update channel full")
	}
}

// ReportAgentPerformance reports agent performance metrics
func (conn *AGIAgentConnection) ReportAgentPerformance(metrics *agibridge.AgentPerformanceReport) error {
	if metrics == nil {
		return fmt.Errorf("performance report cannot be nil")
	}

	// Set agent ID if not set
	if metrics.AgentID == "" {
		metrics.AgentID = conn.agentID
	}

	// Set timestamp if not set
	if metrics.Timestamp.IsZero() {
		metrics.Timestamp = time.Now()
	}

	// Stream performance data to AGI
	streamData := &agibridge.AGIStreamData{
		Type:      agibridge.AGIStreamTypeMetrics,
		AgentID:   conn.agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"performance_report": metrics,
		},
		Priority: 6, // Medium-high priority for performance data
	}

	return conn.StreamToAGI(streamData)
}

// ReportAgentInsight reports insights discovered by the agent
func (conn *AGIAgentConnection) ReportAgentInsight(insight *agibridge.AgentInsight) error {
	if insight == nil {
		return fmt.Errorf("insight cannot be nil")
	}

	// Set agent ID if not set
	if insight.AgentID == "" {
		insight.AgentID = conn.agentID
	}

	// Set timestamp if not set
	if insight.Timestamp.IsZero() {
		insight.Timestamp = time.Now()
	}

	// Determine priority based on insight impact
	priority := int(insight.Impact.Magnitude * 10)
	if priority < 1 {
		priority = 1
	}
	if priority > 10 {
		priority = 10
	}

	// Stream insight data to AGI
	streamData := &agibridge.AGIStreamData{
		Type:      agibridge.AGIStreamTypeInsight,
		AgentID:   conn.agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"insight": insight,
		},
		Priority: priority,
	}

	return conn.StreamToAGI(streamData)
}

// RegisterAGICapabilities registers capabilities that the agent can provide to AGI
func (conn *AGIAgentConnection) RegisterAGICapabilities(capabilities []agibridge.AGICapability) error {
	if len(capabilities) == 0 {
		return fmt.Errorf("capabilities list cannot be empty")
	}

	// Stream capability registration to AGI
	streamData := &agibridge.AGIStreamData{
		Type:      agibridge.AGIStreamTypeCapabilityUpdate,
		AgentID:   conn.agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"capabilities": capabilities,
			"action":       "register",
		},
		Priority: 7, // High priority for capability updates
	}

	return conn.StreamToAGI(streamData)
}

// RequestAGIGuidance requests guidance from AGI
func (conn *AGIAgentConnection) RequestAGIGuidance(request *agibridge.AGIGuidanceRequest) (*agibridge.AGIGuidanceResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("guidance request cannot be nil")
	}

	// Set agent ID if not set
	if request.AgentID == "" {
		request.AgentID = conn.agentID
	}

	// Stream guidance request to AGI
	streamData := &agibridge.AGIStreamData{
		Type:      agibridge.AGIStreamTypeGuidance,
		AgentID:   conn.agentID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"guidance_request": request,
		},
		Priority: request.Priority,
		Metadata: map[string]string{
			"type":         "request",
			"target_agent": conn.agentID,
		},
	}

	if err := conn.StreamToAGI(streamData); err != nil {
		return nil, fmt.Errorf("failed to send guidance request: %w", err)
	}

	// For now, return a synchronous response
	// In a full implementation, this might be asynchronous with a response channel
	return conn.generateGuidanceResponse(request), nil
}

// sendToAgent sends stream data to the agent (called by bridge)
func (conn *AGIAgentConnection) sendToAgent(data *agibridge.AGIStreamData) {
	conn.mu.RLock()
	handlers := make([]agibridge.AGIStreamHandler, len(conn.streamHandlers))
	copy(handlers, conn.streamHandlers)
	isSubscribed := conn.isSubscribed
	conn.mu.RUnlock()

	if !isSubscribed || len(handlers) == 0 {
		return
	}

	// Call all stream handlers
	for _, handler := range handlers {
		if err := handler(data); err != nil {
			log.Printf("Error in AGI stream handler for agent %s: %v", conn.agentID, err)
		}
	}

	conn.mu.Lock()
	conn.messagesReceived++
	conn.lastActivity = time.Now()
	conn.mu.Unlock()
}

// GetConnectionMetrics returns connection metrics
func (conn *AGIAgentConnection) GetConnectionMetrics() map[string]interface{} {
	conn.mu.RLock()
	defer conn.mu.RUnlock()

	return map[string]interface{}{
		"agent_id":          conn.agentID,
		"messages_sent":     conn.messagesSent,
		"messages_received": conn.messagesReceived,
		"last_activity":     conn.lastActivity,
		"connection_time":   conn.connectionTime,
		"is_subscribed":     conn.isSubscribed,
		"handlers_count":    len(conn.streamHandlers),
		"uptime_seconds":    time.Since(conn.connectionTime).Seconds(),
	}
}

// generateGuidanceResponse generates a guidance response (placeholder implementation)
func (conn *AGIAgentConnection) generateGuidanceResponse(request *agibridge.AGIGuidanceRequest) *agibridge.AGIGuidanceResponse {
	// This is a placeholder implementation
	// In a full system, this would involve actual AGI processing

	return &agibridge.AGIGuidanceResponse{
		RequestID:      fmt.Sprintf("guid_%s_%d", conn.agentID, time.Now().Unix()),
		Recommendation: fmt.Sprintf("AGI guidance for %s: Consider optimizing %s", request.Context, request.Challenge),
		Reasoning:      "Based on current AGI analysis and agent performance patterns",
		Confidence:     0.75,
		AlternativeOptions: []string{
			"Alternative approach A",
			"Alternative approach B",
		},
		Resources: []string{
			"AGI knowledge base",
			"Performance optimization guidelines",
		},
		FollowUp: []string{
			"Monitor implementation results",
			"Report back on effectiveness",
		},
		Metadata: map[string]interface{}{
			"agi_intelligence_level": conn.bridge.GetCurrentAGIState().IntelligenceLevel,
			"processing_time_ms":     50, // Simulated processing time
		},
		Timestamp: time.Now(),
	}
}

// IsHealthy checks if the connection is healthy
func (conn *AGIAgentConnection) IsHealthy() bool {
	conn.mu.RLock()
	defer conn.mu.RUnlock()

	// Consider connection unhealthy if no activity for more than 5 minutes
	return time.Since(conn.lastActivity) < 5*time.Minute
}

// Reset resets connection metrics
func (conn *AGIAgentConnection) Reset() {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.messagesSent = 0
	conn.messagesReceived = 0
	conn.connectionTime = time.Now()
	conn.lastActivity = time.Now()
}
