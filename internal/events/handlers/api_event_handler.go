package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/agents"
	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/internal/economy"
	"github.com/loreum-org/cortex/internal/events"
	"github.com/loreum-org/cortex/internal/p2p"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/internal/services"
	"github.com/loreum-org/cortex/pkg/types"
)

// APIEventHandler handles API-related events and converts them to service calls
type APIEventHandler struct {
	// Service dependencies
	agentRegistry   *agents.AgentRegistry
	ragSystem       *rag.RAGSystem
	economicEngine  *economy.EconomicEngine
	serviceRegistry *services.ServiceRegistryManager
	p2pNode         *p2p.P2PNode
	embeddedManager *ai.EmbeddedModelManager
	contextManager  *rag.ContextManager

	// Event routing
	eventBus    *events.EventBus
	eventRouter *events.EventRouter
}

// NewAPIEventHandler creates a new API event handler
func NewAPIEventHandler(
	agentRegistry *agents.AgentRegistry,
	ragSystem *rag.RAGSystem,
	economicEngine *economy.EconomicEngine,
	serviceRegistry *services.ServiceRegistryManager,
	p2pNode *p2p.P2PNode,
	embeddedManager *ai.EmbeddedModelManager,
	contextManager *rag.ContextManager,
	eventBus *events.EventBus,
	eventRouter *events.EventRouter,
) *APIEventHandler {
	return &APIEventHandler{
		agentRegistry:   agentRegistry,
		ragSystem:       ragSystem,
		economicEngine:  economicEngine,
		serviceRegistry: serviceRegistry,
		p2pNode:         p2pNode,
		embeddedManager: embeddedManager,
		contextManager:  contextManager,
		eventBus:        eventBus,
		eventRouter:     eventRouter,
	}
}

// Handle processes API events
func (h *APIEventHandler) Handle(ctx context.Context, event events.Event) error {
	log.Printf("[APIEventHandler] Handling event: %s (correlation: %s)", event.Type, event.CorrelationID)

	switch event.Type {
	case events.EventTypeGetModels:
		return h.handleGetModels(ctx, event)
	case events.EventTypeGetAgents:
		return h.handleGetAgents(ctx, event)
	case events.EventTypeGetStatus:
		return h.handleGetStatus(ctx, event)
	case events.EventTypeGetMetrics:
		return h.handleGetMetrics(ctx, event)
	case events.EventTypeQuerySubmit:
		return h.handleQuerySubmit(ctx, event)
	case events.EventTypeModelDownload:
		return h.handleModelDownload(ctx, event)
	case events.EventTypeModelDelete:
		return h.handleModelDelete(ctx, event)
	case events.EventTypeGetTransactions:
		return h.handleGetTransactions(ctx, event)
	case events.EventTypeGetAccounts:
		return h.handleGetAccounts(ctx, event)
	case events.EventTypeWalletBalance:
		return h.handleWalletBalance(ctx, event)
	case events.EventTypeWalletTransfer:
		return h.handleWalletTransfer(ctx, event)
	case events.EventTypeGetServices:
		return h.handleGetServices(ctx, event)
	case events.EventTypeGetConversationHistory:
		return h.handleGetConversationHistory(ctx, event)
	case events.EventTypeServiceRegister:
		return h.handleServiceRegister(ctx, event)
	case events.EventTypeServiceDeregister:
		return h.handleServiceDeregister(ctx, event)
	case events.EventTypeGetNodeInfo:
		return h.handleGetNodeInfo(ctx, event)
	case events.EventTypeGetPeers:
		return h.handleGetPeers(ctx, event)
	default:
		return fmt.Errorf("unsupported event type: %s", event.Type)
	}
}

// SubscribedEvents returns the event types this handler subscribes to
func (h *APIEventHandler) SubscribedEvents() []string {
	return []string{
		events.EventTypeGetModels,
		events.EventTypeGetAgents,
		events.EventTypeGetStatus,
		events.EventTypeGetMetrics,
		events.EventTypeQuerySubmit,
		events.EventTypeModelDownload,
		events.EventTypeModelDelete,
		events.EventTypeGetTransactions,
		events.EventTypeGetAccounts,
		events.EventTypeWalletBalance,
		events.EventTypeWalletTransfer,
		events.EventTypeGetServices,
		events.EventTypeGetConversationHistory,
		events.EventTypeServiceRegister,
		events.EventTypeServiceDeregister,
		events.EventTypeGetNodeInfo,
		events.EventTypeGetPeers,
	}
}

// HandlerName returns the handler name
func (h *APIEventHandler) HandlerName() string {
	return "api_event_handler"
}

// Handler implementations

func (h *APIEventHandler) handleGetModels(ctx context.Context, event events.Event) error {
	var response interface{}
	var err error

	if h.embeddedManager != nil {
		// Get models from embedded manager
		installedModels := []interface{}{} // Placeholder - need to implement
		availableModels, _ := h.embeddedManager.GetAvailableModels(context.Background())

		response = map[string]interface{}{
			"installed": installedModels,
			"available": availableModels,
		}
	} else {
		response = map[string]interface{}{
			"installed": []interface{}{},
			"available": []interface{}{},
			"error":     "Embedded manager not available",
		}
	}

	return h.sendResponse(ctx, event, events.EventTypeModelsData, response, err)
}

func (h *APIEventHandler) handleGetAgents(ctx context.Context, event events.Event) error {
	if h.agentRegistry == nil {
		return h.sendResponse(ctx, event, events.EventTypeAgentsData, map[string]interface{}{
			"agents": []interface{}{},
			"error":  "Agent registry not available",
		}, nil)
	}

	agents := h.agentRegistry.GetAllAgents()
	agentList := make([]map[string]interface{}, 0, len(agents))

	for _, agentInfo := range agents {
		agentData := map[string]interface{}{
			"id":           agentInfo.ID,
			"name":         agentInfo.Name,
			"type":         agentInfo.Type,
			"status":       string(agentInfo.Status),
			"capabilities": agentInfo.Capabilities,
			"created_at":   agentInfo.CreatedAt.Unix(),
		}

		// Note: metrics would be added here if available

		agentList = append(agentList, agentData)
	}

	response := map[string]interface{}{
		"agents": agentList,
		"count":  len(agentList),
	}

	return h.sendResponse(ctx, event, events.EventTypeAgentsData, response, nil)
}

func (h *APIEventHandler) handleGetStatus(ctx context.Context, event events.Event) error {
	status := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(time.Now()).Seconds(), // This would be actual uptime in real implementation
		"version":   "1.0.0",
		"components": map[string]interface{}{
			"agent_registry":   h.agentRegistry != nil,
			"rag_system":       h.ragSystem != nil,
			"economic_engine":  h.economicEngine != nil,
			"service_registry": h.serviceRegistry != nil,
			"p2p_node":         h.p2pNode != nil,
			"embedded_manager": h.embeddedManager != nil,
		},
	}

	if h.p2pNode != nil {
		status["node_id"] = h.p2pNode.Host.ID().String()
		status["peer_count"] = len(h.p2pNode.Host.Network().Peers())
	}

	if h.agentRegistry != nil {
		status["active_agents"] = len(h.agentRegistry.GetAllAgents())
	}

	return h.sendResponse(ctx, event, events.EventTypeStatusData, status, nil)
}

func (h *APIEventHandler) handleGetMetrics(ctx context.Context, event events.Event) error {
	metrics := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"system": map[string]interface{}{
			"cpu_usage":    0.0, // Would implement actual metrics
			"memory_usage": 0.0,
			"disk_usage":   0.0,
		},
	}

	if h.agentRegistry != nil {
		agents := h.agentRegistry.GetAllAgents()
		agentMetrics := make(map[string]interface{})

		for _, agentInfo := range agents {
			// Add basic agent info to metrics
			agentMetrics[agentInfo.ID] = map[string]interface{}{
				"status": string(agentInfo.Status),
				"type":   agentInfo.Type,
			}
		}

		metrics["agents"] = agentMetrics
	}

	if h.p2pNode != nil {
		metrics["network"] = map[string]interface{}{
			"peer_count":     len(h.p2pNode.Host.Network().Peers()),
			"connections":    len(h.p2pNode.Host.Network().Conns()),
			"bytes_sent":     0, // Would implement actual metrics
			"bytes_received": 0,
		}
	}

	return h.sendResponse(ctx, event, events.EventTypeMetricsData, metrics, nil)
}

func (h *APIEventHandler) handleQuerySubmit(ctx context.Context, event events.Event) error {
	// Extract query data from event
	queryData, ok := event.Data.(events.QueryData)
	if !ok {
		return fmt.Errorf("invalid query data type: expected QueryData, got %T", event.Data)
	}

	queryText := queryData.Text
	if queryText == "" {
		return fmt.Errorf("missing or empty query text")
	}

	useRAG := queryData.UseRAG

	// Create query object
	query := &types.Query{
		ID:        fmt.Sprintf("query_%d", time.Now().UnixNano()),
		Text:      queryText,
		Type:      "user_query", // Use consistent type with agent handler
		Metadata:  make(map[string]string),
		Timestamp: time.Now().Unix(),
	}

	// Add query metadata
	query.Metadata["use_rag"] = fmt.Sprintf("%v", useRAG)
	query.Metadata["user_id"] = queryData.UserID
	query.Metadata["connection_id"] = queryData.ConnectionID

	// Route query through agent registry
	var response *types.Response
	var err error

	if h.agentRegistry != nil {
		response, err = h.agentRegistry.RouteQuery(ctx, query)
	} else {
		err = fmt.Errorf("agent registry not available")
	}

	// Prepare response data
	responseData := events.QueryResultData{
		QueryID: query.ID,
	}

	if err != nil {
		responseData.Error = err.Error()
	} else {
		responseData.Result = response.Text
		if response.Metadata != nil {
			// Convert map[string]string to map[string]interface{}
			metadata := make(map[string]interface{})
			for k, v := range response.Metadata {
				metadata[k] = v
			}
			responseData.Metadata = metadata
		}
	}

	// Send response event
	responseEvent := events.NewEventWithCorrelation(
		"query.response",
		"api_handler",
		responseData,
		event.CorrelationID,
	)

	// Propagate metadata from original event (including connection_id and original_query_id)
	if event.Metadata != nil {
		for key, value := range event.Metadata {
			responseEvent = responseEvent.WithMetadata(key, value)
		}
	}

	// Ensure we have the original query ID for WebSocket response matching
	if _, hasOriginalQueryID := responseEvent.Metadata["original_query_id"]; !hasOriginalQueryID {
		// If no original_query_id in metadata, use the correlation ID
		responseEvent = responseEvent.WithMetadata("original_query_id", event.CorrelationID)
	}

	// Publish to event bus for WebSocket handler and other subscribers
	if err := h.eventBus.Publish(responseEvent); err != nil {
		log.Printf("❌ API Handler: Failed to publish query response event: %v", err)
	}

	// Also route through correlation system for WebSocket bridge
	return h.eventRouter.RouteResponse(ctx, responseEvent)
}

func (h *APIEventHandler) handleModelDownload(ctx context.Context, event events.Event) error {
	// Extract model name from event data
	wsData, ok := event.Data.(events.WebSocketMessageData)
	if !ok {
		return fmt.Errorf("invalid event data type")
	}

	data, ok := wsData.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data format")
	}

	modelName, ok := data["model_name"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid model_name")
	}

	var err error
	if h.embeddedManager != nil {
		// Start model download asynchronously
		go func() {
			downloadErr := h.embeddedManager.PullModel(ctx, modelName, true)
			// Send progress/completion events
			statusEvent := events.NewEventWithCorrelation(
				"model.download.status",
				"api_handler",
				map[string]interface{}{
					"model_name": modelName,
					"status":     "completed",
					"success":    downloadErr == nil,
					"error": func() string {
						if downloadErr != nil {
							return downloadErr.Error()
						} else {
							return ""
						}
					}(),
				},
				event.CorrelationID,
			)
			h.eventRouter.RouteResponse(ctx, statusEvent)
		}()
		err = nil // Download started successfully
	} else {
		err = fmt.Errorf("embedded manager not available")
	}

	responseData := map[string]interface{}{
		"model_name": modelName,
		"success":    err == nil,
	}

	if err != nil {
		responseData["error"] = err.Error()
	}

	return h.sendResponse(ctx, event, "model.download.response", responseData, nil)
}

func (h *APIEventHandler) handleModelDelete(ctx context.Context, event events.Event) error {
	// Extract model name from event data
	wsData, ok := event.Data.(events.WebSocketMessageData)
	if !ok {
		return fmt.Errorf("invalid event data type")
	}

	data, ok := wsData.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data format")
	}

	modelName, ok := data["model_name"].(string)
	if !ok {
		return fmt.Errorf("missing or invalid model_name")
	}

	var err error
	if h.embeddedManager != nil {
		// Model deletion not implemented yet
		err = fmt.Errorf("model deletion not implemented")
	} else {
		err = fmt.Errorf("embedded manager not available")
	}

	responseData := map[string]interface{}{
		"model_name": modelName,
		"success":    err == nil,
	}

	if err != nil {
		responseData["error"] = err.Error()
	}

	return h.sendResponse(ctx, event, "model.delete.response", responseData, nil)
}

func (h *APIEventHandler) handleGetTransactions(ctx context.Context, event events.Event) error {
	var transactions []interface{}
	var err error

	if h.economicEngine != nil {
		// Get transactions from economic engine
		// This would be implemented based on the economic engine interface
		transactions = []interface{}{} // Placeholder
	} else {
		err = fmt.Errorf("economic engine not available")
	}

	response := map[string]interface{}{
		"transactions": transactions,
		"count":        len(transactions),
	}

	return h.sendResponse(ctx, event, events.EventTypeTransactionsData, response, err)
}

func (h *APIEventHandler) handleGetAccounts(ctx context.Context, event events.Event) error {
	var accounts []interface{}
	var err error

	if h.economicEngine != nil {
		// Get accounts from economic engine
		accounts = []interface{}{} // Placeholder
	} else {
		err = fmt.Errorf("economic engine not available")
	}

	response := map[string]interface{}{
		"accounts": accounts,
		"count":    len(accounts),
	}

	return h.sendResponse(ctx, event, events.EventTypeAccountsData, response, err)
}

func (h *APIEventHandler) handleWalletBalance(ctx context.Context, event events.Event) error {
	response := map[string]interface{}{
		"balance":  "0",
		"currency": "CTX",
	}

	if h.economicEngine == nil {
		response["error"] = "Economic engine not available"
	}

	return h.sendResponse(ctx, event, events.EventTypeWalletData, response, nil)
}

func (h *APIEventHandler) handleWalletTransfer(ctx context.Context, event events.Event) error {
	response := map[string]interface{}{
		"success": false,
		"error":   "Transfer functionality not implemented",
	}

	return h.sendResponse(ctx, event, events.EventTypeWalletData, response, nil)
}

func (h *APIEventHandler) handleGetServices(ctx context.Context, event events.Event) error {
	var services []interface{}
	var err error

	if h.serviceRegistry != nil {
		// Get services from service registry
		// Note: GetAllServices method needs to be implemented
		services = []interface{}{} // Placeholder
	} else {
		err = fmt.Errorf("service registry not available")
	}

	response := map[string]interface{}{
		"services": services,
		"count":    len(services),
	}

	return h.sendResponse(ctx, event, events.EventTypeServicesData, response, err)
}

func (h *APIEventHandler) handleGetConversationHistory(ctx context.Context, event events.Event) error {
	limit := 10 // Default limit
	
	if event.Data != nil {
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			if limitVal, exists := dataMap["limit"]; exists {
				if limitInt, ok := limitVal.(int); ok {
					limit = limitInt
				} else if limitFloat, ok := limitVal.(float64); ok {
					limit = int(limitFloat)
				}
			}
		}
	}
	
	// Get context manager instance (this should be injected via dependency injection)
	contextManager := h.contextManager
	if contextManager == nil {
		response := map[string]interface{}{
			"messages":        []interface{}{},
			"conversation_id": "",
			"total_events":    0,
			"error":          "Context manager not available",
		}
		return h.sendResponse(ctx, event, "conversation_history.response", response, fmt.Errorf("context manager not available"))
	}
	
	// Get recent conversation history
	historyEvents, err := contextManager.GetRecentConversationHistory(ctx, limit)
	if err != nil {
		response := map[string]interface{}{
			"messages":        []interface{}{},
			"conversation_id": "",
			"total_events":    0,
			"error":          fmt.Sprintf("Failed to get conversation history: %v", err),
		}
		return h.sendResponse(ctx, event, "conversation_history.response", response, err)
	}
	
	// Convert ActivityEvent to chat messages
	messages := make([]interface{}, 0, len(historyEvents))
	conversationID := ""
	
	for _, activityEvent := range historyEvents {
		// Extract conversation ID from the first event
		if conversationID == "" && activityEvent.ConversationID != "" {
			conversationID = activityEvent.ConversationID
		}
		
		// Convert to chat message format
		message := map[string]interface{}{
			"id":        activityEvent.ID,
			"role":      "user", // Default to user, could be enhanced based on event type
			"content":   activityEvent.Content,
			"timestamp": activityEvent.Timestamp,
		}
		
		// Determine role based on event type or content
		if activityEvent.Type == "ai_response" || activityEvent.Type == "agent_response" {
			message["role"] = "assistant"
		} else if activityEvent.Type == "user_query" || activityEvent.Type == "user_message" {
			message["role"] = "user"
		}
		
		messages = append(messages, message)
	}
	
	response := map[string]interface{}{
		"messages":        messages,
		"conversation_id": conversationID,
		"total_events":    len(historyEvents),
	}
	
	return h.sendResponse(ctx, event, events.EventTypeConversationHistoryData, response, nil)
}

func (h *APIEventHandler) handleServiceRegister(ctx context.Context, event events.Event) error {
	response := map[string]interface{}{
		"success": false,
		"error":   "Service registration not implemented",
	}

	return h.sendResponse(ctx, event, "service.register.response", response, nil)
}

func (h *APIEventHandler) handleServiceDeregister(ctx context.Context, event events.Event) error {
	response := map[string]interface{}{
		"success": false,
		"error":   "Service deregistration not implemented",
	}

	return h.sendResponse(ctx, event, "service.deregister.response", response, nil)
}

func (h *APIEventHandler) handleGetNodeInfo(ctx context.Context, event events.Event) error {
	nodeInfo := map[string]interface{}{
		"timestamp": time.Now().Unix(),
	}

	if h.p2pNode != nil {
		nodeInfo["node_id"] = h.p2pNode.Host.ID().String()
		nodeInfo["addresses"] = h.p2pNode.Host.Addrs()
		nodeInfo["peer_count"] = len(h.p2pNode.Host.Network().Peers())
		nodeInfo["protocol_version"] = h.p2pNode.Host.ID().String() // Placeholder
	} else {
		nodeInfo["error"] = "P2P node not available"
	}

	return h.sendResponse(ctx, event, events.EventTypeNodeInfoData, nodeInfo, nil)
}

func (h *APIEventHandler) handleGetPeers(ctx context.Context, event events.Event) error {
	var peers []interface{}

	if h.p2pNode != nil {
		peerIDs := h.p2pNode.Host.Network().Peers()
		peers = make([]interface{}, 0, len(peerIDs))

		for _, peerID := range peerIDs {
			peerInfo := map[string]interface{}{
				"id": peerID.String(),
			}

			if conns := h.p2pNode.Host.Network().ConnsToPeer(peerID); len(conns) > 0 {
				peerInfo["address"] = conns[0].RemoteMultiaddr().String()
				peerInfo["connected"] = true
			} else {
				peerInfo["connected"] = false
			}

			peers = append(peers, peerInfo)
		}
	}

	response := map[string]interface{}{
		"peers": peers,
		"count": len(peers),
	}

	return h.sendResponse(ctx, event, events.EventTypePeersData, response, nil)
}

// Helper method to send responses
func (h *APIEventHandler) sendResponse(ctx context.Context, originalEvent events.Event, responseType string, data interface{}, err error) error {
	responseData := data
	if err != nil {
		responseData = map[string]interface{}{
			"error":   err.Error(),
			"success": false,
		}
	}

	responseEvent := events.NewEventWithCorrelation(
		responseType,
		"api_handler",
		responseData,
		originalEvent.CorrelationID,
	)

	// Copy metadata from original event to preserve connection_id and other context
	if originalEvent.Metadata != nil {
		for key, value := range originalEvent.Metadata {
			responseEvent = responseEvent.WithMetadata(key, value)
		}
	}

	return h.eventRouter.RouteResponse(ctx, responseEvent)
}
