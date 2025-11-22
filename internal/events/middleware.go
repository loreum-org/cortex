package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// LoggingMiddleware logs all events passing through the bus
type LoggingMiddleware struct {
	LogLevel LogLevel
}

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func NewLoggingMiddleware(level LogLevel) *LoggingMiddleware {
	return &LoggingMiddleware{
		LogLevel: level,
	}
}

func (lm *LoggingMiddleware) Process(ctx context.Context, event Event, next func(context.Context, Event) error) error {
	start := time.Now()
	
	if lm.LogLevel <= LogLevelInfo {
		log.Printf("🔄 Event Processing: %s (type: %s, source: %s, correlation: %s)", 
			event.ID, event.Type, event.Source, event.CorrelationID)
	}
	
	err := next(ctx, event)
	
	duration := time.Since(start)
	
	if err != nil {
		if lm.LogLevel <= LogLevelError {
			log.Printf("❌ Event Failed: %s in %v - Error: %v", 
				event.ID, duration, err)
		}
	} else {
		if lm.LogLevel <= LogLevelDebug {
			log.Printf("✅ Event Completed: %s in %v", event.ID, duration)
		}
	}
	
	return err
}

// MetricsMiddleware collects detailed metrics on event processing
type MetricsMiddleware struct {
	eventMetrics map[string]*EventTypeMetrics
	mu          sync.RWMutex
}

type EventTypeMetrics struct {
	Count           int64         `json:"count"`
	SuccessCount    int64         `json:"success_count"`
	ErrorCount      int64         `json:"error_count"`
	TotalDuration   time.Duration `json:"total_duration"`
	AverageDuration time.Duration `json:"average_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	MaxDuration     time.Duration `json:"max_duration"`
	LastProcessed   time.Time     `json:"last_processed"`
}

func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{
		eventMetrics: make(map[string]*EventTypeMetrics),
	}
}

func (mm *MetricsMiddleware) Process(ctx context.Context, event Event, next func(context.Context, Event) error) error {
	start := time.Now()
	
	err := next(ctx, event)
	
	duration := time.Since(start)
	mm.updateMetrics(event.Type, duration, err == nil)
	
	return err
}

func (mm *MetricsMiddleware) updateMetrics(eventType string, duration time.Duration, success bool) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	metrics := mm.eventMetrics[eventType]
	if metrics == nil {
		metrics = &EventTypeMetrics{
			MinDuration: duration,
			MaxDuration: duration,
		}
		mm.eventMetrics[eventType] = metrics
	}
	
	metrics.Count++
	metrics.TotalDuration += duration
	metrics.AverageDuration = metrics.TotalDuration / time.Duration(metrics.Count)
	metrics.LastProcessed = time.Now()
	
	if success {
		metrics.SuccessCount++
	} else {
		metrics.ErrorCount++
	}
	
	if duration < metrics.MinDuration {
		metrics.MinDuration = duration
	}
	if duration > metrics.MaxDuration {
		metrics.MaxDuration = duration
	}
}

func (mm *MetricsMiddleware) GetMetrics() map[string]*EventTypeMetrics {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	// Return a copy to prevent concurrent access issues
	result := make(map[string]*EventTypeMetrics)
	for k, v := range mm.eventMetrics {
		metricsCopy := *v
		result[k] = &metricsCopy
	}
	
	return result
}

// TracingMiddleware adds distributed tracing capabilities
type TracingMiddleware struct {
	ServiceName string
}

func NewTracingMiddleware(serviceName string) *TracingMiddleware {
	return &TracingMiddleware{
		ServiceName: serviceName,
	}
}

func (tm *TracingMiddleware) Process(ctx context.Context, event Event, next func(context.Context, Event) error) error {
	// Add tracing metadata to event
	if event.Metadata == nil {
		event.Metadata = make(map[string]interface{})
	}
	
	// Add trace information
	event.Metadata["trace_service"] = tm.ServiceName
	event.Metadata["trace_start"] = time.Now()
	
	// Add correlation context if available
	if event.CorrelationID != "" {
		event.Metadata["trace_correlation_id"] = event.CorrelationID
	}
	
	err := next(ctx, event)
	
	// Add completion information
	event.Metadata["trace_end"] = time.Now()
	event.Metadata["trace_success"] = err == nil
	
	return err
}

// ValidationMiddleware validates events before processing
type ValidationMiddleware struct {
	RequiredFields []string
}

func NewValidationMiddleware(requiredFields []string) *ValidationMiddleware {
	return &ValidationMiddleware{
		RequiredFields: requiredFields,
	}
}

func (vm *ValidationMiddleware) Process(ctx context.Context, event Event, next func(context.Context, Event) error) error {
	// Validate required fields
	if event.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	
	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}
	
	if event.Source == "" {
		return fmt.Errorf("event source is required")
	}
	
	if event.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}
	
	// Validate custom required fields
	for _, field := range vm.RequiredFields {
		if event.Metadata == nil {
			return fmt.Errorf("required field %s is missing", field)
		}
		if _, exists := event.Metadata[field]; !exists {
			return fmt.Errorf("required field %s is missing", field)
		}
	}
	
	return next(ctx, event)
}

// RateLimitMiddleware provides basic rate limiting per event type
type RateLimitMiddleware struct {
	limits    map[string]*rateLimiter
	mu        sync.RWMutex
	defaultLimit int
	window    time.Duration
}

type rateLimiter struct {
	count     int
	window    time.Duration
	lastReset time.Time
	limit     int
	mu        sync.Mutex
}

func NewRateLimitMiddleware(defaultLimit int, window time.Duration) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limits:       make(map[string]*rateLimiter),
		defaultLimit: defaultLimit,
		window:       window,
	}
}

func (rlm *RateLimitMiddleware) Process(ctx context.Context, event Event, next func(context.Context, Event) error) error {
	limiter := rlm.getLimiter(event.Type)
	
	if !limiter.allow() {
		return fmt.Errorf("rate limit exceeded for event type %s", event.Type)
	}
	
	return next(ctx, event)
}

func (rlm *RateLimitMiddleware) getLimiter(eventType string) *rateLimiter {
	rlm.mu.RLock()
	limiter, exists := rlm.limits[eventType]
	rlm.mu.RUnlock()
	
	if exists {
		return limiter
	}
	
	rlm.mu.Lock()
	defer rlm.mu.Unlock()
	
	// Double-check after acquiring write lock
	if limiter, exists := rlm.limits[eventType]; exists {
		return limiter
	}
	
	limiter = &rateLimiter{
		window:    rlm.window,
		lastReset: time.Now(),
		limit:     rlm.defaultLimit,
	}
	rlm.limits[eventType] = limiter
	
	return limiter
}

func (rl *rateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// Reset if window has passed
	if now.Sub(rl.lastReset) >= rl.window {
		rl.count = 0
		rl.lastReset = now
	}
	
	if rl.count >= rl.limit {
		return false
	}
	
	rl.count++
	return true
}

// SetLimit sets a custom limit for a specific event type
func (rlm *RateLimitMiddleware) SetLimit(eventType string, limit int) {
	rlm.mu.Lock()
	defer rlm.mu.Unlock()
	
	if limiter, exists := rlm.limits[eventType]; exists {
		limiter.mu.Lock()
		limiter.limit = limit
		limiter.mu.Unlock()
	} else {
		rlm.limits[eventType] = &rateLimiter{
			window:    rlm.window,
			lastReset: time.Now(),
			limit:     limit,
		}
	}
}