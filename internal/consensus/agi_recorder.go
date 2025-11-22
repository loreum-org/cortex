package consensus

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// AGIRecorder manages blockchain recording of AGI domain knowledge across the network
type AGIRecorder struct {
	consensusService *ConsensusService
	nodeID           string
	networkAGIState  map[string]*types.NodeAGIState // nodeID -> AGI state
	lastSnapshot     *types.NetworkAGISnapshot

	// Recording intervals
	updateInterval   time.Duration // How often to record AGI updates
	snapshotInterval time.Duration // How often to create network snapshots

	// Thresholds for recording
	minIntelligenceChange float64 // Minimum intelligence change to record
	minDomainChange       float64 // Minimum domain expertise change to record

	// Thread safety
	mu sync.RWMutex

	// Background tasks
	stopChan chan struct{}
	running  bool
}

// NewAGIRecorder creates a new AGI blockchain recorder
func NewAGIRecorder(consensusService *ConsensusService, nodeID string) *AGIRecorder {
	return &AGIRecorder{
		consensusService:      consensusService,
		nodeID:                nodeID,
		networkAGIState:       make(map[string]*types.NodeAGIState),
		updateInterval:        5 * time.Minute,  // Record updates every 5 minutes
		snapshotInterval:      30 * time.Minute, // Network snapshots every 30 minutes
		minIntelligenceChange: 0.5,              // Record if intelligence changes by 0.5+
		minDomainChange:       1.0,              // Record if domain expertise changes by 1.0+
		stopChan:              make(chan struct{}),
	}
}

// Start begins the AGI recording background processes
func (a *AGIRecorder) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return fmt.Errorf("AGI recorder already running")
	}

	a.running = true

	// Start background recording tasks
	go a.periodicAGIUpdates(ctx)
	go a.periodicNetworkSnapshots(ctx)

	log.Printf("AGI blockchain recorder started for node %s", a.nodeID)
	return nil
}

// Stop halts the AGI recording processes
func (a *AGIRecorder) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	close(a.stopChan)
	a.running = false

	log.Printf("AGI blockchain recorder stopped")
	return nil
}

// RecordAGIUpdate records an AGI knowledge update to the blockchain
func (a *AGIRecorder) RecordAGIUpdate(ctx context.Context, nodeID string, agiState interface{}) error {
	// Extract AGI state information
	intelligenceLevel, domainKnowledge, version, err := a.extractAGIStateData(agiState)
	if err != nil {
		return fmt.Errorf("failed to extract AGI state: %w", err)
	}

	// Check if update is significant enough to record
	if !a.shouldRecordUpdate(nodeID, intelligenceLevel, domainKnowledge) {
		return nil // Skip recording for minor updates
	}

	// Create AGI transaction data
	agiData := &types.AGITransactionData{
		NodeID:            nodeID,
		AGIVersion:        version,
		IntelligenceLevel: intelligenceLevel,
		DomainKnowledge:   domainKnowledge,
		UpdateType:        "knowledge_update",
		LearningMetrics:   a.calculateLearningMetrics(nodeID, agiState),
	}

	// Create transaction
	txID := a.generateTransactionID("agi_update", nodeID)
	parentIDs := a.getRecentTransactionIDs(3)
	transaction := types.NewAGITransaction(txID, agiData, parentIDs)

	// Submit to consensus
	if err := a.consensusService.SubmitTransaction(ctx, transaction); err != nil {
		return fmt.Errorf("failed to submit AGI update transaction: %w", err)
	}

	// Update local state
	a.updateLocalAGIState(nodeID, intelligenceLevel, domainKnowledge, version)

	log.Printf("Recorded AGI update for node %s: intelligence=%.1f, domains=%d",
		nodeID, intelligenceLevel, len(domainKnowledge))

	return nil
}

// RecordAGIEvolution records a significant AGI evolution event to the blockchain
func (a *AGIRecorder) RecordAGIEvolution(ctx context.Context, nodeID string, evolutionEvent interface{}) error {
	// Extract evolution event data
	triggerType, intelligenceGain, domainsAffected, impact, description, err := a.extractEvolutionData(evolutionEvent)
	if err != nil {
		return fmt.Errorf("failed to extract evolution data: %w", err)
	}

	// Get current AGI state
	intelligenceLevel, domainKnowledge, version, err := a.getCurrentAGIState(nodeID)
	if err != nil {
		return fmt.Errorf("failed to get current AGI state: %w", err)
	}

	// Create evolution event
	evolutionEventData := &types.AGIEvolutionEvent{
		TriggerType:          triggerType,
		IntelligenceGain:     intelligenceGain,
		DomainsAffected:      domainsAffected,
		CapabilitiesGained:   []string{"enhanced_reasoning", "improved_learning"},
		EvolutionDescription: description,
		Impact:               impact,
	}

	// Create AGI transaction data
	agiData := &types.AGITransactionData{
		NodeID:            nodeID,
		AGIVersion:        version,
		IntelligenceLevel: intelligenceLevel,
		DomainKnowledge:   domainKnowledge,
		EvolutionEvent:    evolutionEventData,
		UpdateType:        "evolution",
	}

	// Create transaction
	txID := a.generateTransactionID("agi_evolution", nodeID)
	parentIDs := a.getRecentTransactionIDs(3)
	transaction := types.NewAGITransaction(txID, agiData, parentIDs)

	// Submit to consensus
	if err := a.consensusService.SubmitTransaction(ctx, transaction); err != nil {
		return fmt.Errorf("failed to submit AGI evolution transaction: %w", err)
	}

	// Update local state
	a.updateLocalAGIState(nodeID, intelligenceLevel, domainKnowledge, version)

	log.Printf("Recorded AGI evolution for node %s: trigger=%s, gain=%.2f, impact=%.2f",
		nodeID, triggerType, intelligenceGain, impact)

	return nil
}

// GetNetworkAGISnapshot returns the current network-wide AGI snapshot
func (a *AGIRecorder) GetNetworkAGISnapshot() *types.NetworkAGISnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.lastSnapshot == nil {
		return a.createNetworkSnapshot()
	}

	return a.lastSnapshot
}

// GetNodeAGIStates returns AGI states for all known nodes
func (a *AGIRecorder) GetNodeAGIStates() map[string]*types.NodeAGIState {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Return a copy
	result := make(map[string]*types.NodeAGIState)
	for nodeID, state := range a.networkAGIState {
		stateCopy := *state
		result[nodeID] = &stateCopy
	}

	return result
}

// GetDomainLeaders returns the nodes with highest expertise in each domain
func (a *AGIRecorder) GetDomainLeaders() map[string]string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	domainLeaders := make(map[string]string)
	domainExpertise := make(map[string]float64)

	// Find the node with highest expertise in each domain
	for nodeID, state := range a.networkAGIState {
		for domain, expertise := range state.DomainKnowledge {
			if expertise > domainExpertise[domain] {
				domainExpertise[domain] = expertise
				domainLeaders[domain] = nodeID
			}
		}
	}

	return domainLeaders
}

// RegisterAGIStateUpdate allows the AGI system to register state updates
func (a *AGIRecorder) RegisterAGIStateUpdate(nodeID string, intelligenceLevel float64, domainKnowledge map[string]float64, version int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Update or create the AGI state
	a.networkAGIState[nodeID] = &types.NodeAGIState{
		NodeID:            nodeID,
		IntelligenceLevel: intelligenceLevel,
		DomainKnowledge:   domainKnowledge,
		LastUpdate:        time.Now().Unix(),
		Version:           version,
		Reputation:        0.8,  // Would be calculated from performance
		TotalQueries:      1000, // Would be tracked
		SuccessRate:       0.95, // Would be calculated
	}

	log.Printf("AGI recorder updated state for node %s: intelligence=%.1f, version=%d",
		nodeID, intelligenceLevel, version)
}

// periodicAGIUpdates runs periodic AGI update recording
func (a *AGIRecorder) periodicAGIUpdates(ctx context.Context) {
	ticker := time.NewTicker(a.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopChan:
			return
		case <-ticker.C:
			// This would be triggered by external AGI updates
			// For now, we rely on explicit calls to RecordAGIUpdate
		}
	}
}

// periodicNetworkSnapshots creates periodic network-wide AGI snapshots
func (a *AGIRecorder) periodicNetworkSnapshots(ctx context.Context) {
	ticker := time.NewTicker(a.snapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopChan:
			return
		case <-ticker.C:
			if err := a.createAndRecordNetworkSnapshot(ctx); err != nil {
				log.Printf("Failed to create network AGI snapshot: %v", err)
			}
		}
	}
}

// createAndRecordNetworkSnapshot creates and records a network-wide AGI snapshot
func (a *AGIRecorder) createAndRecordNetworkSnapshot(ctx context.Context) error {
	snapshot := a.createNetworkSnapshot()

	// Serialize snapshot for blockchain storage
	snapshotData, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// Create transaction for network snapshot
	txID := a.generateTransactionID("network_snapshot", "network")
	parentIDs := a.getRecentTransactionIDs(3)

	transaction := types.NewTransaction(txID, string(snapshotData), parentIDs)
	transaction.Type = "agi_network_snapshot"
	transaction.Metadata["snapshot_type"] = "network_agi"
	transaction.Metadata["total_nodes"] = fmt.Sprintf("%d", snapshot.TotalNodes)
	transaction.Metadata["avg_intelligence"] = fmt.Sprintf("%.2f", snapshot.NetworkMetrics.AverageIntelligence)

	// Submit to consensus
	if err := a.consensusService.SubmitTransaction(ctx, transaction); err != nil {
		return fmt.Errorf("failed to submit network snapshot: %w", err)
	}

	// Update local snapshot
	a.mu.Lock()
	a.lastSnapshot = snapshot
	a.mu.Unlock()

	log.Printf("Recorded network AGI snapshot: %d nodes, avg intelligence %.2f",
		snapshot.TotalNodes, snapshot.NetworkMetrics.AverageIntelligence)

	return nil
}

// Helper methods

func (a *AGIRecorder) shouldRecordUpdate(nodeID string, intelligence float64, domains map[string]float64) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	existing, exists := a.networkAGIState[nodeID]
	if !exists {
		return true // Always record first update for a node
	}

	// Check intelligence change
	if abs(intelligence-existing.IntelligenceLevel) >= a.minIntelligenceChange {
		return true
	}

	// Check domain expertise changes
	for domain, expertise := range domains {
		if existingExpertise, exists := existing.DomainKnowledge[domain]; !exists ||
			abs(expertise-existingExpertise) >= a.minDomainChange {
			return true
		}
	}

	return false
}

func (a *AGIRecorder) extractAGIStateData(agiState interface{}) (float64, map[string]float64, int, error) {
	// This would extract data from the actual AGI state object
	// For now, return mock data - in practice this would interface with the AGI system
	return 25.5, map[string]float64{
		"technology": 30.0,
		"reasoning":  25.0,
		"language":   28.0,
	}, 1, nil
}

func (a *AGIRecorder) extractEvolutionData(evolutionEvent interface{}) (string, float64, []string, float64, string, error) {
	// Extract evolution event data - mock implementation
	return "learning_threshold", 2.5, []string{"reasoning", "technology"}, 0.3, "Reasoning capability breakthrough", nil
}

func (a *AGIRecorder) getCurrentAGIState(nodeID string) (float64, map[string]float64, int, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// First try to get from local cache
	if state, exists := a.networkAGIState[nodeID]; exists {
		return state.IntelligenceLevel, state.DomainKnowledge, state.Version, nil
	}

	// If not in cache, try to extract from the mock data (in a real implementation,
	// this would query the AGI system directly)
	log.Printf("AGI state not found in cache for node %s, using default values", nodeID)

	// Return reasonable default values for now
	// In a real implementation, this would interface with the actual AGI system
	defaultDomains := map[string]float64{
		"technology": 30.0,
		"reasoning":  25.0,
		"language":   28.0,
		"general":    26.0,
	}

	// Create and cache the default state
	defaultState := &types.NodeAGIState{
		NodeID:            nodeID,
		IntelligenceLevel: 27.0, // Average of domain scores
		DomainKnowledge:   defaultDomains,
		LastUpdate:        time.Now().Unix(),
		Version:           1,
		Reputation:        0.8,
		TotalQueries:      1000,
		SuccessRate:       0.95,
	}

	a.networkAGIState[nodeID] = defaultState

	return defaultState.IntelligenceLevel, defaultState.DomainKnowledge, defaultState.Version, nil
}

func (a *AGIRecorder) calculateLearningMetrics(nodeID string, agiState interface{}) *types.AGILearningMetrics {
	// Calculate learning metrics from AGI state
	return &types.AGILearningMetrics{
		ConceptsLearned:    50,
		PatternsIdentified: 25,
		LearningRate:       0.15,
		KnowledgeGrowth: map[string]float64{
			"technology": 0.2,
			"reasoning":  0.1,
		},
		LastLearningEvent: time.Now().Unix(),
		TotalInteractions: 1000,
	}
}

func (a *AGIRecorder) updateLocalAGIState(nodeID string, intelligence float64, domains map[string]float64, version int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.networkAGIState[nodeID] = &types.NodeAGIState{
		NodeID:            nodeID,
		IntelligenceLevel: intelligence,
		DomainKnowledge:   domains,
		LastUpdate:        time.Now().Unix(),
		Version:           version,
		Reputation:        0.8, // Would be calculated from performance
		TotalQueries:      1000,
		SuccessRate:       0.95,
	}
}

func (a *AGIRecorder) createNetworkSnapshot() *types.NetworkAGISnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Calculate network metrics
	totalIntelligence := 0.0
	domainDistribution := make(map[string]int)
	nodeCount := len(a.networkAGIState)

	for _, state := range a.networkAGIState {
		totalIntelligence += state.IntelligenceLevel
		for domain := range state.DomainKnowledge {
			domainDistribution[domain]++
		}
	}

	avgIntelligence := totalIntelligence / float64(nodeCount)
	if nodeCount == 0 {
		avgIntelligence = 0
	}

	// Calculate collective intelligence (network effect)
	collectiveIntelligence := avgIntelligence * (1.0 + float64(nodeCount)*0.01) // Network effect bonus

	return &types.NetworkAGISnapshot{
		Timestamp:     time.Now().Unix(),
		BlockHash:     "current_block_hash", // Would get from consensus
		TotalNodes:    nodeCount,
		NodeAGIStates: a.networkAGIState,
		NetworkMetrics: &types.NetworkAGIMetrics{
			AverageIntelligence:     avgIntelligence,
			TotalKnowledgeDomains:   len(domainDistribution),
			DomainDistribution:      domainDistribution,
			NetworkLearningRate:     0.12,
			CollectiveIntelligence:  collectiveIntelligence,
			KnowledgeSpecialization: a.calculateSpecialization(),
		},
		DomainLeaders: a.GetDomainLeaders(),
	}
}

func (a *AGIRecorder) calculateSpecialization() map[string]float64 {
	// Calculate how specialized each domain is across the network
	specialization := make(map[string]float64)

	for domain := range a.getDomains() {
		specialization[domain] = 0.5 // Mock specialization score
	}

	return specialization
}

func (a *AGIRecorder) getDomains() map[string]bool {
	domains := make(map[string]bool)
	for _, state := range a.networkAGIState {
		for domain := range state.DomainKnowledge {
			domains[domain] = true
		}
	}
	return domains
}

func (a *AGIRecorder) generateTransactionID(txType, nodeID string) string {
	data := fmt.Sprintf("%s_%s_%d", txType, nodeID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8])
}

func (a *AGIRecorder) getRecentTransactionIDs(count int) []string {
	// Would get recent transaction IDs from consensus service
	return []string{"parent1", "parent2", "parent3"}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
