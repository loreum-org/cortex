package economy

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// MinimumUserStake is the minimum amount a user can stake to a node
	MinimumUserStake = new(big.Int).Mul(big.NewInt(10), TokenUnit) // 10 LORE minimum

	// UserStakeWithdrawalDelay is the time users must wait before withdrawing stake
	UserStakeWithdrawalDelay = 7 * 24 * time.Hour // 7 days

	// MaxUserStakePerNode is the maximum amount a single user can stake to one node
	MaxUserStakePerNode = new(big.Int).Mul(big.NewInt(10000), TokenUnit) // 10,000 LORE max

	// ErrStakeNotFound occurs when a stake record is not found
	ErrStakeNotFound = errors.New("stake not found")

	// ErrWithdrawalDelayNotMet occurs when trying to withdraw before delay period
	ErrWithdrawalDelayNotMet = errors.New("withdrawal delay not met")

	// ErrExceedsMaxStake occurs when staking would exceed maximum per user per node
	ErrExceedsMaxStake = errors.New("exceeds maximum stake per user per node")
)

// UserStake represents a user's stake on a specific node
type UserStake struct {
	ID         string    `json:"id"`
	Account    string    `json:"account"`
	NodeID     string    `json:"node_id"`
	Amount     *big.Int  `json:"amount"`
	StakedAt   time.Time `json:"staked_at"`
	LastUpdate time.Time `json:"last_update"`
	// Withdrawal fields
	WithdrawalRequested bool      `json:"withdrawal_requested"`
	WithdrawalAt        time.Time `json:"withdrawal_at,omitempty"`
	WithdrawalAmount    *big.Int  `json:"withdrawal_amount,omitempty"`
	// Rewards tracking
	AccumulatedRewards *big.Int  `json:"accumulated_rewards"`
	LastRewardTime     time.Time `json:"last_reward_time"`
}

// NodeStakeInfo aggregates staking information for a node
type NodeStakeInfo struct {
	NodeID          string           `json:"node_id"`
	TotalUserStake  *big.Int         `json:"total_user_stake"`
	UserStakes      []*UserStake     `json:"user_stakes"`
	StakeCount      int              `json:"stake_count"`
	ReputationBoost float64          `json:"reputation_boost"`
	PerformanceData *NodePerformance `json:"performance_data"`
}

// NodePerformance tracks node performance metrics for staking rewards/slashing
type NodePerformance struct {
	NodeID              string    `json:"node_id"`
	QueriesCompleted    int64     `json:"queries_completed"`
	QueriesFailed       int64     `json:"queries_failed"`
	SuccessRate         float64   `json:"success_rate"`
	AverageResponseTime int64     `json:"avg_response_time_ms"`
	LastActive          time.Time `json:"last_active"`
	// Slashing history
	SlashingEvents []SlashingEvent `json:"slashing_events"`
	TotalSlashed   *big.Int        `json:"total_slashed"`
}

// SlashingEvent represents a slashing event
type SlashingEvent struct {
	ID            string    `json:"id"`
	NodeID        string    `json:"node_id"`
	Reason        string    `json:"reason"`
	Severity      float64   `json:"severity"` // 0.0 to 1.0
	SlashedAmount *big.Int  `json:"slashed_amount"`
	AffectedUsers []string  `json:"affected_users"` // User IDs who lost stake
	Timestamp     time.Time `json:"timestamp"`
	QueryID       string    `json:"query_id,omitempty"`
}

// UserStakingManager manages user staking operations
type UserStakingManager struct {
	economicEngine  *EconomicEngine
	userStakes      map[string]*UserStake            // stake ID -> UserStake
	nodeStakes      map[string][]*UserStake          // node ID -> list of user stakes
	userNodeStakes  map[string]map[string]*UserStake // user ID -> node ID -> UserStake
	nodePerformance map[string]*NodePerformance      // node ID -> performance data
	stakingConfig   *StakingConfig
	mu              sync.RWMutex
}

// StakingConfig contains configuration for the staking system
type StakingConfig struct {
	ReputationMultiplier float64    `json:"reputation_multiplier"` // How much stake boosts reputation
	RewardRate           *big.Float `json:"reward_rate"`           // Daily reward rate for stakers
	SlashingEnabled      bool       `json:"slashing_enabled"`
	MinSlashingThreshold float64    `json:"min_slashing_threshold"` // Minimum failure rate to trigger slashing
	MaxSlashingPercent   float64    `json:"max_slashing_percent"`   // Maximum % of stake to slash
	ReputationDecayRate  float64    `json:"reputation_decay_rate"`  // Daily decay without performance
}

// NewUserStakingManager creates a new user staking manager
func NewUserStakingManager(economicEngine *EconomicEngine) *UserStakingManager {
	return &UserStakingManager{
		economicEngine:  economicEngine,
		userStakes:      make(map[string]*UserStake),
		nodeStakes:      make(map[string][]*UserStake),
		userNodeStakes:  make(map[string]map[string]*UserStake),
		nodePerformance: make(map[string]*NodePerformance),
		stakingConfig: &StakingConfig{
			ReputationMultiplier: 0.001,              // 0.1% reputation boost per 1000 LORE staked
			RewardRate:           big.NewFloat(0.05), // 5% annual rate
			SlashingEnabled:      true,
			MinSlashingThreshold: 0.1,  // 10% failure rate triggers slashing
			MaxSlashingPercent:   0.3,  // Max 30% slash
			ReputationDecayRate:  0.01, // 1% daily decay without performance
		},
	}
}

// StakeToNode allows a user to stake LORE tokens to a specific node
func (usm *UserStakingManager) StakeToNode(account, nodeID string, amount *big.Int) (*UserStake, error) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	// Validate amount
	if amount.Cmp(MinimumUserStake) < 0 {
		return nil, fmt.Errorf("stake amount %s is below minimum %s", amount.String(), MinimumUserStake.String())
	}

	// Check if user exists and has sufficient balance
	userAccount, err := usm.economicEngine.GetUserAccount(account)
	if err != nil {
		return nil, fmt.Errorf("user account not found: %w", err)
	}

	if userAccount.Balance.Cmp(amount) < 0 {
		return nil, ErrInsufficientBalance
	}

	// Check if node exists
	_, err = usm.economicEngine.GetNodeAccount(nodeID)
	if err != nil {
		return nil, fmt.Errorf("node account not found: %w", err)
	}

	// Check maximum stake per user per node
	if usm.userNodeStakes[account] == nil {
		usm.userNodeStakes[account] = make(map[string]*UserStake)
	}

	existingStake := usm.userNodeStakes[account][nodeID]
	totalStake := new(big.Int).Set(amount)
	if existingStake != nil {
		totalStake.Add(totalStake, existingStake.Amount)
	}

	if totalStake.Cmp(MaxUserStakePerNode) > 0 {
		return nil, ErrExceedsMaxStake
	}

	// Create transaction to move funds from user to staking pool
	tx, err := usm.economicEngine.Transfer(account, "staking_pool", amount, "User stake to node "+nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to transfer funds to staking pool: %w", err)
	}

	// Create or update user stake
	var userStake *UserStake
	if existingStake != nil {
		// Add to existing stake
		existingStake.Amount.Add(existingStake.Amount, amount)
		existingStake.LastUpdate = time.Now()
		userStake = existingStake
	} else {
		// Create new stake
		userStake = &UserStake{
			ID:                 uuid.New().String(),
			Account:            account,
			NodeID:             nodeID,
			Amount:             new(big.Int).Set(amount),
			StakedAt:           time.Now(),
			LastUpdate:         time.Now(),
			AccumulatedRewards: big.NewInt(0),
			LastRewardTime:     time.Now(),
		}

		// Store stake
		usm.userStakes[userStake.ID] = userStake
		usm.nodeStakes[nodeID] = append(usm.nodeStakes[nodeID], userStake)
		usm.userNodeStakes[account][nodeID] = userStake
	}

	log.Printf("User %s staked %s LORE to node %s (tx: %s)", account, amount.String(), nodeID, tx.ID)

	// Update node reputation based on new stake
	usm.updateNodeReputation(nodeID)

	return userStake, nil
}

// RequestWithdrawal initiates a withdrawal request for staked tokens
func (usm *UserStakingManager) RequestWithdrawal(account, nodeID string, amount *big.Int) error {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	userStake := usm.getUserNodeStake(account, nodeID)
	if userStake == nil {
		return ErrStakeNotFound
	}

	if amount.Cmp(userStake.Amount) > 0 {
		return fmt.Errorf("withdrawal amount exceeds staked amount")
	}

	// Set withdrawal request
	userStake.WithdrawalRequested = true
	userStake.WithdrawalAt = time.Now().Add(UserStakeWithdrawalDelay)
	userStake.WithdrawalAmount = new(big.Int).Set(amount)

	log.Printf("User %s requested withdrawal of %s LORE from node %s (available at: %s)",
		account, amount.String(), nodeID, userStake.WithdrawalAt.Format(time.RFC3339))

	return nil
}

// ExecuteWithdrawal completes a withdrawal request if delay period has passed
func (usm *UserStakingManager) ExecuteWithdrawal(account, nodeID string) (*Transaction, error) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	userStake := usm.getUserNodeStake(account, nodeID)
	if userStake == nil {
		return nil, ErrStakeNotFound
	}

	if !userStake.WithdrawalRequested {
		return nil, fmt.Errorf("no withdrawal request found")
	}

	if time.Now().Before(userStake.WithdrawalAt) {
		return nil, ErrWithdrawalDelayNotMet
	}

	amount := userStake.WithdrawalAmount

	// Transfer from staking pool back to user
	tx, err := usm.economicEngine.Transfer("staking_pool", account, amount, "Stake withdrawal from node "+nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to transfer funds from staking pool: %w", err)
	}

	// Update stake
	userStake.Amount.Sub(userStake.Amount, amount)
	userStake.WithdrawalRequested = false
	userStake.WithdrawalAt = time.Time{}
	userStake.WithdrawalAmount = nil
	userStake.LastUpdate = time.Now()

	// Remove stake if amount is zero
	if userStake.Amount.Cmp(big.NewInt(0)) == 0 {
		usm.removeUserStake(userStake)
	}

	log.Printf("User %s withdrew %s LORE from node %s (tx: %s)", account, amount.String(), nodeID, tx.ID)

	// Update node reputation
	usm.updateNodeReputation(nodeID)

	return tx, nil
}

// SlashNode slashes stakes for a node due to poor performance
func (usm *UserStakingManager) SlashNode(nodeID, reason string, severity float64, queryID string) (*SlashingEvent, error) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	if !usm.stakingConfig.SlashingEnabled {
		return nil, fmt.Errorf("slashing is disabled")
	}

	if severity <= 0 || severity > 1 {
		return nil, fmt.Errorf("severity must be between 0 and 1 (exclusive of 0)")
	}

	nodeStakes := usm.nodeStakes[nodeID]
	if len(nodeStakes) == 0 {
		return nil, fmt.Errorf("no stakes found for node %s", nodeID)
	}

	// Calculate slash percentage based on severity
	slashPercent := severity * usm.stakingConfig.MaxSlashingPercent

	var totalSlashed big.Int
	var affectedUsers []string

	// Slash each user's stake
	for _, stake := range nodeStakes {
		if stake.Amount.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		// Calculate slash amount for this stake
		slashFloat := new(big.Float).Mul(
			new(big.Float).SetInt(stake.Amount),
			big.NewFloat(slashPercent),
		)

		var slashAmount big.Int
		slashFloat.Int(&slashAmount)

		if slashAmount.Cmp(big.NewInt(0)) > 0 {
			// Reduce stake
			stake.Amount.Sub(stake.Amount, &slashAmount)
			stake.LastUpdate = time.Now()

			totalSlashed.Add(&totalSlashed, &slashAmount)
			affectedUsers = append(affectedUsers, stake.Account)

			log.Printf("Slashed %s LORE from user %s's stake on node %s",
				slashAmount.String(), stake.Account, nodeID)
		}
	}

	// Create slashing event
	slashingEvent := &SlashingEvent{
		ID:            uuid.New().String(),
		NodeID:        nodeID,
		Reason:        reason,
		Severity:      severity,
		SlashedAmount: &totalSlashed,
		AffectedUsers: affectedUsers,
		Timestamp:     time.Now(),
		QueryID:       queryID,
	}

	// Update node performance
	if usm.nodePerformance[nodeID] == nil {
		usm.nodePerformance[nodeID] = &NodePerformance{
			NodeID:         nodeID,
			SlashingEvents: []SlashingEvent{},
			TotalSlashed:   big.NewInt(0),
		}
	}

	performance := usm.nodePerformance[nodeID]
	performance.SlashingEvents = append(performance.SlashingEvents, *slashingEvent)
	performance.TotalSlashed.Add(performance.TotalSlashed, &totalSlashed)

	// Update node reputation (negative impact)
	usm.updateNodeReputation(nodeID)

	log.Printf("Slashed node %s: %s LORE total, reason: %s, severity: %.2f",
		nodeID, totalSlashed.String(), reason, severity)

	return slashingEvent, nil
}

// RecordQueryResult records the result of a query for performance tracking
func (usm *UserStakingManager) RecordQueryResult(nodeID, queryID string, success bool, responseTimeMs int64) {
	usm.mu.Lock()
	defer usm.mu.Unlock()

	if usm.nodePerformance[nodeID] == nil {
		usm.nodePerformance[nodeID] = &NodePerformance{
			NodeID:         nodeID,
			SlashingEvents: []SlashingEvent{},
			TotalSlashed:   big.NewInt(0),
		}
	}

	performance := usm.nodePerformance[nodeID]
	performance.LastActive = time.Now()

	if success {
		performance.QueriesCompleted++
	} else {
		performance.QueriesFailed++
	}

	// Update success rate
	total := performance.QueriesCompleted + performance.QueriesFailed
	if total > 0 {
		performance.SuccessRate = float64(performance.QueriesCompleted) / float64(total)
	}

	// Update average response time
	if responseTimeMs > 0 {
		if performance.AverageResponseTime == 0 {
			performance.AverageResponseTime = responseTimeMs
		} else {
			// Exponential moving average
			performance.AverageResponseTime = (performance.AverageResponseTime*9 + responseTimeMs) / 10
		}
	}

	// Check if slashing should be triggered
	if performance.SuccessRate < (1.0-usm.stakingConfig.MinSlashingThreshold) && total >= 10 {
		// Auto-slash for poor performance
		severity := (1.0 - performance.SuccessRate) * 0.5 // Up to 50% severity
		go func() {
			usm.SlashNode(nodeID, fmt.Sprintf("Poor success rate: %.2f%%", performance.SuccessRate*100),
				severity, queryID)
		}()
	}
}

// Helper functions

func (usm *UserStakingManager) getUserNodeStake(account, nodeID string) *UserStake {
	userStakes := usm.userNodeStakes[account]
	if userStakes == nil {
		return nil
	}
	return userStakes[nodeID]
}

func (usm *UserStakingManager) removeUserStake(stake *UserStake) {
	delete(usm.userStakes, stake.ID)

	// Remove from node stakes
	nodeStakes := usm.nodeStakes[stake.NodeID]
	for i, s := range nodeStakes {
		if s.ID == stake.ID {
			usm.nodeStakes[stake.NodeID] = append(nodeStakes[:i], nodeStakes[i+1:]...)
			break
		}
	}

	// Remove from user stakes
	delete(usm.userNodeStakes[stake.Account], stake.NodeID)
	if len(usm.userNodeStakes[stake.Account]) == 0 {
		delete(usm.userNodeStakes, stake.Account)
	}
}

func (usm *UserStakingManager) updateNodeReputation(nodeID string) {
	// Calculate total stake for this node
	totalStake := big.NewInt(0)
	for _, stake := range usm.nodeStakes[nodeID] {
		totalStake.Add(totalStake, stake.Amount)
	}

	// Calculate reputation boost based on stake
	stakeFloat, _ := new(big.Float).SetInt(totalStake).Float64()
	reputationBoost := stakeFloat * usm.stakingConfig.ReputationMultiplier

	// Apply performance penalties
	if performance := usm.nodePerformance[nodeID]; performance != nil {
		// Reduce reputation based on failure rate
		if performance.SuccessRate < 1.0 {
			penalty := (1.0 - performance.SuccessRate) * reputationBoost * 0.5
			reputationBoost -= penalty
		}
	}

	// Update reputation through economic engine
	if nodeAccount, err := usm.economicEngine.GetNodeAccount(nodeID); err == nil {
		// Apply performance penalties for slashing
		if performance := usm.nodePerformance[nodeID]; performance != nil && len(performance.SlashingEvents) > 0 {
			// Recent slashing should reduce reputation
			recentSlashingPenalty := 5.0 // Reduce by 5 points for recent slashing
			nodeAccount.Reputation -= recentSlashingPenalty
			if nodeAccount.Reputation < 0 {
				nodeAccount.Reputation = 0
			}
		}
		log.Printf("Updated reputation for node %s: reputation=%.2f, stake_boost=%.2f", nodeID, nodeAccount.Reputation, reputationBoost)
	} else {
		log.Printf("Could not update reputation for node %s: %v (boost: %.2f)", nodeID, err, reputationBoost)
	}
}

// Public query methods

// GetNodeStakeInfo returns staking information for a node
func (usm *UserStakingManager) GetNodeStakeInfo(nodeID string) *NodeStakeInfo {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	nodeStakes := usm.nodeStakes[nodeID]
	totalStake := big.NewInt(0)

	for _, stake := range nodeStakes {
		totalStake.Add(totalStake, stake.Amount)
	}

	stakeFloat, _ := new(big.Float).SetInt(totalStake).Float64()
	reputationBoost := stakeFloat * usm.stakingConfig.ReputationMultiplier

	return &NodeStakeInfo{
		NodeID:          nodeID,
		TotalUserStake:  totalStake,
		UserStakes:      append([]*UserStake{}, nodeStakes...), // Copy slice
		StakeCount:      len(nodeStakes),
		ReputationBoost: reputationBoost,
		PerformanceData: usm.nodePerformance[nodeID],
	}
}

// GetUserStakes returns all stakes for a user
func (usm *UserStakingManager) GetUserStakes(account string) []*UserStake {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	var stakes []*UserStake
	userStakes := usm.userNodeStakes[account]
	for _, stake := range userStakes {
		stakes = append(stakes, stake)
	}
	return stakes
}

// GetAllNodeStakes returns staking info for all nodes
func (usm *UserStakingManager) GetAllNodeStakes() map[string]*NodeStakeInfo {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	result := make(map[string]*NodeStakeInfo)
	for nodeID := range usm.nodeStakes {
		result[nodeID] = usm.GetNodeStakeInfo(nodeID)
	}
	return result
}

// GetStakingConfig returns the current staking configuration
func (usm *UserStakingManager) GetStakingConfig() *StakingConfig {
	usm.mu.RLock()
	defer usm.mu.RUnlock()

	// Return a copy
	config := *usm.stakingConfig
	return &config
}
