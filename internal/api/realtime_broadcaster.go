package api

import (
	"bufio"
	"log"
	"os"
	"strings"
	"time"
)

// RealtimeBroadcaster handles real-time broadcasting of system events
type RealtimeBroadcaster struct {
	wsManager *WebSocketManager
	server    *Server

	// Broadcasting channels
	systemEvents  chan *WebSocketMessage
	ollamaEvents  chan *WebSocketMessage
	queryEvents   chan *WebSocketMessage
	metricsEvents chan *WebSocketMessage

	// Control
	stopChan chan struct{}
	started  bool
}

// NewRealtimeBroadcaster creates a new real-time broadcaster
func NewRealtimeBroadcaster(wsManager *WebSocketManager, server *Server) *RealtimeBroadcaster {
	return &RealtimeBroadcaster{
		wsManager:     wsManager,
		server:        server,
		systemEvents:  make(chan *WebSocketMessage, 100),
		ollamaEvents:  make(chan *WebSocketMessage, 100),
		queryEvents:   make(chan *WebSocketMessage, 100),
		metricsEvents: make(chan *WebSocketMessage, 100),
		stopChan:      make(chan struct{}),
	}
}

// Start begins real-time broadcasting
func (rb *RealtimeBroadcaster) Start() {
	if rb.started {
		return
	}

	rb.started = true
	log.Printf("Starting real-time broadcaster")

	// Start event processors
	go rb.processSystemEvents()
	go rb.processOllamaEvents()
	go rb.processQueryEvents()
	go rb.processMetricsEvents()

	// Start periodic broadcasters
	go rb.broadcastPeriodicMetrics()
	go rb.broadcastConsciousnessState()
	go rb.broadcastOllamaStatus()

	// Start log file watcher
	go rb.watchCortexLogs()
}

// Stop stops the real-time broadcaster
func (rb *RealtimeBroadcaster) Stop() {
	if !rb.started {
		return
	}

	log.Printf("Stopping real-time broadcaster")
	close(rb.stopChan)
	rb.started = false
}

// BroadcastSystemEvent broadcasts a system event
func (rb *RealtimeBroadcaster) BroadcastSystemEvent(eventType string, data interface{}) {
	if !rb.started {
		return
	}

	message := &WebSocketMessage{
		Type: WSMsgTypeNotification,
		Data: map[string]interface{}{
			"event_type": eventType,
			"data":       data,
			"source":     "system",
		},
		Timestamp: time.Now(),
	}

	select {
	case rb.systemEvents <- message:
	default:
		log.Printf("System events channel full, dropping event")
	}
}

// BroadcastOllamaEvent broadcasts an Ollama-related event
func (rb *RealtimeBroadcaster) BroadcastOllamaEvent(eventType string, data interface{}) {
	if !rb.started {
		return
	}

	message := &WebSocketMessage{
		Type: WSMsgTypeOllamaStatus,
		Data: map[string]interface{}{
			"event_type": eventType,
			"data":       data,
			"source":     "ollama",
		},
		Timestamp: time.Now(),
	}

	select {
	case rb.ollamaEvents <- message:
	default:
		log.Printf("Ollama events channel full, dropping event")
	}
}

// BroadcastQueryEvent broadcasts a query-related event
func (rb *RealtimeBroadcaster) BroadcastQueryEvent(queryID, eventType string, data interface{}) {
	if !rb.started {
		return
	}

	message := &WebSocketMessage{
		Type: WSMsgTypeNotification,
		ID:   queryID,
		Data: map[string]interface{}{
			"event_type": eventType,
			"data":       data,
			"source":     "query",
		},
		Timestamp: time.Now(),
	}

	select {
	case rb.queryEvents <- message:
	default:
		log.Printf("Query events channel full, dropping event")
	}
}

// BroadcastModelEvent broadcasts a model management event
func (rb *RealtimeBroadcaster) BroadcastModelEvent(eventType string, data interface{}) {
	if !rb.started {
		return
	}

	message := &WebSocketMessage{
		Type: WSMsgTypeOllamaStatus,
		Data: map[string]interface{}{
			"event_type": eventType,
			"data":       data,
			"source":     "model_management",
		},
		Timestamp: time.Now(),
	}

	select {
	case rb.ollamaEvents <- message:
	default:
		log.Printf("Model events channel full, dropping event")
	}
}

// BroadcastLogEntry broadcasts a log entry to connected clients
func (rb *RealtimeBroadcaster) BroadcastLogEntry(level, message, source string) {
	if !rb.started {
		return
	}

	logMessage := &WebSocketMessage{
		Type: "ollama_logs",
		Data: map[string]interface{}{
			"level":   level,
			"message": message,
			"source":  source,
		},
		Timestamp: time.Now(),
	}

	select {
	case rb.ollamaEvents <- logMessage:
	default:
		log.Printf("Log events channel full, dropping log entry")
	}
}

// BroadcastCortexLogEntry broadcasts a cortex log entry to connected clients
func (rb *RealtimeBroadcaster) BroadcastCortexLogEntry(level, message, source string) {
	if !rb.started {
		return
	}

	logMessage := &WebSocketMessage{
		Type: "cortex_logs",
		Data: map[string]interface{}{
			"level":   level,
			"message": message,
			"source":  source,
		},
		Timestamp: time.Now(),
	}

	select {
	case rb.systemEvents <- logMessage:
	default:
		log.Printf("Cortex log events channel full, dropping log entry")
	}
}

// processSystemEvents processes and broadcasts system events
func (rb *RealtimeBroadcaster) processSystemEvents() {
	for {
		select {
		case event := <-rb.systemEvents:
			rb.wsManager.BroadcastToSubscribers(SubTypeSystemEvents, event)

		case <-rb.stopChan:
			return
		}
	}
}

// processOllamaEvents processes and broadcasts Ollama events
func (rb *RealtimeBroadcaster) processOllamaEvents() {
	for {
		select {
		case event := <-rb.ollamaEvents:
			rb.wsManager.BroadcastToSubscribers(SubTypeOllamaStatus, event)

		case <-rb.stopChan:
			return
		}
	}
}

// processQueryEvents processes and broadcasts query events
func (rb *RealtimeBroadcaster) processQueryEvents() {
	for {
		select {
		case event := <-rb.queryEvents:
			rb.wsManager.BroadcastToSubscribers(SubTypeQueryResults, event)

		case <-rb.stopChan:
			return
		}
	}
}

// processMetricsEvents processes and broadcasts metrics events
func (rb *RealtimeBroadcaster) processMetricsEvents() {
	for {
		select {
		case event := <-rb.metricsEvents:
			rb.wsManager.BroadcastToSubscribers(SubTypeMetrics, event)

		case <-rb.stopChan:
			return
		}
	}
}

// broadcastPeriodicMetrics broadcasts system metrics periodically
func (rb *RealtimeBroadcaster) broadcastPeriodicMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rb.broadcastSystemMetrics()

		case <-rb.stopChan:
			return
		}
	}
}

// broadcastSystemMetrics broadcasts current system metrics
func (rb *RealtimeBroadcaster) broadcastSystemMetrics() {
	if rb.server == nil || rb.server.Metrics == nil {
		return
	}

	rb.server.Metrics.mu.RLock()
	metrics := map[string]interface{}{
		"queries_processed":     rb.server.Metrics.QueriesProcessed,
		"query_successes":       rb.server.Metrics.QuerySuccesses,
		"query_failures":        rb.server.Metrics.QueryFailures,
		"documents_added":       rb.server.Metrics.DocumentsAdded,
		"rag_queries_total":     rb.server.Metrics.RAGQueriesTotal,
		"rag_query_failures":    rb.server.Metrics.RAGQueryFailures,
		"websocket_connections": rb.wsManager.GetConnectionCount(),
	}
	rb.server.Metrics.mu.RUnlock()

	message := &WebSocketMessage{
		Type: WSMsgTypeMetrics,
		Data: map[string]interface{}{
			"system_metrics": metrics,
			"timestamp":      time.Now(),
		},
		Timestamp: time.Now(),
	}

	rb.wsManager.BroadcastToSubscribers(SubTypeMetrics, message)
}

// broadcastConsciousnessState broadcasts consciousness state updates
func (rb *RealtimeBroadcaster) broadcastConsciousnessState() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if rb.server.RAGSystem != nil {
				agiSystem := rb.server.RAGSystem.GetAGISystem()
				if agiSystem != nil {
					state := agiSystem.GetConsciousnessState()

					message := &WebSocketMessage{
						Type: WSMsgTypeConsciousness,
						Data: map[string]interface{}{
							"consciousness_state": state,
							"timestamp":           time.Now(),
						},
						Timestamp: time.Now(),
					}

					rb.wsManager.BroadcastToSubscribers(SubTypeConsciousness, message)
				}
			}

		case <-rb.stopChan:
			return
		}
	}
}

// broadcastOllamaStatus broadcasts Ollama status updates
func (rb *RealtimeBroadcaster) broadcastOllamaStatus() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if rb.server.RAGSystem != nil {
				status := rb.server.RAGSystem.GetEmbeddedStatus()

				message := &WebSocketMessage{
					Type: WSMsgTypeOllamaStatus,
					Data: map[string]interface{}{
						"ollama_status": status,
						"timestamp":     time.Now(),
					},
					Timestamp: time.Now(),
				}

				rb.wsManager.BroadcastToSubscribers(SubTypeOllamaStatus, message)
			}

		case <-rb.stopChan:
			return
		}
	}
}

// watchCortexLogs watches the cortex.log file and broadcasts new log entries
func (rb *RealtimeBroadcaster) watchCortexLogs() {
	// Find the cortex.log file
	logPath := "cortex.log"
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		// Try common locations
		possiblePaths := []string{
			"./cortex.log",
			"../cortex.log",
			"/var/log/cortex.log",
			"/tmp/cortex.log",
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				logPath = path
				break
			}
		}
	}

	log.Printf("Starting cortex log watcher for: %s", logPath)

	for {
		select {
		case <-rb.stopChan:
			return
		default:
			rb.tailLogFile(logPath)
			// If tailing fails, wait a bit before retrying
			time.Sleep(5 * time.Second)
		}
	}
}

// tailLogFile tails the log file and broadcasts new entries
func (rb *RealtimeBroadcaster) tailLogFile(logPath string) {
	file, err := os.Open(logPath)
	if err != nil {
		log.Printf("Failed to open cortex log file %s: %v", logPath, err)
		return
	}
	defer file.Close()

	// Get file size to start reading from the end
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Failed to get cortex log file info: %v", err)
		return
	}

	// Start from the end of the file for new logs
	currentSize := fileInfo.Size()
	file.Seek(currentSize, 0)

	scanner := bufio.NewScanner(file)

	// Watch for new lines
	for {
		select {
		case <-rb.stopChan:
			return
		default:
			if scanner.Scan() {
				line := scanner.Text()
				if line != "" {
					rb.parseCortexLogLine(line)
				}
			} else {
				// Check if file has grown
				newFileInfo, err := file.Stat()
				if err != nil {
					log.Printf("Error checking cortex log file: %v", err)
					return
				}

				if newFileInfo.Size() < currentSize {
					// File was truncated or rotated, restart from beginning
					log.Printf("Cortex log file rotated, restarting watcher")
					return
				}

				// No new data, wait a bit
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// parseCortexLogLine parses a cortex log line and broadcasts it
func (rb *RealtimeBroadcaster) parseCortexLogLine(line string) {
	// Parse cortex log format: "2025/06/30 21:55:50 message"
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		// Fallback for malformed lines
		rb.BroadcastCortexLogEntry("INFO", line, "cortex")
		return
	}

	// Extract timestamp and message
	timestamp := parts[0] + " " + parts[1]
	message := parts[2]

	// Determine log level from message content
	level := "INFO"
	messageLower := strings.ToLower(message)

	if strings.Contains(messageLower, "error") || strings.Contains(messageLower, "failed") {
		level = "ERROR"
	} else if strings.Contains(messageLower, "warning") || strings.Contains(messageLower, "warn") {
		level = "WARN"
	} else if strings.Contains(messageLower, "debug") {
		level = "DEBUG"
	}

	// Extract source from message if possible
	source := "cortex"
	if strings.Contains(messageLower, "ollama") {
		source = "ollama"
	} else if strings.Contains(messageLower, "websocket") {
		source = "websocket"
	} else if strings.Contains(messageLower, "agent") {
		source = "agents"
	} else if strings.Contains(messageLower, "rag") || strings.Contains(messageLower, "consciousness") {
		source = "rag"
	} else if strings.Contains(messageLower, "p2p") || strings.Contains(messageLower, "network") {
		source = "p2p"
	}

	// Include timestamp in the message for better context
	fullMessage := "[" + timestamp + "] " + message

	rb.BroadcastCortexLogEntry(level, fullMessage, source)
}
