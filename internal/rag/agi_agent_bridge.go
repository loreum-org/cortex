package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/loreum-org/cortex/internal/agibridge"
)

// AGIAgentBridge connects the AGI consciousness runtime with the agent system
type AGIAgentBridge struct {
	consciousness  *ConsciousnessRuntime
	agiSystem      *AGIPromptSystem
	contextManager *ContextManager

	// Agent communication
	agentConnections map[string]*AGIAgentConnection
	streamChan       chan *agibridge.AGIStreamData
	mu               sync.RWMutex

	// State synchronization
	consciousnessUpdateChan chan *agibridge.ConsciousnessUpdate
	lastConsciousnessSync   time.Time
	syncInterval            time.Duration

	// Performance tracking
	agentPerformanceData map[string]*agibridge.AgentPerformanceReport
	agentInsights        []*agibridge.AgentInsight
	insightsMu           sync.RWMutex

	// Runtime control
	isRunning bool
	stopChan  chan struct{}
}

// NewAGIAgentBridge creates a new AGI-Agent bridge
func NewAGIAgentBridge(consciousness *ConsciousnessRuntime, agiSystem *AGIPromptSystem, contextManager *ContextManager) *AGIAgentBridge {
	return &AGIAgentBridge{
		consciousness:           consciousness,
		agiSystem:               agiSystem,
		contextManager:          contextManager,
		agentConnections:        make(map[string]*AGIAgentConnection),
		streamChan:              make(chan *agibridge.AGIStreamData, 1000),
		consciousnessUpdateChan: make(chan *agibridge.ConsciousnessUpdate, 500),
		agentPerformanceData:    make(map[string]*agibridge.AgentPerformanceReport),
		agentInsights:           make([]*agibridge.AgentInsight, 0, 100),
		syncInterval:            time.Second, // Sync every second for real-time updates
		stopChan:                make(chan struct{}),
	}
}

// Start starts the AGI-Agent bridge
func (bridge *AGIAgentBridge) Start(ctx context.Context) error {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()

	if bridge.isRunning {
		return fmt.Errorf("AGI-Agent bridge is already running")
	}

	bridge.isRunning = true

	// Start stream processing
	go bridge.processStreams(ctx)

	// Start consciousness updates processing
	go bridge.processConsciousnessUpdates(ctx)

	// Start periodic state synchronization
	go bridge.periodicStateSync(ctx)

	// Start agent performance monitoring
	go bridge.monitorAgentPerformance(ctx)

	log.Printf("AGI-Agent bridge started")
	return nil
}

// Stop stops the AGI-Agent bridge
func (bridge *AGIAgentBridge) Stop() error {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()

	if !bridge.isRunning {
		return nil
	}

	close(bridge.stopChan)
	bridge.isRunning = false

	log.Printf("AGI-Agent bridge stopped")
	return nil
}

// CreateAgentConnection creates a new AGI connection for an agent
func (bridge *AGIAgentBridge) CreateAgentConnection(agentID string) *AGIAgentConnection {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()

	connection := &AGIAgentConnection{
		agentID:        agentID,
		bridge:         bridge,
		streamHandlers: make([]agibridge.AGIStreamHandler, 0),
	}

	bridge.agentConnections[agentID] = connection

	log.Printf("Created AGI connection for agent %s", agentID)
	return connection
}

// RemoveAgentConnection removes an AGI connection
func (bridge *AGIAgentBridge) RemoveAgentConnection(agentID string) {
	bridge.mu.Lock()
	defer bridge.mu.Unlock()

	delete(bridge.agentConnections, agentID)
	delete(bridge.agentPerformanceData, agentID)

	log.Printf("Removed AGI connection for agent %s", agentID)
}

// GetCurrentAGIState returns the current AGI state
func (bridge *AGIAgentBridge) GetCurrentAGIState() *agibridge.AGIState {
	// Get consciousness state
	consciousnessState := bridge.consciousness.GetCurrentState()

	// Get AGI intelligence state
	agiState := bridge.agiSystem.GetCurrentAGIState()

	// Convert to agent-compatible format
	return &agibridge.AGIState{
		IntelligenceLevel: agiState.IntelligenceLevel,
		ConsciousnessState: &agibridge.ConsciousnessSnapshot{
			CycleCount:     consciousnessState.CurrentCycle,
			AttentionFocus: consciousnessState.Attention.CurrentFocus,
			ActiveThoughts: bridge.extractActiveThoughts(consciousnessState.ActiveThoughts),
			WorkingMemory:  bridge.convertWorkingMemory(consciousnessState.WorkingMemory),
			DecisionState:  consciousnessState.DecisionState,
			EnergyLevel:    consciousnessState.EnergyLevel,
			ProcessingLoad: consciousnessState.ProcessingLoad,
		},
		DomainKnowledge: bridge.extractDomainKnowledge(agiState.KnowledgeDomains),
		ActiveGoals:     bridge.extractActiveGoals(consciousnessState.Goals),
		CurrentAttention: &agibridge.AttentionFocus{
			Topic:     consciousnessState.Attention.CurrentFocus,
			Intensity: consciousnessState.Attention.Intensity,
			Duration:  consciousnessState.Attention.Duration,
			Context:   consciousnessState.Attention.Context,
			Agents:    bridge.getAttentionAgents(consciousnessState.Attention),
		},
		EmotionalState: consciousnessState.Emotions,
		LearningRate:   bridge.calculateLearningRate(agiState),
		Timestamp:      time.Now(),
	}
}

// processStreams processes incoming stream data from agents
func (bridge *AGIAgentBridge) processStreams(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-bridge.stopChan:
			return
		case streamData := <-bridge.streamChan:
			bridge.handleAgentStream(streamData)
		}
	}
}

// handleAgentStream handles stream data from agents
func (bridge *AGIAgentBridge) handleAgentStream(data *agibridge.AGIStreamData) {
	switch data.Type {
	case agibridge.AGIStreamTypeQuery:
		bridge.handleAgentQuery(data)
	case agibridge.AGIStreamTypeMetrics:
		bridge.handleAgentMetrics(data)
	case agibridge.AGIStreamTypeInsight:
		bridge.handleAgentInsight(data)
	case agibridge.AGIStreamTypeError:
		bridge.handleAgentError(data)
	case agibridge.AGIStreamTypeFeedback:
		bridge.handleAgentFeedback(data)
	default:
		log.Printf("Unknown stream type from agent %s: %s", data.AgentID, data.Type)
	}
}

// handleAgentQuery handles queries from agents
func (bridge *AGIAgentBridge) handleAgentQuery(data *agibridge.AGIStreamData) {
	// Create sensor input for consciousness runtime
	sensorInput := &SensorInput{
		Type:      "agent_query",
		Content:   fmt.Sprintf("Agent query from %s", data.AgentID),
		Priority:  float64(data.Priority) / 10.0, // Convert to 0-1 scale
		Source:    fmt.Sprintf("agent:%s", data.AgentID),
		Metadata:  convertStringMapToInterface(data.Metadata),
		Timestamp: data.Timestamp,
		Context:   data.Data,
	}

	// Feed to consciousness runtime
	if err := bridge.consciousness.AddSensorInput(sensorInput); err != nil {
		log.Printf("Failed to add agent query to consciousness: %v", err)
	}

	// Update attention to focus on this agent if priority is high
	if data.Priority >= 8 {
		bridge.updateAttentionForAgent(data.AgentID, "high_priority_query")
	}
}

// handleAgentMetrics handles performance metrics from agents
func (bridge *AGIAgentBridge) handleAgentMetrics(data *agibridge.AGIStreamData) {
	// Parse performance report
	reportBytes, err := json.Marshal(data.Data)
	if err != nil {
		log.Printf("Failed to marshal agent metrics: %v", err)
		return
	}

	var perfReport agibridge.AgentPerformanceReport
	if err := json.Unmarshal(reportBytes, &perfReport); err != nil {
		log.Printf("Failed to unmarshal agent performance report: %v", err)
		return
	}

	bridge.mu.Lock()
	bridge.agentPerformanceData[data.AgentID] = &perfReport
	bridge.mu.Unlock()

	// Update AGI system with performance insights
	bridge.updateAGIWithPerformance(&perfReport)
}

// handleAgentInsight handles insights from agents
func (bridge *AGIAgentBridge) handleAgentInsight(data *agibridge.AGIStreamData) {
	// Parse insight
	insightBytes, err := json.Marshal(data.Data)
	if err != nil {
		log.Printf("Failed to marshal agent insight: %v", err)
		return
	}

	var insight agibridge.AgentInsight
	if err := json.Unmarshal(insightBytes, &insight); err != nil {
		log.Printf("Failed to unmarshal agent insight: %v", err)
		return
	}

	// Store insight
	bridge.insightsMu.Lock()
	bridge.agentInsights = append(bridge.agentInsights, &insight)

	// Keep only last 100 insights
	if len(bridge.agentInsights) > 100 {
		bridge.agentInsights = bridge.agentInsights[1:]
	}
	bridge.insightsMu.Unlock()

	// Feed insight to AGI system for learning
	bridge.feedInsightToAGI(&insight)

	// Update consciousness with significant insights
	if insight.Impact.Magnitude > 0.7 {
		bridge.updateConsciousnessWithInsight(&insight)
	}
}

// handleAgentError handles errors from agents
func (bridge *AGIAgentBridge) handleAgentError(data *agibridge.AGIStreamData) {
	log.Printf("Agent %s reported error: %v", data.AgentID, data.Data)

	// Create consciousness update for error awareness
	update := &agibridge.ConsciousnessUpdate{
		AgentID:    data.AgentID,
		UpdateType: agibridge.ConsciousnessUpdateAttention,
		Data: map[string]interface{}{
			"error_type":         "agent_error",
			"error_data":         data.Data,
			"requires_attention": true,
		},
		Confidence: 0.9,
		Impact:     0.3,
		Timestamp:  time.Now(),
	}

	bridge.consciousnessUpdateChan <- update
}

// handleAgentFeedback handles feedback from agents
func (bridge *AGIAgentBridge) handleAgentFeedback(data *agibridge.AGIStreamData) {
	// Process agent feedback to improve AGI decision-making
	log.Printf("Received feedback from agent %s: %v", data.AgentID, data.Data)

	// This could be used to improve AGI prompts, adjust consciousness parameters, etc.
	bridge.incorporateFeedbackIntoAGI(data.AgentID, data.Data)
}

// processConsciousnessUpdates processes consciousness updates from agents
func (bridge *AGIAgentBridge) processConsciousnessUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-bridge.stopChan:
			return
		case update := <-bridge.consciousnessUpdateChan:
			bridge.applyConsciousnessUpdate(update)
		}
	}
}

// applyConsciousnessUpdate applies a consciousness update
func (bridge *AGIAgentBridge) applyConsciousnessUpdate(update *agibridge.ConsciousnessUpdate) {
	switch update.UpdateType {
	case agibridge.ConsciousnessUpdateAttention:
		bridge.updateAttention(update)
	case agibridge.ConsciousnessUpdateEmotion:
		bridge.updateEmotion(update)
	case agibridge.ConsciousnessUpdateGoal:
		bridge.updateGoal(update)
	case agibridge.ConsciousnessUpdateMemory:
		bridge.updateMemory(update)
	case agibridge.ConsciousnessUpdateInsight:
		bridge.updateInsight(update)
	case agibridge.ConsciousnessUpdateLearning:
		bridge.updateLearning(update)
	case agibridge.ConsciousnessUpdateEnergy:
		bridge.updateEnergy(update)
	}
}

// updateAttention updates consciousness attention based on agent feedback
func (bridge *AGIAgentBridge) updateAttention(update *agibridge.ConsciousnessUpdate) {
	// This would integrate with the consciousness runtime's attention system
	log.Printf("Updating consciousness attention based on agent %s feedback", update.AgentID)

	// Example: Focus attention on urgent agent concerns
	if requiresAttention, ok := update.Data["requires_attention"].(bool); ok && requiresAttention {
		bridge.updateAttentionForAgent(update.AgentID, "agent_feedback")
	}
}

// updateAttentionForAgent updates attention to focus on a specific agent
func (bridge *AGIAgentBridge) updateAttentionForAgent(agentID, reason string) {
	// Create attention update for consciousness runtime
	sensorInput := &SensorInput{
		Type:      "attention_request",
		Content:   fmt.Sprintf("Attention request from agent %s", agentID),
		Priority:  0.9,
		Source:    fmt.Sprintf("bridge:agent:%s", agentID),
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"agent_id":        agentID,
			"reason":          reason,
			"focus_intensity": 0.8,
		},
	}

	bridge.consciousness.AddSensorInput(sensorInput)
}

// periodicStateSync periodically synchronizes state with agents
func (bridge *AGIAgentBridge) periodicStateSync(ctx context.Context) {
	ticker := time.NewTicker(bridge.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bridge.stopChan:
			return
		case <-ticker.C:
			bridge.syncStateWithAgents()
		}
	}
}

// syncStateWithAgents synchronizes AGI state with all connected agents
func (bridge *AGIAgentBridge) syncStateWithAgents() {
	agiState := bridge.GetCurrentAGIState()

	bridge.mu.RLock()
	connections := make(map[string]*AGIAgentConnection)
	for id, conn := range bridge.agentConnections {
		connections[id] = conn
	}
	bridge.mu.RUnlock()

	// Send state update to all agents
	stateData := &agibridge.AGIStreamData{
		Type:      agibridge.AGIStreamTypeStateUpdate,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"agi_state": agiState,
		},
		Priority: 5, // Medium priority
	}

	for agentID, connection := range connections {
		stateData.AgentID = agentID
		connection.sendToAgent(stateData)
	}

	bridge.lastConsciousnessSync = time.Now()
}

// monitorAgentPerformance monitors agent performance and provides feedback
func (bridge *AGIAgentBridge) monitorAgentPerformance(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Monitor every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bridge.stopChan:
			return
		case <-ticker.C:
			bridge.analyzeAgentPerformance()
		}
	}
}

// analyzeAgentPerformance analyzes agent performance and provides guidance
func (bridge *AGIAgentBridge) analyzeAgentPerformance() {
	bridge.mu.RLock()
	performanceData := make(map[string]*agibridge.AgentPerformanceReport)
	for id, report := range bridge.agentPerformanceData {
		performanceData[id] = report
	}
	connections := make(map[string]*AGIAgentConnection)
	for id, conn := range bridge.agentConnections {
		connections[id] = conn
	}
	bridge.mu.RUnlock()

	// Analyze performance and provide guidance
	for agentID, report := range performanceData {
		if report.PerformanceScore < 0.7 { // Low performance threshold
			guidance := bridge.generatePerformanceGuidance(report)

			if connection, exists := connections[agentID]; exists {
				guidanceData := &agibridge.AGIStreamData{
					Type:      agibridge.AGIStreamTypeGuidance,
					AgentID:   agentID,
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"guidance":           guidance,
						"performance_report": report,
					},
					Priority: 7, // High priority for performance issues
				}

				connection.sendToAgent(guidanceData)
			}
		}
	}
}

// Helper methods for data conversion and processing

func (bridge *AGIAgentBridge) convertWorkingMemory(wm *WorkingMemory) map[string]interface{} {
	return map[string]interface{}{
		"recent_inputs":    wm.RecentInputs,
		"current_context":  wm.CurrentContext,
		"active_beliefs":   wm.ActiveBeliefs,
		"predictions":      wm.Predictions,
		"temporal_context": wm.TemporalContext,
	}
}

func (bridge *AGIAgentBridge) extractActiveGoals(goals []ConsciousGoal) []string {
	activeGoals := make([]string, 0, len(goals))
	for _, goal := range goals {
		if goal.Status == "active" {
			activeGoals = append(activeGoals, goal.Description)
		}
	}
	return activeGoals
}

func (bridge *AGIAgentBridge) getAttentionAgents(attention *AttentionState) []string {
	// Extract agent IDs that are currently in focus
	agents := make([]string, 0)

	// This would extract relevant agent IDs from the attention context
	// For now, return empty slice - would be implemented based on attention structure

	return agents
}

func (bridge *AGIAgentBridge) updateAGIWithPerformance(report *agibridge.AgentPerformanceReport) {
	// Feed performance data to AGI system for learning
	context := fmt.Sprintf("Agent %s performance: score=%.2f, success_rate=%.2f, response_time=%v",
		report.AgentID, report.PerformanceScore, report.SuccessRate, report.ResponseTime)

	// This would be integrated with the AGI learning system
	log.Printf("AGI learning from agent performance: %s", context)
}

func (bridge *AGIAgentBridge) feedInsightToAGI(insight *agibridge.AgentInsight) {
	// Feed insight to AGI prompt system for learning
	insightContext := fmt.Sprintf("Agent %s discovered insight: %s - %s (confidence: %.2f)",
		insight.AgentID, insight.Title, insight.Description, insight.Confidence)

	// This would integrate with the AGI learning mechanisms
	log.Printf("AGI learning from agent insight: %s", insightContext)
}

func (bridge *AGIAgentBridge) updateConsciousnessWithInsight(insight *agibridge.AgentInsight) {
	// Create consciousness update for significant insights
	sensorInput := &SensorInput{
		Type:      "agent_insight",
		Content:   fmt.Sprintf("Insight from agent %s: %s", insight.AgentID, insight.Title),
		Priority:  insight.Confidence * insight.Impact.Magnitude,
		Source:    fmt.Sprintf("agent:%s", insight.AgentID),
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"insight":      insight,
			"significance": insight.Impact.Magnitude,
		},
	}

	bridge.consciousness.AddSensorInput(sensorInput)
}

func (bridge *AGIAgentBridge) incorporateFeedbackIntoAGI(agentID string, feedback map[string]interface{}) {
	// Process agent feedback to improve AGI decision-making
	log.Printf("Incorporating feedback from agent %s into AGI system", agentID)

	// This would be integrated with AGI learning and improvement mechanisms
}

func (bridge *AGIAgentBridge) generatePerformanceGuidance(report *agibridge.AgentPerformanceReport) *agibridge.AGIGuidanceResponse {
	// Generate performance improvement guidance
	guidance := &agibridge.AGIGuidanceResponse{
		RequestID:      fmt.Sprintf("perf_%s_%d", report.AgentID, time.Now().Unix()),
		Recommendation: "Performance improvement suggestions",
		Reasoning:      fmt.Sprintf("Based on performance score of %.2f", report.PerformanceScore),
		Confidence:     0.8,
		AlternativeOptions: []string{
			"Optimize response time",
			"Improve success rate",
			"Enhance quality metrics",
		},
		Resources: []string{
			"Performance monitoring dashboard",
			"Best practices documentation",
		},
		FollowUp: []string{
			"Monitor performance over next hour",
			"Report improvement results",
		},
		Timestamp: time.Now(),
	}

	// Add specific recommendations based on performance issues
	if report.ResponseTime > 5*time.Second {
		guidance.Recommendation += "; Focus on reducing response time"
	}
	if report.SuccessRate < 0.8 {
		guidance.Recommendation += "; Improve error handling and reliability"
	}

	return guidance
}

// updateEmotion, updateGoal, updateMemory, updateInsight, updateLearning, updateEnergy
// These methods would implement specific consciousness updates for each type
func (bridge *AGIAgentBridge) updateEmotion(update *agibridge.ConsciousnessUpdate) {
	log.Printf("Updating consciousness emotion based on agent %s feedback", update.AgentID)
}

func (bridge *AGIAgentBridge) updateGoal(update *agibridge.ConsciousnessUpdate) {
	log.Printf("Updating consciousness goals based on agent %s feedback", update.AgentID)
}

func (bridge *AGIAgentBridge) updateMemory(update *agibridge.ConsciousnessUpdate) {
	log.Printf("Updating consciousness memory based on agent %s feedback", update.AgentID)
}

func (bridge *AGIAgentBridge) updateInsight(update *agibridge.ConsciousnessUpdate) {
	log.Printf("Updating consciousness insights based on agent %s feedback", update.AgentID)
}

func (bridge *AGIAgentBridge) updateLearning(update *agibridge.ConsciousnessUpdate) {
	log.Printf("Updating consciousness learning based on agent %s feedback", update.AgentID)
}

func (bridge *AGIAgentBridge) updateEnergy(update *agibridge.ConsciousnessUpdate) {
	log.Printf("Updating consciousness energy based on agent %s feedback", update.AgentID)
}

// Helper method to extract active thoughts as strings
func (bridge *AGIAgentBridge) extractActiveThoughts(thoughts []Thought) []string {
	result := make([]string, len(thoughts))
	for i, thought := range thoughts {
		result[i] = thought.Content
	}
	return result
}

// Helper methods for AGI state conversion

// extractDomainKnowledge extracts domain knowledge from AGI knowledge domains
func (bridge *AGIAgentBridge) extractDomainKnowledge(knowledgeDomains map[string]*Domain) map[string]float64 {
	domainKnowledge := make(map[string]float64)
	for domainName, domain := range knowledgeDomains {
		if domain != nil {
			// Use the domain's expertise level
			domainKnowledge[domainName] = domain.Expertise
		} else {
			// Fallback for nil domains
			domainKnowledge[domainName] = 0.5
		}
	}
	return domainKnowledge
}

// calculateLearningRate calculates the current learning rate based on AGI state
func (bridge *AGIAgentBridge) calculateLearningRate(agiState *AGIState) float64 {
	// Base learning rate
	baseLearningRate := 0.1

	// Adjust based on intelligence level
	intelligenceMultiplier := agiState.IntelligenceLevel / 10.0

	// Adjust based on consciousness state from bridge's consciousness runtime
	consciousnessState := bridge.consciousness.GetCurrentState()
	if consciousnessState != nil {
		// Lower learning rate when processing load is high
		processingFactor := 1.0 - (consciousnessState.ProcessingLoad * 0.3)
		baseLearningRate *= processingFactor

		// Adjust based on energy level
		energyFactor := consciousnessState.EnergyLevel
		baseLearningRate *= energyFactor
	}

	// Apply intelligence multiplier
	learningRate := baseLearningRate * intelligenceMultiplier

	// Ensure learning rate stays within reasonable bounds
	if learningRate < 0.01 {
		learningRate = 0.01
	}
	if learningRate > 0.5 {
		learningRate = 0.5
	}

	return learningRate
}

// Helper function to convert map[string]string to map[string]interface{}
func convertStringMapToInterface(stringMap map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range stringMap {
		result[k] = v
	}
	return result
}
