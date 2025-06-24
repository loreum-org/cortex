package consensus_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/loreum-org/cortex/internal/consensus"
	"github.com/loreum-org/cortex/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testPubKey  ed25519.PublicKey
	testPrivKey ed25519.PrivateKey
)

func TestMain(m *testing.M) {
	var err error
	testPubKey, testPrivKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic("Failed to generate test keys: " + err.Error())
	}
	m.Run()
}

// createTestConsensusTransaction is a helper to create transactions for consensus tests.
func createTestConsensusTransaction(t *testing.T, data string, parents []string) *types.Transaction {
	tx, err := consensus.CreateTransaction(data, parents, testPrivKey)
	require.NoError(t, err)
	return tx
}

func TestNewConsensusService(t *testing.T) {
	cs := consensus.NewConsensusService()
	require.NotNil(t, cs, "NewConsensusService should return a non-nil instance")

	assert.NotNil(t, cs.DAG, "DAG should be initialized")
	assert.Empty(t, cs.DAG.Nodes, "DAG should be empty initially")

	assert.NotNil(t, cs.TransactionPool, "TransactionPool should be initialized")
	assert.Empty(t, cs.TransactionPool.Transactions, "TransactionPool should be empty initially")

	assert.NotNil(t, cs.ReputationManager, "ReputationManager should be initialized")
	assert.Empty(t, cs.ReputationManager.Scores, "ReputationManager scores should be empty initially")

	assert.NotNil(t, cs.ValidationRules, "ValidationRules slice should be initialized (not nil)")
	assert.Empty(t, cs.ValidationRules, "ValidationRules should be empty initially")

	assert.Equal(t, 0.67, cs.FinalizationThreshold, "Default FinalizationThreshold is incorrect")
}

func TestConsensusService_AddTransaction(t *testing.T) {
	cs := consensus.NewConsensusService()
	tx1 := createTestConsensusTransaction(t, "tx1 data", nil)

	t.Run("AddValidTransaction", func(t *testing.T) {
		err := cs.AddTransaction(tx1)
		require.NoError(t, err)

		poolTx, exists := cs.TransactionPool.GetTransaction(tx1.ID)
		assert.True(t, exists, "Transaction should be in the pool")
		assert.Equal(t, tx1, poolTx, "Stored transaction mismatch")
	})

	t.Run("AddTransactionFailingValidation", func(t *testing.T) {
		csWithRule := consensus.NewConsensusService()
		// Add a validation rule that always fails for a specific transaction data
		failingRule := func(tx *types.Transaction) bool {
			return tx.Data != "fail me"
		}
		csWithRule.ValidationRules = append(csWithRule.ValidationRules, failingRule)

		txFail := createTestConsensusTransaction(t, "fail me", nil)
		err := csWithRule.AddTransaction(txFail)
		assert.Error(t, err, "Should return an error for a transaction failing validation")
		assert.Contains(t, err.Error(), "transaction validation failed")

		_, exists := csWithRule.TransactionPool.GetTransaction(txFail.ID)
		assert.False(t, exists, "Transaction failing validation should not be in the pool")
	})

	t.Run("ConcurrentAddTransaction", func(t *testing.T) {
		csConcurrent := consensus.NewConsensusService()
		numTransactions := 100
		var wg sync.WaitGroup
		wg.Add(numTransactions)

		for i := 0; i < numTransactions; i++ {
			go func(i int) {
				defer wg.Done()
				tx := createTestConsensusTransaction(t, fmt.Sprintf("concurrent_tx_%d", i), nil)
				err := csConcurrent.AddTransaction(tx)
				assert.NoError(t, err, "Error adding transaction concurrently")
			}(i)
		}
		wg.Wait()

		csConcurrent.TransactionPool.Lock() // Manual lock for reading size
		poolSize := len(csConcurrent.TransactionPool.Transactions)
		csConcurrent.TransactionPool.Unlock()
		assert.Equal(t, numTransactions, poolSize, "All concurrently added transactions should be in the pool")
	})
}

func TestConsensusService_ProcessTransactions(t *testing.T) {
	t.Run("ProcessAndFinalizeTransactions", func(t *testing.T) {
		cs := consensus.NewConsensusService()
		tx1 := createTestConsensusTransaction(t, "proc_tx1", nil)
		tx2 := createTestConsensusTransaction(t, "proc_tx2", []string{tx1.ID})

		err := cs.AddTransaction(tx1)
		require.NoError(t, err)
		err = cs.AddTransaction(tx2)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		go cs.ProcessTransactions(ctx)

		// Allow some time for processing
		time.Sleep(1500 * time.Millisecond) // Needs to be > 1s processing loop + some buffer

		cancel() // Stop processing

		// Check DAG
		dagNode1, exists1 := cs.GetTransaction(tx1.ID)
		assert.True(t, exists1, "tx1 should be in the DAG")
		require.NotNil(t, dagNode1)

		dagNode2, exists2 := cs.GetTransaction(tx2.ID)
		assert.True(t, exists2, "tx2 should be in the DAG")
		require.NotNil(t, dagNode2)

		// Check scores (simple check, current logic sets score to 1.0)
		cs.DAG.Lock() // Need to lock DAG to read Node scores if ProcessTransactions might still be running
		node1FromDAG, _ := cs.DAG.GetNode(tx1.ID)
		node2FromDAG, _ := cs.DAG.GetNode(tx2.ID)
		cs.DAG.Unlock()

		assert.Equal(t, 1.0, node1FromDAG.Score, "tx1 score incorrect")
		assert.Equal(t, 1.0, node2FromDAG.Score, "tx2 score incorrect")

		// Check finalization (score 1.0 >= threshold 0.67)
		assert.True(t, cs.IsFinalized(tx1.ID), "tx1 should be finalized")
		assert.True(t, cs.IsFinalized(tx2.ID), "tx2 should be finalized")

		// Check pool is empty
		cs.TransactionPool.Lock()
		poolSize := len(cs.TransactionPool.Transactions)
		cs.TransactionPool.Unlock()
		assert.Equal(t, 0, poolSize, "Transaction pool should be empty after processing")
	})

	t.Run("ProcessWithEmptyPool", func(t *testing.T) {
		cs := consensus.NewConsensusService()
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond) // Short run
		defer cancel()

		go cs.ProcessTransactions(ctx)
		<-ctx.Done() // Wait for context to be done

		cs.DAG.Lock()
		dagSize := len(cs.DAG.Nodes)
		cs.DAG.Unlock()
		assert.Equal(t, 0, dagSize, "DAG should be empty if pool was empty")
	})

	t.Run("ContextCancellationStopsProcessing", func(t *testing.T) {
		cs := consensus.NewConsensusService()
		tx1 := createTestConsensusTransaction(t, "cancel_tx", nil)
		cs.AddTransaction(tx1)

		ctx, cancel := context.WithCancel(context.Background())

		processingDone := make(chan struct{})
		go func() {
			cs.ProcessTransactions(ctx)
			close(processingDone)
		}()

		cancel() // Cancel immediately

		select {
		case <-processingDone:
			// Test passed, ProcessTransactions exited
		case <-time.After(2 * time.Second): // ProcessTransactions loop has 1s sleep
			t.Fatal("ProcessTransactions did not stop after context cancellation")
		}
	})
}

func TestConsensusService_GetTransactionAndIsFinalized(t *testing.T) {
	cs := consensus.NewConsensusService()
	tx1 := createTestConsensusTransaction(t, "get_tx1", nil)
	cs.AddTransaction(tx1)

	ctx, cancel := context.WithCancel(context.Background())
	go cs.ProcessTransactions(ctx)
	time.Sleep(1500 * time.Millisecond) // Allow processing and finalization
	cancel()

	t.Run("GetExistingTransaction", func(t *testing.T) {
		retrievedTx, exists := cs.GetTransaction(tx1.ID)
		assert.True(t, exists, "Should find existing transaction in DAG")
		require.NotNil(t, retrievedTx)
		assert.Equal(t, tx1.ID, retrievedTx.ID)
		assert.Equal(t, tx1.Data, retrievedTx.Data)
	})

	t.Run("GetNonExistentTransaction", func(t *testing.T) {
		_, exists := cs.GetTransaction("non-existent-id")
		assert.False(t, exists, "Should not find non-existent transaction")
	})

	t.Run("IsFinalizedTrue", func(t *testing.T) {
		isFin := cs.IsFinalized(tx1.ID)
		assert.True(t, isFin, "Transaction tx1 should be finalized")
	})

	t.Run("IsFinalizedFalseForNonExistent", func(t *testing.T) {
		isFin := cs.IsFinalized("non-existent-id-for-finalize-check")
		assert.False(t, isFin, "Non-existent transaction should not be finalized")
	})

	// Test for a transaction that exists but is not finalized (harder with current simple finalization)
	// To test this, we'd need a way for a node to have a score < FinalizationThreshold.
	// Current logic: node.Score = 1.0. FinalizationThreshold = 0.67. So all processed nodes finalize.
	// If scoring or threshold changes, this test case would be more relevant.
	// For now, we can simulate by manually adding a node to DAG with low score and not finalized.
	t.Run("IsFinalizedFalseForExistingNotFinalized", func(t *testing.T) {
		manualCS := consensus.NewConsensusService()
		manualTx := createTestConsensusTransaction(t, "manual_unfinalized", nil)

		// Manually add to DAG without processing through ProcessTransactions
		manualCS.DAG.Lock()
		node := manualCS.DAG.AddNode(manualTx)
		node.Score = 0.5           // Score below threshold
		manualTx.Finalized = false // Explicitly ensure it's not marked finalized
		manualCS.DAG.Unlock()

		isFin := manualCS.IsFinalized(manualTx.ID)
		assert.False(t, isFin, "Manually added transaction with low score should not be finalized")
	})
}

func TestNewTransactionPool(t *testing.T) {
	tp := consensus.NewTransactionPool()
	require.NotNil(t, tp)
	assert.NotNil(t, tp.Transactions, "Transactions map should be initialized")
	assert.Empty(t, tp.Transactions, "New pool should be empty")
}

func TestTransactionPool_AddGetTransaction(t *testing.T) {
	tp := consensus.NewTransactionPool()
	tx1 := createTestConsensusTransaction(t, "pool_tx1", nil)

	t.Run("AddAndGet", func(t *testing.T) {
		tp.AddTransaction(tx1)
		retrievedTx, exists := tp.GetTransaction(tx1.ID)
		assert.True(t, exists, "Transaction should exist in pool after adding")
		assert.Equal(t, tx1, retrievedTx, "Retrieved transaction mismatch")
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, exists := tp.GetTransaction("non-existent-pool-tx")
		assert.False(t, exists, "Should not find non-existent transaction in pool")
	})
}

func TestTransactionPool_ConcurrentAccess(t *testing.T) {
	tp := consensus.NewTransactionPool()
	numTransactions := 200
	var wg sync.WaitGroup
	wg.Add(numTransactions * 2) // For add and get operations

	// Concurrent additions
	for i := 0; i < numTransactions; i++ {
		go func(i int) {
			defer wg.Done()
			tx := createTestConsensusTransaction(t, fmt.Sprintf("tp_concurrent_%d", i), nil)
			tp.AddTransaction(tx)
		}(i)
	}

	// Concurrent gets (some might be before add completes, some after)
	_ = make([]string, numTransactions)
	for i := 0; i < numTransactions; i++ {
		// Need to generate IDs in the same way AddTransaction does, or pre-create tx objects
		// For simplicity, let's assume we know the IDs or pre-create them.
		// Here, we'll just try to get, acknowledging some might not exist yet.
		// A better test would be to pre-generate transactions and their IDs.
		// Let's create transactions first, then concurrently add and get them.
	}
	// Re-think this part for better assertion.
	// Let's add all first, then concurrently get. Or mix.

	// Simpler: Add all, then concurrently get.
	preAddedTxs := make([]*types.Transaction, numTransactions)
	for i := 0; i < numTransactions; i++ {
		tx := createTestConsensusTransaction(t, fmt.Sprintf("tp_concurrent_known_%d", i), nil)
		preAddedTxs[i] = tx
		tp.AddTransaction(tx) // Add them sequentially first
	}

	// Now concurrently get
	for i := 0; i < numTransactions; i++ {
		go func(i int) {
			defer wg.Done()
			txID := preAddedTxs[i].ID
			retrievedTx, exists := tp.GetTransaction(txID)
			// Assertions inside goroutines need care, or collect results.
			// For this, we can assume GetTransaction is safe.
			if !exists {
				t.Errorf("Expected transaction %s to exist, but it didn't", txID)
			}
			if retrievedTx == nil || retrievedTx.ID != txID {
				t.Errorf("Retrieved transaction mismatch for ID %s", txID)
			}
		}(i)
	}

	wg.Wait()
	tp.Lock()
	assert.Equal(t, numTransactions, len(tp.Transactions), "Pool size mismatch after concurrent operations")
	tp.Unlock()
}

func TestNewReputationManager(t *testing.T) {
	rm := consensus.NewReputationManager()
	require.NotNil(t, rm)
	assert.NotNil(t, rm.Scores, "Scores map should be initialized")
	assert.Empty(t, rm.Scores, "New manager should have empty scores")
}

func TestReputationManager_SetGetScore(t *testing.T) {
	rm := consensus.NewReputationManager()
	nodeID1 := "node1"
	score1 := 0.75

	t.Run("SetAndGet", func(t *testing.T) {
		rm.SetScore(nodeID1, score1)
		retrievedScore := rm.GetScore(nodeID1)
		assert.Equal(t, score1, retrievedScore, "Retrieved score mismatch")
	})

	t.Run("GetNonExistentNode", func(t *testing.T) {
		retrievedScore := rm.GetScore("non-existent-node-for-rep")
		assert.Equal(t, 0.0, retrievedScore, "Score for non-existent node should be 0.0")
	})

	t.Run("UpdateScore", func(t *testing.T) {
		newScore := 0.90
		rm.SetScore(nodeID1, newScore)
		retrievedScore := rm.GetScore(nodeID1)
		assert.Equal(t, newScore, retrievedScore, "Updated score mismatch")
	})
}

func TestReputationManager_ConcurrentAccess(t *testing.T) {
	rm := consensus.NewReputationManager()
	numNodes := 100
	var wg sync.WaitGroup
	wg.Add(numNodes * 2) // For set and get operations

	nodeIDs := make([]string, numNodes)
	scores := make([]float64, numNodes)

	for i := 0; i < numNodes; i++ {
		nodeIDs[i] = fmt.Sprintf("rep_node_%d", i)
		scores[i] = float64(i) / float64(numNodes)
	}

	// Concurrent sets
	for i := 0; i < numNodes; i++ {
		go func(i int) {
			defer wg.Done()
			rm.SetScore(nodeIDs[i], scores[i])
		}(i)
	}

	// Concurrent gets
	for i := 0; i < numNodes; i++ {
		go func(i int) {
			defer wg.Done()
			// Wait a bit to increase chance of Set completing for this ID
			// This is not a perfect synchronization but helps for testing.
			// A more robust test might use channels or multiple phases.
			time.Sleep(time.Duration(i%10) * time.Millisecond)

			_ = rm.GetScore(nodeIDs[i])
			// Because SetScore might not have run yet for this specific i,
			// the score could be 0.0 or scores[i].
			// We are primarily testing for race conditions, not specific values here
			// unless all Sets are guaranteed to finish before Gets start.
			// Let's ensure all sets are done first for clearer assertion.
		}(i)
	}
	wg.Wait() // Wait for all goroutines (sets and initial gets)

	// Now, verify all scores after all sets are done
	for i := 0; i < numNodes; i++ {
		retrievedScore := rm.GetScore(nodeIDs[i])
		assert.Equal(t, scores[i], retrievedScore, "Score mismatch after concurrent sets for node %s", nodeIDs[i])
	}

	rm.Lock() // Manual lock for reading size
	assert.Equal(t, numNodes, len(rm.Scores), "Scores map size mismatch")
	rm.Unlock()
}
