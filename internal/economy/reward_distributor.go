package economy

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RewardDistributorConfig defines configuration for the automatic reward distribution service
type RewardDistributorConfig struct {
	// DistributionInterval is how often rewards are calculated and distributed
	DistributionInterval time.Duration

	// BaseRewardRate is the base annual percentage rate for staking rewards
	// Expressed as a decimal (0.05 = 5% APR)
	BaseRewardRate *big.Float

	// MinimumStakeForRewards is the minimum stake required to receive rewards
	MinimumStakeForRewards *big.Int

	// PerformanceMultiplier determines how much performance affects rewards
	// 1.0 = linear relationship, higher values increase the impact of performance
	PerformanceMultiplier float64

	// ReputationMultiplier determines how much reputation affects rewards
	// 1.0 = linear relationship, higher values increase the impact of reputation
	ReputationMultiplier float64

	// StakeMultiplier determines how much stake affects rewards
	// 1.0 = linear relationship, higher values increase rewards for higher stakes
	StakeMultiplier float64

	// MaxRewardPerDistribution caps the maximum reward per distribution cycle
	MaxRewardPerDistribution *big.Int

	// NetworkRewardPool is the account ID that holds the reward pool
	NetworkRewardPool string

	// EnablePerformanceSlashing enables penalties for poor performance
	EnablePerformanceSlashing bool

	// MinimumPerformanceThreshold is the minimum performance score to avoid slashing
	// Expressed as a decimal (0.7 = 70% performance required)
	MinimumPerformanceThreshold float64
}

// RewardDistributionEvent represents a reward distribution event
type RewardDistributionEvent struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	NodeID          string    `json:"node_id"`
	Amount          *big.Int  `json:"amount"`
	Reason          string    `json:"reason"`
	PerformanceData map[string]float64 `json:"performance_data,omitempty"`
	StakeAmount     *big.Int  `json:"stake_amount"`
	TransactionID   string    `json:"transaction_id,omitempty"`
}

// RewardDistributor handles automatic reward distribution to node operators
type RewardDistributor struct {
	config             RewardDistributorConfig
	economicEngine     *EconomicEngine
	lastDistribution   time.Time
	distributionEvents []*RewardDistributionEvent
	nodePerformance    map[string]*NodePerformance
	mu                 sync.RWMutex
	ctx                context.Context
	cancel             context.CancelFunc
}

// NewRewardDistributor creates a new reward distributor with the given configuration
func NewRewardDistributor(engine *EconomicEngine, config RewardDistributorConfig) *RewardDistributor {
	// Set default values if not provided
	if config.DistributionInterval == 0 {
		config.DistributionInterval = 24 * time.Hour // Default to daily distribution
	}

	if config.BaseRewardRate == nil {
		config.BaseRewardRate = big.NewFloat(0.05) // Default to 5% APR
	}

	if config.MinimumStakeForRewards == nil {
		config.MinimumStakeForRewards = MinimumStake // Use global minimum stake
	}

	if config.MaxRewardPerDistribution == nil {
		// Default to 100 tokens per distribution
		config.MaxRewardPerDistribution = new(big.Int).Mul(big.NewInt(100), TokenUnit)
	}

	if config.NetworkRewardPool == "" {
		config.NetworkRewardPool = "network" // Default to network account
	}

	if config.PerformanceMultiplier == 0 {
		config.PerformanceMultiplier = 1.0
	}

	if config.ReputationMultiplier == 0 {
		config.ReputationMultiplier = 1.0
	}

	if config.StakeMultiplier == 0 {
		config.StakeMultiplier = 1.0
	}

	if config.MinimumPerformanceThreshold == 0 {
		config.MinimumPerformanceThreshold = 0.7 // Default to 70% minimum performance
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &RewardDistributor{
		config:             config,
		economicEngine:     engine,
		lastDistribution:   time.Now(),
		distributionEvents: make([]*RewardDistributionEvent, 0),
		nodePerformance:    make(map[string]*NodePerformance),
		ctx:                ctx,
		cancel:             cancel,
	}
}

// Start begins the periodic reward distribution process
func (rd *RewardDistributor) Start() {
	log.Printf("Starting automatic reward distribution service (interval: %v)", rd.config.DistributionInterval)
	
	// Start the distribution loop in a goroutine
	go rd.distributionLoop()
}

// Stop halts the reward distribution process
func (rd *RewardDistributor) Stop() {
	if rd.cancel != nil {
		rd.cancel()
	}
}

// distributionLoop runs the periodic reward distribution
func (rd *RewardDistributor) distributionLoop() {
	ticker := time.NewTicker(rd.config.DistributionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := rd.DistributeRewards(); err != nil {
				log.Printf("Error during reward distribution: %v", err)
			}
		case <-rd.ctx.Done():
			log.Printf("Reward distribution service stopped")
			return
		}
	}
}

// DistributeRewards calculates and distributes rewards to all eligible nodes
func (rd *RewardDistributor) DistributeRewards() error {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	log.Printf("Starting reward distribution cycle")
	
	// Get all node stakes
	allNodeStakes := rd.economicEngine.GetAllNodeStakes()
	if len(allNodeStakes) == 0 {
		log.Printf("No staked nodes found, skipping reward distribution")
		return nil
	}

	// Track distribution statistics
	totalDistributed := big.NewInt(0)
	distributionCount := 0
	distributionEvents := make([]*RewardDistributionEvent, 0)
	
	// Calculate time since last distribution for prorating rewards
	timeSinceLastDist := time.Since(rd.lastDistribution)
	yearFraction := big.NewFloat(timeSinceLastDist.Hours() / (365 * 24))
	
	// Process each node
	for nodeID, stakeInfo := range allNodeStakes {
		// Skip nodes with insufficient stake
		if stakeInfo.TotalUserStake.Cmp(rd.config.MinimumStakeForRewards) < 0 {
			continue
		}

		// Calculate and distribute reward for this node
		reward, event, err := rd.calculateAndDistributeNodeReward(nodeID, stakeInfo, yearFraction)
		if err != nil {
			log.Printf("Error distributing reward to node %s: %v", nodeID, err)
			continue
		}

		if reward.Cmp(big.NewInt(0)) > 0 {
			totalDistributed = new(big.Int).Add(totalDistributed, reward)
			distributionCount++
			distributionEvents = append(distributionEvents, event)
		}
	}

	// Update last distribution time
	rd.lastDistribution = time.Now()
	
	// Add events to history
	rd.distributionEvents = append(distributionEvents, rd.distributionEvents...)
	// Limit event history size
	if len(rd.distributionEvents) > 1000 {
		rd.distributionEvents = rd.distributionEvents[:1000]
	}

	log.Printf("Reward distribution complete: %d nodes received rewards, total distributed: %s tokens",
		distributionCount, formatBigInt(totalDistributed))

	return nil
}

// calculateAndDistributeNodeReward calculates and distributes reward for a single node
func (rd *RewardDistributor) calculateAndDistributeNodeReward(
	nodeID string, 
	stakeInfo *NodeStakeInfo, 
	yearFraction *big.Float,
) (*big.Int, *RewardDistributionEvent, error) {
	// Get node account
	nodeAccount, err := rd.economicEngine.GetNodeAccount(nodeID)
	if err != nil {
		return big.NewInt(0), nil, fmt.Errorf("node account not found: %w", err)
	}

	// Get performance data
	performanceData := rd.getNodePerformance(nodeID, stakeInfo)
	
	// Calculate base reward (stake * base rate * time fraction)
	baseRewardFloat := new(big.Float).SetInt(stakeInfo.TotalUserStake)
	baseRewardFloat = baseRewardFloat.Mul(baseRewardFloat, rd.config.BaseRewardRate)
	baseRewardFloat = baseRewardFloat.Mul(baseRewardFloat, yearFraction)
	
	var baseReward big.Int
	baseRewardFloat.Int(&baseReward)

	// Apply performance multiplier
	performanceScore := rd.calculatePerformanceScore(performanceData)
	
	// Check if node meets minimum performance threshold
	if performanceScore < rd.config.MinimumPerformanceThreshold && rd.config.EnablePerformanceSlashing {
		// Node performed poorly, potentially apply slashing instead of rewards
		return rd.handlePoorPerformance(nodeID, stakeInfo, performanceScore)
	}

	// Calculate performance multiplier (ranges from 0.5 to 2.0 based on performance)
	perfMultiplier := 0.5 + (1.5 * performanceScore * rd.config.PerformanceMultiplier)
	if perfMultiplier > 2.0 {
		perfMultiplier = 2.0 // Cap at 2x
	}

	// Apply reputation multiplier
	repMultiplier := 1.0
	if nodeAccount.Reputation > 0 {
		// Scale from 0.8 to 1.5 based on reputation (0-100)
		repMultiplier = 0.8 + (nodeAccount.Reputation / 100.0 * 0.7 * rd.config.ReputationMultiplier)
		if repMultiplier > 1.5 {
			repMultiplier = 1.5 // Cap at 1.5x
		}
	}

	// Apply stake multiplier (logarithmic scale to benefit smaller stakers)
	stakeMultiplier := 1.0
	if rd.config.StakeMultiplier > 0 {
		// Calculate stake size relative to minimum stake
		stakeRatio := new(big.Float).SetInt(stakeInfo.TotalUserStake)
		minStakeFloat := new(big.Float).SetInt(rd.config.MinimumStakeForRewards)
		stakeRatio = stakeRatio.Quo(stakeRatio, minStakeFloat)
		
		// Convert to float64 for calculation
		stakeRatioFloat, _ := stakeRatio.Float64()
		
		// Logarithmic scale: 1.0 + 0.2 * log10(stakeRatio) * stakeMultiplier
		if stakeRatioFloat > 1.0 {
			import "math"
			logFactor := 1.0 + 0.2 * math.Log10(stakeRatioFloat) * rd.config.StakeMultiplier
			if logFactor > 2.0 {
				logFactor = 2.0 // Cap at 2x
			}
			stakeMultiplier = logFactor
		}
	}

	// Calculate final reward with all multipliers
	totalMultiplier := perfMultiplier * repMultiplier * stakeMultiplier
	rewardFloat := new(big.Float).SetInt(&baseReward)
	rewardFloat = rewardFloat.Mul(rewardFloat, big.NewFloat(totalMultiplier))
	
	var finalReward big.Int
	rewardFloat.Int(&finalReward)

	// Ensure reward doesn't exceed maximum
	if finalReward.Cmp(rd.config.MaxRewardPerDistribution) > 0 {
		finalReward = *rd.config.MaxRewardPerDistribution
	}

	// Skip if reward is zero
	if finalReward.Cmp(big.NewInt(0)) <= 0 {
		return big.NewInt(0), nil, nil
	}

	// Create distribution event
	event := &RewardDistributionEvent{
		ID:              uuid.New().String(),
		Timestamp:       time.Now(),
		NodeID:          nodeID,
		Amount:          &finalReward,
		Reason:          "periodic_staking_reward",
		PerformanceData: map[string]float64{
			"performance_score": performanceScore,
			"perf_multiplier":   perfMultiplier,
			"rep_multiplier":    repMultiplier,
			"stake_multiplier":  stakeMultiplier,
			"total_multiplier":  totalMultiplier,
		},
		StakeAmount:     stakeInfo.TotalUserStake,
	}

	// Distribute the reward
	tx, err := rd.economicEngine.Transfer(
		rd.config.NetworkRewardPool,
		nodeID,
		&finalReward,
		fmt.Sprintf("Staking reward: perf=%.2f, rep=%.2f", performanceScore, nodeAccount.Reputation),
	)
	
	if err != nil {
		return big.NewInt(0), nil, fmt.Errorf("failed to transfer reward: %w", err)
	}

	// Update event with transaction ID
	event.TransactionID = tx.ID

	// Log the reward
	log.Printf("Distributed reward of %s tokens to node %s (perf: %.2f, rep: %.2f, stake: %s)",
		formatBigInt(&finalReward), nodeID, performanceScore, nodeAccount.Reputation, formatBigInt(stakeInfo.TotalUserStake))

	return &finalReward, event, nil
}

// handlePoorPerformance handles nodes that performed below threshold
func (rd *RewardDistributor) handlePoorPerformance(
	nodeID string,
	stakeInfo *NodeStakeInfo,
	performanceScore float64,
) (*big.Int, *RewardDistributionEvent, error) {
	// For now, just skip rewards rather than slashing
	log.Printf("Node %s performed below threshold (%.2f < %.2f), skipping rewards",
		nodeID, performanceScore, rd.config.MinimumPerformanceThreshold)
	
	// Create event to track the skipped reward
	event := &RewardDistributionEvent{
		ID:              uuid.New().String(),
		Timestamp:       time.Now(),
		NodeID:          nodeID,
		Amount:          big.NewInt(0),
		Reason:          "below_performance_threshold",
		PerformanceData: map[string]float64{
			"performance_score": performanceScore,
			"threshold":         rd.config.MinimumPerformanceThreshold,
		},
		StakeAmount:     stakeInfo.TotalUserStake,
	}
	
	// In a full implementation, we might implement slashing here
	// For now, we just return zero reward
	return big.NewInt(0), event, nil
}

// getNodePerformance retrieves or initializes performance data for a node
func (rd *RewardDistributor) getNodePerformance(nodeID string, stakeInfo *NodeStakeInfo) *NodePerformance {
	// Check if we already have performance data
	if perf, exists := rd.nodePerformance[nodeID]; exists {
		return perf
	}
	
	// Use performance data from stake info if available
	if stakeInfo.PerformanceData != nil {
		rd.nodePerformance[nodeID] = stakeInfo.PerformanceData
		return stakeInfo.PerformanceData
	}
	
	// Create default performance data
	perf := &NodePerformance{
		NodeID:             nodeID,
		QueriesCompleted:   0,
		QueriesFailed:      0,
		SuccessRate:        1.0, // Assume perfect initially
		AverageResponseTime: 0,
		LastActive:         time.Now(),
		SlashingEvents:     []SlashingEvent{},
		TotalSlashed:       big.NewInt(0),
	}
	
	rd.nodePerformance[nodeID] = perf
	return perf
}

// calculatePerformanceScore calculates a normalized performance score (0.0-1.0)
func (rd *RewardDistributor) calculatePerformanceScore(perf *NodePerformance) float64 {
	if perf == nil {
		return 0.5 // Default score for unknown performance
	}
	
	// Calculate base score from success rate (0.0-0.8)
	baseScore := perf.SuccessRate * 0.8
	
	// Add bonus for recent activity (0.0-0.2)
	activityBonus := 0.0
	hoursSinceActive := time.Since(perf.LastActive).Hours()
	if hoursSinceActive < 24 {
		activityBonus = 0.2 * (24 - hoursSinceActive) / 24
	}
	
	// Penalize for response time (0.0-0.2)
	responseTimePenalty := 0.0
	if perf.AverageResponseTime > 0 {
		// Normalize response time (0ms = 0.0 penalty, 5000ms+ = 0.2 penalty)
		responseTimePenalty = math.Min(0.2, float64(perf.AverageResponseTime) / 25000.0)
	}
	
	// Penalize for slashing events
	slashingPenalty := 0.0
	if len(perf.SlashingEvents) > 0 {
		// More recent slashing events have higher impact
		for _, event := range perf.SlashingEvents {
			daysSince := time.Since(event.Timestamp).Hours() / 24
			if daysSince < 30 { // Only consider events in the last 30 days
				// Severity ranges from 0.0 to 1.0
				impact := event.Severity * (30 - daysSince) / 30
				slashingPenalty += impact * 0.1 // Each event can reduce score by up to 10%
			}
		}
		slashingPenalty = math.Min(0.5, slashingPenalty) // Cap at 50% reduction
	}
	
	// Calculate final score
	finalScore := baseScore + activityBonus - responseTimePenalty - slashingPenalty
	
	// Ensure score is between 0.0 and 1.0
	return math.Max(0.0, math.Min(1.0, finalScore))
}

// GetDistributionEvents returns recent distribution events
func (rd *RewardDistributor) GetDistributionEvents(limit int) []*RewardDistributionEvent {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	
	if limit <= 0 || limit > len(rd.distributionEvents) {
		limit = len(rd.distributionEvents)
	}
	
	return rd.distributionEvents[:limit]
}

// GetNodeRewardHistory returns reward history for a specific node
func (rd *RewardDistributor) GetNodeRewardHistory(nodeID string, limit int) []*RewardDistributionEvent {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	
	var nodeEvents []*RewardDistributionEvent
	for _, event := range rd.distributionEvents {
		if event.NodeID == nodeID {
			nodeEvents = append(nodeEvents, event)
			if limit > 0 && len(nodeEvents) >= limit {
				break
			}
		}
	}
	
	return nodeEvents
}

// UpdateNodePerformance updates the performance metrics for a node
func (rd *RewardDistributor) UpdateNodePerformance(nodeID string, performance *NodePerformance) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	
	rd.nodePerformance[nodeID] = performance
}

// GetRewardStats returns statistics about reward distribution
func (rd *RewardDistributor) GetRewardStats() map[string]interface{} {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	
	totalDistributed := big.NewInt(0)
	nodeCount := make(map[string]bool)
	
	for _, event := range rd.distributionEvents {
		if event.Amount != nil {
			totalDistributed = new(big.Int).Add(totalDistributed, event.Amount)
		}
		nodeCount[event.NodeID] = true
	}
	
	return map[string]interface{}{
		"total_distributed":      totalDistributed.String(),
		"distribution_count":     len(rd.distributionEvents),
		"unique_nodes_rewarded":  len(nodeCount),
		"last_distribution_time": rd.lastDistribution,
		"next_distribution_time": rd.lastDistribution.Add(rd.config.DistributionInterval),
		"distribution_interval":  rd.config.DistributionInterval.String(),
		"base_reward_rate":       rd.config.BaseRewardRate.String(),
	}
}

// formatBigInt formats a big.Int for logging
func formatBigInt(n *big.Int) string {
	if n == nil {
		return "0"
	}
	
	// Convert to LORE tokens for readability
	tokenUnit := new(big.Float).SetInt(TokenUnit)
	tokens := new(big.Float).SetInt(n)
	tokens = tokens.Quo(tokens, tokenUnit)
	
	return tokens.Text('f', 4)
}
