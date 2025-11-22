package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// EventAggregator provides intelligent event aggregation to reduce message volume
type EventAggregator struct {
	// Aggregation rules and policies
	aggregationRules   map[string]*AggregationRule
	aggregationPolicies map[string]*AggregationPolicy
	
	// Active aggregation buckets
	buckets            map[string]*AggregationBucket
	
	// Configuration
	maxBuckets         int
	defaultWindow      time.Duration
	maxBucketSize      int
	flushInterval      time.Duration
	
	// Memory management
	maxMemoryBytes     int64
	currentMemoryBytes int64
	memoryPressure     float64
	emergencyMode     bool
	
	// Statistics
	stats              AggregationStats
	
	// Synchronization and lifecycle
	mu                 sync.RWMutex
	ctx                context.Context
	cancel             context.CancelFunc
	isRunning          bool
	
	// Output channel for aggregated events
	outputChan         chan Event
	eventBus           *EventBus
}

// AggregationRule defines how events should be aggregated
type AggregationRule struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	EventTypePattern   string                 `json:"event_type_pattern"`
	SourcePattern      string                 `json:"source_pattern,omitempty"`
	AggregationKey     string                 `json:"aggregation_key"`     // JSON path for grouping
	AggregationMethod  AggregationMethod      `json:"aggregation_method"`
	WindowSize         time.Duration          `json:"window_size"`
	MaxEvents          int                    `json:"max_events"`
	MinEvents          int                    `json:"min_events"`
	Triggers           []AggregationTrigger   `json:"triggers"`
	OutputFormat       AggregationOutputFormat `json:"output_format"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	Priority           int                    `json:"priority"`
	Enabled            bool                   `json:"enabled"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// AggregationPolicy defines system-wide aggregation behavior
type AggregationPolicy struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	DefaultWindowSize  time.Duration          `json:"default_window_size"`
	MaxEventsPerBucket int                    `json:"max_events_per_bucket"`
	CompressionEnabled bool                   `json:"compression_enabled"`
	BatchingEnabled    bool                   `json:"batching_enabled"`
	SamplingRate       float64                `json:"sampling_rate"`
	EmergencySettings  EmergencyAggregation   `json:"emergency_settings"`
	UserSettings       map[string]interface{} `json:"user_settings"`
	SystemSettings     map[string]interface{} `json:"system_settings"`
}

// AggregationMethod defines how events are combined
type AggregationMethod string

const (
	AggregationMethodCount       AggregationMethod = "count"
	AggregationMethodSum         AggregationMethod = "sum"
	AggregationMethodAverage     AggregationMethod = "average"
	AggregationMethodMin         AggregationMethod = "min"
	AggregationMethodMax         AggregationMethod = "max"
	AggregationMethodConcat      AggregationMethod = "concat"
	AggregationMethodMerge       AggregationMethod = "merge"
	AggregationMethodSample      AggregationMethod = "sample"
	AggregationMethodCompress    AggregationMethod = "compress"
)

// AggregationTrigger defines when to flush aggregated events
type AggregationTrigger struct {
	Type      TriggerType `json:"type"`
	Condition string      `json:"condition"`
	Value     interface{} `json:"value"`
}

// TriggerType defines trigger types
type TriggerType string

const (
	TriggerTypeTime     TriggerType = "time"
	TriggerTypeCount    TriggerType = "count"
	TriggerTypeSize     TriggerType = "size"
	TriggerTypeValue    TriggerType = "value"
	TriggerTypePattern  TriggerType = "pattern"
)

// AggregationOutputFormat defines output format
type AggregationOutputFormat struct {
	Type              string                 `json:"type"`
	IncludeOriginals  bool                   `json:"include_originals"`
	IncludeStatistics bool                   `json:"include_statistics"`
	IncludeMetadata   bool                   `json:"include_metadata"`
	CompressionLevel  int                    `json:"compression_level"`
	CustomFields      map[string]interface{} `json:"custom_fields"`
}

// EmergencyAggregation defines emergency aggregation settings
type EmergencyAggregation struct {
	Enabled           bool          `json:"enabled"`
	TriggerThreshold  int           `json:"trigger_threshold"`
	MaxWindowSize     time.Duration `json:"max_window_size"`
	MinWindowSize     time.Duration `json:"min_window_size"`
	AggressiveMode    bool          `json:"aggressive_mode"`
}

// AggregationBucket holds events for aggregation
type AggregationBucket struct {
	ID             string                 `json:"id"`
	Key            string                 `json:"key"`
	Rule           *AggregationRule       `json:"rule"`
	Events         []Event                `json:"events"`
	StartTime      time.Time              `json:"start_time"`
	LastUpdate     time.Time              `json:"last_update"`
	FlushTime      time.Time              `json:"flush_time"`
	Statistics     BucketStatistics       `json:"statistics"`
	Metadata       map[string]interface{} `json:"metadata"`
	mu             sync.RWMutex
}

// BucketStatistics tracks statistics for an aggregation bucket
type BucketStatistics struct {
	EventCount        int                    `json:"event_count"`
	TotalSize         int64                  `json:"total_size"`
	AverageSize       int64                  `json:"average_size"`
	FirstEventTime    time.Time              `json:"first_event_time"`
	LastEventTime     time.Time              `json:"last_event_time"`
	EventTypes        map[string]int         `json:"event_types"`
	Sources           map[string]int         `json:"sources"`
	NumericAggregates map[string]float64     `json:"numeric_aggregates"`
	StringAggregates  map[string]interface{} `json:"string_aggregates"`
}

// AggregationStats tracks overall aggregation statistics
type AggregationStats struct {
	TotalEventsProcessed   int64                      `json:"total_events_processed"`
	TotalEventsAggregated  int64                      `json:"total_events_aggregated"`
	TotalBucketsCreated    int64                      `json:"total_buckets_created"`
	TotalBucketsFlushed    int64                      `json:"total_buckets_flushed"`
	AverageAggregationSize int                        `json:"average_aggregation_size"`
	CompressionRatio       float64                    `json:"compression_ratio"`
	ProcessingLatency      time.Duration              `json:"processing_latency"`
	BucketPerformance      map[string]time.Duration   `json:"bucket_performance"`
	RulePerformance        map[string]RulePerformance `json:"rule_performance"`
	LastUpdate             time.Time                  `json:"last_update"`
}

// RulePerformance tracks performance for aggregation rules
type RulePerformance struct {
	EventsProcessed    int64         `json:"events_processed"`
	BucketsCreated     int64         `json:"buckets_created"`
	AverageLatency     time.Duration `json:"average_latency"`
	CompressionRatio   float64       `json:"compression_ratio"`
	LastActivity       time.Time     `json:"last_activity"`
}

// NewEventAggregator creates a new event aggregator
func NewEventAggregator(eventBus *EventBus) *EventAggregator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &EventAggregator{
		aggregationRules:    make(map[string]*AggregationRule),
		aggregationPolicies: make(map[string]*AggregationPolicy),
		buckets:             make(map[string]*AggregationBucket),
		maxBuckets:          1000,  // Reduced from 10000
		defaultWindow:       5 * time.Second,
		maxBucketSize:       100,   // Reduced from 1000
		flushInterval:       1 * time.Second,
		maxMemoryBytes:      100 * 1024 * 1024, // 100MB limit
		currentMemoryBytes:  0,
		memoryPressure:     0.0,
		emergencyMode:      false,
		ctx:                 ctx,
		cancel:              cancel,
		outputChan:          make(chan Event, 1000),
		eventBus:            eventBus,
		stats: AggregationStats{
			BucketPerformance: make(map[string]time.Duration),
			RulePerformance:   make(map[string]RulePerformance),
		},
	}
}

// Start begins the event aggregator
func (ea *EventAggregator) Start() error {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	
	if ea.isRunning {
		return fmt.Errorf("event aggregator already running")
	}
	
	ea.isRunning = true
	
	// Start flush routine
	go ea.flushRoutine()
	
	// Start output routine
	go ea.outputRoutine()
	
	// Start cleanup routine
	go ea.cleanupRoutine()
	
	// Start memory monitoring routine
	go ea.memoryMonitoringRoutine()
	
	log.Printf("📊 Event Aggregator: Started with %.1fMB memory limit", float64(ea.maxMemoryBytes)/(1024*1024))
	return nil
}

// Stop stops the event aggregator
func (ea *EventAggregator) Stop() error {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	
	if !ea.isRunning {
		return nil
	}
	
	ea.cancel()
	
	// Flush all remaining buckets
	ea.flushAllBuckets()
	
	ea.isRunning = false
	
	log.Printf("📊 Event Aggregator: Stopped")
	return nil
}

// AggregateEvent processes an event for aggregation
func (ea *EventAggregator) AggregateEvent(event Event) error {
	start := time.Now()
	
	// Find matching aggregation rule
	rule := ea.findMatchingRule(event)
	if rule == nil {
		// No aggregation rule, pass through
		return ea.eventBus.Publish(event)
	}
	
	// Generate aggregation key
	key := ea.generateAggregationKey(event, rule)
	
	// Get or create bucket
	bucket := ea.getOrCreateBucket(key, rule)
	
	// Add event to bucket
	if err := ea.addEventToBucket(bucket, event); err != nil {
		return fmt.Errorf("failed to add event to bucket: %w", err)
	}
	
	// Check if bucket should be flushed (safe check)
	if ea.shouldFlushBucket(bucket) {
		// Remove bucket from map to prevent concurrent access
		ea.mu.Lock()
		if _, exists := ea.buckets[key]; exists {
			bucketMemory := ea.estimateBucketMemory(bucket)
			ea.currentMemoryBytes -= bucketMemory
			if ea.currentMemoryBytes < 0 {
				ea.currentMemoryBytes = 0
			}
			delete(ea.buckets, key)
			ea.mu.Unlock()
			
			// Flush bucket safely
			go ea.flushBucketSafe(bucket)
		} else {
			ea.mu.Unlock()
			// Bucket was already removed by another goroutine
		}
	}
	
	// Update statistics
	ea.updateStats(rule, time.Since(start))
	
	return nil
}

// AddAggregationRule adds a new aggregation rule
func (ea *EventAggregator) AddAggregationRule(rule AggregationRule) error {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	
	if err := ea.validateAggregationRule(rule); err != nil {
		return fmt.Errorf("invalid aggregation rule: %w", err)
	}
	
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	
	ea.aggregationRules[rule.ID] = &rule
	
	log.Printf("📊 Event Aggregator: Added aggregation rule '%s' (%s)", rule.Name, rule.ID)
	return nil
}

// RemoveAggregationRule removes an aggregation rule
func (ea *EventAggregator) RemoveAggregationRule(ruleID string) error {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	
	if _, exists := ea.aggregationRules[ruleID]; !exists {
		return fmt.Errorf("aggregation rule not found: %s", ruleID)
	}
	
	delete(ea.aggregationRules, ruleID)
	
	// Flush and remove related buckets
	ea.flushBucketsForRule(ruleID)
	
	log.Printf("📊 Event Aggregator: Removed aggregation rule %s", ruleID)
	return nil
}

// findMatchingRule finds the best matching aggregation rule for an event
func (ea *EventAggregator) findMatchingRule(event Event) *AggregationRule {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	
	var bestRule *AggregationRule
	highestPriority := math.MaxInt32
	
	for _, rule := range ea.aggregationRules {
		if !rule.Enabled {
			continue
		}
		
		if ea.matchesRule(event, rule) && rule.Priority < highestPriority {
			bestRule = rule
			highestPriority = rule.Priority
		}
	}
	
	return bestRule
}

// matchesRule checks if an event matches an aggregation rule
func (ea *EventAggregator) matchesRule(event Event, rule *AggregationRule) bool {
	// Check event type pattern
	if rule.EventTypePattern != "" && !ea.matchesPattern(event.Type, rule.EventTypePattern) {
		return false
	}
	
	// Check source pattern
	if rule.SourcePattern != "" && !ea.matchesPattern(event.Source, rule.SourcePattern) {
		return false
	}
	
	return true
}

// matchesPattern checks if a value matches a pattern (supports wildcards)
func (ea *EventAggregator) matchesPattern(value, pattern string) bool {
	if pattern == "*" {
		return true
	}
	
	// Simple pattern matching - could be enhanced with regex
	return value == pattern
}

// generateAggregationKey generates a key for event aggregation
func (ea *EventAggregator) generateAggregationKey(event Event, rule *AggregationRule) string {
	baseKey := fmt.Sprintf("%s:%s", rule.ID, event.Type)
	
	if rule.AggregationKey != "" {
		// Extract value from event data using the aggregation key
		if value := ea.extractValueFromEvent(event, rule.AggregationKey); value != nil {
			baseKey = fmt.Sprintf("%s:%v", baseKey, value)
		}
	}
	
	// Add time window to key for time-based aggregation
	if rule.WindowSize > 0 {
		windowStart := time.Now().Truncate(rule.WindowSize)
		baseKey = fmt.Sprintf("%s:%d", baseKey, windowStart.Unix())
	}
	
	return baseKey
}

// extractValueFromEvent extracts a value from event using a JSON path
func (ea *EventAggregator) extractValueFromEvent(event Event, path string) interface{} {
	// Simple path extraction - could be enhanced with proper JSON path
	if dataMap, ok := event.Data.(map[string]interface{}); ok {
		return dataMap[path]
	}
	return nil
}

// getOrCreateBucket gets an existing bucket or creates a new one (thread-safe)
func (ea *EventAggregator) getOrCreateBucket(key string, rule *AggregationRule) *AggregationBucket {
	// First try to get existing bucket with read lock (common case)
	ea.mu.RLock()
	if bucket, exists := ea.buckets[key]; exists {
		// Bucket exists, return it immediately
		ea.mu.RUnlock()
		return bucket
	}
	ea.mu.RUnlock()
	
	// Need to create bucket, acquire write lock
	ea.mu.Lock()
	defer ea.mu.Unlock()
	
	// Double-check after acquiring write lock (another goroutine might have created it)
	if bucket, exists := ea.buckets[key]; exists {
		return bucket
	}
	
	// Check memory pressure before creating new bucket
	ea.updateMemoryPressure()
	
	// Handle memory pressure
	if ea.memoryPressure > 0.8 || ea.currentMemoryBytes > ea.maxMemoryBytes {
		ea.handleMemoryPressureUnsafe() // Call unsafe version since we already hold the lock
	}
	
	// Check bucket limit (adaptive based on memory pressure)
	maxBucketsAdjusted := ea.getAdjustedMaxBuckets()
	if len(ea.buckets) >= maxBucketsAdjusted {
		// Force flush older buckets to make space
		ea.forceFlushOldestBucketsUnsafe(10) // Call unsafe version since we already hold the lock
	}
	
	// Create new bucket
	bucket := &AggregationBucket{
		ID:        generateBucketID(),
		Key:       key,
		Rule:      rule,
		Events:    make([]Event, 0),
		StartTime: time.Now(),
		LastUpdate: time.Now(),
		FlushTime:  time.Now().Add(rule.WindowSize),
		Statistics: BucketStatistics{
			EventTypes:        make(map[string]int),
			Sources:           make(map[string]int),
			NumericAggregates: make(map[string]float64),
			StringAggregates:  make(map[string]interface{}),
		},
		Metadata: make(map[string]interface{}),
	}
	
	// Estimate and track memory usage
	bucketMemory := ea.estimateBucketMemory(bucket)
	ea.currentMemoryBytes += bucketMemory
	
	ea.buckets[key] = bucket
	ea.stats.TotalBucketsCreated++
	
	return bucket
}

// addEventToBucket adds an event to an aggregation bucket
func (ea *EventAggregator) addEventToBucket(bucket *AggregationBucket, event Event) error {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	
	// Get adaptive bucket size limit based on memory pressure
	adaptiveBucketSize := ea.getAdaptiveBucketSize()
	
	// Check bucket size limit
	if len(bucket.Events) >= adaptiveBucketSize {
		// Force flush bucket if it's at capacity
		go ea.flushBucket(bucket)
		return fmt.Errorf("bucket size limit exceeded, flushing bucket")
	}
	
	// Calculate event size before adding
	eventSize := int64(0)
	if eventData, err := json.Marshal(event); err == nil {
		eventSize = int64(len(eventData))
		
		// Check if adding this event would exceed memory limit
		if ea.currentMemoryBytes + eventSize > ea.maxMemoryBytes {
			// Trigger emergency flush
			go ea.emergencyFlush()
			return fmt.Errorf("memory limit would be exceeded, triggering emergency flush")
		}
	}
	
	// Add event
	bucket.Events = append(bucket.Events, event)
	bucket.LastUpdate = time.Now()
	
	// Update memory tracking
	ea.mu.Lock()
	ea.currentMemoryBytes += eventSize
	ea.mu.Unlock()
	
	// Update statistics
	bucket.Statistics.EventCount++
	bucket.Statistics.EventTypes[event.Type]++
	bucket.Statistics.Sources[event.Source]++
	
	if bucket.Statistics.FirstEventTime.IsZero() {
		bucket.Statistics.FirstEventTime = event.Timestamp
	}
	bucket.Statistics.LastEventTime = event.Timestamp
	
	// Update size statistics
	bucket.Statistics.TotalSize += eventSize
	bucket.Statistics.AverageSize = bucket.Statistics.TotalSize / int64(bucket.Statistics.EventCount)
	
	// Update numeric aggregates based on aggregation method
	ea.updateNumericAggregates(bucket, event)
	
	ea.stats.TotalEventsProcessed++
	
	return nil
}

// updateNumericAggregates updates numeric aggregates for the bucket
func (ea *EventAggregator) updateNumericAggregates(bucket *AggregationBucket, event Event) {
	if bucket.Rule.AggregationMethod == AggregationMethodSum ||
		bucket.Rule.AggregationMethod == AggregationMethodAverage ||
		bucket.Rule.AggregationMethod == AggregationMethodMin ||
		bucket.Rule.AggregationMethod == AggregationMethodMax {
		
		// Extract numeric values from event data
		if dataMap, ok := event.Data.(map[string]interface{}); ok {
			for key, value := range dataMap {
				if numValue, ok := value.(float64); ok {
					switch bucket.Rule.AggregationMethod {
					case AggregationMethodSum, AggregationMethodAverage:
						bucket.Statistics.NumericAggregates[key] += numValue
					case AggregationMethodMin:
						if existing, exists := bucket.Statistics.NumericAggregates[key]; !exists || numValue < existing {
							bucket.Statistics.NumericAggregates[key] = numValue
						}
					case AggregationMethodMax:
						if existing, exists := bucket.Statistics.NumericAggregates[key]; !exists || numValue > existing {
							bucket.Statistics.NumericAggregates[key] = numValue
						}
					}
				}
			}
		}
	}
}

// shouldFlushBucket determines if a bucket should be flushed (thread-safe)
func (ea *EventAggregator) shouldFlushBucket(bucket *AggregationBucket) bool {
	// Try to acquire read lock with timeout to avoid deadlocks
	lockAcquired := make(chan bool, 1)
	go func() {
		bucket.mu.RLock()
		lockAcquired <- true
	}()
	
	select {
	case <-lockAcquired:
		defer bucket.mu.RUnlock()
	case <-time.After(100 * time.Millisecond):
		// Couldn't acquire lock within timeout, assume bucket is being flushed
		return false
	}
	
	now := time.Now()
	
	// Check time trigger
	if now.After(bucket.FlushTime) {
		return true
	}
	
	// Check count trigger
	if bucket.Rule.MaxEvents > 0 && len(bucket.Events) >= bucket.Rule.MaxEvents {
		return true
	}
	
	// Check custom triggers
	for _, trigger := range bucket.Rule.Triggers {
		if ea.evaluateTriggerSafe(bucket, trigger) {
			return true
		}
	}
	
	return false
}

// evaluateTrigger evaluates a custom trigger
func (ea *EventAggregator) evaluateTrigger(bucket *AggregationBucket, trigger AggregationTrigger) bool {
	switch trigger.Type {
	case TriggerTypeCount:
		if threshold, ok := trigger.Value.(float64); ok {
			return len(bucket.Events) >= int(threshold)
		}
	case TriggerTypeSize:
		if threshold, ok := trigger.Value.(float64); ok {
			return bucket.Statistics.TotalSize >= int64(threshold)
		}
	case TriggerTypeTime:
		if duration, ok := trigger.Value.(string); ok {
			if d, err := time.ParseDuration(duration); err == nil {
				return time.Since(bucket.StartTime) >= d
			}
		}
	}
	return false
}

// evaluateTriggerSafe evaluates a custom trigger safely (assumes bucket lock is held)
func (ea *EventAggregator) evaluateTriggerSafe(bucket *AggregationBucket, trigger AggregationTrigger) bool {
	// This method assumes the caller already holds the bucket.mu.RLock()
	return ea.evaluateTrigger(bucket, trigger)
}

// flushBucket flushes an aggregation bucket
func (ea *EventAggregator) flushBucket(bucket *AggregationBucket) {
	start := time.Now()
	
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	
	if len(bucket.Events) < bucket.Rule.MinEvents {
		// Don't flush if below minimum events
		return
	}
	
	// Create aggregated event
	aggregatedEvent := ea.createAggregatedEvent(bucket)
	
	// Send to output channel
	select {
	case ea.outputChan <- aggregatedEvent:
		// Update statistics
		ea.stats.TotalEventsAggregated += int64(len(bucket.Events))
		ea.stats.TotalBucketsFlushed++
		
		// Calculate compression ratio
		originalSize := bucket.Statistics.TotalSize
		aggregatedData, _ := json.Marshal(aggregatedEvent)
		aggregatedSize := int64(len(aggregatedData))
		
		compressionRatio := float64(1.0) // Default to no compression
		if originalSize > 0 {
			compressionRatio = float64(aggregatedSize) / float64(originalSize)
			ea.updateCompressionRatio(compressionRatio)
		}
		
		// Remove bucket and update memory tracking
		ea.mu.Lock()
		if _, exists := ea.buckets[bucket.Key]; exists {
			bucketMemory := ea.estimateBucketMemory(bucket)
			ea.currentMemoryBytes -= bucketMemory
			if ea.currentMemoryBytes < 0 {
				ea.currentMemoryBytes = 0
			}
			delete(ea.buckets, bucket.Key)
		}
		ea.mu.Unlock()
		
		// Update performance stats
		flushTime := time.Since(start)
		ea.mu.Lock()
		ea.stats.BucketPerformance[bucket.ID] = flushTime
		ea.mu.Unlock()
		
		log.Printf("📊 Event Aggregator: Flushed bucket %s with %d events (compression: %.2f%%)",
			bucket.ID, len(bucket.Events), (1.0-compressionRatio)*100)
		
	default:
		log.Printf("⚠️ Event Aggregator: Output channel full, dropping aggregated event")
	}
}

// flushBucketSafe flushes a bucket safely without accessing the buckets map
func (ea *EventAggregator) flushBucketSafe(bucket *AggregationBucket) {
	start := time.Now()
	
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	
	if len(bucket.Events) < bucket.Rule.MinEvents {
		// Don't flush if below minimum events
		return
	}
	
	// Create aggregated event
	aggregatedEvent := ea.createAggregatedEvent(bucket)
	
	// Send to output channel
	select {
	case ea.outputChan <- aggregatedEvent:
		// Update statistics
		ea.mu.Lock()
		ea.stats.TotalEventsAggregated += int64(len(bucket.Events))
		ea.stats.TotalBucketsFlushed++
		ea.mu.Unlock()
		
		// Calculate compression ratio
		originalSize := bucket.Statistics.TotalSize
		aggregatedData, _ := json.Marshal(aggregatedEvent)
		aggregatedSize := int64(len(aggregatedData))
		
		compressionRatio := float64(1.0) // Default to no compression
		if originalSize > 0 {
			compressionRatio = float64(aggregatedSize) / float64(originalSize)
			ea.updateCompressionRatio(compressionRatio)
		}
		
		// Update performance stats
		flushTime := time.Since(start)
		ea.mu.Lock()
		ea.stats.BucketPerformance[bucket.ID] = flushTime
		ea.mu.Unlock()
		
		log.Printf("📊 Event Aggregator: Safe flushed bucket %s with %d events (compression: %.2f%%)",
			bucket.ID, len(bucket.Events), (1.0-compressionRatio)*100)
		
	default:
		log.Printf("⚠️ Event Aggregator: Output channel full, dropping aggregated event from safe flush")
	}
}

// createAggregatedEvent creates an aggregated event from a bucket
func (ea *EventAggregator) createAggregatedEvent(bucket *AggregationBucket) Event {
	aggregatedEvent := Event{
		ID:            generateEventID(),
		Type:          fmt.Sprintf("%s.aggregated", bucket.Events[0].Type),
		Source:        "event_aggregator",
		Target:        bucket.Events[0].Target,
		CorrelationID: bucket.Events[0].CorrelationID,
		Timestamp:     time.Now(),
		Metadata: map[string]interface{}{
			"aggregation_rule":  bucket.Rule.ID,
			"bucket_id":         bucket.ID,
			"event_count":       len(bucket.Events),
			"aggregation_method": bucket.Rule.AggregationMethod,
			"time_window":       bucket.Rule.WindowSize.String(),
			"start_time":        bucket.StartTime,
			"end_time":          bucket.LastUpdate,
		},
	}
	
	// Create aggregated data based on method
	switch bucket.Rule.AggregationMethod {
	case AggregationMethodCount:
		aggregatedEvent.Data = map[string]interface{}{
			"count":      len(bucket.Events),
			"statistics": bucket.Statistics,
		}
		
	case AggregationMethodSum, AggregationMethodAverage:
		data := map[string]interface{}{
			"aggregates": bucket.Statistics.NumericAggregates,
			"statistics": bucket.Statistics,
		}
		
		if bucket.Rule.AggregationMethod == AggregationMethodAverage {
			averages := make(map[string]float64)
			for key, sum := range bucket.Statistics.NumericAggregates {
				averages[key] = sum / float64(len(bucket.Events))
			}
			data["averages"] = averages
		}
		
		aggregatedEvent.Data = data
		
	case AggregationMethodMerge:
		// Merge all event data into a single structure
		mergedData := make(map[string]interface{})
		events := make([]interface{}, len(bucket.Events))
		
		for i, event := range bucket.Events {
			events[i] = event.Data
		}
		
		mergedData["events"] = events
		mergedData["statistics"] = bucket.Statistics
		aggregatedEvent.Data = mergedData
		
	case AggregationMethodSample:
		// Sample a subset of events
		sampleSize := int(math.Min(10, float64(len(bucket.Events))))
		sampledEvents := make([]Event, sampleSize)
		
		// Simple sampling - could be enhanced with better algorithms
		step := len(bucket.Events) / sampleSize
		for i := 0; i < sampleSize; i++ {
			sampledEvents[i] = bucket.Events[i*step]
		}
		
		aggregatedEvent.Data = map[string]interface{}{
			"sample":     sampledEvents,
			"sample_size": sampleSize,
			"total_count": len(bucket.Events),
			"statistics": bucket.Statistics,
		}
		
	case AggregationMethodCompress:
		// Compress event data
		compressedData := ea.compressEventData(bucket.Events)
		aggregatedEvent.Data = map[string]interface{}{
			"compressed_data": compressedData,
			"compression_info": map[string]interface{}{
				"original_count": len(bucket.Events),
				"compression_method": "simple",
			},
			"statistics": bucket.Statistics,
		}
		
	default:
		// Default: include all events
		aggregatedEvent.Data = map[string]interface{}{
			"events":     bucket.Events,
			"statistics": bucket.Statistics,
		}
	}
	
	// Add custom fields from output format
	if bucket.Rule.OutputFormat.IncludeOriginals {
		aggregatedEvent.Metadata["original_events"] = bucket.Events
	}
	
	if bucket.Rule.OutputFormat.IncludeStatistics {
		aggregatedEvent.Metadata["detailed_statistics"] = bucket.Statistics
	}
	
	return aggregatedEvent
}

// compressEventData compresses event data
func (ea *EventAggregator) compressEventData(events []Event) map[string]interface{} {
	// Simple compression: group by event type and extract common fields
	compressed := make(map[string]interface{})
	eventsByType := make(map[string][]Event)
	
	for _, event := range events {
		eventsByType[event.Type] = append(eventsByType[event.Type], event)
	}
	
	for eventType, typeEvents := range eventsByType {
		// Extract common fields
		commonFields := make(map[string]interface{})
		uniqueFields := make([]map[string]interface{}, 0)
		
		if len(typeEvents) > 0 {
			// Use first event as template
			if dataMap, ok := typeEvents[0].Data.(map[string]interface{}); ok {
				for key, value := range dataMap {
					isCommon := true
					for _, event := range typeEvents[1:] {
						if eventDataMap, ok := event.Data.(map[string]interface{}); ok {
							if eventDataMap[key] != value {
								isCommon = false
								break
							}
						}
					}
					
					if isCommon {
						commonFields[key] = value
					}
				}
				
				// Extract unique fields
				for _, event := range typeEvents {
					if eventDataMap, ok := event.Data.(map[string]interface{}); ok {
						uniqueData := make(map[string]interface{})
						for key, value := range eventDataMap {
							if _, isCommon := commonFields[key]; !isCommon {
								uniqueData[key] = value
							}
						}
						if len(uniqueData) > 0 {
							uniqueFields = append(uniqueFields, uniqueData)
						}
					}
				}
			}
		}
		
		compressed[eventType] = map[string]interface{}{
			"count":        len(typeEvents),
			"common_fields": commonFields,
			"unique_fields": uniqueFields,
		}
	}
	
	return compressed
}

// flushRoutine periodically flushes buckets
func (ea *EventAggregator) flushRoutine() {
	ticker := time.NewTicker(ea.flushInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ea.flushExpiredBuckets()
		case <-ea.ctx.Done():
			return
		}
	}
}

// flushExpiredBuckets flushes buckets that have exceeded their time window
func (ea *EventAggregator) flushExpiredBuckets() {
	type bucketInfo struct {
		bucket *AggregationBucket
		key    string
	}
	
	ea.mu.Lock()
	expiredBuckets := make([]bucketInfo, 0)
	
	for key, bucket := range ea.buckets {
		if ea.shouldFlushBucket(bucket) {
			expiredBuckets = append(expiredBuckets, bucketInfo{
				bucket: bucket,
				key:    key,
			})
			// Remove bucket from map immediately to prevent race conditions
			delete(ea.buckets, key)
			
			// Update memory tracking
			bucketMemory := ea.estimateBucketMemory(bucket)
			ea.currentMemoryBytes -= bucketMemory
			if ea.currentMemoryBytes < 0 {
				ea.currentMemoryBytes = 0
			}
		}
	}
	ea.mu.Unlock()
	
	// Flush expired buckets safely
	for _, bucketInfo := range expiredBuckets {
		go ea.flushBucketSafe(bucketInfo.bucket)
	}
}

// outputRoutine handles output of aggregated events
func (ea *EventAggregator) outputRoutine() {
	for {
		select {
		case aggregatedEvent := <-ea.outputChan:
			if err := ea.eventBus.Publish(aggregatedEvent); err != nil {
				log.Printf("❌ Event Aggregator: Failed to publish aggregated event: %v", err)
			}
		case <-ea.ctx.Done():
			return
		}
	}
}

// cleanupRoutine periodically cleans up old buckets
func (ea *EventAggregator) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ea.cleanupOldBuckets()
		case <-ea.ctx.Done():
			return
		}
	}
}

// cleanupOldBuckets removes old and stale buckets
func (ea *EventAggregator) cleanupOldBuckets() {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	
	cutoff := time.Now().Add(-10 * time.Minute)
	
	for key, bucket := range ea.buckets {
		if bucket.LastUpdate.Before(cutoff) {
			delete(ea.buckets, key)
			log.Printf("🧹 Event Aggregator: Cleaned up stale bucket %s", bucket.ID)
		}
	}
}

// Helper methods

func (ea *EventAggregator) removeOldestBucket() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, bucket := range ea.buckets {
		if oldestTime.IsZero() || bucket.StartTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = bucket.StartTime
		}
	}
	
	if oldestKey != "" {
		// Flush the oldest bucket before removing
		if bucket := ea.buckets[oldestKey]; bucket != nil {
			go ea.flushBucket(bucket)
		}
		delete(ea.buckets, oldestKey)
	}
}

func (ea *EventAggregator) flushAllBuckets() {
	ea.mu.RLock()
	buckets := make([]*AggregationBucket, 0, len(ea.buckets))
	for _, bucket := range ea.buckets {
		buckets = append(buckets, bucket)
	}
	ea.mu.RUnlock()
	
	for _, bucket := range buckets {
		ea.flushBucket(bucket)
	}
}

func (ea *EventAggregator) flushBucketsForRule(ruleID string) {
	ea.mu.RLock()
	bucketsToFlush := make([]*AggregationBucket, 0)
	for _, bucket := range ea.buckets {
		if bucket.Rule.ID == ruleID {
			bucketsToFlush = append(bucketsToFlush, bucket)
		}
	}
	ea.mu.RUnlock()
	
	for _, bucket := range bucketsToFlush {
		ea.flushBucket(bucket)
	}
}

func (ea *EventAggregator) validateAggregationRule(rule AggregationRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	
	if rule.WindowSize <= 0 {
		return fmt.Errorf("window size must be positive")
	}
	
	return nil
}

func (ea *EventAggregator) updateStats(rule *AggregationRule, latency time.Duration) {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	
	// Update overall latency
	if ea.stats.ProcessingLatency == 0 {
		ea.stats.ProcessingLatency = latency
	} else {
		ea.stats.ProcessingLatency = (ea.stats.ProcessingLatency + latency) / 2
	}
	
	// Update rule performance
	rulePerf := ea.stats.RulePerformance[rule.ID]
	rulePerf.EventsProcessed++
	rulePerf.LastActivity = time.Now()
	
	if rulePerf.AverageLatency == 0 {
		rulePerf.AverageLatency = latency
	} else {
		rulePerf.AverageLatency = (rulePerf.AverageLatency + latency) / 2
	}
	
	ea.stats.RulePerformance[rule.ID] = rulePerf
	ea.stats.LastUpdate = time.Now()
}

func (ea *EventAggregator) updateCompressionRatio(ratio float64) {
	if ea.stats.CompressionRatio == 0 {
		ea.stats.CompressionRatio = ratio
	} else {
		ea.stats.CompressionRatio = (ea.stats.CompressionRatio + ratio) / 2
	}
}

func generateBucketID() string {
	return fmt.Sprintf("bucket_%d", time.Now().UnixNano())
}

// GetAggregationStats returns current aggregation statistics
func (ea *EventAggregator) GetAggregationStats() AggregationStats {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	return ea.stats
}

// GetActiveBuckets returns information about active aggregation buckets
func (ea *EventAggregator) GetActiveBuckets() map[string]*AggregationBucket {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	
	result := make(map[string]*AggregationBucket)
	for key, bucket := range ea.buckets {
		result[key] = bucket
	}
	return result
}

// Memory management methods

// updateMemoryPressure calculates current memory pressure
func (ea *EventAggregator) updateMemoryPressure() {
	if ea.maxMemoryBytes <= 0 {
		ea.memoryPressure = 0.0
		return
	}
	
	ea.memoryPressure = float64(ea.currentMemoryBytes) / float64(ea.maxMemoryBytes)
	
	// Enter emergency mode if memory pressure is critical
	if ea.memoryPressure > 0.95 {
		ea.emergencyMode = true
		log.Printf("⚠️ Event Aggregator: Entering emergency mode (%.1f%% memory usage)", ea.memoryPressure*100)
	} else if ea.memoryPressure < 0.7 && ea.emergencyMode {
		ea.emergencyMode = false
		log.Printf("✅ Event Aggregator: Exiting emergency mode (%.1f%% memory usage)", ea.memoryPressure*100)
	}
}

// handleMemoryPressure takes actions to reduce memory usage
func (ea *EventAggregator) handleMemoryPressure() {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	ea.handleMemoryPressureUnsafe()
}

// handleMemoryPressureUnsafe is the unsafe version that doesn't acquire locks
func (ea *EventAggregator) handleMemoryPressureUnsafe() {
	if ea.memoryPressure > 0.9 {
		// Critical pressure: force flush 50% of buckets
		numToFlush := len(ea.buckets) / 2
		ea.forceFlushOldestBucketsUnsafe(numToFlush)
		log.Printf("🚨 Event Aggregator: Critical memory pressure, flushed %d buckets", numToFlush)
	} else if ea.memoryPressure > 0.8 {
		// High pressure: force flush 25% of buckets
		numToFlush := len(ea.buckets) / 4
		ea.forceFlushOldestBucketsUnsafe(numToFlush)
		log.Printf("⚠️ Event Aggregator: High memory pressure, flushed %d buckets", numToFlush)
	}
}

// getAdjustedMaxBuckets returns adjusted max buckets based on memory pressure
func (ea *EventAggregator) getAdjustedMaxBuckets() int {
	baseMax := ea.maxBuckets
	
	if ea.emergencyMode {
		// Emergency mode: reduce to 10% of normal
		return baseMax / 10
	} else if ea.memoryPressure > 0.8 {
		// High pressure: reduce to 25% of normal
		return baseMax / 4
	} else if ea.memoryPressure > 0.6 {
		// Medium pressure: reduce to 50% of normal
		return baseMax / 2
	}
	
	return baseMax
}

// getAdaptiveBucketSize returns bucket size adjusted for memory pressure
func (ea *EventAggregator) getAdaptiveBucketSize() int {
	baseSize := ea.maxBucketSize
	
	if ea.emergencyMode {
		// Emergency mode: very small buckets
		return 10
	} else if ea.memoryPressure > 0.8 {
		// High pressure: small buckets
		return baseSize / 4
	} else if ea.memoryPressure > 0.6 {
		// Medium pressure: reduced buckets
		return baseSize / 2
	}
	
	return baseSize
}

// estimateBucketMemory estimates memory usage of a bucket
func (ea *EventAggregator) estimateBucketMemory(bucket *AggregationBucket) int64 {
	// Base bucket overhead (structs, maps, slices)
	baseOverhead := int64(1024) // 1KB base overhead
	
	// Event storage estimation
	eventCapacity := cap(bucket.Events)
	eventMemory := int64(eventCapacity) * 512 // Estimate 512 bytes per event slot
	
	// Statistics maps overhead
	statsOverhead := int64(256) // Estimate for maps and other data
	
	return baseOverhead + eventMemory + statsOverhead
}

// forceFlushOldestBuckets force flushes the N oldest buckets
func (ea *EventAggregator) forceFlushOldestBuckets(count int) {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	ea.forceFlushOldestBucketsUnsafe(count)
}

// forceFlushOldestBucketsUnsafe is the unsafe version that doesn't acquire locks
func (ea *EventAggregator) forceFlushOldestBucketsUnsafe(count int) {
	if count <= 0 || len(ea.buckets) == 0 {
		return
	}
	
	// Find oldest buckets
	type bucketAge struct {
		key    string
		bucket *AggregationBucket
		age    time.Time
	}
	
	bucketAges := make([]bucketAge, 0, len(ea.buckets))
	for key, bucket := range ea.buckets {
		bucketAges = append(bucketAges, bucketAge{
			key:    key,
			bucket: bucket,
			age:    bucket.StartTime,
		})
	}
	
	// Sort by age (oldest first)
	for i := 0; i < len(bucketAges); i++ {
		for j := i + 1; j < len(bucketAges); j++ {
			if bucketAges[i].age.After(bucketAges[j].age) {
				bucketAges[i], bucketAges[j] = bucketAges[j], bucketAges[i]
			}
		}
	}
	
	// Flush oldest buckets
	flushCount := count
	if flushCount > len(bucketAges) {
		flushCount = len(bucketAges)
	}
	
	for i := 0; i < flushCount; i++ {
		bucketToFlush := bucketAges[i]
		
		// Update memory tracking before flushing
		bucketMemory := ea.estimateBucketMemory(bucketToFlush.bucket)
		ea.currentMemoryBytes -= bucketMemory
		if ea.currentMemoryBytes < 0 {
			ea.currentMemoryBytes = 0
		}
		
		// Remove from buckets map to prevent concurrent access
		delete(ea.buckets, bucketToFlush.key)
		
		// Flush the bucket asynchronously
		go ea.flushBucketSafe(bucketToFlush.bucket)
	}
}

// emergencyFlush performs emergency memory cleanup
func (ea *EventAggregator) emergencyFlush() {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	
	log.Printf("🚨 Event Aggregator: Emergency flush triggered (%.1f%% memory usage)", ea.memoryPressure*100)
	
	// Flush all buckets immediately
	for key, bucket := range ea.buckets {
		// Update memory tracking
		bucketMemory := ea.estimateBucketMemory(bucket)
		ea.currentMemoryBytes -= bucketMemory
		
		// Remove from map immediately
		delete(ea.buckets, key)
		
		// Flush asynchronously
		go ea.flushBucket(bucket)
	}
	
	// Reset memory tracking
	ea.currentMemoryBytes = 0
	ea.emergencyMode = true
	
	log.Printf("✅ Event Aggregator: Emergency flush completed, %d buckets flushed", len(ea.buckets))
}

// GetMemoryStats returns current memory statistics
func (ea *EventAggregator) GetMemoryStats() map[string]interface{} {
	ea.mu.RLock()
	defer ea.mu.RUnlock()
	
	return map[string]interface{}{
		"current_memory_bytes": ea.currentMemoryBytes,
		"max_memory_bytes":     ea.maxMemoryBytes,
		"memory_pressure":      ea.memoryPressure,
		"emergency_mode":       ea.emergencyMode,
		"active_buckets":       len(ea.buckets),
		"max_buckets":          ea.maxBuckets,
		"adjusted_max_buckets": ea.getAdjustedMaxBuckets(),
		"adaptive_bucket_size": ea.getAdaptiveBucketSize(),
		"memory_usage_mb":      float64(ea.currentMemoryBytes) / (1024 * 1024),
	}
}

// memoryMonitoringRoutine periodically monitors memory usage
func (ea *EventAggregator) memoryMonitoringRoutine() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ea.mu.Lock()
			ea.updateMemoryPressure()
			
			// Log memory status periodically
			if ea.memoryPressure > 0.5 {
				log.Printf("📋 Event Aggregator: Memory usage %.1f%% (%d buckets, %.1fMB)",
					ea.memoryPressure*100,
					len(ea.buckets),
					float64(ea.currentMemoryBytes)/(1024*1024))
			}
			
			// Proactive memory management
			if ea.memoryPressure > 0.75 {
				ea.handleMemoryPressure()
			}
			
			ea.mu.Unlock()
			
		case <-ea.ctx.Done():
			return
		}
	}
}