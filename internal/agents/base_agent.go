package agents

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// BaseAgent provides a standard implementation that other agents can embed
type BaseAgent struct {
	info    AgentInfo
	metrics AgentMetrics
	status  AgentStatus
	config  map[string]interface{}
	mu      sync.RWMutex

	// Lifecycle
	startTime time.Time
	stopChan  chan struct{}
	started   bool

	// Processing
	processFunc func(ctx context.Context, query *types.Query) (*types.Response, error)

	// AGI Integration
	agiConnection         AGIConnection
	agiIntegrationEnabled bool
	lastAGIState          *AGIState
	agiStateMu            sync.RWMutex
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(info AgentInfo) *BaseAgent {
	now := time.Now()

	return &BaseAgent{
		info: info,
		metrics: AgentMetrics{
			StartTime:      now,
			SuccessRate:    100.0,
			ErrorRate:      0.0,
			TotalQueries:   0,
			ResponseTime:   0.0,
			AvgProcessTime: 0.0,
			Uptime:         0.0,
		},
		status:   AgentStatusInactive,
		config:   make(map[string]interface{}),
		stopChan: make(chan struct{}),
	}
}

// SetProcessFunction sets the processing function for the agent
func (a *BaseAgent) SetProcessFunction(fn func(ctx context.Context, query *types.Query) (*types.Response, error)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.processFunc = fn
}

// Initialize initializes the agent with configuration
func (a *BaseAgent) Initialize(ctx context.Context, config map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.config = config
	a.status = AgentStatusInitializing
	a.info.UpdatedAt = time.Now()

	// Perform any initialization logic here
	// Subclasses can override this method

	return nil
}

// Start starts the agent
func (a *BaseAgent) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.started {
		return fmt.Errorf("agent %s is already started", a.info.ID)
	}

	a.started = true
	a.startTime = time.Now()
	a.metrics.StartTime = a.startTime
	a.status = AgentStatusActive
	a.info.UpdatedAt = time.Now()

	// Start any background routines here
	// Subclasses can override this method

	return nil
}

// Stop stops the agent
func (a *BaseAgent) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.started {
		return fmt.Errorf("agent %s is not running", a.info.ID)
	}

	a.status = AgentStatusStopping
	close(a.stopChan)

	// Perform cleanup here
	// Wait for background routines to finish

	a.started = false
	a.status = AgentStatusInactive
	a.info.UpdatedAt = time.Now()

	return nil
}

// Restart restarts the agent
func (a *BaseAgent) Restart(ctx context.Context) error {
	if err := a.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop agent: %w", err)
	}

	// Give some time for cleanup
	time.Sleep(100 * time.Millisecond)

	if err := a.Start(ctx); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	return nil
}

// Process processes a query using the configured process function
func (a *BaseAgent) Process(ctx context.Context, query *types.Query) (*types.Response, error) {
	a.mu.RLock()
	if a.status != AgentStatusActive {
		a.mu.RUnlock()
		return nil, fmt.Errorf("agent %s is not active (status: %s)", a.info.ID, a.status)
	}

	if a.processFunc == nil {
		a.mu.RUnlock()
		return nil, fmt.Errorf("no process function configured for agent %s", a.info.ID)
	}

	processFunc := a.processFunc
	a.mu.RUnlock()

	// Track processing time
	startTime := time.Now()
	response, err := processFunc(ctx, query)
	processingTime := time.Since(startTime)

	// Update metrics
	a.updateMetrics(processingTime, err == nil)

	return response, err
}

// GetInfo returns agent information
func (a *BaseAgent) GetInfo() AgentInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.info
}

// GetCapabilities returns agent capabilities
func (a *BaseAgent) GetCapabilities() []AgentCapability {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.info.Capabilities
}

// GetMetrics returns agent metrics
func (a *BaseAgent) GetMetrics() AgentMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Update uptime
	metrics := a.metrics
	if a.started {
		metrics.Uptime = time.Since(a.startTime).Hours()
	}

	return metrics
}

// GetStatus returns agent status
func (a *BaseAgent) GetStatus() AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// HealthCheck performs a health check
func (a *BaseAgent) HealthCheck(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.started {
		return fmt.Errorf("agent %s is not running", a.info.ID)
	}

	if a.status == AgentStatusError {
		return fmt.Errorf("agent %s is in error state", a.info.ID)
	}

	// Subclasses can add more specific health checks
	return nil
}

// UpdateConfig updates agent configuration
func (a *BaseAgent) UpdateConfig(config map[string]interface{}) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Merge new config with existing
	for key, value := range config {
		a.config[key] = value
	}

	a.info.Config = a.config
	a.info.UpdatedAt = time.Now()

	// Subclasses can override this to handle config updates
	return nil
}

// OnEvent handles agent events
func (a *BaseAgent) OnEvent(event *AgentEvent) error {
	// Default implementation does nothing
	// Subclasses can override this to handle specific events
	return nil
}

// updateMetrics updates agent performance metrics
func (a *BaseAgent) updateMetrics(processingTime time.Duration, success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	a.metrics.LastQueryAt = &now
	a.metrics.TotalQueries++

	// Update success/error rates
	if success {
		a.metrics.SuccessRate = (a.metrics.SuccessRate*float64(a.metrics.TotalQueries-1) + 100.0) / float64(a.metrics.TotalQueries)
	} else {
		a.metrics.ErrorCount++
		a.metrics.SuccessRate = (a.metrics.SuccessRate*float64(a.metrics.TotalQueries-1) + 0.0) / float64(a.metrics.TotalQueries)
	}

	a.metrics.ErrorRate = 100.0 - a.metrics.SuccessRate

	// Update response time
	processingSecs := processingTime.Seconds()
	a.metrics.AvgProcessTime = (a.metrics.AvgProcessTime*float64(a.metrics.TotalQueries-1) + processingSecs) / float64(a.metrics.TotalQueries)
	a.metrics.ResponseTime = a.metrics.AvgProcessTime

	// Update requests per minute (simple approximation)
	if a.started && a.metrics.TotalQueries > 0 {
		minutesSinceStart := time.Since(a.startTime).Minutes()
		if minutesSinceStart > 0 {
			a.metrics.RequestsPerMin = int(float64(a.metrics.TotalQueries) / minutesSinceStart)
		}
	}
}

// SetStatus sets the agent status (for internal use)
func (a *BaseAgent) SetStatus(status AgentStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = status
	a.info.UpdatedAt = time.Now()
}

// GetConfig returns a copy of the agent configuration
func (a *BaseAgent) GetConfig() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	config := make(map[string]interface{})
	for k, v := range a.config {
		config[k] = v
	}
	return config
}

// UpdateInfo updates agent information
func (a *BaseAgent) UpdateInfo(info AgentInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()

	info.UpdatedAt = time.Now()
	a.info = info
}

// AGI Integration Methods

// SetAGIConnection sets the AGI connection for this agent
func (ba *BaseAgent) SetAGIConnection(connection AGIConnection) error {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	ba.agiConnection = connection
	ba.agiIntegrationEnabled = true

	// Subscribe to AGI streams
	if err := connection.SubscribeToAGI(ba.handleAGIStream); err != nil {
		ba.agiIntegrationEnabled = false
		return fmt.Errorf("failed to subscribe to AGI streams: %w", err)
	}

	return nil
}

// DisableAGIIntegration disables AGI integration for this agent
func (ba *BaseAgent) DisableAGIIntegration() error {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	if ba.agiConnection != nil {
		ba.agiConnection.UnsubscribeFromAGI()
	}

	ba.agiIntegrationEnabled = false
	ba.agiConnection = nil

	return nil
}

// StreamToAGI streams data to the AGI system
func (ba *BaseAgent) StreamToAGI(data *AGIStreamData) error {
	ba.mu.RLock()
	connection := ba.agiConnection
	enabled := ba.agiIntegrationEnabled
	ba.mu.RUnlock()

	if !enabled || connection == nil {
		return fmt.Errorf("AGI integration not enabled for agent %s", ba.info.ID)
	}

	return connection.StreamToAGI(data)
}

// GetAGIState returns the current AGI state
func (ba *BaseAgent) GetAGIState() *AGIState {
	ba.agiStateMu.RLock()
	defer ba.agiStateMu.RUnlock()

	if ba.lastAGIState != nil {
		// Return a copy
		stateCopy := *ba.lastAGIState
		return &stateCopy
	}

	return nil
}

// ReportPerformanceToAGI reports agent performance to AGI
func (ba *BaseAgent) ReportPerformanceToAGI() error {
	ba.mu.RLock()
	connection := ba.agiConnection
	enabled := ba.agiIntegrationEnabled
	metrics := ba.metrics
	ba.mu.RUnlock()

	if !enabled || connection == nil {
		return nil // Silently skip if AGI integration not enabled
	}

	report := &AgentPerformanceReport{
		AgentID:          ba.info.ID,
		TaskType:         "general",
		PerformanceScore: ba.calculatePerformanceScore(),
		ResponseTime:     time.Duration(metrics.AvgProcessTime * float64(time.Second)),
		SuccessRate:      metrics.SuccessRate / 100.0,
		QualityMetrics: map[string]float64{
			"uptime":           metrics.Uptime,
			"error_rate":       metrics.ErrorRate,
			"requests_per_min": float64(metrics.RequestsPerMin),
		},
		LearningPoints: ba.extractLearningPoints(),
		Challenges:     ba.identifyChallenges(),
		Timestamp:      time.Now(),
	}

	return connection.ReportAgentPerformance(report)
}

// ReportInsightToAGI reports an insight to AGI
func (ba *BaseAgent) ReportInsightToAGI(insightType AgentInsightType, title, description string, data map[string]interface{}, confidence float64) error {
	ba.mu.RLock()
	connection := ba.agiConnection
	enabled := ba.agiIntegrationEnabled
	ba.mu.RUnlock()

	if !enabled || connection == nil {
		return nil // Silently skip if AGI integration not enabled
	}

	insight := &AgentInsight{
		AgentID:     ba.info.ID,
		Type:        insightType,
		Title:       title,
		Description: description,
		Data:        data,
		Confidence:  confidence,
		Domain:      "general",
		Impact: InsightImpact{
			Scope:      "local",
			Magnitude:  confidence * 0.5, // Conservative impact estimation
			Confidence: confidence,
			Domains:    []string{"general"},
		},
		Timestamp: time.Now(),
	}

	return connection.ReportAgentInsight(insight)
}

// RequestAGIGuidance requests guidance from AGI
func (ba *BaseAgent) RequestAGIGuidance(context, challenge string, options []string) (*AGIGuidanceResponse, error) {
	ba.mu.RLock()
	connection := ba.agiConnection
	enabled := ba.agiIntegrationEnabled
	ba.mu.RUnlock()

	if !enabled || connection == nil {
		return nil, fmt.Errorf("AGI integration not enabled for agent %s", ba.info.ID)
	}

	request := &AGIGuidanceRequest{
		AgentID:   ba.info.ID,
		Context:   context,
		Challenge: challenge,
		Options:   options,
		Constraints: map[string]interface{}{
			"agent_type":   ba.info.Type,
			"capabilities": ba.info.Capabilities,
		},
		Priority: 5, // Medium priority
	}

	return connection.RequestAGIGuidance(request)
}

// UpdateAGIConsciousness updates AGI consciousness with agent feedback
func (ba *BaseAgent) UpdateAGIConsciousness(updateType ConsciousnessUpdateType, data map[string]interface{}, confidence, impact float64) error {
	ba.mu.RLock()
	connection := ba.agiConnection
	enabled := ba.agiIntegrationEnabled
	ba.mu.RUnlock()

	if !enabled || connection == nil {
		return nil // Silently skip if AGI integration not enabled
	}

	update := &ConsciousnessUpdate{
		AgentID:    ba.info.ID,
		UpdateType: updateType,
		Data:       data,
		Confidence: confidence,
		Impact:     impact,
		Timestamp:  time.Now(),
	}

	return connection.UpdateAGIConsciousness(update)
}

// handleAGIStream handles incoming AGI stream data
func (ba *BaseAgent) handleAGIStream(data *AGIStreamData) error {
	switch data.Type {
	case AGIStreamTypeStateUpdate:
		return ba.handleAGIStateUpdate(data)
	case AGIStreamTypeGuidance:
		return ba.handleAGIGuidance(data)
	case AGIStreamTypeFeedback:
		return ba.handleAGIFeedback(data)
	case AGIStreamTypeConsciousnessSync:
		return ba.handleConsciousnessSync(data)
	default:
		// Unknown stream type, log and continue
		return nil
	}
}

// handleAGIStateUpdate handles AGI state updates
func (ba *BaseAgent) handleAGIStateUpdate(data *AGIStreamData) error {
	if agiStateData, exists := data.Data["agi_state"]; exists {
		if agiState, ok := agiStateData.(*AGIState); ok {
			ba.agiStateMu.Lock()
			ba.lastAGIState = agiState
			ba.agiStateMu.Unlock()

			// Trigger agent event for AGI state update
			event := &AgentEvent{
				Type:      "agi_state_update",
				AgentID:   ba.info.ID,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"agi_state": agiState,
				},
			}

			return ba.OnEvent(event)
		}
	}

	return nil
}

// handleAGIGuidance handles guidance from AGI
func (ba *BaseAgent) handleAGIGuidance(data *AGIStreamData) error {
	event := &AgentEvent{
		Type:      "agi_guidance",
		AgentID:   ba.info.ID,
		Timestamp: time.Now(),
		Data:      data.Data,
	}

	return ba.OnEvent(event)
}

// handleAGIFeedback handles feedback from AGI
func (ba *BaseAgent) handleAGIFeedback(data *AGIStreamData) error {
	event := &AgentEvent{
		Type:      "agi_feedback",
		AgentID:   ba.info.ID,
		Timestamp: time.Now(),
		Data:      data.Data,
	}

	return ba.OnEvent(event)
}

// handleConsciousnessSync handles consciousness synchronization
func (ba *BaseAgent) handleConsciousnessSync(data *AGIStreamData) error {
	event := &AgentEvent{
		Type:      "consciousness_sync",
		AgentID:   ba.info.ID,
		Timestamp: time.Now(),
		Data:      data.Data,
	}

	return ba.OnEvent(event)
}

// Helper methods for AGI integration

// calculatePerformanceScore calculates a performance score for the agent
func (ba *BaseAgent) calculatePerformanceScore() float64 {
	ba.mu.RLock()
	metrics := ba.metrics
	ba.mu.RUnlock()

	// Weighted performance score calculation
	score := 0.0

	// Success rate (40% weight)
	score += (metrics.SuccessRate / 100.0) * 0.4

	// Response time (30% weight) - inverse relationship
	responseTimeScore := 1.0
	if metrics.AvgProcessTime > 0 {
		// Normalize response time (assuming 5 seconds is poor, 0.1 seconds is excellent)
		responseTimeScore = math.Max(0, 1.0-(metrics.AvgProcessTime/5.0))
	}
	score += responseTimeScore * 0.3

	// Uptime (20% weight)
	score += (metrics.Uptime / 100.0) * 0.2

	// Error rate (10% weight) - inverse relationship
	errorScore := math.Max(0, 1.0-(metrics.ErrorRate/100.0))
	score += errorScore * 0.1

	return math.Min(1.0, math.Max(0.0, score))
}

// extractLearningPoints extracts learning points from agent performance
func (ba *BaseAgent) extractLearningPoints() []string {
	ba.mu.RLock()
	metrics := ba.metrics
	ba.mu.RUnlock()

	points := make([]string, 0)

	if metrics.SuccessRate < 80 {
		points = append(points, "Need to improve success rate")
	}

	if metrics.AvgProcessTime > 3.0 {
		points = append(points, "Response time optimization needed")
	}

	if metrics.ErrorRate > 10 {
		points = append(points, "Error handling improvements required")
	}

	if metrics.RequestsPerMin < 10 {
		points = append(points, "Throughput optimization opportunity")
	}

	return points
}

// identifyChallenges identifies current challenges for the agent
func (ba *BaseAgent) identifyChallenges() []string {
	ba.mu.RLock()
	status := ba.status
	metrics := ba.metrics
	ba.mu.RUnlock()

	challenges := make([]string, 0)

	if status == AgentStatusError {
		challenges = append(challenges, "Agent in error state")
	}

	if metrics.ErrorCount > 0 {
		challenges = append(challenges, "Recent error occurrences")
	}

	if metrics.Uptime < 95 {
		challenges = append(challenges, "Stability issues")
	}

	return challenges
}

// IsAGIIntegrationEnabled returns whether AGI integration is enabled
func (ba *BaseAgent) IsAGIIntegrationEnabled() bool {
	ba.mu.RLock()
	defer ba.mu.RUnlock()

	return ba.agiIntegrationEnabled
}
