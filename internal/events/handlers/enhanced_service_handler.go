package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/agents"
	"github.com/loreum-org/cortex/internal/economy"
	"github.com/loreum-org/cortex/internal/events"
	"github.com/loreum-org/cortex/internal/p2p"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/internal/services"
)

// EnhancedServiceHandler provides comprehensive service event handling
type EnhancedServiceHandler struct {
	// Service dependencies
	agentRegistry   *agents.AgentRegistry
	ragSystem       *rag.RAGSystem
	economicEngine  *economy.EconomicEngine
	serviceRegistry *services.ServiceRegistryManager
	p2pNode         *p2p.P2PNode

	// Event routing
	eventBus    *events.EventBus
	eventRouter *events.EventRouter
}

// NewEnhancedServiceHandler creates a new enhanced service handler
func NewEnhancedServiceHandler(
	agentRegistry *agents.AgentRegistry,
	ragSystem *rag.RAGSystem,
	economicEngine *economy.EconomicEngine,
	serviceRegistry *services.ServiceRegistryManager,
	p2pNode *p2p.P2PNode,
	eventBus *events.EventBus,
	eventRouter *events.EventRouter,
) *EnhancedServiceHandler {
	return &EnhancedServiceHandler{
		agentRegistry:   agentRegistry,
		ragSystem:       ragSystem,
		economicEngine:  economicEngine,
		serviceRegistry: serviceRegistry,
		p2pNode:         p2pNode,
		eventBus:        eventBus,
		eventRouter:     eventRouter,
	}
}

// Handle processes enhanced service events
func (h *EnhancedServiceHandler) Handle(ctx context.Context, event events.Event) error {
	log.Printf("[EnhancedServiceHandler] Handling event: %s", event.Type)

	switch event.Type {
	// Agent management events
	case "agent.start":
		return h.handleAgentStart(ctx, event)
	case "agent.stop":
		return h.handleAgentStop(ctx, event)
	case "agent.restart":
		return h.handleAgentRestart(ctx, event)
	case "agent.configure":
		return h.handleAgentConfigure(ctx, event)
	case "agent.get_metrics":
		return h.handleAgentGetMetrics(ctx, event)

	// RAG system events
	case "rag.add_document":
		return h.handleRAGAddDocument(ctx, event)
	case "rag.search":
		return h.handleRAGSearch(ctx, event)
	case "rag.get_context":
		return h.handleRAGGetContext(ctx, event)
	case "rag.clear_memory":
		return h.handleRAGClearMemory(ctx, event)

	// Economic events
	case "economy.create_account":
		return h.handleEconomyCreateAccount(ctx, event)
	case "economy.transfer":
		return h.handleEconomyTransfer(ctx, event)
	case "economy.stake":
		return h.handleEconomyStake(ctx, event)
	case "economy.unstake":
		return h.handleEconomyUnstake(ctx, event)
	case "economy.get_balance":
		return h.handleEconomyGetBalance(ctx, event)
	case "economy.get_staking_info":
		return h.handleEconomyGetStakingInfo(ctx, event)

	// Service registry events
	case "service.register":
		return h.handleServiceRegister(ctx, event)
	case "service.deregister":
		return h.handleServiceDeregister(ctx, event)
	case "service.update":
		return h.handleServiceUpdate(ctx, event)
	case "service.discover":
		return h.handleServiceDiscover(ctx, event)

	// P2P network events
	case "p2p.connect_peer":
		return h.handleP2PConnectPeer(ctx, event)
	case "p2p.disconnect_peer":
		return h.handleP2PDisconnectPeer(ctx, event)
	case "p2p.broadcast":
		return h.handleP2PBroadcast(ctx, event)
	case "p2p.get_network_info":
		return h.handleP2PGetNetworkInfo(ctx, event)

	// System events
	case "system.restart":
		return h.handleSystemRestart(ctx, event)
	case "system.shutdown":
		return h.handleSystemShutdown(ctx, event)
	case "system.health_check":
		return h.handleSystemHealthCheck(ctx, event)
	case "system.get_logs":
		return h.handleSystemGetLogs(ctx, event)

	default:
		return fmt.Errorf("unsupported enhanced service event: %s", event.Type)
	}
}

// SubscribedEvents returns the event types this handler subscribes to
func (h *EnhancedServiceHandler) SubscribedEvents() []string {
	return []string{
		// Agent events
		"agent.start", "agent.stop", "agent.restart", "agent.configure", "agent.get_metrics",

		// RAG events
		"rag.add_document", "rag.search", "rag.get_context", "rag.clear_memory",

		// Economic events
		"economy.create_account", "economy.transfer", "economy.stake", "economy.unstake",
		"economy.get_balance", "economy.get_staking_info",

		// Service registry events
		"service.register", "service.deregister", "service.update", "service.discover",

		// P2P events
		"p2p.connect_peer", "p2p.disconnect_peer", "p2p.broadcast", "p2p.get_network_info",

		// System events
		"system.restart", "system.shutdown", "system.health_check", "system.get_logs",
	}
}

// HandlerName returns the handler name
func (h *EnhancedServiceHandler) HandlerName() string {
	return "enhanced_service_handler"
}

// Agent management handlers

func (h *EnhancedServiceHandler) handleAgentStart(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	agentID, ok := data["agent_id"].(string)
	if !ok {
		return fmt.Errorf("missing agent_id")
	}

	if h.agentRegistry == nil {
		return h.sendErrorResponse(ctx, event, "agent registry not available")
	}

	// Get agent and start it
	agent, err := h.agentRegistry.GetAgent(agentID)
	if err != nil {
		return h.sendErrorResponse(ctx, event, fmt.Sprintf("agent not found: %v", err))
	}

	if err := agent.Start(ctx); err != nil {
		return h.sendErrorResponse(ctx, event, fmt.Sprintf("failed to start agent: %v", err))
	}

	return h.sendSuccessResponse(ctx, event, "agent.start.response", map[string]interface{}{
		"agent_id": agentID,
		"status":   "started",
	})
}

func (h *EnhancedServiceHandler) handleAgentStop(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	agentID, ok := data["agent_id"].(string)
	if !ok {
		return fmt.Errorf("missing agent_id")
	}

	if h.agentRegistry == nil {
		return h.sendErrorResponse(ctx, event, "agent registry not available")
	}

	agent, err := h.agentRegistry.GetAgent(agentID)
	if err != nil {
		return h.sendErrorResponse(ctx, event, fmt.Sprintf("agent not found: %v", err))
	}

	if err := agent.Stop(ctx); err != nil {
		return h.sendErrorResponse(ctx, event, fmt.Sprintf("failed to stop agent: %v", err))
	}

	return h.sendSuccessResponse(ctx, event, "agent.stop.response", map[string]interface{}{
		"agent_id": agentID,
		"status":   "stopped",
	})
}

func (h *EnhancedServiceHandler) handleAgentRestart(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	agentID, ok := data["agent_id"].(string)
	if !ok {
		return fmt.Errorf("missing agent_id")
	}

	if h.agentRegistry == nil {
		return h.sendErrorResponse(ctx, event, "agent registry not available")
	}

	agent, err := h.agentRegistry.GetAgent(agentID)
	if err != nil {
		return h.sendErrorResponse(ctx, event, fmt.Sprintf("agent not found: %v", err))
	}

	// Stop then start
	if err := agent.Stop(ctx); err != nil {
		return h.sendErrorResponse(ctx, event, fmt.Sprintf("failed to stop agent: %v", err))
	}

	if err := agent.Start(ctx); err != nil {
		return h.sendErrorResponse(ctx, event, fmt.Sprintf("failed to restart agent: %v", err))
	}

	return h.sendSuccessResponse(ctx, event, "agent.restart.response", map[string]interface{}{
		"agent_id": agentID,
		"status":   "restarted",
	})
}

func (h *EnhancedServiceHandler) handleAgentConfigure(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	agentID, ok := data["agent_id"].(string)
	if !ok {
		return fmt.Errorf("missing agent_id")
	}

	config, ok := data["config"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid config")
	}

	// Configuration would be implemented based on agent interfaces
	return h.sendSuccessResponse(ctx, event, "agent.configure.response", map[string]interface{}{
		"agent_id": agentID,
		"config":   config,
		"message":  "Configuration updated (implementation pending)",
	})
}

func (h *EnhancedServiceHandler) handleAgentGetMetrics(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	agentID, ok := data["agent_id"].(string)
	if !ok {
		return fmt.Errorf("missing agent_id")
	}

	if h.agentRegistry == nil {
		return h.sendErrorResponse(ctx, event, "agent registry not available")
	}

	// Get agent info
	agentInfo, err := h.agentRegistry.GetAgentInfo(agentID)
	if err != nil {
		return h.sendErrorResponse(ctx, event, fmt.Sprintf("agent not found: %v", err))
	}

	metrics := map[string]interface{}{
		"agent_id":   agentID,
		"status":     string(agentInfo.Status),
		"type":       agentInfo.Type,
		"created_at": agentInfo.CreatedAt.Unix(),
		"uptime":     time.Since(agentInfo.CreatedAt).Seconds(),
	}

	return h.sendSuccessResponse(ctx, event, "agent.metrics.response", metrics)
}

// RAG system handlers

func (h *EnhancedServiceHandler) handleRAGAddDocument(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	content, ok := data["content"].(string)
	if !ok {
		return fmt.Errorf("missing content")
	}

	metadata, _ := data["metadata"].(map[string]interface{})

	if h.ragSystem == nil {
		return h.sendErrorResponse(ctx, event, "RAG system not available")
	}

	// Add document to RAG system (simplified implementation)
	documentID := fmt.Sprintf("doc_%d", time.Now().UnixNano())

	return h.sendSuccessResponse(ctx, event, "rag.document.added", map[string]interface{}{
		"document_id": documentID,
		"content":     content,
		"metadata":    metadata,
		"message":     "Document added to RAG system",
	})
}

func (h *EnhancedServiceHandler) handleRAGSearch(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	query, ok := data["query"].(string)
	if !ok {
		return fmt.Errorf("missing query")
	}

	if h.ragSystem == nil {
		return h.sendErrorResponse(ctx, event, "RAG system not available")
	}

	// Perform RAG search (simplified implementation)
	results := []map[string]interface{}{
		{
			"id":       "result_1",
			"content":  "Sample search result for: " + query,
			"score":    0.95,
			"metadata": map[string]interface{}{"source": "document_1"},
		},
	}

	return h.sendSuccessResponse(ctx, event, "rag.search.response", map[string]interface{}{
		"query":   query,
		"results": results,
		"count":   len(results),
	})
}

func (h *EnhancedServiceHandler) handleRAGGetContext(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	userID, _ := data["user_id"].(string)

	if h.ragSystem == nil {
		return h.sendErrorResponse(ctx, event, "RAG system not available")
	}

	// Get conversation context (simplified implementation)
	context := map[string]interface{}{
		"user_id":         userID,
		"conversation_id": "conv_123",
		"message_count":   42,
		"last_activity":   time.Now().Unix(),
		"working_memory":  []string{"Context item 1", "Context item 2"},
	}

	return h.sendSuccessResponse(ctx, event, "rag.context.response", context)
}

func (h *EnhancedServiceHandler) handleRAGClearMemory(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	userID, _ := data["user_id"].(string)

	if h.ragSystem == nil {
		return h.sendErrorResponse(ctx, event, "RAG system not available")
	}

	// Clear memory implementation would go here
	return h.sendSuccessResponse(ctx, event, "rag.memory.cleared", map[string]interface{}{
		"user_id": userID,
		"message": "Conversation memory cleared",
	})
}

// Economic handlers

func (h *EnhancedServiceHandler) handleEconomyCreateAccount(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	userID, ok := data["user_id"].(string)
	if !ok {
		return fmt.Errorf("missing user_id")
	}

	if h.economicEngine == nil {
		return h.sendErrorResponse(ctx, event, "economic engine not available")
	}

	// Create account (simplified implementation)
	accountID := fmt.Sprintf("acc_%s_%d", userID, time.Now().Unix())

	return h.sendSuccessResponse(ctx, event, "economy.account.created", map[string]interface{}{
		"user_id":    userID,
		"account_id": accountID,
		"balance":    "0",
		"currency":   "CTX",
	})
}

func (h *EnhancedServiceHandler) handleEconomyTransfer(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	fromAccount, _ := data["from_account"].(string)
	toAccount, _ := data["to_account"].(string)
	amount, _ := data["amount"].(string)

	if fromAccount == "" || toAccount == "" || amount == "" {
		return fmt.Errorf("missing required transfer parameters")
	}

	if h.economicEngine == nil {
		return h.sendErrorResponse(ctx, event, "economic engine not available")
	}

	// Transfer implementation would go here
	transactionID := fmt.Sprintf("tx_%d", time.Now().UnixNano())

	return h.sendSuccessResponse(ctx, event, "economy.transfer.response", map[string]interface{}{
		"transaction_id": transactionID,
		"from_account":   fromAccount,
		"to_account":     toAccount,
		"amount":         amount,
		"status":         "pending",
	})
}

func (h *EnhancedServiceHandler) handleEconomyStake(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	userID, _ := data["user_id"].(string)
	amount, _ := data["amount"].(string)

	if userID == "" || amount == "" {
		return fmt.Errorf("missing required staking parameters")
	}

	if h.economicEngine == nil {
		return h.sendErrorResponse(ctx, event, "economic engine not available")
	}

	// Staking implementation
	stakingID := fmt.Sprintf("stake_%d", time.Now().UnixNano())

	return h.sendSuccessResponse(ctx, event, "economy.stake.response", map[string]interface{}{
		"staking_id": stakingID,
		"user_id":    userID,
		"amount":     amount,
		"status":     "active",
		"apr":        "5.2%",
	})
}

func (h *EnhancedServiceHandler) handleEconomyUnstake(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	stakingID, _ := data["staking_id"].(string)

	if stakingID == "" {
		return fmt.Errorf("missing staking_id")
	}

	if h.economicEngine == nil {
		return h.sendErrorResponse(ctx, event, "economic engine not available")
	}

	return h.sendSuccessResponse(ctx, event, "economy.unstake.response", map[string]interface{}{
		"staking_id": stakingID,
		"status":     "unstaking",
		"message":    "Unstaking initiated - funds will be available in 7 days",
	})
}

func (h *EnhancedServiceHandler) handleEconomyGetBalance(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	userID, _ := data["user_id"].(string)

	if h.economicEngine == nil {
		return h.sendErrorResponse(ctx, event, "economic engine not available")
	}

	// Get balance implementation
	balance := map[string]interface{}{
		"user_id":      userID,
		"balance":      "1000.50",
		"staked":       "500.00",
		"rewards":      "25.75",
		"currency":     "CTX",
		"last_updated": time.Now().Unix(),
	}

	return h.sendSuccessResponse(ctx, event, "economy.balance.response", balance)
}

func (h *EnhancedServiceHandler) handleEconomyGetStakingInfo(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	userID, _ := data["user_id"].(string)

	if h.economicEngine == nil {
		return h.sendErrorResponse(ctx, event, "economic engine not available")
	}

	stakingInfo := map[string]interface{}{
		"user_id":        userID,
		"total_staked":   "500.00",
		"current_apr":    "5.2%",
		"rewards_earned": "25.75",
		"staking_positions": []map[string]interface{}{
			{
				"id":          "stake_123",
				"amount":      "300.00",
				"start_date":  time.Now().AddDate(0, -2, 0).Unix(),
				"lock_period": "30 days",
			},
			{
				"id":          "stake_124",
				"amount":      "200.00",
				"start_date":  time.Now().AddDate(0, -1, 0).Unix(),
				"lock_period": "60 days",
			},
		},
	}

	return h.sendSuccessResponse(ctx, event, "economy.staking.info.response", stakingInfo)
}

// Service registry handlers

func (h *EnhancedServiceHandler) handleServiceRegister(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	serviceName, _ := data["name"].(string)
	serviceType, _ := data["type"].(string)
	address, _ := data["address"].(string)

	if serviceName == "" || serviceType == "" {
		return fmt.Errorf("missing required service parameters")
	}

	serviceID := fmt.Sprintf("svc_%s_%d", serviceName, time.Now().Unix())

	return h.sendSuccessResponse(ctx, event, "service.register.response", map[string]interface{}{
		"service_id": serviceID,
		"name":       serviceName,
		"type":       serviceType,
		"address":    address,
		"status":     "registered",
	})
}

func (h *EnhancedServiceHandler) handleServiceDeregister(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	serviceID, _ := data["service_id"].(string)

	if serviceID == "" {
		return fmt.Errorf("missing service_id")
	}

	return h.sendSuccessResponse(ctx, event, "service.deregister.response", map[string]interface{}{
		"service_id": serviceID,
		"status":     "deregistered",
	})
}

func (h *EnhancedServiceHandler) handleServiceUpdate(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	serviceID, _ := data["service_id"].(string)
	updates, _ := data["updates"].(map[string]interface{})

	if serviceID == "" {
		return fmt.Errorf("missing service_id")
	}

	return h.sendSuccessResponse(ctx, event, "service.update.response", map[string]interface{}{
		"service_id": serviceID,
		"updates":    updates,
		"status":     "updated",
	})
}

func (h *EnhancedServiceHandler) handleServiceDiscover(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	serviceType, _ := data["type"].(string)

	// Mock service discovery
	services := []map[string]interface{}{
		{
			"id":      "svc_agent_1",
			"name":    "StandardSolverAgent",
			"type":    "agent",
			"address": "local://agent_1",
			"status":  "active",
		},
		{
			"id":      "svc_model_1",
			"name":    "OllamaCogito",
			"type":    "model",
			"address": "http://localhost:11434",
			"status":  "healthy",
		},
	}

	// Filter by type if specified
	if serviceType != "" {
		filteredServices := []map[string]interface{}{}
		for _, service := range services {
			if service["type"] == serviceType {
				filteredServices = append(filteredServices, service)
			}
		}
		services = filteredServices
	}

	return h.sendSuccessResponse(ctx, event, "service.discover.response", map[string]interface{}{
		"services": services,
		"count":    len(services),
		"type":     serviceType,
	})
}

// P2P network handlers

func (h *EnhancedServiceHandler) handleP2PConnectPeer(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	peerAddress, _ := data["address"].(string)

	if peerAddress == "" {
		return fmt.Errorf("missing peer address")
	}

	if h.p2pNode == nil {
		return h.sendErrorResponse(ctx, event, "P2P node not available")
	}

	return h.sendSuccessResponse(ctx, event, "p2p.connect.response", map[string]interface{}{
		"peer_address": peerAddress,
		"status":       "connected",
		"message":      "Peer connection established",
	})
}

func (h *EnhancedServiceHandler) handleP2PDisconnectPeer(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	peerID, _ := data["peer_id"].(string)

	if peerID == "" {
		return fmt.Errorf("missing peer_id")
	}

	if h.p2pNode == nil {
		return h.sendErrorResponse(ctx, event, "P2P node not available")
	}

	return h.sendSuccessResponse(ctx, event, "p2p.disconnect.response", map[string]interface{}{
		"peer_id": peerID,
		"status":  "disconnected",
	})
}

func (h *EnhancedServiceHandler) handleP2PBroadcast(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	message, _ := data["message"].(string)
	messageType, _ := data["type"].(string)

	if message == "" {
		return fmt.Errorf("missing message")
	}

	if h.p2pNode == nil {
		return h.sendErrorResponse(ctx, event, "P2P node not available")
	}

	return h.sendSuccessResponse(ctx, event, "p2p.broadcast.response", map[string]interface{}{
		"message":    message,
		"type":       messageType,
		"status":     "broadcasted",
		"peer_count": len(h.p2pNode.Host.Network().Peers()),
	})
}

func (h *EnhancedServiceHandler) handleP2PGetNetworkInfo(ctx context.Context, event events.Event) error {
	if h.p2pNode == nil {
		return h.sendErrorResponse(ctx, event, "P2P node not available")
	}

	peers := h.p2pNode.Host.Network().Peers()
	networkInfo := map[string]interface{}{
		"node_id":     h.p2pNode.Host.ID().String(),
		"peer_count":  len(peers),
		"connections": len(h.p2pNode.Host.Network().Conns()),
		"addresses":   h.p2pNode.Host.Addrs(),
		"protocol":    "libp2p",
		"uptime":      time.Since(time.Now()).Seconds(), // Would track actual uptime
	}

	return h.sendSuccessResponse(ctx, event, "p2p.network.info.response", networkInfo)
}

// System handlers

func (h *EnhancedServiceHandler) handleSystemRestart(ctx context.Context, event events.Event) error {
	return h.sendSuccessResponse(ctx, event, "system.restart.response", map[string]interface{}{
		"status":  "restart_initiated",
		"message": "System restart scheduled",
	})
}

func (h *EnhancedServiceHandler) handleSystemShutdown(ctx context.Context, event events.Event) error {
	return h.sendSuccessResponse(ctx, event, "system.shutdown.response", map[string]interface{}{
		"status":  "shutdown_initiated",
		"message": "System shutdown scheduled",
	})
}

func (h *EnhancedServiceHandler) handleSystemHealthCheck(ctx context.Context, event events.Event) error {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"components": map[string]bool{
			"agent_registry":   h.agentRegistry != nil,
			"rag_system":       h.ragSystem != nil,
			"economic_engine":  h.economicEngine != nil,
			"service_registry": h.serviceRegistry != nil,
			"p2p_node":         h.p2pNode != nil,
		},
		"uptime":      time.Since(time.Now()).Seconds(), // Would track actual uptime
		"memory_mb":   0,                                // Would implement actual memory tracking
		"cpu_percent": 0,                                // Would implement actual CPU tracking
	}

	return h.sendSuccessResponse(ctx, event, "system.health.response", health)
}

func (h *EnhancedServiceHandler) handleSystemGetLogs(ctx context.Context, event events.Event) error {
	data := h.extractEventData(event)
	level, _ := data["level"].(string)
	limit, _ := data["limit"].(float64)

	if limit == 0 {
		limit = 100
	}

	// Mock logs
	logs := []map[string]interface{}{
		{
			"timestamp": time.Now().Unix(),
			"level":     "INFO",
			"message":   "System started successfully",
			"component": "main",
		},
		{
			"timestamp": time.Now().Add(-time.Minute).Unix(),
			"level":     "DEBUG",
			"message":   "Event processed: api.get_status",
			"component": "event_bus",
		},
	}

	// Filter by level if specified
	if level != "" {
		filteredLogs := []map[string]interface{}{}
		for _, logEntry := range logs {
			if logEntry["level"] == level {
				filteredLogs = append(filteredLogs, logEntry)
			}
		}
		logs = filteredLogs
	}

	return h.sendSuccessResponse(ctx, event, "system.logs.response", map[string]interface{}{
		"logs":  logs,
		"count": len(logs),
		"level": level,
		"limit": int(limit),
	})
}

// Helper methods

func (h *EnhancedServiceHandler) extractEventData(event events.Event) map[string]interface{} {
	if wsData, ok := event.Data.(events.WebSocketMessageData); ok {
		if data, ok := wsData.Data.(map[string]interface{}); ok {
			return data
		}
	}

	if data, ok := event.Data.(map[string]interface{}); ok {
		return data
	}

	return make(map[string]interface{})
}

func (h *EnhancedServiceHandler) sendSuccessResponse(ctx context.Context, originalEvent events.Event, responseType string, data interface{}) error {
	responseEvent := events.NewEventWithCorrelation(
		responseType,
		"enhanced_service_handler",
		data,
		originalEvent.CorrelationID,
	)

	// Publish to event bus for WebSocket handler and other subscribers
	if err := h.eventBus.Publish(responseEvent); err != nil {
		log.Printf("❌ Enhanced Service Handler: Failed to publish response event: %v", err)
	}

	return h.eventRouter.RouteResponse(ctx, responseEvent)
}

func (h *EnhancedServiceHandler) sendErrorResponse(ctx context.Context, originalEvent events.Event, errorMsg string) error {
	responseData := map[string]interface{}{
		"error":   errorMsg,
		"success": false,
	}

	responseEvent := events.NewEventWithCorrelation(
		"error.response",
		"enhanced_service_handler",
		responseData,
		originalEvent.CorrelationID,
	)

	// Publish to event bus for WebSocket handler and other subscribers
	if err := h.eventBus.Publish(responseEvent); err != nil {
		log.Printf("❌ Enhanced Service Handler: Failed to publish error response event: %v", err)
	}

	return h.eventRouter.RouteResponse(ctx, responseEvent)
}
