package rag

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/pkg/types"
)

// ConsciousnessRuntime implements the consciousness loop pattern for AGI
// This is the core "runtime orchestration loop" that sits atop deeper unconscious processes
type ConsciousnessRuntime struct {
	agiSystem      *AGIPromptSystem
	ragSystem      *RAGSystem
	contextManager *ContextManager
	nodeID         string
	
	// Consciousness state
	currentState   *ConsciousnessState
	workingMemory  *WorkingMemory
	intentAnalyzer *IntentAnalyzer
	decisionEngine *DecisionEngine
	actionExecutor *ActionExecutor
	
	// Runtime control
	isRunning      bool
	mu             sync.RWMutex
	stopChan       chan struct{}
	cycleInterval  time.Duration
	
	// Sensors and inputs
	inputQueue     chan *SensorInput
	pendingInputs  []*SensorInput
	
	// Metrics and monitoring
	cycleCount     int64
	lastCycleTime  time.Time
	avgCycleTime   time.Duration
}

// ConsciousnessState represents the current state of consciousness
type ConsciousnessState struct {
	CurrentCycle    int64                  `json:"current_cycle"`
	Attention       *AttentionState        `json:"attention"`
	Emotions        map[string]float64     `json:"emotions"`
	Goals           []ConsciousGoal        `json:"goals"`
	ActiveThoughts  []Thought              `json:"active_thoughts"`
	Intentions      []Intent               `json:"intentions"`
	LastUpdate      time.Time              `json:"last_update"`
	EnergyLevel     float64                `json:"energy_level"`     // 0-1 scale
	FocusLevel      float64                `json:"focus_level"`      // 0-1 scale
	AlertnessLevel  float64                `json:"alertness_level"`  // 0-1 scale
}

// AttentionState tracks what the consciousness is focused on
type AttentionState struct {
	PrimaryFocus    string                 `json:"primary_focus"`
	SecondaryFoci   []string               `json:"secondary_foci"`
	FocusStrength   float64                `json:"focus_strength"`
	FocusDuration   time.Duration          `json:"focus_duration"`
	DistractionLevel float64               `json:"distraction_level"`
}

// WorkingMemory holds the current context and recent experiences
type WorkingMemory struct {
	RecentInputs    []*SensorInput         `json:"recent_inputs"`
	ActiveContext   map[string]interface{} `json:"active_context"`
	ShortTermMemory []MemoryFragment       `json:"short_term_memory"`
	Goals           []Goal                 `json:"goals"`
	Beliefs         map[string]float64     `json:"beliefs"`
	Predictions     []Prediction           `json:"predictions"`
	LastUpdated     time.Time              `json:"last_updated"`
}

// SensorInput represents input from the environment
type SensorInput struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "query", "system", "feedback", "observation"
	Content     string                 `json:"content"`
	Metadata    map[string]interface{} `json:"metadata"`
	Timestamp   time.Time              `json:"timestamp"`
	Priority    float64                `json:"priority"` // 0-1 scale
	Source      string                 `json:"source"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// Intent represents the understood intention from input
type Intent struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"` // "question", "request", "command", "conversation"
	Description     string                 `json:"description"`
	Confidence      float64                `json:"confidence"`
	RequiredActions []string               `json:"required_actions"`
	ExpectedOutcome string                 `json:"expected_outcome"`
	Priority        float64                `json:"priority"`
	Metadata        map[string]interface{} `json:"metadata"`
	Timestamp       time.Time              `json:"timestamp"`
}

// Decision represents a decision made by the consciousness
type Decision struct {
	ID              string                 `json:"id"`
	Intent          *Intent                `json:"intent"`
	ChosenAction    string                 `json:"chosen_action"`
	ActionParams    map[string]interface{} `json:"action_params"`
	Reasoning       string                 `json:"reasoning"`
	Confidence      float64                `json:"confidence"`
	ExpectedOutcome string                 `json:"expected_outcome"`
	Alternatives    []string               `json:"alternatives"`
	RiskAssessment  float64                `json:"risk_assessment"`
	Timestamp       time.Time              `json:"timestamp"`
}

// ActionResult represents the outcome of an executed action
type ActionResult struct {
	DecisionID      string                 `json:"decision_id"`
	Action          string                 `json:"action"`
	Success         bool                   `json:"success"`
	Result          interface{}            `json:"result"`
	Duration        time.Duration          `json:"duration"`
	ErrorMsg        string                 `json:"error_msg,omitempty"`
	Feedback        map[string]interface{} `json:"feedback,omitempty"`
	LearningPoints  []string               `json:"learning_points,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
}

// ConsciousGoal represents a high-level goal the consciousness is working toward
type ConsciousGoal struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Priority    float64   `json:"priority"`
	Progress    float64   `json:"progress"`
	Deadline    time.Time `json:"deadline,omitempty"`
	Status      string    `json:"status"` // "active", "paused", "completed", "abandoned"
}

// Thought represents an active thought in consciousness
type Thought struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	Type        string                 `json:"type"` // "analytical", "creative", "memory", "prediction"
	Confidence  float64                `json:"confidence"`
	Relevance   float64                `json:"relevance"`
	Associations []string              `json:"associations"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MemoryFragment represents a piece of short-term memory
type MemoryFragment struct {
	ID          string                 `json:"id"`
	Content     string                 `json:"content"`
	Type        string                 `json:"type"`
	Importance  float64                `json:"importance"`
	Associations []string              `json:"associations"`
	Timestamp   time.Time              `json:"timestamp"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Prediction represents a prediction about future events
type Prediction struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Confidence  float64   `json:"confidence"`
	Timeframe   string    `json:"timeframe"`
	Basis       []string  `json:"basis"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewConsciousnessRuntime creates a new consciousness runtime
func NewConsciousnessRuntime(agiSystem *AGIPromptSystem, ragSystem *RAGSystem, contextManager *ContextManager, nodeID string) *ConsciousnessRuntime {
	runtime := &ConsciousnessRuntime{
		agiSystem:      agiSystem,
		ragSystem:      ragSystem,
		contextManager: contextManager,
		nodeID:         nodeID,
		cycleInterval:  time.Millisecond * 100, // 10 Hz consciousness cycle
		inputQueue:     make(chan *SensorInput, 100),
		stopChan:       make(chan struct{}),
		pendingInputs:  make([]*SensorInput, 0),
	}
	
	// Initialize consciousness state
	runtime.currentState = &ConsciousnessState{
		CurrentCycle:   0,
		Attention:      &AttentionState{FocusStrength: 0.5},
		Emotions:       make(map[string]float64),
		Goals:          make([]ConsciousGoal, 0),
		ActiveThoughts: make([]Thought, 0),
		Intentions:     make([]Intent, 0),
		LastUpdate:     time.Now(),
		EnergyLevel:    1.0,
		FocusLevel:     0.7,
		AlertnessLevel: 0.8,
	}
	
	// Initialize working memory
	runtime.workingMemory = &WorkingMemory{
		RecentInputs:    make([]*SensorInput, 0, 10),
		ActiveContext:   make(map[string]interface{}),
		ShortTermMemory: make([]MemoryFragment, 0),
		Goals:           make([]Goal, 0),
		Beliefs:         make(map[string]float64),
		Predictions:     make([]Prediction, 0),
		LastUpdated:     time.Now(),
	}
	
	// Initialize components
	runtime.intentAnalyzer = NewIntentAnalyzer(agiSystem)
	runtime.decisionEngine = NewDecisionEngine(agiSystem)
	runtime.actionExecutor = NewActionExecutor(ragSystem, agiSystem)
	
	return runtime
}

// SetModelManager configures the consciousness runtime with access to AI models
func (cr *ConsciousnessRuntime) SetModelManager(modelManager interface{}, defaultModel string) {
	if cr.actionExecutor != nil {
		if mm, ok := modelManager.(*ai.ModelManager); ok {
			cr.actionExecutor.SetModelManager(mm, defaultModel)
		}
	}
}

// Start begins the consciousness runtime loop
func (cr *ConsciousnessRuntime) Start(ctx context.Context) error {
	cr.mu.Lock()
	if cr.isRunning {
		cr.mu.Unlock()
		return fmt.Errorf("consciousness runtime already running")
	}
	cr.isRunning = true
	cr.mu.Unlock()
	
	log.Printf("[Consciousness] Starting consciousness runtime for node %s", cr.nodeID)
	
	// Start the main consciousness loop
	go cr.consciousnessLoop(ctx)
	
	return nil
}

// Stop halts the consciousness runtime
func (cr *ConsciousnessRuntime) Stop() error {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	
	if !cr.isRunning {
		return fmt.Errorf("consciousness runtime not running")
	}
	
	close(cr.stopChan)
	cr.isRunning = false
	
	log.Printf("[Consciousness] Stopping consciousness runtime for node %s", cr.nodeID)
	return nil
}

// consciousnessLoop implements the main consciousness loop
func (cr *ConsciousnessRuntime) consciousnessLoop(ctx context.Context) {
	ticker := time.NewTicker(cr.cycleInterval)
	defer ticker.Stop()
	
	log.Printf("[Consciousness] Beginning consciousness loop with %v cycle interval", cr.cycleInterval)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-cr.stopChan:
			return
		case <-ticker.C:
			cr.executeCycle(ctx)
		}
	}
}

// executeCycle executes one complete consciousness cycle
func (cr *ConsciousnessRuntime) executeCycle(ctx context.Context) {
	cycleStart := time.Now()
	cr.mu.Lock()
	cr.cycleCount++
	currentCycle := cr.cycleCount
	cr.mu.Unlock()
	
	// Update consciousness state
	cr.currentState.CurrentCycle = currentCycle
	cr.currentState.LastUpdate = cycleStart
	
	// 1. ReadSensors() - Gather inputs from environment
	inputs := cr.readSensors(ctx)
	
	// 2. LoadWorkingMemory() - Load current context and memory
	context := cr.loadWorkingMemory(ctx)
	
	// 3. EvaluateIntent() - Understand what needs to be done
	intents := cr.evaluateIntent(ctx, inputs, context)
	
	// 4. MakeDecision() - Decide on actions to take
	decisions := cr.makeDecision(ctx, intents, context)
	
	// 5. ExecuteAction() - Take the decided actions
	results := cr.executeAction(ctx, decisions)
	
	// 6. UpdateMemory() - Learn from the experience
	cr.updateMemory(ctx, inputs, decisions, results)
	
	// Update cycle metrics
	cycleTime := time.Since(cycleStart)
	cr.lastCycleTime = cycleStart
	if cr.avgCycleTime == 0 {
		cr.avgCycleTime = cycleTime
	} else {
		cr.avgCycleTime = (cr.avgCycleTime + cycleTime) / 2
	}
	
	// Log significant cycles
	if len(inputs) > 0 || len(decisions) > 0 {
		log.Printf("[Consciousness] Cycle %d: %d inputs, %d intents, %d decisions, %d results (%v)", 
			currentCycle, len(inputs), len(intents), len(decisions), len(results), cycleTime)
	}
}

// ProcessQuery processes a user query through the consciousness loop
func (cr *ConsciousnessRuntime) ProcessQuery(ctx context.Context, query *types.Query) (*types.Response, error) {
	// Create sensor input for the query
	input := &SensorInput{
		ID:        fmt.Sprintf("query_%s", query.ID),
		Type:      "query",
		Content:   query.Text,
		Metadata:  map[string]interface{}{"query_type": query.Type, "query_id": query.ID},
		Timestamp: time.Now(),
		Priority:  1.0, // High priority for direct queries
		Source:    "user",
		Context:   map[string]interface{}{"metadata": query.Metadata},
	}
	
	// Add to input queue
	select {
	case cr.inputQueue <- input:
		// Input queued successfully
	default:
		// Queue full, process immediately in sync mode
		log.Printf("[Consciousness] Input queue full, processing query synchronously")
		return cr.processQueryDirectly(ctx, query, input)
	}
	
	// Wait for the consciousness loop to process this input and generate a response
	// For now, we'll process it directly to maintain compatibility
	return cr.processQueryDirectly(ctx, query, input)
}

// processQueryDirectly processes a query directly through consciousness loop steps
func (cr *ConsciousnessRuntime) processQueryDirectly(ctx context.Context, query *types.Query, input *SensorInput) (*types.Response, error) {
	startTime := time.Now()
	
	log.Printf("[Consciousness] Processing query directly: %s", query.Text)
	
	// 1. ReadSensors - we already have the input
	inputs := []*SensorInput{input}
	
	// 2. LoadWorkingMemory
	context := cr.loadWorkingMemory(ctx)
	
	// 3. EvaluateIntent
	intents := cr.evaluateIntent(ctx, inputs, context)
	
	// 4. MakeDecision
	decisions := cr.makeDecision(ctx, intents, context)
	
	// 5. ExecuteAction
	results := cr.executeAction(ctx, decisions)
	
	// 6. UpdateMemory
	cr.updateMemory(ctx, inputs, decisions, results)
	
	// Format response from results
	if len(results) > 0 {
		result := results[0]
		if result.Success {
			return &types.Response{
				QueryID:   query.ID,
				Text:      fmt.Sprintf("%v", result.Result),
				Status:    "success",
				Timestamp: time.Now().Unix(),
				Metadata: map[string]string{
					"consciousness_cycle": fmt.Sprintf("%d", cr.currentState.CurrentCycle),
					"processing_time":     time.Since(startTime).String(),
					"decision_id":         result.DecisionID,
					"action_type":         result.Action,
					"confidence":          fmt.Sprintf("%.2f", 0.85), // TODO: Get from decision
				},
			}, nil
		} else {
			return nil, fmt.Errorf("consciousness processing failed: %s", result.ErrorMsg)
		}
	}
	
	return &types.Response{
		QueryID:   query.ID,
		Text:      "I understand your request, but I'm still processing how to respond appropriately.",
		Status:    "partial",
		Timestamp: time.Now().Unix(),
		Metadata: map[string]string{
			"consciousness_cycle": fmt.Sprintf("%d", cr.currentState.CurrentCycle),
			"processing_time":     time.Since(startTime).String(),
		},
	}, nil
}

// GetConsciousnessState returns the current consciousness state
func (cr *ConsciousnessRuntime) GetConsciousnessState() *ConsciousnessState {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	state := *cr.currentState
	return &state
}

// GetWorkingMemory returns the current working memory
func (cr *ConsciousnessRuntime) GetWorkingMemory() *WorkingMemory {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	memory := *cr.workingMemory
	return &memory
}

// GetMetrics returns consciousness runtime metrics
func (cr *ConsciousnessRuntime) GetMetrics() map[string]interface{} {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	
	return map[string]interface{}{
		"cycle_count":      cr.cycleCount,
		"is_running":       cr.isRunning,
		"avg_cycle_time":   cr.avgCycleTime.String(),
		"last_cycle_time":  cr.lastCycleTime.Format(time.RFC3339),
		"input_queue_size": len(cr.inputQueue),
		"pending_inputs":   len(cr.pendingInputs),
		"energy_level":     cr.currentState.EnergyLevel,
		"focus_level":      cr.currentState.FocusLevel,
		"alertness_level":  cr.currentState.AlertnessLevel,
	}
}