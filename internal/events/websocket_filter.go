package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
)

// WebSocketEventFilter provides intelligent filtering for WebSocket events
type WebSocketEventFilter struct {
	// Filter rules
	filterRules       map[string]*FilterRule
	userFilters       map[string]*UserFilterProfile
	globalFilters     *GlobalFilterConfig
	
	// Rate limiting
	rateLimiters      map[string]*RateLimiter
	bandwidthTracking map[string]*BandwidthTracker
	
	// Statistics
	filterStats       FilterStats
	
	// Configuration
	maxFilterRules    int
	maxBandwidthPerUser int64 // bytes per second
	maxEventsPerUser    int   // events per second
	
	// Synchronization
	mu                sync.RWMutex
}

// FilterRule defines event filtering criteria
type FilterRule struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Priority         int                    `json:"priority"`
	EventTypes       []string               `json:"event_types,omitempty"`
	Sources          []string               `json:"sources,omitempty"`
	MetadataFilters  []MetadataFilter       `json:"metadata_filters,omitempty"`
	ContentFilters   []ContentFilter        `json:"content_filters,omitempty"`
	RateLimit        *RateLimit             `json:"rate_limit,omitempty"`
	SizeLimit        *SizeLimit             `json:"size_limit,omitempty"`
	TimeWindow       *TimeWindow            `json:"time_window,omitempty"`
	Action           FilterAction           `json:"action"`
	Enabled          bool                   `json:"enabled"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// MetadataFilter filters based on event metadata
type MetadataFilter struct {
	Key      string      `json:"key"`
	Operator string      `json:"operator"` // eq, ne, contains, matches, gt, lt
	Value    interface{} `json:"value"`
}

// ContentFilter filters based on event content
type ContentFilter struct {
	Path     string `json:"path"`     // JSON path in event data
	Operator string `json:"operator"` // eq, ne, contains, matches, gt, lt
	Value    interface{} `json:"value"`
	Regex    string `json:"regex,omitempty"`
}

// RateLimit defines rate limiting for events
type RateLimit struct {
	MaxEvents    int           `json:"max_events"`
	TimeWindow   time.Duration `json:"time_window"`
	BurstAllowed int           `json:"burst_allowed"`
}

// SizeLimit defines size limits for events
type SizeLimit struct {
	MaxSize      int64 `json:"max_size"`      // bytes
	SampleRate   float64 `json:"sample_rate"` // 0.0 - 1.0
	CompressLarge bool  `json:"compress_large"`
}

// TimeWindow defines time-based filtering
type TimeWindow struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Timezone  string    `json:"timezone"`
}

// FilterAction defines what to do with filtered events
type FilterAction string

const (
	FilterActionAllow      FilterAction = "allow"
	FilterActionBlock      FilterAction = "block"
	FilterActionSample     FilterAction = "sample"
	FilterActionCompress   FilterAction = "compress"
	FilterActionDelay      FilterAction = "delay"
	FilterActionTransform  FilterAction = "transform"
)

// UserFilterProfile defines user-specific filtering preferences
type UserFilterProfile struct {
	UserID              string                 `json:"user_id"`
	BandwidthLimit      int64                  `json:"bandwidth_limit"`
	EventRateLimit      int                    `json:"event_rate_limit"`
	PreferredEventTypes []string               `json:"preferred_event_types"`
	BlockedEventTypes   []string               `json:"blocked_event_types"`
	CompressionEnabled  bool                   `json:"compression_enabled"`
	SamplingRate        float64                `json:"sampling_rate"`
	CustomFilters       map[string]interface{} `json:"custom_filters"`
	Priority            int                    `json:"priority"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// GlobalFilterConfig defines system-wide filtering policies
type GlobalFilterConfig struct {
	DefaultBandwidthLimit    int64                  `json:"default_bandwidth_limit"`
	DefaultEventRateLimit    int                    `json:"default_event_rate_limit"`
	DefaultCompressionEnabled bool                  `json:"default_compression_enabled"`
	DefaultSamplingRate      float64                `json:"default_sampling_rate"`
	SystemFilters            []FilterRule           `json:"system_filters"`
	EmergencyMode            bool                   `json:"emergency_mode"`
	MaintenanceMode          bool                   `json:"maintenance_mode"`
	GlobalSettings           map[string]interface{} `json:"global_settings"`
}

// RateLimiter tracks rate limits for users/connections
type RateLimiter struct {
	UserID       string
	EventCount   int
	LastReset    time.Time
	WindowSize   time.Duration
	MaxEvents    int
	BurstBuffer  int
}

// BandwidthTracker tracks bandwidth usage
type BandwidthTracker struct {
	UserID         string
	BytesSent      int64
	LastReset      time.Time
	WindowSize     time.Duration
	MaxBandwidth   int64
	CurrentRate    int64
}

// FilterStats tracks filtering statistics
type FilterStats struct {
	TotalEventsProcessed    int64                    `json:"total_events_processed"`
	EventsFiltered          int64                    `json:"events_filtered"`
	EventsBlocked           int64                    `json:"events_blocked"`
	EventsCompressed        int64                    `json:"events_compressed"`
	EventsSampled           int64                    `json:"events_sampled"`
	BandwidthSaved          int64                    `json:"bandwidth_saved"`
	FilteringLatency        time.Duration            `json:"filtering_latency"`
	FilterRulePerformance   map[string]time.Duration `json:"filter_rule_performance"`
	UserFilteringStats      map[string]UserStats     `json:"user_filtering_stats"`
	LastUpdate              time.Time                `json:"last_update"`
}

// UserStats tracks per-user filtering statistics
type UserStats struct {
	EventsProcessed   int64         `json:"events_processed"`
	EventsBlocked     int64         `json:"events_blocked"`
	BandwidthUsed     int64         `json:"bandwidth_used"`
	BandwidthSaved    int64         `json:"bandwidth_saved"`
	AverageEventSize  int64         `json:"average_event_size"`
	LastActivity      time.Time     `json:"last_activity"`
}

// FilterResult represents the result of event filtering
type FilterResult struct {
	Action         FilterAction           `json:"action"`
	Allowed        bool                   `json:"allowed"`
	Modified       bool                   `json:"modified"`
	OriginalSize   int64                  `json:"original_size"`
	FilteredSize   int64                  `json:"filtered_size"`
	AppliedRules   []string               `json:"applied_rules"`
	Metadata       map[string]interface{} `json:"metadata"`
	ProcessingTime time.Duration          `json:"processing_time"`
}

// NewWebSocketEventFilter creates a new WebSocket event filter
func NewWebSocketEventFilter() *WebSocketEventFilter {
	return &WebSocketEventFilter{
		filterRules:         make(map[string]*FilterRule),
		userFilters:         make(map[string]*UserFilterProfile),
		rateLimiters:        make(map[string]*RateLimiter),
		bandwidthTracking:   make(map[string]*BandwidthTracker),
		maxFilterRules:      1000,
		maxBandwidthPerUser: 1024 * 1024, // 1MB per second
		maxEventsPerUser:    100,          // 100 events per second
		globalFilters: &GlobalFilterConfig{
			DefaultBandwidthLimit:     1024 * 1024, // 1MB per second
			DefaultEventRateLimit:     100,         // 100 events per second
			DefaultCompressionEnabled: true,
			DefaultSamplingRate:       1.0,
			SystemFilters:             make([]FilterRule, 0),
			EmergencyMode:             false,
			MaintenanceMode:           false,
			GlobalSettings:            make(map[string]interface{}),
		},
		filterStats: FilterStats{
			FilterRulePerformance: make(map[string]time.Duration),
			UserFilteringStats:    make(map[string]UserStats),
		},
	}
}

// FilterEvent applies filtering rules to an event
func (wef *WebSocketEventFilter) FilterEvent(
	ctx context.Context,
	event Event,
	userID string,
	connectionID string,
) (*FilterResult, error) {
	start := time.Now()
	
	result := &FilterResult{
		Action:       FilterActionAllow,
		Allowed:      true,
		Modified:     false,
		AppliedRules: make([]string, 0),
		Metadata:     make(map[string]interface{}),
	}
	
	// Calculate original size
	eventData, _ := json.Marshal(event)
	result.OriginalSize = int64(len(eventData))
	
	// Check global emergency/maintenance mode
	if wef.globalFilters.EmergencyMode {
		result.Action = FilterActionBlock
		result.Allowed = false
		result.AppliedRules = append(result.AppliedRules, "emergency_mode")
		return result, nil
	}
	
	if wef.globalFilters.MaintenanceMode {
		// Only allow critical events during maintenance
		if !wef.isCriticalEvent(event) {
			result.Action = FilterActionBlock
			result.Allowed = false
			result.AppliedRules = append(result.AppliedRules, "maintenance_mode")
			return result, nil
		}
	}
	
	// Check rate limits
	if wef.checkRateLimit(userID, connectionID) {
		result.Action = FilterActionBlock
		result.Allowed = false
		result.AppliedRules = append(result.AppliedRules, "rate_limit_exceeded")
		wef.updateUserStats(userID, false, result.OriginalSize, 0)
		return result, nil
	}
	
	// Check bandwidth limits
	if wef.checkBandwidthLimit(userID, result.OriginalSize) {
		result.Action = FilterActionBlock
		result.Allowed = false
		result.AppliedRules = append(result.AppliedRules, "bandwidth_limit_exceeded")
		wef.updateUserStats(userID, false, result.OriginalSize, 0)
		return result, nil
	}
	
	// Apply user-specific filters
	userProfile := wef.getUserFilterProfile(userID)
	if userProfile != nil {
		if filterResult := wef.applyUserFilters(event, userProfile, result); !filterResult {
			wef.updateUserStats(userID, false, result.OriginalSize, 0)
			return result, nil
		}
	}
	
	// Apply global filter rules
	wef.mu.RLock()
	rules := make([]*FilterRule, 0, len(wef.filterRules))
	for _, rule := range wef.filterRules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	wef.mu.RUnlock()
	
	// Sort rules by priority
	wef.sortRulesByPriority(rules)
	
	// Apply each rule
	for _, rule := range rules {
		ruleStart := time.Now()
		
		if wef.matchesRule(event, rule) {
			result.AppliedRules = append(result.AppliedRules, rule.ID)
			
			switch rule.Action {
			case FilterActionBlock:
				result.Action = FilterActionBlock
				result.Allowed = false
				
			case FilterActionSample:
				if userProfile != nil && userProfile.SamplingRate < 1.0 {
					if !wef.shouldSample(userProfile.SamplingRate) {
						result.Action = FilterActionBlock
						result.Allowed = false
					}
				}
				
			case FilterActionCompress:
				if wef.shouldCompress(event, rule) {
					compressedEvent := wef.compressEvent(event)
					if compressedData, err := json.Marshal(compressedEvent); err == nil {
						result.FilteredSize = int64(len(compressedData))
						result.Modified = true
						result.Action = FilterActionCompress
					}
				}
				
			case FilterActionTransform:
				transformedEvent := wef.transformEvent(event, rule)
				if transformedData, err := json.Marshal(transformedEvent); err == nil {
					result.FilteredSize = int64(len(transformedData))
					result.Modified = true
					result.Action = FilterActionTransform
				}
			}
		}
		
		// Update rule performance stats
		ruleTime := time.Since(ruleStart)
		wef.mu.Lock()
		wef.filterStats.FilterRulePerformance[rule.ID] = ruleTime
		wef.mu.Unlock()
		
		// If blocked, stop processing further rules
		if !result.Allowed {
			break
		}
	}
	
	// Set filtered size if not set
	if result.FilteredSize == 0 {
		result.FilteredSize = result.OriginalSize
	}
	
	// Update bandwidth tracking
	wef.updateBandwidthTracking(userID, result.FilteredSize)
	
	// Update statistics
	result.ProcessingTime = time.Since(start)
	wef.updateFilterStats(result)
	wef.updateUserStats(userID, result.Allowed, result.OriginalSize, result.FilteredSize)
	
	return result, nil
}

// AddFilterRule adds a new filter rule
func (wef *WebSocketEventFilter) AddFilterRule(rule FilterRule) error {
	wef.mu.Lock()
	defer wef.mu.Unlock()
	
	if len(wef.filterRules) >= wef.maxFilterRules {
		return fmt.Errorf("maximum filter rules exceeded")
	}
	
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	
	wef.filterRules[rule.ID] = &rule
	
	log.Printf("🔧 WebSocket Filter: Added filter rule '%s' (%s)", rule.Name, rule.ID)
	return nil
}

// RemoveFilterRule removes a filter rule
func (wef *WebSocketEventFilter) RemoveFilterRule(ruleID string) error {
	wef.mu.Lock()
	defer wef.mu.Unlock()
	
	if _, exists := wef.filterRules[ruleID]; !exists {
		return fmt.Errorf("filter rule not found: %s", ruleID)
	}
	
	delete(wef.filterRules, ruleID)
	
	log.Printf("🔧 WebSocket Filter: Removed filter rule %s", ruleID)
	return nil
}

// SetUserFilterProfile sets a user's filter profile
func (wef *WebSocketEventFilter) SetUserFilterProfile(profile UserFilterProfile) {
	wef.mu.Lock()
	defer wef.mu.Unlock()
	
	profile.UpdatedAt = time.Now()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = time.Now()
	}
	
	wef.userFilters[profile.UserID] = &profile
	
	log.Printf("🔧 WebSocket Filter: Updated filter profile for user %s", profile.UserID)
}

// Helper methods

func (wef *WebSocketEventFilter) getUserFilterProfile(userID string) *UserFilterProfile {
	wef.mu.RLock()
	defer wef.mu.RUnlock()
	
	return wef.userFilters[userID]
}

func (wef *WebSocketEventFilter) checkRateLimit(userID, connectionID string) bool {
	key := fmt.Sprintf("%s:%s", userID, connectionID)
	
	wef.mu.Lock()
	defer wef.mu.Unlock()
	
	limiter, exists := wef.rateLimiters[key]
	if !exists {
		limiter = &RateLimiter{
			UserID:      userID,
			EventCount:  0,
			LastReset:   time.Now(),
			WindowSize:  time.Minute,
			MaxEvents:   wef.maxEventsPerUser,
			BurstBuffer: wef.maxEventsPerUser / 10,
		}
		wef.rateLimiters[key] = limiter
	}
	
	now := time.Now()
	if now.Sub(limiter.LastReset) >= limiter.WindowSize {
		limiter.EventCount = 0
		limiter.LastReset = now
	}
	
	limiter.EventCount++
	return limiter.EventCount > limiter.MaxEvents
}

func (wef *WebSocketEventFilter) checkBandwidthLimit(userID string, eventSize int64) bool {
	wef.mu.Lock()
	defer wef.mu.Unlock()
	
	tracker, exists := wef.bandwidthTracking[userID]
	if !exists {
		tracker = &BandwidthTracker{
			UserID:       userID,
			BytesSent:    0,
			LastReset:    time.Now(),
			WindowSize:   time.Minute,
			MaxBandwidth: wef.maxBandwidthPerUser,
		}
		wef.bandwidthTracking[userID] = tracker
	}
	
	now := time.Now()
	if now.Sub(tracker.LastReset) >= tracker.WindowSize {
		tracker.BytesSent = 0
		tracker.LastReset = now
	}
	
	return tracker.BytesSent+eventSize > tracker.MaxBandwidth
}

func (wef *WebSocketEventFilter) updateBandwidthTracking(userID string, size int64) {
	wef.mu.Lock()
	defer wef.mu.Unlock()
	
	if tracker, exists := wef.bandwidthTracking[userID]; exists {
		tracker.BytesSent += size
		tracker.CurrentRate = tracker.BytesSent / int64(time.Since(tracker.LastReset).Seconds())
	}
}

func (wef *WebSocketEventFilter) applyUserFilters(
	event Event,
	profile *UserFilterProfile,
	result *FilterResult,
) bool {
	// Check blocked event types
	for _, blockedType := range profile.BlockedEventTypes {
		if event.Type == blockedType {
			result.Action = FilterActionBlock
			result.Allowed = false
			result.AppliedRules = append(result.AppliedRules, "user_blocked_type")
			return false
		}
	}
	
	// Check preferred event types (if specified, only allow these)
	if len(profile.PreferredEventTypes) > 0 {
		found := false
		for _, preferredType := range profile.PreferredEventTypes {
			if event.Type == preferredType {
				found = true
				break
			}
		}
		if !found {
			result.Action = FilterActionBlock
			result.Allowed = false
			result.AppliedRules = append(result.AppliedRules, "user_not_preferred_type")
			return false
		}
	}
	
	return true
}

func (wef *WebSocketEventFilter) matchesRule(event Event, rule *FilterRule) bool {
	// Check event types
	if len(rule.EventTypes) > 0 {
		found := false
		for _, eventType := range rule.EventTypes {
			if event.Type == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check sources
	if len(rule.Sources) > 0 {
		found := false
		for _, source := range rule.Sources {
			if event.Source == source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check metadata filters
	for _, filter := range rule.MetadataFilters {
		if !wef.evaluateMetadataFilter(event, filter) {
			return false
		}
	}
	
	// Check content filters
	for _, filter := range rule.ContentFilters {
		if !wef.evaluateContentFilter(event, filter) {
			return false
		}
	}
	
	return true
}

func (wef *WebSocketEventFilter) evaluateMetadataFilter(event Event, filter MetadataFilter) bool {
	if event.Metadata == nil {
		return false
	}
	
	value, exists := event.Metadata[filter.Key]
	if !exists {
		return false
	}
	
	return wef.evaluateFilterOperator(value, filter.Operator, filter.Value)
}

func (wef *WebSocketEventFilter) evaluateContentFilter(event Event, filter ContentFilter) bool {
	// Simple JSON path evaluation - could be enhanced with proper JSON path library
	value := wef.extractValueFromPath(event.Data, filter.Path)
	if value == nil {
		return false
	}
	
	if filter.Regex != "" {
		if str, ok := value.(string); ok {
			matched, _ := regexp.MatchString(filter.Regex, str)
			return matched
		}
		return false
	}
	
	return wef.evaluateFilterOperator(value, filter.Operator, filter.Value)
}

func (wef *WebSocketEventFilter) evaluateFilterOperator(value interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq":
		return value == expected
	case "ne":
		return value != expected
	case "contains":
		if str, ok := value.(string); ok {
			if substr, ok := expected.(string); ok {
				return contains(str, substr)
			}
		}
	case "gt":
		return wef.compareValues(value, expected) > 0
	case "lt":
		return wef.compareValues(value, expected) < 0
	}
	return false
}

func (wef *WebSocketEventFilter) compareValues(a, b interface{}) int {
	// Simple numeric comparison
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

func (wef *WebSocketEventFilter) extractValueFromPath(data interface{}, path string) interface{} {
	// Simple path extraction - could be enhanced with proper JSON path
	if dataMap, ok := data.(map[string]interface{}); ok {
		return dataMap[path]
	}
	return nil
}

func (wef *WebSocketEventFilter) isCriticalEvent(event Event) bool {
	criticalTypes := []string{"error", "alert", "system.critical", "security.alert"}
	for _, criticalType := range criticalTypes {
		if event.Type == criticalType {
			return true
		}
	}
	return false
}

func (wef *WebSocketEventFilter) shouldSample(rate float64) bool {
	// Simple random sampling
	return time.Now().UnixNano()%100 < int64(rate*100)
}

func (wef *WebSocketEventFilter) shouldCompress(event Event, rule *FilterRule) bool {
	if rule.SizeLimit == nil {
		return false
	}
	
	eventData, _ := json.Marshal(event)
	return int64(len(eventData)) > rule.SizeLimit.MaxSize
}

func (wef *WebSocketEventFilter) compressEvent(event Event) Event {
	// Simple event compression - remove verbose fields
	compressedEvent := event
	if compressedEvent.Metadata == nil {
		compressedEvent.Metadata = make(map[string]interface{})
	}
	compressedEvent.Metadata["compressed"] = true
	
	// Remove large data fields or compress them
	if dataMap, ok := compressedEvent.Data.(map[string]interface{}); ok {
		for key, value := range dataMap {
			if str, ok := value.(string); ok && len(str) > 1000 {
				dataMap[key] = str[:100] + "...[compressed]"
			}
		}
	}
	
	return compressedEvent
}

func (wef *WebSocketEventFilter) transformEvent(event Event, rule *FilterRule) Event {
	// Event transformation based on rule - placeholder implementation
	transformedEvent := event
	if transformedEvent.Metadata == nil {
		transformedEvent.Metadata = make(map[string]interface{})
	}
	transformedEvent.Metadata["transformed"] = true
	transformedEvent.Metadata["rule_id"] = rule.ID
	
	return transformedEvent
}

func (wef *WebSocketEventFilter) sortRulesByPriority(rules []*FilterRule) {
	// Sort by priority (lower number = higher priority)
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority > rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}

func (wef *WebSocketEventFilter) updateFilterStats(result *FilterResult) {
	wef.mu.Lock()
	defer wef.mu.Unlock()
	
	wef.filterStats.TotalEventsProcessed++
	
	if !result.Allowed {
		wef.filterStats.EventsBlocked++
	}
	
	if result.Modified {
		wef.filterStats.EventsFiltered++
		
		switch result.Action {
		case FilterActionCompress:
			wef.filterStats.EventsCompressed++
		case FilterActionSample:
			wef.filterStats.EventsSampled++
		}
		
		if result.OriginalSize > result.FilteredSize {
			wef.filterStats.BandwidthSaved += (result.OriginalSize - result.FilteredSize)
		}
	}
	
	// Update average latency
	if wef.filterStats.FilteringLatency == 0 {
		wef.filterStats.FilteringLatency = result.ProcessingTime
	} else {
		wef.filterStats.FilteringLatency = (wef.filterStats.FilteringLatency + result.ProcessingTime) / 2
	}
	
	wef.filterStats.LastUpdate = time.Now()
}

func (wef *WebSocketEventFilter) updateUserStats(userID string, allowed bool, originalSize, filteredSize int64) {
	wef.mu.Lock()
	defer wef.mu.Unlock()
	
	stats, exists := wef.filterStats.UserFilteringStats[userID]
	if !exists {
		stats = UserStats{}
	}
	
	stats.EventsProcessed++
	if !allowed {
		stats.EventsBlocked++
	}
	
	stats.BandwidthUsed += filteredSize
	if originalSize > filteredSize {
		stats.BandwidthSaved += (originalSize - filteredSize)
	}
	
	// Update average event size
	if stats.AverageEventSize == 0 {
		stats.AverageEventSize = originalSize
	} else {
		stats.AverageEventSize = (stats.AverageEventSize + originalSize) / 2
	}
	
	stats.LastActivity = time.Now()
	
	wef.filterStats.UserFilteringStats[userID] = stats
}

// GetFilterStats returns current filtering statistics
func (wef *WebSocketEventFilter) GetFilterStats() FilterStats {
	wef.mu.RLock()
	defer wef.mu.RUnlock()
	return wef.filterStats
}

// GetUserStats returns statistics for a specific user
func (wef *WebSocketEventFilter) GetUserStats(userID string) (UserStats, bool) {
	wef.mu.RLock()
	defer wef.mu.RUnlock()
	
	stats, exists := wef.filterStats.UserFilteringStats[userID]
	return stats, exists
}