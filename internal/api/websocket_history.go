package api

import (
	"context"
	"fmt"
	"time"
)

// handleGetConversationHistory handles conversation history retrieval requests
func (c *WebSocketConnection) handleGetConversationHistory(msg *WebSocketMessage) {
	if c.Manager.server.RAGSystem == nil || c.Manager.server.RAGSystem.ContextManager == nil {
		c.sendError(msg.ID, "Context manager not available")
		return
	}

	// Extract limit from request data
	requestData, ok := msg.Data.(map[string]interface{})
	if !ok {
		requestData = make(map[string]interface{})
	}

	limit := 10 // Default limit
	if limitVal, exists := requestData["limit"]; exists {
		if limitFloat, ok := limitVal.(float64); ok {
			limit = int(limitFloat)
		} else if limitInt, ok := limitVal.(int); ok {
			limit = limitInt
		}
	}

	// Cap limit to prevent excessive memory usage
	if limit > 50 {
		limit = 50
	}

	go func() {
		ctx := context.Background()
		contextManager := c.Manager.server.RAGSystem.ContextManager
		
		// Get recent conversation history
		history, err := contextManager.GetRecentConversationHistory(ctx, limit)
		if err != nil {
			c.Send <- &WebSocketMessage{
				Type: WSMsgTypeResponse,
				ID:   msg.ID,
				Data: map[string]interface{}{
					"error": fmt.Sprintf("Failed to retrieve conversation history: %v", err),
				},
				Timestamp: time.Now(),
			}
			return
		}

		// Convert ActivityEvent slice to chat message format for frontend
		chatMessages := make([]map[string]interface{}, 0)
		
		for _, event := range history {
			// Convert each activity event to a chat message
			if event.Type == "query" || event.Type == "response" {
				role := "user"
				content := event.Content
				
				if event.Type == "response" {
					role = "assistant"
					if event.Response != "" {
						content = event.Response
					}
				}
				
				chatMessage := map[string]interface{}{
					"id":        event.ID,
					"content":   content,
					"role":      role,
					"timestamp": event.Timestamp.Format(time.RFC3339),
				}
				
				chatMessages = append(chatMessages, chatMessage)
			}
		}

		// Send response
		c.Send <- &WebSocketMessage{
			Type: WSMsgTypeResponse,
			ID:   msg.ID,
			Data: map[string]interface{}{
				"messages":        chatMessages,
				"conversation_id": c.ConversationID,
				"total_events":    len(history),
				"retrieved_at":    time.Now(),
			},
			Timestamp: time.Now(),
		}
	}()
}