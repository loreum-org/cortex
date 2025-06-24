package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/loreum-org/cortex/internal/agenthub"
	"github.com/loreum-org/cortex/internal/consensus"
	"github.com/loreum-org/cortex/internal/economy" // Import the economy package
	"github.com/loreum-org/cortex/internal/p2p"
	"github.com/loreum-org/cortex/internal/rag"
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
	Router           *mux.Router
	P2PNode          *p2p.P2PNode
	ConsensusService *consensus.ConsensusService
	SolverAgent      *agenthub.SolverAgent
	RAGSystem        *rag.RAGSystem
	EconomicEngine   *economy.EconomicEngine // Add EconomicEngine field
	EconomyService   *EconomyService         // Add EconomyService for API handlers
	httpServer       *http.Server
	Metrics          *ServerMetrics
}

// NewServer creates a new API server
func NewServer(p2pNode *p2p.P2PNode, consensusService *consensus.ConsensusService, solverAgent *agenthub.SolverAgent, ragSystem *rag.RAGSystem, economicEngine *economy.EconomicEngine) *Server {
	s := &Server{
		Router:           mux.NewRouter(),
		P2PNode:          p2pNode,
		ConsensusService: consensusService,
		SolverAgent:      solverAgent,
		RAGSystem:        ragSystem,
		EconomicEngine:   economicEngine, // Initialize EconomicEngine
		Metrics: &ServerMetrics{
			StartTime:       time.Now(),
			QueryLatencies:  make([]time.Duration, 0, 100),
			RAGQueryLatency: make([]time.Duration, 0, 100),
			Events:          make([]Event, 0, 100),
		},
	}
	s.EconomyService = NewEconomyService(economicEngine) // Initialize EconomyService

	s.setupRoutes()
	return s
}

// setupRoutes sets up the routes for the API server
func (s *Server) setupRoutes() {
	// Add CORS middleware
	s.Router.Use(s.corsMiddleware)

	// Health check
	s.Router.HandleFunc("/health", s.healthHandler).Methods("GET")

	// Node info
	s.Router.HandleFunc("/node/info", s.nodeInfoHandler).Methods("GET")
	s.Router.HandleFunc("/node/peers", s.peersHandler).Methods("GET")

	// Transactions
	s.Router.HandleFunc("/transactions", s.listTransactionsHandler).Methods("GET")
	s.Router.HandleFunc("/transactions", s.createTransactionHandler).Methods("POST")
	s.Router.HandleFunc("/transactions/{id}", s.getTransactionHandler).Methods("GET")

	// Queries
	s.Router.HandleFunc("/queries", s.submitQueryHandler).Methods("POST")
	s.Router.HandleFunc("/queries/{id}", s.getQueryResultHandler).Methods("GET")

	// RAG
	s.Router.HandleFunc("/rag/documents", s.addDocumentHandler).Methods("POST")
	s.Router.HandleFunc("/rag/query", s.ragQueryHandler).Methods("POST")

	// Metrics endpoints
	s.Router.HandleFunc("/metrics/network", s.networkMetricsHandler).Methods("GET")
	s.Router.HandleFunc("/metrics/queries", s.queryMetricsHandler).Methods("GET")
	s.Router.HandleFunc("/metrics/system", s.systemMetricsHandler).Methods("GET")
	s.Router.HandleFunc("/metrics/rag", s.ragMetricsHandler).Methods("GET")
	s.Router.HandleFunc("/metrics/economy", s.economyMetricsHandler).Methods("GET")

	// Events endpoint
	s.Router.HandleFunc("/events", s.eventsHandler).Methods("GET")

	// Register economic routes
	if s.EconomyService != nil {
		s.EconomyService.RegisterRoutes(s.Router)
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
	Text     string            `json:"text"`
	Type     string            `json:"type"`
	Metadata map[string]string `json:"metadata"`
	UserID   string            `json:"user_id,omitempty"` // Added for economic integration
}

// submitQueryHandler handles submit query requests
func (s *Server) submitQueryHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := &types.Query{
		ID:        generateID(),
		Text:      req.Text,
		Type:      req.Type,
		Metadata:  req.Metadata,
		Timestamp: time.Now().Unix(),
	}

	// Economic integration - Process payment if economic engine is available
	var paymentTx *economy.Transaction
	var userID string

	// Get user ID from request, header, or use a default
	if req.UserID != "" {
		userID = req.UserID
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

	// Determine query type
	queryType := economy.QueryTypeCompletion
	if req.Type == "chat" {
		queryType = economy.QueryTypeChat
	} else if req.Type == "code" || req.Type == "programming" {
		queryType = economy.QueryTypeCodeGen
	} else if req.Type == "embedding" {
		queryType = economy.QueryTypeEmbedding
	} else if req.Type == "rag" {
		queryType = economy.QueryTypeRAG
	}

	// Calculate input size
	inputSize := int64(len(req.Text))

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
			queryType,
			inputSize,
		)

		if err != nil {
			// If payment fails, still process the query in demo mode
			log.Printf("Payment processing failed: %v. Processing query in demo mode.", err)
			s.AddEvent("payment_failed", map[string]interface{}{
				"user_id": userID,
				"error":   err.Error(),
			})
		} else {
			// Add payment info to query metadata
			if query.Metadata == nil {
				query.Metadata = make(map[string]string)
			}
			query.Metadata["payment_tx_id"] = paymentTx.ID
			query.Metadata["user_id"] = userID

			s.AddEvent("payment_processed", map[string]interface{}{
				"user_id":    userID,
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
	Text string `json:"text"`
}

// ragQueryHandler handles RAG query requests
func (s *Server) ragQueryHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var req RAGQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.RAGSystem.Query(r.Context(), req.Text)

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

// generateID generates a simple ID
func generateID() string {
	return "id_" + time.Now().Format("20060102150405")
}
