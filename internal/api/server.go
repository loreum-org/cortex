package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/loreum-org/cortex/internal/agenthub"
	"github.com/loreum-org/cortex/internal/consensus"
	"github.com/loreum-org/cortex/internal/p2p"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/pkg/types"
)

// Server represents the API server
type Server struct {
	Router           *mux.Router
	P2PNode          *p2p.P2PNode
	ConsensusService *consensus.ConsensusService
	SolverAgent      *agenthub.SolverAgent
	RAGSystem        *rag.RAGSystem
	httpServer       *http.Server
}

// NewServer creates a new API server
func NewServer(p2pNode *p2p.P2PNode, consensusService *consensus.ConsensusService, solverAgent *agenthub.SolverAgent, ragSystem *rag.RAGSystem) *Server {
	s := &Server{
		Router:           mux.NewRouter(),
		P2PNode:          p2pNode,
		ConsensusService: consensusService,
		SolverAgent:      solverAgent,
		RAGSystem:        ragSystem,
	}

	s.setupRoutes()
	return s
}

// setupRoutes sets up the routes for the API server
func (s *Server) setupRoutes() {
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
}

// submitQueryHandler handles submit query requests
func (s *Server) submitQueryHandler(w http.ResponseWriter, r *http.Request) {
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

	response, err := s.SolverAgent.Process(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
	var req RAGQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := s.RAGSystem.Query(r.Context(), req.Text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": response})
}

// generateID generates a simple ID
func generateID() string {
	return "id_" + time.Now().Format("20060102150405")
}
