package consensus

import (
	"fmt"
	"log"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// ConflictType represents the type of conflict between transactions
type ConflictType string

const (
	ConflictTypeDoubleSpend     ConflictType = "double_spend"
	ConflictTypeResourceLock    ConflictType = "resource_lock"
	ConflictTypeCausalViolation ConflictType = "causal_violation"
	ConflictTypeTemporalOrder   ConflictType = "temporal_order"
)

// ConflictInfo represents information about a detected conflict
type ConflictInfo struct {
	ID               string         `json:"id"`
	Type             ConflictType   `json:"type"`
	TransactionA     string         `json:"transaction_a"`
	TransactionB     string         `json:"transaction_b"`
	Description      string         `json:"description"`
	Severity         float64        `json:"severity"` // 0.0 to 1.0
	DetectedAt       time.Time      `json:"detected_at"`
	ResolvedAt       *time.Time     `json:"resolved_at,omitempty"`
	ResolutionMethod string         `json:"resolution_method,omitempty"`
	Details          map[string]any `json:"details,omitempty"`
}

// ConflictResolver manages conflict detection and resolution in the DAG
type ConflictResolver struct {
	dag             *types.DAG
	conflicts       map[string]*ConflictInfo // conflict ID -> conflict info
	resourceLocks   map[string][]string      // resource -> list of transaction IDs using it
	causalGraph     map[string][]string      // transaction ID -> dependencies
	mu              sync.RWMutex
	resolutionRules []ConflictResolutionRule
}

// ConflictResolutionRule defines how to resolve a specific type of conflict
type ConflictResolutionRule struct {
	ConflictType ConflictType
	Priority     int                                             // Higher number = higher priority
	ResolverFunc func(*ConflictInfo, *types.DAG) (string, error) // Returns winning transaction ID
	Description  string
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(dag *types.DAG) *ConflictResolver {
	cr := &ConflictResolver{
		dag:           dag,
		conflicts:     make(map[string]*ConflictInfo),
		resourceLocks: make(map[string][]string),
		causalGraph:   make(map[string][]string),
		resolutionRules: []ConflictResolutionRule{
			{
				ConflictType: ConflictTypeDoubleSpend,
				Priority:     100,
				ResolverFunc: resolveDoubleSpendByTimestamp,
				Description:  "Resolve double spend conflicts using timestamp ordering",
			},
			{
				ConflictType: ConflictTypeResourceLock,
				Priority:     90,
				ResolverFunc: resolveResourceLockByReputation,
				Description:  "Resolve resource conflicts using reputation weighting",
			},
			{
				ConflictType: ConflictTypeCausalViolation,
				Priority:     95,
				ResolverFunc: resolveCausalViolationByDAGOrder,
				Description:  "Resolve causal violations using DAG ordering",
			},
			{
				ConflictType: ConflictTypeTemporalOrder,
				Priority:     85,
				ResolverFunc: resolveTemporalOrderByTimestamp,
				Description:  "Resolve temporal order conflicts by timestamp",
			},
		},
	}

	// Sort resolution rules by priority
	sort.Slice(cr.resolutionRules, func(i, j int) bool {
		return cr.resolutionRules[i].Priority > cr.resolutionRules[j].Priority
	})

	return cr
}

// DetectConflicts detects conflicts in the DAG for a new transaction
func (cr *ConflictResolver) DetectConflicts(newTx *types.Transaction) []*ConflictInfo {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	var conflicts []*ConflictInfo

	// Check for various types of conflicts
	conflicts = append(conflicts, cr.detectDoubleSpendConflicts(newTx)...)
	conflicts = append(conflicts, cr.detectResourceLockConflicts(newTx)...)
	conflicts = append(conflicts, cr.detectCausalViolations(newTx)...)
	conflicts = append(conflicts, cr.detectTemporalOrderConflicts(newTx)...)

	// Store detected conflicts
	for _, conflict := range conflicts {
		cr.conflicts[conflict.ID] = conflict
		log.Printf("Conflict detected: %s between transactions %s and %s (%s)",
			conflict.Type, conflict.TransactionA, conflict.TransactionB, conflict.Description)
	}

	return conflicts
}

// detectDoubleSpendConflicts detects double spending conflicts
func (cr *ConflictResolver) detectDoubleSpendConflicts(newTx *types.Transaction) []*ConflictInfo {
	var conflicts []*ConflictInfo

	if newTx.Type != types.TransactionTypeEconomic || newTx.EconomicData == nil {
		return conflicts
	}

	fromAccount := newTx.EconomicData.FromAccount
	if fromAccount == "" {
		return conflicts // Minting transactions don't have double spend conflicts
	}

	// Check against all non-finalized transactions
	for _, node := range cr.dag.Nodes {
		existingTx := node.Transaction
		if existingTx.ID == newTx.ID || existingTx.Finalized {
			continue
		}

		if existingTx.Type == types.TransactionTypeEconomic &&
			existingTx.EconomicData != nil &&
			existingTx.EconomicData.FromAccount == fromAccount {

			// Calculate total amount being spent
			newAmount := new(big.Int).Add(newTx.EconomicData.Amount, newTx.EconomicData.Fee)
			existingAmount := new(big.Int).Add(existingTx.EconomicData.Amount, existingTx.EconomicData.Fee)

			conflict := &ConflictInfo{
				ID:           fmt.Sprintf("double_spend_%s_%s", newTx.ID, existingTx.ID),
				Type:         ConflictTypeDoubleSpend,
				TransactionA: newTx.ID,
				TransactionB: existingTx.ID,
				Description:  fmt.Sprintf("Double spend from account %s", fromAccount),
				Severity:     1.0, // Double spending is always high severity
				DetectedAt:   time.Now(),
				Details: map[string]any{
					"account":         fromAccount,
					"new_amount":      newAmount.String(),
					"existing_amount": existingAmount.String(),
				},
			}
			conflicts = append(conflicts, conflict)
		}
	}

	return conflicts
}

// detectResourceLockConflicts detects conflicts over shared resources
func (cr *ConflictResolver) detectResourceLockConflicts(newTx *types.Transaction) []*ConflictInfo {
	var conflicts []*ConflictInfo

	// Extract resources used by the new transaction
	resources := cr.extractTransactionResources(newTx)

	for resource := range resources {
		if existingTxs, exists := cr.resourceLocks[resource]; exists {
			for _, existingTxID := range existingTxs {
				// Check if the existing transaction is still active
				if node, exists := cr.dag.Nodes[existingTxID]; exists && !node.Transaction.Finalized {
					conflict := &ConflictInfo{
						ID:           fmt.Sprintf("resource_lock_%s_%s_%s", resource, newTx.ID, existingTxID),
						Type:         ConflictTypeResourceLock,
						TransactionA: newTx.ID,
						TransactionB: existingTxID,
						Description:  fmt.Sprintf("Resource lock conflict on %s", resource),
						Severity:     0.7,
						DetectedAt:   time.Now(),
						Details: map[string]any{
							"resource": resource,
						},
					}
					conflicts = append(conflicts, conflict)
				}
			}
		}

		// Add the new transaction to resource locks
		cr.resourceLocks[resource] = append(cr.resourceLocks[resource], newTx.ID)
	}

	return conflicts
}

// detectCausalViolations detects violations of causal ordering
func (cr *ConflictResolver) detectCausalViolations(newTx *types.Transaction) []*ConflictInfo {
	var conflicts []*ConflictInfo

	// Build causal dependencies for the new transaction
	dependencies := cr.buildCausalDependencies(newTx)
	cr.causalGraph[newTx.ID] = dependencies

	// Check for causal violations
	for _, depID := range dependencies {
		if depNode, exists := cr.dag.Nodes[depID]; exists {
			// Check if dependency timestamp is after this transaction
			if depNode.Transaction.Timestamp > newTx.Timestamp {
				conflict := &ConflictInfo{
					ID:           fmt.Sprintf("causal_violation_%s_%s", newTx.ID, depID),
					Type:         ConflictTypeCausalViolation,
					TransactionA: newTx.ID,
					TransactionB: depID,
					Description:  fmt.Sprintf("Causal violation: transaction depends on future transaction"),
					Severity:     0.9,
					DetectedAt:   time.Now(),
					Details: map[string]any{
						"tx_timestamp":  newTx.Timestamp,
						"dep_timestamp": depNode.Transaction.Timestamp,
					},
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

// detectTemporalOrderConflicts detects temporal ordering conflicts
func (cr *ConflictResolver) detectTemporalOrderConflicts(newTx *types.Transaction) []*ConflictInfo {
	var conflicts []*ConflictInfo

	// Check for transactions that are too close in time but claim different ordering
	timeWindow := int64(5) // 5 second window

	for _, node := range cr.dag.Nodes {
		existingTx := node.Transaction
		if existingTx.ID == newTx.ID || existingTx.Finalized {
			continue
		}

		timeDiff := newTx.Timestamp - existingTx.Timestamp
		if timeDiff < timeWindow && timeDiff > -timeWindow {
			// Transactions are very close in time, check for ordering conflicts
			if cr.hasOrderingConflict(newTx, existingTx) {
				conflict := &ConflictInfo{
					ID:           fmt.Sprintf("temporal_order_%s_%s", newTx.ID, existingTx.ID),
					Type:         ConflictTypeTemporalOrder,
					TransactionA: newTx.ID,
					TransactionB: existingTx.ID,
					Description:  "Temporal ordering conflict between near-simultaneous transactions",
					Severity:     0.5,
					DetectedAt:   time.Now(),
					Details: map[string]any{
						"time_diff":   timeDiff,
						"time_window": timeWindow,
					},
				}
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

// ResolveConflicts resolves detected conflicts using resolution rules
func (cr *ConflictResolver) ResolveConflicts(conflicts []*ConflictInfo) error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	for _, conflict := range conflicts {
		if conflict.ResolvedAt != nil {
			continue // Already resolved
		}

		// Find appropriate resolution rule
		var rule *ConflictResolutionRule
		for _, r := range cr.resolutionRules {
			if r.ConflictType == conflict.Type {
				rule = &r
				break
			}
		}

		if rule == nil {
			log.Printf("No resolution rule found for conflict type %s", conflict.Type)
			continue
		}

		// Apply resolution rule
		winnerID, err := rule.ResolverFunc(conflict, cr.dag)
		if err != nil {
			log.Printf("Failed to resolve conflict %s: %v", conflict.ID, err)
			continue
		}

		// Mark conflict as resolved
		now := time.Now()
		conflict.ResolvedAt = &now
		conflict.ResolutionMethod = rule.Description

		// Handle the resolution
		loserID := conflict.TransactionB
		if winnerID == conflict.TransactionB {
			loserID = conflict.TransactionA
		}

		log.Printf("Conflict %s resolved: winner=%s, loser=%s, method=%s",
			conflict.ID, winnerID, loserID, rule.Description)

		// Remove losing transaction from DAG
		if err := cr.removeLosingTransaction(loserID); err != nil {
			log.Printf("Failed to remove losing transaction %s: %v", loserID, err)
		}
	}

	return nil
}

// extractTransactionResources extracts resources used by a transaction
func (cr *ConflictResolver) extractTransactionResources(tx *types.Transaction) map[string]bool {
	resources := make(map[string]bool)

	if tx.Type == types.TransactionTypeEconomic && tx.EconomicData != nil {
		// Account resources
		if tx.EconomicData.FromAccount != "" {
			resources[fmt.Sprintf("account:%s", tx.EconomicData.FromAccount)] = true
		}
		if tx.EconomicData.ToAccount != "" {
			resources[fmt.Sprintf("account:%s", tx.EconomicData.ToAccount)] = true
		}
	}

	// Add other resource types as needed
	return resources
}

// buildCausalDependencies builds causal dependencies for a transaction
func (cr *ConflictResolver) buildCausalDependencies(tx *types.Transaction) []string {
	var dependencies []string

	// Add explicit parent dependencies
	dependencies = append(dependencies, tx.ParentIDs...)

	// Add implicit dependencies based on transaction type
	if tx.Type == types.TransactionTypeEconomic && tx.EconomicData != nil {
		// Economic transactions depend on the last transaction affecting the same accounts
		if tx.EconomicData.FromAccount != "" {
			if lastTx := cr.findLastTransactionForAccount(tx.EconomicData.FromAccount, tx.ID); lastTx != "" {
				dependencies = append(dependencies, lastTx)
			}
		}
	}

	return dependencies
}

// findLastTransactionForAccount finds the last transaction affecting an account
func (cr *ConflictResolver) findLastTransactionForAccount(account, excludeID string) string {
	var lastTx string
	var lastTimestamp int64

	for _, node := range cr.dag.Nodes {
		tx := node.Transaction
		if tx.ID == excludeID {
			continue
		}

		if tx.Type == types.TransactionTypeEconomic && tx.EconomicData != nil {
			if tx.EconomicData.FromAccount == account || tx.EconomicData.ToAccount == account {
				if tx.Timestamp > lastTimestamp {
					lastTimestamp = tx.Timestamp
					lastTx = tx.ID
				}
			}
		}
	}

	return lastTx
}

// hasOrderingConflict checks if two transactions have an ordering conflict
func (cr *ConflictResolver) hasOrderingConflict(txA, txB *types.Transaction) bool {
	// Check if transactions affect the same resources
	resourcesA := cr.extractTransactionResources(txA)
	resourcesB := cr.extractTransactionResources(txB)

	for resource := range resourcesA {
		if resourcesB[resource] {
			return true
		}
	}

	return false
}

// removeLosingTransaction removes a losing transaction from the DAG
func (cr *ConflictResolver) removeLosingTransaction(txID string) error {
	// Remove from DAG
	delete(cr.dag.Nodes, txID)

	// Clean up resource locks
	for resource, txList := range cr.resourceLocks {
		for i, id := range txList {
			if id == txID {
				cr.resourceLocks[resource] = append(txList[:i], txList[i+1:]...)
				break
			}
		}
	}

	// Clean up causal dependencies
	delete(cr.causalGraph, txID)

	log.Printf("Removed losing transaction %s from DAG", txID)
	return nil
}

// Resolution rule implementations

// resolveDoubleSpendByTimestamp resolves double spend conflicts using timestamp ordering
func resolveDoubleSpendByTimestamp(conflict *ConflictInfo, dag *types.DAG) (string, error) {
	nodeA, existsA := dag.Nodes[conflict.TransactionA]
	nodeB, existsB := dag.Nodes[conflict.TransactionB]

	if !existsA || !existsB {
		return "", fmt.Errorf("one or both transactions not found in DAG")
	}

	// Earlier transaction wins
	if nodeA.Transaction.Timestamp < nodeB.Transaction.Timestamp {
		return conflict.TransactionA, nil
	}
	return conflict.TransactionB, nil
}

// resolveResourceLockByReputation resolves resource conflicts using reputation
func resolveResourceLockByReputation(conflict *ConflictInfo, dag *types.DAG) (string, error) {
	nodeA, existsA := dag.Nodes[conflict.TransactionA]
	nodeB, existsB := dag.Nodes[conflict.TransactionB]

	if !existsA || !existsB {
		return "", fmt.Errorf("one or both transactions not found in DAG")
	}

	// Higher reputation wins (using node score as proxy for reputation)
	if nodeA.Score > nodeB.Score {
		return conflict.TransactionA, nil
	}
	return conflict.TransactionB, nil
}

// resolveCausalViolationByDAGOrder resolves causal violations using DAG ordering
func resolveCausalViolationByDAGOrder(conflict *ConflictInfo, dag *types.DAG) (string, error) {
	nodeA, existsA := dag.Nodes[conflict.TransactionA]
	nodeB, existsB := dag.Nodes[conflict.TransactionB]

	if !existsA || !existsB {
		return "", fmt.Errorf("one or both transactions not found in DAG")
	}

	// Transaction that respects causal ordering wins
	// In this case, we favor the earlier transaction
	if nodeA.Transaction.Timestamp < nodeB.Transaction.Timestamp {
		return conflict.TransactionA, nil
	}
	return conflict.TransactionB, nil
}

// resolveTemporalOrderByTimestamp resolves temporal order conflicts by timestamp
func resolveTemporalOrderByTimestamp(conflict *ConflictInfo, dag *types.DAG) (string, error) {
	nodeA, existsA := dag.Nodes[conflict.TransactionA]
	nodeB, existsB := dag.Nodes[conflict.TransactionB]

	if !existsA || !existsB {
		return "", fmt.Errorf("one or both transactions not found in DAG")
	}

	// Use timestamp to break ties, but also consider transaction ID for determinism
	if nodeA.Transaction.Timestamp != nodeB.Transaction.Timestamp {
		if nodeA.Transaction.Timestamp < nodeB.Transaction.Timestamp {
			return conflict.TransactionA, nil
		}
		return conflict.TransactionB, nil
	}

	// If timestamps are equal, use lexicographic ordering of transaction IDs
	if nodeA.Transaction.ID < nodeB.Transaction.ID {
		return conflict.TransactionA, nil
	}
	return conflict.TransactionB, nil
}

// GetConflicts returns all detected conflicts
func (cr *ConflictResolver) GetConflicts() map[string]*ConflictInfo {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	conflicts := make(map[string]*ConflictInfo)
	for id, conflict := range cr.conflicts {
		conflicts[id] = conflict
	}
	return conflicts
}

// GetUnresolvedConflicts returns conflicts that haven't been resolved yet
func (cr *ConflictResolver) GetUnresolvedConflicts() []*ConflictInfo {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	var unresolved []*ConflictInfo
	for _, conflict := range cr.conflicts {
		if conflict.ResolvedAt == nil {
			unresolved = append(unresolved, conflict)
		}
	}
	return unresolved
}

// CleanupResolvedConflicts removes old resolved conflicts
func (cr *ConflictResolver) CleanupResolvedConflicts(olderThan time.Duration) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	for id, conflict := range cr.conflicts {
		if conflict.ResolvedAt != nil && conflict.ResolvedAt.Before(cutoff) {
			delete(cr.conflicts, id)
		}
	}
}

// GetConflictsCount returns the total number of conflicts detected
func (cr *ConflictResolver) GetConflictsCount() int {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return len(cr.conflicts)
}
