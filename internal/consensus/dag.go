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
	return &ConsensusService{
		DAG: types.NewDAG(),
		TransactionPool: &TransactionPool{
			Transactions: make(map[string]*types.Transaction),
		},
		ReputationManager: &ReputationManager{
			Scores: make(map[string]float64),
		},
		ValidationRules:       make([]ValidationRule, 0),
		FinalizationThreshold: 0.67, // 2/3 majority
	}
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
				cs.lock.Lock()
				node := cs.DAG.AddNode(tx)
				cs.lock.Unlock()

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
