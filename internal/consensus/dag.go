package consensus

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// ConsensusService manages the DAG and consensus mechanism
type ConsensusService struct {
	DAG                   *types.DAG
	TransactionPool       *TransactionPool
	ReputationManager     *ReputationManager
	ConflictResolver      *ConflictResolver
	CausalOrderingManager *CausalOrderingManager
	AGIRecorder          *AGIRecorder // AGI blockchain recorder
	ValidationRules       []ValidationRule
	FinalizationThreshold float64
	lock                  sync.RWMutex
}

// TransactionPool holds transactions that are waiting to be added to the DAG
type TransactionPool struct {
	Transactions map[string]*types.Transaction
	lock         sync.RWMutex
}

// ValidationRule is a function that validates a transaction
type ValidationRule func(*types.Transaction) bool

// ReputationManager manages reputation scores for nodes
type ReputationManager struct {
	Scores map[string]float64 // Map of node ID to reputation score
	lock   sync.RWMutex
}

// NewConsensusService creates a new consensus service
func NewConsensusService() *ConsensusService {
	dag := types.NewDAG()
	cs := &ConsensusService{
		DAG: dag,
		TransactionPool: &TransactionPool{
			Transactions: make(map[string]*types.Transaction),
		},
		ReputationManager: &ReputationManager{
			Scores: make(map[string]float64),
		},
		ValidationRules:       make([]ValidationRule, 0),
		FinalizationThreshold: 0.67, // 2/3 majority
	}

	// Initialize conflict resolver
	cs.ConflictResolver = NewConflictResolver(dag)

	// Initialize causal ordering manager
	cs.CausalOrderingManager = NewCausalOrderingManager(dag)

	return cs
}

// AddTransaction adds a transaction to the pool
func (cs *ConsensusService) AddTransaction(tx *types.Transaction) error {
	// Validate the transaction
	valid := true
	for _, rule := range cs.ValidationRules {
		if !rule(tx) {
			valid = false
			break
		}
	}

	if !valid {
		return fmt.Errorf("transaction validation failed")
	}

	// Add to the pool
	cs.TransactionPool.lock.Lock()
	cs.TransactionPool.Transactions[tx.ID] = tx
	cs.TransactionPool.lock.Unlock()

	return nil
}

// ProcessTransactions processes transactions from the pool and adds them to the DAG
func (cs *ConsensusService) ProcessTransactions(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get transactions from the pool
			cs.TransactionPool.lock.Lock()
			transactions := make([]*types.Transaction, 0, len(cs.TransactionPool.Transactions))
			for _, tx := range cs.TransactionPool.Transactions {
				transactions = append(transactions, tx)
			}
			cs.TransactionPool.Transactions = make(map[string]*types.Transaction)
			cs.TransactionPool.lock.Unlock()

			// Process each transaction
			for _, tx := range transactions {
				// Check for conflicts before adding to DAG
				conflicts := cs.ConflictResolver.DetectConflicts(tx)

				if len(conflicts) > 0 {
					// Attempt to resolve conflicts
					if err := cs.ConflictResolver.ResolveConflicts(conflicts); err != nil {
						log.Printf("Failed to resolve conflicts for transaction %s: %v", tx.ID, err)
						continue // Skip this transaction
					}

					// Check if this transaction was removed during conflict resolution
					if _, exists := cs.DAG.GetNode(tx.ID); !exists {
						log.Printf("Transaction %s was removed during conflict resolution", tx.ID)
						continue
					}
				}

				cs.lock.Lock()
				node := cs.DAG.AddNode(tx)
				cs.lock.Unlock()

				// Update causal ordering
				if err := cs.CausalOrderingManager.UpdateCausalOrdering(tx); err != nil {
					log.Printf("Failed to update causal ordering for transaction %s: %v", tx.ID, err)
				}

				// Calculate score for the node
				cs.calculateNodeScore(node)
			}

			// Check for finalization
			cs.finalizeTransactions()

			time.Sleep(1 * time.Second) // Process every second
		}
	}
}

// calculateNodeScore calculates the score for a node based on reputation
func (cs *ConsensusService) calculateNodeScore(node *types.DAGNode) {
	// For simplicity, we'll use a basic scoring mechanism
	// In a real implementation, this would be more complex
	score := 1.0

	// Add the node's score
	cs.lock.Lock()
	node.Score = score
	cs.lock.Unlock()
}

// finalizeTransactions finalizes transactions that meet the finalization threshold
func (cs *ConsensusService) finalizeTransactions() {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	// Calculate which transactions can be finalized
	for id, node := range cs.DAG.Nodes {
		if node.Transaction.Finalized {
			continue
		}

		// Check if the node has enough score to be finalized
		// In a real implementation, this would involve checking the consensus
		if node.Score >= cs.FinalizationThreshold {
			node.Transaction.Finalized = true
			log.Printf("Transaction %s finalized\n", id)
		}
	}
}

// GetTransaction gets a transaction from the DAG
func (cs *ConsensusService) GetTransaction(id string) (*types.Transaction, bool) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	node, exists := cs.DAG.GetNode(id)
	if !exists {
		return nil, false
	}

	return node.Transaction, true
}

// IsFinalized checks if a transaction is finalized
func (cs *ConsensusService) IsFinalized(id string) bool {
	cs.lock.RLock()
	defer cs.lock.RUnlock()

	return cs.DAG.IsFinalized(id)
}

// NewTransactionPool creates a new transaction pool
func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		Transactions: make(map[string]*types.Transaction),
	}
}

// AddTransaction adds a transaction to the pool
func (tp *TransactionPool) AddTransaction(tx *types.Transaction) {
	tp.lock.Lock()
	defer tp.lock.Unlock()

	tp.Transactions[tx.ID] = tx
}

// GetTransaction gets a transaction from the pool
func (tp *TransactionPool) GetTransaction(id string) (*types.Transaction, bool) {
	tp.lock.RLock()
	defer tp.lock.RUnlock()

	tx, exists := tp.Transactions[id]
	return tx, exists
}

// Lock locks the transaction pool for writing
func (tp *TransactionPool) Lock() {
	tp.lock.Lock()
}

// Unlock unlocks the transaction pool
func (tp *TransactionPool) Unlock() {
	tp.lock.Unlock()
}

// NewReputationManager creates a new reputation manager
func NewReputationManager() *ReputationManager {
	return &ReputationManager{
		Scores: make(map[string]float64),
	}
}

// SetScore sets the reputation score for a node
func (rm *ReputationManager) SetScore(nodeID string, score float64) {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	rm.Scores[nodeID] = score
}

// GetScore gets the reputation score for a node
func (rm *ReputationManager) GetScore(nodeID string) float64 {
	rm.lock.RLock()
	defer rm.lock.RUnlock()

	score, exists := rm.Scores[nodeID]
	if !exists {
		return 0.0
	}

	return score
}

// Lock locks the reputation manager for writing
func (rm *ReputationManager) Lock() {
	rm.lock.Lock()
}

// Unlock unlocks the reputation manager
func (rm *ReputationManager) Unlock() {
	rm.lock.Unlock()
}
// SubmitTransaction submits a transaction to the consensus service
func (cs *ConsensusService) SubmitTransaction(ctx context.Context, tx interface{}) error {
	// Convert tx to the expected type
	switch t := tx.(type) {
	case *types.Transaction:
		return cs.AddTransaction(t)
	default:
		return fmt.Errorf("unsupported transaction type: %T", tx)
	}
}

// IsTransactionFinalized checks if a transaction is finalized
func (cs *ConsensusService) IsTransactionFinalized(txID string) bool {
	cs.lock.RLock()
	defer cs.lock.RUnlock()
	
	node, exists := cs.DAG.GetNode(txID)
	if !exists {
		return false
	}
	return node.Transaction.Finalized
}