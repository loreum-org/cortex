package publishers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/events"
	"github.com/loreum-org/cortex/internal/rag"
)

// PublisherManager manages all event publishers
type PublisherManager struct {
	eventBus              *events.EventBus
	metricsPublisher      *MetricsPublisher
	consciousnessPublisher *ConsciousnessPublisher
	systemPublisher       *SystemPublisher
	
	// Configuration
	config *PublisherConfig
	
	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// PublisherConfig configures the publisher manager
type PublisherConfig struct {
	MetricsInterval      time.Duration `json:"metrics_interval"`
	ConsciousnessInterval time.Duration `json:"consciousness_interval"`
	SystemInterval       time.Duration `json:"system_interval"`
	
	EnableMetrics      bool `json:"enable_metrics"`
	EnableConsciousness bool `json:"enable_consciousness"`
	EnableSystem       bool `json:"enable_system"`
}

// DefaultPublisherConfig returns default configuration
func DefaultPublisherConfig() *PublisherConfig {
	return &PublisherConfig{
		MetricsInterval:       5 * time.Second,
		ConsciousnessInterval: 2 * time.Second,
		SystemInterval:        10 * time.Second,
		EnableMetrics:         true,
		EnableConsciousness:   true,
		EnableSystem:          true,
	}
}

// NewPublisherManager creates a new publisher manager
func NewPublisherManager(eventBus *events.EventBus, config *PublisherConfig) *PublisherManager {
	if config == nil {
		config = DefaultPublisherConfig()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PublisherManager{
		eventBus: eventBus,
		config:   config,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Initialize sets up all publishers with their dependencies
func (pm *PublisherManager) Initialize(server ServerInterface, ragSystem interface{}, agentRegistry interface{}) error {
	log.Printf("🚀 Publisher Manager: Initializing publishers...")
	
	// Initialize metrics publisher
	if pm.config.EnableMetrics {
		pm.metricsPublisher = NewMetricsPublisher(
			pm.eventBus,
			server,
			pm.config.MetricsInterval,
		)
		log.Printf("✅ Metrics Publisher: Initialized (interval: %v)", pm.config.MetricsInterval)
	}
	
	// Initialize consciousness publisher
	if pm.config.EnableConsciousness {
		if ragSystemTyped, ok := ragSystem.(*rag.RAGSystem); ok {
			pm.consciousnessPublisher = NewConsciousnessPublisher(
				pm.eventBus,
				ragSystemTyped,
				pm.config.ConsciousnessInterval,
			)
			log.Printf("✅ Consciousness Publisher: Initialized (interval: %v)", pm.config.ConsciousnessInterval)
		} else {
			log.Printf("⚠️ Consciousness Publisher: RAG system not compatible, skipping")
		}
	}
	
	// Initialize system publisher
	if pm.config.EnableSystem {
		pm.systemPublisher = NewSystemPublisher(
			pm.eventBus,
			server,
			pm.config.SystemInterval,
		)
		
		// Register component checkers
		if ragSystem != nil {
			pm.systemPublisher.RegisterComponent(NewRAGComponentChecker(ragSystem))
		}
		
		if agentRegistry != nil {
			pm.systemPublisher.RegisterComponent(NewAgentComponentChecker(agentRegistry))
		}
		
		pm.systemPublisher.RegisterComponent(NewEventBusComponentChecker(pm.eventBus))
		
		log.Printf("✅ System Publisher: Initialized (interval: %v)", pm.config.SystemInterval)
	}
	
	log.Printf("✅ Publisher Manager: All publishers initialized")
	return nil
}

// Start starts all enabled publishers
func (pm *PublisherManager) Start() error {
	log.Printf("🚀 Publisher Manager: Starting all publishers...")
	
	if pm.metricsPublisher != nil {
		pm.metricsPublisher.Start()
	}
	
	if pm.consciousnessPublisher != nil {
		pm.consciousnessPublisher.Start()
	}
	
	if pm.systemPublisher != nil {
		pm.systemPublisher.Start()
	}
	
	log.Printf("✅ Publisher Manager: All publishers started")
	return nil
}

// Stop stops all publishers
func (pm *PublisherManager) Stop() error {
	log.Printf("🔄 Publisher Manager: Stopping all publishers...")
	
	if pm.cancel != nil {
		pm.cancel()
	}
	
	if pm.metricsPublisher != nil {
		pm.metricsPublisher.Stop()
	}
	
	if pm.consciousnessPublisher != nil {
		pm.consciousnessPublisher.Stop()
	}
	
	if pm.systemPublisher != nil {
		pm.systemPublisher.Stop()
	}
	
	log.Printf("✅ Publisher Manager: All publishers stopped")
	return nil
}

// GetStatus returns the status of all publishers
func (pm *PublisherManager) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled_publishers": []string{},
		"intervals": map[string]interface{}{},
		"health": map[string]interface{}{},
	}
	
	enabledPublishers := []string{}
	intervals := map[string]interface{}{}
	health := map[string]interface{}{}
	
	if pm.metricsPublisher != nil {
		enabledPublishers = append(enabledPublishers, "metrics")
		intervals["metrics"] = pm.metricsPublisher.GetInterval()
		health["metrics"] = "active"
	}
	
	if pm.consciousnessPublisher != nil {
		enabledPublishers = append(enabledPublishers, "consciousness")
		intervals["consciousness"] = pm.consciousnessPublisher.GetInterval()
		health["consciousness"] = "active"
	}
	
	if pm.systemPublisher != nil {
		enabledPublishers = append(enabledPublishers, "system")
		intervals["system"] = pm.systemPublisher.GetInterval()
		health["system"] = "active"
		
		// Add component status
		status["components"] = pm.systemPublisher.GetComponentStatus()
	}
	
	status["enabled_publishers"] = enabledPublishers
	status["intervals"] = intervals
	status["health"] = health
	status["timestamp"] = time.Now()
	
	return status
}

// PublishCustomEvent allows manual event publishing through any publisher
func (pm *PublisherManager) PublishCustomEvent(publisherType, eventType string, data map[string]interface{}) error {
	switch publisherType {
	case "metrics":
		if pm.metricsPublisher != nil {
			return pm.metricsPublisher.PublishCustomMetrics(eventType, data)
		}
		return fmt.Errorf("metrics publisher not available")
		
	case "consciousness":
		if pm.consciousnessPublisher != nil {
			return pm.consciousnessPublisher.PublishConsciousnessEvent(eventType, "custom", data)
		}
		return fmt.Errorf("consciousness publisher not available")
		
	case "system":
		if pm.systemPublisher != nil {
			return pm.systemPublisher.PublishSystemEvent(eventType, "custom", data)
		}
		return fmt.Errorf("system publisher not available")
		
	default:
		return fmt.Errorf("unknown publisher type: %s", publisherType)
	}
}

// UpdateIntervals allows updating publisher intervals at runtime
func (pm *PublisherManager) UpdateIntervals(config *PublisherConfig) error {
	log.Printf("🔄 Publisher Manager: Updating intervals...")
	
	if pm.metricsPublisher != nil && config.MetricsInterval > 0 {
		pm.metricsPublisher.SetInterval(config.MetricsInterval)
		pm.config.MetricsInterval = config.MetricsInterval
	}
	
	if pm.consciousnessPublisher != nil && config.ConsciousnessInterval > 0 {
		pm.consciousnessPublisher.SetInterval(config.ConsciousnessInterval)
		pm.config.ConsciousnessInterval = config.ConsciousnessInterval
	}
	
	if pm.systemPublisher != nil && config.SystemInterval > 0 {
		pm.systemPublisher.SetInterval(config.SystemInterval)
		pm.config.SystemInterval = config.SystemInterval
	}
	
	log.Printf("✅ Publisher Manager: Intervals updated")
	return nil
}

// GetConfig returns the current configuration
func (pm *PublisherManager) GetConfig() *PublisherConfig {
	// Return a copy to prevent modification
	return &PublisherConfig{
		MetricsInterval:       pm.config.MetricsInterval,
		ConsciousnessInterval: pm.config.ConsciousnessInterval,
		SystemInterval:        pm.config.SystemInterval,
		EnableMetrics:         pm.config.EnableMetrics,
		EnableConsciousness:   pm.config.EnableConsciousness,
		EnableSystem:          pm.config.EnableSystem,
	}
}

// TriggerHealthCheck manually triggers a system health check
func (pm *PublisherManager) TriggerHealthCheck() error {
	if pm.systemPublisher != nil {
		return pm.systemPublisher.PublishHealthCheck()
	}
	return fmt.Errorf("system publisher not available")
}

// GetMetrics returns metrics from all publishers
func (pm *PublisherManager) GetMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"publisher_manager": map[string]interface{}{
			"status":    "active",
			"timestamp": time.Now(),
		},
	}
	
	// Add individual publisher status
	metrics["publishers"] = pm.GetStatus()
	
	return metrics
}