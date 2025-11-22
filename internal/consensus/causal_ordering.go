package consensus

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// CausalOrder represents the causal ordering relationship
type CausalOrder string

const (
	CausalOrderBefore     CausalOrder = "before"     // A happens before B
	CausalOrderAfter      CausalOrder = "after"      // A happens after B
	CausalOrderConcurrent CausalOrder = "concurrent" // A and B are concurrent
)

// CausalRelation represents a causal relationship between two transactions
type CausalRelation struct {
	TransactionA string      `json:"transaction_a"`
	TransactionB string      `json:"transaction_b"`
	Relation     CausalOrder `json:"relation"`
	Confidence   float64     `json:"confidence"` // 0.0 to 1.0
	Evidence     []string    `json:"evidence"`   // List of evidence supporting this relation
	DetectedAt   time.Time   `json:"detected_at"`
}

// CausalOrderingManager manages causal ordering in the DAG
type CausalOrderingManager struct {
	dag              *types.DAG
	causalChains     map[string][]string         // transaction ID -> ordered list of causally dependent transactions
	vectorClocks     map[string]map[string]int64 // node ID -> vector clock
	causalRelations  map[string]*CausalRelation  // relation ID -> causal relation
	topologicalOrder []string                    // topologically sorted transaction IDs
	mu               sync.RWMutex
	lastOrderingTime time.Time
}

// VectorClock represents a vector clock for causal ordering
type VectorClock struct {
	Clock  map[string]int64 `json:"clock"`
	NodeID string           `json:"node_id"`
}

// NewCausalOrderingManager creates a new causal ordering manager
func NewCausalOrderingManager(dag *types.DAG) *CausalOrderingManager {
	return &CausalOrderingManager{
		dag:              dag,
		causalChains:     make(map[string][]string),
		vectorClocks:     make(map[string]map[string]int64),
		causalRelations:  make(map[string]*CausalRelation),
		topologicalOrder: make([]string, 0),
	}
}

// UpdateCausalOrdering updates the causal ordering when a new transaction is added
func (com *CausalOrderingManager) UpdateCausalOrdering(tx *types.Transaction) error {
	com.mu.Lock()
	defer com.mu.Unlock()

	log.Printf("Updating causal ordering for transaction %s", tx.ID)

	// Step 1: Analyze direct causal dependencies
	directDeps := com.analyzeDirectDependencies(tx)

	// Step 2: Analyze implicit causal dependencies
	implicitDeps := com.analyzeImplicitDependencies(tx)

	// Step 3: Combine and validate dependencies
	allDeps := com.combineDependencies(directDeps, implicitDeps)

	// Step 4: Update causal chains
	com.updateCausalChains(tx.ID, allDeps)

	// Step 5: Detect and establish causal relations
	com.establishCausalRelations(tx, allDeps)

	// Step 6: Update topological ordering
	return com.updateTopologicalOrdering()
}

// analyzeDirectDependencies analyzes direct causal dependencies from parent IDs
func (com *CausalOrderingManager) analyzeDirectDependencies(tx *types.Transaction) []string {
	var dependencies []string

	// Direct dependencies are explicitly listed in ParentIDs
	for _, parentID := range tx.ParentIDs {
		if com.isValidDependency(parentID, tx.ID) {
			dependencies = append(dependencies, parentID)
		}
	}

	log.Printf("Transaction %s has %d direct dependencies: %v", tx.ID, len(dependencies), dependencies)
	return dependencies
}

// analyzeImplicitDependencies analyzes implicit causal dependencies
func (com *CausalOrderingManager) analyzeImplicitDependencies(tx *types.Transaction) []string {
	var dependencies []string

	// For economic transactions, find dependencies based on account interactions
	if tx.Type == types.TransactionTypeEconomic && tx.EconomicData != nil {
		dependencies = append(dependencies, com.findAccountDependencies(tx)...)
	}

	// Find dependencies based on data dependencies
	dependencies = append(dependencies, com.findDataDependencies(tx)...)

	// Find dependencies based on temporal ordering constraints
	dependencies = append(dependencies, com.findTemporalDependencies(tx)...)

	log.Printf("Transaction %s has %d implicit dependencies: %v", tx.ID, len(dependencies), dependencies)
	return dependencies
}

// findAccountDependencies finds dependencies based on account interactions
func (com *CausalOrderingManager) findAccountDependencies(tx *types.Transaction) []string {
	var dependencies []string

	accounts := make(map[string]bool)
	if tx.EconomicData.FromAccount != "" {
		accounts[tx.EconomicData.FromAccount] = true
	}
	if tx.EconomicData.ToAccount != "" {
		accounts[tx.EconomicData.ToAccount] = true
	}

	// Find the most recent transaction affecting these accounts
	for _, node := range com.dag.Nodes {
		if node.Transaction.ID == tx.ID {
			continue
		}

		if node.Transaction.Type == types.TransactionTypeEconomic && node.Transaction.EconomicData != nil {
			affectedAccounts := make(map[string]bool)
			if node.Transaction.EconomicData.FromAccount != "" {
				affectedAccounts[node.Transaction.EconomicData.FromAccount] = true
			}
			if node.Transaction.EconomicData.ToAccount != "" {
				affectedAccounts[node.Transaction.EconomicData.ToAccount] = true
			}

			// Check for overlap
			for account := range accounts {
				if affectedAccounts[account] && node.Transaction.Timestamp < tx.Timestamp {
					dependencies = append(dependencies, node.Transaction.ID)
					break
				}
			}
		}
	}

	return dependencies
}

// findDataDependencies finds dependencies based on data interactions
func (com *CausalOrderingManager) findDataDependencies(tx *types.Transaction) []string {
	var dependencies []string

	// Check if this transaction's data references other transactions
	for _, node := range com.dag.Nodes {
		if node.Transaction.ID == tx.ID {
			continue
		}

		// Simple heuristic: if transaction data contains reference to another transaction
		if com.containsReference(tx.Data, node.Transaction.ID) {
			dependencies = append(dependencies, node.Transaction.ID)
		}
	}

	return dependencies
}

// findTemporalDependencies finds dependencies based on temporal constraints
func (com *CausalOrderingManager) findTemporalDependencies(tx *types.Transaction) []string {
	var dependencies []string

	// Find transactions that must happen before this one based on business logic
	for _, node := range com.dag.Nodes {
		if node.Transaction.ID == tx.ID {
			continue
		}

		// If there's a clear temporal dependency (e.g., payment before reward)
		if com.hasTemporalDependency(node.Transaction, tx) {
			dependencies = append(dependencies, node.Transaction.ID)
		}
	}

	return dependencies
}

// combineDependencies combines and deduplicates dependencies
func (com *CausalOrderingManager) combineDependencies(direct, implicit []string) []string {
	depSet := make(map[string]bool)

	for _, dep := range direct {
		depSet[dep] = true
	}
	for _, dep := range implicit {
		depSet[dep] = true
	}

	var result []string
	for dep := range depSet {
		result = append(result, dep)
	}

	return result
}

// updateCausalChains updates the causal chains with new dependencies
func (com *CausalOrderingManager) updateCausalChains(txID string, dependencies []string) {
	// Build the causal chain for this transaction
	chain := make([]string, 0)

	// Add all dependencies to the chain
	for _, dep := range dependencies {
		chain = append(chain, dep)

		// Add transitive dependencies
		if depChain, exists := com.causalChains[dep]; exists {
			chain = append(chain, depChain...)
		}
	}

	// Remove duplicates and sort
	chainSet := make(map[string]bool)
	for _, txID := range chain {
		chainSet[txID] = true
	}

	var uniqueChain []string
	for txID := range chainSet {
		uniqueChain = append(uniqueChain, txID)
	}

	// Sort by timestamp for consistent ordering
	sort.Slice(uniqueChain, func(i, j int) bool {
		nodeI, existsI := com.dag.Nodes[uniqueChain[i]]
		nodeJ, existsJ := com.dag.Nodes[uniqueChain[j]]

		if !existsI || !existsJ {
			return uniqueChain[i] < uniqueChain[j] // Fallback to lexicographic
		}

		return nodeI.Transaction.Timestamp < nodeJ.Transaction.Timestamp
	})

	com.causalChains[txID] = uniqueChain
	log.Printf("Updated causal chain for %s: %d dependencies", txID, len(uniqueChain))
}

// establishCausalRelations establishes causal relations between transactions
func (com *CausalOrderingManager) establishCausalRelations(tx *types.Transaction, dependencies []string) {
	for _, depID := range dependencies {
		relationID := fmt.Sprintf("%s_before_%s", depID, tx.ID)

		relation := &CausalRelation{
			TransactionA: depID,
			TransactionB: tx.ID,
			Relation:     CausalOrderBefore,
			Confidence:   com.calculateCausalConfidence(depID, tx.ID),
			Evidence:     com.collectCausalEvidence(depID, tx.ID),
			DetectedAt:   time.Now(),
		}

		com.causalRelations[relationID] = relation
	}

	// Also establish concurrent relations with transactions that don't have causal dependencies
	for _, node := range com.dag.Nodes {
		otherTxID := node.Transaction.ID
		if otherTxID == tx.ID {
			continue
		}

		if !com.hasCausalRelation(otherTxID, tx.ID) && !com.hasCausalRelation(tx.ID, otherTxID) {
			relationID := fmt.Sprintf("%s_concurrent_%s", tx.ID, otherTxID)

			relation := &CausalRelation{
				TransactionA: tx.ID,
				TransactionB: otherTxID,
				Relation:     CausalOrderConcurrent,
				Confidence:   0.8, // Lower confidence for concurrent relations
				Evidence:     []string{"no_causal_dependency_detected"},
				DetectedAt:   time.Now(),
			}

			com.causalRelations[relationID] = relation
		}
	}
}

// updateTopologicalOrdering updates the topological ordering of transactions
func (com *CausalOrderingManager) updateTopologicalOrdering() error {
	// Build adjacency list for topological sorting
	adjList := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize all nodes
	for txID := range com.dag.Nodes {
		adjList[txID] = make([]string, 0)
		inDegree[txID] = 0
	}

	// Build the graph from causal relations
	for _, relation := range com.causalRelations {
		if relation.Relation == CausalOrderBefore {
			adjList[relation.TransactionA] = append(adjList[relation.TransactionA], relation.TransactionB)
			inDegree[relation.TransactionB]++
		}
	}

	// Kahn's algorithm for topological sorting
	queue := make([]string, 0)

	// Find all nodes with no incoming edges
	for txID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, txID)
		}
	}

	var result []string

	for len(queue) > 0 {
		// Remove a node from the queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// For each neighbor of the current node
		for _, neighbor := range adjList[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(result) != len(com.dag.Nodes) {
		return fmt.Errorf("cycle detected in causal ordering")
	}

	com.topologicalOrder = result
	com.lastOrderingTime = time.Now()

	log.Printf("Updated topological ordering with %d transactions", len(result))
	return nil
}

// Helper functions

// isValidDependency checks if a dependency is valid
func (com *CausalOrderingManager) isValidDependency(depID, txID string) bool {
	if depID == txID {
		return false // Self-dependency
	}

	depNode, exists := com.dag.Nodes[depID]
	if !exists {
		return false // Dependency doesn't exist
	}

	// Check for temporal consistency (dependency should be earlier)
	if node, exists := com.dag.Nodes[txID]; exists {
		return depNode.Transaction.Timestamp < node.Transaction.Timestamp
	}

	return true
}

// containsReference checks if data contains a reference to another transaction
func (com *CausalOrderingManager) containsReference(data, txID string) bool {
	// Simple string containment check - could be more sophisticated
	return len(data) > 0 && len(txID) > 0 &&
		(data == txID || (len(data) > len(txID) &&
			(data[:len(txID)] == txID || data[len(data)-len(txID):] == txID)))
}

// hasTemporalDependency checks if there's a temporal dependency between transactions
func (com *CausalOrderingManager) hasTemporalDependency(earlier, later *types.Transaction) bool {
	// Economic logic: payment must come before reward
	if earlier.Type == types.TransactionTypeEconomic && later.Type == types.TransactionTypeReward {
		if earlier.EconomicData != nil && later.Metadata != nil {
			if paymentID, exists := later.Metadata["payment_tx_id"]; exists {
				return paymentID == earlier.ID
			}
		}
	}

	// Query processing: query must come before response
	if earlier.Type == types.TransactionTypeQuery && later.Type == types.TransactionTypeReward {
		if later.Metadata != nil {
			if queryID, exists := later.Metadata["query_id"]; exists {
				return queryID == earlier.ID
			}
		}
	}

	return false
}

// calculateCausalConfidence calculates confidence in a causal relationship
func (com *CausalOrderingManager) calculateCausalConfidence(txA, txB string) float64 {
	confidence := 0.5 // Base confidence

	nodeA, existsA := com.dag.Nodes[txA]
	nodeB, existsB := com.dag.Nodes[txB]

	if !existsA || !existsB {
		return 0.0
	}

	// Higher confidence for explicit parent relationships
	for _, parentID := range nodeB.Transaction.ParentIDs {
		if parentID == txA {
			confidence += 0.4
			break
		}
	}

	// Higher confidence for clear temporal ordering
	timeDiff := nodeB.Transaction.Timestamp - nodeA.Transaction.Timestamp
	if timeDiff > 0 {
		confidence += 0.2
	}

	// Higher confidence for business logic dependencies
	if com.hasTemporalDependency(nodeA.Transaction, nodeB.Transaction) {
		confidence += 0.3
	}

	return min(1.0, confidence)
}

// collectCausalEvidence collects evidence for a causal relationship
func (com *CausalOrderingManager) collectCausalEvidence(txA, txB string) []string {
	var evidence []string

	nodeA, existsA := com.dag.Nodes[txA]
	nodeB, existsB := com.dag.Nodes[txB]

	if !existsA || !existsB {
		return evidence
	}

	// Check for explicit parent relationship
	for _, parentID := range nodeB.Transaction.ParentIDs {
		if parentID == txA {
			evidence = append(evidence, "explicit_parent_relationship")
			break
		}
	}

	// Check for temporal ordering
	if nodeB.Transaction.Timestamp > nodeA.Transaction.Timestamp {
		evidence = append(evidence, "temporal_ordering")
	}

	// Check for business logic dependencies
	if com.hasTemporalDependency(nodeA.Transaction, nodeB.Transaction) {
		evidence = append(evidence, "business_logic_dependency")
	}

	// Check for account dependencies
	if nodeA.Transaction.Type == types.TransactionTypeEconomic &&
		nodeB.Transaction.Type == types.TransactionTypeEconomic &&
		nodeA.Transaction.EconomicData != nil &&
		nodeB.Transaction.EconomicData != nil {

		if nodeA.Transaction.EconomicData.ToAccount == nodeB.Transaction.EconomicData.FromAccount ||
			nodeA.Transaction.EconomicData.FromAccount == nodeB.Transaction.EconomicData.FromAccount {
			evidence = append(evidence, "account_dependency")
		}
	}

	return evidence
}

// hasCausalRelation checks if there's already a causal relation between two transactions
func (com *CausalOrderingManager) hasCausalRelation(txA, txB string) bool {
	relationID1 := fmt.Sprintf("%s_before_%s", txA, txB)
	relationID2 := fmt.Sprintf("%s_before_%s", txB, txA)
	relationID3 := fmt.Sprintf("%s_concurrent_%s", txA, txB)
	relationID4 := fmt.Sprintf("%s_concurrent_%s", txB, txA)

	_, exists1 := com.causalRelations[relationID1]
	_, exists2 := com.causalRelations[relationID2]
	_, exists3 := com.causalRelations[relationID3]
	_, exists4 := com.causalRelations[relationID4]

	return exists1 || exists2 || exists3 || exists4
}

// Public query methods

// GetCausalOrder returns the causal relationship between two transactions
func (com *CausalOrderingManager) GetCausalOrder(txA, txB string) (CausalOrder, float64) {
	com.mu.RLock()
	defer com.mu.RUnlock()

	relationID1 := fmt.Sprintf("%s_before_%s", txA, txB)
	relationID2 := fmt.Sprintf("%s_before_%s", txB, txA)
	relationID3 := fmt.Sprintf("%s_concurrent_%s", txA, txB)
	relationID4 := fmt.Sprintf("%s_concurrent_%s", txB, txA)

	if relation, exists := com.causalRelations[relationID1]; exists {
		return CausalOrderBefore, relation.Confidence
	}
	if relation, exists := com.causalRelations[relationID2]; exists {
		return CausalOrderAfter, relation.Confidence
	}
	if relation, exists := com.causalRelations[relationID3]; exists {
		return CausalOrderConcurrent, relation.Confidence
	}
	if relation, exists := com.causalRelations[relationID4]; exists {
		return CausalOrderConcurrent, relation.Confidence
	}

	return CausalOrderConcurrent, 0.0 // Default to concurrent with low confidence
}

// GetTopologicalOrder returns the current topological ordering
func (com *CausalOrderingManager) GetTopologicalOrder() []string {
	com.mu.RLock()
	defer com.mu.RUnlock()

	result := make([]string, len(com.topologicalOrder))
	copy(result, com.topologicalOrder)
	return result
}

// GetCausalChain returns the causal chain for a transaction
func (com *CausalOrderingManager) GetCausalChain(txID string) []string {
	com.mu.RLock()
	defer com.mu.RUnlock()

	if chain, exists := com.causalChains[txID]; exists {
		result := make([]string, len(chain))
		copy(result, chain)
		return result
	}

	return []string{}
}

// GetCausalRelations returns all causal relations
func (com *CausalOrderingManager) GetCausalRelations() map[string]*CausalRelation {
	com.mu.RLock()
	defer com.mu.RUnlock()

	relations := make(map[string]*CausalRelation)
	for id, relation := range com.causalRelations {
		relations[id] = relation
	}
	return relations
}

// Helper function for min
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
