package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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
	isRunning     bool
	mu            sync.RWMutex
	stopChan      chan struct{}
	cycleInterval time.Duration

	// Sensors and inputs
	inputQueue    chan *SensorInput
	pendingInputs []*SensorInput

	// Metrics and monitoring
	cycleCount    int64
	lastCycleTime time.Time
	avgCycleTime  time.Duration
}

// ConsciousnessState represents the current state of consciousness
type ConsciousnessState struct {
	CurrentCycle   int64              `json:"current_cycle"`
	Attention      *AttentionState    `json:"attention"`
	Emotions       map[string]float64 `json:"emotions"`
	Goals          []ConsciousGoal    `json:"goals"`
	ActiveThoughts []Thought          `json:"active_thoughts"`
	Intentions     []Intent           `json:"intentions"`
	LastUpdate     time.Time          `json:"last_update"`
	EnergyLevel    float64            `json:"energy_level"`    // 0-1 scale
	FocusLevel     float64            `json:"focus_level"`     // 0-1 scale
	AlertnessLevel float64            `json:"alertness_level"` // 0-1 scale
	WorkingMemory  *WorkingMemory     `json:"working_memory"`
	DecisionState  string             `json:"decision_state"`
	ProcessingLoad float64            `json:"processing_load"`
}

// AttentionState tracks what the consciousness is focused on
type AttentionState struct {
	PrimaryFocus     string        `json:"primary_focus"`
	SecondaryFoci    []string      `json:"secondary_foci"`
	FocusStrength    float64       `json:"focus_strength"`
	FocusDuration    time.Duration `json:"focus_duration"`
	DistractionLevel float64       `json:"distraction_level"`
	CurrentFocus     string        `json:"current_focus"`
	Intensity        float64       `json:"intensity"`
	Duration         time.Duration `json:"duration"`
	Context          string        `json:"context"`
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
	CurrentContext  map[string]interface{} `json:"current_context"`
	ActiveBeliefs   map[string]float64     `json:"active_beliefs"`
	TemporalContext map[string]interface{} `json:"temporal_context"`
}

// SensorInput represents input from the environment
type SensorInput struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // "query", "system", "feedback", "observation"
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	Priority  float64                `json:"priority"` // 0-1 scale
	Source    string                 `json:"source"`
	Context   map[string]interface{} `json:"context,omitempty"`
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
	DecisionID     string                 `json:"decision_id"`
	Action         string                 `json:"action"`
	Success        bool                   `json:"success"`
	Result         interface{}            `json:"result"`
	Duration       time.Duration          `json:"duration"`
	ErrorMsg       string                 `json:"error_msg,omitempty"`
	Feedback       map[string]interface{} `json:"feedback,omitempty"`
	LearningPoints []string               `json:"learning_points,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
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
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	Type         string                 `json:"type"` // "analytical", "creative", "memory", "prediction"
	Confidence   float64                `json:"confidence"`
	Relevance    float64                `json:"relevance"`
	Associations []string               `json:"associations"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// MemoryFragment represents a piece of short-term memory
type MemoryFragment struct {
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	Type         string                 `json:"type"`
	Importance   float64                `json:"importance"`
	Associations []string               `json:"associations"`
	Timestamp    time.Time              `json:"timestamp"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
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
		cycleInterval:  time.Second * 2, // 0.5 Hz consciousness cycle - reduced frequency
		inputQueue:     make(chan *SensorInput, 100),
		stopChan:       make(chan struct{}),
		pendingInputs:  make([]*SensorInput, 0),
	}

	// Initialize consciousness state
	runtime.currentState = &ConsciousnessState{
		CurrentCycle: 0,
		Attention: &AttentionState{
			FocusStrength: 0.5,
			CurrentFocus:  "initialization",
			Intensity:     0.7,
			Context:       "startup",
		},
		Emotions:       make(map[string]float64),
		Goals:          make([]ConsciousGoal, 0),
		ActiveThoughts: make([]Thought, 0),
		Intentions:     make([]Intent, 0),
		LastUpdate:     time.Now(),
		EnergyLevel:    1.0,
		FocusLevel:     0.7,
		AlertnessLevel: 0.8,
		DecisionState:  "ready",
		ProcessingLoad: 0.2,
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
		CurrentContext:  make(map[string]interface{}),
		ActiveBeliefs:   make(map[string]float64),
		TemporalContext: make(map[string]interface{}),
	}

	// Link working memory to consciousness state
	runtime.currentState.WorkingMemory = runtime.workingMemory

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

	// Skip cycle if no inputs to process
	if len(inputs) == 0 {
		return
	}

	// 2. LoadWorkingMemory() - Load current context and memory
	context := cr.loadWorkingMemory(ctx)

	// 2.5. LoadUserContext() - Load comprehensive user context including conversations
	cr.LoadUserContext(ctx)

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
	// Ensure user context is loaded before processing
	if err := cr.LoadUserContext(ctx); err != nil {
		log.Printf("[Consciousness] Warning: Failed to load user context before query processing: %v", err)
	}

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

	// 2.5. LoadUserContext - Load all user context including past conversations
	if err := cr.LoadUserContext(ctx); err != nil {
		log.Printf("[Consciousness] Warning: Failed to load user context: %v", err)
	}

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

// GetCurrentState returns the current consciousness state (for AGI integration)
func (cr *ConsciousnessRuntime) GetCurrentState() *ConsciousnessState {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	// Return a copy of the current state
	if cr.currentState != nil {
		stateCopy := *cr.currentState
		return &stateCopy
	}

	return nil
}

// AddSensorInput adds a sensor input to the consciousness runtime
func (cr *ConsciousnessRuntime) AddSensorInput(input *SensorInput) error {
	if input == nil {
		return fmt.Errorf("sensor input cannot be nil")
	}

	select {
	case cr.inputQueue <- input:
		return nil
	default:
		return fmt.Errorf("input queue is full")
	}
}

// LoadUserContext loads comprehensive user context including past conversations
func (cr *ConsciousnessRuntime) LoadUserContext(ctx context.Context) error {
	if cr.ragSystem == nil || cr.contextManager == nil {
		return fmt.Errorf("RAG system or context manager not available")
	}

	nodeUserID := fmt.Sprintf("node_user_%s", cr.nodeID)

	// Load user profile with all conversation history
	if cr.ragSystem.UserProfileManager != nil {
		userProfile := cr.ragSystem.UserProfileManager.GetOrCreateUserProfile(nodeUserID)
		if userProfile != nil {
			// Store user context in working memory
			if cr.workingMemory != nil {
				cr.workingMemory.ActiveContext["user_profile"] = userProfile
				cr.workingMemory.ActiveContext["user_name"] = userProfile.Name
				cr.workingMemory.ActiveContext["user_conversations"] = userProfile.ConversationIDs

				log.Printf("[Consciousness] Loaded user context for %s with %d past conversations",
					userProfile.Name, len(userProfile.ConversationIDs))
			}

			// Load recent conversations from all user's past sessions
			if len(userProfile.ConversationIDs) > 0 {
				// Get the most recent conversations across all sessions
				allRecentEvents, err := cr.loadRecentEventsFromAllConversations(ctx, userProfile.ConversationIDs, 20)
				if err == nil && len(allRecentEvents) > 0 {
					cr.workingMemory.ActiveContext["all_user_history"] = allRecentEvents
					cr.workingMemory.TemporalContext["conversation_timeline"] = cr.buildConversationTimeline(allRecentEvents)
					log.Printf("[Consciousness] Loaded %d historical events from user's past conversations", len(allRecentEvents))

					// Update working memory with conversation context
					cr.updateWorkingMemoryWithConversationContext(allRecentEvents)
				}
			}
		}
	}

	// Load current session context
	if cr.contextManager != nil {
		// Get recent conversation history from current session
		recentHistory, err := cr.contextManager.GetRecentConversationHistory(ctx, 10)
		if err == nil && len(recentHistory) > 0 {
			cr.workingMemory.ActiveContext["current_session_history"] = recentHistory
			cr.workingMemory.TemporalContext["session_start"] = cr.contextManager.GetSessionStartTime()
			log.Printf("[Consciousness] Loaded %d events from current session", len(recentHistory))
		}

		// Store conversation ID for context tracking
		cr.workingMemory.ActiveContext["current_conversation_id"] = cr.contextManager.GetConversationID()
	}

	return nil
}

// loadRecentEventsFromAllConversations retrieves recent events from all user conversations
func (cr *ConsciousnessRuntime) loadRecentEventsFromAllConversations(ctx context.Context, conversationIDs []string, limit int) ([]ActivityEvent, error) {
	allEvents := make([]ActivityEvent, 0)

	if cr.ragSystem == nil || cr.ragSystem.VectorDB == nil {
		return allEvents, nil
	}

	// Search for events from all conversations
	for _, convID := range conversationIDs {
		searchQuery := fmt.Sprintf("conversation_id:%s user events messages", convID)

		// Generate embedding for search
		model, err := cr.ragSystem.ModelManager.GetModel(cr.ragSystem.QueryProcessor.ModelID)
		if err != nil {
			continue
		}

		embedding, err := model.GenerateEmbedding(ctx, searchQuery)
		if err != nil {
			continue
		}

		// Search vector DB for this conversation
		query := types.VectorQuery{
			Embedding:     embedding,
			MaxResults:    10,
			MinSimilarity: 0.4,
		}

		results, err := cr.ragSystem.VectorDB.SearchSimilar(query)
		if err != nil {
			continue
		}

		// Convert results to activity events
		for _, result := range results {
			// Parse the stored JSON from the document
			if strings.Contains(result.Text, "JSON:") {
				jsonStart := strings.LastIndex(result.Text, "JSON:")
				if jsonStart != -1 {
					jsonStr := result.Text[jsonStart+5:]
					var event ActivityEvent
					if err := json.Unmarshal([]byte(jsonStr), &event); err == nil {
						allEvents = append(allEvents, event)
					}
				}
			}
		}
	}

	// Sort by timestamp (most recent first) and limit results
	for i := 0; i < len(allEvents)-1; i++ {
		for j := i + 1; j < len(allEvents); j++ {
			if allEvents[i].Timestamp.Before(allEvents[j].Timestamp) {
				allEvents[i], allEvents[j] = allEvents[j], allEvents[i]
			}
		}
	}

	if len(allEvents) > limit {
		allEvents = allEvents[:limit]
	}

	return allEvents, nil
}

// buildConversationTimeline creates a timeline representation of conversations
func (cr *ConsciousnessRuntime) buildConversationTimeline(events []ActivityEvent) map[string]interface{} {
	timeline := make(map[string]interface{})

	if len(events) == 0 {
		return timeline
	}

	// Group events by conversation ID
	conversations := make(map[string][]ActivityEvent)
	for _, event := range events {
		conversations[event.ConversationID] = append(conversations[event.ConversationID], event)
	}

	// Create timeline summary
	timeline["total_conversations"] = len(conversations)
	timeline["total_events"] = len(events)
	timeline["earliest_event"] = events[len(events)-1].Timestamp
	timeline["latest_event"] = events[0].Timestamp

	// Track conversation summaries
	convSummaries := make([]map[string]interface{}, 0)
	for convID, convEvents := range conversations {
		summary := map[string]interface{}{
			"conversation_id": convID,
			"event_count":     len(convEvents),
			"start_time":      convEvents[len(convEvents)-1].Timestamp,
			"end_time":        convEvents[0].Timestamp,
		}
		convSummaries = append(convSummaries, summary)
	}
	timeline["conversations"] = convSummaries

	return timeline
}

// updateWorkingMemoryWithConversationContext updates working memory with conversation insights
func (cr *ConsciousnessRuntime) updateWorkingMemoryWithConversationContext(events []ActivityEvent) {
	if cr.workingMemory == nil || len(events) == 0 {
		return
	}

	// Extract conversation patterns and insights
	patterns := make(map[string]interface{})

	// Count types of interactions
	queryCount := 0
	responseCount := 0
	actionCount := 0

	for _, event := range events {
		switch event.Type {
		case "query":
			queryCount++
		case "response":
			responseCount++
		case "action":
			actionCount++
		}
	}

	patterns["total_queries"] = queryCount
	patterns["total_responses"] = responseCount
	patterns["total_actions"] = actionCount
	patterns["interaction_ratio"] = float64(responseCount) / float64(max(queryCount, 1))

	// Store patterns in working memory
	cr.workingMemory.ActiveContext["conversation_patterns"] = patterns

	// Update beliefs based on conversation history
	if cr.workingMemory.ActiveBeliefs == nil {
		cr.workingMemory.ActiveBeliefs = make(map[string]float64)
	}

	// Build confidence in user relationship
	if queryCount > 5 {
		cr.workingMemory.ActiveBeliefs["established_relationship"] = 0.8
	} else if queryCount > 1 {
		cr.workingMemory.ActiveBeliefs["developing_relationship"] = 0.6
	} else {
		cr.workingMemory.ActiveBeliefs["new_relationship"] = 0.9
	}

	// Update temporal context
	if len(events) > 0 {
		lastInteraction := events[0].Timestamp
		timeSinceLastInteraction := time.Since(lastInteraction)
		cr.workingMemory.TemporalContext["time_since_last_interaction"] = timeSinceLastInteraction
		cr.workingMemory.TemporalContext["last_interaction_time"] = lastInteraction
	}
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetContextualResponsePrompt builds a prompt that includes conversation awareness
func (cr *ConsciousnessRuntime) GetContextualResponsePrompt(ctx context.Context, currentQuery string) string {
	if cr.workingMemory == nil {
		return currentQuery
	}

	var promptBuilder strings.Builder

	// Add user context if available
	if userProfile, ok := cr.workingMemory.ActiveContext["user_profile"].(*UserProfile); ok {
		if userProfile.Name != "" {
			promptBuilder.WriteString(fmt.Sprintf("User: %s\n", userProfile.Name))
		}
		if userProfile.WorkContext != "" {
			promptBuilder.WriteString(fmt.Sprintf("Work Context: %s\n", userProfile.WorkContext))
		}
	}

	// Add conversation timeline awareness
	if timeline, ok := cr.workingMemory.TemporalContext["conversation_timeline"].(map[string]interface{}); ok {
		if totalConversations, ok := timeline["total_conversations"].(int); ok && totalConversations > 1 {
			promptBuilder.WriteString(fmt.Sprintf("Note: We have had %d previous conversations.\n", totalConversations))
		}
		if totalEvents, ok := timeline["total_events"].(int); ok && totalEvents > 0 {
			promptBuilder.WriteString(fmt.Sprintf("Our conversation history includes %d interactions.\n", totalEvents))
		}
	}

	// Add relationship context
	if cr.workingMemory.ActiveBeliefs != nil {
		if established, ok := cr.workingMemory.ActiveBeliefs["established_relationship"]; ok && established > 0.5 {
			promptBuilder.WriteString("Note: We have an established conversation history.\n")
		} else if developing, ok := cr.workingMemory.ActiveBeliefs["developing_relationship"]; ok && developing > 0.5 {
			promptBuilder.WriteString("Note: We are building our conversation relationship.\n")
		}
	}

	// Add current session context
	if sessionHistory, ok := cr.workingMemory.ActiveContext["current_session_history"].([]ActivityEvent); ok && len(sessionHistory) > 0 {
		promptBuilder.WriteString(fmt.Sprintf("Current session context (%d recent interactions):\n", len(sessionHistory)))
		for i, event := range sessionHistory {
			if i >= 5 { // Limit to 5 most recent
				break
			}
			if event.Type == "query" {
				promptBuilder.WriteString(fmt.Sprintf("- User asked: %s\n", event.Content))
			} else if event.Type == "response" && event.Success {
				promptBuilder.WriteString(fmt.Sprintf("- I responded: %s\n", event.Content))
			}
		}
	}

	// Add temporal awareness
	if lastInteraction, ok := cr.workingMemory.TemporalContext["last_interaction_time"].(time.Time); ok {
		timeSince := time.Since(lastInteraction)
		if timeSince > time.Hour {
			promptBuilder.WriteString(fmt.Sprintf("Note: It's been %v since our last interaction.\n", timeSince.Truncate(time.Minute)))
		}
	}

	// Add the current query
	promptBuilder.WriteString(fmt.Sprintf("\nCurrent query: %s\n", currentQuery))
	promptBuilder.WriteString("\nPlease respond with awareness of our conversation history and context above.")

	return promptBuilder.String()
}

// SetCodeReflectionAgent integrates a code reflection agent with the consciousness runtime
func (cr *ConsciousnessRuntime) SetCodeReflectionAgent(codeAgent interface{}) {
	// This will allow the consciousness runtime to receive code quality insights
	if cr.agiSystem != nil {
		// Add code quality awareness to the AGI system
		cr.agiSystem.ProcessInput(
			context.Background(),
			"system_integration",
			"Code Reflection Agent integrated for continuous codebase monitoring",
			map[string]interface{}{
				"agent_type":       "code_reflection",
				"capabilities":     []string{"static_analysis", "issue_detection", "quality_assessment"},
				"monitoring_scope": []string{"internal", "pkg", "cmd"},
			},
		)
	}

	log.Printf("[Consciousness] Code Reflection Agent integrated with consciousness runtime")
}
