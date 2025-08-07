package consensus

import (
	"context"
	"testing"
	"time"
)

func TestAGIRecorderStateUpdate(t *testing.T) {
	// Create consensus service
	cs := NewConsensusService()

	// Create AGI recorder
	nodeID := "test-node-123"
	recorder := NewAGIRecorder(cs, nodeID)

	// Test RegisterAGIStateUpdate
	intelligenceLevel := 42.5
	domainKnowledge := map[string]float64{
		"technology": 45.0,
		"reasoning":  40.0,
		"language":   42.0,
	}
	version := 2

	// Register the state update
	recorder.RegisterAGIStateUpdate(nodeID, intelligenceLevel, domainKnowledge, version)

	// Verify the state was stored
	retrievedIntelligence, retrievedDomains, retrievedVersion, err := recorder.getCurrentAGIState(nodeID)
	if err != nil {
		t.Fatalf("Failed to retrieve AGI state: %v", err)
	}

	// Check intelligence level
	if retrievedIntelligence != intelligenceLevel {
		t.Errorf("Expected intelligence level %.1f, got %.1f", intelligenceLevel, retrievedIntelligence)
	}

	// Check version
	if retrievedVersion != version {
		t.Errorf("Expected version %d, got %d", version, retrievedVersion)
	}

	// Check domain knowledge
	for domain, expectedExpertise := range domainKnowledge {
		if actualExpertise, exists := retrievedDomains[domain]; !exists {
			t.Errorf("Domain %s not found in retrieved state", domain)
		} else if actualExpertise != expectedExpertise {
			t.Errorf("Domain %s: expected expertise %.1f, got %.1f", domain, expectedExpertise, actualExpertise)
		}
	}

	t.Logf("✅ AGI state successfully registered and retrieved for node %s", nodeID)
}

func TestAGIRecorderEvolutionWithState(t *testing.T) {
	// Create consensus service
	cs := NewConsensusService()

	// Create AGI recorder
	nodeID := "test-node-456"
	recorder := NewAGIRecorder(cs, nodeID)

	// Register initial state
	recorder.RegisterAGIStateUpdate(nodeID, 30.0, map[string]float64{
		"technology": 32.0,
		"reasoning":  28.0,
	}, 1)

	// Start the recorder
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := recorder.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start AGI recorder: %v", err)
	}
	defer recorder.Stop()

	// Test recording an evolution event
	evolutionEvent := map[string]interface{}{
		"trigger_type":      "learning_threshold",
		"intelligence_gain": 2.5,
		"domains_affected":  []string{"reasoning", "technology"},
		"impact":            0.3,
		"description":       "Test evolution event",
	}

	err = recorder.RecordAGIEvolution(ctx, nodeID, evolutionEvent)
	if err != nil {
		t.Errorf("Failed to record AGI evolution: %v", err)
	} else {
		t.Logf("✅ AGI evolution successfully recorded for node %s", nodeID)
	}
}

func TestAGIRecorderDefaultState(t *testing.T) {
	// Create consensus service
	cs := NewConsensusService()

	// Create AGI recorder
	nodeID := "test-node-789"
	recorder := NewAGIRecorder(cs, nodeID)

	// Try to get state for a node that hasn't been registered
	// Should return default values instead of error
	intelligence, domains, version, err := recorder.getCurrentAGIState(nodeID)
	if err != nil {
		t.Fatalf("Expected default state, got error: %v", err)
	}

	// Verify default values are reasonable
	if intelligence <= 0 {
		t.Errorf("Expected positive intelligence level, got %.1f", intelligence)
	}

	if len(domains) == 0 {
		t.Error("Expected default domains to be populated")
	}

	if version <= 0 {
		t.Errorf("Expected positive version, got %d", version)
	}

	t.Logf("✅ Default AGI state successfully created for node %s: intelligence=%.1f, domains=%d, version=%d",
		nodeID, intelligence, len(domains), version)
}
