package economy

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/loreum-org/cortex/pkg/types"
)

const (
	// TokenDecimals represents the number of decimal places for the LOREUM token
	TokenDecimals = 18

	// NetworkFeePercent is the percentage of each transaction that goes to the network
	NetworkFeePercent = 5

	// ReputationDecayFactor is how much reputation decays per day (in percentage)
	ReputationDecayFactor = 0.01

	// InitialReputationScore is the starting reputation for new nodes
	InitialReputationScore = 50.0

	// MaxReputationScore is the maximum reputation a node can have
	MaxReputationScore = 100.0
)

var (
	// TokenUnit represents 1 LOREUM token in its smallest unit (10^18)
	TokenUnit = new(big.Int).Exp(big.NewInt(10), big.NewInt(TokenDecimals), nil)

	// MinimumStake is the minimum amount of tokens required to participate as a node
	MinimumStake = new(big.Int).Mul(big.NewInt(1000), TokenUnit)

	// ErrInsufficientBalance occurs when an account has insufficient balance for a transaction
	ErrInsufficientBalance = errors.New("insufficient balance")

	// ErrInsufficientStake occurs when a node has insufficient stake
	ErrInsufficientStake = errors.New("insufficient stake to participate")

	// ErrAccountNotFound occurs when an account is not found
	ErrAccountNotFound = errors.New("account not found")

	// ErrInvalidAmount occurs when an invalid amount is provided
	ErrInvalidAmount = errors.New("invalid amount")

	// ErrUnauthorized occurs when an operation is not authorized
	ErrUnauthorized = errors.New("unauthorized operation")

	// ErrTransactionFailed occurs when a transaction fails
	ErrTransactionFailed = errors.New("transaction failed")
)

// TransactionType represents the type of economic transaction
type TransactionType string

const (
	// TransactionTypeQueryPayment represents a payment for a query
	TransactionTypeQueryPayment TransactionType = "query_payment"

	// TransactionTypeReward represents a reward for providing inference
	TransactionTypeReward TransactionType = "reward"

	// TransactionTypeStake represents a staking transaction
	TransactionTypeStake TransactionType = "stake"

	// TransactionTypeUnstake represents an unstaking transaction
	TransactionTypeUnstake TransactionType = "unstake"

	// TransactionTypeTransfer represents a token transfer
	TransactionTypeTransfer TransactionType = "transfer"

	// TransactionTypeNetworkFee represents a network fee
	TransactionTypeNetworkFee TransactionType = "network_fee"

	// TransactionTypeSlash represents a slashing penalty
	TransactionTypeSlash TransactionType = "slash"

	// TransactionTypeMint represents token minting
	TransactionTypeMint TransactionType = "mint"
)

// ModelTier represents the pricing tier of an AI model
type ModelTier string

const (
	// ModelTierBasic represents basic models (e.g., small LLMs)
	ModelTierBasic ModelTier = "basic"

	// ModelTierStandard represents standard models (e.g., medium LLMs)
	ModelTierStandard ModelTier = "standard"

	// ModelTierPremium represents premium models (e.g., large LLMs)
	ModelTierPremium ModelTier = "premium"

	// ModelTierEnterprise represents enterprise-grade models
	ModelTierEnterprise ModelTier = "enterprise"
)

// QueryType represents the type of AI query
type QueryType string

const (
	// QueryTypeCompletion represents a text completion query
	QueryTypeCompletion QueryType = "completion"

	// QueryTypeChat represents a chat interaction
	QueryTypeChat QueryType = "chat"

	// QueryTypeEmbedding represents an embedding generation
	QueryTypeEmbedding QueryType = "embedding"

	// QueryTypeCodeGen represents code generation
	QueryTypeCodeGen QueryType = "codegen"

	// QueryTypeRAG represents a RAG query
	QueryTypeRAG QueryType = "rag"
)

// Account represents a token account in the Loreum Network
type Account struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Balance   *big.Int  `json:"balance"`
	Stake     *big.Int  `json:"stake"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NodeAccount extends Account with node-specific fields
type NodeAccount struct {
	*Account
	Reputation       float64            `json:"reputation"`
	TotalEarned      *big.Int           `json:"total_earned"`
	QueriesProcessed int64              `json:"queries_processed"`
	SuccessRate      float64            `json:"success_rate"`
	AvgResponseTime  int64              `json:"avg_response_time_ms"`
	LastActive       time.Time          `json:"last_active"`
	Capabilities     map[string]bool    `json:"capabilities"`
	Performance      map[string]float64 `json:"performance"`
}

// UserAccount extends Account with user-specific fields
type UserAccount struct {
	*Account
	TotalSpent       *big.Int          `json:"total_spent"`
	QueriesSubmitted int64             `json:"queries_submitted"`
	UsageByModel     map[string]int64  `json:"usage_by_model"`
	UsageByType      map[QueryType]int `json:"usage_by_type"`
}

// Transaction represents an economic transaction in the Loreum Network
type Transaction struct {
	ID          string          `json:"id"`
	Type        TransactionType `json:"type"`
	FromID      string          `json:"from_id"`
	ToID        string          `json:"to_id"`
	Amount      *big.Int        `json:"amount"`
	Fee         *big.Int        `json:"fee"`
	Description string          `json:"description"`
	Metadata    map[string]any  `json:"metadata"`
	Timestamp   time.Time       `json:"timestamp"`
	Status      string          `json:"status"`
}

// PricingRule defines pricing for different models and query types
type PricingRule struct {
	ModelTier        ModelTier  `json:"model_tier"`
	QueryType        QueryType  `json:"query_type"`
	BasePrice        *big.Int   `json:"base_price"`
	TokenPerMS       *big.Float `json:"token_per_ms"`
	TokenPerKB       *big.Float `json:"token_per_kb"`
	MinPrice         *big.Int   `json:"min_price"`
	MaxPrice         *big.Int   `json:"max_price"`
	DemandMultiplier *big.Float `json:"demand_multiplier"`
}

// RewardConfig defines how rewards are calculated
type RewardConfig struct {
	BaseReward           *big.Int   `json:"base_reward"`
	ReputationMultiplier *big.Float `json:"reputation_multiplier"`
	SpeedBonus           *big.Float `json:"speed_bonus"`
	QualityBonus         *big.Float `json:"quality_bonus"`
	StakeMultiplier      *big.Float `json:"stake_multiplier"`
}

// EconomicEngine is the core component managing the Loreum Network economy
type EconomicEngine struct {
	accounts        map[string]*Account
	nodeAccounts    map[string]*NodeAccount
	userAccounts    map[string]*UserAccount
	transactions    []*Transaction
	pricingRules    map[ModelTier]map[QueryType]*PricingRule
	rewardConfig    RewardConfig
	networkAccount  string
	totalSupply     *big.Int
	demandFactors   map[ModelTier]*big.Float
	mutex           sync.RWMutex
	queryCounter    map[ModelTier]int64
	lastPriceUpdate time.Time

	// User staking system
	userStakingManager *UserStakingManager

	// Reward distribution system
	rewardDistributor *RewardDistributor

	// Consensus integration
	consensusBridge interface {
		ProcessEconomicTransaction(ctx context.Context, tx *Transaction) (*types.Transaction, error)
		IsEconomicTransactionFinalized(txID string) bool
	}

	// Storage integration
	storage interface {
		StoreAccount(account *Account) error
		LoadAccount(accountID string) (*Account, error)
		StoreNodeAccount(nodeAccount *NodeAccount) error
		LoadNodeAccount(nodeID string) (*NodeAccount, error)
		StoreUserAccount(userAccount *UserAccount) error
		LoadUserAccount(userID string) (*UserAccount, error)
		StoreTransaction(tx *Transaction) error
		LoadTransaction(txID string) (*Transaction, error)
		GetTransactionsByAccount(accountID string, limit, offset int) ([]*Transaction, error)
		InvalidateCache(accountID string)
	}
}

// NewEconomicEngine creates a new economic engine
func NewEconomicEngine() *EconomicEngine {
	// Create network account
	networkAccountID := "network"

	// Initialize with default pricing rules
	pricingRules := make(map[ModelTier]map[QueryType]*PricingRule)

	// Basic tier pricing
	pricingRules[ModelTierBasic] = map[QueryType]*PricingRule{
		QueryTypeCompletion: {
			ModelTier:        ModelTierBasic,
			QueryType:        QueryTypeCompletion,
			BasePrice:        new(big.Int).Mul(big.NewInt(1), TokenUnit),
			TokenPerMS:       big.NewFloat(0.0001),
			TokenPerKB:       big.NewFloat(0.01),
			MinPrice:         new(big.Int).Mul(big.NewInt(1), TokenUnit),
			MaxPrice:         new(big.Int).Mul(big.NewInt(10), TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
		QueryTypeChat: {
			ModelTier:        ModelTierBasic,
			QueryType:        QueryTypeChat,
			BasePrice:        new(big.Int).Mul(big.NewInt(2), TokenUnit),
			TokenPerMS:       big.NewFloat(0.0002),
			TokenPerKB:       big.NewFloat(0.02),
			MinPrice:         new(big.Int).Mul(big.NewInt(2), TokenUnit),
			MaxPrice:         new(big.Int).Mul(big.NewInt(20), TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
		QueryTypeEmbedding: {
			ModelTier:        ModelTierBasic,
			QueryType:        QueryTypeEmbedding,
			BasePrice:        new(big.Int).Div(TokenUnit, big.NewInt(2)), // 0.5 tokens
			TokenPerMS:       big.NewFloat(0.00005),
			TokenPerKB:       big.NewFloat(0.005),
			MinPrice:         new(big.Int).Div(TokenUnit, big.NewInt(2)),
			MaxPrice:         new(big.Int).Mul(big.NewInt(5), TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
	}

	// Standard tier pricing
	pricingRules[ModelTierStandard] = map[QueryType]*PricingRule{
		QueryTypeCompletion: {
			ModelTier:        ModelTierStandard,
			QueryType:        QueryTypeCompletion,
			BasePrice:        new(big.Int).Mul(big.NewInt(5), TokenUnit),
			TokenPerMS:       big.NewFloat(0.0005),
			TokenPerKB:       big.NewFloat(0.05),
			MinPrice:         new(big.Int).Mul(big.NewInt(5), TokenUnit),
			MaxPrice:         new(big.Int).Mul(big.NewInt(50), TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
		QueryTypeChat: {
			ModelTier:        ModelTierStandard,
			QueryType:        QueryTypeChat,
			BasePrice:        new(big.Int).Mul(big.NewInt(10), TokenUnit),
			TokenPerMS:       big.NewFloat(0.001),
			TokenPerKB:       big.NewFloat(0.1),
			MinPrice:         new(big.Int).Mul(big.NewInt(10), TokenUnit),
			MaxPrice:         new(big.Int).Mul(big.NewInt(100), TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
		QueryTypeCodeGen: {
			ModelTier:        ModelTierStandard,
			QueryType:        QueryTypeCodeGen,
			BasePrice:        new(big.Int).Mul(big.NewInt(15), TokenUnit),
			TokenPerMS:       big.NewFloat(0.0015),
			TokenPerKB:       big.NewFloat(0.15),
			MinPrice:         new(big.Int).Mul(big.NewInt(15), TokenUnit),
			MaxPrice:         new(big.Int).Mul(big.NewInt(150), TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
	}

	// Premium tier pricing
	pricingRules[ModelTierPremium] = map[QueryType]*PricingRule{
		QueryTypeCompletion: {
			ModelTier:        ModelTierPremium,
			QueryType:        QueryTypeCompletion,
			BasePrice:        new(big.Int).Mul(big.NewInt(20), TokenUnit),
			TokenPerMS:       big.NewFloat(0.002),
			TokenPerKB:       big.NewFloat(0.2),
			MinPrice:         new(big.Int).Mul(big.NewInt(20), TokenUnit),
			MaxPrice:         new(big.Int).Mul(big.NewInt(200), TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
		QueryTypeChat: {
			ModelTier:        ModelTierPremium,
			QueryType:        QueryTypeChat,
			BasePrice:        new(big.Int).Mul(big.NewInt(30), TokenUnit),
			TokenPerMS:       big.NewFloat(0.003),
			TokenPerKB:       big.NewFloat(0.3),
			MinPrice:         new(big.Int).Mul(big.NewInt(30), TokenUnit),
			MaxPrice:         new(big.Int).Mul(big.NewInt(300), TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
		QueryTypeRAG: {
			ModelTier:        ModelTierPremium,
			QueryType:        QueryTypeRAG,
			BasePrice:        new(big.Int).Mul(big.NewInt(40), TokenUnit),
			TokenPerMS:       big.NewFloat(0.004),
			TokenPerKB:       big.NewFloat(0.4),
			MinPrice:         new(big.Int).Mul(big.NewInt(40), TokenUnit),
			MaxPrice:         new(big.Int).Mul(big.NewInt(400), TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
	}

	// Initialize reward config
	rewardConfig := RewardConfig{
		BaseReward:           new(big.Int).Mul(big.NewInt(1), TokenUnit),
		ReputationMultiplier: big.NewFloat(0.01),
		SpeedBonus:           big.NewFloat(0.5),
		QualityBonus:         big.NewFloat(0.5),
		StakeMultiplier:      big.NewFloat(0.0001),
	}

	// Initialize demand factors
	demandFactors := map[ModelTier]*big.Float{
		ModelTierBasic:      big.NewFloat(1.0),
		ModelTierStandard:   big.NewFloat(1.0),
		ModelTierPremium:    big.NewFloat(1.0),
		ModelTierEnterprise: big.NewFloat(1.0),
	}

	// Initialize query counters
	queryCounter := map[ModelTier]int64{
		ModelTierBasic:      0,
		ModelTierStandard:   0,
		ModelTierPremium:    0,
		ModelTierEnterprise: 0,
	}

	// Create the engine
	engine := &EconomicEngine{
		accounts:        make(map[string]*Account),
		nodeAccounts:    make(map[string]*NodeAccount),
		userAccounts:    make(map[string]*UserAccount),
		transactions:    make([]*Transaction, 0),
		pricingRules:    pricingRules,
		rewardConfig:    rewardConfig,
		networkAccount:  networkAccountID,
		totalSupply:     big.NewInt(0),
		demandFactors:   demandFactors,
		queryCounter:    queryCounter,
		lastPriceUpdate: time.Now(),
	}

	// Initialize user staking manager
	engine.userStakingManager = NewUserStakingManager(engine)

	// Create network account with initial supply
	initialSupply := new(big.Int).Mul(big.NewInt(1000000000), TokenUnit) // 1 billion tokens
	engine.createAccount(networkAccountID, "network")
	engine.mintTokens(networkAccountID, initialSupply, "Initial token supply")

	// Create staking pool account
	engine.createAccount("staking_pool", "staking_pool")

	return engine
}

// SetConsensusBridge sets the consensus bridge for the economic engine
func (e *EconomicEngine) SetConsensusBridge(bridge interface {
	ProcessEconomicTransaction(ctx context.Context, tx *Transaction) (*types.Transaction, error)
	IsEconomicTransactionFinalized(txID string) bool
}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.consensusBridge = bridge
}

// GetUserStakingManager returns the user staking manager
func (e *EconomicEngine) GetUserStakingManager() *UserStakingManager {
	return e.userStakingManager
}

// InitRewardDistributor creates a reward distributor with the given config and stores
// it on the engine (does NOT start it).
func (e *EconomicEngine) InitRewardDistributor(cfg RewardDistributorConfig) *RewardDistributor {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	rd := NewRewardDistributor(e, cfg)
	e.rewardDistributor = rd
	return rd
}

// GetRewardDistributor returns the reward distributor if it has been initialised.
func (e *EconomicEngine) GetRewardDistributor() *RewardDistributor {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.rewardDistributor
}

// StartRewardDistributor starts the distributor’s background loop if it exists.
func (e *EconomicEngine) StartRewardDistributor() {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	if e.rewardDistributor != nil {
		e.rewardDistributor.Start()
	}
}

// StopRewardDistributor stops the background loop if running.
func (e *EconomicEngine) StopRewardDistributor() {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	if e.rewardDistributor != nil {
		e.rewardDistributor.Stop()
	}
}

// StakeToNode allows a user to stake LORE to a node
func (e *EconomicEngine) StakeToNode(account, nodeID string, amount *big.Int) (*UserStake, error) {
	return e.userStakingManager.StakeToNode(account, nodeID, amount)
}

// RequestStakeWithdrawal initiates a stake withdrawal request
func (e *EconomicEngine) RequestStakeWithdrawal(account, nodeID string, amount *big.Int) error {
	return e.userStakingManager.RequestWithdrawal(account, nodeID, amount)
}

// ExecuteStakeWithdrawal completes a stake withdrawal
func (e *EconomicEngine) ExecuteStakeWithdrawal(account, nodeID string) (*Transaction, error) {
	return e.userStakingManager.ExecuteWithdrawal(account, nodeID)
}

// RecordQueryResult records a query result for performance tracking and potential slashing
func (e *EconomicEngine) RecordQueryResult(nodeID, queryID string, success bool, responseTimeMs int64) {
	e.userStakingManager.RecordQueryResult(nodeID, queryID, success, responseTimeMs)
}

// SlashNode slashes stakes for a node due to poor performance
func (e *EconomicEngine) SlashNode(nodeID, reason string, severity float64, queryID string) (*SlashingEvent, error) {
	return e.userStakingManager.SlashNode(nodeID, reason, severity, queryID)
}

// GetNodeStakeInfo returns staking information for a node
func (e *EconomicEngine) GetNodeStakeInfo(nodeID string) *NodeStakeInfo {
	return e.userStakingManager.GetNodeStakeInfo(nodeID)
}

// GetUserStakes returns all stakes for a user
func (e *EconomicEngine) GetUserStakes(account string) []*UserStake {
	return e.userStakingManager.GetUserStakes(account)
}

// GetAllNodeStakes returns staking info for all nodes
func (e *EconomicEngine) GetAllNodeStakes() map[string]*NodeStakeInfo {
	return e.userStakingManager.GetAllNodeStakes()
}

// SetStorage sets the storage backend for the economic engine
func (e *EconomicEngine) SetStorage(storage interface {
	StoreAccount(account *Account) error
	LoadAccount(accountID string) (*Account, error)
	StoreNodeAccount(nodeAccount *NodeAccount) error
	LoadNodeAccount(nodeID string) (*NodeAccount, error)
	StoreUserAccount(userAccount *UserAccount) error
	LoadUserAccount(userID string) (*UserAccount, error)
	StoreTransaction(tx *Transaction) error
	LoadTransaction(txID string) (*Transaction, error)
	GetTransactionsByAccount(accountID string, limit, offset int) ([]*Transaction, error)
	InvalidateCache(accountID string)
}) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.storage = storage
}

// createAccount creates a new account
func (e *EconomicEngine) createAccount(id, address string) *Account {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	account := &Account{
		ID:        id,
		Address:   address,
		Balance:   big.NewInt(0),
		Stake:     big.NewInt(0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	e.accounts[id] = account
	return account
}

// CreateUserAccount creates a new user account
func (e *EconomicEngine) CreateUserAccount(id, address string) (*UserAccount, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if _, exists := e.accounts[id]; exists {
		return nil, fmt.Errorf("account with ID %s already exists", id)
	}

	account := &Account{
		ID:        id,
		Address:   address,
		Balance:   big.NewInt(0),
		Stake:     big.NewInt(0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	userAccount := &UserAccount{
		Account:          account,
		TotalSpent:       big.NewInt(0),
		QueriesSubmitted: 0,
		UsageByModel:     make(map[string]int64),
		UsageByType:      make(map[QueryType]int),
	}

	e.accounts[id] = account
	e.userAccounts[id] = userAccount

	// Persist to storage
	if e.storage != nil {
		go func() {
			if err := e.storage.StoreUserAccount(userAccount); err != nil {
				log.Printf("Failed to persist user account %s: %v", id, err)
			}
		}()
	}

	return userAccount, nil
}

// CreateNodeAccount creates a new node account using libp2p peer ID
func (e *EconomicEngine) CreateNodeAccount(peerID string) (*NodeAccount, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if _, exists := e.accounts[peerID]; exists {
		return nil, fmt.Errorf("account with peer ID %s already exists", peerID)
	}

	// Validate that this is a proper peer ID (skip for test peer IDs)
	var address string
	if len(peerID) >= 10 && peerID[:10] == "test-peer-" {
		// For testing, create a simple address
		hash := sha256.Sum256([]byte(peerID))
		address = fmt.Sprintf("0x%x", hash)[:42]
	} else {
		parsedPeerID, err := ValidatePeerID(peerID)
		if err != nil {
			return nil, fmt.Errorf("invalid peer ID %s: %v", peerID, err)
		}
		// Derive a deterministic address from the peer ID
		address = DeriveAddressFromPeerID(parsedPeerID)
	}

	account := &Account{
		ID:        peerID, // Use peer ID as the primary identifier
		Address:   address, // Derived from peer ID
		Balance:   big.NewInt(0),
		Stake:     big.NewInt(0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	nodeAccount := &NodeAccount{
		Account:          account,
		Reputation:       InitialReputationScore,
		TotalEarned:      big.NewInt(0),
		QueriesProcessed: 0,
		SuccessRate:      1.0,
		AvgResponseTime:  0,
		LastActive:       time.Now(),
		Capabilities:     make(map[string]bool),
		Performance:      make(map[string]float64),
	}

	e.accounts[peerID] = account
	e.nodeAccounts[peerID] = nodeAccount

	// Persist to storage
	if e.storage != nil {
		go func() {
			if err := e.storage.StoreNodeAccount(nodeAccount); err != nil {
				log.Printf("Failed to persist node account %s: %v", peerID, err)
			}
		}()
	}

	return nodeAccount, nil
}

// GetAccount retrieves an account by ID
func (e *EconomicEngine) GetAccount(id string) (*Account, error) {
	e.mutex.RLock()
	// Try memory cache first
	account, exists := e.accounts[id]
	if exists {
		e.mutex.RUnlock()
		return account, nil
	}
	e.mutex.RUnlock()

	// Load from persistent storage if available
	if e.storage != nil {
		storedAccount, err := e.storage.LoadAccount(id)
		if err == nil {
			// Cache in memory
			e.mutex.Lock()
			e.accounts[id] = storedAccount
			e.mutex.Unlock()
			return storedAccount, nil
		}
	}

	return nil, ErrAccountNotFound
}

// GetNodeAccount retrieves a node account by ID
func (e *EconomicEngine) GetNodeAccount(id string) (*NodeAccount, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	nodeAccount, exists := e.nodeAccounts[id]
	if !exists {
		return nil, ErrAccountNotFound
	}

	return nodeAccount, nil
}

// GetUserAccount retrieves a user account by ID
func (e *EconomicEngine) GetUserAccount(id string) (*UserAccount, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	userAccount, exists := e.userAccounts[id]
	if !exists {
		return nil, ErrAccountNotFound
	}

	return userAccount, nil
}

// GetBalance retrieves the balance of an account
func (e *EconomicEngine) GetBalance(id string) (*big.Int, error) {
	account, err := e.GetAccount(id)
	if err != nil {
		return nil, err
	}

	return new(big.Int).Set(account.Balance), nil
}

// GetStake retrieves the stake of an account
func (e *EconomicEngine) GetStake(id string) (*big.Int, error) {
	account, err := e.GetAccount(id)
	if err != nil {
		return nil, err
	}

	return new(big.Int).Set(account.Stake), nil
}

// Transfer transfers tokens between accounts
func (e *EconomicEngine) Transfer(fromID, toID string, amount *big.Int, description string) (*Transaction, error) {
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, ErrInvalidAmount
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()

	fromAccount, exists := e.accounts[fromID]
	if !exists {
		return nil, ErrAccountNotFound
	}

	toAccount, exists := e.accounts[toID]
	if !exists {
		return nil, ErrAccountNotFound
	}

	// Check if the sender has sufficient balance
	if fromAccount.Balance.Cmp(amount) < 0 {
		return nil, ErrInsufficientBalance
	}

	// Perform the transfer
	fromAccount.Balance = new(big.Int).Sub(fromAccount.Balance, amount)
	toAccount.Balance = new(big.Int).Add(toAccount.Balance, amount)

	// Update timestamps
	now := time.Now()
	fromAccount.UpdatedAt = now
	toAccount.UpdatedAt = now

	// Create a transaction record
	tx := &Transaction{
		ID:          uuid.New().String(),
		Type:        TransactionTypeTransfer,
		FromID:      fromID,
		ToID:        toID,
		Amount:      new(big.Int).Set(amount),
		Fee:         big.NewInt(0),
		Description: description,
		Metadata:    make(map[string]any),
		Timestamp:   now,
		Status:      "completed",
	}

	e.transactions = append(e.transactions, tx)

	// Persist transaction to storage
	if e.storage != nil {
		go func() {
			if err := e.storage.StoreTransaction(tx); err != nil {
				log.Printf("Failed to persist transaction %s: %v", tx.ID, err)
			}
		}()
	}

	// Persist account updates to storage
	if e.storage != nil {
		go func() {
			if err := e.storage.StoreAccount(fromAccount); err != nil {
				log.Printf("Failed to persist account %s: %v", fromAccount.ID, err)
			}
			if err := e.storage.StoreAccount(toAccount); err != nil {
				log.Printf("Failed to persist account %s: %v", toAccount.ID, err)
			}
		}()
	}

	// Process through consensus if bridge is available
	if e.consensusBridge != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			_, err := e.consensusBridge.ProcessEconomicTransaction(ctx, tx)
			if err != nil {
				log.Printf("Failed to process transfer transaction %s through consensus: %v", tx.ID, err)
				// Update transaction status to indicate consensus failure
				e.mutex.Lock()
				tx.Status = "consensus_failed"
				e.mutex.Unlock()

				// Update persisted transaction status
				if e.storage != nil {
					e.storage.StoreTransaction(tx)
				}
			} else {
				log.Printf("Transfer transaction %s successfully added to consensus", tx.ID)
			}
		}()
	}

	return tx, nil
}

// Stake stakes tokens for a node
func (e *EconomicEngine) Stake(nodeID string, amount *big.Int) (*Transaction, error) {
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, ErrInvalidAmount
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()

	nodeAccount, exists := e.nodeAccounts[nodeID]
	if !exists {
		return nil, ErrAccountNotFound
	}

	// Check if the node has sufficient balance
	if nodeAccount.Balance.Cmp(amount) < 0 {
		return nil, ErrInsufficientBalance
	}

	// Perform the staking
	nodeAccount.Balance = new(big.Int).Sub(nodeAccount.Balance, amount)
	nodeAccount.Stake = new(big.Int).Add(nodeAccount.Stake, amount)

	// Update timestamp
	now := time.Now()
	nodeAccount.UpdatedAt = now

	// Create a transaction record
	tx := &Transaction{
		ID:          uuid.New().String(),
		Type:        TransactionTypeStake,
		FromID:      nodeID,
		ToID:        nodeID,
		Amount:      new(big.Int).Set(amount),
		Fee:         big.NewInt(0),
		Description: "Stake tokens",
		Metadata:    make(map[string]any),
		Timestamp:   now,
		Status:      "completed",
	}

	e.transactions = append(e.transactions, tx)

	return tx, nil
}

// Unstake unstakes tokens for a node
func (e *EconomicEngine) Unstake(nodeID string, amount *big.Int) (*Transaction, error) {
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, ErrInvalidAmount
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()

	nodeAccount, exists := e.nodeAccounts[nodeID]
	if !exists {
		return nil, ErrAccountNotFound
	}

	// Check if the node has sufficient stake
	if nodeAccount.Stake.Cmp(amount) < 0 {
		return nil, ErrInsufficientStake
	}

	// Ensure minimum stake is maintained
	remainingStake := new(big.Int).Sub(nodeAccount.Stake, amount)
	if remainingStake.Cmp(big.NewInt(0)) > 0 && remainingStake.Cmp(MinimumStake) < 0 {
		return nil, fmt.Errorf("unstaking would leave less than minimum required stake")
	}

	// Perform the unstaking
	nodeAccount.Stake = new(big.Int).Sub(nodeAccount.Stake, amount)
	nodeAccount.Balance = new(big.Int).Add(nodeAccount.Balance, amount)

	// Update timestamp
	now := time.Now()
	nodeAccount.UpdatedAt = now

	// Create a transaction record
	tx := &Transaction{
		ID:          uuid.New().String(),
		Type:        TransactionTypeUnstake,
		FromID:      nodeID,
		ToID:        nodeID,
		Amount:      new(big.Int).Set(amount),
		Fee:         big.NewInt(0),
		Description: "Unstake tokens",
		Metadata:    make(map[string]any),
		Timestamp:   now,
		Status:      "completed",
	}

	e.transactions = append(e.transactions, tx)

	return tx, nil
}

// CalculateQueryPrice calculates the price for a query
func (e *EconomicEngine) CalculateQueryPrice(modelTier ModelTier, queryType QueryType, inputSize int64) (*big.Int, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// Get pricing rules for the model tier and query type
	tierRules, exists := e.pricingRules[modelTier]
	if !exists {
		return nil, fmt.Errorf("no pricing rules for model tier: %s", modelTier)
	}

	rule, exists := tierRules[queryType]
	if !exists {
		return nil, fmt.Errorf("no pricing rules for query type: %s", queryType)
	}

	// Calculate base price
	price := new(big.Int).Set(rule.BasePrice)

	// Add size-based component
	if inputSize > 0 {
		// Convert KB to tokens
		inputKB := float64(inputSize) / 1024.0
		tokenPerKB, _ := rule.TokenPerKB.Float64()
		sizeComponent := inputKB * tokenPerKB

		// Convert to big.Int
		sizeTokens := new(big.Float).Mul(big.NewFloat(sizeComponent), new(big.Float).SetInt(TokenUnit))
		var sizeInt big.Int
		sizeTokens.Int(&sizeInt)

		// Add to price
		price = new(big.Int).Add(price, &sizeInt)
	}

	// Apply demand multiplier
	demandFactor, exists := e.demandFactors[modelTier]
	if exists {
		demandFloat := new(big.Float).SetInt(price)
		demandFloat = new(big.Float).Mul(demandFloat, demandFactor)

		var demandPrice big.Int
		demandFloat.Int(&demandPrice)
		price = &demandPrice
	}

	// Ensure price is within bounds
	if price.Cmp(rule.MinPrice) < 0 {
		price = new(big.Int).Set(rule.MinPrice)
	} else if price.Cmp(rule.MaxPrice) > 0 {
		price = new(big.Int).Set(rule.MaxPrice)
	}

	return price, nil
}

// ProcessQueryPayment processes a payment for a query
func (e *EconomicEngine) ProcessQueryPayment(
	ctx context.Context,
	userID string,
	modelID string,
	modelTier ModelTier,
	queryType QueryType,
	inputSize int64,
) (*Transaction, error) {
	// Calculate the price
	price, err := e.CalculateQueryPrice(modelTier, queryType, inputSize)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate price: %w", err)
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Get user account
	userAccount, exists := e.userAccounts[userID]
	if !exists {
		return nil, ErrAccountNotFound
	}

	// Check if user has sufficient balance
	if userAccount.Balance.Cmp(price) < 0 {
		return nil, ErrInsufficientBalance
	}

	// Calculate network fee
	feePercent := NetworkFeePercent
	feeFloat := new(big.Float).Mul(
		new(big.Float).SetInt(price),
		new(big.Float).Quo(
			big.NewFloat(float64(feePercent)),
			big.NewFloat(100.0),
		),
	)

	var fee big.Int
	feeFloat.Int(&fee)

	// Calculate amount after fee
	amountAfterFee := new(big.Int).Sub(price, &fee)

	// Deduct from user's balance
	userAccount.Balance = new(big.Int).Sub(userAccount.Balance, price)
	userAccount.TotalSpent = new(big.Int).Add(userAccount.TotalSpent, price)
	userAccount.QueriesSubmitted++

	// Update usage statistics
	userAccount.UsageByType[queryType]++
	if _, exists := userAccount.UsageByModel[modelID]; !exists {
		userAccount.UsageByModel[modelID] = 0
	}
	userAccount.UsageByModel[modelID]++

	// Add fee to network account
	networkAccount, _ := e.accounts[e.networkAccount]
	networkAccount.Balance = new(big.Int).Add(networkAccount.Balance, &fee)

	// Update timestamps
	now := time.Now()
	userAccount.UpdatedAt = now
	networkAccount.UpdatedAt = now

	// Create a transaction record
	tx := &Transaction{
		ID:          uuid.New().String(),
		Type:        TransactionTypeQueryPayment,
		FromID:      userID,
		ToID:        e.networkAccount, // Initially to network account
		Amount:      new(big.Int).Set(price),
		Fee:         new(big.Int).Set(&fee),
		Description: fmt.Sprintf("Payment for %s query using %s model", queryType, modelTier),
		Metadata: map[string]any{
			"model_id":         modelID,
			"model_tier":       string(modelTier),
			"query_type":       string(queryType),
			"input_size":       inputSize,
			"amount_after_fee": amountAfterFee.String(),
		},
		Timestamp: now,
		Status:    "pending_reward", // Will be completed after reward distribution
	}

	e.transactions = append(e.transactions, tx)

	// Update query counter for demand-based pricing
	e.queryCounter[modelTier]++

	// Update demand factors if needed
	if time.Since(e.lastPriceUpdate) > 1*time.Hour {
		e.updateDemandFactors()
	}

	return tx, nil
}

// DistributeQueryReward distributes the reward for a processed query
func (e *EconomicEngine) DistributeQueryReward(
	ctx context.Context,
	txID string,
	nodeID string,
	responseTime int64,
	success bool,
	qualityScore float64,
) (*Transaction, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Find the payment transaction
	var paymentTx *Transaction
	for _, tx := range e.transactions {
		if tx.ID == txID && tx.Type == TransactionTypeQueryPayment && tx.Status == "pending_reward" {
			paymentTx = tx
			break
		}
	}

	if paymentTx == nil {
		return nil, fmt.Errorf("payment transaction not found or not in pending state")
	}

	// Get node account
	nodeAccount, exists := e.nodeAccounts[nodeID]
	if !exists {
		return nil, ErrAccountNotFound
	}

	// Get network account
	networkAccount, _ := e.accounts[e.networkAccount]

	// Parse amount after fee from metadata
	amountAfterFeeStr, ok := paymentTx.Metadata["amount_after_fee"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid payment transaction metadata")
	}

	amountAfterFee, ok := new(big.Int).SetString(amountAfterFeeStr, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount format in metadata")
	}

	// Calculate reward with multipliers
	reward := new(big.Int).Set(amountAfterFee)

	// Apply reputation multiplier
	if nodeAccount.Reputation > 0 {
		repMultiplier := 1.0 + (nodeAccount.Reputation/100.0)*0.5 // 50% bonus at max reputation
		repFloat := new(big.Float).Mul(
			new(big.Float).SetInt(reward),
			big.NewFloat(repMultiplier),
		)

		var repReward big.Int
		repFloat.Int(&repReward)
		reward = &repReward
	}

	// Apply speed bonus for fast responses
	if responseTime > 0 {
		// Assume 1000ms is the baseline, faster responses get bonus
		speedFactor := 1000.0 / float64(responseTime)
		if speedFactor > 2.0 {
			speedFactor = 2.0 // Cap at 2x bonus
		}

		speedFloat := new(big.Float).Mul(
			new(big.Float).SetInt(reward),
			big.NewFloat(speedFactor),
		)

		var speedReward big.Int
		speedFloat.Int(&speedReward)
		reward = &speedReward
	}

	// Apply quality multiplier
	if qualityScore > 0 {
		qualityMultiplier := qualityScore // 0.0 to 1.0
		qualityFloat := new(big.Float).Mul(
			new(big.Float).SetInt(reward),
			big.NewFloat(qualityMultiplier),
		)

		var qualityReward big.Int
		qualityFloat.Int(&qualityReward)
		reward = &qualityReward
	}

	// Apply stake multiplier (bonus for higher stake)
	if nodeAccount.Stake.Cmp(big.NewInt(0)) > 0 {
		// Calculate stake bonus (0.01% per staked token)
		stakeMultiplier, _ := new(big.Float).Mul(
			new(big.Float).SetInt(nodeAccount.Stake),
			big.NewFloat(0.0001),
		).Float64()

		if stakeMultiplier > 2.0 {
			stakeMultiplier = 2.0 // Cap at 2x bonus
		}

		stakeFloat := new(big.Float).Mul(
			new(big.Float).SetInt(reward),
			big.NewFloat(1.0+stakeMultiplier),
		)

		var stakeReward big.Int
		stakeFloat.Int(&stakeReward)
		reward = &stakeReward
	}

	// Transfer reward from network to node
	networkAccount.Balance = new(big.Int).Sub(networkAccount.Balance, reward)
	nodeAccount.Balance = new(big.Int).Add(nodeAccount.Balance, reward)
	nodeAccount.TotalEarned = new(big.Int).Add(nodeAccount.TotalEarned, reward)

	// Update node metrics
	nodeAccount.QueriesProcessed++

	// Update success rate (weighted average)
	successValue := 0.0
	if success {
		successValue = 1.0
	}
	nodeAccount.SuccessRate = (nodeAccount.SuccessRate*0.95 + successValue*0.05)

	// Update average response time (weighted average)
	if responseTime > 0 {
		if nodeAccount.AvgResponseTime == 0 {
			nodeAccount.AvgResponseTime = responseTime
		} else {
			nodeAccount.AvgResponseTime = int64(float64(nodeAccount.AvgResponseTime)*0.95 + float64(responseTime)*0.05)
		}
	}

	// Update reputation based on quality and success
	reputationDelta := 0.0
	if success {
		reputationDelta += 1.0
		reputationDelta += qualityScore * 2.0 // Quality has higher impact
	} else {
		reputationDelta -= 2.0
	}

	// Apply reputation change with bounds
	nodeAccount.Reputation += reputationDelta
	if nodeAccount.Reputation < 0 {
		nodeAccount.Reputation = 0
	} else if nodeAccount.Reputation > MaxReputationScore {
		nodeAccount.Reputation = MaxReputationScore
	}

	// Update timestamps
	now := time.Now()
	nodeAccount.UpdatedAt = now
	nodeAccount.LastActive = now
	networkAccount.UpdatedAt = now

	// Update payment transaction status
	paymentTx.Status = "completed"

	// Create a reward transaction record
	tx := &Transaction{
		ID:          uuid.New().String(),
		Type:        TransactionTypeReward,
		FromID:      e.networkAccount,
		ToID:        nodeID,
		Amount:      new(big.Int).Set(reward),
		Fee:         big.NewInt(0),
		Description: "Reward for query processing",
		Metadata: map[string]any{
			"payment_tx_id": paymentTx.ID,
			"response_time": responseTime,
			"success":       success,
			"quality_score": qualityScore,
			"reputation":    nodeAccount.Reputation,
		},
		Timestamp: now,
		Status:    "completed",
	}

	e.transactions = append(e.transactions, tx)

	return tx, nil
}

// mintTokens creates new tokens and adds them to an account
func (e *EconomicEngine) mintTokens(accountID string, amount *big.Int, reason string) (*Transaction, error) {
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, ErrInvalidAmount
	}

	account, exists := e.accounts[accountID]
	if !exists {
		return nil, ErrAccountNotFound
	}

	// Add tokens to the account
	account.Balance = new(big.Int).Add(account.Balance, amount)

	// Update total supply
	e.totalSupply = new(big.Int).Add(e.totalSupply, amount)

	// Update timestamp
	now := time.Now()
	account.UpdatedAt = now

	// Create a transaction record
	tx := &Transaction{
		ID:          uuid.New().String(),
		Type:        TransactionTypeMint,
		FromID:      "",
		ToID:        accountID,
		Amount:      new(big.Int).Set(amount),
		Fee:         big.NewInt(0),
		Description: reason,
		Metadata:    make(map[string]any),
		Timestamp:   now,
		Status:      "completed",
	}

	e.transactions = append(e.transactions, tx)

	return tx, nil
}

// MintTokens creates new tokens and adds them to an account (public version)
func (e *EconomicEngine) MintTokens(accountID string, amount *big.Int, reason string) (*Transaction, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	return e.mintTokens(accountID, amount, reason)
}

// GetTotalSupply returns the total token supply
func (e *EconomicEngine) GetTotalSupply() *big.Int {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	return new(big.Int).Set(e.totalSupply)
}

// GetTransactions returns transactions for an account
func (e *EconomicEngine) GetTransactions(accountID string, limit, offset int) ([]*Transaction, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	if _, exists := e.accounts[accountID]; !exists {
		return nil, ErrAccountNotFound
	}

	var accountTxs []*Transaction
	for _, tx := range e.transactions {
		if tx.FromID == accountID || tx.ToID == accountID {
			accountTxs = append(accountTxs, tx)
		}
	}

	// Sort by timestamp (newest first)
	// In a real implementation, this would be done with a database query

	// Apply pagination
	if offset >= len(accountTxs) {
		return []*Transaction{}, nil
	}

	end := offset + limit
	if end > len(accountTxs) {
		end = len(accountTxs)
	}

	return accountTxs[offset:end], nil
}

// GetTransaction returns a transaction by ID
func (e *EconomicEngine) GetTransaction(txID string) (*Transaction, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	for _, tx := range e.transactions {
		if tx.ID == txID {
			return tx, nil
		}
	}

	return nil, fmt.Errorf("transaction not found")
}

// GetNetworkStats returns statistics about the economic system
func (e *EconomicEngine) GetNetworkStats() map[string]interface{} {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// Count active nodes (staked)
	activeNodes := 0
	totalStaked := big.NewInt(0)
	totalReputation := 0.0

	for _, node := range e.nodeAccounts {
		if node.Stake.Cmp(MinimumStake) >= 0 {
			activeNodes++
			totalStaked = new(big.Int).Add(totalStaked, node.Stake)
			totalReputation += node.Reputation
		}
	}

	// Calculate average reputation
	avgReputation := 0.0
	if activeNodes > 0 {
		avgReputation = totalReputation / float64(activeNodes)
	}

	// Count transactions by type
	txCounts := make(map[TransactionType]int)
	for _, tx := range e.transactions {
		txCounts[tx.Type]++
	}

	// Calculate total fees collected
	totalFees := big.NewInt(0)
	for _, tx := range e.transactions {
		totalFees = new(big.Int).Add(totalFees, tx.Fee)
	}

	return map[string]interface{}{
		"total_supply":       e.totalSupply.String(),
		"active_nodes":       activeNodes,
		"total_staked":       totalStaked.String(),
		"avg_reputation":     avgReputation,
		"total_accounts":     len(e.accounts),
		"total_users":        len(e.userAccounts),
		"total_nodes":        len(e.nodeAccounts),
		"total_transactions": len(e.transactions),
		"transaction_counts": txCounts,
		"total_fees":         totalFees.String(),
		"demand_factors":     e.demandFactors,
	}
}

// UpdatePricingRule updates a pricing rule
func (e *EconomicEngine) UpdatePricingRule(rule *PricingRule) error {
	if rule == nil {
		return fmt.Errorf("pricing rule cannot be nil")
	}

	e.mutex.Lock()
	defer e.mutex.Unlock()

	tierRules, exists := e.pricingRules[rule.ModelTier]
	if !exists {
		tierRules = make(map[QueryType]*PricingRule)
		e.pricingRules[rule.ModelTier] = tierRules
	}

	tierRules[rule.QueryType] = rule

	return nil
}

// GetPricingRule returns a pricing rule
func (e *EconomicEngine) GetPricingRule(modelTier ModelTier, queryType QueryType) (*PricingRule, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	tierRules, exists := e.pricingRules[modelTier]
	if !exists {
		return nil, fmt.Errorf("no pricing rules for model tier: %s", modelTier)
	}

	rule, exists := tierRules[queryType]
	if !exists {
		return nil, fmt.Errorf("no pricing rules for query type: %s", queryType)
	}

	return rule, nil
}

// updateDemandFactors updates pricing based on demand
func (e *EconomicEngine) updateDemandFactors() {
	// Simple algorithm: if queries > threshold, increase price
	for tier, count := range e.queryCounter {
		// Reset counter
		e.queryCounter[tier] = 0

		// Update demand factor based on query volume
		factor := 1.0

		if count > 1000 {
			factor = 1.5 // High demand
		} else if count > 500 {
			factor = 1.2 // Medium demand
		} else if count < 100 {
			factor = 0.9 // Low demand
		}

		e.demandFactors[tier] = big.NewFloat(factor)
	}

	e.lastPriceUpdate = time.Now()
}

// DecayReputations periodically decays node reputations
func (e *EconomicEngine) DecayReputations() {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for _, node := range e.nodeAccounts {
		// Decay reputation slightly each day for inactive nodes
		if time.Since(node.LastActive) > 24*time.Hour {
			decayAmount := node.Reputation * ReputationDecayFactor
			node.Reputation -= decayAmount
			if node.Reputation < 0 {
				node.Reputation = 0
			}
		}
	}
}

// RunMaintenanceTasks runs periodic maintenance tasks
func (e *EconomicEngine) RunMaintenanceTasks(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.DecayReputations()
			e.updateDemandFactors()
		case <-ctx.Done():
			return
		}
	}
}
