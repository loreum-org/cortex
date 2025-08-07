package events

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// SmartEventRouter provides intelligent event routing with priority handling
type SmartEventRouter struct {
	eventBus           *EventBus
	routingRules       []RoutingRule
	priorityQueues     map[EventPriority]*PriorityQueue
	handlerPriorities  map[string]HandlerPriority
	routingStats       RoutingStats
	
	// Configuration
	maxQueueSize       int
	processingTimeout  time.Duration
	enableLoadBalancing bool
	enableCircuitBreaker bool
	
	// Runtime state
	mu                 sync.RWMutex
	ctx                context.Context
	cancel             context.CancelFunc
	isRunning          bool
	workers            []*RoutingWorker
	workerCount        int
}

// RoutingRule defines how events should be routed
type RoutingRule struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	EventTypePattern string                 `json:"event_type_pattern"`
	SourcePattern    string                 `json:"source_pattern,omitempty"`
	TargetPattern    string                 `json:"target_pattern,omitempty"`
	Conditions       []RoutingCondition     `json:"conditions,omitempty"`
	Priority         EventPriority          `json:"priority"`
	HandlerTargets   []string               `json:"handler_targets"`
	LoadBalancing    LoadBalancingStrategy  `json:"load_balancing"`
	RetryPolicy      RetryPolicy            `json:"retry_policy"`
	CircuitBreaker   CircuitBreakerConfig   `json:"circuit_breaker"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Enabled          bool                   `json:"enabled"`
}

// RoutingCondition defines conditional routing logic
type RoutingCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, contains, matches
	Value    interface{} `json:"value"`
}

// EventPriority defines event priority levels (alias for Priority)
type EventPriority = Priority

const (
	PriorityMedium EventPriority = 1 // Between PriorityNormal and PriorityHigh
	PriorityBackground EventPriority = 4 // After PriorityCritical
)

// HandlerPriority defines handler processing priorities
type HandlerPriority struct {
	HandlerName    string        `json:"handler_name"`
	Priority       EventPriority `json:"priority"`
	MaxConcurrency int           `json:"max_concurrency"`
	Timeout        time.Duration `json:"timeout"`
	Weight         int           `json:"weight"` // For load balancing
}

// LoadBalancingStrategy defines load balancing options
type LoadBalancingStrategy string

const (
	LoadBalanceRoundRobin LoadBalancingStrategy = "round_robin"
	LoadBalanceWeighted   LoadBalancingStrategy = "weighted"
	LoadBalanceLeastConn  LoadBalancingStrategy = "least_connections"
	LoadBalanceRandom     LoadBalancingStrategy = "random"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// CircuitBreakerConfig defines circuit breaker settings
type CircuitBreakerConfig struct {
	Enabled           bool          `json:"enabled"`
	FailureThreshold  int           `json:"failure_threshold"`
	TimeoutThreshold  time.Duration `json:"timeout_threshold"`
	RecoveryTimeout   time.Duration `json:"recovery_timeout"`
}

// PriorityQueue manages events by priority
type PriorityQueue struct {
	events   []PriorityEvent
	mu       sync.Mutex
	maxSize  int
	priority EventPriority
}

// PriorityEvent wraps an event with routing metadata
type PriorityEvent struct {
	Event       Event
	Rule        *RoutingRule
	Timestamp   time.Time
	Attempts    int
	LastAttempt time.Time
}

// RoutingWorker processes events from priority queues
type RoutingWorker struct {
	id        int
	router    *SmartEventRouter
	ctx       context.Context
	cancel    context.CancelFunc
	isRunning bool
	stats     WorkerStats
}

// WorkerStats tracks worker performance
type WorkerStats struct {
	EventsProcessed int64         `json:"events_processed"`
	EventsFailed    int64         `json:"events_failed"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastActivity    time.Time     `json:"last_activity"`
}

// RoutingStats tracks overall routing statistics
type RoutingStats struct {
	TotalEventsRouted    int64                            `json:"total_events_routed"`
	EventsByPriority     map[EventPriority]int64          `json:"events_by_priority"`
	EventsByRule         map[string]int64                 `json:"events_by_rule"`
	AverageRoutingTime   time.Duration                    `json:"average_routing_time"`
	FailedRoutings       int64                            `json:"failed_routings"`
	CircuitBreakerTrips  int64                            `json:"circuit_breaker_trips"`
	HandlerPerformance   map[string]HandlerPerformance    `json:"handler_performance"`
	QueueSizes           map[EventPriority]int            `json:"queue_sizes"`
	LastUpdate           time.Time                        `json:"last_update"`
}

// HandlerPerformance tracks individual handler performance
type HandlerPerformance struct {
	EventsProcessed   int64         `json:"events_processed"`
	EventsFailed      int64         `json:"events_failed"`
	AverageLatency    time.Duration `json:"average_latency"`
	CurrentLoad       int           `json:"current_load"`
	CircuitBreakerOpen bool         `json:"circuit_breaker_open"`
	LastActivity      time.Time     `json:"last_activity"`
}

// NewSmartEventRouter creates a new smart event router
func NewSmartEventRouter(eventBus *EventBus, workerCount int) *SmartEventRouter {
	if workerCount <= 0 {
		workerCount = 4
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	router := &SmartEventRouter{
		eventBus:            eventBus,
		routingRules:        make([]RoutingRule, 0),
		priorityQueues:      make(map[EventPriority]*PriorityQueue),
		handlerPriorities:   make(map[string]HandlerPriority),
		maxQueueSize:        10000,
		processingTimeout:   30 * time.Second,
		enableLoadBalancing: true,
		enableCircuitBreaker: true,
		ctx:                 ctx,
		cancel:              cancel,
		workerCount:         workerCount,
		routingStats: RoutingStats{
			EventsByPriority:   make(map[EventPriority]int64),
			EventsByRule:       make(map[string]int64),
			HandlerPerformance: make(map[string]HandlerPerformance),
			QueueSizes:         make(map[EventPriority]int),
		},
	}
	
	// Initialize priority queues
	priorities := []EventPriority{
		PriorityCritical, PriorityHigh, PriorityMedium, PriorityLow, PriorityBackground,
	}
	
	for _, priority := range priorities {
		router.priorityQueues[priority] = &PriorityQueue{
			events:   make([]PriorityEvent, 0),
			maxSize:  router.maxQueueSize,
			priority: priority,
		}
	}
	
	return router
}

// Start begins the smart event router
func (ser *SmartEventRouter) Start() error {
	ser.mu.Lock()
	defer ser.mu.Unlock()
	
	if ser.isRunning {
		return fmt.Errorf("smart event router already running")
	}
	
	// Start workers
	ser.workers = make([]*RoutingWorker, ser.workerCount)
	for i := 0; i < ser.workerCount; i++ {
		worker := ser.createWorker(i)
		ser.workers[i] = worker
		go worker.run()
	}
	
	// Start stats collection
	go ser.statsCollectionRoutine()
	
	ser.isRunning = true
	
	log.Printf("🧠 Smart Event Router: Started with %d workers", ser.workerCount)
	return nil
}

// Stop stops the smart event router
func (ser *SmartEventRouter) Stop() error {
	ser.mu.Lock()
	defer ser.mu.Unlock()
	
	if !ser.isRunning {
		return nil
	}
	
	ser.cancel()
	
	// Stop all workers
	for _, worker := range ser.workers {
		worker.stop()
	}
	
	ser.isRunning = false
	
	log.Printf("🧠 Smart Event Router: Stopped")
	return nil
}

// RouteEvent routes an event through the smart routing system
func (ser *SmartEventRouter) RouteEvent(event Event) error {
	start := time.Now()
	
	// Find matching routing rule
	rule := ser.findMatchingRule(event)
	if rule == nil {
		// No specific rule, use default routing
		return ser.eventBus.Publish(event)
	}
	
	// Create priority event
	priorityEvent := PriorityEvent{
		Event:     event,
		Rule:      rule,
		Timestamp: time.Now(),
		Attempts:  0,
	}
	
	// Add to appropriate priority queue
	queue := ser.priorityQueues[rule.Priority]
	if err := queue.enqueue(priorityEvent); err != nil {
		ser.updateRoutingStats(rule, false, time.Since(start))
		return fmt.Errorf("failed to enqueue event: %w", err)
	}
	
	ser.updateRoutingStats(rule, true, time.Since(start))
	return nil
}

// AddRoutingRule adds a new routing rule
func (ser *SmartEventRouter) AddRoutingRule(rule RoutingRule) error {
	ser.mu.Lock()
	defer ser.mu.Unlock()
	
	// Validate rule
	if err := ser.validateRoutingRule(rule); err != nil {
		return fmt.Errorf("invalid routing rule: %w", err)
	}
	
	// Add rule
	ser.routingRules = append(ser.routingRules, rule)
	
	// Sort rules by priority
	ser.sortRoutingRules()
	
	log.Printf("🧠 Smart Router: Added routing rule '%s' with priority %d", rule.Name, rule.Priority)
	return nil
}

// RemoveRoutingRule removes a routing rule by ID
func (ser *SmartEventRouter) RemoveRoutingRule(ruleID string) error {
	ser.mu.Lock()
	defer ser.mu.Unlock()
	
	for i, rule := range ser.routingRules {
		if rule.ID == ruleID {
			ser.routingRules = append(ser.routingRules[:i], ser.routingRules[i+1:]...)
			log.Printf("🧠 Smart Router: Removed routing rule '%s'", rule.Name)
			return nil
		}
	}
	
	return fmt.Errorf("routing rule not found: %s", ruleID)
}

// SetHandlerPriority sets the priority for a specific handler
func (ser *SmartEventRouter) SetHandlerPriority(handlerName string, priority HandlerPriority) {
	ser.mu.Lock()
	defer ser.mu.Unlock()
	
	ser.handlerPriorities[handlerName] = priority
	log.Printf("🧠 Smart Router: Set priority %d for handler '%s'", priority.Priority, handlerName)
}

// findMatchingRule finds the best matching routing rule for an event
func (ser *SmartEventRouter) findMatchingRule(event Event) *RoutingRule {
	ser.mu.RLock()
	defer ser.mu.RUnlock()
	
	for _, rule := range ser.routingRules {
		if !rule.Enabled {
			continue
		}
		
		if ser.matchesRule(event, rule) {
			return &rule
		}
	}
	
	return nil
}

// matchesRule checks if an event matches a routing rule
func (ser *SmartEventRouter) matchesRule(event Event, rule RoutingRule) bool {
	// Check event type pattern
	if rule.EventTypePattern != "" && !ser.matchesPattern(event.Type, rule.EventTypePattern) {
		return false
	}
	
	// Check source pattern
	if rule.SourcePattern != "" && !ser.matchesPattern(event.Source, rule.SourcePattern) {
		return false
	}
	
	// Check target pattern
	if rule.TargetPattern != "" && !ser.matchesPattern(event.Target, rule.TargetPattern) {
		return false
	}
	
	// Check conditions
	for _, condition := range rule.Conditions {
		if !ser.evaluateCondition(event, condition) {
			return false
		}
	}
	
	return true
}

// matchesPattern checks if a value matches a pattern (supports wildcards)
func (ser *SmartEventRouter) matchesPattern(value, pattern string) bool {
	if pattern == "*" {
		return true
	}
	
	// Simple pattern matching - could be enhanced with regex
	return value == pattern
}

// evaluateCondition evaluates a routing condition
func (ser *SmartEventRouter) evaluateCondition(event Event, condition RoutingCondition) bool {
	var fieldValue interface{}
	
	// Extract field value based on field path
	switch condition.Field {
	case "type":
		fieldValue = event.Type
	case "source":
		fieldValue = event.Source
	case "target":
		fieldValue = event.Target
	case "correlation_id":
		fieldValue = event.CorrelationID
	default:
		// Check in metadata
		if event.Metadata != nil {
			fieldValue = event.Metadata[condition.Field]
		}
	}
	
	// Evaluate condition based on operator
	switch condition.Operator {
	case "eq":
		return fieldValue == condition.Value
	case "ne":
		return fieldValue != condition.Value
	case "contains":
		if str, ok := fieldValue.(string); ok {
			if substr, ok := condition.Value.(string); ok {
				return contains(str, substr)
			}
		}
	case "gt":
		return ser.compareValues(fieldValue, condition.Value) > 0
	case "lt":
		return ser.compareValues(fieldValue, condition.Value) < 0
	}
	
	return false
}

// compareValues compares two values
func (ser *SmartEventRouter) compareValues(a, b interface{}) int {
	// Simple comparison for numeric values
	if aFloat, ok := a.(float64); ok {
		if bFloat, ok := b.(float64); ok {
			if aFloat > bFloat {
				return 1
			} else if aFloat < bFloat {
				return -1
			}
			return 0
		}
	}
	return 0
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(substr) > 0 && containsRune(s, substr)))
}

func containsRune(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// sortRoutingRules sorts routing rules by priority
func (ser *SmartEventRouter) sortRoutingRules() {
	sort.Slice(ser.routingRules, func(i, j int) bool {
		return ser.routingRules[i].Priority < ser.routingRules[j].Priority
	})
}

// validateRoutingRule validates a routing rule
func (ser *SmartEventRouter) validateRoutingRule(rule RoutingRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	
	if len(rule.HandlerTargets) == 0 {
		return fmt.Errorf("at least one handler target is required")
	}
	
	return nil
}

// updateRoutingStats updates routing statistics
func (ser *SmartEventRouter) updateRoutingStats(rule *RoutingRule, success bool, duration time.Duration) {
	ser.routingStats.TotalEventsRouted++
	ser.routingStats.EventsByPriority[rule.Priority]++
	ser.routingStats.EventsByRule[rule.ID]++
	
	if !success {
		ser.routingStats.FailedRoutings++
	}
	
	// Update average routing time
	if ser.routingStats.AverageRoutingTime == 0 {
		ser.routingStats.AverageRoutingTime = duration
	} else {
		ser.routingStats.AverageRoutingTime = (ser.routingStats.AverageRoutingTime + duration) / 2
	}
	
	ser.routingStats.LastUpdate = time.Now()
}

// PriorityQueue methods

func (pq *PriorityQueue) enqueue(event PriorityEvent) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	
	if len(pq.events) >= pq.maxSize {
		return fmt.Errorf("queue full")
	}
	
	pq.events = append(pq.events, event)
	return nil
}

func (pq *PriorityQueue) dequeue() (PriorityEvent, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	
	if len(pq.events) == 0 {
		return PriorityEvent{}, false
	}
	
	event := pq.events[0]
	pq.events = pq.events[1:]
	return event, true
}

func (pq *PriorityQueue) size() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.events)
}

// Worker methods

func (ser *SmartEventRouter) createWorker(id int) *RoutingWorker {
	ctx, cancel := context.WithCancel(ser.ctx)
	return &RoutingWorker{
		id:     id,
		router: ser,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (rw *RoutingWorker) run() {
	rw.isRunning = true
	defer func() {
		rw.isRunning = false
	}()
	
	for {
		select {
		case <-rw.ctx.Done():
			return
		default:
			rw.processNextEvent()
		}
	}
}

func (rw *RoutingWorker) processNextEvent() {
	// Process events by priority order
	priorities := []EventPriority{
		PriorityCritical, PriorityHigh, PriorityMedium, PriorityLow, PriorityBackground,
	}
	
	for _, priority := range priorities {
		queue := rw.router.priorityQueues[priority]
		if event, ok := queue.dequeue(); ok {
			rw.handleEvent(event)
			return
		}
	}
	
	// No events available, sleep briefly
	time.Sleep(10 * time.Millisecond)
}

func (rw *RoutingWorker) handleEvent(priorityEvent PriorityEvent) {
	start := time.Now()
	
	// Process event according to rule
	err := rw.routeEventToHandlers(priorityEvent)
	
	duration := time.Since(start)
	rw.updateWorkerStats(err == nil, duration)
	
	if err != nil {
		log.Printf("❌ Worker %d: Failed to route event %s: %v", rw.id, priorityEvent.Event.ID, err)
		rw.handleFailedEvent(priorityEvent, err)
	}
}

func (rw *RoutingWorker) routeEventToHandlers(priorityEvent PriorityEvent) error {
	// Route to target handlers specified in the rule
	for _, handlerName := range priorityEvent.Rule.HandlerTargets {
		if err := rw.router.eventBus.PublishToHandler(priorityEvent.Event, handlerName); err != nil {
			return fmt.Errorf("failed to route to handler %s: %w", handlerName, err)
		}
	}
	
	return nil
}

func (rw *RoutingWorker) handleFailedEvent(priorityEvent PriorityEvent, err error) {
	// Implement retry logic based on retry policy
	retryPolicy := priorityEvent.Rule.RetryPolicy
	
	if priorityEvent.Attempts < retryPolicy.MaxRetries {
		// Calculate delay
		delay := retryPolicy.InitialDelay
		for i := 0; i < priorityEvent.Attempts; i++ {
			delay = time.Duration(float64(delay) * retryPolicy.BackoffFactor)
			if delay > retryPolicy.MaxDelay {
				delay = retryPolicy.MaxDelay
				break
			}
		}
		
		// Schedule retry
		go func() {
			time.Sleep(delay)
			priorityEvent.Attempts++
			priorityEvent.LastAttempt = time.Now()
			
			queue := rw.router.priorityQueues[priorityEvent.Rule.Priority]
			queue.enqueue(priorityEvent)
		}()
	}
}

func (rw *RoutingWorker) updateWorkerStats(success bool, duration time.Duration) {
	if success {
		rw.stats.EventsProcessed++
	} else {
		rw.stats.EventsFailed++
	}
	
	rw.stats.LastActivity = time.Now()
	
	// Update average latency
	if rw.stats.AverageLatency == 0 {
		rw.stats.AverageLatency = duration
	} else {
		rw.stats.AverageLatency = (rw.stats.AverageLatency + duration) / 2
	}
}

func (rw *RoutingWorker) stop() {
	rw.cancel()
}

// statsCollectionRoutine collects and updates routing statistics
func (ser *SmartEventRouter) statsCollectionRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ser.updateQueueSizeStats()
		case <-ser.ctx.Done():
			return
		}
	}
}

func (ser *SmartEventRouter) updateQueueSizeStats() {
	for priority, queue := range ser.priorityQueues {
		ser.routingStats.QueueSizes[priority] = queue.size()
	}
}

// GetRoutingStats returns current routing statistics
func (ser *SmartEventRouter) GetRoutingStats() RoutingStats {
	ser.mu.RLock()
	defer ser.mu.RUnlock()
	return ser.routingStats
}

// GetWorkerStats returns statistics for all workers
func (ser *SmartEventRouter) GetWorkerStats() []WorkerStats {
	stats := make([]WorkerStats, len(ser.workers))
	for i, worker := range ser.workers {
		stats[i] = worker.stats
	}
	return stats
}