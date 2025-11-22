package p2p

import (
	"log"
	"math/big"
	"time"

	"github.com/loreum-org/cortex/internal/economy"
)

// EconomicBroadcaster handles broadcasting economic events over the P2P network
type EconomicBroadcaster struct {
	p2pNode *P2PNode
	nodeID  string
}

// NewEconomicBroadcaster creates a new economic broadcaster
func NewEconomicBroadcaster(p2pNode *P2PNode, nodeID string) *EconomicBroadcaster {
	broadcaster := &EconomicBroadcaster{
		p2pNode: p2pNode,
		nodeID:  nodeID,
	}

	// Register handlers for incoming economic messages
	broadcaster.registerHandlers()

	return broadcaster
}

// registerHandlers registers handlers for incoming economic messages
func (eb *EconomicBroadcaster) registerHandlers() {
	// Handle incoming payment broadcasts
	eb.p2pNode.RegisterMessageHandler(MessageTypePayment, func(msg *NetworkMessage) error {
		log.Printf("Received payment broadcast from %s", msg.SenderID)
		return eb.handlePaymentBroadcast(msg)
	})

	// Handle incoming reward broadcasts
	eb.p2pNode.RegisterMessageHandler(MessageTypeReward, func(msg *NetworkMessage) error {
		log.Printf("Received reward broadcast from %s", msg.SenderID)
		return eb.handleRewardBroadcast(msg)
	})

	// Handle incoming stake update broadcasts
	eb.p2pNode.RegisterMessageHandler(MessageTypeStakeUpdate, func(msg *NetworkMessage) error {
		log.Printf("Received stake update broadcast from %s", msg.SenderID)
		return eb.handleStakeUpdateBroadcast(msg)
	})

	// Handle incoming balance update broadcasts
	eb.p2pNode.RegisterMessageHandler(MessageTypeBalanceUpdate, func(msg *NetworkMessage) error {
		log.Printf("Received balance update broadcast from %s", msg.SenderID)
		return eb.handleBalanceUpdateBroadcast(msg)
	})

	// Handle incoming economic sync broadcasts
	eb.p2pNode.RegisterMessageHandler(MessageTypeEconomicSync, func(msg *NetworkMessage) error {
		log.Printf("Received economic sync broadcast from %s", msg.SenderID)
		return eb.handleEconomicSyncBroadcast(msg)
	})
}

// BroadcastPaymentTransaction broadcasts a payment transaction to the network
func (eb *EconomicBroadcaster) BroadcastPaymentTransaction(tx *economy.Transaction) error {
	// Extract metadata
	modelTier := ""
	queryType := ""
	if tier, exists := tx.Metadata["model_tier"]; exists {
		if tierStr, ok := tier.(string); ok {
			modelTier = tierStr
		}
	}
	if qType, exists := tx.Metadata["query_type"]; exists {
		if qTypeStr, ok := qType.(string); ok {
			queryType = qTypeStr
		}
	}

	payment := &PaymentBroadcast{
		TransactionID: tx.ID,
		FromAccount:   tx.FromID,
		ToAccount:     tx.ToID,
		Amount:        tx.Amount.String(),
		Fee:           tx.Fee.String(),
		ModelTier:     modelTier,
		QueryType:     queryType,
		Timestamp:     tx.Timestamp.Unix(),
	}

	log.Printf("Broadcasting payment transaction %s to network", tx.ID)
	return eb.p2pNode.BroadcastPayment(payment)
}

// BroadcastRewardDistribution broadcasts a reward distribution to the network
func (eb *EconomicBroadcaster) BroadcastRewardDistribution(tx *economy.Transaction) error {
	// Extract metadata
	paymentTxID := ""
	responseTime := int64(0)
	qualityScore := 0.0
	success := false
	reputationDelta := 0.0

	if ptxID, exists := tx.Metadata["payment_tx_id"]; exists {
		if ptxIDStr, ok := ptxID.(string); ok {
			paymentTxID = ptxIDStr
		}
	}
	if rt, exists := tx.Metadata["response_time"]; exists {
		if rtInt, ok := rt.(int64); ok {
			responseTime = rtInt
		}
	}
	if qs, exists := tx.Metadata["quality_score"]; exists {
		if qsFloat, ok := qs.(float64); ok {
			qualityScore = qsFloat
		}
	}
	if s, exists := tx.Metadata["success"]; exists {
		if sBool, ok := s.(bool); ok {
			success = sBool
		}
	}
	if rd, exists := tx.Metadata["reputation_delta"]; exists {
		if rdFloat, ok := rd.(float64); ok {
			reputationDelta = rdFloat
		}
	}

	reward := &RewardBroadcast{
		TransactionID:   tx.ID,
		PaymentTxID:     paymentTxID,
		NodeID:          tx.ToID,
		Amount:          tx.Amount.String(),
		ResponseTime:    responseTime,
		QualityScore:    qualityScore,
		Success:         success,
		ReputationDelta: reputationDelta,
		Timestamp:       tx.Timestamp.Unix(),
	}

	log.Printf("Broadcasting reward distribution %s to network", tx.ID)
	return eb.p2pNode.BroadcastReward(reward)
}

// BroadcastStakeChange broadcasts a stake change to the network
func (eb *EconomicBroadcaster) BroadcastStakeChange(tx *economy.Transaction, operation string, totalStake *big.Int) error {
	stakeUpdate := &StakeUpdateBroadcast{
		NodeID:        tx.FromID,
		Amount:        tx.Amount.String(),
		Operation:     operation,
		TransactionID: tx.ID,
		TotalStake:    totalStake.String(),
		Timestamp:     tx.Timestamp.Unix(),
	}

	log.Printf("Broadcasting stake %s for node %s to network", operation, tx.FromID)
	return eb.p2pNode.BroadcastStakeUpdate(stakeUpdate)
}

// BroadcastBalanceChange broadcasts a balance change to the network
func (eb *EconomicBroadcaster) BroadcastBalanceChange(accountID string, newBalance *big.Int, changeAmount *big.Int, changeType string, txID string) error {
	balanceUpdate := &BalanceUpdateBroadcast{
		AccountID:     accountID,
		NewBalance:    newBalance.String(),
		ChangeAmount:  changeAmount.String(),
		ChangeType:    changeType,
		TransactionID: txID,
		Timestamp:     time.Now().Unix(),
	}

	log.Printf("Broadcasting balance %s for account %s to network", changeType, accountID)
	return eb.p2pNode.BroadcastBalanceUpdate(balanceUpdate)
}

// BroadcastEconomicStateSync broadcasts economic state synchronization to the network
func (eb *EconomicBroadcaster) BroadcastEconomicStateSync(economicHeight int64, stateHash string, nodeBalances map[string]string, nodeStakes map[string]string) error {
	syncMsg := &EconomicSyncMessage{
		SenderID:       eb.nodeID,
		EconomicHeight: economicHeight,
		StateHash:      stateHash,
		NodeBalances:   nodeBalances,
		NodeStakes:     nodeStakes,
		Timestamp:      time.Now().Unix(),
	}

	log.Printf("Broadcasting economic state sync (height: %d) to network", economicHeight)
	return eb.p2pNode.BroadcastEconomicSync(syncMsg)
}

// handlePaymentBroadcast handles incoming payment broadcasts
func (eb *EconomicBroadcaster) handlePaymentBroadcast(msg *NetworkMessage) error {
	// Extract payment data from payload
	paymentData, exists := msg.Payload["payment"]
	if !exists {
		log.Printf("Payment broadcast missing payment data")
		return nil
	}

	// In a real implementation, this would:
	// 1. Validate the payment transaction
	// 2. Update local economic state if valid
	// 3. Forward to consensus if needed
	log.Printf("Processing payment broadcast: %+v", paymentData)

	return nil
}

// handleRewardBroadcast handles incoming reward broadcasts
func (eb *EconomicBroadcaster) handleRewardBroadcast(msg *NetworkMessage) error {
	// Extract reward data from payload
	rewardData, exists := msg.Payload["reward"]
	if !exists {
		log.Printf("Reward broadcast missing reward data")
		return nil
	}

	// In a real implementation, this would:
	// 1. Validate the reward distribution
	// 2. Update node reputation metrics
	// 3. Record performance statistics
	log.Printf("Processing reward broadcast: %+v", rewardData)

	return nil
}

// handleStakeUpdateBroadcast handles incoming stake update broadcasts
func (eb *EconomicBroadcaster) handleStakeUpdateBroadcast(msg *NetworkMessage) error {
	// Extract stake update data from payload
	stakeData, exists := msg.Payload["stake_update"]
	if !exists {
		log.Printf("Stake update broadcast missing stake data")
		return nil
	}

	// In a real implementation, this would:
	// 1. Validate the stake change
	// 2. Update network consensus parameters
	// 3. Adjust node selection weights
	log.Printf("Processing stake update broadcast: %+v", stakeData)

	return nil
}

// handleBalanceUpdateBroadcast handles incoming balance update broadcasts
func (eb *EconomicBroadcaster) handleBalanceUpdateBroadcast(msg *NetworkMessage) error {
	// Extract balance update data from payload
	balanceData, exists := msg.Payload["balance_update"]
	if !exists {
		log.Printf("Balance update broadcast missing balance data")
		return nil
	}

	// In a real implementation, this would:
	// 1. Validate the balance change
	// 2. Update account tracking
	// 3. Sync with local economic state
	log.Printf("Processing balance update broadcast: %+v", balanceData)

	return nil
}

// handleEconomicSyncBroadcast handles incoming economic sync broadcasts
func (eb *EconomicBroadcaster) handleEconomicSyncBroadcast(msg *NetworkMessage) error {
	// Extract economic sync data from payload
	syncData, exists := msg.Payload["economic_sync"]
	if !exists {
		log.Printf("Economic sync broadcast missing sync data")
		return nil
	}

	// In a real implementation, this would:
	// 1. Compare economic state hashes
	// 2. Request missing transactions if behind
	// 3. Validate and merge economic state
	log.Printf("Processing economic sync broadcast: %+v", syncData)

	return nil
}
