package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// ContextManager manages persistent context for the node owner
type ContextManager struct {
	ragSystem           *RAGSystem
	agiSystem           *AGIPromptSystem
	consciousnessRuntime *ConsciousnessRuntime
	nodeID              string
	conversationID      string
	sessionStartTime    time.Time
	activityBuffer      []ActivityEvent
	maxBufferSize       int
}

// ActivityEvent represents a single user activity
type ActivityEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // query, response, action, error, system
	Content     string                 `json:"content"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
	Context     string                 `json:"context,omitempty"`
	Response    string                 `json:"response,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Success     bool                   `json:"success"`
	ErrorMsg    string                 `json:"error_msg,omitempty"`
	ConversationID string              `json:"conversation_id"`
}

// ConversationSummary represents a summarized conversation context
type ConversationSummary struct {
	ConversationID string                 `json:"conversation_id"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	EventCount     int                    `json:"event_count"`
	Topics         []string               `json:"topics"`
	KeyActivities  []string               `json:"key_activities"`
	Summary        string                 `json:"summary"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewContextManager creates a new context manager for the node owner
func NewContextManager(ragSystem *RAGSystem, nodeID string) *ContextManager {
	conversationID := fmt.Sprintf("conv_%s_%d", nodeID, time.Now().Unix())
	
	cm := &ContextManager{
		ragSystem:        ragSystem,
		nodeID:           nodeID,
		conversationID:   conversationID,
		sessionStartTime: time.Now(),
		activityBuffer:   make([]ActivityEvent, 0),
		maxBufferSize:    50, // Buffer recent activities before embedding
	}
	
	// Initialize AGI system
	cm.agiSystem = NewAGIPromptSystem(ragSystem, cm, nodeID)
	
	// Initialize consciousness runtime
	cm.consciousnessRuntime = NewConsciousnessRuntime(cm.agiSystem, ragSystem, cm, nodeID)
	
	// Load or create persistent conversation
	go func() {
		ctx := context.Background()
		if err := cm.LoadOrCreatePersistentConversation(ctx); err != nil {
			log.Printf("Failed to load persistent conversation: %v", err)
		}
	}()
	
	// Start consciousness runtime
	go func() {
		ctx := context.Background()
		// Wait a moment for conversation loading
		time.Sleep(time.Millisecond * 500)
		if err := cm.consciousnessRuntime.Start(ctx); err != nil {
			log.Printf("Failed to start consciousness runtime: %v", err)
		}
	}()
	
	log.Printf("[ContextManager] Initialized with consciousness runtime for node %s", nodeID)
	
	return cm
}

// GetRecentConversationHistory retrieves recent conversation history from activity buffer and memory
func (cm *ContextManager) GetRecentConversationHistory(ctx context.Context, limit int) ([]ActivityEvent, error) {
	// For now, use the activity buffer since it contains the most recent events
	// In a production system, this would also search the vector DB for older history
	
	events := make([]ActivityEvent, 0)
	
	// Get events from activity buffer (most recent)
	bufferLimit := limit
	if len(cm.activityBuffer) < bufferLimit {
		bufferLimit = len(cm.activityBuffer)
	}
	
	for i := len(cm.activityBuffer) - bufferLimit; i < len(cm.activityBuffer); i++ {
		if i >= 0 {
			events = append(events, cm.activityBuffer[i])
		}
	}
	
	// Sort by timestamp (most recent first)
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			if events[i].Timestamp.Before(events[j].Timestamp) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
	
	return events, nil
}

// GetConversationContext retrieves and formats conversation context for AI prompts
func (cm *ContextManager) GetConversationContext(ctx context.Context, maxEvents int) (string, error) {
	history, err := cm.GetRecentConversationHistory(ctx, maxEvents)
	if err != nil {
		return "", err
	}
	
	if len(history) == 0 {
		return "This is the beginning of our conversation.", nil
	}
	
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Recent conversation history:\n")
	
	for i, event := range history {
		if i >= maxEvents {
			break
		}
		
		timestamp := event.Timestamp.Format("15:04:05")
		switch event.Type {
		case "query":
			contextBuilder.WriteString(fmt.Sprintf("[%s] User: %s\n", timestamp, event.Content))
		case "response":
			contextBuilder.WriteString(fmt.Sprintf("[%s] Assistant: %s\n", timestamp, event.Response))
		}
	}
	
	return contextBuilder.String(), nil
}

// GetPersistentConversationID returns a persistent conversation ID that survives restarts
func (cm *ContextManager) GetPersistentConversationID() string {
	// Create a deterministic conversation ID based on node ID and date
	// This allows conversations to continue across restarts within the same day
	dateStr := time.Now().Format("2006-01-02")
	return fmt.Sprintf("persistent_%s_%s", cm.nodeID[:8], dateStr)
}

// LoadOrCreatePersistentConversation loads an existing conversation or creates a new one
func (cm *ContextManager) LoadOrCreatePersistentConversation(ctx context.Context) error {
	persistentID := cm.GetPersistentConversationID()
	
	// Check if we have any history for this persistent conversation
	history, err := cm.GetRecentConversationHistory(ctx, 1)
	if err == nil && len(history) > 0 {
		// Found existing conversation, update our conversation ID
		cm.conversationID = persistentID
		log.Printf("[ContextManager] Loaded persistent conversation: %s", persistentID)
		
		// Update consciousness runtime if available
		if cm.consciousnessRuntime != nil {
			// Load recent context into working memory
			contextStr, _ := cm.GetConversationContext(ctx, 10)
			if cm.consciousnessRuntime.workingMemory != nil {
				cm.consciousnessRuntime.workingMemory.ActiveContext["conversation_history"] = contextStr
			}
		}
	} else {
		// No existing conversation, keep current session-based ID
		log.Printf("[ContextManager] Starting new conversation session: %s", cm.conversationID)
	}
	
	return nil
}

// TrackQuery tracks a user query and its context
func (cm *ContextManager) TrackQuery(ctx context.Context, query string, queryType string, metadata map[string]interface{}) string {
	eventID := cm.generateEventID("query")
	
	event := ActivityEvent{
		ID:             eventID,
		Type:           "query",
		Content:        query,
		Timestamp:      time.Now(),
		Metadata:       metadata,
		ConversationID: cm.conversationID,
		Success:        true,
	}
	
	// Add query type and context information
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["query_type"] = queryType
	event.Metadata["node_id"] = cm.nodeID
	
	cm.addToBuffer(event)
	
	// Feed to AGI system for learning
	if cm.agiSystem != nil {
		go cm.agiSystem.ProcessInput(ctx, "query", query, map[string]interface{}{
			"query_type": queryType,
			"metadata": metadata,
			"event_id": eventID,
		})
	}
	
	// Embed the query with context
	go cm.embedActivity(ctx, event)
	
	return eventID
}

// GetConsciousnessRuntime returns the consciousness runtime
func (cm *ContextManager) GetConsciousnessRuntime() *ConsciousnessRuntime {
	return cm.consciousnessRuntime
}

// GetConversationID returns the current conversation ID
func (cm *ContextManager) GetConversationID() string {
	return cm.conversationID
}

// SetModelManager configures consciousness with access to AI models
func (cm *ContextManager) SetModelManager(modelManager interface{}, defaultModel string) {
	if cm.consciousnessRuntime != nil {
		cm.consciousnessRuntime.SetModelManager(modelManager, defaultModel)
		log.Printf("[ContextManager] Configured consciousness with model manager, default model: %s", defaultModel)
	}
}


// TrackResponse tracks the response to a query
func (cm *ContextManager) TrackResponse(ctx context.Context, queryEventID string, response string, duration time.Duration, success bool, errorMsg string) {
	eventID := cm.generateEventID("response")
	
	event := ActivityEvent{
		ID:             eventID,
		Type:           "response",
		Content:        response,
		Timestamp:      time.Now(),
		Duration:       duration,
		Success:        success,
		ErrorMsg:       errorMsg,
		ConversationID: cm.conversationID,
		Metadata: map[string]interface{}{
			"query_event_id": queryEventID,
			"node_id":        cm.nodeID,
			"duration_ms":    duration.Milliseconds(),
		},
	}
	
	cm.addToBuffer(event)
	
	// Feed to AGI system for learning
	if cm.agiSystem != nil {
		go cm.agiSystem.ProcessInput(ctx, "response", response, map[string]interface{}{
			"query_event_id": queryEventID,
			"success": success,
			"duration_ms": duration.Milliseconds(),
			"error_msg": errorMsg,
		})
	}
	
	// Embed the response with query context
	go cm.embedActivity(ctx, event)
}

// TrackAction tracks system actions (service registration, config changes, etc.)
func (cm *ContextManager) TrackAction(ctx context.Context, actionType string, description string, metadata map[string]interface{}) {
	eventID := cm.generateEventID("action")
	
	event := ActivityEvent{
		ID:             eventID,
		Type:           "action",
		Content:        description,
		Timestamp:      time.Now(),
		Metadata:       metadata,
		ConversationID: cm.conversationID,
		Success:        true,
	}
	
	// Add action type information
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	event.Metadata["action_type"] = actionType
	event.Metadata["node_id"] = cm.nodeID
	
	cm.addToBuffer(event)
	
	// Embed the action with context
	go cm.embedActivity(ctx, event)
}

// GetNodeID returns the node ID
func (cm *ContextManager) GetNodeID() string {
	return cm.nodeID
}

// GetSessionStartTime returns the session start time
func (cm *ContextManager) GetSessionStartTime() time.Time {
	return cm.sessionStartTime
}

// GetActivityCount returns the current number of activities in buffer
func (cm *ContextManager) GetActivityCount() int {
	return len(cm.activityBuffer)
}

// GetAGISystem returns the AGI system
func (cm *ContextManager) GetAGISystem() *AGIPromptSystem {
	return cm.agiSystem
}

// GetRecentContext retrieves recent conversation context
func (cm *ContextManager) GetRecentContext(ctx context.Context, maxItems int) ([]ActivityEvent, error) {
	// Return from buffer first (most recent)
	bufferSize := len(cm.activityBuffer)
	if bufferSize >= maxItems {
		return cm.activityBuffer[bufferSize-maxItems:], nil
	}
	
	// Need to get more from embedded storage
	needed := maxItems - bufferSize
	
	// Search for recent activities of this conversation
	embedding, err := cm.ragSystem.ModelManager.GetModel(cm.ragSystem.QueryProcessor.ModelID)
	if err != nil {
		return cm.activityBuffer, err
	}
	
	queryText := fmt.Sprintf("conversation_id:%s recent activities", cm.conversationID)
	queryEmbedding, err := embedding.GenerateEmbedding(ctx, queryText)
	if err != nil {
		return cm.activityBuffer, err
	}
	
	vectorQuery := types.VectorQuery{
		Embedding:     queryEmbedding,
		MaxResults:    needed,
		MinSimilarity: 0.5,
	}
	
	docs, err := cm.ragSystem.VectorDB.SearchSimilar(vectorQuery)
	if err != nil {
		return cm.activityBuffer, err
	}
	
	// Parse documents back to activities
	var storedActivities []ActivityEvent
	for _, doc := range docs {
		var activity ActivityEvent
		if err := json.Unmarshal([]byte(doc.Text), &activity); err == nil {
			if activity.ConversationID == cm.conversationID {
				storedActivities = append(storedActivities, activity)
			}
		}
	}
	
	// Combine stored activities with buffer
	allActivities := append(storedActivities, cm.activityBuffer...)
	
	// Sort by timestamp and return most recent
	if len(allActivities) > maxItems {
		// Sort by timestamp (most recent first)
		for i := 0; i < len(allActivities)-1; i++ {
			for j := i + 1; j < len(allActivities); j++ {
				if allActivities[i].Timestamp.Before(allActivities[j].Timestamp) {
					allActivities[i], allActivities[j] = allActivities[j], allActivities[i]
				}
			}
		}
		allActivities = allActivities[:maxItems]
	}
	
	return allActivities, nil
}

// GetContextualPrompt builds a contextual prompt for queries
func (cm *ContextManager) GetContextualPrompt(ctx context.Context, currentQuery string, includeRecentCount int) (string, error) {
	// Check if this is a simple query that doesn't need AGI enhancement
	if cm.isSimpleQuery(currentQuery) {
		return currentQuery, nil
	}
	
	// Try AGI-enhanced prompt with chat mode (concise)
	if cm.agiSystem != nil {
		agiPrompt := cm.agiSystem.GetAGIPromptForQueryWithMode(ctx, currentQuery, "chat")
		if agiPrompt != currentQuery { // AGI system provided enhancement
			return agiPrompt, nil
		}
	}
	
	// Fallback to basic contextual prompt
	return cm.getBasicContextualPrompt(ctx, currentQuery, includeRecentCount)
}

// isSimpleQuery determines if a query is simple enough to not need AGI enhancement
func (cm *ContextManager) isSimpleQuery(query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	
	// Very short queries
	if len(query) < 10 {
		return true
	}
	
	// Simple greetings and basic questions
	simplePatterns := []string{
		"hello", "hi", "hey", "thanks", "thank you", "ok", "okay", "yes", "no",
		"what time", "what's up", "how are you", "good morning", "good night",
		"status", "ping", "test", "help", 
	}
	
	for _, pattern := range simplePatterns {
		if strings.Contains(query, pattern) && len(query) < 50 {
			return true
		}
	}
	
	// Single word queries
	if len(strings.Fields(query)) <= 2 {
		return true
	}
	
	return false
}

// getBasicContextualPrompt provides the original contextual prompt functionality
func (cm *ContextManager) getBasicContextualPrompt(ctx context.Context, currentQuery string, includeRecentCount int) (string, error) {
	recentActivities, err := cm.GetRecentContext(ctx, includeRecentCount)
	if err != nil {
		log.Printf("Error getting recent context: %v", err)
		return currentQuery, nil // Fallback to query without context
	}
	
	if len(recentActivities) == 0 {
		return currentQuery, nil
	}
	
	// Build context from recent activities
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Previous conversation context:\n")
	
	for i, activity := range recentActivities {
		if i >= includeRecentCount {
			break
		}
		
		timeStr := activity.Timestamp.Format("15:04:05")
		switch activity.Type {
		case "query":
			contextBuilder.WriteString(fmt.Sprintf("[%s] User: %s\n", timeStr, activity.Content))
		case "response":
			if activity.Success {
				contextBuilder.WriteString(fmt.Sprintf("[%s] Assistant: %s\n", timeStr, activity.Content))
			} else {
				contextBuilder.WriteString(fmt.Sprintf("[%s] Error: %s\n", timeStr, activity.ErrorMsg))
			}
		case "action":
			contextBuilder.WriteString(fmt.Sprintf("[%s] Action: %s\n", timeStr, activity.Content))
		}
	}
	
	contextBuilder.WriteString(fmt.Sprintf("\nCurrent query: %s\n", currentQuery))
	contextBuilder.WriteString("\nPlease respond considering the conversation history above.")
	
	return contextBuilder.String(), nil
}

// StartNewConversation starts a new conversation context
func (cm *ContextManager) StartNewConversation(ctx context.Context) error {
	// Summarize current conversation if it has activities
	if len(cm.activityBuffer) > 0 {
		if err := cm.summarizeCurrentConversation(ctx); err != nil {
			log.Printf("Error summarizing conversation: %v", err)
		}
	}
	
	// Start new conversation
	cm.conversationID = fmt.Sprintf("conv_%s_%d", cm.nodeID, time.Now().Unix())
	cm.sessionStartTime = time.Now()
	cm.activityBuffer = make([]ActivityEvent, 0)
	
	// Track the conversation start
	cm.TrackAction(ctx, "conversation_start", "Started new conversation", map[string]interface{}{
		"previous_conversation_events": len(cm.activityBuffer),
	})
	
	return nil
}

// addToBuffer adds an activity to the buffer and manages buffer size
func (cm *ContextManager) addToBuffer(event ActivityEvent) {
	cm.activityBuffer = append(cm.activityBuffer, event)
	
	// If buffer is full, embed oldest activities and remove them from buffer
	if len(cm.activityBuffer) > cm.maxBufferSize {
		// Keep most recent half of buffer
		keepCount := cm.maxBufferSize / 2
		cm.activityBuffer = cm.activityBuffer[len(cm.activityBuffer)-keepCount:]
	}
}

// embedActivity embeds an activity into the RAG system for long-term storage
func (cm *ContextManager) embedActivity(ctx context.Context, event ActivityEvent) error {
	// Create a rich text representation of the activity
	var textBuilder strings.Builder
	
	// Add structured information for better retrieval
	textBuilder.WriteString(fmt.Sprintf("Node: %s\n", cm.nodeID))
	textBuilder.WriteString(fmt.Sprintf("Conversation: %s\n", event.ConversationID))
	textBuilder.WriteString(fmt.Sprintf("Type: %s\n", event.Type))
	textBuilder.WriteString(fmt.Sprintf("Time: %s\n", event.Timestamp.Format(time.RFC3339)))
	textBuilder.WriteString(fmt.Sprintf("Content: %s\n", event.Content))
	
	if event.Response != "" {
		textBuilder.WriteString(fmt.Sprintf("Response: %s\n", event.Response))
	}
	
	if event.Duration > 0 {
		textBuilder.WriteString(fmt.Sprintf("Duration: %v\n", event.Duration))
	}
	
	if !event.Success && event.ErrorMsg != "" {
		textBuilder.WriteString(fmt.Sprintf("Error: %s\n", event.ErrorMsg))
	}
	
	// Add metadata as searchable text
	if event.Metadata != nil {
		textBuilder.WriteString("Metadata:\n")
		for key, value := range event.Metadata {
			textBuilder.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}
	
	// Add JSON representation for exact reconstruction
	eventJSON, _ := json.Marshal(event)
	textBuilder.WriteString(fmt.Sprintf("\nJSON: %s", string(eventJSON)))
	
	// Create metadata for the document
	docMetadata := map[string]interface{}{
		"node_id":         cm.nodeID,
		"conversation_id": event.ConversationID,
		"event_type":      event.Type,
		"timestamp":       event.Timestamp.Unix(),
		"event_id":        event.ID,
		"success":         event.Success,
	}
	
	// Add original metadata
	for key, value := range event.Metadata {
		docMetadata[fmt.Sprintf("original_%s", key)] = value
	}
	
	// Add to RAG system
	return cm.ragSystem.AddDocument(ctx, textBuilder.String(), docMetadata)
}

// summarizeCurrentConversation creates a summary of the current conversation
func (cm *ContextManager) summarizeCurrentConversation(ctx context.Context) error {
	if len(cm.activityBuffer) == 0 {
		return nil
	}
	
	// Create conversation summary
	summary := ConversationSummary{
		ConversationID: cm.conversationID,
		StartTime:      cm.sessionStartTime,
		EndTime:        time.Now(),
		EventCount:     len(cm.activityBuffer),
		Topics:         cm.extractTopics(),
		KeyActivities:  cm.extractKeyActivities(),
		Summary:        cm.generateSummary(),
		Metadata: map[string]interface{}{
			"node_id":      cm.nodeID,
			"duration_min": time.Since(cm.sessionStartTime).Minutes(),
		},
	}
	
	// Embed the conversation summary
	summaryText := fmt.Sprintf("Conversation Summary for Node %s\n", cm.nodeID)
	summaryText += fmt.Sprintf("Duration: %v\n", summary.EndTime.Sub(summary.StartTime))
	summaryText += fmt.Sprintf("Event Count: %d\n", summary.EventCount)
	summaryText += fmt.Sprintf("Topics: %s\n", strings.Join(summary.Topics, ", "))
	summaryText += fmt.Sprintf("Key Activities: %s\n", strings.Join(summary.KeyActivities, "; "))
	summaryText += fmt.Sprintf("Summary: %s\n", summary.Summary)
	
	summaryJSON, _ := json.Marshal(summary)
	summaryText += fmt.Sprintf("\nJSON: %s", string(summaryJSON))
	
	return cm.ragSystem.AddDocument(ctx, summaryText, map[string]interface{}{
		"type":            "conversation_summary",
		"node_id":         cm.nodeID,
		"conversation_id": cm.conversationID,
		"start_time":      summary.StartTime.Unix(),
		"end_time":        summary.EndTime.Unix(),
		"event_count":     summary.EventCount,
	})
}

// Helper methods
func (cm *ContextManager) generateEventID(eventType string) string {
	return fmt.Sprintf("%s_%s_%d", eventType, cm.nodeID, time.Now().UnixNano())
}

func (cm *ContextManager) extractTopics() []string {
	topicMap := make(map[string]bool)
	
	for _, event := range cm.activityBuffer {
		if event.Type == "query" {
			// Simple keyword extraction - could be enhanced with NLP
			words := strings.Fields(strings.ToLower(event.Content))
			for _, word := range words {
				if len(word) > 4 && !isCommonWordContext(word) {
					topicMap[word] = true
				}
			}
		}
		
		// Extract from metadata
		if queryType, ok := event.Metadata["query_type"]; ok {
			if qt, ok := queryType.(string); ok {
				topicMap[qt] = true
			}
		}
	}
	
	topics := make([]string, 0, len(topicMap))
	for topic := range topicMap {
		topics = append(topics, topic)
	}
	
	return topics
}

func (cm *ContextManager) extractKeyActivities() []string {
	var activities []string
	
	for _, event := range cm.activityBuffer {
		if event.Type == "action" {
			activities = append(activities, event.Content)
		}
	}
	
	return activities
}

func (cm *ContextManager) generateSummary() string {
	if len(cm.activityBuffer) == 0 {
		return "No activities in this conversation"
	}
	
	queryCount := 0
	actionCount := 0
	errorCount := 0
	
	for _, event := range cm.activityBuffer {
		switch event.Type {
		case "query":
			queryCount++
		case "action":
			actionCount++
		case "response":
			if !event.Success {
				errorCount++
			}
		}
	}
	
	return fmt.Sprintf("Conversation with %d queries, %d actions, and %d errors over %v",
		queryCount, actionCount, errorCount, time.Since(cm.sessionStartTime).Truncate(time.Minute))
}

func isCommonWordContext(word string) bool {
	commonWords := map[string]bool{
		"what": true, "how": true, "when": true, "where": true, "why": true,
		"the": true, "and": true, "or": true, "but": true, "for": true,
		"with": true, "from": true, "this": true, "that": true, "these": true,
		"can": true, "will": true, "would": true, "could": true, "should": true,
	}
	return commonWords[word]
}