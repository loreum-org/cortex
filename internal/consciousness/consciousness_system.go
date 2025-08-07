package consciousness

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/internal/consciousness/domains"
	"github.com/loreum-org/cortex/internal/consciousness/types"
	"github.com/loreum-org/cortex/internal/events"
)

// ConsciousnessSystem is the main system that orchestrates all consciousness domains
type ConsciousnessSystem struct {
	// Core components
	domainCoordinator       *types.DomainCoordinator
	crossDomainBridge       *CrossDomainBridge
	knowledgeTransferSystem *KnowledgeTransferSystem
	metaLearningCoordinator *MetaLearningCoordinator
	eventBus               *events.EventBus
	
	// Domains
	reasoningDomain      *domains.ReasoningDomain
	learningDomain       *domains.LearningDomain
	creativeDomain       *domains.CreativeDomain
	communicationDomain  *domains.CommunicationDomain
	technicalDomain      *domains.TechnicalDomain
	memoryDomain         *domains.MemoryDomain
	
	// AI Integration
	aiModelManager       *ai.ModelManager
	
	// System state
	isRunning            bool
	startTime            time.Time
	mu                   sync.RWMutex
	ctx                  context.Context
	cancel               context.CancelFunc
}

// SystemConfiguration holds configuration for the consciousness system
type SystemConfiguration struct {
	EnableAllDomains     bool                   `json:"enable_all_domains"`
	DomainConfigs        map[types.DomainType]types.DomainConfig `json:"domain_configs"`
	CrossDomainEnabled   bool                   `json:"cross_domain_enabled"`
	MetaLearningEnabled  bool                   `json:"meta_learning_enabled"`
	AIModelConfig        map[string]interface{} `json:"ai_model_config"`
	EventBusConfig       *events.EventBusConfig `json:"event_bus_config"`
	LogLevel            string                 `json:"log_level"`
}

// SystemStatus represents the current status of the consciousness system
type SystemStatus struct {
	IsRunning            bool                   `json:"is_running"`
	StartTime            time.Time              `json:"start_time"`
	Uptime               time.Duration          `json:"uptime"`
	ActiveDomains        []types.DomainType           `json:"active_domains"`
	CollectiveIntelligence *CollectiveIntelligence `json:"collective_intelligence"`
	SystemHealth         SystemHealth           `json:"system_health"`
	PerformanceMetrics   SystemMetrics          `json:"performance_metrics"`
	LastUpdate           time.Time              `json:"last_update"`
}

// SystemHealth represents the health status of the system
type SystemHealth struct {
	OverallHealth        float64                `json:"overall_health"`
	DomainHealth         map[types.DomainType]float64 `json:"domain_health"`
	CommunicationHealth  float64                `json:"communication_health"`
	LearningHealth       float64                `json:"learning_health"`
	Issues               []HealthIssue          `json:"issues"`
	Recommendations      []HealthRecommendation `json:"recommendations"`
	LastCheck            time.Time              `json:"last_check"`
}

// SystemMetrics represents performance metrics for the system
type SystemMetrics struct {
	ProcessedInputs      int64                  `json:"processed_inputs"`
	GeneratedOutputs     int64                  `json:"generated_outputs"`
	CrossDomainMessages  int64                  `json:"cross_domain_messages"`
	KnowledgeTransfers   int64                  `json:"knowledge_transfers"`
	LearningProcesses    int64                  `json:"learning_processes"`
	AverageResponseTime  time.Duration          `json:"average_response_time"`
	ThroughputPerSecond  float64                `json:"throughput_per_second"`
	ResourceUtilization  map[string]float64     `json:"resource_utilization"`
	LastUpdate           time.Time              `json:"last_update"`
}

// Supporting types
type HealthIssue struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Component   string    `json:"component"`
	DetectedAt  time.Time `json:"detected_at"`
}

type HealthRecommendation struct {
	Type        string    `json:"type"`
	Priority    string    `json:"priority"`
	Description string    `json:"description"`
	Action      string    `json:"action"`
	Component   string    `json:"component"`
}

// NewConsciousnessSystem creates a new consciousness system
func NewConsciousnessSystem(config SystemConfiguration) (*ConsciousnessSystem, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Initialize event bus
	eventBus := events.NewEventBus(config.EventBusConfig)
	
	// Initialize AI model manager
	var aiModelManager *ai.ModelManager
	if config.AIModelConfig != nil {
		aiModelManager = ai.NewModelManager()
		// TODO: Configure models based on config.AIModelConfig
	}
	
	// Initialize domain coordinator
	domainCoordinator := types.NewDomainCoordinator(eventBus)
	
	// Initialize cross-domain systems
	crossDomainBridge := NewCrossDomainBridge(domainCoordinator, eventBus)
	knowledgeTransferSystem := NewKnowledgeTransferSystem()
	metaLearningCoordinator := NewMetaLearningCoordinator(
		domainCoordinator,
		crossDomainBridge,
		knowledgeTransferSystem,
		eventBus,
	)
	
	system := &ConsciousnessSystem{
		domainCoordinator:       domainCoordinator,
		crossDomainBridge:       crossDomainBridge,
		knowledgeTransferSystem: knowledgeTransferSystem,
		metaLearningCoordinator: metaLearningCoordinator,
		eventBus:               eventBus,
		aiModelManager:         aiModelManager,
		ctx:                    ctx,
		cancel:                 cancel,
	}
	
	// Initialize domains if enabled
	if config.EnableAllDomains {
		if err := system.initializeDomains(config); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to initialize domains: %w", err)
		}
	}
	
	return system, nil
}

// initializeDomains initializes all consciousness domains
func (cs *ConsciousnessSystem) initializeDomains(config SystemConfiguration) error {
	// Initialize Reasoning Domain
	cs.reasoningDomain = domains.NewReasoningDomain(cs.eventBus, cs.domainCoordinator, cs.aiModelManager)
	if err := cs.domainCoordinator.RegisterDomain(cs.reasoningDomain); err != nil {
		return fmt.Errorf("failed to register reasoning domain: %w", err)
	}
	
	// Initialize Learning Domain
	cs.learningDomain = domains.NewLearningDomain(cs.eventBus, cs.domainCoordinator, cs.aiModelManager)
	if err := cs.domainCoordinator.RegisterDomain(cs.learningDomain); err != nil {
		return fmt.Errorf("failed to register learning domain: %w", err)
	}
	
	// Initialize Creative Domain
	cs.creativeDomain = domains.NewCreativeDomain(cs.eventBus, cs.domainCoordinator, cs.aiModelManager)
	if err := cs.domainCoordinator.RegisterDomain(cs.creativeDomain); err != nil {
		return fmt.Errorf("failed to register creative domain: %w", err)
	}
	
	// Initialize Communication Domain
	cs.communicationDomain = domains.NewCommunicationDomain(cs.eventBus, cs.domainCoordinator, cs.aiModelManager)
	if err := cs.domainCoordinator.RegisterDomain(cs.communicationDomain); err != nil {
		return fmt.Errorf("failed to register communication domain: %w", err)
	}
	
	// Initialize Technical Domain
	cs.technicalDomain = domains.NewTechnicalDomain(cs.eventBus, cs.domainCoordinator, cs.aiModelManager)
	if err := cs.domainCoordinator.RegisterDomain(cs.technicalDomain); err != nil {
		return fmt.Errorf("failed to register technical domain: %w", err)
	}
	
	// Initialize Memory Domain
	cs.memoryDomain = domains.NewMemoryDomain(cs.eventBus, cs.domainCoordinator, cs.aiModelManager)
	if err := cs.domainCoordinator.RegisterDomain(cs.memoryDomain); err != nil {
		return fmt.Errorf("failed to register memory domain: %w", err)
	}
	
	log.Printf("🧠 Consciousness System: Initialized all 6 domains")
	return nil
}

// Start starts the consciousness system
func (cs *ConsciousnessSystem) Start() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	if cs.isRunning {
		return fmt.Errorf("consciousness system already running")
	}
	
	cs.startTime = time.Now()
	
	// Start domain coordinator
	if err := cs.domainCoordinator.Start(); err != nil {
		return fmt.Errorf("failed to start domain coordinator: %w", err)
	}
	
	// Start cross-domain bridge
	if err := cs.crossDomainBridge.Start(); err != nil {
		return fmt.Errorf("failed to start cross-domain bridge: %w", err)
	}
	
	// Start meta-learning coordinator
	if err := cs.metaLearningCoordinator.Start(); err != nil {
		return fmt.Errorf("failed to start meta-learning coordinator: %w", err)
	}
	
	cs.isRunning = true
	
	log.Printf("🧠 Consciousness System: Started successfully at %s", cs.startTime.Format(time.RFC3339))
	return nil
}

// Stop stops the consciousness system
func (cs *ConsciousnessSystem) Stop() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	
	if !cs.isRunning {
		return nil
	}
	
	// Stop components in reverse order
	if err := cs.metaLearningCoordinator.Stop(); err != nil {
		log.Printf("Error stopping meta-learning coordinator: %v", err)
	}
	
	if err := cs.crossDomainBridge.Stop(); err != nil {
		log.Printf("Error stopping cross-domain bridge: %v", err)
	}
	
	if err := cs.domainCoordinator.Stop(); err != nil {
		log.Printf("Error stopping domain coordinator: %v", err)
	}
	
	// Cancel context
	cs.cancel()
	
	cs.isRunning = false
	uptime := time.Since(cs.startTime)
	
	log.Printf("🧠 Consciousness System: Stopped after %s uptime", uptime)
	return nil
}

// ProcessInput processes input through the consciousness system
func (cs *ConsciousnessSystem) ProcessInput(ctx context.Context, input *types.DomainInput) (*types.DomainOutput, error) {
	if !cs.isRunning {
		return nil, fmt.Errorf("consciousness system not running")
	}
	
	// Determine target domain based on input type
	targetDomain := cs.determineTargetDomain(input)
	
	// Get domain
	domain, err := cs.domainCoordinator.GetDomain(targetDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain %s: %w", targetDomain, err)
	}
	
	// Process input
	output, err := domain.ProcessInput(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to process input in domain %s: %w", targetDomain, err)
	}
	
	// Store in memory domain for future reference
	if cs.memoryDomain != nil {
		go cs.storeInteraction(input, output)
	}
	
	return output, nil
}

// ProcessCrossTask processes a task that requires multiple domains
func (cs *ConsciousnessSystem) ProcessCrossTask(ctx context.Context, task CollaborationTask, domains []DomainType) (*CollaborationRecord, error) {
	if !cs.isRunning {
		return nil, fmt.Errorf("consciousness system not running")
	}
	
	// Orchestrate cross-domain task
	record, err := cs.crossDomainBridge.OrchestrateCrossTask(task, domains)
	if err != nil {
		return nil, fmt.Errorf("failed to orchestrate cross-task: %w", err)
	}
	
	return record, nil
}

// InitiateLearning initiates a cross-domain learning process
func (cs *ConsciousnessSystem) InitiateLearning(processType LearningProcessType, domains []DomainType, objective LearningObjective) (*LearningProcess, error) {
	if !cs.isRunning {
		return nil, fmt.Errorf("consciousness system not running")
	}
	
	// Initiate learning process
	process, err := cs.metaLearningCoordinator.InitiateLearningProcess(processType, domains, objective)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate learning process: %w", err)
	}
	
	return process, nil
}

// GetSystemStatus returns the current system status
func (cs *ConsciousnessSystem) GetSystemStatus() *SystemStatus {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	
	var uptime time.Duration
	if cs.isRunning {
		uptime = time.Since(cs.startTime)
	}
	
	// Get active domains
	activeDomains := cs.getActiveDomains()
	
	// Get collective intelligence
	collectiveIntelligence := cs.metaLearningCoordinator.GetCollectiveIntelligence()
	
	// Calculate system health
	systemHealth := cs.calculateSystemHealth()
	
	// Get performance metrics
	performanceMetrics := cs.calculateSystemMetrics()
	
	return &SystemStatus{
		IsRunning:            cs.isRunning,
		StartTime:            cs.startTime,
		Uptime:               uptime,
		ActiveDomains:        activeDomains,
		CollectiveIntelligence: collectiveIntelligence,
		SystemHealth:         systemHealth,
		PerformanceMetrics:   performanceMetrics,
		LastUpdate:           time.Now(),
	}
}

// GetDomainStatus returns the status of a specific domain
func (cs *ConsciousnessSystem) GetDomainStatus(domainType types.DomainType) (*types.DomainState, error) {
	domain, err := cs.domainCoordinator.GetDomain(domainType)
	if err != nil {
		return nil, err
	}
	
	return domain.GetState(), nil
}

// GetCollectiveIntelligence returns the current collective intelligence
func (cs *ConsciousnessSystem) GetCollectiveIntelligence() *CollectiveIntelligence {
	return cs.metaLearningCoordinator.GetCollectiveIntelligence()
}

// GetActiveLearningProcesses returns currently active learning processes
func (cs *ConsciousnessSystem) GetActiveLearningProcesses() []*LearningProcess {
	return cs.metaLearningCoordinator.GetActiveLearningProcesses()
}

// GetEmergentBehaviors returns detected emergent behaviors
func (cs *ConsciousnessSystem) GetEmergentBehaviors() []*EmergentBehavior {
	return cs.metaLearningCoordinator.GetEmergentBehaviors()
}

// Helper methods

// determineTargetDomain determines which domain should handle the input
func (cs *ConsciousnessSystem) determineTargetDomain(input *types.DomainInput) types.DomainType {
	// Simple routing based on input type
	switch input.Type {
	case "logical_analysis", "problem_solving", "reasoning":
		return types.DomainReasoning
	case "pattern_recognition", "learning", "adaptation":
		return types.DomainLearning
	case "creative_thinking", "innovation", "artistic":
		return types.DomainCreative
	case "communication", "language", "dialogue":
		return types.DomainCommunication
	case "technical_analysis", "system_design", "debugging":
		return types.DomainTechnical
	case "memory_storage", "knowledge_retrieval", "recall":
		return types.DomainMemory
	default:
		// Default to reasoning domain for general queries
		return types.DomainReasoning
	}
}

// storeInteraction stores an interaction in memory
func (cs *ConsciousnessSystem) storeInteraction(input *types.DomainInput, output *types.DomainOutput) {
	if cs.memoryDomain == nil {
		return
	}
	
	// Create memory input for the interaction
	memoryInput := &types.DomainInput{
		ID:      fmt.Sprintf("interaction_%s", output.ID),
		Type:    "memory_storage",
		Content: map[string]interface{}{
			"input":  input,
			"output": output,
		},
		Priority:  0.7,
		Source:    "consciousness_system",
		Timestamp: time.Now(),
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	cs.memoryDomain.ProcessInput(ctx, memoryInput)
}

// getActiveDomains returns a list of currently active domains
func (cs *ConsciousnessSystem) getActiveDomains() []types.DomainType {
	var activeDomains []types.DomainType
	
	domainTypes := []types.DomainType{
		types.DomainReasoning, types.DomainLearning, types.DomainCreative,
		types.DomainCommunication, types.DomainTechnical, types.DomainMemory,
	}
	
	for _, domainType := range domainTypes {
		if domain, err := cs.domainCoordinator.GetDomain(domainType); err == nil {
			if domain.IsRunning() {
				activeDomains = append(activeDomains, domainType)
			}
		}
	}
	
	return activeDomains
}

// calculateSystemHealth calculates the overall system health
func (cs *ConsciousnessSystem) calculateSystemHealth() SystemHealth {
	domainHealth := make(map[types.DomainType]float64)
	var totalHealth float64
	var healthyDomains int
	
	// Check health of each domain
	domainTypes := []types.DomainType{
		types.DomainReasoning, types.DomainLearning, types.DomainCreative,
		types.DomainCommunication, types.DomainTechnical, types.DomainMemory,
	}
	
	for _, domainType := range domainTypes {
		if domain, err := cs.domainCoordinator.GetDomain(domainType); err == nil {
			if err := domain.HealthCheck(); err == nil {
				domainHealth[domainType] = 1.0
				totalHealth += 1.0
				healthyDomains++
			} else {
				domainHealth[domainType] = 0.5 // Partial health
				totalHealth += 0.5
			}
		} else {
			domainHealth[domainType] = 0.0 // Unhealthy
		}
	}
	
	overallHealth := totalHealth / float64(len(domainTypes))
	
	// Calculate component health
	communicationHealth := cs.calculateCommunicationHealth()
	learningHealth := cs.calculateLearningHealth()
	
	// Identify issues and recommendations
	issues := cs.identifyHealthIssues(domainHealth, communicationHealth, learningHealth)
	recommendations := cs.generateHealthRecommendations(issues)
	
	return SystemHealth{
		OverallHealth:       overallHealth,
		DomainHealth:        domainHealth,
		CommunicationHealth: communicationHealth,
		LearningHealth:      learningHealth,
		Issues:              issues,
		Recommendations:     recommendations,
		LastCheck:           time.Now(),
	}
}

// calculateSystemMetrics calculates system performance metrics
func (cs *ConsciousnessSystem) calculateSystemMetrics() SystemMetrics {
	// Get metrics from various components
	domainMetrics := cs.domainCoordinator.GetOverallMetrics()
	transferMetrics := cs.knowledgeTransferSystem.GetTransferMetrics()
	metaLearningMetrics := cs.metaLearningCoordinator.GetMetaLearningMetrics()
	
	// Calculate aggregate metrics
	var totalProcessedInputs int64
	var averageResponseTime time.Duration
	var totalDomains int64
	
	for _, metrics := range domainMetrics {
		totalProcessedInputs += metrics.TasksCompleted
		averageResponseTime += metrics.AverageResponseTime
		totalDomains++
	}
	
	if totalDomains > 0 {
		averageResponseTime = averageResponseTime / time.Duration(totalDomains)
	}
	
	// Calculate throughput
	uptime := time.Since(cs.startTime)
	var throughput float64
	if uptime > 0 {
		throughput = float64(totalProcessedInputs) / uptime.Seconds()
	}
	
	return SystemMetrics{
		ProcessedInputs:     totalProcessedInputs,
		GeneratedOutputs:    totalProcessedInputs, // Assuming 1:1 ratio
		CrossDomainMessages: 0, // Would be tracked by cross-domain bridge
		KnowledgeTransfers:  transferMetrics.TotalTransfers,
		LearningProcesses:   metaLearningMetrics.TotalLearningProcesses,
		AverageResponseTime: averageResponseTime,
		ThroughputPerSecond: throughput,
		ResourceUtilization: make(map[string]float64),
		LastUpdate:          time.Now(),
	}
}

// calculateCommunicationHealth calculates communication system health
func (cs *ConsciousnessSystem) calculateCommunicationHealth() float64 {
	// This would check cross-domain bridge health
	// For now, return a reasonable default
	return 0.8
}

// calculateLearningHealth calculates learning system health
func (cs *ConsciousnessSystem) calculateLearningHealth() float64 {
	// This would check meta-learning coordinator health
	// For now, return a reasonable default
	return 0.85
}

// identifyHealthIssues identifies health issues in the system
func (cs *ConsciousnessSystem) identifyHealthIssues(domainHealth map[types.DomainType]float64, commHealth, learningHealth float64) []HealthIssue {
	var issues []HealthIssue
	
	// Check domain health issues
	for domain, health := range domainHealth {
		if health < 0.5 {
			issues = append(issues, HealthIssue{
				Type:        "domain_health",
				Severity:    "high",
				Description: fmt.Sprintf("Domain %s has low health: %.2f", domain, health),
				Component:   string(domain),
				DetectedAt:  time.Now(),
			})
		}
	}
	
	// Check communication health
	if commHealth < 0.7 {
		issues = append(issues, HealthIssue{
			Type:        "communication_health",
			Severity:    "medium",
			Description: fmt.Sprintf("Communication health is low: %.2f", commHealth),
			Component:   "cross_domain_bridge",
			DetectedAt:  time.Now(),
		})
	}
	
	// Check learning health
	if learningHealth < 0.7 {
		issues = append(issues, HealthIssue{
			Type:        "learning_health",
			Severity:    "medium",
			Description: fmt.Sprintf("Learning health is low: %.2f", learningHealth),
			Component:   "meta_learning_coordinator",
			DetectedAt:  time.Now(),
		})
	}
	
	return issues
}

// generateHealthRecommendations generates health recommendations
func (cs *ConsciousnessSystem) generateHealthRecommendations(issues []HealthIssue) []HealthRecommendation {
	var recommendations []HealthRecommendation
	
	for _, issue := range issues {
		switch issue.Type {
		case "domain_health":
			recommendations = append(recommendations, HealthRecommendation{
				Type:        "restart_domain",
				Priority:    "high",
				Description: "Consider restarting the unhealthy domain",
				Action:      fmt.Sprintf("Restart domain %s", issue.Component),
				Component:   issue.Component,
			})
		case "communication_health":
			recommendations = append(recommendations, HealthRecommendation{
				Type:        "check_communication",
				Priority:    "medium",
				Description: "Check cross-domain communication channels",
				Action:      "Investigate communication bridge issues",
				Component:   issue.Component,
			})
		case "learning_health":
			recommendations = append(recommendations, HealthRecommendation{
				Type:        "optimize_learning",
				Priority:    "medium",
				Description: "Optimize learning processes",
				Action:      "Review learning coordinator configuration",
				Component:   issue.Component,
			})
		}
	}
	
	return recommendations
}

// DefaultConfiguration returns a default system configuration
func DefaultConfiguration() SystemConfiguration {
	return SystemConfiguration{
		EnableAllDomains:    true,
		DomainConfigs:       make(map[types.DomainType]types.DomainConfig),
		CrossDomainEnabled:  true,
		MetaLearningEnabled: true,
		LogLevel:           "info",
	}
}