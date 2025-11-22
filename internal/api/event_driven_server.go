package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/loreum-org/cortex/internal/agents"
	"github.com/loreum-org/cortex/internal/ai"
	"github.com/loreum-org/cortex/internal/consensus"
	"github.com/loreum-org/cortex/internal/economy"
	"github.com/loreum-org/cortex/internal/events"
	"github.com/loreum-org/cortex/internal/events/handlers"
	"github.com/loreum-org/cortex/internal/p2p"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/internal/services"
)

// EventDrivenServer is a pure event-driven API server using WebSocket-only communication
type EventDrivenServer struct {
	// Core components
	router     *mux.Router
	httpServer *http.Server

	// Event system
	eventBus    *events.EventBus
	eventRouter *events.EventRouter
	wsBridge    *events.WebSocketEventBridge

	// Service dependencies
	p2pNode          *p2p.P2PNode
	consensusService *consensus.ConsensusService
	ragSystem        *rag.RAGSystem
	economicEngine   *economy.EconomicEngine
	serviceRegistry  *services.ServiceRegistryManager
	agentRegistry    *agents.AgentRegistry
	embeddedManager  *ai.EmbeddedModelManager
	standardSolver   *agents.StandardSolverAgent

	// Event handlers
	apiHandler             *handlers.APIEventHandler
	enhancedServiceHandler *handlers.EnhancedServiceHandler
	websocketHandler       *handlers.WebSocketEventHandler

	// WebSocket upgrader
	upgrader websocket.Upgrader

	// Server state
	isRunning bool
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc

	// Metrics
	metrics *EventDrivenServerMetrics
}

// EventDrivenServerMetrics tracks server performance
type EventDrivenServerMetrics struct {
	StartTime         time.Time `json:"start_time"`
	ConnectionsTotal  int64     `json:"connections_total"`
	ConnectionsActive int       `json:"connections_active"`
	EventsProcessed   int64     `json:"events_processed"`
	EventsPublished   int64     `json:"events_published"`
	EventsFailed      int64     `json:"events_failed"`
	RequestsProcessed int64     `json:"requests_processed"`
	ResponsesSent     int64     `json:"responses_sent"`

	mu sync.RWMutex
}

// EventDrivenServerConfig configures the event-driven server
type EventDrivenServerConfig struct {
	// HTTP configuration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	MaxHeaderBytes int

	// WebSocket configuration
	MaxConnections int
	CheckOrigin    func(*http.Request) bool

	// Event system configuration
	EventBusConfig *events.EventBusConfig
	RouterConfig   *events.RouterConfig
	BridgeConfig   *events.BridgeConfig

	// CORS settings
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// DefaultEventDrivenServerConfig returns a default configuration
func DefaultEventDrivenServerConfig() *EventDrivenServerConfig {
	return &EventDrivenServerConfig{
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
		MaxConnections: 1000,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
		EventBusConfig: events.DefaultEventBusConfig(),
		RouterConfig:   events.DefaultRouterConfig(),
		BridgeConfig:   events.DefaultBridgeConfig(),
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}
}

// NewEventDrivenServer creates a new event-driven API server
func NewEventDrivenServer(
	p2pNode *p2p.P2PNode,
	consensusService *consensus.ConsensusService,
	ragSystem *rag.RAGSystem,
	economicEngine *economy.EconomicEngine,
	serviceRegistry *services.ServiceRegistryManager,
	config *EventDrivenServerConfig,
) *EventDrivenServer {
	if config == nil {
		config = DefaultEventDrivenServerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize event system
	eventBus := events.NewEventBus(config.EventBusConfig)
	eventRouter := events.NewEventRouter(eventBus, config.RouterConfig)
	wsBridge := events.NewWebSocketEventBridge(eventRouter, config.BridgeConfig)

	// Initialize agent registry
	agentRegistry := agents.NewAgentRegistry()

	// Get embedded manager from RAG system (don't create a new one)
	var embeddedManager *ai.EmbeddedModelManager
	if ragSystem != nil {
		embeddedManager = ragSystem.EmbeddedManager
	}
	if embeddedManager == nil {
		// Fallback: create a new one if RAG system doesn't have one
		embeddedManager = ai.NewEmbeddedModelManager(ai.DefaultEmbeddedManagerConfig())
	}

	server := &EventDrivenServer{
		router:           mux.NewRouter(),
		eventBus:         eventBus,
		eventRouter:      eventRouter,
		wsBridge:         wsBridge,
		p2pNode:          p2pNode,
		consensusService: consensusService,
		ragSystem:        ragSystem,
		economicEngine:   economicEngine,
		serviceRegistry:  serviceRegistry,
		agentRegistry:    agentRegistry,
		embeddedManager:  embeddedManager,
		ctx:              ctx,
		cancel:           cancel,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     config.CheckOrigin,
		},
		metrics: &EventDrivenServerMetrics{
			StartTime: time.Now(),
		},
	}

	// Initialize API event handler
	server.apiHandler = handlers.NewAPIEventHandler(
		agentRegistry,
		ragSystem,
		economicEngine,
		serviceRegistry,
		p2pNode,
		embeddedManager,
		ragSystem.ContextManager,
		eventBus,
		eventRouter,
	)

	// Initialize enhanced service handler
	server.enhancedServiceHandler = handlers.NewEnhancedServiceHandler(
		agentRegistry,
		ragSystem,
		economicEngine,
		serviceRegistry,
		p2pNode,
		eventBus,
		eventRouter,
	)

	// Initialize WebSocket event handler
	server.websocketHandler = handlers.NewWebSocketEventHandler(eventBus)

	// Setup routes and handlers
	server.setupRoutes()
	server.registerEventHandlers()
	server.initializeAgentRegistry()

	return server
}

// Start starts the event-driven server
func (s *EventDrivenServer) Start(address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("server is already running")
	}

	// Event bus auto-starts, no explicit start needed

	// Start agent registry monitoring
	s.agentRegistry.StartMonitoring()

	// Setup HTTP server
	s.httpServer = &http.Server{
		Addr:           address,
		Handler:        s.router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	s.isRunning = true

	log.Printf("[EventDrivenServer] Starting event-driven server on %s", address)
	log.Printf("[EventDrivenServer] WebSocket endpoint: ws://%s/ws", address)

	// Start metrics publishing
	go s.startMetricsPublisher()

	// Start the HTTP server
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.isRunning = false
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil
}

// Stop stops the event-driven server
func (s *EventDrivenServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	log.Printf("[EventDrivenServer] Stopping event-driven server...")

	// Cancel context
	s.cancel()

	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("[EventDrivenServer] Error stopping HTTP server: %v", err)
	}

	// Stop agent registry (if it has a stop method)
	// s.agentRegistry.Stop() // Not implemented

	// Stop event bus (if it has a stop method)
	// s.eventBus.Stop() // Not implemented

	s.isRunning = false
	log.Printf("[EventDrivenServer] Server stopped successfully")

	return nil
}

// GetMetrics returns server metrics
func (s *EventDrivenServer) GetMetrics() *EventDrivenServerMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *s.metrics
	metrics.ConnectionsActive = s.wsBridge.GetConnectionCount()

	return &metrics
}

// GetEventBus returns the event bus for external use
func (s *EventDrivenServer) GetEventBus() *events.EventBus {
	return s.eventBus
}

// GetEventRouter returns the event router for external use
func (s *EventDrivenServer) GetEventRouter() *events.EventRouter {
	return s.eventRouter
}

// GetWebSocketBridge returns the WebSocket bridge for external use
func (s *EventDrivenServer) GetWebSocketBridge() *events.WebSocketEventBridge {
	return s.wsBridge
}

// Private methods

func (s *EventDrivenServer) setupRoutes() {
	// CORS middleware
	s.router.Use(s.corsMiddleware)

	// Health check endpoint
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// WebSocket endpoint - the only client communication endpoint
	s.router.HandleFunc("/ws", s.handleWebSocket).Methods("GET")

	// Static file serving for the frontend (optional)
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/"))).Methods("GET")
}

func (s *EventDrivenServer) registerEventHandlers() {
	// Register API event handler
	if err := s.eventBus.Subscribe(s.apiHandler); err != nil {
		log.Printf("[EventDrivenServer] Failed to subscribe API handler: %v", err)
	} else {
		log.Printf("[EventDrivenServer] API handler registered for %d event types", len(s.apiHandler.SubscribedEvents()))
	}

	// Register enhanced service handler
	if err := s.eventBus.Subscribe(s.enhancedServiceHandler); err != nil {
		log.Printf("[EventDrivenServer] Failed to subscribe enhanced service handler: %v", err)
	} else {
		log.Printf("[EventDrivenServer] Enhanced service handler registered for %d event types", len(s.enhancedServiceHandler.SubscribedEvents()))
	}

	// Register WebSocket event handler
	if err := s.eventBus.Subscribe(s.websocketHandler); err != nil {
		log.Printf("[EventDrivenServer] Failed to subscribe WebSocket handler: %v", err)
	} else {
		log.Printf("[EventDrivenServer] WebSocket handler registered for %d event types", len(s.websocketHandler.SubscribedEvents()))
	}

	// Register RAG event handler if available
	if s.ragSystem != nil {
		ragHandler := handlers.NewRAGEventHandler(s.ragSystem, s.eventBus)
		if err := s.eventBus.Subscribe(ragHandler); err != nil {
			log.Printf("[EventDrivenServer] Failed to subscribe RAG handler: %v", err)
		} else {
			log.Printf("[EventDrivenServer] RAG handler registered for %d event types", len(ragHandler.SubscribedEvents()))
		}
	}

	// Register Agent event handler if available
	if s.agentRegistry != nil {
		agentHandler := handlers.NewAgentEventHandler(s.agentRegistry, s.eventBus)
		if err := s.eventBus.Subscribe(agentHandler); err != nil {
			log.Printf("[EventDrivenServer] Failed to subscribe Agent handler: %v", err)
		} else {
			log.Printf("[EventDrivenServer] Agent handler registered for %d event types", len(agentHandler.SubscribedEvents()))
		}
	}

	totalEventTypes := len(s.apiHandler.SubscribedEvents()) + len(s.enhancedServiceHandler.SubscribedEvents()) + len(s.websocketHandler.SubscribedEvents())
	log.Printf("[EventDrivenServer] All event handlers registered for %d+ event types", totalEventTypes)
}

func (s *EventDrivenServer) initializeAgentRegistry() {
	// Register the standard solver agent
	if s.p2pNode != nil {
		nodeID := s.p2pNode.Host.ID().String()

		solverConfig := &agents.SolverConfig{
			DefaultModel: "ollama-cogito:latest",
		}

		s.standardSolver = agents.NewStandardSolverAgent(nodeID, solverConfig, s.ragSystem)
		if s.economicEngine != nil {
			s.standardSolver.SetEconomicEngine(s.economicEngine)
		}

		// Set up callback to update solver agent when embedded manager updates default model
		if s.embeddedManager != nil {
			s.embeddedManager.OnDefaultModelReady = func(modelID string) {
				log.Printf("[EventDrivenServer] Embedded manager default model ready: %s", modelID)
				if s.standardSolver != nil {
					s.standardSolver.SetModelManager(s.embeddedManager.ModelManager, modelID)
				}
				if s.ragSystem != nil && s.ragSystem.ContextManager != nil {
					log.Printf("Updating consciousness runtime to use embedded model: %s", modelID)
					s.ragSystem.ContextManager.SetModelManager(s.embeddedManager.ModelManager, modelID)
				}
			}

			// If the embedded manager already has a default model ready, call the callback now
			models := s.embeddedManager.ModelManager.ListModels()
			for _, model := range models {
				if strings.HasPrefix(model.ID, "ollama-cogito") {
					log.Printf("[EventDrivenServer] Embedded manager already has default model: %s", model.ID)
					if s.standardSolver != nil {
						s.standardSolver.SetModelManager(s.embeddedManager.ModelManager, model.ID)
					}
					break
				}
			}
		}

		if err := s.agentRegistry.RegisterAgent(s.standardSolver); err != nil {
			log.Printf("[EventDrivenServer] Warning: Failed to register standard solver agent: %v", err)
		} else {
			log.Printf("[EventDrivenServer] Standard solver agent registered successfully")
		}
	}
}

func (s *EventDrivenServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(s.metrics.StartTime).Seconds(),
		"components": map[string]bool{
			"event_bus":        s.eventBus != nil,
			"event_router":     s.eventRouter != nil,
			"websocket_bridge": s.wsBridge != nil,
			"agent_registry":   s.agentRegistry != nil,
			"rag_system":       s.ragSystem != nil,
			"economic_engine":  s.economicEngine != nil,
			"p2p_node":         s.p2pNode != nil,
		},
		"metrics": s.GetMetrics(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Simple JSON encoding
	fmt.Fprintf(w, `{
		"status": "%s",
		"timestamp": %d,
		"uptime": %.2f,
		"active_connections": %d,
		"events_processed": %d
	}`,
		health["status"],
		health["timestamp"],
		health["uptime"],
		s.wsBridge.GetConnectionCount(),
		s.metrics.EventsProcessed,
	)
}

func (s *EventDrivenServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[EventDrivenServer] WebSocket upgrade failed: %v", err)
		return
	}

	// Extract user ID from request (this would be from authentication in real implementation)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "anonymous"
	}

	// Handle the WebSocket connection through the bridge
	wsConn, err := s.wsBridge.HandleWebSocketConnection(conn, userID)
	if err != nil {
		log.Printf("[EventDrivenServer] Failed to handle WebSocket connection: %v", err)
		conn.Close()
		return
	}

	// Register connection with WebSocket handler - create proper bridge between event systems
	handlerConn := &handlers.WebSocketConnection{
		ID:       wsConn.ID,
		UserID:   userID,
		Send:     make(chan interface{}, 256),
		LastSeen: wsConn.LastActivity,
	}
	s.websocketHandler.RegisterConnection(handlerConn)

	// Bridge messages between the WebSocket bridge and the event handler
	go s.bridgeWebSocketMessages(wsConn, handlerConn)

	// Update metrics
	s.metrics.mu.Lock()
	s.metrics.ConnectionsTotal++
	s.metrics.mu.Unlock()

	log.Printf("[EventDrivenServer] WebSocket connection established: %s (user: %s)", wsConn.ID, userID)
}

// bridgeWebSocketMessages bridges messages between the WebSocket bridge and event handler
func (s *EventDrivenServer) bridgeWebSocketMessages(wsConn *events.WebSocketConnection, handlerConn *handlers.WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[EventDrivenServer] Recovered from panic in WebSocket bridge: %v", r)
		}
		// Cleanup connections
		s.websocketHandler.UnregisterConnection(handlerConn.ID)
	}()

	// Forward messages from handler to WebSocket bridge
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[EventDrivenServer] Recovered from panic in handler->bridge forwarding: %v", r)
			}
		}()

		for {
			select {
			case msg := <-handlerConn.Send:
				if wsConn.Active {
					// Convert handler message to bridge message
					if bridgeMsg, ok := s.convertToBridgeMessage(msg); ok {
						select {
						case wsConn.SendChan <- bridgeMsg:
							// Message sent successfully
						default:
							log.Printf("[EventDrivenServer] WebSocket bridge send channel full")
							return
						}
					}
				} else {
					return
				}
			case <-wsConn.Context.Done():
				return
			}
		}
	}()

	// Keep connection alive and handle context cancellation
	<-wsConn.Context.Done()
}

// convertToBridgeMessage converts handler messages to bridge messages
func (s *EventDrivenServer) convertToBridgeMessage(msg interface{}) (events.WebSocketMessage, bool) {
	switch m := msg.(type) {
	case *handlers.UnifiedMessage:
		return events.WebSocketMessage{
			Type:      m.Type,
			Method:    m.Method,
			ID:        m.ID,
			Data:      m.Data,
			Error:     m.Error,
			Timestamp: time.Now(), // Convert string timestamp to time.Time
			Metadata:  m.Metadata,
		}, true
	case handlers.UnifiedMessage:
		return events.WebSocketMessage{
			Type:      m.Type,
			Method:    m.Method,
			ID:        m.ID,
			Data:      m.Data,
			Error:     m.Error,
			Timestamp: time.Now(),
			Metadata:  m.Metadata,
		}, true
	default:
		// Try to handle generic messages
		return events.WebSocketMessage{
			Type:      "notification",
			Data:      msg,
			Timestamp: time.Now(),
		}, true
	}
}

func (s *EventDrivenServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *EventDrivenServer) startMetricsPublisher() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.publishMetrics()
		}
	}
}

func (s *EventDrivenServer) publishMetrics() {
	metrics := s.GetMetrics()

	metricsEvent := events.NewEvent(
		events.EventTypeMetricsUpdated,
		"event_driven_server",
		events.MetricsData{
			Type:   "server",
			Source: "event_driven_server",
			Values: map[string]interface{}{
				"connections_total":  metrics.ConnectionsTotal,
				"connections_active": metrics.ConnectionsActive,
				"events_processed":   metrics.EventsProcessed,
				"events_published":   metrics.EventsPublished,
				"events_failed":      metrics.EventsFailed,
				"requests_processed": metrics.RequestsProcessed,
				"responses_sent":     metrics.ResponsesSent,
				"uptime_seconds":     time.Since(metrics.StartTime).Seconds(),
			},
			Timestamp: time.Now(),
		},
	)

	// Broadcast metrics to all subscribers
	if err := s.wsBridge.BroadcastEvent(s.ctx, metricsEvent); err != nil {
		log.Printf("[EventDrivenServer] Failed to broadcast metrics: %v", err)
	}
}

// Event handling helpers

// PublishEvent publishes an event to the event bus
func (s *EventDrivenServer) PublishEvent(ctx context.Context, event events.Event) error {
	s.metrics.mu.Lock()
	s.metrics.EventsPublished++
	s.metrics.mu.Unlock()

	return s.eventBus.Publish(event)
}

// BroadcastToClients broadcasts an event to all WebSocket clients
func (s *EventDrivenServer) BroadcastToClients(ctx context.Context, event events.Event) error {
	return s.wsBridge.BroadcastEvent(ctx, event)
}

// SendToClient sends an event to a specific WebSocket client
func (s *EventDrivenServer) SendToClient(connectionID string, event events.Event) error {
	return s.wsBridge.SendEventToConnection(connectionID, event)
}
