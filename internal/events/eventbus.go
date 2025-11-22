package events

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// EventBus is the central event bus for the system
type EventBus struct {
	// Core components
	subscribers map[string][]EventHandler
	eventQueue  chan Event
	priorityQueue chan Event  // High priority event queue
	eventStore  EventStore
	
	// Synchronization
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	
	// Configuration
	config      *EventBusConfig
	
	// Backpressure management
	backpressureActive bool
	lastBackpressure   time.Time
	
	// Metrics and monitoring
	metrics     *EventMetrics
	middleware  []EventMiddleware
}

// EventBusConfig configures the event bus
type EventBusConfig struct {
	// Queue configuration
	QueueSize      int           `json:"queue_size"`
	WorkerCount    int           `json:"worker_count"`
	
	// Backpressure configuration
	PublishTimeout   time.Duration `json:"publish_timeout"`
	EnableBackpressure bool        `json:"enable_backpressure"`
	BackpressureThreshold float64  `json:"backpressure_threshold"`
	PriorityQueueSize int          `json:"priority_queue_size"`
	
	// Timeouts
	HandlerTimeout time.Duration `json:"handler_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	
	// Error handling
	MaxRetries     int           `json:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"`
	
	// Monitoring
	EnableMetrics  bool          `json:"enable_metrics"`
	LogEvents      bool          `json:"log_events"`
	
	// Event Store
	EnableEventStore bool          `json:"enable_event_store"`
	MaxStoredEvents  int           `json:"max_stored_events"`
}

// DefaultEventBusConfig returns a default configuration
func DefaultEventBusConfig() *EventBusConfig {
	return &EventBusConfig{
		QueueSize:        1000,
		WorkerCount:      10,
		PublishTimeout:   5 * time.Second,
		EnableBackpressure: true,
		BackpressureThreshold: 0.8, // 80% queue capacity
		PriorityQueueSize: 100,     // High priority events
		HandlerTimeout:   30 * time.Second,
		ShutdownTimeout:  10 * time.Second,
		MaxRetries:       3,
		RetryDelay:       1 * time.Second,
		EnableMetrics:    true,
		LogEvents:        true,
		EnableEventStore: true,
		MaxStoredEvents:  10000,
	}
}

// EventMetrics tracks event bus performance
type EventMetrics struct {
	EventsPublished   int64     `json:"events_published"`
	EventsProcessed   int64     `json:"events_processed"`
	EventsFailed      int64     `json:"events_failed"`
	EventsDropped     int64     `json:"events_dropped"`
	EventsPriority    int64     `json:"events_priority"`
	HandlersRegistered int      `json:"handlers_registered"`
	AverageLatency    time.Duration `json:"average_latency"`
	QueueUtilization  float64   `json:"queue_utilization"`
	BackpressureEvents int64    `json:"backpressure_events"`
	StartTime         time.Time `json:"start_time"`
	mu               sync.RWMutex
}

// EventMiddleware allows intercepting events
type EventMiddleware interface {
	Process(ctx context.Context, event Event, next func(context.Context, Event) error) error
}

// NewEventBus creates a new event bus
func NewEventBus(config *EventBusConfig) *EventBus {
	if config == nil {
		config = DefaultEventBusConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	eb := &EventBus{
		subscribers: make(map[string][]EventHandler),
		eventQueue:  make(chan Event, config.QueueSize),
		priorityQueue: make(chan Event, config.PriorityQueueSize),
		ctx:         ctx,
		cancel:      cancel,
		config:      config,
		backpressureActive: false,
		metrics: &EventMetrics{
			StartTime: time.Now(),
		},
		middleware: make([]EventMiddleware, 0),
	}
	
	// Initialize event store if enabled
	if config.EnableEventStore {
		store := NewInMemoryEventStore(config.MaxStoredEvents)
		eb.eventStore = NewEventStoreMiddleware(store)
		log.Printf("EventBus: Event store enabled (max events: %d)", config.MaxStoredEvents)
	}
	
	// Start worker goroutines
	for i := 0; i < config.WorkerCount; i++ {
		eb.wg.Add(1)
		go eb.worker(i)
	}
	
	log.Printf("EventBus started with %d workers, queue size %d", 
		config.WorkerCount, config.QueueSize)
	
	return eb
}

// Subscribe registers an event handler for specific event types
func (eb *EventBus) Subscribe(handler EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	handlerName := handler.HandlerName()
	eventTypes := handler.SubscribedEvents()
	
	if len(eventTypes) == 0 {
		return fmt.Errorf("handler %s has no subscribed events", handlerName)
	}
	
	for _, eventType := range eventTypes {
		if eb.subscribers[eventType] == nil {
			eb.subscribers[eventType] = make([]EventHandler, 0)
		}
		
		// Check if handler is already subscribed
		for _, existing := range eb.subscribers[eventType] {
			if existing.HandlerName() == handlerName {
				return fmt.Errorf("handler %s already subscribed to %s", handlerName, eventType)
			}
		}
		
		eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
		log.Printf("EventBus: Handler %s subscribed to %s (total handlers for this event: %d)", handlerName, eventType, len(eb.subscribers[eventType]))
	}
	
	eb.metrics.mu.Lock()
	eb.metrics.HandlersRegistered++
	eb.metrics.mu.Unlock()
	
	return nil
}

// Unsubscribe removes an event handler
func (eb *EventBus) Unsubscribe(handler EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	handlerName := handler.HandlerName()
	eventTypes := handler.SubscribedEvents()
	
	for _, eventType := range eventTypes {
		handlers := eb.subscribers[eventType]
		for i, existing := range handlers {
			if existing.HandlerName() == handlerName {
				// Remove handler from slice
				eb.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
				log.Printf("EventBus: Handler %s unsubscribed from %s", handlerName, eventType)
				break
			}
		}
		
		// Clean up empty slices
		if len(eb.subscribers[eventType]) == 0 {
			delete(eb.subscribers, eventType)
		}
	}
	
	eb.metrics.mu.Lock()
	eb.metrics.HandlersRegistered--
	eb.metrics.mu.Unlock()
	
	return nil
}

// Publish sends an event to all subscribed handlers with backpressure support
func (eb *EventBus) Publish(event Event) error {
	return eb.PublishWithTimeout(event, eb.config.PublishTimeout)
}

// PublishWithTimeout publishes an event with a specific timeout
func (eb *EventBus) PublishWithTimeout(event Event, timeout time.Duration) error {
	// Check for high priority events
	if eb.isHighPriorityEvent(event) {
		return eb.publishToPriorityQueue(event, timeout)
	}
	
	// Check backpressure conditions
	if eb.config.EnableBackpressure {
		eb.updateQueueUtilization()
		
		queueUtilization := eb.getQueueUtilization()
		if queueUtilization >= eb.config.BackpressureThreshold {
			return eb.handleBackpressure(event, timeout)
		}
	}
	
	// Try immediate publish
	select {
	case eb.eventQueue <- event:
		eb.updatePublishMetrics(event, false, false)
		return nil
		
	case <-eb.ctx.Done():
		return fmt.Errorf("event bus is shutting down")
		
	default:
		// Queue is full, apply backpressure strategy
		if !eb.config.EnableBackpressure {
			// Old behavior: drop event
			eb.metrics.mu.Lock()
			eb.metrics.EventsDropped++
			eb.metrics.mu.Unlock()
			return fmt.Errorf("event queue is full, dropping event %s", event.ID)
		}
		
		return eb.handleBackpressure(event, timeout)
	}
}

// publishToPriorityQueue publishes high priority events to priority queue
func (eb *EventBus) publishToPriorityQueue(event Event, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(eb.ctx, timeout)
	defer cancel()
	
	select {
	case eb.priorityQueue <- event:
		eb.updatePublishMetrics(event, true, false)
		return nil
		
	case <-ctx.Done():
		// Even priority queue is full, try to make space
		if ctx.Err() == context.DeadlineExceeded {
			// Try emergency publish with immediate processing
			return eb.emergencyPublish(event)
		}
		return fmt.Errorf("priority queue publish cancelled: %w", ctx.Err())
		
	case <-eb.ctx.Done():
		return fmt.Errorf("event bus is shutting down")
	}
}

// handleBackpressure implements backpressure strategies
func (eb *EventBus) handleBackpressure(event Event, timeout time.Duration) error {
	// Update backpressure metrics
	eb.metrics.mu.Lock()
	eb.metrics.BackpressureEvents++
	eb.metrics.mu.Unlock()
	
	if !eb.backpressureActive {
		eb.backpressureActive = true
		eb.lastBackpressure = time.Now()
		log.Printf("⚠️ EventBus: Backpressure activated (queue utilization: %.1f%%)", 
			eb.getQueueUtilization()*100)
	}
	
	// Strategy 1: Wait with timeout for queue space
	ctx, cancel := context.WithTimeout(eb.ctx, timeout)
	defer cancel()
	
	select {
	case eb.eventQueue <- event:
		// Successfully published after waiting
		eb.updatePublishMetrics(event, false, true)
		// Check if we can deactivate backpressure
		if eb.getQueueUtilization() < eb.config.BackpressureThreshold*0.7 {
			eb.backpressureActive = false
			log.Printf("✅ EventBus: Backpressure deactivated")
		}
		return nil
		
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			// Strategy 2: Try priority queue if event is critical
			if eb.isCriticalEvent(event) {
				select {
				case eb.priorityQueue <- event:
					eb.updatePublishMetrics(event, true, true)
					return nil
				default:
					// Both queues full, emergency publish
					return eb.emergencyPublish(event)
				}
			}
			
			// Strategy 3: Drop event with backpressure notification
			eb.metrics.mu.Lock()
			eb.metrics.EventsDropped++
			eb.metrics.mu.Unlock()
			
			log.Printf("⚠️ EventBus: Dropping event %s due to backpressure (timeout: %v)", 
				event.ID, timeout)
			return fmt.Errorf("event %s dropped due to backpressure after %v timeout", event.ID, timeout)
		}
		return fmt.Errorf("backpressure handling cancelled: %w", ctx.Err())
		
	case <-eb.ctx.Done():
		return fmt.Errorf("event bus is shutting down")
	}
}

// PublishSync publishes an event and waits for all handlers to complete
func (eb *EventBus) PublishSync(ctx context.Context, event Event) error {
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()
	
	if len(handlers) == 0 {
		if eb.config.LogEvents {
			if event.IsServiceEvent() {
				serviceType, serviceID := event.GetServiceInfo()
				log.Printf("EventBus: No handlers for %s event %s (service: %s:%s, category: %s)", 
					event.Type, event.ID, serviceType, serviceID, event.Category)
			} else {
				log.Printf("EventBus: No handlers for %s event %s (category: %s)", 
					event.Type, event.ID, event.Category)
			}
		}
		return nil
	}
	
	// Process with middleware chain
	return eb.processEventWithMiddleware(ctx, event, handlers)
}

// PublishToHandler publishes an event to a specific handler by name
func (eb *EventBus) PublishToHandler(event Event, handlerName string) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	// Find handler by name across all event types
	var targetHandler EventHandler
	for _, handlers := range eb.subscribers {
		for _, handler := range handlers {
			if handler.HandlerName() == handlerName {
				targetHandler = handler
				break
			}
		}
		if targetHandler != nil {
			break
		}
	}
	
	if targetHandler == nil {
		return fmt.Errorf("handler not found: %s", handlerName)
	}
	
	// Check if handler subscribes to this event type
	subscribedEvents := targetHandler.SubscribedEvents()
	isSubscribed := false
	for _, eventType := range subscribedEvents {
		if eventType == event.Type {
			isSubscribed = true
			break
		}
	}
	
	if !isSubscribed {
		return fmt.Errorf("handler %s is not subscribed to event type %s", handlerName, event.Type)
	}
	
	// Process event with handler
	ctx, cancel := context.WithTimeout(eb.ctx, eb.config.HandlerTimeout)
	defer cancel()
	
	start := time.Now()
	err := eb.processEventWithMiddleware(ctx, event, []EventHandler{targetHandler})
	
	// Update metrics
	duration := time.Since(start)
	eb.updateMetrics(err == nil, duration)
	
	if err != nil {
		log.Printf("EventBus: Error processing event %s with handler %s: %v", event.ID, handlerName, err)
	} else if eb.config.LogEvents {
		log.Printf("EventBus: Successfully routed event %s to handler %s", event.ID, handlerName)
	}
	
	return err
}

// worker processes events from both regular and priority queues
func (eb *EventBus) worker(workerID int) {
	defer eb.wg.Done()
	
	log.Printf("EventBus worker %d started", workerID)
	
	for {
		select {
		// Priority queue takes precedence
		case event := <-eb.priorityQueue:
			eb.processEvent(event, true, workerID)
			
		case event := <-eb.eventQueue:
			eb.processEvent(event, false, workerID)
			
		case <-eb.ctx.Done():
			log.Printf("EventBus worker %d stopping", workerID)
			return
		}
	}
}

// processEvent handles the actual event processing
func (eb *EventBus) processEvent(event Event, isPriority bool, workerID int) {
	start := time.Now()
	
	// Get handlers for this event type
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()
	
	if len(handlers) == 0 {
		if eb.config.LogEvents {
			prefixStr := ""
			if isPriority {
				prefixStr = "[PRIORITY] "
			}
			
			if event.IsServiceEvent() {
				serviceType, serviceID := event.GetServiceInfo()
				log.Printf("EventBus: %sNo handlers for %s event %s (service: %s:%s, category: %s)", 
					prefixStr, event.Type, event.ID, serviceType, serviceID, event.Category)
			} else {
				log.Printf("EventBus: %sNo handlers for %s event %s (category: %s)", 
					prefixStr, event.Type, event.ID, event.Category)
			}
		}
		return
	}
	
	// Store event if event store is enabled
	if eb.eventStore != nil {
		if storeErr := eb.eventStore.Store(event); storeErr != nil {
			log.Printf("EventBus: Failed to store event %s: %v", event.ID, storeErr)
		}
	}
	
	// Process event with timeout
	ctx, cancel := context.WithTimeout(eb.ctx, eb.config.HandlerTimeout)
	err := eb.processEventWithMiddleware(ctx, event, handlers)
	cancel()
	
	// Update metrics
	duration := time.Since(start)
	eb.updateMetrics(err == nil, duration)
	
	if err != nil {
		prefixStr := ""
		if isPriority {
			prefixStr = "[PRIORITY] "
		}
		log.Printf("EventBus: %sError processing event %s: %v", prefixStr, event.ID, err)
	} else if eb.config.LogEvents && isPriority {
		log.Printf("EventBus: [PRIORITY] Successfully processed event %s by worker %d", event.ID, workerID)
	}
}

// processEventWithMiddleware processes an event through the middleware chain
func (eb *EventBus) processEventWithMiddleware(ctx context.Context, event Event, handlers []EventHandler) error {
	// Create middleware chain
	next := func(ctx context.Context, event Event) error {
		return eb.processEventHandlers(ctx, event, handlers)
	}
	
	// Apply middleware in reverse order
	for i := len(eb.middleware) - 1; i >= 0; i-- {
		middleware := eb.middleware[i]
		currentNext := next
		next = func(ctx context.Context, event Event) error {
			return middleware.Process(ctx, event, currentNext)
		}
	}
	
	return next(ctx, event)
}

// processEventHandlers processes an event with all its handlers
func (eb *EventBus) processEventHandlers(ctx context.Context, event Event, handlers []EventHandler) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(handlers))
	
	// Process handlers concurrently based on priority
	for _, handler := range handlers {
		wg.Add(1)
		go func(h EventHandler) {
			defer wg.Done()
			
			defer func() {
				if r := recover(); r != nil {
					errChan <- fmt.Errorf("handler %s panicked: %v", h.HandlerName(), r)
				}
			}()
			
			err := eb.executeHandlerWithRetry(ctx, h, event)
			if err != nil {
				errChan <- fmt.Errorf("handler %s failed: %w", h.HandlerName(), err)
			}
		}(handler)
	}
	
	wg.Wait()
	close(errChan)
	
	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("handler errors: %v", errors)
	}
	
	return nil
}

// executeHandlerWithRetry executes a handler with retry logic
func (eb *EventBus) executeHandlerWithRetry(ctx context.Context, handler EventHandler, event Event) error {
	var lastErr error
	
	for attempt := 0; attempt <= eb.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(eb.config.RetryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		err := handler.Handle(ctx, event)
		if err == nil {
			return nil
		}
		
		lastErr = err
		
		if eb.config.LogEvents {
			log.Printf("EventBus: Handler %s attempt %d failed: %v", 
				handler.HandlerName(), attempt+1, err)
		}
	}
	
	return fmt.Errorf("handler failed after %d attempts: %w", eb.config.MaxRetries+1, lastErr)
}

// AddMiddleware adds middleware to the event processing chain
func (eb *EventBus) AddMiddleware(middleware EventMiddleware) {
	eb.middleware = append(eb.middleware, middleware)
}

// GetMetrics returns current event bus metrics
func (eb *EventBus) GetMetrics() EventMetrics {
	eb.metrics.mu.RLock()
	defer eb.metrics.mu.RUnlock()
	
	return *eb.metrics
}

// updateMetrics updates performance metrics
func (eb *EventBus) updateMetrics(success bool, duration time.Duration) {
	eb.metrics.mu.Lock()
	defer eb.metrics.mu.Unlock()
	
	if success {
		eb.metrics.EventsProcessed++
	} else {
		eb.metrics.EventsFailed++
	}
	
	// Update average latency (simple moving average)
	if eb.metrics.EventsProcessed == 1 {
		eb.metrics.AverageLatency = duration
	} else {
		eb.metrics.AverageLatency = (eb.metrics.AverageLatency + duration) / 2
	}
}

// Shutdown gracefully shuts down the event bus
func (eb *EventBus) Shutdown() error {
	log.Printf("EventBus: Starting shutdown...")
	
	// Cancel context to stop workers
	eb.cancel()
	
	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		eb.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Printf("EventBus: All workers stopped")
	case <-time.After(eb.config.ShutdownTimeout):
		log.Printf("EventBus: Shutdown timeout, forcing stop")
	}
	
	// Close event queue
	close(eb.eventQueue)
	
	// Close event store
	if eb.eventStore != nil {
		if err := eb.eventStore.Close(); err != nil {
			log.Printf("EventBus: Error closing event store: %v", err)
		} else {
			log.Printf("EventBus: Event store closed")
		}
	}
	
	// Log final metrics
	metrics := eb.GetMetrics()
	log.Printf("EventBus: Final metrics - Published: %d, Processed: %d, Failed: %d, Uptime: %v",
		metrics.EventsPublished, metrics.EventsProcessed, metrics.EventsFailed,
		time.Since(metrics.StartTime))
	
	return nil
}

// GetSubscriberCount returns the number of subscribers for an event type
func (eb *EventBus) GetSubscriberCount(eventType string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	return len(eb.subscribers[eventType])
}

// GetEventTypes returns all event types that have subscribers
func (eb *EventBus) GetEventTypes() []string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	eventTypes := make([]string, 0, len(eb.subscribers))
	for eventType := range eb.subscribers {
		eventTypes = append(eventTypes, eventType)
	}
	
	return eventTypes
}

// GetEventStore returns the event store if available
func (eb *EventBus) GetEventStore() EventStore {
	return eb.eventStore
}

// SetEventStore sets the event store
func (eb *EventBus) SetEventStore(store EventStore) {
	eb.eventStore = store
	log.Printf("EventBus: Event store configured")
}

// GetMiddleware returns the middleware stack
func (eb *EventBus) GetMiddleware() []EventMiddleware {
	return eb.middleware
}

// PublishServiceEvent publishes an event with service categorization
func (eb *EventBus) PublishServiceEvent(eventType, source, serviceType, serviceID string, data interface{}) error {
	event := NewServiceEvent(eventType, source, serviceType, serviceID, data)
	return eb.Publish(event)
}

// GetEventsByCategory returns all subscribers for a specific category
func (eb *EventBus) GetEventsByCategory(category EventCategory) map[string][]EventHandler {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	categoryEvents := make(map[string][]EventHandler)
	for eventType, handlers := range eb.subscribers {
		// Check if any events of this type match the category
		// This is a simplified check - in practice you might want to maintain a category index
		if len(handlers) > 0 {
			categoryEvents[eventType] = handlers
		}
	}
	return categoryEvents
}

// GetServiceEventTypes returns all event types associated with a service type
func (eb *EventBus) GetServiceEventTypes(serviceType string) []string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	
	var serviceEventTypes []string
	for eventType := range eb.subscribers {
		// Check if this event type is associated with the service type
		// This could be enhanced with better indexing
		if strings.Contains(eventType, serviceType) {
			serviceEventTypes = append(serviceEventTypes, eventType)
		}
	}
	return serviceEventTypes
}

// Backpressure helper methods

// isHighPriorityEvent determines if an event should use priority queue
func (eb *EventBus) isHighPriorityEvent(event Event) bool {
	// Define high priority event types
	highPriorityTypes := []string{
		"system.shutdown",
		"system.emergency",
		"error.critical",
		"security.alert",
		"health.critical",
	}
	
	for _, priorityType := range highPriorityTypes {
		if event.Type == priorityType {
			return true
		}
	}
	
	// Check for priority in metadata
	if priority, exists := event.Metadata["priority"]; exists {
		if priorityStr, ok := priority.(string); ok {
			return priorityStr == "high" || priorityStr == "critical"
		}
	}
	
	return false
}

// isCriticalEvent determines if an event is critical enough for emergency handling
func (eb *EventBus) isCriticalEvent(event Event) bool {
	criticalTypes := []string{
		"system.shutdown",
		"system.emergency",
		"security.alert",
	}
	
	for _, criticalType := range criticalTypes {
		if event.Type == criticalType {
			return true
		}
	}
	
	return false
}

// getQueueUtilization returns current queue utilization as a percentage
func (eb *EventBus) getQueueUtilization() float64 {
	queueLen := float64(len(eb.eventQueue))
	queueCap := float64(cap(eb.eventQueue))
	
	if queueCap == 0 {
		return 0.0
	}
	
	return queueLen / queueCap
}

// updateQueueUtilization updates queue utilization metrics
func (eb *EventBus) updateQueueUtilization() {
	utilization := eb.getQueueUtilization()
	
	eb.metrics.mu.Lock()
	eb.metrics.QueueUtilization = utilization
	eb.metrics.mu.Unlock()
}

// updatePublishMetrics updates metrics for published events
func (eb *EventBus) updatePublishMetrics(event Event, isPriority, isBackpressure bool) {
	eb.metrics.mu.Lock()
	eb.metrics.EventsPublished++
	if isPriority {
		eb.metrics.EventsPriority++
	}
	eb.metrics.mu.Unlock()
	
	if eb.config.LogEvents {
		prefixParts := []string{}
		if isPriority {
			prefixParts = append(prefixParts, "PRIORITY")
		}
		if isBackpressure {
			prefixParts = append(prefixParts, "BACKPRESSURE")
		}
		
		prefix := ""
		if len(prefixParts) > 0 {
			prefix = fmt.Sprintf("[%s] ", strings.Join(prefixParts, ","))
		}
		
		if event.IsServiceEvent() {
			serviceType, serviceID := event.GetServiceInfo()
			log.Printf("EventBus: %sPublished event %s (type: %s, source: %s, service: %s:%s, category: %s)", 
				prefix, event.ID, event.Type, event.Source, serviceType, serviceID, event.Category)
		} else {
			log.Printf("EventBus: %sPublished event %s (type: %s, source: %s, category: %s)", 
				prefix, event.ID, event.Type, event.Source, event.Category)
		}
	}
}

// emergencyPublish bypasses queues and processes event immediately
func (eb *EventBus) emergencyPublish(event Event) error {
	log.Printf("🚨 EventBus: Emergency publish for event %s (both queues full)", event.ID)
	
	// Get handlers immediately
	eb.mu.RLock()
	handlers := eb.subscribers[event.Type]
	eb.mu.RUnlock()
	
	if len(handlers) == 0 {
		log.Printf("EventBus: No handlers for emergency event %s", event.ID)
		return nil
	}
	
	// Process immediately in a separate goroutine to avoid blocking
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), eb.config.HandlerTimeout)
		defer cancel()
		
		start := time.Now()
		err := eb.processEventWithMiddleware(ctx, event, handlers)
		duration := time.Since(start)
		
		eb.updateMetrics(err == nil, duration)
		
		if err != nil {
			log.Printf("⚠️ EventBus: Emergency event %s processing failed: %v", event.ID, err)
		} else {
			log.Printf("✅ EventBus: Emergency event %s processed successfully", event.ID)
		}
	}()
	
	// Update metrics for emergency publish
	eb.metrics.mu.Lock()
	eb.metrics.EventsPublished++
	eb.metrics.mu.Unlock()
	
	return nil
}

// GetBackpressureStats returns backpressure statistics
func (eb *EventBus) GetBackpressureStats() map[string]interface{} {
	eb.metrics.mu.RLock()
	defer eb.metrics.mu.RUnlock()
	
	return map[string]interface{}{
		"backpressure_active":  eb.backpressureActive,
		"last_backpressure":    eb.lastBackpressure,
		"queue_utilization":    eb.metrics.QueueUtilization,
		"queue_length":         len(eb.eventQueue),
		"queue_capacity":       cap(eb.eventQueue),
		"priority_queue_length": len(eb.priorityQueue),
		"priority_queue_capacity": cap(eb.priorityQueue),
		"backpressure_threshold": eb.config.BackpressureThreshold,
		"events_dropped":       eb.metrics.EventsDropped,
		"events_priority":      eb.metrics.EventsPriority,
		"backpressure_events":  eb.metrics.BackpressureEvents,
	}
}