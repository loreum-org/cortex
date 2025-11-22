package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// ContextManager manages persistent context for the node owner
type ContextManager struct {
	ragSystem            *RAGSystem
	agiSystem            *AGIPromptSystem
	nodeID               string
	conversationID       string
	sessionStartTime     time.Time
	activityBuffer       []ActivityEvent
	maxBufferSize        int
}

// ActivityEvent represents a single user activity
type ActivityEvent struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"` // query, response, action, error, system
	Content        string                 `json:"content"`
	Timestamp      time.Time              `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata"`
	Context        string                 `json:"context,omitempty"`
	Response       string                 `json:"response,omitempty"`
	Duration       time.Duration          `json:"duration,omitempty"`
	Success        bool                   `json:"success"`
	ErrorMsg       string                 `json:"error_msg,omitempty"`
	ConversationID string                 `json:"conversation_id"`
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
	// Use persistent conversation ID based on node ID only - will be updated in LoadOrCreatePersistentConversation
	conversationID := fmt.Sprintf("persistent_conv_%s", nodeID)

	cm := &ContextManager{
		ragSystem:        ragSystem,
		nodeID:           nodeID,
		conversationID:   conversationID,
		sessionStartTime: time.Now(),
		activityBuffer:   make([]ActivityEvent, 0),
		maxBufferSize:    50, // Buffer recent activities before embedding
	}

	// Initialize AGI system with integrated consciousness
	cm.agiSystem = NewAGIPromptSystem(ragSystem, cm, nodeID)

	// Load or create persistent conversation and then start AGI consciousness
	go func() {
		ctx := context.Background()
		
		// Load conversation context first
		if err := cm.LoadOrCreatePersistentConversation(ctx); err != nil {
			log.Printf("Failed to load persistent conversation: %v", err)
		} else {
			log.Printf("[ContextManager] Loaded persistent conversation context for node %s", nodeID)
		}
		
		// Wait a moment for context to be fully loaded
		time.Sleep(time.Millisecond * 1000)
		
		// Start AGI consciousness loop after context is loaded
		if err := cm.agiSystem.StartConsciousnessLoop(ctx); err != nil {
			log.Printf("Failed to start AGI consciousness loop: %v", err)
		} else {
			log.Printf("[ContextManager] AGI consciousness loop started after conversation context loading")
		}
	}()

	log.Printf("[ContextManager] Initialized with persistent conversation memory for node %s", nodeID)

	return cm
}

// GetRecentConversationHistory retrieves recent conversation history from activity buffer and memory
func (cm *ContextManager) GetRecentConversationHistory(ctx context.Context, limit int) ([]ActivityEvent, error) {
	events := make([]ActivityEvent, 0)

	// First, get events from activity buffer (most recent)
	bufferLimit := limit
	if len(cm.activityBuffer) < bufferLimit {
		bufferLimit = len(cm.activityBuffer)
	}

	for i := len(cm.activityBuffer) - bufferLimit; i < len(cm.activityBuffer); i++ {
		if i >= 0 {
			events = append(events, cm.activityBuffer[i])
		}
	}

	// If we need more events and have vector DB access, search it
	if len(events) < limit && cm.ragSystem != nil && cm.ragSystem.VectorDB != nil {
		// Search for conversation documents in vector DB
		needed := limit - len(events)
		vectorEvents := cm.searchVectorDBForHistory(ctx, needed)

		// Add vector DB events (they're older than buffer events)
		for _, event := range vectorEvents {
			events = append(events, event)
		}

		// Only log if we actually found events
		if len(vectorEvents) > 0 {
			log.Printf("[ContextManager] Found %d events in vector DB", len(vectorEvents))
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

	// Only log if we're returning substantial events or if debugging
	if len(events) > 0 {
		log.Printf("[ContextManager] Returning %d conversation history events", len(events))
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
	// Use a stable node ID that persists across restarts
	// instead of the P2P node ID which changes every restart
	stableID := cm.getStableNodeID()
	return fmt.Sprintf("persistent_conv_%s", stableID)
}

// getStableNodeID returns a stable node identifier that persists across restarts
func (cm *ContextManager) getStableNodeID() string {
	// Try to load existing stable ID from file
	stableIDPath := "data/.cortex_node_id"
	if data, err := os.ReadFile(stableIDPath); err == nil {
		return strings.TrimSpace(string(data))
	}

	// If file doesn't exist, create a new stable ID based on hostname or generate one
	stableID := "cortex_node_stable_001"
	if hostname, err := os.Hostname(); err == nil {
		// Use hostname as part of stable ID
		stableID = fmt.Sprintf("cortex_node_%s", hostname)
	}

	// Save the stable ID for future use
	os.MkdirAll("data", 0755)
	if err := os.WriteFile(stableIDPath, []byte(stableID), 0644); err != nil {
		log.Printf("[ContextManager] Warning: Failed to save stable node ID: %v", err)
	} else {
		log.Printf("[ContextManager] Created new stable node ID: %s", stableID)
	}

	return stableID
}

// GetUserPersistentConversationID returns a user-specific persistent conversation ID
func (cm *ContextManager) GetUserPersistentConversationID(userID string) string {
	// Create a deterministic conversation ID based on user ID and date
	// This allows user conversations to continue across restarts within the same day
	dateStr := time.Now().Format("2006-01-02")
	
	// Use user ID hash for privacy and consistent length
	userHash := cm.hashUserID(userID)
	return fmt.Sprintf("user_persistent_%s_%s_%s", cm.nodeID[:8], userHash, dateStr)
}

// hashUserID creates a consistent hash of user ID for conversation naming
func (cm *ContextManager) hashUserID(userID string) string {
	// Simple hash implementation
	hash := 0
	for _, char := range userID {
		hash = hash*31 + int(char)
	}
	
	// Convert to positive hex string (8 characters)
	if hash < 0 {
		hash = -hash
	}
	
	return fmt.Sprintf("%08x", hash)[:8]
}

// LoadOrCreatePersistentConversation loads an existing conversation or creates a new one
func (cm *ContextManager) LoadOrCreatePersistentConversation(ctx context.Context) error {
	persistentID := cm.GetPersistentConversationID()

	// Always use the persistent conversation ID
	cm.conversationID = persistentID
	
	// First check vector database directly for conversation history
	// This is critical for loading context after node restarts
	vectorHistory := cm.searchVectorDBForHistory(ctx, 10) // Get some recent history
	
	// Also check activity buffer (will be empty on restart)
	bufferHistory, err := cm.GetRecentConversationHistory(ctx, 5)
	
	hasVectorHistory := len(vectorHistory) > 0
	hasBufferHistory := err == nil && len(bufferHistory) > 0
	
	if hasVectorHistory || hasBufferHistory {
		// Found existing conversation data
		log.Printf("[ContextManager] Loaded persistent conversation: %s (%d vector events, %d buffer events)", 
			persistentID, len(vectorHistory), len(bufferHistory))

		// Pre-populate activity buffer with recent vector history for immediate context
		if hasVectorHistory && len(cm.activityBuffer) < 5 {
			// Convert vector history back to activity events and add to buffer
			for i, vectorEvent := range vectorHistory {
				if i >= 5 { // Limit to prevent buffer overflow
					break
				}
				// Add as context restoration events
				contextEvent := ActivityEvent{
					ID:             fmt.Sprintf("restored_%d_%s", i, cm.nodeID),
					Type:           "context_restoration", 
					Content:        vectorEvent.Content,
					Timestamp:      vectorEvent.Timestamp,
					ConversationID: cm.conversationID,
					Success:        true,
					Metadata: map[string]interface{}{
						"restored_from": "vector_db",
						"original_type": vectorEvent.Type,
					},
				}
				cm.activityBuffer = append(cm.activityBuffer, contextEvent)
			}
			log.Printf("[ContextManager] Restored %d events to activity buffer from vector database", len(vectorHistory))
		}

		// Update AGI system with conversation context if available
		if cm.agiSystem != nil {
			log.Printf("[ContextManager] AGI system will load conversation context automatically")
		}
	} else {
		// No existing conversation, keep current session-based ID
		log.Printf("[ContextManager] Starting new conversation session: %s (no prior history found)", cm.conversationID)
	}

	// Ensure the conversation ID is tracked in user profile
	if cm.ragSystem != nil && cm.ragSystem.UserProfileManager != nil {
		nodeUserID := fmt.Sprintf("node_user_%s", cm.nodeID)
		if err := cm.ragSystem.UserProfileManager.AddConversationToProfile(nodeUserID, cm.conversationID); err != nil {
			log.Printf("[ContextManager] Warning: Failed to add conversation to user profile: %v", err)
		}
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
			"metadata":   metadata,
			"event_id":   eventID,
		})
	}

	// Embed the query with context
	go cm.embedActivity(ctx, event)

	return eventID
}

// GetConsciousnessRuntime returns the consciousness runtime
// GetConsciousnessRuntime returns nil - DEPRECATED: Use GetAGISystem instead
// Consciousness functionality is now integrated into AGI system
func (cm *ContextManager) GetConsciousnessRuntime() interface{} {
	return nil
}

// GetConversationID returns the current conversation ID
func (cm *ContextManager) GetConversationID() string {
	return cm.conversationID
}

// SetModelManager configures AGI system with access to AI models
func (cm *ContextManager) SetModelManager(modelManager interface{}, defaultModel string) {
	if cm.agiSystem != nil {
		cm.agiSystem.SetModelManager(modelManager, defaultModel)
		log.Printf("[ContextManager] Configured AGI system with model manager, default model: %s", defaultModel)
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
			"success":        success,
			"duration_ms":    duration.Milliseconds(),
			"error_msg":      errorMsg,
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

	// Create comprehensive metadata for the document with enhanced indexing
	docMetadata := map[string]interface{}{
		"node_id":         cm.nodeID,
		"conversation_id": event.ConversationID,
		"event_type":      event.Type,
		"timestamp":       event.Timestamp.Unix(),
		"event_id":        event.ID,
		"success":         event.Success,
		
		// Enhanced indexing fields
		"date":            event.Timestamp.Format("2006-01-02"),
		"hour":            event.Timestamp.Format("15"),
		"day_of_week":     event.Timestamp.Weekday().String(),
		"content_length":  len(event.Content),
		"has_response":    event.Response != "",
		"response_length": len(event.Response),
		
		// Searchable content analysis
		"content_words":   len(strings.Fields(event.Content)),
		"content_hash":    cm.hashContent(event.Content),
		
		// Session and user context
		"session_time":    event.Timestamp.Sub(cm.sessionStartTime).Minutes(),
		
		// Performance metrics
		"duration_ms":     event.Duration.Milliseconds(),
		"has_error":       !event.Success,
	}

	// Add user-specific metadata if available
	if cm.ragSystem.UserProfileManager != nil {
		// Try to identify user from conversation
		if userID := cm.extractUserIDFromEvent(event); userID != "" {
			docMetadata["user_id"] = userID
			docMetadata["user_hash"] = cm.hashUserID(userID)
			
			// Get user profile information
			if profile := cm.ragSystem.UserProfileManager.GetOrCreateUserProfile(userID); profile != nil {
				docMetadata["user_name"] = profile.Name
				docMetadata["user_preferred_name"] = profile.PreferredName
				docMetadata["user_conversations_count"] = len(profile.ConversationIDs)
			}
		}
	}

	// Add topic extraction for better categorization
	if event.Type == "query" {
		topics := cm.extractTopicsFromContent(event.Content)
		if len(topics) > 0 {
			docMetadata["topics"] = topics
			docMetadata["topic_count"] = len(topics)
		}
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

// AGI agent bridge functionality is now integrated into the AGI system directly

// searchVectorDBForHistory searches the vector database for conversation history
func (cm *ContextManager) searchVectorDBForHistory(ctx context.Context, limit int) []ActivityEvent {
	events := make([]ActivityEvent, 0)

	if cm.ragSystem == nil || cm.ragSystem.VectorDB == nil {
		return events
	}

	// Check vector DB stats to ensure compatibility
	stats := cm.ragSystem.VectorDB.GetStats()
	if stats.Size == 0 {
		// No documents in vector DB yet
		return events
	}

	// Create search query for conversation history - use persistent conversation ID
	persistentID := cm.GetPersistentConversationID()
	searchQuery := fmt.Sprintf("conversation %s history events messages node %s", 
		persistentID, cm.nodeID)

	// Try to generate embedding for the search query using the same model as RAG
	var embedding []float32
	var err error

	if cm.ragSystem.ModelManager != nil {
		// Use the RAG system's default model to ensure dimension compatibility
		model, err := cm.ragSystem.ModelManager.GetModel(cm.ragSystem.QueryProcessor.ModelID)
		if err == nil {
			embedding, err = model.GenerateEmbedding(ctx, searchQuery)
			if err != nil {
				log.Printf("[ContextManager] Failed to generate embedding with default model: %v", err)
			}
		}
		
		// If default model fails, try embedding models specifically
		if len(embedding) == 0 {
			models := cm.ragSystem.ModelManager.ListModels()
			for _, modelInfo := range models {
				model, err := cm.ragSystem.ModelManager.GetModel(modelInfo.ID)
				if err == nil {
					// Check if model supports embedding
					for _, capability := range modelInfo.Capabilities {
						if string(capability) == "embedding" {
							embedding, err = model.GenerateEmbedding(ctx, searchQuery)
							if err == nil && len(embedding) == stats.Dimensions {
								break
							}
						}
					}
					if len(embedding) == stats.Dimensions {
						break
					}
				}
			}
		}
	}

	if len(embedding) == 0 {
		// Don't log error repeatedly - just return empty
		return events
	}

	// Verify embedding dimensions match vector DB
	if len(embedding) != stats.Dimensions {
		// Don't log error repeatedly for dimension mismatch
		return events
	}

	// Search vector database
	query := types.VectorQuery{
		Embedding:     embedding,
		MaxResults:    limit * 2, // Get more results to filter
		MinSimilarity: 0.3,       // Lower threshold for conversation history
	}

	results, err := cm.ragSystem.VectorDB.SearchSimilar(query)
	if err != nil {
		// Don't log repeated search failures
		return events
	}

	// Convert vector documents back to ActivityEvents
	// persistentID already declared above
	for i, result := range results {
		if i >= limit {
			break
		}

		// Check if this is a conversation document - match against persistent conversation ID
		isRelevant := strings.Contains(result.ID, persistentID) ||
					  strings.Contains(result.Text, persistentID)
					  
		// Also check metadata for conversation_id match
		if !isRelevant {
			if convID, ok := result.Metadata["conversation_id"].(string); ok {
				isRelevant = convID == persistentID
			}
		}
		
		if isRelevant {
			// Parse metadata to reconstruct ActivityEvent
			event := ActivityEvent{
				Type:      "conversation_retrieval",
				Content:   result.Text, // Use Text field instead of Content
				Timestamp: time.Now(),  // Default timestamp - will be overridden if found in metadata
				ConversationID: persistentID,
				Success:   true,
				Metadata: map[string]interface{}{
					"retrieved_from": "vector_db",
					"document_id":    result.ID,
					"similarity":     result.Metadata["similarity"],
				},
			}

			// Try to extract original event data from metadata
			if eventType, ok := result.Metadata["event_type"].(string); ok {
				event.Type = eventType
			}
			if timestamp, ok := result.Metadata["timestamp"]; ok {
				// Handle both string and numeric timestamps
				switch ts := timestamp.(type) {
				case string:
					if parsedTime, err := time.Parse(time.RFC3339, ts); err == nil {
						event.Timestamp = parsedTime
					}
				case float64:
					event.Timestamp = time.Unix(int64(ts), 0)
				case int64:
					event.Timestamp = time.Unix(ts, 0)
				}
			}
			
			// Try to extract original event JSON if present
			if strings.Contains(result.Text, `"JSON":`) {
				// Extract JSON part for more accurate reconstruction
				jsonStart := strings.Index(result.Text, `"JSON":`)
				if jsonStart != -1 {
					jsonPart := strings.TrimSpace(result.Text[jsonStart+7:])
					var originalEvent ActivityEvent
					if err := json.Unmarshal([]byte(jsonPart), &originalEvent); err == nil {
						// Use original event data with some safety checks
						event = originalEvent
						event.ConversationID = persistentID // Ensure persistent conversation ID
						event.Metadata["retrieved_from"] = "vector_db"
						event.Metadata["document_id"] = result.ID
					}
				}
			}

			events = append(events, event)
		}
	}

	return events
}

// hashContent creates a hash of content for deduplication and indexing
func (cm *ContextManager) hashContent(content string) string {
	// Simple hash implementation for content fingerprinting
	hash := 0
	for _, char := range content {
		hash = hash*31 + int(char)
	}
	
	if hash < 0 {
		hash = -hash
	}
	
	return fmt.Sprintf("%08x", hash)
}

// extractUserIDFromEvent tries to extract user ID from event metadata
func (cm *ContextManager) extractUserIDFromEvent(event ActivityEvent) string {
	// Check metadata for user ID
	if event.Metadata != nil {
		if userID, ok := event.Metadata["user_id"].(string); ok {
			return userID
		}
	}
	
	// Could implement more sophisticated user identification here
	// For now, return empty if not found in metadata
	return ""
}

// extractTopicsFromContent extracts topics from content text for better categorization
func (cm *ContextManager) extractTopicsFromContent(content string) []string {
	topics := make([]string, 0)
	
	// Simple keyword-based topic extraction
	content = strings.ToLower(content)
	words := strings.Fields(content)
	
	// Look for significant words (longer than 4 characters, not common words)
	topicMap := make(map[string]bool)
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:()")
		if len(word) > 4 && !isCommonWordContext(word) {
			// Check if it's a meaningful topic word
			if cm.isMeaningfulTopic(word) {
				topicMap[word] = true
			}
		}
	}
	
	// Convert map to slice
	for topic := range topicMap {
		topics = append(topics, topic)
	}
	
	// Limit to prevent metadata bloat
	if len(topics) > 10 {
		topics = topics[:10]
	}
	
	return topics
}

// isMeaningfulTopic determines if a word is likely to be a meaningful topic
func (cm *ContextManager) isMeaningfulTopic(word string) bool {
	// Skip very common words even if they're long
	skipWords := map[string]bool{
		"something": true,
		"anything":  true,
		"everything": true,
		"nothing":   true,
		"someone":   true,
		"everyone":  true,
		"because":   true,
		"through":   true,
		"between":   true,
		"without":   true,
		"before":    true,
		"after":     true,
		"during":    true,
		"please":    true,
		"thanks":    true,
		"thank":     true,
	}
	
	if skipWords[word] {
		return false
	}
	
	// Include technical terms, proper nouns, and domain-specific words
	// This is a simple heuristic - could be enhanced with NLP
	return true
}
