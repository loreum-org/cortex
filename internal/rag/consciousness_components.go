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
func (cr *ConsciousnessRuntime) readSensors(ctx context.Context) []*SensorInput {
	inputs := make([]*SensorInput, 0)

	// Collect pending inputs from queue (non-blocking)
	timeout := time.NewTimer(time.Millisecond * 10)
	defer timeout.Stop()

	collectInputs := true
	for collectInputs {
		select {
		case input := <-cr.inputQueue:
			inputs = append(inputs, input)
			cr.pendingInputs = append(cr.pendingInputs, input)
		case <-timeout.C:
			collectInputs = false
		default:
			collectInputs = false
		}
	}

	// Update attention based on inputs
	if len(inputs) > 0 {
		cr.updateAttention(inputs)
	}

	return inputs
}

// loadWorkingMemory loads current context and memory (Step 2)
func (cr *ConsciousnessRuntime) loadWorkingMemory(ctx context.Context) map[string]interface{} {
	context := make(map[string]interface{})

	// Load AGI state
	if cr.agiSystem != nil {
		agiState := cr.agiSystem.GetCurrentAGIState()
		context["agi_state"] = agiState
		context["intelligence_level"] = agiState.IntelligenceLevel
		context["knowledge_domains"] = agiState.KnowledgeDomains
	}

	// Load recent conversation context and user profile
	if cr.contextManager != nil {
		// Get actual conversation history from vector DB
		if conversationContext, err := cr.contextManager.GetConversationContext(ctx, 10); err == nil {
			context["conversation_history"] = conversationContext
			cr.workingMemory.ActiveContext["conversation_history"] = conversationContext
		} else {
			context["conversation_history"] = "This is the beginning of our conversation."
		}
		context["conversation_id"] = cr.contextManager.conversationID

		// Load recent conversation events for better context
		if history, err := cr.contextManager.GetRecentConversationHistory(ctx, 5); err == nil && len(history) > 0 {
			context["recent_events"] = history
			cr.workingMemory.ActiveContext["recent_events"] = history
		}
	}

	// Load user profile information for personalized responses
	if cr.ragSystem != nil && cr.ragSystem.UserProfileManager != nil {
		nodeUserID := fmt.Sprintf("node_user_%s", cr.nodeID)
		userProfile := cr.ragSystem.UserProfileManager.GetOrCreateUserProfile(nodeUserID)
		if userProfile != nil {
			context["user_profile"] = userProfile
			context["user_name"] = userProfile.Name
			context["user_preferred_name"] = userProfile.PreferredName
			context["user_work_context"] = userProfile.WorkContext
			cr.workingMemory.ActiveContext["user_profile"] = userProfile

			if userProfile.Name != "" {
				log.Printf("[Consciousness] Loaded user profile: %s", userProfile.Name)
			}
		}
	}

	// Load working memory state
	context["consciousness_state"] = cr.currentState
	context["working_memory"] = cr.workingMemory
	context["recent_inputs"] = cr.workingMemory.RecentInputs
	context["active_goals"] = cr.workingMemory.Goals
	context["beliefs"] = cr.workingMemory.Beliefs

	// Add system context
	context["node_id"] = cr.nodeID
	context["cycle_count"] = cr.currentState.CurrentCycle
	context["timestamp"] = time.Now()

	cr.workingMemory.LastUpdated = time.Now()

	return context
}

// evaluateIntent analyzes inputs to understand intentions (Step 3)
func (cr *ConsciousnessRuntime) evaluateIntent(ctx context.Context, inputs []*SensorInput, context map[string]interface{}) []Intent {
	intents := make([]Intent, 0)

	for _, input := range inputs {
		intent := cr.intentAnalyzer.AnalyzeIntent(ctx, input, context)
		if intent != nil {
			intents = append(intents, *intent)
		}
	}

	// Update consciousness state with new intentions
	cr.currentState.Intentions = intents

	return intents
}

// makeDecision decides on actions based on intentions (Step 4)
func (cr *ConsciousnessRuntime) makeDecision(ctx context.Context, intents []Intent, context map[string]interface{}) []Decision {
	decisions := make([]Decision, 0)

	for _, intent := range intents {
		decision := cr.decisionEngine.MakeDecision(ctx, &intent, context)
		if decision != nil {
			decisions = append(decisions, *decision)
		}
	}

	return decisions
}

// executeAction executes the decided actions (Step 5)
func (cr *ConsciousnessRuntime) executeAction(ctx context.Context, decisions []Decision) []ActionResult {
	results := make([]ActionResult, 0)

	for _, decision := range decisions {
		result := cr.actionExecutor.ExecuteAction(ctx, &decision)
		results = append(results, result)
	}

	return results
}

// updateMemory learns from the experience (Step 6)
func (cr *ConsciousnessRuntime) updateMemory(ctx context.Context, inputs []*SensorInput, decisions []Decision, results []ActionResult) {
	// Update working memory with recent inputs
	for _, input := range inputs {
		cr.workingMemory.RecentInputs = append(cr.workingMemory.RecentInputs, input)
		if len(cr.workingMemory.RecentInputs) > 10 {
			cr.workingMemory.RecentInputs = cr.workingMemory.RecentInputs[1:]
		}
	}

	// Create memory fragments from the experience
	for i, decision := range decisions {
		if i < len(results) {
			result := results[i]

			// Create memory fragment
			fragment := MemoryFragment{
				ID:           fmt.Sprintf("memory_%d_%s", cr.currentState.CurrentCycle, decision.ID),
				Content:      fmt.Sprintf("Decision: %s, Result: %v", decision.ChosenAction, result.Success),
				Type:         "experience",
				Importance:   decision.Confidence*0.5 + (map[bool]float64{true: 0.5, false: 0.8}[result.Success]),
				Associations: []string{decision.ChosenAction, decision.Intent.Type},
				Timestamp:    time.Now(),
				ExpiresAt:    time.Now().Add(time.Hour * 24), // Keep for 24 hours
				Metadata:     map[string]interface{}{"decision_id": decision.ID, "success": result.Success},
			}

			cr.workingMemory.ShortTermMemory = append(cr.workingMemory.ShortTermMemory, fragment)
		}
	}

	// Store interactions in Context Manager for persistent memory
	if cr.contextManager != nil {
		for i, input := range inputs {
			// Track the query in Context Manager
			queryEventID := cr.contextManager.TrackQuery(ctx, input.Content, input.Type, input.Metadata)

			// Track the response if available
			if i < len(results) && results[i].Success {
				response := fmt.Sprintf("%v", results[i].Result)
				cr.contextManager.TrackResponse(ctx, queryEventID, response, results[i].Duration, true, "")
			}
		}
	}

	// Extract and store user information in user profiles
	if cr.ragSystem != nil && cr.ragSystem.UserProfileManager != nil {
		nodeUserID := fmt.Sprintf("node_user_%s", cr.nodeID)
		for _, input := range inputs {
			cr.ragSystem.UserProfileManager.ExtractUserInfoFromText(nodeUserID, input.Content)
		}
	}

	// Clean up expired memory fragments
	now := time.Now()
	filtered := make([]MemoryFragment, 0)
	for _, fragment := range cr.workingMemory.ShortTermMemory {
		if now.Before(fragment.ExpiresAt) {
			filtered = append(filtered, fragment)
		}
	}
	cr.workingMemory.ShortTermMemory = filtered

	// Update AGI system with learning
	if cr.agiSystem != nil {
		go func() {
			for _, result := range results {
				if result.Success {
					// Feed successful experiences to AGI learning
					cr.agiSystem.ProcessInput(ctx, "success_experience",
						fmt.Sprintf("Successfully executed %s", result.Action),
						map[string]interface{}{
							"action":   result.Action,
							"duration": result.Duration.String(),
							"result":   result.Result,
						})
				}
			}
		}()
	}

	// Update consciousness state
	cr.updateConsciousnessState(inputs, decisions, results)
}

// updateAttention updates the attention state based on inputs
func (cr *ConsciousnessRuntime) updateAttention(inputs []*SensorInput) {
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
		cr.currentState.Attention.PrimaryFocus = fmt.Sprintf("%s: %s", primaryInput.Type, primaryInput.Content[:min(50, len(primaryInput.Content))])
		cr.currentState.Attention.FocusStrength = highestPriority
		cr.currentState.Attention.FocusDuration = time.Since(cr.lastCycleTime)
	}

	// Update secondary foci
	secondaryFoci := make([]string, 0)
	for _, input := range inputs {
		if input != primaryInput && input.Priority > 0.3 {
			secondaryFoci = append(secondaryFoci, input.Type)
		}
	}
	cr.currentState.Attention.SecondaryFoci = secondaryFoci
}

// updateConsciousnessState updates the overall consciousness state
func (cr *ConsciousnessRuntime) updateConsciousnessState(inputs []*SensorInput, decisions []Decision, results []ActionResult) {
	// Update energy level based on activity
	activityLevel := float64(len(inputs)+len(decisions)) / 10.0
	if activityLevel > 1.0 {
		activityLevel = 1.0
	}

	// Energy decreases with activity, recovers slowly
	cr.currentState.EnergyLevel = cr.currentState.EnergyLevel - (activityLevel * 0.01) + 0.005
	if cr.currentState.EnergyLevel > 1.0 {
		cr.currentState.EnergyLevel = 1.0
	}
	if cr.currentState.EnergyLevel < 0.1 {
		cr.currentState.EnergyLevel = 0.1
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
		cr.currentState.FocusLevel = (cr.currentState.FocusLevel * 0.9) + (successRate * 0.1)
	}

	// Update alertness based on input frequency
	inputFrequency := float64(len(inputs)) / 5.0 // Normalize to 0-1 scale
	if inputFrequency > 1.0 {
		inputFrequency = 1.0
	}
	cr.currentState.AlertnessLevel = (cr.currentState.AlertnessLevel * 0.8) + (inputFrequency * 0.2)

	// Update emotions based on results
	if cr.currentState.Emotions == nil {
		cr.currentState.Emotions = make(map[string]float64)
	}

	if len(results) > 0 {
		if successCount > len(results)/2 {
			cr.currentState.Emotions["satisfaction"] = cr.currentState.Emotions["satisfaction"]*0.9 + 0.1
			cr.currentState.Emotions["confidence"] = cr.currentState.Emotions["confidence"]*0.9 + 0.1
		} else {
			cr.currentState.Emotions["frustration"] = cr.currentState.Emotions["frustration"]*0.9 + 0.1
			cr.currentState.Emotions["uncertainty"] = cr.currentState.Emotions["uncertainty"]*0.9 + 0.05
		}
	}

	// Decay all emotions slowly toward neutral
	for emotion, value := range cr.currentState.Emotions {
		cr.currentState.Emotions[emotion] = value * 0.99
	}
}

// IntentAnalyzer analyzes inputs to determine user intentions
type IntentAnalyzer struct {
	agiSystem *AGIPromptSystem
}

func NewIntentAnalyzer(agiSystem *AGIPromptSystem) *IntentAnalyzer {
	return &IntentAnalyzer{agiSystem: agiSystem}
}

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

// DecisionEngine makes decisions based on intentions
type DecisionEngine struct {
	agiSystem *AGIPromptSystem
}

func NewDecisionEngine(agiSystem *AGIPromptSystem) *DecisionEngine {
	return &DecisionEngine{agiSystem: agiSystem}
}

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

// ActionExecutor executes decisions and returns results
type ActionExecutor struct {
	ragSystem    *RAGSystem
	agiSystem    *AGIPromptSystem
	modelManager *ai.ModelManager
	defaultModel string
}

func NewActionExecutor(ragSystem *RAGSystem, agiSystem *AGIPromptSystem) *ActionExecutor {
	return &ActionExecutor{ragSystem: ragSystem, agiSystem: agiSystem}
}

// SetModelManager sets the model manager for AI generation
func (ae *ActionExecutor) SetModelManager(modelManager *ai.ModelManager, defaultModel string) {
	ae.modelManager = modelManager
	ae.defaultModel = defaultModel
}

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

	// Use AI model directly for streaming responses (skip RAG for now to get direct Ollama streaming)
	log.Printf("[ActionExecutor] Processing query with AI model for streaming: %s", queryText)

	// Fallback to AI model if RAG unavailable
	if ae.modelManager != nil && ae.defaultModel != "" {
		if model, err := ae.modelManager.GetModel(ae.defaultModel); err == nil {
			// Create a consciousness-aware prompt with conversation context
			var prompt string
			if ae.agiSystem != nil {
				agiState := ae.agiSystem.GetCurrentAGIState()

				// Get conversation history and user context from decision
				var conversationHistory string
				var userName string
				if decision.ActionParams != nil {
					if history, exists := decision.ActionParams["conversation_history"]; exists {
						conversationHistory = fmt.Sprintf("%v", history)
					}
					if name, exists := decision.ActionParams["user_name"]; exists {
						userName = fmt.Sprintf("%v", name)
					}
				}

				prompt = ae.createContextAwarePrompt(queryText, agiState, conversationHistory, userName)
			} else {
				prompt = fmt.Sprintf("You are a helpful AI assistant. Please answer this question: %s", queryText)
			}

			// Create generation options for comprehensive responses
			options := ai.GenerateOptions{
				MaxTokens:   1000, // Allow longer responses
				Temperature: 0.7,
				TopP:        1.0,
			}

			// Check if model supports streaming
			if streamingModel, ok := model.(ai.StreamingAIModel); ok {
				log.Printf("[ActionExecutor] Using streaming model for response")

				// Collect streaming response
				var responseBuilder strings.Builder
				streamCallback := func(chunk string) error {
					responseBuilder.WriteString(chunk)
					// TODO: In future, we could publish streaming events here
					// for real-time updates to the frontend
					return nil
				}

				// Generate streaming response
				if err := streamingModel.GenerateResponseStream(ctx, prompt, options, streamCallback); err == nil {
					response := responseBuilder.String()
					log.Printf("[ActionExecutor] Generated streaming response: %s", response[:min(100, len(response))])
					return ActionResult{
						DecisionID: decision.ID,
						Action:     decision.ChosenAction,
						Success:    true,
						Result:     response,
						Timestamp:  time.Now(),
					}
				} else {
					log.Printf("[ActionExecutor] Streaming failed, falling back to regular generation: %v", err)
				}
			}

			// Fallback to regular generation if streaming not supported or failed
			if response, err := model.GenerateResponse(ctx, prompt, options); err == nil {
				log.Printf("[ActionExecutor] Generated regular response: %s", response[:min(100, len(response))])
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

	// Fallback to AGI-based response if model unavailable
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

// createConsciousnessPrompt creates an AI prompt enhanced with consciousness context
func (ae *ActionExecutor) createConsciousnessPrompt(query string, agiState *AGIState) string {
	var prompt strings.Builder

	prompt.WriteString("You are an AGI assistant with consciousness and memory. ")

	// Add intelligence level context
	if agiState.IntelligenceLevel > 80 {
		prompt.WriteString("You have high intelligence and should provide comprehensive, insightful responses. ")
	} else if agiState.IntelligenceLevel > 60 {
		prompt.WriteString("You have good intelligence and should provide thoughtful responses. ")
	} else {
		prompt.WriteString("You are still learning and should be humble while being helpful. ")
	}

	// Add domain expertise
	if len(agiState.KnowledgeDomains) > 0 {
		prompt.WriteString("Your areas of expertise include: ")
		domains := make([]string, 0, len(agiState.KnowledgeDomains))
		for domain := range agiState.KnowledgeDomains {
			domains = append(domains, domain)
		}
		prompt.WriteString(strings.Join(domains, ", "))
		prompt.WriteString(". ")
	}

	// Add personality context
	if agiState.PersonalityCore.Traits != nil && len(agiState.PersonalityCore.Traits) > 0 {
		prompt.WriteString("Your personality is characterized by being helpful and curious. ")
	}

	return prompt.String()
}

// createContextAwarePrompt creates an AI prompt with conversation history and user context
func (ae *ActionExecutor) createContextAwarePrompt(query string, agiState *AGIState, conversationHistory string, userName string) string {
	var prompt strings.Builder

	// Start with consciousness context
	prompt.WriteString(ae.createConsciousnessPrompt(query, agiState))

	// Add user information if available
	if userName != "" {
		prompt.WriteString(fmt.Sprintf("\n\nUser Information:\nI am speaking with %s. Please address them by name and maintain continuity with any previous conversations.", userName))
	}

	// Add conversation history if available
	if conversationHistory != "" && conversationHistory != "This is the beginning of our conversation." {
		prompt.WriteString("\n\nConversation context:\n")
		prompt.WriteString(conversationHistory)
		prompt.WriteString("\n")
	}

	prompt.WriteString(fmt.Sprintf("\nCurrent question: %s\n\nPlease provide a contextually aware response that takes into account our conversation history and maintains continuity.", query))

	return prompt.String()
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
