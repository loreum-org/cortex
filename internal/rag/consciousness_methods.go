package rag

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/loreum-org/cortex/internal/ai"
)

// readSensors gathers inputs from the environment (Step 1)
func (agi *AGIPromptSystem) readSensors(ctx context.Context) []*SensorInput {
	inputs := make([]*SensorInput, 0)

	// Collect pending inputs from queue (non-blocking)
	timeout := time.NewTimer(time.Millisecond * 10)
	defer timeout.Stop()

	collectInputs := true
	for collectInputs {
		select {
		case input := <-agi.inputQueue:
			inputs = append(inputs, input)
			agi.pendingInputs = append(agi.pendingInputs, input)
		case <-timeout.C:
			collectInputs = false
		default:
			collectInputs = false
		}
	}

	// Update attention based on inputs
	if len(inputs) > 0 {
		agi.updateAttention(inputs)
	}

	return inputs
}

// loadWorkingMemory loads current context and memory (Step 2)
func (agi *AGIPromptSystem) loadWorkingMemory(ctx context.Context) map[string]interface{} {
	context := make(map[string]interface{})

	// Load AGI state
	if agi.currentState != nil {
		context["agi_state"] = agi.currentState
		context["intelligence_level"] = agi.currentState.IntelligenceLevel
		context["knowledge_domains"] = agi.currentState.KnowledgeDomains
	}

	// Load recent conversation context and user profile
	if agi.contextManager != nil {
		// Get actual conversation history from vector DB
		if conversationContext, err := agi.contextManager.GetConversationContext(ctx, 10); err == nil {
			context["conversation_history"] = conversationContext
			agi.workingMemory.ActiveContext["conversation_history"] = conversationContext
		} else {
			context["conversation_history"] = "This is the beginning of our conversation."
		}
		context["conversation_id"] = agi.contextManager.conversationID

		// Load recent conversation events for better context
		if history, err := agi.contextManager.GetRecentConversationHistory(ctx, 5); err == nil && len(history) > 0 {
			context["recent_events"] = history
			agi.workingMemory.ActiveContext["recent_events"] = history
		}
	}

	// Load user profile information for personalized responses
	if agi.ragSystem != nil && agi.ragSystem.UserProfileManager != nil {
		nodeUserID := fmt.Sprintf("node_user_%s", agi.nodeID)
		userProfile := agi.ragSystem.UserProfileManager.GetOrCreateUserProfile(nodeUserID)
		if userProfile != nil {
			context["user_profile"] = userProfile
			context["user_name"] = userProfile.Name
			context["user_preferred_name"] = userProfile.PreferredName
			context["user_work_context"] = userProfile.WorkContext
			agi.workingMemory.ActiveContext["user_profile"] = userProfile

			if userProfile.Name != "" {
				log.Printf("[AGI-Consciousness] Loaded user profile: %s", userProfile.Name)
			}
		}
	}

	// Load working memory state
	context["consciousness_state"] = agi.currentState
	context["working_memory"] = agi.workingMemory
	context["recent_inputs"] = agi.workingMemory.RecentInputs
	context["active_goals"] = agi.workingMemory.Goals
	context["beliefs"] = agi.workingMemory.Beliefs

	// Add system context
	context["node_id"] = agi.nodeID
	context["cycle_count"] = agi.currentState.CurrentCycle
	context["timestamp"] = time.Now()

	agi.workingMemory.LastUpdated = time.Now()

	return context
}

// evaluateIntent analyzes inputs to understand intentions (Step 3)
func (agi *AGIPromptSystem) evaluateIntent(ctx context.Context, inputs []*SensorInput, context map[string]interface{}) []Intent {
	intents := make([]Intent, 0)

	for _, input := range inputs {
		intent := agi.intentAnalyzer.AnalyzeIntent(ctx, input, context)
		if intent != nil {
			intents = append(intents, *intent)
		}
	}

	// Update consciousness state with new intentions
	agi.currentState.Intentions = intents

	return intents
}

// makeDecision decides on actions based on intentions (Step 4)
func (agi *AGIPromptSystem) makeDecision(ctx context.Context, intents []Intent, context map[string]interface{}) []Decision {
	decisions := make([]Decision, 0)

	for _, intent := range intents {
		decision := agi.decisionEngine.MakeDecision(ctx, &intent, context)
		if decision != nil {
			decisions = append(decisions, *decision)
		}
	}

	return decisions
}

// executeAction executes the decided actions (Step 5)
func (agi *AGIPromptSystem) executeAction(ctx context.Context, decisions []Decision) []ActionResult {
	results := make([]ActionResult, 0)

	for _, decision := range decisions {
		result := agi.actionExecutor.ExecuteAction(ctx, &decision)
		results = append(results, result)
	}

	return results
}

// updateMemory learns from the experience (Step 6)
func (agi *AGIPromptSystem) updateMemory(ctx context.Context, inputs []*SensorInput, decisions []Decision, results []ActionResult) {
	// Update working memory with recent inputs
	for _, input := range inputs {
		agi.workingMemory.RecentInputs = append(agi.workingMemory.RecentInputs, input)
		if len(agi.workingMemory.RecentInputs) > 10 {
			agi.workingMemory.RecentInputs = agi.workingMemory.RecentInputs[1:]
		}
	}

	// Create memory fragments from the experience
	for i, decision := range decisions {
		if i < len(results) {
			result := results[i]

			// Create memory fragment
			fragment := MemoryFragment{
				ID:           fmt.Sprintf("memory_%d_%s", agi.currentState.CurrentCycle, decision.ID),
				Content:      fmt.Sprintf("Decision: %s, Result: %v", decision.ChosenAction, result.Success),
				Type:         "experience",
				Importance:   decision.Confidence*0.5 + (map[bool]float64{true: 0.5, false: 0.8}[result.Success]),
				Associations: []string{decision.ChosenAction, decision.Intent.Type},
				Timestamp:    time.Now(),
				ExpiresAt:    time.Now().Add(time.Hour * 24), // Keep for 24 hours
				Metadata:     map[string]interface{}{"decision_id": decision.ID, "success": result.Success},
			}

			agi.workingMemory.ShortTermMemory = append(agi.workingMemory.ShortTermMemory, fragment)
		}
	}

	// Store interactions in Context Manager for persistent memory
	if agi.contextManager != nil {
		for i, input := range inputs {
			// Track the query in Context Manager
			queryEventID := agi.contextManager.TrackQuery(ctx, input.Content, input.Type, input.Metadata)

			// Track the response if available
			if i < len(results) && results[i].Success {
				response := fmt.Sprintf("%v", results[i].Result)
				agi.contextManager.TrackResponse(ctx, queryEventID, response, results[i].Duration, true, "")
			}
		}
	}

	// Extract and store user information in user profiles
	if agi.ragSystem != nil && agi.ragSystem.UserProfileManager != nil {
		nodeUserID := fmt.Sprintf("node_user_%s", agi.nodeID)
		for _, input := range inputs {
			agi.ragSystem.UserProfileManager.ExtractUserInfoFromText(nodeUserID, input.Content)
		}
	}

	// Clean up expired memory fragments
	now := time.Now()
	filtered := make([]MemoryFragment, 0)
	for _, fragment := range agi.workingMemory.ShortTermMemory {
		if now.Before(fragment.ExpiresAt) {
			filtered = append(filtered, fragment)
		}
	}
	agi.workingMemory.ShortTermMemory = filtered

	// Update AGI system with learning - this integrates with the existing learning
	go func() {
		for _, result := range results {
			if result.Success {
				// Feed successful experiences to AGI learning
				agi.ProcessInput(ctx, "success_experience",
					fmt.Sprintf("Successfully executed %s", result.Action),
					map[string]interface{}{
						"action":   result.Action,
						"duration": result.Duration.String(),
						"result":   result.Result,
					})
			}
		}
	}()

	// Update consciousness state
	agi.updateConsciousnessState(inputs, decisions, results)
}

// updateAttention updates the attention state based on inputs
func (agi *AGIPromptSystem) updateAttention(inputs []*SensorInput) {
	if len(inputs) == 0 {
		return
	}

	// Find highest priority input
	highestPriority := 0.0
	var primaryInput *SensorInput
	for _, input := range inputs {
		if input.Priority > highestPriority {
			highestPriority = input.Priority
			primaryInput = input
		}
	}

	if primaryInput != nil {
		agi.currentState.Attention.PrimaryFocus = fmt.Sprintf("%s: %s", primaryInput.Type, primaryInput.Content[:min(50, len(primaryInput.Content))])
		agi.currentState.Attention.FocusStrength = highestPriority
		agi.currentState.Attention.FocusDuration = time.Since(agi.lastCycleTime)
	}

	// Update secondary foci
	secondaryFoci := make([]string, 0)
	for _, input := range inputs {
		if input != primaryInput && input.Priority > 0.3 {
			secondaryFoci = append(secondaryFoci, input.Type)
		}
	}
	agi.currentState.Attention.SecondaryFoci = secondaryFoci
}

// updateConsciousnessState updates the overall consciousness state
func (agi *AGIPromptSystem) updateConsciousnessState(inputs []*SensorInput, decisions []Decision, results []ActionResult) {
	// Update energy level based on activity
	activityLevel := float64(len(inputs)+len(decisions)) / 10.0
	if activityLevel > 1.0 {
		activityLevel = 1.0
	}

	// Energy decreases with activity, recovers slowly
	agi.currentState.EnergyLevel = agi.currentState.EnergyLevel - (activityLevel * 0.01) + 0.005
	if agi.currentState.EnergyLevel > 1.0 {
		agi.currentState.EnergyLevel = 1.0
	}
	if agi.currentState.EnergyLevel < 0.1 {
		agi.currentState.EnergyLevel = 0.1
	}

	// Update focus level based on success rate
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	if len(results) > 0 {
		successRate := float64(successCount) / float64(len(results))
		agi.currentState.FocusLevel = (agi.currentState.FocusLevel * 0.9) + (successRate * 0.1)
	}

	// Update alertness based on input frequency
	inputFrequency := float64(len(inputs)) / 5.0 // Normalize to 0-1 scale
	if inputFrequency > 1.0 {
		inputFrequency = 1.0
	}
	agi.currentState.AlertnessLevel = (agi.currentState.AlertnessLevel * 0.8) + (inputFrequency * 0.2)

	// Update emotions based on results
	if agi.currentState.Emotions == nil {
		agi.currentState.Emotions = make(map[string]float64)
	}

	if len(results) > 0 {
		if successCount > len(results)/2 {
			agi.currentState.Emotions["satisfaction"] = agi.currentState.Emotions["satisfaction"]*0.9 + 0.1
			agi.currentState.Emotions["confidence"] = agi.currentState.Emotions["confidence"]*0.9 + 0.1
		} else {
			agi.currentState.Emotions["frustration"] = agi.currentState.Emotions["frustration"]*0.9 + 0.1
			agi.currentState.Emotions["uncertainty"] = agi.currentState.Emotions["uncertainty"]*0.9 + 0.05
		}
	}

	// Decay all emotions slowly toward neutral
	for emotion, value := range agi.currentState.Emotions {
		agi.currentState.Emotions[emotion] = value * 0.99
	}
}

// AnalyzeIntent analyzes inputs to determine user intentions
func (ia *IntentAnalyzer) AnalyzeIntent(ctx context.Context, input *SensorInput, context map[string]interface{}) *Intent {
	// Analyze the input content to determine intent
	intentType := "unknown"
	confidence := 0.5
	description := ""
	requiredActions := make([]string, 0)
	expectedOutcome := ""
	priority := input.Priority

	content := strings.ToLower(input.Content)

	// Enhanced rule-based intent classification
	if input.Type == "query" {
		// Direct question patterns
		if strings.Contains(content, "?") || strings.Contains(content, "what") ||
			strings.Contains(content, "how") || strings.Contains(content, "why") ||
			strings.Contains(content, "when") || strings.Contains(content, "where") ||
			strings.Contains(content, "who") || strings.Contains(content, "which") {
			intentType = "question"
			description = "User is asking a direct question"
			requiredActions = []string{"analyze_question", "search_knowledge", "generate_response"}
			expectedOutcome = "Provide informative answer"
			confidence = 0.9
		} else if strings.Contains(content, "tell me") || strings.Contains(content, "explain") ||
			strings.Contains(content, "describe") || strings.Contains(content, "about") {
			intentType = "question"
			description = "User is requesting information"
			requiredActions = []string{"analyze_question", "search_knowledge", "generate_response"}
			expectedOutcome = "Provide informative answer"
			confidence = 0.85
		} else if strings.Contains(content, "please") || strings.Contains(content, "can you") ||
			strings.Contains(content, "help") || strings.Contains(content, "show") {
			intentType = "request"
			description = "User is making a request for assistance"
			requiredActions = []string{"understand_request", "plan_assistance", "execute_help"}
			expectedOutcome = "Provide requested assistance"
			confidence = 0.75
		} else if strings.Contains(content, "do ") || strings.Contains(content, "create") ||
			strings.Contains(content, "make") || strings.Contains(content, "generate") {
			intentType = "command"
			description = "User is giving a command or instruction"
			requiredActions = []string{"parse_command", "validate_command", "execute_command"}
			expectedOutcome = "Execute the requested command"
			confidence = 0.7
		} else if len(content) > 5 && !strings.Contains(content, "hello") &&
			!strings.Contains(content, "hi") && !strings.Contains(content, "thanks") {
			// Default substantive content to question if it's not a greeting
			intentType = "question"
			description = "User is seeking information or discussion"
			requiredActions = []string{"analyze_context", "generate_response"}
			expectedOutcome = "Provide relevant response"
			confidence = 0.7
		} else {
			intentType = "conversation"
			description = "User is engaging in casual conversation"
			requiredActions = []string{"understand_context", "generate_response"}
			expectedOutcome = "Continue natural conversation"
			confidence = 0.6
		}
	}

	return &Intent{
		ID:              fmt.Sprintf("intent_%d_%s", time.Now().UnixNano(), input.ID),
		Type:            intentType,
		Description:     description,
		Confidence:      confidence,
		RequiredActions: requiredActions,
		ExpectedOutcome: expectedOutcome,
		Priority:        priority,
		Metadata: map[string]interface{}{
			"input_id":      input.ID,
			"input_type":    input.Type,
			"query_text":    input.Content,
			"input_content": input.Content,
		},
		Timestamp: time.Now(),
	}
}

// MakeDecision makes decisions based on intentions
func (de *DecisionEngine) MakeDecision(ctx context.Context, intent *Intent, context map[string]interface{}) *Decision {
	// Choose the best action based on the intent and context
	chosenAction := "default_response"
	actionParams := make(map[string]interface{})
	reasoning := ""
	confidence := intent.Confidence
	expectedOutcome := intent.ExpectedOutcome
	alternatives := make([]string, 0)
	riskAssessment := 0.1 // Low risk by default

	// Decision logic based on intent type
	switch intent.Type {
	case "question":
		chosenAction = "answer_question"
		reasoning = "User asked a question, providing informative answer"
		actionParams["query_text"] = intent.Metadata["query_text"]
		actionParams["search_type"] = "knowledge_search"
		// Include conversation history from context
		if conversationHistory, exists := context["conversation_history"]; exists {
			actionParams["conversation_history"] = conversationHistory
		}
		alternatives = []string{"clarify_question", "request_more_info"}

	case "request":
		chosenAction = "answer_question" // Use AI model for requests too
		reasoning = "User made a request, providing helpful response"
		actionParams["query_text"] = intent.Metadata["query_text"]
		actionParams["request_type"] = "assistance"
		// Include conversation history from context
		if conversationHistory, exists := context["conversation_history"]; exists {
			actionParams["conversation_history"] = conversationHistory
		}
		alternatives = []string{"clarify_request", "suggest_alternative"}
		riskAssessment = 0.2

	case "command":
		chosenAction = "answer_question" // Use AI model for commands too
		reasoning = "User gave a command, responding appropriately"
		actionParams["query_text"] = intent.Metadata["query_text"]
		actionParams["command_type"] = "user_instruction"
		actionParams["safety_check"] = true
		// Include conversation history from context
		if conversationHistory, exists := context["conversation_history"]; exists {
			actionParams["conversation_history"] = conversationHistory
		}
		alternatives = []string{"clarify_command", "suggest_safer_alternative"}
		riskAssessment = 0.4 // Higher risk for commands

	case "conversation":
		chosenAction = "engage_conversation"
		reasoning = "User is conversing, maintaining natural dialogue"
		actionParams["conversation_style"] = "friendly_helpful"
		alternatives = []string{"ask_clarifying_question", "change_topic"}

	default:
		chosenAction = "acknowledge_input"
		reasoning = "Intent unclear, acknowledging input and asking for clarification"
		actionParams["response_type"] = "clarification_request"
		alternatives = []string{"ignore_input", "provide_general_help"}
		confidence = 0.3
	}

	return &Decision{
		ID:              fmt.Sprintf("decision_%d_%s", time.Now().UnixNano(), intent.ID),
		Intent:          intent,
		ChosenAction:    chosenAction,
		ActionParams:    actionParams,
		Reasoning:       reasoning,
		Confidence:      confidence,
		ExpectedOutcome: expectedOutcome,
		Alternatives:    alternatives,
		RiskAssessment:  riskAssessment,
		Timestamp:       time.Now(),
	}
}

// ExecuteAction executes decisions and returns results
func (ae *ActionExecutor) ExecuteAction(ctx context.Context, decision *Decision) ActionResult {
	startTime := time.Now()

	result := ActionResult{
		DecisionID: decision.ID,
		Action:     decision.ChosenAction,
		Success:    false,
		Result:     nil,
		Duration:   0,
		Timestamp:  startTime,
	}

	// Execute based on chosen action
	switch decision.ChosenAction {
	case "answer_question":
		result = ae.executeAnswerQuestion(ctx, decision)

	case "fulfill_request":
		result = ae.executeFulfillRequest(ctx, decision)

	case "execute_command":
		result = ae.executeCommand(ctx, decision)

	case "engage_conversation":
		result = ae.executeEngageConversation(ctx, decision)

	case "acknowledge_input":
		result = ae.executeAcknowledgeInput(ctx, decision)

	default:
		result.ErrorMsg = fmt.Sprintf("Unknown action: %s", decision.ChosenAction)
		result.Success = false
		result.Result = "I'm not sure how to handle that request."
	}

	result.Duration = time.Since(startTime)
	return result
}

func (ae *ActionExecutor) executeAnswerQuestion(ctx context.Context, decision *Decision) ActionResult {
	// Get the original query from the decision action params
	var queryText string
	if decision.ActionParams != nil {
		if query, exists := decision.ActionParams["query_text"]; exists {
			queryText = fmt.Sprintf("%v", query)
		}
	}

	// Fallback to intent metadata
	if queryText == "" && decision.Intent != nil && len(decision.Intent.Metadata) > 0 {
		if query, exists := decision.Intent.Metadata["query_text"]; exists {
			queryText = fmt.Sprintf("%v", query)
		}
	}

	// If we still couldn't extract the query, use a generic approach
	if queryText == "" {
		queryText = "Please provide a helpful and informative response."
	}

	// Use AGI prompt system for response generation
	log.Printf("[ActionExecutor] Processing query with AGI: %s", queryText)
	
	// Get AGI-enhanced prompt
	agiPrompt := ae.agiSystem.GetAGIPromptForQuery(ctx, queryText)
	
	// Try to use AI model if available
	if ae.modelManager != nil {
		if mm, ok := ae.modelManager.(*ai.ModelManager); ok && ae.defaultModel != "" {
			if model, err := mm.GetModel(ae.defaultModel); err == nil {
				// Create generation options for comprehensive responses
				options := ai.GenerateOptions{
					MaxTokens:   1000,
					Temperature: 0.7,
					TopP:        1.0,
				}

				// Generate response using AGI prompt
				if response, err := model.GenerateResponse(ctx, agiPrompt, options); err == nil {
					log.Printf("[ActionExecutor] Generated AGI response: %s", response[:min(100, len(response))])
					return ActionResult{
						DecisionID: decision.ID,
						Action:     decision.ChosenAction,
						Success:    true,
						Result:     response,
						Timestamp:  time.Now(),
					}
				}
			}
		}
	}

	// Fallback to AGI-based response
	var response string
	if ae.agiSystem != nil {
		agiState := ae.agiSystem.GetCurrentAGIState()
		intelligenceLevel := agiState.IntelligenceLevel

		if intelligenceLevel > 80 {
			response = "Based on my understanding and knowledge, I can provide you with a comprehensive answer. Let me analyze your question and draw from my accumulated knowledge to give you the most helpful response."
		} else if intelligenceLevel > 60 {
			response = "I'll do my best to answer your question. Let me think about this and provide you with what I know."
		} else {
			response = "I'll try to help you with that question. I'm still learning, so please let me know if you need clarification."
		}
	} else {
		response = "I understand you have a question. While I'm processing how to best answer it, could you provide more details about what you'd like to know?"
	}

	return ActionResult{
		DecisionID: decision.ID,
		Action:     decision.ChosenAction,
		Success:    true,
		Result:     response,
		Timestamp:  time.Now(),
	}
}

func (ae *ActionExecutor) executeFulfillRequest(ctx context.Context, decision *Decision) ActionResult {
	response := "I understand your request and I'm working on fulfilling it. I'm designed to be helpful and will do my best to assist you."

	return ActionResult{
		DecisionID: decision.ID,
		Action:     decision.ChosenAction,
		Success:    true,
		Result:     response,
		Timestamp:  time.Now(),
	}
}

func (ae *ActionExecutor) executeCommand(ctx context.Context, decision *Decision) ActionResult {
	// For safety, we'll acknowledge commands but execute them carefully
	response := "I acknowledge your instruction. I'm analyzing the best way to carry it out safely and effectively."

	return ActionResult{
		DecisionID: decision.ID,
		Action:     decision.ChosenAction,
		Success:    true,
		Result:     response,
		Timestamp:  time.Now(),
	}
}

func (ae *ActionExecutor) executeEngageConversation(ctx context.Context, decision *Decision) ActionResult {
	var response string

	if ae.agiSystem != nil {
		agiState := ae.agiSystem.GetCurrentAGIState()

		// Vary response based on personality and intelligence
		if agiState.IntelligenceLevel > 70 {
			response = "I enjoy our conversation! I'm always interested in learning from our interactions and sharing insights. What's on your mind?"
		} else {
			response = "I'm here to chat and help however I can. What would you like to talk about?"
		}
	} else {
		response = "Thank you for engaging with me. I'm here to help and have a meaningful conversation with you."
	}

	return ActionResult{
		DecisionID: decision.ID,
		Action:     decision.ChosenAction,
		Success:    true,
		Result:     response,
		Timestamp:  time.Now(),
	}
}

func (ae *ActionExecutor) executeAcknowledgeInput(ctx context.Context, decision *Decision) ActionResult {
	response := "I see that you've shared something with me. Could you help me understand what you'd like me to do? I'm here to assist you."

	return ActionResult{
		DecisionID: decision.ID,
		Action:     decision.ChosenAction,
		Success:    true,
		Result:     response,
		Timestamp:  time.Now(),
	}
}

// SetModelManager sets the model manager for AI generation
func (ae *ActionExecutor) SetModelManager(modelManager interface{}, defaultModel string) {
	ae.modelManager = modelManager
	ae.defaultModel = defaultModel
}