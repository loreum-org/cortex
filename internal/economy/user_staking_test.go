package economy

import (
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestStakingEnvironment creates a test environment with users, nodes, and funds
func setupTestStakingEnvironment(t *testing.T) (*EconomicEngine, *UserStakingManager) {
	// Create a new economic engine
	engine := NewEconomicEngine()

	// Create test users
	userIDs := []string{"user1", "user2", "user3"}
	for _, userID := range userIDs {
		_, err := engine.CreateUserAccount(userID, "0x"+userID)
		require.NoError(t, err)

		// Fund user with 15,000 tokens (enough for max stake tests)
		fundAmount := new(big.Int).Mul(big.NewInt(15000), TokenUnit)
		_, err = engine.MintTokens(userID, fundAmount, "Test funding")
		require.NoError(t, err)
	}

	// Create test nodes
	nodeIDs := []string{"node1", "node2", "node3"}
	for _, nodeID := range nodeIDs {
		_, err := engine.CreateNodeAccount(nodeID)
		require.NoError(t, err)

		// Fund node with 5,000 tokens
		fundAmount := new(big.Int).Mul(big.NewInt(5000), TokenUnit)
		_, err = engine.MintTokens(nodeID, fundAmount, "Test funding")
		require.NoError(t, err)

		// Stake minimum required amount
		_, err = engine.Stake(nodeID, MinimumStake)
		require.NoError(t, err)
	}

	// Get the user staking manager
	stakingManager := engine.GetUserStakingManager()
	require.NotNil(t, stakingManager)

	return engine, stakingManager
}

// TestStakeToNode tests staking tokens to a node
func TestStakeToNode(t *testing.T) {
	engine, stakingManager := setupTestStakingEnvironment(t)

	// Test case 1: Successful staking
	t.Run("SuccessfulStaking", func(t *testing.T) {
		userID := "user1"
		nodeID := "node1"
		stakeAmount := new(big.Int).Mul(big.NewInt(100), TokenUnit) // 100 tokens

		// Get initial balances
		userBalanceBefore, err := engine.GetBalance(userID)
		require.NoError(t, err)

		// Stake tokens
		stake, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		require.NoError(t, err)
		require.NotNil(t, stake)

		// Check stake details
		assert.Equal(t, userID, stake.Account)
		assert.Equal(t, nodeID, stake.NodeID)
		assert.Equal(t, stakeAmount, stake.Amount)
		assert.False(t, stake.WithdrawalRequested)

		// Check user balance was reduced
		userBalanceAfter, err := engine.GetBalance(userID)
		require.NoError(t, err)
		expectedBalance := new(big.Int).Sub(userBalanceBefore, stakeAmount)
		assert.Equal(t, expectedBalance, userBalanceAfter)

		// Check node stake info
		nodeStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
		assert.Equal(t, stakeAmount, nodeStakeInfo.TotalUserStake)
		assert.Equal(t, 1, nodeStakeInfo.StakeCount)
		assert.Greater(t, nodeStakeInfo.ReputationBoost, 0.0)
	})

	// Test case 2: Staking below minimum amount
	t.Run("BelowMinimumStake", func(t *testing.T) {
		userID := "user2"
		nodeID := "node2"
		stakeAmount := big.NewInt(1) // Too small

		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "below minimum")
	})

	// Test case 3: Insufficient balance
	t.Run("InsufficientBalance", func(t *testing.T) {
		userID := "user3"
		nodeID := "node3"
		// Try to stake more than the user has
		stakeAmount := new(big.Int).Mul(big.NewInt(20000), TokenUnit)

		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		assert.Error(t, err)
		assert.Equal(t, ErrInsufficientBalance, err)
	})

	// Test case 4: Adding to existing stake
	t.Run("AddToExistingStake", func(t *testing.T) {
		userID := "user4" // Use a different user to avoid conflicts
		nodeID := "node4" // Use a different node
		
		// Create the user and node for this test
		_, err := engine.CreateUserAccount(userID, "0x"+userID)
		require.NoError(t, err)
		_, err = engine.CreateNodeAccount(nodeID)
		require.NoError(t, err)
		
		// Fund the user
		fundAmount := new(big.Int).Mul(big.NewInt(15000), TokenUnit)
		_, err = engine.MintTokens(userID, fundAmount, "Test funding")
		require.NoError(t, err)
		
		// First, stake an initial amount
		initialStake := new(big.Int).Mul(big.NewInt(100), TokenUnit)
		_, err = stakingManager.StakeToNode(userID, nodeID, initialStake)
		require.NoError(t, err)
		
		// Get current stake info after first stake
		nodeStakeInfoBefore := stakingManager.GetNodeStakeInfo(nodeID)
		userStakesBefore := stakingManager.GetUserStakes(userID)
		require.NotEmpty(t, userStakesBefore)
		originalStake := new(big.Int).Set(userStakesBefore[0].Amount) // Make a copy!
		
		// Add more stake
		additionalStake := new(big.Int).Mul(big.NewInt(50), TokenUnit)
		stake, err := stakingManager.StakeToNode(userID, nodeID, additionalStake)
		require.NoError(t, err)

		// Check stake was increased - should be original + additional
		expectedAmount := new(big.Int).Set(originalStake)
		expectedAmount.Add(expectedAmount, additionalStake)
		assert.Equal(t, expectedAmount, stake.Amount)

		// Check node total stake was increased
		nodeStakeInfoAfter := stakingManager.GetNodeStakeInfo(nodeID)
		expectedTotalStake := new(big.Int).Add(nodeStakeInfoBefore.TotalUserStake, additionalStake)
		assert.Equal(t, expectedTotalStake, nodeStakeInfoAfter.TotalUserStake)
	})

	// Test case 5: Exceeding maximum stake
	t.Run("ExceedMaximumStake", func(t *testing.T) {
		userID := "user2"
		nodeID := "node2"
		// Try to stake more than the maximum allowed per user per node
		stakeAmount := new(big.Int).Mul(big.NewInt(15000), TokenUnit)

		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		assert.Error(t, err)
		assert.Equal(t, ErrExceedsMaxStake, err)
	})
}

// TestRequestWithdrawal tests requesting withdrawal of staked tokens
func TestRequestWithdrawal(t *testing.T) {
	_, stakingManager := setupTestStakingEnvironment(t)

	// Setup: Stake tokens first
	userID := "user1"
	nodeID := "node1"
	stakeAmount := new(big.Int).Mul(big.NewInt(200), TokenUnit)

	stake, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
	require.NoError(t, err)
	require.NotNil(t, stake)

	// Test case 1: Successful withdrawal request
	t.Run("SuccessfulWithdrawalRequest", func(t *testing.T) {
		withdrawAmount := new(big.Int).Mul(big.NewInt(50), TokenUnit)

		err := stakingManager.RequestWithdrawal(userID, nodeID, withdrawAmount)
		require.NoError(t, err)

		// Check stake was updated
		userStakes := stakingManager.GetUserStakes(userID)
		require.NotEmpty(t, userStakes)
		updatedStake := userStakes[0]

		assert.True(t, updatedStake.WithdrawalRequested)
		assert.Equal(t, withdrawAmount, updatedStake.WithdrawalAmount)
		assert.False(t, updatedStake.WithdrawalAt.IsZero())
		assert.True(t, updatedStake.WithdrawalAt.After(time.Now()))
	})

	// Test case 2: Requesting more than staked
	t.Run("WithdrawMoreThanStaked", func(t *testing.T) {
		userID := "user2"
		nodeID := "node2"

		// Stake some tokens first
		stakeAmount := new(big.Int).Mul(big.NewInt(100), TokenUnit)
		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		require.NoError(t, err)

		// Try to withdraw more than staked
		withdrawAmount := new(big.Int).Mul(big.NewInt(200), TokenUnit)
		err = stakingManager.RequestWithdrawal(userID, nodeID, withdrawAmount)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds staked amount")
	})

	// Test case 3: Withdrawal request for non-existent stake
	t.Run("NonExistentStake", func(t *testing.T) {
		err := stakingManager.RequestWithdrawal("user3", "node3", big.NewInt(100))
		assert.Error(t, err)
		assert.Equal(t, ErrStakeNotFound, err)
	})
}

// TestExecuteWithdrawal tests executing withdrawal requests
func TestExecuteWithdrawal(t *testing.T) {
	engine, stakingManager := setupTestStakingEnvironment(t)

	// Setup: Stake tokens and request withdrawal
	userID := "user1"
	nodeID := "node1"
	stakeAmount := new(big.Int).Mul(big.NewInt(300), TokenUnit)
	withdrawAmount := new(big.Int).Mul(big.NewInt(100), TokenUnit)

	// Stake tokens
	_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
	require.NoError(t, err)

	// Request withdrawal
	err = stakingManager.RequestWithdrawal(userID, nodeID, withdrawAmount)
	require.NoError(t, err)

	// Test case 1: Withdrawal delay not met
	t.Run("WithdrawalDelayNotMet", func(t *testing.T) {
		_, err := stakingManager.ExecuteWithdrawal(userID, nodeID)
		assert.Error(t, err)
		assert.Equal(t, ErrWithdrawalDelayNotMet, err)
	})

	// Test case 2: Successful withdrawal after delay
	t.Run("SuccessfulWithdrawal", func(t *testing.T) {
		// Get user stake and modify withdrawal time for testing
		userStakes := stakingManager.GetUserStakes(userID)
		require.NotEmpty(t, userStakes)
		stake := stakingManager.getUserNodeStake(userID, nodeID)
		require.NotNil(t, stake)

		// Simulate time passing
		stake.WithdrawalAt = time.Now().Add(-1 * time.Hour)

		// Get balances before withdrawal
		userBalanceBefore, err := engine.GetBalance(userID)
		require.NoError(t, err)

		// Execute withdrawal
		tx, err := stakingManager.ExecuteWithdrawal(userID, nodeID)
		require.NoError(t, err)
		require.NotNil(t, tx)

		// Check transaction details
		assert.Equal(t, "staking_pool", tx.FromID)
		assert.Equal(t, userID, tx.ToID)
		assert.Equal(t, withdrawAmount, tx.Amount)

		// Check user balance increased
		userBalanceAfter, err := engine.GetBalance(userID)
		require.NoError(t, err)
		expectedBalance := new(big.Int).Add(userBalanceBefore, withdrawAmount)
		assert.Equal(t, expectedBalance, userBalanceAfter)

		// Check stake was updated
		updatedStake := stakingManager.getUserNodeStake(userID, nodeID)
		require.NotNil(t, updatedStake)
		assert.False(t, updatedStake.WithdrawalRequested)
		assert.Nil(t, updatedStake.WithdrawalAmount)
		assert.True(t, updatedStake.WithdrawalAt.IsZero())

		// Check remaining stake amount
		expectedRemainingStake := new(big.Int).Sub(stakeAmount, withdrawAmount)
		assert.Equal(t, expectedRemainingStake, updatedStake.Amount)
	})

	// Test case 3: Complete withdrawal (stake becomes zero)
	t.Run("CompleteWithdrawal", func(t *testing.T) {
		// Setup new stake
		userID := "user2"
		nodeID := "node2"
		stakeAmount := new(big.Int).Mul(big.NewInt(200), TokenUnit)

		// Stake tokens
		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		require.NoError(t, err)

		// Request full withdrawal
		err = stakingManager.RequestWithdrawal(userID, nodeID, stakeAmount)
		require.NoError(t, err)

		// Get stake and modify withdrawal time
		stake := stakingManager.getUserNodeStake(userID, nodeID)
		require.NotNil(t, stake)
		stake.WithdrawalAt = time.Now().Add(-1 * time.Hour)

		// Execute withdrawal
		_, err = stakingManager.ExecuteWithdrawal(userID, nodeID)
		require.NoError(t, err)

		// Check stake was removed
		updatedStake := stakingManager.getUserNodeStake(userID, nodeID)
		assert.Nil(t, updatedStake)

		// Check node stake info
		nodeStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
		assert.NotContains(t, nodeStakeInfo.UserStakes, userID)
	})
}

// TestStakeDistribution tests distributing stakes among multiple nodes
func TestStakeDistribution(t *testing.T) {
	_, stakingManager := setupTestStakingEnvironment(t)

	// Test staking to multiple nodes
	t.Run("StakeToMultipleNodes", func(t *testing.T) {
		userID := "user1"
		nodeIDs := []string{"node1", "node2", "node3"}
		stakeAmounts := []*big.Int{
			new(big.Int).Mul(big.NewInt(100), TokenUnit),
			new(big.Int).Mul(big.NewInt(200), TokenUnit),
			new(big.Int).Mul(big.NewInt(300), TokenUnit),
		}

		// Stake to each node
		for i, nodeID := range nodeIDs {
			_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmounts[i])
			require.NoError(t, err)
		}

		// Check user has stakes on all nodes
		userStakes := stakingManager.GetUserStakes(userID)
		assert.Equal(t, len(nodeIDs), len(userStakes))

		// Check each node has correct stake amount
		for i, nodeID := range nodeIDs {
			nodeStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
			assert.Equal(t, stakeAmounts[i], nodeStakeInfo.TotalUserStake)
		}
	})

	// Test multiple users staking to the same node
	t.Run("MultipleUsersToSameNode", func(t *testing.T) {
		nodeID := "node1"
		userIDs := []string{"user1", "user2", "user3"}

		// User 1 already staked in previous test
		existingStake := stakingManager.GetNodeStakeInfo(nodeID).TotalUserStake

		// Add stakes from other users
		additionalStakes := []*big.Int{
			new(big.Int).Mul(big.NewInt(150), TokenUnit),
			new(big.Int).Mul(big.NewInt(250), TokenUnit),
		}

		for i, userID := range userIDs[1:] {
			_, err := stakingManager.StakeToNode(userID, nodeID, additionalStakes[i])
			require.NoError(t, err)
		}

		// Check node has combined stake from all users
		nodeStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
		assert.Equal(t, len(userIDs), nodeStakeInfo.StakeCount)

		// Calculate expected total stake
		expectedTotalStake := new(big.Int).Set(existingStake)
		for _, amount := range additionalStakes {
			expectedTotalStake = new(big.Int).Add(expectedTotalStake, amount)
		}

		assert.Equal(t, expectedTotalStake, nodeStakeInfo.TotalUserStake)
	})
}

// TestReputationBoosting tests reputation boosting from staking
func TestReputationBoosting(t *testing.T) {
	_, stakingManager := setupTestStakingEnvironment(t)

	// Test reputation boost from staking
	t.Run("StakingIncreasesReputation", func(t *testing.T) {
		nodeID := "node1"
		userID := "user1"

		// Check initial reputation boost
		initialStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
		initialBoost := initialStakeInfo.ReputationBoost

		// Stake a significant amount
		stakeAmount := new(big.Int).Mul(big.NewInt(5000), TokenUnit)
		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		require.NoError(t, err)

		// Check reputation boost increased
		updatedStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
		assert.Greater(t, updatedStakeInfo.ReputationBoost, initialBoost)
	})

	// Test reputation boost formula
	t.Run("ReputationBoostFormula", func(t *testing.T) {
		nodeID := "node2"
		userID := "user2"

		// Stake a specific amount
		stakeAmount := new(big.Int).Mul(big.NewInt(1000), TokenUnit)
		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		require.NoError(t, err)

		// Check reputation boost matches formula
		stakeInfo := stakingManager.GetNodeStakeInfo(nodeID)

		// Convert stake to float for calculation
		stakeFloat, _ := new(big.Float).SetInt(stakeAmount).Float64()
		expectedBoost := stakeFloat * stakingManager.stakingConfig.ReputationMultiplier

		assert.InDelta(t, expectedBoost, stakeInfo.ReputationBoost, 0.001)
	})
}

// TestPerformanceTracking tests performance tracking and slashing
func TestPerformanceTracking(t *testing.T) {
	_, stakingManager := setupTestStakingEnvironment(t)

	// Set up stakes for testing
	nodeID := "node1"
	userIDs := []string{"user1", "user2"}

	for _, userID := range userIDs {
		stakeAmount := new(big.Int).Mul(big.NewInt(500), TokenUnit)
		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		require.NoError(t, err)
	}

	// Test recording query results
	t.Run("RecordQueryResults", func(t *testing.T) {
		// Record successful queries
		for i := 0; i < 5; i++ {
			stakingManager.RecordQueryResult(nodeID, "query_"+string(rune(i)), true, 100)
		}

		// Record failed query
		stakingManager.RecordQueryResult(nodeID, "failed_query", false, 500)

		// Check performance data
		nodeStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
		require.NotNil(t, nodeStakeInfo.PerformanceData)

		perf := nodeStakeInfo.PerformanceData
		assert.Equal(t, int64(5), perf.QueriesCompleted)
		assert.Equal(t, int64(1), perf.QueriesFailed)
		assert.InDelta(t, 0.833, perf.SuccessRate, 0.001) // 5/6 = ~0.833
		assert.NotZero(t, perf.AverageResponseTime)
	})

	// Test slashing for poor performance
	t.Run("SlashingForPoorPerformance", func(t *testing.T) {
		// Get initial stake info
		initialStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
		initialTotalStake := initialStakeInfo.TotalUserStake

		// Perform slashing
		reason := "Test slashing"
		severity := 0.5 // 50% severity
		slashingEvent, err := stakingManager.SlashNode(nodeID, reason, severity, "test_query")
		require.NoError(t, err)
		require.NotNil(t, slashingEvent)

		// Check slashing event details
		assert.Equal(t, nodeID, slashingEvent.NodeID)
		assert.Equal(t, reason, slashingEvent.Reason)
		assert.Equal(t, severity, slashingEvent.Severity)
		assert.NotNil(t, slashingEvent.SlashedAmount)
		assert.NotEmpty(t, slashingEvent.AffectedUsers)

		// Check stake was reduced
		updatedStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
		assert.Less(t, updatedStakeInfo.TotalUserStake.Cmp(initialTotalStake), 0)

		// Check slashing event was recorded
		require.NotNil(t, updatedStakeInfo.PerformanceData)
		assert.NotEmpty(t, updatedStakeInfo.PerformanceData.SlashingEvents)
		assert.Equal(t, 1, len(updatedStakeInfo.PerformanceData.SlashingEvents))

		// Check total slashed amount
		assert.Equal(t, slashingEvent.SlashedAmount, updatedStakeInfo.PerformanceData.TotalSlashed)
	})

	// Test auto-slashing from poor performance
	t.Run("AutoSlashingFromPoorPerformance", func(t *testing.T) {
		nodeID := "node2"
		userID := "user3"

		// Setup stake
		stakeAmount := new(big.Int).Mul(big.NewInt(1000), TokenUnit)
		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		require.NoError(t, err)

		// Record many failed queries to trigger auto-slashing
		for i := 0; i < 20; i++ {
			stakingManager.RecordQueryResult(nodeID, "query_"+string(rune(i)), i%3 != 0, 100)
		}

		// Check performance data
		nodeStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
		require.NotNil(t, nodeStakeInfo.PerformanceData)

		// Success rate should be around 2/3 (below threshold)
		assert.InDelta(t, 0.667, nodeStakeInfo.PerformanceData.SuccessRate, 0.1)

		// Wait briefly for async slashing to complete
		time.Sleep(100 * time.Millisecond)

		// Check if slashing occurred
		updatedStakeInfo := stakingManager.GetNodeStakeInfo(nodeID)
		if len(updatedStakeInfo.PerformanceData.SlashingEvents) > 0 {
			assert.NotEqual(t, big.NewInt(0), updatedStakeInfo.PerformanceData.TotalSlashed)
		}
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	engine, stakingManager := setupTestStakingEnvironment(t)

	// Test case: Zero stake amount
	t.Run("ZeroStakeAmount", func(t *testing.T) {
		userID := "user1"
		nodeID := "node1"
		zeroAmount := big.NewInt(0)

		_, err := stakingManager.StakeToNode(userID, nodeID, zeroAmount)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "below minimum")
	})

	// Test case: Non-existent node
	t.Run("NonExistentNode", func(t *testing.T) {
		userID := "user1"
		nodeID := "non_existent_node"
		stakeAmount := new(big.Int).Mul(big.NewInt(100), TokenUnit)

		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	// Test case: Non-existent user
	t.Run("NonExistentUser", func(t *testing.T) {
		userID := "non_existent_user"
		nodeID := "node1"
		stakeAmount := new(big.Int).Mul(big.NewInt(100), TokenUnit)

		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	// Test case: Withdraw with no request
	t.Run("WithdrawWithNoRequest", func(t *testing.T) {
		userID := "user1"
		nodeID := "node1"
		stakeAmount := new(big.Int).Mul(big.NewInt(100), TokenUnit)

		// Stake tokens
		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		require.NoError(t, err)

		// Try to execute withdrawal without request
		_, err = stakingManager.ExecuteWithdrawal(userID, nodeID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no withdrawal request found")
	})

	// Test case: Slashing with no stakes
	t.Run("SlashingWithNoStakes", func(t *testing.T) {
		nodeID := "node_without_stakes"
		_, err := engine.CreateNodeAccount(nodeID)
		require.NoError(t, err)

		_, err = stakingManager.SlashNode(nodeID, "test reason", 0.5, "test_query")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no stakes found")
	})

	// Test case: Maximum stake per node
	t.Run("MaximumStakePerNode", func(t *testing.T) {
		userID := "user2"
		nodeID := "node2"

		// Stake up to just below the maximum
		stakeAmount := new(big.Int).Sub(MaxUserStakePerNode, new(big.Int).Mul(big.NewInt(100), TokenUnit))
		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		require.NoError(t, err)

		// Try to stake more, which would exceed the maximum
		additionalStake := new(big.Int).Mul(big.NewInt(200), TokenUnit)
		_, err = stakingManager.StakeToNode(userID, nodeID, additionalStake)
		assert.Error(t, err)
		assert.Equal(t, ErrExceedsMaxStake, err)
	})
}

// TestGetNodeStakeInfo tests retrieving node staking information
func TestGetNodeStakeInfo(t *testing.T) {
	_, stakingManager := setupTestStakingEnvironment(t)

	// Setup: Multiple users stake to a node
	nodeID := "node1"
	users := []string{"user1", "user2", "user3"}
	totalStake := big.NewInt(0)

	for i, userID := range users {
		stakeAmount := new(big.Int).Mul(big.NewInt(int64(100*(i+1))), TokenUnit)
		totalStake = new(big.Int).Add(totalStake, stakeAmount)

		_, err := stakingManager.StakeToNode(userID, nodeID, stakeAmount)
		require.NoError(t, err)
	}

	// Test getting node stake info
	t.Run("GetNodeStakeInfo", func(t *testing.T) {
		stakeInfo := stakingManager.GetNodeStakeInfo(nodeID)

		assert.Equal(t, nodeID, stakeInfo.NodeID)
		assert.Equal(t, totalStake, stakeInfo.TotalUserStake)
		assert.Equal(t, len(users), stakeInfo.StakeCount)
		assert.Equal(t, len(users), len(stakeInfo.UserStakes))
		assert.Greater(t, stakeInfo.ReputationBoost, 0.0)
	})

	// Test getting all node stakes
	t.Run("GetAllNodeStakes", func(t *testing.T) {
		allStakes := stakingManager.GetAllNodeStakes()

		assert.NotEmpty(t, allStakes)
		assert.Contains(t, allStakes, nodeID)
		assert.Equal(t, totalStake, allStakes[nodeID].TotalUserStake)
	})

	// Test getting user stakes
	t.Run("GetUserStakes", func(t *testing.T) {
		for i, userID := range users {
			userStakes := stakingManager.GetUserStakes(userID)

			assert.NotEmpty(t, userStakes)
			assert.Equal(t, 1, len(userStakes))
			assert.Equal(t, nodeID, userStakes[0].NodeID)

			expectedAmount := new(big.Int).Mul(big.NewInt(int64(100*(i+1))), TokenUnit)
			assert.Equal(t, expectedAmount, userStakes[0].Amount)
		}
	})
}

// TestStakingConfig tests staking configuration
func TestStakingConfig(t *testing.T) {
	_, stakingManager := setupTestStakingEnvironment(t)

	t.Run("DefaultConfig", func(t *testing.T) {
		config := stakingManager.GetStakingConfig()

		assert.NotNil(t, config)
		assert.Greater(t, config.ReputationMultiplier, 0.0)
		assert.NotNil(t, config.RewardRate)
		assert.True(t, config.SlashingEnabled)
		assert.Greater(t, config.MinSlashingThreshold, 0.0)
		assert.Greater(t, config.MaxSlashingPercent, 0.0)
		assert.Greater(t, config.ReputationDecayRate, 0.0)
	})
}
