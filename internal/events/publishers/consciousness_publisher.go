package publishers

import (
	"context"
	"log"
	"time"

	"github.com/loreum-org/cortex/internal/events"
	"github.com/loreum-org/cortex/internal/rag"
)

// ConsciousnessPublisher publishes consciousness state events
type ConsciousnessPublisher struct {
	eventBus   *events.EventBus
	ragSystem  *rag.RAGSystem
	ticker     *time.Ticker
	ctx        context.Context
	cancel     context.CancelFunc
	interval   time.Duration
}

// NewConsciousnessPublisher creates a new consciousness publisher
func NewConsciousnessPublisher(eventBus *events.EventBus, ragSystem *rag.RAGSystem, interval time.Duration) *ConsciousnessPublisher {
	if interval == 0 {
		interval = 2 * time.Second // Default consciousness update interval
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ConsciousnessPublisher{
		eventBus:  eventBus,
		ragSystem: ragSystem,
		interval:  interval,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins publishing consciousness events
func (cp *ConsciousnessPublisher) Start() {
	if cp.ragSystem == nil {
		log.Printf("⚠️ Consciousness Publisher: RAG system not available, skipping")
		return
	}
	
	cp.ticker = time.NewTicker(cp.interval)
	
	go func() {
		defer cp.ticker.Stop()
		
		log.Printf("🧠 Consciousness Publisher: Started (interval: %v)", cp.interval)
		
		for {
			select {
			case <-cp.ticker.C:
				cp.publishConsciousnessState()
				
			case <-cp.ctx.Done():
				log.Printf("🧠 Consciousness Publisher: Stopped")
				return
			}
		}
	}()
}

// Stop stops the consciousness publisher
func (cp *ConsciousnessPublisher) Stop() {
	if cp.cancel != nil {
		cp.cancel()
	}
}

// publishConsciousnessState collects and publishes AGI consciousness state
func (cp *ConsciousnessPublisher) publishConsciousnessState() {
	// Get AGI system instead of consciousness runtime
	agiSystem := cp.ragSystem.GetAGISystem()
	if agiSystem == nil {
		return
	}
	
	// Get current AGI consciousness state
	consciousnessStateMap := agiSystem.GetConsciousnessState()
	if consciousnessStateMap == nil {
		return
	}
	
	// Get AGI state
	agiState := agiSystem.GetCurrentAGIState()
	if agiState == nil {
		return
	}
	
	// Convert AGI metrics to consciousness metrics format
	metrics := map[string]interface{}{
		"cycle_count":      consciousnessStateMap["cycle_count"],
		"is_running":       consciousnessStateMap["is_running"],
		"avg_cycle_time":   consciousnessStateMap["avg_cycle_time"],
		"intelligence_level": agiState.IntelligenceLevel,
		"domain_count":     len(agiState.KnowledgeDomains),
	}
	
	// Extract working memory equivalent from AGI state
	workingMemory := map[string]interface{}{
		"recent_inputs_count":     0,
		"short_term_memory_count": 0,
		"current_context_keys":    []string{},
		"active_beliefs":          make(map[string]float64),
		"predictions_count":       0,
		"last_updated":            time.Now(),
	}
	
	// Create AGI consciousness data
	consciousnessData := events.ConsciousnessData{
		State: map[string]interface{}{
			"cycle_count":     consciousnessStateMap["current_cycle"],
			"energy_level":    consciousnessStateMap["energy_level"],
			"focus_level":     consciousnessStateMap["focus_level"],
			"alertness_level": consciousnessStateMap["alertness_level"],
			"decision_state":  consciousnessStateMap["decision_state"],
			"processing_load": consciousnessStateMap["processing_load"],
			"intelligence_level": agiState.IntelligenceLevel,
			"attention": consciousnessStateMap["attention"],
			"active_thoughts": cp.extractAGIThoughtContents(agiState.ActiveThoughts),
			"active_goals":    cp.extractAGIGoalDescriptions(agiState.Goals),
			"emotions":        agiState.Emotions,
			"domains":         consciousnessStateMap["domains"],
		},
		Metrics: metrics,
		WorkingMemory: workingMemory,
		NodeID: cp.getAGINodeID(agiState),
	}
	
	// Create and publish consciousness event
	consciousnessEvent := events.NewEvent(
		events.EventTypeConsciousnessUpdated,
		"consciousness_publisher",
		consciousnessData,
	)
	
	// Add AGI self-awareness metrics
	consciousnessEvent = consciousnessEvent.WithMetadata("agi_awareness", map[string]interface{}{
		"is_conscious":           agiState.CurrentCycle > 0,
		"intelligence_level":     agiState.IntelligenceLevel,
		"processing_capacity":    1.0 - agiState.ProcessingLoad,
		"attention_coherence":    cp.calculateAGIAttentionCoherence(agiState.Attention),
		"emotional_stability":    cp.calculateEmotionalStability(agiState.Emotions),
		"domain_expertise":      cp.calculateDomainExpertise(agiState.KnowledgeDomains),
		"learning_progress":     agiState.LearningState.LearningProgress,
	})
	
	// Publish event
	if err := cp.eventBus.Publish(consciousnessEvent); err != nil {
		log.Printf("❌ Consciousness Publisher: Failed to publish consciousness state: %v", err)
	}
}

// Helper methods for AGI consciousness analysis

func (cp *ConsciousnessPublisher) extractAGIThoughtContents(thoughts []rag.Thought) []string {
	contents := make([]string, len(thoughts))
	for i, thought := range thoughts {
		contents[i] = thought.Content
	}
	return contents
}

func (cp *ConsciousnessPublisher) extractAGIGoalDescriptions(goals []rag.Goal) []string {
	descriptions := make([]string, len(goals))
	for i, goal := range goals {
		descriptions[i] = goal.Description
	}
	return descriptions
}

func (cp *ConsciousnessPublisher) extractContextKeys(context map[string]interface{}) []string {
	keys := make([]string, 0, len(context))
	for key := range context {
		keys = append(keys, key)
	}
	return keys
}

func (cp *ConsciousnessPublisher) calculateAGIAttentionCoherence(attention *rag.AttentionState) float64 {
	if attention == nil {
		return 0.5
	}
	// Simple attention coherence calculation
	return attention.FocusStrength * (1.0 - attention.DistractionLevel)
}

func (cp *ConsciousnessPublisher) calculateEmotionalStability(emotions map[string]float64) float64 {
	if len(emotions) == 0 {
		return 1.0
	}
	
	// Calculate variance of emotional states
	var sum, variance float64
	for _, value := range emotions {
		sum += value
	}
	
	mean := sum / float64(len(emotions))
	
	for _, value := range emotions {
		variance += (value - mean) * (value - mean)
	}
	
	variance = variance / float64(len(emotions))
	
	// Return stability as inverse of variance (normalized)
	return 1.0 / (1.0 + variance)
}

func (cp *ConsciousnessPublisher) calculateDomainExpertise(domains map[string]*rag.Domain) float64 {
	if len(domains) == 0 {
		return 0.0
	}
	
	// Calculate average domain expertise
	var totalExpertise float64
	for _, domain := range domains {
		if domain != nil {
			totalExpertise += domain.Expertise
		}
	}
	
	return totalExpertise / float64(len(domains))
}

func (cp *ConsciousnessPublisher) getAGINodeID(agiState *rag.AGIState) string {
	if agiState != nil && agiState.NodeID != "" {
		return agiState.NodeID
	}
	// Get node ID from RAG system if available
	if cp.ragSystem != nil {
		return "agi_node"
	}
	return "unknown_agi_node"
}

// PublishConsciousnessEvent allows manual publishing of consciousness events
func (cp *ConsciousnessPublisher) PublishConsciousnessEvent(eventType, description string, data map[string]interface{}) error {
	consciousnessEvent := events.NewEvent(
		eventType,
		"consciousness_publisher",
		events.ConsciousnessData{
			State: data,
			NodeID: cp.getNodeID(),
		},
	).WithMetadata("description", description)
	
	return cp.eventBus.Publish(consciousnessEvent)
}

// GetInterval returns the current publishing interval
func (cp *ConsciousnessPublisher) GetInterval() time.Duration {
	return cp.interval
}

// SetInterval updates the publishing interval
func (cp *ConsciousnessPublisher) SetInterval(interval time.Duration) {
	if interval <= 0 {
		return
	}
	
	cp.interval = interval
	
	// Restart ticker with new interval if running
	if cp.ticker != nil {
		cp.ticker.Stop()
		cp.ticker = time.NewTicker(interval)
		log.Printf("🧠 Consciousness Publisher: Interval updated to %v", interval)
	}
}