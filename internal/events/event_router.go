package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// EventRouter provides advanced event routing with correlation tracking and response handling
type EventRouter struct {
	eventBus         *EventBus
	correlationMap   map[string]*CorrelationContext
	responseHandlers map[string]ResponseHandler
	subscriptions    map[string]*SubscriptionManager
	mu               sync.RWMutex
	
	// Cleanup
	cleanupInterval  time.Duration
	maxCorrelationTTL time.Duration
}

// CorrelationContext tracks the context for correlated request/response patterns
type CorrelationContext struct {
	ID               string
	RequestEvent     Event
	ResponseChan     chan Event
	CreatedAt        time.Time
	TTL              time.Duration
	ConnectionID     string
	UserID           string
	RequestType      string
	Metadata         map[string]interface{}
}

// ResponseHandler handles responses for specific event types
type ResponseHandler interface {
	HandleResponse(ctx context.Context, requestEvent Event, responseEvent Event) error
	CanHandle(requestType, responseType string) bool
}

// SubscriptionManager manages event subscriptions for connections
type SubscriptionManager struct {
	ConnectionID    string
	UserID          string
	EventFilters    map[string]SubscriptionFilter
	ResponseChan    chan Event
	Active          bool
	CreatedAt       time.Time
	LastActivity    time.Time
}

// SubscriptionFilter defines filtering criteria for event subscriptions
type SubscriptionFilter struct {
	EventTypes       []string
	Categories       []EventCategory
	ServiceTypes     []string
	SourcePatterns   []string
	CustomFilters    map[string]interface{}
	IncludeMetadata  bool
}

// RouterConfig configures the event router
type RouterConfig struct {
	CleanupInterval      time.Duration
	MaxCorrelationTTL    time.Duration
	MaxSubscriptionsPerConnection int
	EnableResponseCaching bool
}

// DefaultRouterConfig returns a default router configuration
func DefaultRouterConfig() *RouterConfig {
	return &RouterConfig{
		CleanupInterval:               5 * time.Minute,
		MaxCorrelationTTL:            30 * time.Minute,
		MaxSubscriptionsPerConnection: 100,
		EnableResponseCaching:         true,
	}
}

// NewEventRouter creates a new event router
func NewEventRouter(eventBus *EventBus, config *RouterConfig) *EventRouter {
	if config == nil {
		config = DefaultRouterConfig()
	}
	
	router := &EventRouter{
		eventBus:          eventBus,
		correlationMap:    make(map[string]*CorrelationContext),
		responseHandlers:  make(map[string]ResponseHandler),
		subscriptions:     make(map[string]*SubscriptionManager),
		cleanupInterval:   config.CleanupInterval,
		maxCorrelationTTL: config.MaxCorrelationTTL,
	}
	
	// Start cleanup routine
	go router.startCleanupRoutine()
	
	return router
}

// RouteRequestWithCorrelation routes a request event and sets up response tracking
func (r *EventRouter) RouteRequestWithCorrelation(ctx context.Context, requestEvent Event, connectionID, userID string) (*CorrelationContext, error) {
	// Generate correlation ID if not present
	correlationID := requestEvent.CorrelationID
	if correlationID == "" {
		correlationID = r.generateCorrelationID()
		requestEvent.CorrelationID = correlationID
	}
	
	// Create correlation context
	correlationCtx := &CorrelationContext{
		ID:           correlationID,
		RequestEvent: requestEvent,
		ResponseChan: make(chan Event, 1),
		CreatedAt:    time.Now(),
		TTL:          r.maxCorrelationTTL,
		ConnectionID: connectionID,
		UserID:       userID,
		RequestType:  requestEvent.Type,
		Metadata:     make(map[string]interface{}),
	}
	
	// Store correlation context
	r.mu.Lock()
	r.correlationMap[correlationID] = correlationCtx
	r.mu.Unlock()
	
	// Publish the request event
	if err := r.eventBus.Publish(requestEvent); err != nil {
		// Clean up correlation context on error
		r.mu.Lock()
		delete(r.correlationMap, correlationID)
		r.mu.Unlock()
		return nil, fmt.Errorf("failed to publish request event: %w", err)
	}
	
	log.Printf("[EventRouter] Routed request event %s with correlation %s", requestEvent.Type, correlationID)
	return correlationCtx, nil
}

// WaitForResponse waits for a correlated response event
func (r *EventRouter) WaitForResponse(ctx context.Context, correlationID string, timeout time.Duration) (Event, error) {
	r.mu.RLock()
	correlationCtx, exists := r.correlationMap[correlationID]
	r.mu.RUnlock()
	
	if !exists {
		return Event{}, fmt.Errorf("correlation ID %s not found", correlationID)
	}
	
	select {
	case responseEvent := <-correlationCtx.ResponseChan:
		// Clean up correlation context after successful response
		r.mu.Lock()
		delete(r.correlationMap, correlationID)
		r.mu.Unlock()
		return responseEvent, nil
	case <-time.After(timeout):
		// Clean up correlation context on timeout
		r.mu.Lock()
		delete(r.correlationMap, correlationID)
		r.mu.Unlock()
		return Event{}, fmt.Errorf("timeout waiting for response to correlation %s", correlationID)
	case <-ctx.Done():
		return Event{}, ctx.Err()
	}
}

// RouteResponse routes a response event to the appropriate correlation context
func (r *EventRouter) RouteResponse(ctx context.Context, responseEvent Event) error {
	correlationID := responseEvent.CorrelationID
	if correlationID == "" {
		// No correlation ID, just publish normally
		return r.eventBus.Publish(responseEvent)
	}
	
	r.mu.RLock()
	correlationCtx, exists := r.correlationMap[correlationID]
	r.mu.RUnlock()
	
	if !exists {
		log.Printf("[EventRouter] No correlation context found for response %s", correlationID)
		// Still publish the event for other subscribers
		return r.eventBus.Publish(responseEvent)
	}
	
	// Send response to correlation context
	select {
	case correlationCtx.ResponseChan <- responseEvent:
		log.Printf("[EventRouter] Routed response event %s to correlation %s", responseEvent.Type, correlationID)
	default:
		log.Printf("[EventRouter] Response channel full for correlation %s", correlationID)
	}
	
	// Also publish the event for other subscribers
	return r.eventBus.Publish(responseEvent)
}

// SubscribeConnection creates a subscription manager for a WebSocket connection
func (r *EventRouter) SubscribeConnection(connectionID, userID string, filters map[string]SubscriptionFilter) (*SubscriptionManager, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if subscription already exists
	if _, exists := r.subscriptions[connectionID]; exists {
		return nil, fmt.Errorf("subscription already exists for connection %s", connectionID)
	}
	
	subscription := &SubscriptionManager{
		ConnectionID: connectionID,
		UserID:       userID,
		EventFilters: filters,
		ResponseChan: make(chan Event, 100), // Buffer events for the connection
		Active:       true,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	
	r.subscriptions[connectionID] = subscription
	
	log.Printf("[EventRouter] Created subscription for connection %s", connectionID)
	return subscription, nil
}

// UnsubscribeConnection removes a connection subscription
func (r *EventRouter) UnsubscribeConnection(connectionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	subscription, exists := r.subscriptions[connectionID]
	if !exists {
		return fmt.Errorf("no subscription found for connection %s", connectionID)
	}
	
	subscription.Active = false
	close(subscription.ResponseChan)
	delete(r.subscriptions, connectionID)
	
	log.Printf("[EventRouter] Removed subscription for connection %s", connectionID)
	return nil
}

// RouteToSubscribers routes events to subscribed connections
func (r *EventRouter) RouteToSubscribers(ctx context.Context, event Event) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for connectionID, subscription := range r.subscriptions {
		if !subscription.Active {
			continue
		}
		
		// Check if event matches any filters
		if r.eventMatchesFilters(event, subscription.EventFilters) {
			select {
			case subscription.ResponseChan <- event:
				subscription.LastActivity = time.Now()
				log.Printf("[EventRouter] Routed event %s to connection %s", event.Type, connectionID)
			default:
				log.Printf("[EventRouter] Subscription channel full for connection %s", connectionID)
			}
		}
	}
	
	return nil
}

// RegisterResponseHandler registers a handler for specific response types
func (r *EventRouter) RegisterResponseHandler(name string, handler ResponseHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.responseHandlers[name] = handler
}

// GetSubscription returns the subscription manager for a connection
func (r *EventRouter) GetSubscription(connectionID string) (*SubscriptionManager, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	subscription, exists := r.subscriptions[connectionID]
	return subscription, exists
}

// GetActiveCorrelations returns the count of active correlations
func (r *EventRouter) GetActiveCorrelations() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.correlationMap)
}

// GetActiveSubscriptions returns the count of active subscriptions
func (r *EventRouter) GetActiveSubscriptions() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.subscriptions)
}

// Private methods

func (r *EventRouter) generateCorrelationID() string {
	return fmt.Sprintf("corr_%d_%s", time.Now().UnixNano(), randomString(8))
}

func (r *EventRouter) eventMatchesFilters(event Event, filters map[string]SubscriptionFilter) bool {
	// If no filters, match all events
	if len(filters) == 0 {
		return true
	}
	
	for _, filter := range filters {
		if r.eventMatchesFilter(event, filter) {
			return true
		}
	}
	
	return false
}

func (r *EventRouter) eventMatchesFilter(event Event, filter SubscriptionFilter) bool {
	// Check event types
	if len(filter.EventTypes) > 0 {
		matched := false
		for _, eventType := range filter.EventTypes {
			if event.Type == eventType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	
	// Check categories
	if len(filter.Categories) > 0 {
		matched := false
		for _, category := range filter.Categories {
			if event.Category == category {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	
	// Check service types
	if len(filter.ServiceTypes) > 0 {
		matched := false
		for _, serviceType := range filter.ServiceTypes {
			if event.ServiceType == serviceType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	
	// Check source patterns
	if len(filter.SourcePatterns) > 0 {
		matched := false
		for _, pattern := range filter.SourcePatterns {
			if event.Source == pattern {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	
	// Check custom filters
	for key, expectedValue := range filter.CustomFilters {
		if actualValue, exists := event.Metadata[key]; !exists || actualValue != expectedValue {
			return false
		}
	}
	
	return true
}

func (r *EventRouter) startCleanupRoutine() {
	ticker := time.NewTicker(r.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			r.cleanupExpiredCorrelations()
			r.cleanupInactiveSubscriptions()
		}
	}
}

func (r *EventRouter) cleanupExpiredCorrelations() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	expired := make([]string, 0)
	
	for correlationID, ctx := range r.correlationMap {
		if now.Sub(ctx.CreatedAt) > ctx.TTL {
			expired = append(expired, correlationID)
		}
	}
	
	for _, correlationID := range expired {
		delete(r.correlationMap, correlationID)
		log.Printf("[EventRouter] Cleaned up expired correlation %s", correlationID)
	}
}

func (r *EventRouter) cleanupInactiveSubscriptions() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	inactive := make([]string, 0)
	
	for connectionID, subscription := range r.subscriptions {
		// Clean up subscriptions inactive for more than 1 hour
		if !subscription.Active || now.Sub(subscription.LastActivity) > time.Hour {
			inactive = append(inactive, connectionID)
		}
	}
	
	for _, connectionID := range inactive {
		subscription := r.subscriptions[connectionID]
		subscription.Active = false
		close(subscription.ResponseChan)
		delete(r.subscriptions, connectionID)
		log.Printf("[EventRouter] Cleaned up inactive subscription for connection %s", connectionID)
	}
}