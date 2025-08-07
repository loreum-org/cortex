package types

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/loreum-org/cortex/internal/events"
)

// BaseDomain provides common functionality for all consciousness domains
type BaseDomain struct {
	// Identity
	id         string
	domainType DomainType
	state      *DomainState
	config     DomainConfig

	// Runtime
	isRunning    bool
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.RWMutex
	
	// Communication
	eventBus     *events.EventBus
	coordinator  *DomainCoordinator
	messageQueue chan *InterDomainMessage
	
	// Task Management
	taskQueue    chan *DomainTask
	activeTasks  map[string]*DomainTask
	taskResults  map[string]*DomainResult
	
	// Learning
	feedbackQueue chan *DomainFeedback
	insights      []DomainInsight
	
	// Metrics
	metrics      DomainMetrics
	metricsTimer *time.Ticker
	
	// Resource Management
	resourceMonitor *ResourceMonitor
	
	// Lifecycle callbacks
	onStart      func(ctx context.Context) error
	onStop       func(ctx context.Context) error
	onProcess    func(ctx context.Context, input *DomainInput) (*DomainOutput, error)
	onTask       func(ctx context.Context, task *DomainTask) (*DomainResult, error)
	onLearn      func(ctx context.Context, feedback *DomainFeedback) error
	onMessage    func(ctx context.Context, message *InterDomainMessage) error
}

// ResourceMonitor tracks resource usage for a domain
type ResourceMonitor struct {
	startTime       time.Time
	cpuUsage        float64
	memoryUsage     int64
	networkCalls    int
	storageEvents   int
	executionTime   time.Duration
	mu              sync.RWMutex
}

// NewBaseDomain creates a new base domain
func NewBaseDomain(domainType DomainType, eventBus *events.EventBus, coordinator *DomainCoordinator) *BaseDomain {
	id := fmt.Sprintf("%s_%s", domainType, uuid.New().String()[:8])
	
	return &BaseDomain{
		id:           id,
		domainType:   domainType,
		eventBus:     eventBus,
		coordinator:  coordinator,
		messageQueue: make(chan *InterDomainMessage, 100),
		taskQueue:    make(chan *DomainTask, 100),
		activeTasks:  make(map[string]*DomainTask),
		taskResults:  make(map[string]*DomainResult),
		feedbackQueue: make(chan *DomainFeedback, 100),
		insights:     make([]DomainInsight, 0),
		state: &DomainState{
			Type:           domainType,
			Intelligence:   0.0,
			Capabilities:   make(map[string]float64),
			ActiveGoals:    make([]DomainGoal, 0),
			RecentInsights: make([]DomainInsight, 0),
			Configuration:  DefaultDomainConfig(),
			Metrics:        DefaultDomainMetrics(),
			LastUpdate:     time.Now(),
			Version:        1,
		},
		config: DefaultDomainConfig(),
		metrics: DefaultDomainMetrics(),
		resourceMonitor: &ResourceMonitor{
			startTime: time.Now(),
		},
	}
}

// DefaultDomainConfig returns default configuration for a domain
func DefaultDomainConfig() DomainConfig {
	return DomainConfig{
		MaxConcurrentTasks: 10,
		LearningRate:       0.1,
		ForgettingRate:     0.01,
		InsightThreshold:   0.7,
		GoalGenerationRate: 0.1,
		ResourceLimits: DomainResourceLimits{
			MaxMemoryMB:      256,
			MaxCPUPercent:    25.0,
			MaxExecutionTime: 30 * time.Second,
			MaxStorageEvents: 1000,
			MaxNetworkCalls:  100,
		},
		SpecializationParams: make(map[string]interface{}),
	}
}

// DefaultDomainMetrics returns default metrics for a domain
func DefaultDomainMetrics() DomainMetrics {
	return DomainMetrics{
		TasksCompleted:        0,
		SuccessRate:           1.0,
		AverageResponseTime:   time.Millisecond * 100,
		LearningVelocity:      0.0,
		InsightGenerationRate: 0.0,
		ResourceUtilization:   make(map[string]float64),
		ErrorRate:             0.0,
		LastMetricUpdate:      time.Now(),
	}
}

// Interface implementation

// GetType returns the domain type
func (bd *BaseDomain) GetType() DomainType {
	return bd.domainType
}

// GetID returns the domain ID
func (bd *BaseDomain) GetID() string {
	return bd.id
}

// GetState returns the current domain state
func (bd *BaseDomain) GetState() *DomainState {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	stateCopy := *bd.state
	return &stateCopy
}

// Configure updates the domain configuration
func (bd *BaseDomain) Configure(config DomainConfig) error {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	bd.config = config
	bd.state.Configuration = config
	bd.state.LastUpdate = time.Now()
	bd.state.Version++
	
	return nil
}

// ProcessInput processes input through the domain
func (bd *BaseDomain) ProcessInput(ctx context.Context, input *DomainInput) (*DomainOutput, error) {
	if bd.onProcess != nil {
		return bd.onProcess(ctx, input)
	}
	
	// Default implementation
	return &DomainOutput{
		ID:          uuid.New().String(),
		InputID:     input.ID,
		Type:        "processed",
		Content:     input.Content,
		Confidence:  0.5,
		Quality:     0.5,
		Metadata:    map[string]interface{}{"domain": bd.domainType},
		Timestamp:   time.Now(),
		ProcessTime: time.Millisecond * 10,
	}, nil
}

// ProcessTask processes a task
func (bd *BaseDomain) ProcessTask(ctx context.Context, task *DomainTask) (*DomainResult, error) {
	startTime := time.Now()
	
	// Check resource limits
	if err := bd.checkResourceLimits(); err != nil {
		return &DomainResult{
			TaskID:      task.ID,
			Status:      TaskStatusFailed,
			Error:       fmt.Sprintf("resource limit exceeded: %v", err),
			CompletedAt: time.Now(),
			Duration:    time.Since(startTime),
		}, err
	}
	
	// Add to active tasks
	bd.mu.Lock()
	bd.activeTasks[task.ID] = task
	bd.mu.Unlock()
	
	defer func() {
		bd.mu.Lock()
		delete(bd.activeTasks, task.ID)
		bd.mu.Unlock()
	}()
	
	// Call custom task handler if available
	if bd.onTask != nil {
		result, err := bd.onTask(ctx, task)
		bd.updateMetricsFromResult(result, err)
		return result, err
	}
	
	// Default implementation
	result := &DomainResult{
		TaskID:      task.ID,
		Status:      TaskStatusCompleted,
		Result:      fmt.Sprintf("Task %s completed by domain %s", task.Type, bd.domainType),
		Insights:    []DomainInsight{},
		Metrics:     map[string]float64{"processing_time": float64(time.Since(startTime).Milliseconds())},
		Metadata:    map[string]interface{}{"domain": bd.domainType},
		CompletedAt: time.Now(),
		Duration:    time.Since(startTime),
	}
	
	bd.updateMetricsFromResult(result, nil)
	return result, nil
}

// Learn processes feedback for learning
func (bd *BaseDomain) Learn(ctx context.Context, feedback *DomainFeedback) error {
	if bd.onLearn != nil {
		return bd.onLearn(ctx, feedback)
	}
	
	// Default learning implementation
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	// Update capabilities based on feedback
	learningRate := bd.config.LearningRate
	if feedback.Type == FeedbackTypeQuality {
		if quality, exists := bd.state.Capabilities["quality"]; exists {
			bd.state.Capabilities["quality"] = quality + (feedback.Rating-quality)*learningRate
		} else {
			bd.state.Capabilities["quality"] = feedback.Rating * learningRate
		}
	}
	
	// Update intelligence
	intelligenceGain := (feedback.Rating - 0.5) * learningRate * 10 // Scale to 0-100
	bd.state.Intelligence = bd.state.Intelligence + intelligenceGain
	if bd.state.Intelligence > 100 {
		bd.state.Intelligence = 100
	}
	if bd.state.Intelligence < 0 {
		bd.state.Intelligence = 0
	}
	
	bd.state.LastUpdate = time.Now()
	bd.state.Version++
	
	return nil
}

// GenerateInsights generates insights from recent activities
func (bd *BaseDomain) GenerateInsights(ctx context.Context) ([]DomainInsight, error) {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	
	// Simple insight generation based on metrics
	insights := []DomainInsight{}
	
	// Generate insight if success rate is high
	if bd.metrics.SuccessRate > 0.9 {
		insight := DomainInsight{
			ID:         uuid.New().String(),
			Type:       "performance",
			Content:    fmt.Sprintf("Domain %s achieving high success rate: %.2f", bd.domainType, bd.metrics.SuccessRate),
			Confidence: 0.8,
			Relevance:  0.9,
			Sources:    []string{"metrics"},
			Impact: DomainImpact{
				CapabilityChanges: map[string]float64{"reliability": 0.1},
				IntelligenceGain:  1.0,
			},
			CreatedAt: time.Now(),
		}
		insights = append(insights, insight)
	}
	
	// Generate insight if learning velocity is high
	if bd.metrics.LearningVelocity > bd.config.InsightThreshold {
		insight := DomainInsight{
			ID:         uuid.New().String(),
			Type:       "learning",
			Content:    fmt.Sprintf("Domain %s showing rapid learning: %.2f", bd.domainType, bd.metrics.LearningVelocity),
			Confidence: 0.7,
			Relevance:  0.8,
			Sources:    []string{"learning_metrics"},
			Impact: DomainImpact{
				CapabilityChanges: map[string]float64{"learning_efficiency": 0.15},
				IntelligenceGain:  2.0,
			},
			CreatedAt: time.Now(),
		}
		insights = append(insights, insight)
	}
	
	return insights, nil
}

// EvolveCapabilities evolves domain capabilities
func (bd *BaseDomain) EvolveCapabilities(ctx context.Context) error {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	// Apply forgetting rate to reduce unused capabilities
	forgettingRate := bd.config.ForgettingRate
	for capability, value := range bd.state.Capabilities {
		bd.state.Capabilities[capability] = value * (1 - forgettingRate)
		if bd.state.Capabilities[capability] < 0.01 {
			delete(bd.state.Capabilities, capability)
		}
	}
	
	// Generate new goals based on performance
	if bd.metrics.SuccessRate < 0.8 {
		goal := DomainGoal{
			ID:          uuid.New().String(),
			Description: "Improve task success rate",
			Priority:    0.8,
			Progress:    0.0,
			Context:     map[string]interface{}{"current_rate": bd.metrics.SuccessRate},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		bd.state.ActiveGoals = append(bd.state.ActiveGoals, goal)
	}
	
	bd.state.LastUpdate = time.Now()
	bd.state.Version++
	
	return nil
}

// SetGoals sets domain goals
func (bd *BaseDomain) SetGoals(goals []DomainGoal) error {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	bd.state.ActiveGoals = goals
	bd.state.LastUpdate = time.Now()
	bd.state.Version++
	
	return nil
}

// GetGoals returns current domain goals
func (bd *BaseDomain) GetGoals() []DomainGoal {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	
	goals := make([]DomainGoal, len(bd.state.ActiveGoals))
	copy(goals, bd.state.ActiveGoals)
	return goals
}

// UpdateGoalProgress updates progress for a specific goal
func (bd *BaseDomain) UpdateGoalProgress(goalID string, progress float64) error {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	for i, goal := range bd.state.ActiveGoals {
		if goal.ID == goalID {
			bd.state.ActiveGoals[i].Progress = progress
			bd.state.ActiveGoals[i].UpdatedAt = time.Now()
			
			// Remove completed goals
			if progress >= 1.0 {
				bd.state.ActiveGoals = append(bd.state.ActiveGoals[:i], bd.state.ActiveGoals[i+1:]...)
			}
			
			bd.state.LastUpdate = time.Now()
			bd.state.Version++
			return nil
		}
	}
	
	return fmt.Errorf("goal %s not found", goalID)
}

// SendMessage sends a message to another domain
func (bd *BaseDomain) SendMessage(ctx context.Context, targetDomain DomainType, message *InterDomainMessage) error {
	message.FromDomain = bd.domainType
	message.ToDomain = targetDomain
	message.Timestamp = time.Now()
	
	if message.ID == "" {
		message.ID = uuid.New().String()
	}
	
	return bd.coordinator.SendMessage(message)
}

// ReceiveMessage receives a message from another domain
func (bd *BaseDomain) ReceiveMessage(ctx context.Context, message *InterDomainMessage) error {
	if bd.onMessage != nil {
		return bd.onMessage(ctx, message)
	}
	
	// Default message handling
	select {
	case bd.messageQueue <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("message queue full for domain %s", bd.domainType)
	}
}

// Start starts the domain
func (bd *BaseDomain) Start(ctx context.Context) error {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	if bd.isRunning {
		return fmt.Errorf("domain %s already running", bd.domainType)
	}
	
	bd.ctx, bd.cancel = context.WithCancel(ctx)
	
	// Start message processing
	go bd.processMessages()
	
	// Start task processing
	go bd.processTasks()
	
	// Start feedback processing
	go bd.processFeedback()
	
	// Start metrics collection
	bd.metricsTimer = time.NewTicker(10 * time.Second)
	go bd.collectMetrics()
	
	// Start resource monitoring
	go bd.monitorResources()
	
	// Call custom start handler
	if bd.onStart != nil {
		if err := bd.onStart(bd.ctx); err != nil {
			bd.cancel()
			return err
		}
	}
	
	bd.isRunning = true
	log.Printf("🧠 Domain %s (%s) started", bd.domainType, bd.id)
	
	return nil
}

// Stop stops the domain
func (bd *BaseDomain) Stop(ctx context.Context) error {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	if !bd.isRunning {
		return nil
	}
	
	// Call custom stop handler
	if bd.onStop != nil {
		bd.onStop(bd.ctx)
	}
	
	// Stop timers
	if bd.metricsTimer != nil {
		bd.metricsTimer.Stop()
	}
	
	// Cancel context
	if bd.cancel != nil {
		bd.cancel()
	}
	
	bd.isRunning = false
	log.Printf("🧠 Domain %s (%s) stopped", bd.domainType, bd.id)
	
	return nil
}

// IsRunning returns whether the domain is running
func (bd *BaseDomain) IsRunning() bool {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	return bd.isRunning
}

// GetMetrics returns current domain metrics
func (bd *BaseDomain) GetMetrics() DomainMetrics {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	return bd.metrics
}

// HealthCheck performs a health check
func (bd *BaseDomain) HealthCheck() error {
	if !bd.isRunning {
		return fmt.Errorf("domain %s is not running", bd.domainType)
	}
	
	// Check resource usage
	if err := bd.checkResourceLimits(); err != nil {
		return fmt.Errorf("resource limits exceeded: %w", err)
	}
	
	// Check if context is cancelled
	select {
	case <-bd.ctx.Done():
		return fmt.Errorf("domain context cancelled")
	default:
	}
	
	return nil
}

// Helper methods

// processMessages processes incoming messages
func (bd *BaseDomain) processMessages() {
	for {
		select {
		case message := <-bd.messageQueue:
			bd.handleMessage(message)
		case <-bd.ctx.Done():
			return
		}
	}
}

// processTasks processes tasks from the task queue
func (bd *BaseDomain) processTasks() {
	for {
		select {
		case task := <-bd.taskQueue:
			go bd.handleTask(task)
		case <-bd.ctx.Done():
			return
		}
	}
}

// processFeedback processes feedback for learning
func (bd *BaseDomain) processFeedback() {
	for {
		select {
		case feedback := <-bd.feedbackQueue:
			bd.Learn(bd.ctx, feedback)
		case <-bd.ctx.Done():
			return
		}
	}
}

// collectMetrics collects and updates domain metrics
func (bd *BaseDomain) collectMetrics() {
	for {
		select {
		case <-bd.metricsTimer.C:
			bd.updateMetrics()
		case <-bd.ctx.Done():
			return
		}
	}
}

// monitorResources monitors resource usage
func (bd *BaseDomain) monitorResources() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			bd.resourceMonitor.mu.Lock()
			// Update resource metrics (simplified)
			bd.resourceMonitor.executionTime = time.Since(bd.resourceMonitor.startTime)
			bd.resourceMonitor.mu.Unlock()
		case <-bd.ctx.Done():
			return
		}
	}
}

// handleMessage handles an incoming inter-domain message
func (bd *BaseDomain) handleMessage(message *InterDomainMessage) {
	// Default message handling - can be overridden
	log.Printf("🧠 Domain %s received message from %s: %s", bd.domainType, message.FromDomain, message.Type)
}

// handleTask handles a domain task
func (bd *BaseDomain) handleTask(task *DomainTask) {
	ctx, cancel := context.WithTimeout(bd.ctx, bd.config.ResourceLimits.MaxExecutionTime)
	defer cancel()
	
	result, err := bd.ProcessTask(ctx, task)
	
	bd.mu.Lock()
	bd.taskResults[task.ID] = result
	bd.mu.Unlock()
	
	if err != nil {
		log.Printf("🧠 Domain %s task %s failed: %v", bd.domainType, task.ID, err)
	}
}

// updateMetrics updates domain metrics
func (bd *BaseDomain) updateMetrics() {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	// Update metrics based on recent performance
	bd.metrics.LastMetricUpdate = time.Now()
	
	// Calculate success rate from recent tasks
	completedTasks := len(bd.taskResults)
	if completedTasks > 0 {
		successCount := 0
		for _, result := range bd.taskResults {
			if result.Status == TaskStatusCompleted {
				successCount++
			}
		}
		bd.metrics.SuccessRate = float64(successCount) / float64(completedTasks)
	}
	
	// Update resource utilization
	bd.resourceMonitor.mu.RLock()
	bd.metrics.ResourceUtilization = map[string]float64{
		"memory":    float64(bd.resourceMonitor.memoryUsage) / 1024 / 1024, // MB
		"cpu":       bd.resourceMonitor.cpuUsage,
		"network":   float64(bd.resourceMonitor.networkCalls),
		"storage":   float64(bd.resourceMonitor.storageEvents),
	}
	bd.resourceMonitor.mu.RUnlock()
}

// updateMetricsFromResult updates metrics based on task result
func (bd *BaseDomain) updateMetricsFromResult(result *DomainResult, err error) {
	bd.mu.Lock()
	defer bd.mu.Unlock()
	
	bd.metrics.TasksCompleted++
	bd.metrics.AverageResponseTime = (bd.metrics.AverageResponseTime + result.Duration) / 2
	
	if err != nil {
		bd.metrics.ErrorRate = (bd.metrics.ErrorRate + 1.0) / 2.0
	} else {
		bd.metrics.ErrorRate = bd.metrics.ErrorRate * 0.9 // Decay error rate
	}
}

// checkResourceLimits checks if resource limits are exceeded
func (bd *BaseDomain) checkResourceLimits() error {
	bd.resourceMonitor.mu.RLock()
	defer bd.resourceMonitor.mu.RUnlock()
	
	limits := bd.config.ResourceLimits
	
	if bd.resourceMonitor.memoryUsage > int64(limits.MaxMemoryMB)*1024*1024 {
		return fmt.Errorf("memory limit exceeded: %d MB > %d MB", 
			bd.resourceMonitor.memoryUsage/1024/1024, limits.MaxMemoryMB)
	}
	
	if bd.resourceMonitor.cpuUsage > limits.MaxCPUPercent {
		return fmt.Errorf("CPU limit exceeded: %.2f%% > %.2f%%", 
			bd.resourceMonitor.cpuUsage, limits.MaxCPUPercent)
	}
	
	if bd.resourceMonitor.networkCalls > limits.MaxNetworkCalls {
		return fmt.Errorf("network calls limit exceeded: %d > %d", 
			bd.resourceMonitor.networkCalls, limits.MaxNetworkCalls)
	}
	
	return nil
}

// SetCustomHandlers allows setting custom handlers for domain behavior
func (bd *BaseDomain) SetCustomHandlers(
	onStart func(ctx context.Context) error,
	onStop func(ctx context.Context) error,
	onProcess func(ctx context.Context, input *DomainInput) (*DomainOutput, error),
	onTask func(ctx context.Context, task *DomainTask) (*DomainResult, error),
	onLearn func(ctx context.Context, feedback *DomainFeedback) error,
	onMessage func(ctx context.Context, message *InterDomainMessage) error,
) {
	bd.onStart = onStart
	bd.onStop = onStop
	bd.onProcess = onProcess
	bd.onTask = onTask
	bd.onLearn = onLearn
	bd.onMessage = onMessage
}