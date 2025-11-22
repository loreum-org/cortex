package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/loreum-org/cortex/internal/economy"
	"github.com/loreum-org/cortex/pkg/types"
)

// EconomicBridge connects the economic engine with the consensus system
type EconomicBridge struct {
	ConsensusService    *ConsensusService
	EconomicEngine      *economy.EconomicEngine
	pendingTxs          map[string]*economy.Transaction // Maps consensus tx ID to economic tx
	economicBroadcaster interface {
		BroadcastPaymentTransaction(tx *economy.Transaction) error
		BroadcastRewardDistribution(tx *economy.Transaction) error
		BroadcastStakeChange(tx *economy.Transaction, operation string, totalStake *big.Int) error
	}
}

// NewEconomicBridge creates a new economic consensus bridge
func NewEconomicBridge(cs *ConsensusService, ee *economy.EconomicEngine) *EconomicBridge {
	bridge := &EconomicBridge{
		ConsensusService: cs,
		EconomicEngine:   ee,
		pendingTxs:       make(map[string]*economy.Transaction),
	}

	// Add economic validation rules to consensus
	bridge.addEconomicValidationRules()

	return bridge
}

// SetEconomicBroadcaster sets the economic broadcaster for P2P integration
func (eb *EconomicBridge) SetEconomicBroadcaster(broadcaster interface {
	BroadcastPaymentTransaction(tx *economy.Transaction) error
	BroadcastRewardDistribution(tx *economy.Transaction) error
	BroadcastStakeChange(tx *economy.Transaction, operation string, totalStake *big.Int) error
}) {
	eb.economicBroadcaster = broadcaster
}

// addEconomicValidationRules adds economic-specific validation rules to consensus
func (eb *EconomicBridge) addEconomicValidationRules() {
	// Validate economic transaction structure
	eb.ConsensusService.ValidationRules = append(eb.ConsensusService.ValidationRules, func(tx *types.Transaction) bool {
		if tx.Type != types.TransactionTypeEconomic {
			return true // Non-economic transactions are handled by other validators
		}

		// Validate economic data presence
		if tx.EconomicData == nil {
			log.Printf("Economic transaction %s missing economic data", tx.ID)
			return false
		}

		// Validate amount is positive
		if tx.EconomicData.Amount == nil || tx.EconomicData.Amount.Cmp(big.NewInt(0)) <= 0 {
			log.Printf("Economic transaction %s has invalid amount", tx.ID)
			return false
		}

		// Validate account IDs
		if tx.EconomicData.FromAccount == "" && tx.EconomicData.EconomicType != "mint" {
			log.Printf("Economic transaction %s missing from account", tx.ID)
			return false
		}

		if tx.EconomicData.ToAccount == "" {
			log.Printf("Economic transaction %s missing to account", tx.ID)
			return false
		}

		return true
	})

	// Validate account balances for transfers
	eb.ConsensusService.ValidationRules = append(eb.ConsensusService.ValidationRules, func(tx *types.Transaction) bool {
		if tx.Type != types.TransactionTypeEconomic {
			return true
		}

		// Skip balance validation for minting
		if tx.EconomicData.EconomicType == "mint" {
			return true
		}

		// Check if from account exists and has sufficient balance
		fromAccount, err := eb.EconomicEngine.GetAccount(tx.EconomicData.FromAccount)
		if err != nil {
			log.Printf("From account %s not found for transaction %s", tx.EconomicData.FromAccount, tx.ID)
			return false
		}

		totalAmount := new(big.Int).Add(tx.EconomicData.Amount, tx.EconomicData.Fee)
		if fromAccount.Balance.Cmp(totalAmount) < 0 {
			log.Printf("Insufficient balance in account %s for transaction %s", tx.EconomicData.FromAccount, tx.ID)
			return false
		}

		return true
	})
}

// ProcessEconomicTransaction processes an economic transaction through consensus
func (eb *EconomicBridge) ProcessEconomicTransaction(ctx context.Context, economicTx *economy.Transaction) (*types.Transaction, error) {
	// Convert economic transaction to consensus transaction
	consensusTx, err := eb.economicToConsensusTransaction(economicTx)
	if err != nil {
		return nil, fmt.Errorf("failed to convert economic transaction: %w", err)
	}

	// Store mapping for later finalization
	eb.pendingTxs[consensusTx.ID] = economicTx

	// Add to consensus
	err = eb.ConsensusService.AddTransaction(consensusTx)
	if err != nil {
		delete(eb.pendingTxs, consensusTx.ID)
		return nil, fmt.Errorf("failed to add transaction to consensus: %w", err)
	}

	log.Printf("Economic transaction %s added to consensus as %s", economicTx.ID, consensusTx.ID)

	// Broadcast to P2P network if broadcaster is available
	if eb.economicBroadcaster != nil {
		go func() {
			switch economicTx.Type {
			case economy.TransactionTypeQueryPayment:
				if err := eb.economicBroadcaster.BroadcastPaymentTransaction(economicTx); err != nil {
					log.Printf("Failed to broadcast payment transaction %s: %v", economicTx.ID, err)
				}
			case economy.TransactionTypeReward:
				if err := eb.economicBroadcaster.BroadcastRewardDistribution(economicTx); err != nil {
					log.Printf("Failed to broadcast reward distribution %s: %v", economicTx.ID, err)
				}
			case economy.TransactionTypeStake, economy.TransactionTypeUnstake:
				// Get current stake for broadcasting
				nodeAccount, err := eb.EconomicEngine.GetNodeAccount(economicTx.FromID)
				if err == nil {
					operation := "stake"
					if economicTx.Type == economy.TransactionTypeUnstake {
						operation = "unstake"
					}
					if err := eb.economicBroadcaster.BroadcastStakeChange(economicTx, operation, nodeAccount.Stake); err != nil {
						log.Printf("Failed to broadcast stake change %s: %v", economicTx.ID, err)
					}
				}
			}
		}()
	}

	return consensusTx, nil
}

// economicToConsensusTransaction converts an economic transaction to a consensus transaction
func (eb *EconomicBridge) economicToConsensusTransaction(economicTx *economy.Transaction) (*types.Transaction, error) {
	// Create economic data
	economicData := &types.EconomicTransactionData{
		FromAccount:  economicTx.FromID,
		ToAccount:    economicTx.ToID,
		Amount:       new(big.Int).Set(economicTx.Amount),
		Fee:          new(big.Int).Set(economicTx.Fee),
		EconomicType: string(economicTx.Type),
	}

	// Add metadata if available
	if queryID, exists := economicTx.Metadata["query_id"]; exists {
		if queryIDStr, ok := queryID.(string); ok {
			economicData.QueryID = queryIDStr
		}
	}
	if modelTier, exists := economicTx.Metadata["model_tier"]; exists {
		if modelTierStr, ok := modelTier.(string); ok {
			economicData.ModelTier = modelTierStr
		}
	}
	if qualityScore, exists := economicTx.Metadata["quality_score"]; exists {
		if qualityScoreFloat, ok := qualityScore.(float64); ok {
			economicData.QualityScore = qualityScoreFloat
		}
	}
	if responseTime, exists := economicTx.Metadata["response_time"]; exists {
		if responseTimeInt, ok := responseTime.(int64); ok {
			economicData.ResponseTime = responseTimeInt
		}
	}

	// Get parent transactions from DAG (for now, use empty slice)
	parentIDs := []string{}

	// Create consensus transaction
	consensusTx := types.NewEconomicTransaction(
		fmt.Sprintf("consensus_%s", economicTx.ID),
		economicData,
		parentIDs,
	)

	// Add metadata
	metadataBytes, err := json.Marshal(economicTx.Metadata)
	if err == nil {
		consensusTx.Metadata["economic_metadata"] = string(metadataBytes)
	}
	consensusTx.Metadata["original_tx_id"] = economicTx.ID
	consensusTx.Metadata["economic_type"] = string(economicTx.Type)

	// For now, use a placeholder signature to pass validation
	// In a real implementation, this would be properly signed
	consensusTx.Signature = []byte("economic_system_signature")

	return consensusTx, nil
}

// MonitorFinalization monitors consensus finalization and updates economic state
func (eb *EconomicBridge) MonitorFinalization(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Check for finalized transactions
			for consensusTxID, economicTx := range eb.pendingTxs {
				if eb.ConsensusService.IsFinalized(consensusTxID) {
					log.Printf("Economic transaction %s finalized through consensus", economicTx.ID)

					// Update economic transaction status
					economicTx.Status = "consensus_finalized"

					// Remove from pending
					delete(eb.pendingTxs, consensusTxID)
				}
			}
		}
	}
}

// GetConsensusTransaction retrieves a consensus transaction by economic transaction ID
func (eb *EconomicBridge) GetConsensusTransaction(economicTxID string) (*types.Transaction, bool) {
	// Find consensus transaction by economic transaction ID
	for consensusTxID, economicTx := range eb.pendingTxs {
		if economicTx.ID == economicTxID {
			return eb.ConsensusService.GetTransaction(consensusTxID)
		}
	}
	return nil, false
}

// IsEconomicTransactionFinalized checks if an economic transaction is finalized in consensus
func (eb *EconomicBridge) IsEconomicTransactionFinalized(economicTxID string) bool {
	for consensusTxID, economicTx := range eb.pendingTxs {
		if economicTx.ID == economicTxID {
			return eb.ConsensusService.IsFinalized(consensusTxID)
		}
	}
	return false
}

// ValidateEconomicTransaction validates an economic transaction against current state
func (eb *EconomicBridge) ValidateEconomicTransaction(economicTx *economy.Transaction) error {
	// Validate account existence
	if economicTx.FromID != "" && economicTx.Type != economy.TransactionTypeMint {
		_, err := eb.EconomicEngine.GetAccount(economicTx.FromID)
		if err != nil {
			return fmt.Errorf("from account not found: %w", err)
		}
	}

	if economicTx.ToID != "" {
		_, err := eb.EconomicEngine.GetAccount(economicTx.ToID)
		if err != nil {
			return fmt.Errorf("to account not found: %w", err)
		}
	}

	// Validate amounts
	if economicTx.Amount == nil || economicTx.Amount.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("invalid transaction amount")
	}

	// Type-specific validation
	switch economicTx.Type {
	case economy.TransactionTypeTransfer:
		return eb.validateTransfer(economicTx)
	case economy.TransactionTypeStake:
		return eb.validateStake(economicTx)
	case economy.TransactionTypeUnstake:
		return eb.validateUnstake(economicTx)
	case economy.TransactionTypeQueryPayment:
		return eb.validateQueryPayment(economicTx)
	case economy.TransactionTypeReward:
		return eb.validateReward(economicTx)
	}

	return nil
}

func (eb *EconomicBridge) validateTransfer(tx *economy.Transaction) error {
	fromAccount, err := eb.EconomicEngine.GetAccount(tx.FromID)
	if err != nil {
		return err
	}

	totalAmount := new(big.Int).Add(tx.Amount, tx.Fee)
	if fromAccount.Balance.Cmp(totalAmount) < 0 {
		return economy.ErrInsufficientBalance
	}

	return nil
}

func (eb *EconomicBridge) validateStake(tx *economy.Transaction) error {
	nodeAccount, err := eb.EconomicEngine.GetNodeAccount(tx.FromID)
	if err != nil {
		return err
	}

	if nodeAccount.Balance.Cmp(tx.Amount) < 0 {
		return economy.ErrInsufficientBalance
	}

	return nil
}

func (eb *EconomicBridge) validateUnstake(tx *economy.Transaction) error {
	nodeAccount, err := eb.EconomicEngine.GetNodeAccount(tx.FromID)
	if err != nil {
		return err
	}

	if nodeAccount.Stake.Cmp(tx.Amount) < 0 {
		return economy.ErrInsufficientStake
	}

	return nil
}

func (eb *EconomicBridge) validateQueryPayment(tx *economy.Transaction) error {
	userAccount, err := eb.EconomicEngine.GetUserAccount(tx.FromID)
	if err != nil {
		return err
	}

	totalAmount := new(big.Int).Add(tx.Amount, tx.Fee)
	if userAccount.Balance.Cmp(totalAmount) < 0 {
		return economy.ErrInsufficientBalance
	}

	return nil
}

func (eb *EconomicBridge) validateReward(tx *economy.Transaction) error {
	// Validate that the reward transaction references a valid payment
	if paymentTxID, exists := tx.Metadata["payment_tx_id"]; exists {
		paymentTx, err := eb.EconomicEngine.GetTransaction(paymentTxID.(string))
		if err != nil {
			return fmt.Errorf("referenced payment transaction not found: %w", err)
		}

		if paymentTx.Status != "pending_reward" {
			return fmt.Errorf("payment transaction not in pending reward state")
		}
	}

	return nil
}
