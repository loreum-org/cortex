package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/loreum-org/cortex/internal/agents"
	"github.com/loreum-org/cortex/internal/consensus"
	"github.com/loreum-org/cortex/internal/economy" // Import the economy package
	"github.com/loreum-org/cortex/internal/p2p"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/internal/services"
	"github.com/loreum-org/cortex/pkg/types"
)

// ServerMetrics tracks various metrics for the server
type ServerMetrics struct {
	StartTime time.Time
	mu        sync.RWMutex

	// Network metrics
	TotalBytesReceived int64
	TotalBytesSent     int64
	ConnectionsOpened  int64
	ConnectionsClosed  int64

	// Query metrics
	QueriesProcessed int64
	QueryLatencies   []time.Duration
	QuerySuccesses   int64
	QueryFailures    int64

	// RAG metrics
	DocumentsAdded   int64
	RAGQueriesTotal  int64
	RAGQueryLatency  []time.Duration
	RAGQueryFailures int64

	// Events for real-time monitoring
	Events []Event
}

// Event represents a system event for monitoring
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Server represents the API server
type Server struct {
	Router            *mux.Router
	P2PNode           *p2p.P2PNode
	ConsensusService  *consensus.ConsensusService
	SolverAgent       *agents.SolverAgent
	RAGSystem         *rag.RAGSystem
	EconomicEngine    *economy.EconomicEngine          // Add EconomicEngine field
	EconomyService    *EconomyService                  // Add EconomyService for API handlers
	EconomicQueryProc *EconomicQueryProcessor          // Enhanced economic query processor
	ServiceRegistry   *services.ServiceRegistryManager // Service registry manager
	ServiceAPI        *ServiceAPI                      // Service API handler
	WebSocketManager  *WebSocketManager                // WebSocket manager for real-time communication
	Broadcaster       *RealtimeBroadcaster             // Real-time event broadcaster
	httpServer        *http.Server
	Metrics           *ServerMetrics
}

// NewServer creates a new API server
func NewServer(p2pNode *p2p.P2PNode, consensusService *consensus.ConsensusService, solverAgent *agents.SolverAgent, ragSystem *rag.RAGSystem, economicEngine *economy.EconomicEngine, serviceRegistry *services.ServiceRegistryManager) *Server {
	s := &Server{
		Router:           mux.NewRouter(),
		P2PNode:          p2pNode,
		ConsensusService: consensusService,
		SolverAgent:      solverAgent,
		RAGSystem:        ragSystem,
		EconomicEngine:   economicEngine,  // Initialize EconomicEngine
		ServiceRegistry:  serviceRegistry, // Initialize ServiceRegistry
		Metrics: &ServerMetrics{
			StartTime:       time.Now(),
			QueryLatencies:  make([]time.Duration, 0, 100),
			RAGQueryLatency: make([]time.Duration, 0, 100),
			Events:          make([]Event, 0, 100),
		},
	}
	s.EconomyService = NewEconomyService(economicEngine) // Initialize EconomyService

	// Initialize enhanced economic query processor
	s.EconomicQueryProc = &EconomicQueryProcessor{
		EconomicEngine: economicEngine,
		SolverAgent:    solverAgent,
		RAGSystem:      ragSystem,
		P2PNode:        p2pNode,
		Metrics:        s.Metrics,
	}

	// Initialize service API if service registry is available
	if serviceRegistry != nil {
		s.ServiceAPI = NewServiceAPI(serviceRegistry)
	}

	// Initialize WebSocket manager
	s.WebSocketManager = NewWebSocketManager(s)

	// Initialize real-time broadcaster
	s.Broadcaster = NewRealtimeBroadcaster(s.WebSocketManager, s)

	s.setupRoutes()

	// Initialize economic system with default accounts for demo
	s.initializeEconomicSystem()

	return s
}

// setupRoutes sets up the routes for the API server
func (s *Server) setupRoutes() {
	// Add CORS middleware
	s.Router.Use(s.corsMiddleware)

	// WebSocket endpoint
	s.Router.HandleFunc("/ws", s.websocketHandler).Methods("GET")

	// Health check
	s.Router.HandleFunc("/health", s.healthHandler).Methods("GET", "OPTIONS")

	// Node info
	s.Router.HandleFunc("/node/info", s.nodeInfoHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/node/peers", s.peersHandler).Methods("GET", "OPTIONS")

	// Transactions
	s.Router.HandleFunc("/transactions", s.listTransactionsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/transactions", s.createTransactionHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/transactions/{id}", s.getTransactionHandler).Methods("GET", "OPTIONS")

	// Queries
	s.Router.HandleFunc("/queries", s.submitQueryHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/queries/{id}", s.getQueryResultHandler).Methods("GET", "OPTIONS")

	// Enhanced economic queries
	s.Router.HandleFunc("/queries/economic", s.economicQueryHandler).Methods("POST", "OPTIONS")

	// RAG
	s.Router.HandleFunc("/rag/documents", s.addDocumentHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/rag/query", s.ragQueryHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/rag/ollama/status", s.embeddedOllamaStatusHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/rag/ollama/restart", s.embeddedOllamaRestartHandler).Methods("POST", "OPTIONS")

	// Metrics endpoints
	s.Router.HandleFunc("/metrics/network", s.networkMetricsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/metrics/queries", s.queryMetricsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/metrics/system", s.systemMetricsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/metrics/rag", s.ragMetricsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/metrics/economy", s.economyMetricsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/metrics/websockets", s.websocketMetricsHandler).Methods("GET", "OPTIONS")

	// Events endpoint
	s.Router.HandleFunc("/events", s.eventsHandler).Methods("GET", "OPTIONS")

	// Wallet endpoints
	s.Router.HandleFunc("/wallet/balance/{account}", s.walletBalanceHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/wallet/account/{account}", s.walletAccountHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/wallet/transfer", s.walletTransferHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/wallet/transactions/{account}", s.walletTransactionsHandler).Methods("GET", "OPTIONS")

	// Reputation endpoints
	s.Router.HandleFunc("/reputation/scores", s.reputationScoresHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/reputation/score/{node_id}", s.nodeReputationHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/consensus/status", s.consensusStatusHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/consensus/conflicts", s.consensusConflictsHandler).Methods("GET", "OPTIONS")

	// DAG Explorer endpoints
	s.Router.HandleFunc("/explorer/dag", s.dagOverviewHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/explorer/dag/nodes", s.dagNodesHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/explorer/dag/node/{id}", s.dagNodeHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/explorer/transactions", s.explorerTransactionsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/explorer/transaction/{id}", s.explorerTransactionHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/explorer/search", s.explorerSearchHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/explorer/stats", s.explorerStatsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/explorer/dag/topology", s.dagTopologyHandler).Methods("GET", "OPTIONS")

	// Staking endpoints
	s.Router.HandleFunc("/staking/stake", s.stakeToNodeHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/staking/withdraw/request", s.requestWithdrawalHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/staking/withdraw/execute", s.executeWithdrawalHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/staking/user/{account}", s.getUserStakesHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/staking/node/{node_id}", s.getNodeStakeHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/staking/nodes", s.getAllNodeStakesHandler).Methods("GET", "OPTIONS")

	// Register economic routes
	if s.EconomyService != nil {
		s.EconomyService.RegisterRoutes(s.Router)
	}

	// Context management endpoints
	s.Router.HandleFunc("/context/recent", s.recentContextHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/context/conversation/new", s.newConversationHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/context/conversation/summary", s.conversationSummaryHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/context/activity", s.trackActivityHandler).Methods("POST", "OPTIONS")

	// AGI System endpoints
	s.Router.HandleFunc("/agi/state", s.agiStateHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/agi/intelligence", s.agiIntelligenceHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/agi/domains", s.agiDomainsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/agi/evolution", s.agiEvolutionHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/agi/prompt", s.agiPromptHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/agi/cortex/history", s.agiCortexHistoryHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/agi/context/windows", s.agiContextWindowsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/agi/vector/search", s.agiVectorSearchHandler).Methods("POST", "OPTIONS")
	s.Router.HandleFunc("/agi/network/snapshot", s.agiNetworkSnapshotHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/agi/network/nodes", s.agiNetworkNodesHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/agi/network/leaders", s.agiDomainLeadersHandler).Methods("GET", "OPTIONS")

	// Consciousness runtime endpoints
	s.Router.HandleFunc("/agi/consciousness/state", s.consciousnessStateHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/agi/consciousness/metrics", s.consciousnessMetricsHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/agi/consciousness/memory", s.consciousnessMemoryHandler).Methods("GET", "OPTIONS")

	// Conversation context endpoints
	s.Router.HandleFunc("/conversation/history", s.conversationHistoryHandler).Methods("GET", "OPTIONS")
	s.Router.HandleFunc("/conversation/context", s.conversationContextHandler).Methods("GET", "OPTIONS")

	// Register service registry routes
	if s.ServiceAPI != nil {
		s.ServiceAPI.RegisterRoutes(s.Router)
	}
}

// corsMiddleware adds CORS headers to all responses
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start starts the API server
func (s *Server) Start(addr string) error {
	// Start the real-time broadcaster
	if s.Broadcaster != nil {
		s.Broadcaster.Start()
	}

	s.httpServer = &http.Server{
		Handler:      s.Router,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("API server listening on %s\n", addr)
	return s.httpServer.ListenAndServe()
}

// Stop stops the API server
func (s *Server) Stop(ctx context.Context) error {
	// Stop the real-time broadcaster
	if s.Broadcaster != nil {
		s.Broadcaster.Stop()
	}

	return s.httpServer.Shutdown(ctx)
}

// AddEvent adds an event to the event log
func (s *Server) AddEvent(eventType string, data map[string]interface{}) {
	s.Metrics.mu.Lock()
	defer s.Metrics.mu.Unlock()

	event := Event{
		ID:        generateID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	// Add to beginning of slice for most recent first
	s.Metrics.Events = append([]Event{event}, s.Metrics.Events...)

	// Limit event log size
	if len(s.Metrics.Events) > 100 {
		s.Metrics.Events = s.Metrics.Events[:100]
	}
}

// networkMetricsHandler handles network metrics requests
func (s *Server) networkMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.Metrics.mu.RLock()
	defer s.Metrics.mu.RUnlock()

	// Get peer count
	peerCount := len(s.P2PNode.Host.Network().Peers())

	metrics := map[string]interface{}{
		"peer_count":         peerCount,
		"bytes_received":     s.Metrics.TotalBytesReceived,
		"bytes_sent":         s.Metrics.TotalBytesSent,
		"connections_opened": s.Metrics.ConnectionsOpened,
		"connections_closed": s.Metrics.ConnectionsClosed,
		"protocols":          s.P2PNode.Host.Mux().Protocols(),
		"listen_addresses":   s.P2PNode.Host.Addrs(),
	}

	json.NewEncoder(w).Encode(metrics)
}

// queryMetricsHandler handles query metrics requests
func (s *Server) queryMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.Metrics.mu.RLock()
	defer s.Metrics.mu.RUnlock()

	// Calculate average latency
	var avgLatency time.Duration
	if len(s.Metrics.QueryLatencies) > 0 {
		var total time.Duration
		for _, latency := range s.Metrics.QueryLatencies {
			total += latency
		}
		avgLatency = total / time.Duration(len(s.Metrics.QueryLatencies))
	}

	// Calculate success rate
	var successRate float64
	if s.Metrics.QueriesProcessed > 0 {
		successRate = float64(s.Metrics.QuerySuccesses) / float64(s.Metrics.QueriesProcessed) * 100
	}

	metrics := map[string]interface{}{
		"queries_processed": s.Metrics.QueriesProcessed,
		"query_successes":   s.Metrics.QuerySuccesses,
		"query_failures":    s.Metrics.QueryFailures,
		"avg_latency_ms":    avgLatency.Milliseconds(),
		"success_rate":      successRate,
	}

	json.NewEncoder(w).Encode(metrics)
}

// systemMetricsHandler handles system metrics requests
func (s *Server) systemMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptime := time.Since(s.Metrics.StartTime)

	metrics := map[string]interface{}{
		"uptime_seconds":   uptime.Seconds(),
		"goroutines":       runtime.NumGoroutine(),
		"memory_allocated": memStats.Alloc,
		"memory_total":     memStats.TotalAlloc,
		"memory_sys":       memStats.Sys,
		"gc_cycles":        memStats.NumGC,
		"cpu_cores":        runtime.NumCPU(),
		"start_time":       s.Metrics.StartTime,
	}

	json.NewEncoder(w).Encode(metrics)
}

// ragMetricsHandler handles RAG metrics requests
func (s *Server) ragMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.Metrics.mu.RLock()
	defer s.Metrics.mu.RUnlock()

	// Calculate average latency
	var avgLatency time.Duration
	if len(s.Metrics.RAGQueryLatency) > 0 {
		var total time.Duration
		for _, latency := range s.Metrics.RAGQueryLatency {
			total += latency
		}
		avgLatency = total / time.Duration(len(s.Metrics.RAGQueryLatency))
	}

	// Get document count
	stats := s.RAGSystem.VectorDB.GetStats()
	documentCount := stats.Size

	metrics := map[string]interface{}{
		"document_count":     documentCount,
		"documents_added":    s.Metrics.DocumentsAdded,
		"rag_queries_total":  s.Metrics.RAGQueriesTotal,
		"rag_query_failures": s.Metrics.RAGQueryFailures,
		"avg_latency_ms":     avgLatency.Milliseconds(),
	}

	json.NewEncoder(w).Encode(metrics)
}

// economyMetricsHandler handles economy metrics requests
func (s *Server) economyMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.EconomicEngine == nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "economic engine not available"})
		return
	}

	// Get economic network stats
	stats := s.EconomicEngine.GetNetworkStats()
	json.NewEncoder(w).Encode(stats)
}

// eventsHandler handles events requests
func (s *Server) eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.Metrics.mu.RLock()
	defer s.Metrics.mu.RUnlock()

	// Get limit parameter, default to 10
	limit := 10
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := parseInt(limitParam, 1, 100); err == nil {
			limit = parsedLimit
		}
	}

	// Return up to limit events
	events := s.Metrics.Events
	if len(events) > limit {
		events = events[:limit]
	}

	json.NewEncoder(w).Encode(events)
}

// parseInt parses a string to an integer with bounds checking
func parseInt(s string, min, max int) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return 0, err
	}
	if n < min {
		n = min
	}
	if n > max {
		n = max
	}
	return n, nil
}

// healthHandler handles health check requests
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// nodeInfoHandler handles node info requests
func (s *Server) nodeInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := map[string]interface{}{
		"id":         s.P2PNode.Host.ID().String(),
		"addresses":  s.P2PNode.Host.Addrs(),
		"created_at": time.Now(),
	}

	json.NewEncoder(w).Encode(info)
}

// peersHandler handles peers requests
func (s *Server) peersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	peers := s.P2PNode.Host.Network().Peers()
	peerInfos := make([]map[string]interface{}, 0, len(peers))

	for _, peer := range peers {
		protocols, _ := s.P2PNode.Host.Peerstore().GetProtocols(peer)
		peerInfos = append(peerInfos, map[string]interface{}{
			"id":        peer.String(),
			"protocols": protocols,
		})
	}

	json.NewEncoder(w).Encode(peerInfos)
}

// QueryRequest represents a query request
type QueryRequest struct {
	Query    string            `json:"query"`   // Frontend sends "query"
	Text     string            `json:"text"`    // Keep for backwards compatibility
	UseRAG   bool              `json:"use_rag"` // Frontend sends "use_rag"
	Type     string            `json:"type"`
	Metadata map[string]string `json:"metadata"`
	Account  string            `json:"account,omitempty"` // Added for economic integration
}

// submitQueryHandler handles submit query requests
func (s *Server) submitQueryHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Handle both frontend "query" field and backwards compatible "text" field
	queryText := req.Query
	if queryText == "" {
		queryText = req.Text
	}

	// Determine query type based on use_rag flag
	queryType := req.Type
	if queryType == "" {
		if req.UseRAG {
			queryType = "rag"
		} else {
			queryType = "chat"
		}
	}

	query := &types.Query{
		ID:        generateID(),
		Text:      queryText,
		Type:      queryType,
		Metadata:  req.Metadata,
		Timestamp: time.Now().Unix(),
	}

	// Economic integration - Process payment if economic engine is available
	var paymentTx *economy.Transaction
	var userID string

	// Get user ID from request, header, or use a default
	if req.Account != "" {
		userID = req.Account
	} else if userIDHeader := r.Header.Get("X-User-ID"); userIDHeader != "" {
		userID = userIDHeader
	} else {
		userID = "default_user" // Fallback for demo purposes
	}

	// Determine model tier based on query type or metadata
	modelTier := economy.ModelTierStandard
	if req.Type == "code" || req.Type == "programming" {
		modelTier = economy.ModelTierPremium
	} else if req.Type == "simple" || req.Type == "basic" {
		modelTier = economy.ModelTierBasic
	}

	// Determine economic query type
	economicQueryType := economy.QueryTypeCompletion
	if req.Type == "chat" {
		economicQueryType = economy.QueryTypeChat
	} else if req.Type == "code" || req.Type == "programming" {
		economicQueryType = economy.QueryTypeCodeGen
	} else if req.Type == "embedding" {
		economicQueryType = economy.QueryTypeEmbedding
	} else if req.Type == "rag" {
		economicQueryType = economy.QueryTypeRAG
	}

	// Calculate input size
	inputSize := int64(len(queryText))

	// Process payment if economic engine is available
	if s.EconomicEngine != nil {
		// Get model ID (can be specified in metadata or use default)
		modelID := "default"
		if modelIDFromMeta, ok := req.Metadata["model_id"]; ok {
			modelID = modelIDFromMeta
		}

		// Process payment
		var err error
		paymentTx, err = s.EconomicEngine.ProcessQueryPayment(
			r.Context(),
			userID,
			modelID,
			modelTier,
			economicQueryType,
			inputSize,
		)

		if err != nil {
			// If payment fails, still process the query in demo mode
			log.Printf("Payment processing failed: %v. Processing query in demo mode.", err)
			s.AddEvent("payment_failed", map[string]interface{}{
				"account": userID,
				"error":   err.Error(),
			})
		} else {
			// Add payment info to query metadata
			if query.Metadata == nil {
				query.Metadata = make(map[string]string)
			}
			query.Metadata["payment_tx_id"] = paymentTx.ID
			query.Metadata["account"] = userID

			s.AddEvent("payment_processed", map[string]interface{}{
				"account":    userID,
				"amount":     paymentTx.Amount.String(),
				"model_tier": string(modelTier),
			})
		}
	}

	// Broadcast the query to the P2P network
	err := s.P2PNode.BroadcastQuery(query)
	if err != nil {
		log.Printf("Warning: Failed to broadcast query to network: %s", err)
	} else {
		// Add event for successful broadcast
		s.AddEvent("query_broadcast", map[string]interface{}{
			"query_id": query.ID,
			"peers":    len(s.P2PNode.Host.Network().Peers()),
		})
	}

	// Process the query locally
	response, err := s.SolverAgent.Process(r.Context(), query)

	// Update metrics
	s.Metrics.mu.Lock()
	s.Metrics.QueriesProcessed++
	latency := time.Since(startTime)
	s.Metrics.QueryLatencies = append(s.Metrics.QueryLatencies, latency)
	if len(s.Metrics.QueryLatencies) > 100 {
		s.Metrics.QueryLatencies = s.Metrics.QueryLatencies[1:]
	}

	if err != nil {
		s.Metrics.QueryFailures++
		s.Metrics.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)

		// Economic integration - Handle failure
		if paymentTx != nil {
			// In a real implementation, we might refund the payment
			s.AddEvent("query_failed", map[string]interface{}{
				"query_id": query.ID,
				"error":    err.Error(),
			})
		}

		return
	}

	s.Metrics.QuerySuccesses++
	s.Metrics.mu.Unlock()

	// Economic integration - Distribute reward if payment was processed
	if paymentTx != nil && s.EconomicEngine != nil {
		// Get node ID (this node processed the query)
		nodeID := s.P2PNode.Host.ID().String()

		// Calculate quality score (placeholder - in a real system, this would be more sophisticated)
		qualityScore := 1.0

		// Distribute reward
		rewardTx, err := s.EconomicEngine.DistributeQueryReward(
			r.Context(),
			paymentTx.ID,
			nodeID,
			latency.Milliseconds(),
			true, // Success
			qualityScore,
		)

		if err != nil {
			log.Printf("Warning: Failed to distribute reward: %v", err)
		} else {
			s.AddEvent("reward_distributed", map[string]interface{}{
				"node_id":    nodeID,
				"amount":     rewardTx.Amount.String(),
				"query_id":   query.ID,
				"latency_ms": latency.Milliseconds(),
			})
		}
	}

	// Add event
	s.AddEvent("query_processed", map[string]interface{}{
		"query_id":   query.ID,
		"query_type": query.Type,
		"latency_ms": latency.Milliseconds(),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getQueryResultHandler handles get query result requests
func (s *Server) getQueryResultHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// In a real implementation, this would retrieve the query result from a database
	// For now, just return a dummy response
	response := &types.Response{
		QueryID:   id,
		Text:      "This is a dummy response for query " + id,
		Data:      nil,
		Metadata:  make(map[string]string),
		Status:    "success",
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// economicQueryHandler handles enhanced economic query requests
func (s *Server) economicQueryHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var req EconomicQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request payload: %v", err), http.StatusBadRequest)
		return
	}

	// Process the economic query
	response, err := s.EconomicQueryProc.ProcessEconomicQuery(r.Context(), &req)
	if err != nil {
		// Return error response with economic context
		errorResponse := &EconomicQueryResponse{
			QueryID:   generateEconomicQueryID(),
			Status:    "error",
			Timestamp: time.Now().Unix(),
			Success:   false,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":    err.Error(),
			"response": errorResponse,
		})

		// Add event for failed economic query
		s.AddEvent("economic_query_failed", map[string]interface{}{
			"account":    req.Account,
			"error":      err.Error(),
			"latency_ms": time.Since(startTime).Milliseconds(),
		})
		return
	}

	// Add event for successful economic query
	s.AddEvent("economic_query_processed", map[string]interface{}{
		"query_id":      response.QueryID,
		"account":       req.Account,
		"payment_tx_id": response.PaymentTxID,
		"reward_tx_id":  response.RewardTxID,
		"price":         response.Price,
		"quality_score": response.QualityScore,
		"latency_ms":    response.ProcessTime,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createTransactionHandler handles create transaction requests
func (s *Server) createTransactionHandler(w http.ResponseWriter, r *http.Request) {
	// This is just a placeholder implementation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "not implemented"})
}

// getTransactionHandler handles get transaction requests
func (s *Server) getTransactionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	tx, found := s.ConsensusService.GetTransaction(id)
	if !found {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tx)
}

// listTransactionsHandler handles list transactions requests
func (s *Server) listTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	// This is just a placeholder implementation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]string{})
}

// DocumentRequest represents a document request
type DocumentRequest struct {
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata"`
}

// addDocumentHandler handles add document requests
func (s *Server) addDocumentHandler(w http.ResponseWriter, r *http.Request) {
	var req DocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := s.RAGSystem.AddDocument(r.Context(), req.Text, req.Metadata)

	// Update metrics
	s.Metrics.mu.Lock()
	s.Metrics.DocumentsAdded++
	s.Metrics.mu.Unlock()

	// Add event
	s.AddEvent("document_added", map[string]interface{}{
		"document_length": len(req.Text),
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// RAGQueryRequest represents a RAG query request
type RAGQueryRequest struct {
	Text  string `json:"text"`
	Query string `json:"query"`
}

// ragQueryHandler handles RAG query requests with consciousness processing
func (s *Server) ragQueryHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var req RAGQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use either 'query' or 'text' field
	queryText := req.Query
	if queryText == "" {
		queryText = req.Text
	}

	var response string
	var err error

	// Try consciousness processing first if available
	if s.RAGSystem.ContextManager != nil {
		consciousnessRuntime := s.RAGSystem.ContextManager.GetConsciousnessRuntime()
		if consciousnessRuntime != nil {
			log.Printf("Processing query with consciousness runtime: %s", queryText)

			// Create a proper Query object
			query := &types.Query{
				ID:        fmt.Sprintf("api_query_%d", time.Now().UnixNano()),
				Text:      queryText,
				Type:      "user_query",
				Metadata:  map[string]string{"source": "api"},
				Timestamp: time.Now().Unix(),
			}

			resp, err := consciousnessRuntime.ProcessQuery(r.Context(), query)
			if err == nil && resp != nil {
				response = resp.Text
			} else {
				// Fallback to direct RAG query on consciousness error
				response, err = s.RAGSystem.Query(r.Context(), queryText)
			}
		} else {
			// Fallback to direct RAG query
			response, err = s.RAGSystem.Query(r.Context(), queryText)
		}
	} else {
		// Fallback to direct RAG query
		response, err = s.RAGSystem.Query(r.Context(), queryText)
	}

	// Update metrics
	s.Metrics.mu.Lock()
	s.Metrics.RAGQueriesTotal++
	latency := time.Since(startTime)
	s.Metrics.RAGQueryLatency = append(s.Metrics.RAGQueryLatency, latency)
	if len(s.Metrics.RAGQueryLatency) > 100 {
		s.Metrics.RAGQueryLatency = s.Metrics.RAGQueryLatency[1:]
	}

	if err != nil {
		s.Metrics.RAGQueryFailures++
		s.Metrics.mu.Unlock()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.Metrics.mu.Unlock()

	// Add event
	s.AddEvent("rag_query", map[string]interface{}{
		"query_length": len(req.Text),
		"latency_ms":   latency.Milliseconds(),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": response})
}

// embeddedOllamaStatusHandler returns the status of embedded Ollama
func (s *Server) embeddedOllamaStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.RAGSystem == nil {
		http.Error(w, "RAG system not available", http.StatusServiceUnavailable)
		return
	}

	status := s.RAGSystem.GetEmbeddedStatus()
	json.NewEncoder(w).Encode(status)
}

// embeddedOllamaRestartHandler restarts the embedded Ollama server
func (s *Server) embeddedOllamaRestartHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.RAGSystem == nil {
		http.Error(w, "RAG system not available", http.StatusServiceUnavailable)
		return
	}

	if err := s.RAGSystem.RestartEmbeddedOllama(r.Context()); err != nil {
		http.Error(w, fmt.Sprintf("Failed to restart embedded Ollama: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Embedded Ollama restart initiated",
	})
}

// websocketMetricsHandler returns WebSocket connection metrics
func (s *Server) websocketMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.WebSocketManager == nil {
		http.Error(w, "WebSocket manager not available", http.StatusServiceUnavailable)
		return
	}

	metrics := map[string]interface{}{
		"active_connections": s.WebSocketManager.GetConnectionCount(),
		"connections":        s.WebSocketManager.GetConnections(),
		"timestamp":          time.Now(),
	}

	json.NewEncoder(w).Encode(metrics)
}

// initializeEconomicSystem sets up default accounts for the economic system
func (s *Server) initializeEconomicSystem() {
	if s.EconomicEngine == nil {
		return
	}

	// Create default user account
	defaultUserID := "default_user"
	_, err := s.EconomicEngine.CreateUserAccount(defaultUserID, "0xDefaultUser")
	if err != nil {
		log.Printf("Note: Default user account may already exist: %v", err)
	} else {
		// Fund default user with tokens for demo
		fundAmount := new(big.Int).Mul(big.NewInt(10000), economy.TokenUnit)
		_, err = s.EconomicEngine.MintTokens(defaultUserID, fundAmount, "Demo funding")
		if err != nil {
			log.Printf("Failed to fund default user: %v", err)
		} else {
			log.Printf("Created and funded default user account with %s tokens", fundAmount.String())
		}
	}

	// Create node account for this node
	nodeID := s.P2PNode.Host.ID().String()
	_, err = s.EconomicEngine.CreateNodeAccount(nodeID)
	if err != nil {
		log.Printf("Note: Node account may already exist: %v", err)
	} else {
		// Fund and stake the node
		nodeFunding := new(big.Int).Mul(big.NewInt(5000), economy.TokenUnit)
		_, err = s.EconomicEngine.MintTokens(nodeID, nodeFunding, "Node initialization")
		if err != nil {
			log.Printf("Failed to fund node: %v", err)
		} else {
			// Stake the node
			_, err = s.EconomicEngine.Stake(nodeID, economy.MinimumStake)
			if err != nil {
				log.Printf("Failed to stake node: %v", err)
			} else {
				log.Printf("Created, funded, and staked node account: %s", nodeID[:12]+"...")
			}
		}
	}
}

// WalletBalanceResponse represents wallet balance response
type WalletBalanceResponse struct {
	Account          string `json:"account"`
	Balance          string `json:"balance"`
	BalanceFormatted string `json:"balance_formatted"`
	TokenSymbol      string `json:"token_symbol"`
}

// WalletAccountResponse represents wallet account response
type WalletAccountResponse struct {
	Account          string `json:"account"`
	Address          string `json:"address"`
	Balance          string `json:"balance"`
	BalanceFormatted string `json:"balance_formatted"`
	TotalSpent       string `json:"total_spent"`
	QueriesSubmitted int64  `json:"queries_submitted"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	TokenSymbol      string `json:"token_symbol"`
}

// WalletTransferRequest represents wallet transfer request
type WalletTransferRequest struct {
	FromAccount string `json:"from_account"`
	ToAccount   string `json:"to_account"`
	Amount      string `json:"amount"`
	Description string `json:"description,omitempty"`
}

// WalletTransferResponse represents wallet transfer response
type WalletTransferResponse struct {
	TransactionID string `json:"transaction_id"`
	FromAccount   string `json:"from_account"`
	ToAccount     string `json:"to_account"`
	Amount        string `json:"amount"`
	Description   string `json:"description"`
	Status        string `json:"status"`
	CreatedAt     string `json:"created_at"`
}

// walletBalanceHandler handles wallet balance requests
func (s *Server) walletBalanceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	userID := vars["account"]

	if s.EconomicEngine == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Economic engine not available"})
		return
	}

	balance, err := s.EconomicEngine.GetBalance(userID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	response := WalletBalanceResponse{
		Account:          userID,
		Balance:          balance.String(),
		BalanceFormatted: formatTokenAmount(balance),
		TokenSymbol:      "LORE",
	}

	json.NewEncoder(w).Encode(response)
}

// walletAccountHandler handles wallet account details requests
func (s *Server) walletAccountHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	userID := vars["account"]

	if s.EconomicEngine == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Economic engine not available"})
		return
	}

	userAccount, err := s.EconomicEngine.GetUserAccount(userID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	response := WalletAccountResponse{
		Account:          userAccount.ID,
		Address:          userAccount.Address,
		Balance:          userAccount.Balance.String(),
		BalanceFormatted: formatTokenAmount(userAccount.Balance),
		TotalSpent:       userAccount.TotalSpent.String(),
		QueriesSubmitted: userAccount.QueriesSubmitted,
		CreatedAt:        userAccount.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        userAccount.UpdatedAt.Format(time.RFC3339),
		TokenSymbol:      "LORE",
	}

	json.NewEncoder(w).Encode(response)
}

// walletTransferHandler handles wallet transfer requests
func (s *Server) walletTransferHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.EconomicEngine == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Economic engine not available"})
		return
	}

	var req WalletTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request format"})
		return
	}

	// Validate required fields
	if req.FromAccount == "" || req.ToAccount == "" || req.Amount == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing required fields"})
		return
	}

	// Parse amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid amount"})
		return
	}

	// Perform transfer
	description := req.Description
	if description == "" {
		description = "Wallet transfer"
	}

	tx, err := s.EconomicEngine.Transfer(req.FromAccount, req.ToAccount, amount, description)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Add event
	s.AddEvent("wallet_transfer", map[string]interface{}{
		"transaction_id": tx.ID,
		"from_account":   req.FromAccount,
		"to_account":     req.ToAccount,
		"amount":         amount.String(),
		"description":    description,
	})

	response := WalletTransferResponse{
		TransactionID: tx.ID,
		FromAccount:   req.FromAccount,
		ToAccount:     req.ToAccount,
		Amount:        amount.String(),
		Description:   description,
		Status:        tx.Status,
		CreatedAt:     tx.Timestamp.Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// walletTransactionsHandler handles wallet transaction history requests
func (s *Server) walletTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	userID := vars["account"]

	if s.EconomicEngine == nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Economic engine not available"})
		return
	}

	// Get limit parameter
	limit := 50
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := parseInt(limitParam, 1, 100); err == nil {
			limit = parsedLimit
		}
	}

	// Get offset parameter
	offset := 0
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if parsedOffset, err := parseInt(offsetParam, 0, 10000); err == nil {
			offset = parsedOffset
		}
	}

	transactions, err := s.EconomicEngine.GetTransactions(userID, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(transactions)
}

// formatTokenAmount formats a big.Int amount to a human-readable string
func formatTokenAmount(amount *big.Int) string {
	if amount == nil {
		return "0 LORE"
	}

	// Convert from smallest unit to LORE tokens
	loreAmount := new(big.Float).SetInt(amount)
	tokenUnit := new(big.Float).SetInt(economy.TokenUnit)
	loreAmount.Quo(loreAmount, tokenUnit)

	// Format with appropriate precision
	formatted := loreAmount.Text('f', 4)

	// Remove trailing zeros
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")

	return formatted + " LORE"
}

// generateID generates a simple ID
func generateID() string {
	return "id_" + time.Now().Format("20060102150405")
}

// ReputationScoreResponse represents a reputation score response
type ReputationScoreResponse struct {
	NodeID     string  `json:"node_id"`
	Score      float64 `json:"score"`
	LastUpdate string  `json:"last_update"`
}

// reputationScoresHandler returns all reputation scores
func (s *Server) reputationScoresHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.ConsensusService == nil || s.ConsensusService.ReputationManager == nil {
		http.Error(w, "Consensus service not available", http.StatusServiceUnavailable)
		return
	}

	// Get all reputation scores
	scores := make(map[string]float64)
	if s.ConsensusService.ReputationManager != nil {
		s.ConsensusService.ReputationManager.Lock()
		for nodeID, score := range s.ConsensusService.ReputationManager.Scores {
			scores[nodeID] = score
		}
		s.ConsensusService.ReputationManager.Unlock()
	}

	// Convert to response format
	var responses []ReputationScoreResponse
	for nodeID, score := range scores {
		responses = append(responses, ReputationScoreResponse{
			NodeID:     nodeID,
			Score:      score,
			LastUpdate: time.Now().Format(time.RFC3339),
		})
	}

	// If no scores, initialize with current node
	if len(responses) == 0 && s.P2PNode != nil {
		responses = append(responses, ReputationScoreResponse{
			NodeID:     s.P2PNode.Host.ID().String(),
			Score:      1.0, // Default reputation
			LastUpdate: time.Now().Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"reputation_scores": responses,
		"total_nodes":       len(responses),
		"timestamp":         time.Now().Format(time.RFC3339),
	})
}

// nodeReputationHandler returns reputation score for a specific node
func (s *Server) nodeReputationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	nodeID := vars["node_id"]

	if s.ConsensusService == nil || s.ConsensusService.ReputationManager == nil {
		http.Error(w, "Consensus service not available", http.StatusServiceUnavailable)
		return
	}

	score := s.ConsensusService.ReputationManager.GetScore(nodeID)

	response := ReputationScoreResponse{
		NodeID:     nodeID,
		Score:      score,
		LastUpdate: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ConsensusStatusResponse represents consensus status
type ConsensusStatusResponse struct {
	TotalTransactions     int                    `json:"total_transactions"`
	FinalizedTransactions int                    `json:"finalized_transactions"`
	PendingTransactions   int                    `json:"pending_transactions"`
	ConflictsDetected     int                    `json:"conflicts_detected"`
	ConflictsResolved     int                    `json:"conflicts_resolved"`
	DAGNodes              int                    `json:"dag_nodes"`
	TopologicalOrder      []string               `json:"topological_order,omitempty"`
	ReputationStats       map[string]interface{} `json:"reputation_stats"`
	Timestamp             string                 `json:"timestamp"`
}

// consensusStatusHandler returns consensus system status
func (s *Server) consensusStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.ConsensusService == nil {
		http.Error(w, "Consensus service not available", http.StatusServiceUnavailable)
		return
	}

	// Count transactions
	allNodes := s.ConsensusService.DAG.GetAllNodes()

	totalTxs := len(allNodes)
	finalizedTxs := 0
	pendingTxs := 0

	for _, node := range allNodes {
		if node.Transaction.Finalized {
			finalizedTxs++
		} else {
			pendingTxs++
		}
	}

	// Get conflict information
	conflictsDetected := 0
	conflictsResolved := 0
	if s.ConsensusService.ConflictResolver != nil {
		conflicts := s.ConsensusService.ConflictResolver.GetConflicts()
		conflictsDetected = len(conflicts)
		for _, conflict := range conflicts {
			if conflict.ResolvedAt != nil {
				conflictsResolved++
			}
		}
	}

	// Get topological order
	var topologicalOrder []string
	if s.ConsensusService.CausalOrderingManager != nil {
		topologicalOrder = s.ConsensusService.CausalOrderingManager.GetTopologicalOrder()
	}

	// Calculate reputation statistics
	reputationStats := make(map[string]interface{})
	if s.ConsensusService.ReputationManager != nil {
		s.ConsensusService.ReputationManager.Lock()
		scores := make([]float64, 0, len(s.ConsensusService.ReputationManager.Scores))
		for _, score := range s.ConsensusService.ReputationManager.Scores {
			scores = append(scores, score)
		}
		s.ConsensusService.ReputationManager.Unlock()

		if len(scores) > 0 {
			var sum float64
			var max, min float64 = scores[0], scores[0]
			for _, score := range scores {
				sum += score
				if score > max {
					max = score
				}
				if score < min {
					min = score
				}
			}
			reputationStats["average"] = sum / float64(len(scores))
			reputationStats["maximum"] = max
			reputationStats["minimum"] = min
			reputationStats["nodes_count"] = len(scores)
		}
	}

	response := ConsensusStatusResponse{
		TotalTransactions:     totalTxs,
		FinalizedTransactions: finalizedTxs,
		PendingTransactions:   pendingTxs,
		ConflictsDetected:     conflictsDetected,
		ConflictsResolved:     conflictsResolved,
		DAGNodes:              totalTxs,
		TopologicalOrder:      topologicalOrder[:min(len(topologicalOrder), 10)], // Limit to first 10
		ReputationStats:       reputationStats,
		Timestamp:             time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ConflictInfo represents conflict information for API
type ConflictInfo struct {
	ID               string  `json:"id"`
	Type             string  `json:"type"`
	TransactionA     string  `json:"transaction_a"`
	TransactionB     string  `json:"transaction_b"`
	Description      string  `json:"description"`
	Severity         float64 `json:"severity"`
	DetectedAt       string  `json:"detected_at"`
	ResolvedAt       *string `json:"resolved_at,omitempty"`
	ResolutionMethod string  `json:"resolution_method,omitempty"`
}

// consensusConflictsHandler returns current and resolved conflicts
func (s *Server) consensusConflictsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.ConsensusService == nil || s.ConsensusService.ConflictResolver == nil {
		http.Error(w, "Consensus service not available", http.StatusServiceUnavailable)
		return
	}

	// Get all conflicts
	conflicts := s.ConsensusService.ConflictResolver.GetConflicts()

	var apiConflicts []ConflictInfo
	var unresolvedCount, resolvedCount int

	for _, conflict := range conflicts {
		var resolvedAt *string
		if conflict.ResolvedAt != nil {
			resolvedAtStr := conflict.ResolvedAt.Format(time.RFC3339)
			resolvedAt = &resolvedAtStr
			resolvedCount++
		} else {
			unresolvedCount++
		}

		apiConflicts = append(apiConflicts, ConflictInfo{
			ID:               conflict.ID,
			Type:             string(conflict.Type),
			TransactionA:     conflict.TransactionA,
			TransactionB:     conflict.TransactionB,
			Description:      conflict.Description,
			Severity:         conflict.Severity,
			DetectedAt:       conflict.DetectedAt.Format(time.RFC3339),
			ResolvedAt:       resolvedAt,
			ResolutionMethod: conflict.ResolutionMethod,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conflicts":        apiConflicts,
		"total_conflicts":  len(apiConflicts),
		"unresolved_count": unresolvedCount,
		"resolved_count":   resolvedCount,
		"timestamp":        time.Now().Format(time.RFC3339),
	})
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Staking handler types
type StakeRequest struct {
	Account string `json:"account"`
	NodeID  string `json:"node_id"`
	Amount  string `json:"amount"`
}

type WithdrawalRequest struct {
	Account string `json:"account"`
	NodeID  string `json:"node_id"`
	Amount  string `json:"amount"`
}

type ExecuteWithdrawalRequest struct {
	Account string `json:"account"`
	NodeID  string `json:"node_id"`
}

// stakeToNodeHandler handles user staking to nodes
func (s *Server) stakeToNodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req StakeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Account == "" || req.NodeID == "" || req.Amount == "" {
		http.Error(w, "Missing required fields: account, node_id, amount", http.StatusBadRequest)
		return
	}

	// Parse amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		http.Error(w, "Invalid amount format", http.StatusBadRequest)
		return
	}

	// Execute staking
	stake, err := s.EconomicEngine.StakeToNode(req.Account, req.NodeID, amount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to stake: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"stake":   stake,
		"message": fmt.Sprintf("Successfully staked %s LORE to node %s", formatTokenAmount(amount), req.NodeID),
	})
}

// requestWithdrawalHandler handles withdrawal requests
func (s *Server) requestWithdrawalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req WithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Account == "" || req.NodeID == "" || req.Amount == "" {
		http.Error(w, "Missing required fields: account, node_id, amount", http.StatusBadRequest)
		return
	}

	// Parse amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		http.Error(w, "Invalid amount format", http.StatusBadRequest)
		return
	}

	// Execute withdrawal request
	err := s.EconomicEngine.RequestStakeWithdrawal(req.Account, req.NodeID, amount)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to request withdrawal: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Withdrawal request submitted for %s LORE from node %s", formatTokenAmount(amount), req.NodeID),
	})
}

// executeWithdrawalHandler executes a withdrawal request
func (s *Server) executeWithdrawalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req ExecuteWithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Account == "" || req.NodeID == "" {
		http.Error(w, "Missing required fields: account, node_id", http.StatusBadRequest)
		return
	}

	// Execute withdrawal
	tx, err := s.EconomicEngine.ExecuteStakeWithdrawal(req.Account, req.NodeID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute withdrawal: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"transaction": tx,
		"message":     fmt.Sprintf("Successfully withdrew %s LORE from node %s", formatTokenAmount(tx.Amount), req.NodeID),
	})
}

// getUserStakesHandler returns all stakes for a user
func (s *Server) getUserStakesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	userID := vars["account"]

	stakes := s.EconomicEngine.GetUserStakes(userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"account": userID,
		"stakes":  stakes,
		"count":   len(stakes),
	})
}

// getNodeStakeHandler returns staking information for a specific node
func (s *Server) getNodeStakeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	nodeID := vars["node_id"]

	stakeInfo := s.EconomicEngine.GetNodeStakeInfo(nodeID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stakeInfo)
}

// getAllNodeStakesHandler returns staking information for all nodes
func (s *Server) getAllNodeStakesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	allStakes := s.EconomicEngine.GetAllNodeStakes()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes":     allStakes,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// DAG Explorer Handlers

// DAGOverview represents the overall DAG state for the explorer
type DAGOverview struct {
	TotalNodes         int                    `json:"total_nodes"`
	FinalizedNodes     int                    `json:"finalized_nodes"`
	PendingNodes       int                    `json:"pending_nodes"`
	TransactionTypes   map[string]int         `json:"transaction_types"`
	EconomicTypes      map[string]int         `json:"economic_types"`
	RecentTransactions []TransactionSummary   `json:"recent_transactions"`
	TopNodes           []string               `json:"top_nodes"`
	DAGHeight          int                    `json:"dag_height"`
	Timestamp          string                 `json:"timestamp"`
	NetworkStats       map[string]interface{} `json:"network_stats"`
}

// TransactionSummary represents a summary of a transaction for lists
type TransactionSummary struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Finalized    bool    `json:"finalized"`
	Timestamp    int64   `json:"timestamp"`
	ParentCount  int     `json:"parent_count"`
	ChildCount   int     `json:"child_count"`
	Score        float64 `json:"score"`
	EconomicType string  `json:"economic_type,omitempty"`
	Amount       string  `json:"amount,omitempty"`
	FromAccount  string  `json:"from_account,omitempty"`
	ToAccount    string  `json:"to_account,omitempty"`
}

// DAGNodeDetail represents detailed information about a DAG node
type DAGNodeDetail struct {
	Node             *TransactionSummary  `json:"node"`
	Parents          []TransactionSummary `json:"parents"`
	Children         []TransactionSummary `json:"children"`
	Dependencies     []string             `json:"dependencies"`
	Dependents       []string             `json:"dependents"`
	Score            float64              `json:"score"`
	Path             []string             `json:"path"`
	Depth            int                  `json:"depth"`
	Finalized        bool                 `json:"finalized"`
	FinalizationPath []string             `json:"finalization_path"`
}

// DAGTopology represents the DAG structure for visualization
type DAGTopology struct {
	Nodes []DAGTopologyNode `json:"nodes"`
	Edges []DAGTopologyEdge `json:"edges"`
	Stats DAGTopologyStats  `json:"stats"`
}

// DAGTopologyNode represents a node in the topology
type DAGTopologyNode struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Finalized bool                   `json:"finalized"`
	Score     float64                `json:"score"`
	Timestamp int64                  `json:"timestamp"`
	Level     int                    `json:"level"`
	Position  map[string]interface{} `json:"position"`
	Data      map[string]interface{} `json:"data"`
}

// DAGTopologyEdge represents an edge in the topology
type DAGTopologyEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// DAGTopologyStats represents statistics about the topology
type DAGTopologyStats struct {
	TotalNodes   int      `json:"total_nodes"`
	TotalEdges   int      `json:"total_edges"`
	MaxDepth     int      `json:"max_depth"`
	AvgDegree    float64  `json:"avg_degree"`
	Density      float64  `json:"density"`
	Components   int      `json:"components"`
	CriticalPath []string `json:"critical_path"`
}

// ExplorerStats represents comprehensive explorer statistics
type ExplorerStats struct {
	Overview    DAGOverview             `json:"overview"`
	Performance map[string]interface{}  `json:"performance"`
	Economic    map[string]interface{}  `json:"economic"`
	Consensus   map[string]interface{}  `json:"consensus"`
	Activity    []ActivitySummary       `json:"recent_activity"`
	ChartData   map[string][]ChartPoint `json:"chart_data"`
	Timestamp   string                  `json:"timestamp"`
}

// ActivitySummary represents recent network activity
type ActivitySummary struct {
	Timestamp   int64  `json:"timestamp"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Actor       string `json:"actor,omitempty"`
	Amount      string `json:"amount,omitempty"`
	TxID        string `json:"tx_id,omitempty"`
}

// ChartPoint represents a data point for charts
type ChartPoint struct {
	Timestamp int64       `json:"timestamp"`
	Value     interface{} `json:"value"`
	Label     string      `json:"label,omitempty"`
}

// dagOverviewHandler returns an overview of the DAG state
func (s *Server) dagOverviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.ConsensusService == nil || s.ConsensusService.DAG == nil {
		http.Error(w, "Consensus service not available", http.StatusServiceUnavailable)
		return
	}

	// Get all nodes from the DAG
	allNodes := s.ConsensusService.DAG.GetAllNodes()

	// Calculate overview statistics
	overview := DAGOverview{
		TotalNodes:         len(allNodes),
		FinalizedNodes:     0,
		PendingNodes:       0,
		TransactionTypes:   make(map[string]int),
		EconomicTypes:      make(map[string]int),
		RecentTransactions: make([]TransactionSummary, 0),
		TopNodes:           make([]string, 0),
		DAGHeight:          0,
		Timestamp:          time.Now().Format(time.RFC3339),
		NetworkStats:       make(map[string]interface{}),
	}

	// Collect recent transactions and statistics
	var recentTxs []TransactionSummary

	for _, node := range allNodes {
		tx := node.Transaction

		// Count transaction types
		overview.TransactionTypes[string(tx.Type)]++

		// Count finalized vs pending
		if tx.Finalized {
			overview.FinalizedNodes++
		} else {
			overview.PendingNodes++
		}

		// Count economic types
		if tx.EconomicData != nil {
			overview.EconomicTypes[tx.EconomicData.EconomicType]++
		}

		// Create transaction summary
		txSummary := TransactionSummary{
			ID:          tx.ID,
			Type:        string(tx.Type),
			Finalized:   tx.Finalized,
			Timestamp:   tx.Timestamp,
			ParentCount: len(tx.ParentIDs),
			ChildCount:  len(node.Children),
			Score:       node.Score,
		}

		// Add economic details if available
		if tx.EconomicData != nil {
			txSummary.EconomicType = tx.EconomicData.EconomicType
			if tx.EconomicData.Amount != nil {
				txSummary.Amount = tx.EconomicData.Amount.String()
			}
			txSummary.FromAccount = tx.EconomicData.FromAccount
			txSummary.ToAccount = tx.EconomicData.ToAccount
		}

		recentTxs = append(recentTxs, txSummary)
	}

	// Sort by timestamp (most recent first) and take top 20
	if len(recentTxs) > 0 {
		// Simple sort by timestamp (in production, use sort.Slice)
		if len(recentTxs) > 20 {
			overview.RecentTransactions = recentTxs[:20]
		} else {
			overview.RecentTransactions = recentTxs
		}
	}

	// Add network statistics
	if s.P2PNode != nil {
		peerCount := len(s.P2PNode.Host.Network().Peers())
		overview.NetworkStats["peer_count"] = peerCount
		overview.NetworkStats["node_id"] = s.P2PNode.Host.ID().String()
	}

	// Add consensus statistics
	if s.ConsensusService != nil {
		overview.NetworkStats["conflicts_detected"] = s.ConsensusService.ConflictResolver.GetConflictsCount()
		overview.NetworkStats["finalization_threshold"] = s.ConsensusService.FinalizationThreshold
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

// dagNodesHandler returns a list of DAG nodes with pagination and filtering
func (s *Server) dagNodesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.ConsensusService == nil || s.ConsensusService.DAG == nil {
		http.Error(w, "Consensus service not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	limit := 50 // default
	offset := 0 // default
	txType := query.Get("type")
	finalized := query.Get("finalized")
	economicType := query.Get("economic_type")

	// Parse limit and offset
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := query.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get all nodes and apply filters
	allNodes := s.ConsensusService.DAG.GetAllNodes()
	var filteredNodes []TransactionSummary

	for _, node := range allNodes {
		tx := node.Transaction

		// Apply filters
		if txType != "" && string(tx.Type) != txType {
			continue
		}
		if finalized != "" {
			isFinalized := finalized == "true"
			if tx.Finalized != isFinalized {
				continue
			}
		}
		if economicType != "" && (tx.EconomicData == nil || tx.EconomicData.EconomicType != economicType) {
			continue
		}

		// Create summary
		txSummary := TransactionSummary{
			ID:          tx.ID,
			Type:        string(tx.Type),
			Finalized:   tx.Finalized,
			Timestamp:   tx.Timestamp,
			ParentCount: len(tx.ParentIDs),
			ChildCount:  len(node.Children),
			Score:       node.Score,
		}

		if tx.EconomicData != nil {
			txSummary.EconomicType = tx.EconomicData.EconomicType
			if tx.EconomicData.Amount != nil {
				txSummary.Amount = tx.EconomicData.Amount.String()
			}
			txSummary.FromAccount = tx.EconomicData.FromAccount
			txSummary.ToAccount = tx.EconomicData.ToAccount
		}

		filteredNodes = append(filteredNodes, txSummary)
	}

	// Apply pagination
	total := len(filteredNodes)
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var paginatedNodes []TransactionSummary
	if start < end {
		paginatedNodes = filteredNodes[start:end]
	} else {
		paginatedNodes = make([]TransactionSummary, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes":    paginatedNodes,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
		"has_more": end < total,
	})
}

// dagNodeHandler returns detailed information about a specific DAG node
func (s *Server) dagNodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.ConsensusService == nil || s.ConsensusService.DAG == nil {
		http.Error(w, "Consensus service not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	nodeID := vars["id"]

	node, exists := s.ConsensusService.DAG.GetNode(nodeID)
	if !exists {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	tx := node.Transaction

	// Create main node summary
	nodeSummary := &TransactionSummary{
		ID:          tx.ID,
		Type:        string(tx.Type),
		Finalized:   tx.Finalized,
		Timestamp:   tx.Timestamp,
		ParentCount: len(tx.ParentIDs),
		ChildCount:  len(node.Children),
		Score:       node.Score,
	}

	if tx.EconomicData != nil {
		nodeSummary.EconomicType = tx.EconomicData.EconomicType
		if tx.EconomicData.Amount != nil {
			nodeSummary.Amount = tx.EconomicData.Amount.String()
		}
		nodeSummary.FromAccount = tx.EconomicData.FromAccount
		nodeSummary.ToAccount = tx.EconomicData.ToAccount
	}

	// Get parents
	var parents []TransactionSummary
	allNodes := s.ConsensusService.DAG.GetAllNodes()
	for _, parentID := range tx.ParentIDs {
		if parentNode, exists := allNodes[parentID]; exists {
			parentTx := parentNode.Transaction
			parents = append(parents, TransactionSummary{
				ID:          parentTx.ID,
				Type:        string(parentTx.Type),
				Finalized:   parentTx.Finalized,
				Timestamp:   parentTx.Timestamp,
				ParentCount: len(parentTx.ParentIDs),
				ChildCount:  len(parentNode.Children),
				Score:       parentNode.Score,
			})
		}
	}

	// Get children
	var children []TransactionSummary
	for _, childNode := range node.Children {
		childTx := childNode.Transaction
		children = append(children, TransactionSummary{
			ID:          childTx.ID,
			Type:        string(childTx.Type),
			Finalized:   childTx.Finalized,
			Timestamp:   childTx.Timestamp,
			ParentCount: len(childTx.ParentIDs),
			ChildCount:  len(childNode.Children),
			Score:       childNode.Score,
		})
	}

	// Calculate dependencies and dependents (simplified)
	dependencies := tx.ParentIDs
	var dependents []string
	for _, childNode := range node.Children {
		dependents = append(dependents, childNode.Transaction.ID)
	}

	nodeDetail := DAGNodeDetail{
		Node:             nodeSummary,
		Parents:          parents,
		Children:         children,
		Dependencies:     dependencies,
		Dependents:       dependents,
		Score:            node.Score,
		Path:             []string{nodeID},  // Simplified
		Depth:            len(tx.ParentIDs), // Simplified depth calculation
		Finalized:        tx.Finalized,
		FinalizationPath: []string{}, // Would need path calculation algorithm
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeDetail)
}

// explorerTransactionsHandler returns transaction data formatted for the explorer
func (s *Server) explorerTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Delegate to the existing listTransactionsHandler but with explorer-specific formatting
	s.listTransactionsHandler(w, r)
}

// explorerTransactionHandler returns detailed transaction information
func (s *Server) explorerTransactionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	txID := vars["id"]

	if s.ConsensusService == nil || s.ConsensusService.DAG == nil {
		http.Error(w, "Consensus service not available", http.StatusServiceUnavailable)
		return
	}

	node, exists := s.ConsensusService.DAG.GetNode(txID)
	if !exists {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	tx := node.Transaction

	// Create comprehensive transaction details
	txDetail := map[string]interface{}{
		"id":             tx.ID,
		"type":           tx.Type,
		"data":           tx.Data,
		"parent_ids":     tx.ParentIDs,
		"metadata":       tx.Metadata,
		"finalized":      tx.Finalized,
		"timestamp":      tx.Timestamp,
		"score":          node.Score,
		"children_count": len(node.Children),
		"signature":      tx.Signature,
		"created_at":     time.Unix(tx.Timestamp, 0).Format(time.RFC3339),
	}

	// Add economic data if available
	if tx.EconomicData != nil {
		txDetail["economic_data"] = map[string]interface{}{
			"from_account":  tx.EconomicData.FromAccount,
			"to_account":    tx.EconomicData.ToAccount,
			"amount":        tx.EconomicData.Amount.String(),
			"fee":           tx.EconomicData.Fee.String(),
			"economic_type": tx.EconomicData.EconomicType,
			"query_id":      tx.EconomicData.QueryID,
			"model_tier":    tx.EconomicData.ModelTier,
			"quality_score": tx.EconomicData.QualityScore,
			"response_time": tx.EconomicData.ResponseTime,
		}
	}

	// Add children information
	var children []map[string]interface{}
	for _, childNode := range node.Children {
		childTx := childNode.Transaction
		children = append(children, map[string]interface{}{
			"id":        childTx.ID,
			"type":      childTx.Type,
			"timestamp": childTx.Timestamp,
			"finalized": childTx.Finalized,
		})
	}
	txDetail["children"] = children

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txDetail)
}

// explorerSearchHandler handles search queries across transactions and nodes
func (s *Server) explorerSearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.ConsensusService == nil || s.ConsensusService.DAG == nil {
		http.Error(w, "Consensus service not available", http.StatusServiceUnavailable)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	searchType := r.URL.Query().Get("type") // "transaction", "account", "all"
	if searchType == "" {
		searchType = "all"
	}

	var results []map[string]interface{}

	// Search in transactions
	if searchType == "all" || searchType == "transaction" {
		allNodes := s.ConsensusService.DAG.GetAllNodes()
		for _, node := range allNodes {
			tx := node.Transaction

			// Search in transaction ID, data, and metadata
			if strings.Contains(tx.ID, query) ||
				strings.Contains(tx.Data, query) ||
				containsInMetadata(tx.Metadata, query) {

				result := map[string]interface{}{
					"type":             "transaction",
					"id":               tx.ID,
					"transaction_type": tx.Type,
					"timestamp":        tx.Timestamp,
					"finalized":        tx.Finalized,
					"match_field":      getMatchField(tx, query),
				}

				if tx.EconomicData != nil {
					result["economic_type"] = tx.EconomicData.EconomicType
					result["from_account"] = tx.EconomicData.FromAccount
					result["to_account"] = tx.EconomicData.ToAccount
					if tx.EconomicData.Amount != nil {
						result["amount"] = tx.EconomicData.Amount.String()
					}
				}

				results = append(results, result)
			}

			// Search in economic data accounts
			if tx.EconomicData != nil && (searchType == "all" || searchType == "account") {
				if strings.Contains(tx.EconomicData.FromAccount, query) ||
					strings.Contains(tx.EconomicData.ToAccount, query) {

					results = append(results, map[string]interface{}{
						"type":           "account",
						"account":        getAccountMatch(tx.EconomicData, query),
						"transaction_id": tx.ID,
						"timestamp":      tx.Timestamp,
						"economic_type":  tx.EconomicData.EconomicType,
						"amount":         tx.EconomicData.Amount.String(),
						"match_field":    "account",
					})
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   query,
		"type":    searchType,
		"results": results,
		"count":   len(results),
	})
}

// explorerStatsHandler returns comprehensive statistics for the explorer
func (s *Server) explorerStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Get basic DAG overview
	s.dagOverviewHandler(w, r)
}

// dagTopologyHandler returns DAG topology data for visualization
func (s *Server) dagTopologyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if s.ConsensusService == nil || s.ConsensusService.DAG == nil {
		http.Error(w, "Consensus service not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	limit := 100                  // default
	layout := query.Get("layout") // "force", "hierarchical", "circular"
	if layout == "" {
		layout = "force"
	}

	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	allNodes := s.ConsensusService.DAG.GetAllNodes()

	var nodes []DAGTopologyNode
	var edges []DAGTopologyEdge
	nodeCount := 0

	// Convert DAG nodes to topology format
	for _, dagNode := range allNodes {
		if nodeCount >= limit {
			break
		}

		tx := dagNode.Transaction

		// Create topology node
		topologyNode := DAGTopologyNode{
			ID:        tx.ID,
			Type:      string(tx.Type),
			Finalized: tx.Finalized,
			Score:     dagNode.Score,
			Timestamp: tx.Timestamp,
			Level:     len(tx.ParentIDs), // Simple level calculation
			Position:  calculateNodePosition(tx.ID, nodeCount, layout),
			Data: map[string]interface{}{
				"parent_count": len(tx.ParentIDs),
				"child_count":  len(dagNode.Children),
			},
		}

		// Add economic data to node
		if tx.EconomicData != nil {
			topologyNode.Data["economic_type"] = tx.EconomicData.EconomicType
			topologyNode.Data["from_account"] = tx.EconomicData.FromAccount
			topologyNode.Data["to_account"] = tx.EconomicData.ToAccount
			if tx.EconomicData.Amount != nil {
				topologyNode.Data["amount"] = tx.EconomicData.Amount.String()
			}
		}

		nodes = append(nodes, topologyNode)

		// Create edges for parent relationships
		for _, parentID := range tx.ParentIDs {
			edges = append(edges, DAGTopologyEdge{
				Source: parentID,
				Target: tx.ID,
				Type:   "parent",
			})
		}

		nodeCount++
	}

	// Calculate topology statistics
	stats := DAGTopologyStats{
		TotalNodes:   len(nodes),
		TotalEdges:   len(edges),
		MaxDepth:     calculateMaxDepth(nodes),
		AvgDegree:    calculateAvgDegree(nodes, edges),
		Density:      calculateDensity(len(nodes), len(edges)),
		Components:   1,          // Simplified
		CriticalPath: []string{}, // Would need path calculation
	}

	topology := DAGTopology{
		Nodes: nodes,
		Edges: edges,
		Stats: stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topology)
}

// Helper functions for the explorer

func containsInMetadata(metadata map[string]string, query string) bool {
	for key, value := range metadata {
		if strings.Contains(key, query) || strings.Contains(value, query) {
			return true
		}
	}
	return false
}

func getMatchField(tx *types.Transaction, query string) string {
	if strings.Contains(tx.ID, query) {
		return "id"
	}
	if strings.Contains(tx.Data, query) {
		return "data"
	}
	if containsInMetadata(tx.Metadata, query) {
		return "metadata"
	}
	return "unknown"
}

func getAccountMatch(eData *types.EconomicTransactionData, query string) string {
	if strings.Contains(eData.FromAccount, query) {
		return eData.FromAccount
	}
	if strings.Contains(eData.ToAccount, query) {
		return eData.ToAccount
	}
	return ""
}

func calculateNodePosition(nodeID string, index int, layout string) map[string]interface{} {
	switch layout {
	case "circular":
		angle := float64(index) * 2.0 * 3.14159 / 100.0 // Assume 100 nodes in circle
		return map[string]interface{}{
			"x": 200.0 * math.Cos(angle),
			"y": 200.0 * math.Sin(angle),
		}
	case "hierarchical":
		return map[string]interface{}{
			"x": float64(index%10) * 100.0,
			"y": float64(index/10) * 100.0,
		}
	default: // force layout
		// Simple random-ish positioning (in production, use proper force-directed layout)
		hash := hashString(nodeID)
		return map[string]interface{}{
			"x": float64(hash%400) - 200.0,
			"y": float64((hash/400)%400) - 200.0,
		}
	}
}

func calculateMaxDepth(nodes []DAGTopologyNode) int {
	maxLevel := 0
	for _, node := range nodes {
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}
	return maxLevel
}

func calculateAvgDegree(nodes []DAGTopologyNode, edges []DAGTopologyEdge) float64 {
	if len(nodes) == 0 {
		return 0
	}
	return float64(len(edges)*2) / float64(len(nodes))
}

func calculateDensity(nodeCount, edgeCount int) float64 {
	if nodeCount < 2 {
		return 0
	}
	maxEdges := nodeCount * (nodeCount - 1) / 2
	return float64(edgeCount) / float64(maxEdges)
}

func hashString(s string) int {
	hash := 0
	for _, c := range s {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// Context Management Handlers

// recentContextHandler returns recent conversation context
func (s *Server) recentContextHandler(w http.ResponseWriter, r *http.Request) {
	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "Context management not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	maxItemsStr := r.URL.Query().Get("max_items")
	maxItems := 10 // default
	if maxItemsStr != "" {
		if parsed, err := strconv.Atoi(maxItemsStr); err == nil && parsed > 0 {
			maxItems = parsed
		}
	}

	// Get recent context
	recentActivities, err := s.RAGSystem.ContextManager.GetRecentContext(r.Context(), maxItems)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get recent context: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"activities": recentActivities,
		"count":      len(recentActivities),
		"timestamp":  time.Now().Unix(),
	})
}

// newConversationHandler starts a new conversation
func (s *Server) newConversationHandler(w http.ResponseWriter, r *http.Request) {
	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "Context management not available", http.StatusServiceUnavailable)
		return
	}

	// Start new conversation
	if err := s.RAGSystem.ContextManager.StartNewConversation(r.Context()); err != nil {
		http.Error(w, fmt.Sprintf("Failed to start new conversation: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"conversation_id": s.RAGSystem.ContextManager.GetConversationID(),
		"message":         "New conversation started",
		"timestamp":       time.Now().Unix(),
	})
}

// conversationSummaryHandler returns conversation summary
func (s *Server) conversationSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "Context management not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation_id": s.RAGSystem.ContextManager.GetConversationID(),
		"session_start":   s.RAGSystem.ContextManager.GetSessionStartTime().Unix(),
		"activities":      s.RAGSystem.ContextManager.GetActivityCount(),
		"node_id":         s.RAGSystem.ContextManager.GetNodeID(),
		"timestamp":       time.Now().Unix(),
	})
}

// trackActivityHandler manually tracks an activity
func (s *Server) trackActivityHandler(w http.ResponseWriter, r *http.Request) {
	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "Context management not available", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		Type        string                 `json:"type"`
		Description string                 `json:"description"`
		Metadata    map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Track the activity
	s.RAGSystem.ContextManager.TrackAction(r.Context(), request.Type, request.Description, request.Metadata)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"message":   "Activity tracked successfully",
		"timestamp": time.Now().Unix(),
	})
}

// AGI System Handlers

// agiStateHandler returns the current AGI state
func (s *Server) agiStateHandler(w http.ResponseWriter, r *http.Request) {
	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil || s.RAGSystem.ContextManager.GetAGISystem() == nil {
		http.Error(w, "AGI system not available", http.StatusServiceUnavailable)
		return
	}

	agiSystem := s.RAGSystem.ContextManager.GetAGISystem()
	state := agiSystem.GetCurrentAGIState()

	if state == nil {
		http.Error(w, "AGI state not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// agiIntelligenceHandler returns AGI intelligence metrics
func (s *Server) agiIntelligenceHandler(w http.ResponseWriter, r *http.Request) {
	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil || s.RAGSystem.ContextManager.GetAGISystem() == nil {
		http.Error(w, "AGI system not available", http.StatusServiceUnavailable)
		return
	}

	agiSystem := s.RAGSystem.ContextManager.GetAGISystem()
	state := agiSystem.GetCurrentAGIState()

	if state == nil {
		http.Error(w, "AGI state not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"intelligence_level": state.IntelligenceLevel,
		"capabilities": map[string]interface{}{
			"logical_reasoning":   state.Metrics.QueryAccuracy,
			"pattern_recognition": 85.0, // Would come from intelligence core
			"learning_rate":       state.Metrics.LearningRate,
			"creativity":          state.Metrics.CreativityScore,
			"adaptation_speed":    state.Metrics.AdaptationSpeed,
		},
		"learning_state": state.LearningState,
		"version":        state.Version,
		"last_evolution": state.LastEvolution.Unix(),
		"timestamp":      time.Now().Unix(),
	})
}

// agiDomainsHandler returns AGI knowledge domains
func (s *Server) agiDomainsHandler(w http.ResponseWriter, r *http.Request) {
	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil || s.RAGSystem.ContextManager.GetAGISystem() == nil {
		http.Error(w, "AGI system not available", http.StatusServiceUnavailable)
		return
	}

	agiSystem := s.RAGSystem.ContextManager.GetAGISystem()
	state := agiSystem.GetCurrentAGIState()

	if state == nil {
		http.Error(w, "AGI state not initialized", http.StatusInternalServerError)
		return
	}

	// Prepare domain information for response
	domains := make(map[string]interface{})
	for name, domain := range state.KnowledgeDomains {
		domains[name] = map[string]interface{}{
			"name":            domain.Name,
			"description":     domain.Description,
			"expertise_level": domain.Expertise,
			"concept_count":   len(domain.Concepts),
			"pattern_count":   len(domain.Patterns),
			"last_updated":    domain.LastUpdated.Unix(),
			"related_domains": domain.RelatedDomains,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"domains":            domains,
		"cross_domain_links": state.CrossDomainLinks,
		"total_domains":      len(state.KnowledgeDomains),
		"timestamp":          time.Now().Unix(),
	})
}

// agiEvolutionHandler returns AGI evolution history
func (s *Server) agiEvolutionHandler(w http.ResponseWriter, r *http.Request) {
	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil || s.RAGSystem.ContextManager.GetAGISystem() == nil {
		http.Error(w, "AGI system not available", http.StatusServiceUnavailable)
		return
	}

	agiSystem := s.RAGSystem.ContextManager.GetAGISystem()
	state := agiSystem.GetCurrentAGIState()

	if state == nil {
		http.Error(w, "AGI state not initialized", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"current_version":   state.Version,
		"last_evolution":    state.LastEvolution.Unix(),
		"evolution_events":  state.Metrics.EvolutionEvents,
		"capability_growth": state.Metrics.CapabilityGrowth,
		"knowledge_growth":  state.Metrics.KnowledgeGrowth,
		"meta_objectives":   state.MetaObjectives,
		"goals":             state.Goals,
		"recent_insights":   state.LearningState.RecentInsights,
		"timestamp":         time.Now().Unix(),
	})
}

// agiPromptHandler generates an AGI-enhanced prompt for a query
func (s *Server) agiPromptHandler(w http.ResponseWriter, r *http.Request) {
	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil || s.RAGSystem.ContextManager.GetAGISystem() == nil {
		http.Error(w, "AGI system not available", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	agiSystem := s.RAGSystem.ContextManager.GetAGISystem()
	enhancedPrompt := agiSystem.GetAGIPromptForQuery(r.Context(), request.Query)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"original_query":     request.Query,
		"enhanced_prompt":    enhancedPrompt,
		"agi_version":        agiSystem.GetCurrentAGIState().Version,
		"intelligence_level": agiSystem.GetCurrentAGIState().IntelligenceLevel,
		"timestamp":          time.Now().Unix(),
	})
}

// agiCortexHistoryHandler returns historical AGI cortex states from vector DB
func (s *Server) agiCortexHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "AGI system not available", http.StatusServiceUnavailable)
		return
	}

	// Get limit parameter
	limit := 10
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := parseInt(limitParam, 1, 50); err == nil {
			limit = parsedLimit
		}
	}

	agiSystem := s.RAGSystem.ContextManager.GetAGISystem()
	if agiSystem == nil {
		http.Error(w, "AGI system not initialized", http.StatusServiceUnavailable)
		return
	}

	// Get historical states from vector DB
	states, err := agiSystem.GetAGIStateHistory(limit)
	if err != nil {
		log.Printf("Error getting AGI state history: %v", err)
		http.Error(w, "Failed to retrieve cortex history", http.StatusInternalServerError)
		return
	}

	// Create response with historical evolution
	response := map[string]interface{}{
		"node_id":              s.RAGSystem.ContextManager.GetNodeID(),
		"total_states":         len(states),
		"states":               states,
		"current_intelligence": agiSystem.GetCurrentAGIState().IntelligenceLevel,
		"timestamp":            time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

// agiContextWindowsHandler returns context windows stored in vector DB
func (s *Server) agiContextWindowsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "AGI system not available", http.StatusServiceUnavailable)
		return
	}

	// Get parameters
	contextType := r.URL.Query().Get("type")
	if contextType == "" {
		contextType = "interaction" // default
	}

	limit := 20
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := parseInt(limitParam, 1, 100); err == nil {
			limit = parsedLimit
		}
	}

	agiSystem := s.RAGSystem.ContextManager.GetAGISystem()
	if agiSystem == nil {
		http.Error(w, "AGI system not initialized", http.StatusServiceUnavailable)
		return
	}

	// Load context windows from vector DB
	contextWindows, err := agiSystem.LoadContextWindows(contextType, limit)
	if err != nil {
		log.Printf("Error loading context windows: %v", err)
		http.Error(w, "Failed to load context windows", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"node_id":       s.RAGSystem.ContextManager.GetNodeID(),
		"context_type":  contextType,
		"total_windows": len(contextWindows),
		"windows":       contextWindows,
		"timestamp":     time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

// agiVectorSearchHandler allows searching through AGI-related vector data
func (s *Server) agiVectorSearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.RAGSystem == nil {
		http.Error(w, "RAG system not available", http.StatusServiceUnavailable)
		return
	}

	var request struct {
		Query         string  `json:"query"`
		MaxResults    int     `json:"max_results,omitempty"`
		MinSimilarity float32 `json:"min_similarity,omitempty"`
		SearchType    string  `json:"search_type,omitempty"` // "agi_state", "context_window", "all"
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Set defaults
	if request.MaxResults == 0 {
		request.MaxResults = 10
	}
	if request.MinSimilarity == 0 {
		request.MinSimilarity = 0.7
	}
	if request.SearchType == "" {
		request.SearchType = "all"
	}

	// Get model for embedding generation
	model, err := s.RAGSystem.ModelManager.GetModel(s.RAGSystem.QueryProcessor.ModelID)
	if err != nil {
		http.Error(w, "Model not available", http.StatusServiceUnavailable)
		return
	}

	// Generate embedding for search query
	embedding, err := model.GenerateEmbedding(r.Context(), request.Query)
	if err != nil {
		http.Error(w, "Failed to generate embedding", http.StatusInternalServerError)
		return
	}

	// Search vector database
	vectorQuery := types.VectorQuery{
		Embedding:     embedding,
		MaxResults:    request.MaxResults,
		MinSimilarity: request.MinSimilarity,
	}

	docs, err := s.RAGSystem.VectorDB.SearchSimilar(vectorQuery)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Filter by search type if specified
	var filteredDocs []types.VectorDocument
	for _, doc := range docs {
		if request.SearchType == "all" {
			filteredDocs = append(filteredDocs, doc)
		} else if docType, ok := doc.Metadata["type"].(string); ok {
			switch request.SearchType {
			case "agi_state":
				if docType == "agi_cortex_state" {
					filteredDocs = append(filteredDocs, doc)
				}
			case "context_window":
				if docType == "context_window" {
					filteredDocs = append(filteredDocs, doc)
				}
			}
		}
	}

	response := map[string]interface{}{
		"query":         request.Query,
		"search_type":   request.SearchType,
		"total_results": len(filteredDocs),
		"documents":     filteredDocs,
		"timestamp":     time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

// agiNetworkSnapshotHandler returns the current network-wide AGI snapshot
func (s *Server) agiNetworkSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.ConsensusService == nil || s.ConsensusService.AGIRecorder == nil {
		http.Error(w, "AGI blockchain recorder not available", http.StatusServiceUnavailable)
		return
	}

	// Get network snapshot from blockchain
	snapshot := s.ConsensusService.AGIRecorder.GetNetworkAGISnapshot()
	if snapshot == nil {
		http.Error(w, "No network snapshot available", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(snapshot)
}

// agiNetworkNodesHandler returns AGI states for all nodes in the network
func (s *Server) agiNetworkNodesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.ConsensusService == nil || s.ConsensusService.AGIRecorder == nil {
		http.Error(w, "AGI blockchain recorder not available", http.StatusServiceUnavailable)
		return
	}

	// Get all node AGI states
	nodeStates := s.ConsensusService.AGIRecorder.GetNodeAGIStates()

	response := map[string]interface{}{
		"total_nodes": len(nodeStates),
		"node_states": nodeStates,
		"timestamp":   time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

// agiDomainLeadersHandler returns the nodes with highest expertise in each domain
func (s *Server) agiDomainLeadersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.ConsensusService == nil || s.ConsensusService.AGIRecorder == nil {
		http.Error(w, "AGI blockchain recorder not available", http.StatusServiceUnavailable)
		return
	}

	// Get domain leaders
	domainLeaders := s.ConsensusService.AGIRecorder.GetDomainLeaders()

	// Get detailed information for each leader
	nodeStates := s.ConsensusService.AGIRecorder.GetNodeAGIStates()
	leaderDetails := make(map[string]map[string]interface{})

	for domain, nodeID := range domainLeaders {
		if state, exists := nodeStates[nodeID]; exists {
			leaderDetails[domain] = map[string]interface{}{
				"node_id":            nodeID,
				"expertise":          state.DomainKnowledge[domain],
				"intelligence_level": state.IntelligenceLevel,
				"reputation":         state.Reputation,
				"last_update":        state.LastUpdate,
			}
		}
	}

	response := map[string]interface{}{
		"domain_leaders": domainLeaders,
		"leader_details": leaderDetails,
		"timestamp":      time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

// consciousnessStateHandler returns the current consciousness state
func (s *Server) consciousnessStateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "RAG system not available", http.StatusServiceUnavailable)
		return
	}

	consciousnessRuntime := s.RAGSystem.ContextManager.GetConsciousnessRuntime()
	if consciousnessRuntime == nil {
		http.Error(w, "Consciousness runtime not available", http.StatusServiceUnavailable)
		return
	}

	state := consciousnessRuntime.GetConsciousnessState()
	json.NewEncoder(w).Encode(state)
}

// consciousnessMetricsHandler returns consciousness runtime metrics
func (s *Server) consciousnessMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "RAG system not available", http.StatusServiceUnavailable)
		return
	}

	consciousnessRuntime := s.RAGSystem.ContextManager.GetConsciousnessRuntime()
	if consciousnessRuntime == nil {
		http.Error(w, "Consciousness runtime not available", http.StatusServiceUnavailable)
		return
	}

	metrics := consciousnessRuntime.GetMetrics()
	json.NewEncoder(w).Encode(metrics)
}

// consciousnessMemoryHandler returns the current working memory
func (s *Server) consciousnessMemoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "RAG system not available", http.StatusServiceUnavailable)
		return
	}

	consciousnessRuntime := s.RAGSystem.ContextManager.GetConsciousnessRuntime()
	if consciousnessRuntime == nil {
		http.Error(w, "Consciousness runtime not available", http.StatusServiceUnavailable)
		return
	}

	workingMemory := consciousnessRuntime.GetWorkingMemory()
	json.NewEncoder(w).Encode(workingMemory)
}

// conversationHistoryHandler returns conversation history from vector DB
func (s *Server) conversationHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "RAG system not available", http.StatusServiceUnavailable)
		return
	}

	// Get limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	history, err := s.RAGSystem.ContextManager.GetRecentConversationHistory(r.Context(), limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve conversation history: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation_id": s.RAGSystem.ContextManager.GetConversationID(),
		"history":         history,
		"count":           len(history),
		"timestamp":       time.Now().Unix(),
	})
}

// conversationContextHandler returns formatted conversation context for prompts
func (s *Server) conversationContextHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	if s.RAGSystem == nil || s.RAGSystem.ContextManager == nil {
		http.Error(w, "RAG system not available", http.StatusServiceUnavailable)
		return
	}

	// Get max_events parameter
	maxEventsStr := r.URL.Query().Get("max_events")
	maxEvents := 10 // default
	if maxEventsStr != "" {
		if parsedMax, err := strconv.Atoi(maxEventsStr); err == nil && parsedMax > 0 {
			maxEvents = parsedMax
		}
	}

	context, err := s.RAGSystem.ContextManager.GetConversationContext(r.Context(), maxEvents)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve conversation context: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"conversation_id": s.RAGSystem.ContextManager.GetConversationID(),
		"context":         context,
		"max_events":      maxEvents,
		"timestamp":       time.Now().Unix(),
	})
}
