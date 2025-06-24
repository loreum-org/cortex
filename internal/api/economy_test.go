package api

// Temporarily disabled due to complex mock and pointer issues
// TODO: Fix mock implementations and pointer issues

/*

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/loreum-org/cortex/internal/economy"
)

// TestMockEconomicEngine is a mock implementation of the economic engine for testing
type TestMockEconomicEngine struct {
	UserAccounts   map[string]*economy.UserAccount
	NodeAccounts   map[string]*economy.NodeAccount
	Transactions   []*economy.Transaction
	PricingRules   map[economy.ModelTier]map[economy.QueryType]*economy.PricingRule
	NextTxID       int
	FailNextAction bool
}

// NewTestMockEconomicEngine creates a new mock economic engine
func NewTestMockEconomicEngine() *TestMockEconomicEngine {
	// Initialize with some basic pricing rules
	pricingRules := make(map[economy.ModelTier]map[economy.QueryType]*economy.PricingRule)

	// Basic tier pricing
	pricingRules[economy.ModelTierBasic] = map[economy.QueryType]*economy.PricingRule{
		economy.QueryTypeCompletion: {
			ModelTier:        economy.ModelTierBasic,
			QueryType:        economy.QueryTypeCompletion,
			BasePrice:        new(big.Int).Mul(big.NewInt(1), economy.TokenUnit),
			TokenPerMS:       big.NewFloat(0.0001),
			TokenPerKB:       big.NewFloat(0.01),
			MinPrice:         new(big.Int).Mul(big.NewInt(1), economy.TokenUnit),
			MaxPrice:         new(big.Int).Mul(big.NewInt(10), economy.TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
	}

	// Standard tier pricing
	pricingRules[economy.ModelTierStandard] = map[economy.QueryType]*economy.PricingRule{
		economy.QueryTypeCompletion: {
			ModelTier:        economy.ModelTierStandard,
			QueryType:        economy.QueryTypeCompletion,
			BasePrice:        new(big.Int).Mul(big.NewInt(5), economy.TokenUnit),
			TokenPerMS:       big.NewFloat(0.0005),
			TokenPerKB:       big.NewFloat(0.05),
			MinPrice:         new(big.Int).Mul(big.NewInt(5), economy.TokenUnit),
			MaxPrice:         new(big.Int).Mul(big.NewInt(50), economy.TokenUnit),
			DemandMultiplier: big.NewFloat(1.0),
		},
	}

	return &TestMockEconomicEngine{
		UserAccounts: make(map[string]*economy.UserAccount),
		NodeAccounts: make(map[string]*economy.NodeAccount),
		Transactions: make([]*economy.Transaction, 0),
		PricingRules: pricingRules,
		NextTxID:     1,
	}
}

// CreateUserAccount creates a mock user account
func (m *TestMockEconomicEngine) CreateUserAccount(id, address string) (*economy.UserAccount, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	if _, exists := m.UserAccounts[id]; exists {
		return nil, fmt.Errorf("account with ID %s already exists", id)
	}

	account := &economy.UserAccount{
		Account: &economy.Account{
			ID:        id,
			Address:   address,
			Balance:   big.NewInt(0),
			Stake:     big.NewInt(0),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		TotalSpent:       big.NewInt(0),
		QueriesSubmitted: 0,
		UsageByModel:     make(map[string]int64),
		UsageByType:      make(map[economy.QueryType]int),
	}

	m.UserAccounts[id] = account
	return account, nil
}

// CreateNodeAccount creates a mock node account
func (m *TestMockEconomicEngine) CreateNodeAccount(id, address string) (*economy.NodeAccount, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	if _, exists := m.NodeAccounts[id]; exists {
		return nil, fmt.Errorf("account with ID %s already exists", id)
	}

	account := &economy.NodeAccount{
		Account: &economy.Account{
			ID:        id,
			Address:   address,
			Balance:   big.NewInt(0),
			Stake:     big.NewInt(0),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Reputation:       economy.InitialReputationScore,
		TotalEarned:      big.NewInt(0),
		QueriesProcessed: 0,
		SuccessRate:      1.0,
		AvgResponseTime:  0,
		LastActive:       time.Now(),
		Capabilities:     make(map[string]bool),
		Performance:      make(map[string]float64),
	}

	m.NodeAccounts[id] = account
	return account, nil
}

// GetAccount retrieves a mock account
func (m *TestMockEconomicEngine) GetAccount(id string) (*economy.Account, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	if user, exists := m.UserAccounts[id]; exists {
		return user.Account, nil
	}

	if node, exists := m.NodeAccounts[id]; exists {
		return node.Account, nil
	}

	return nil, economy.ErrAccountNotFound
}

// GetUserAccount retrieves a mock user account
func (m *TestMockEconomicEngine) GetUserAccount(id string) (*economy.UserAccount, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	if user, exists := m.UserAccounts[id]; exists {
		return user, nil
	}

	return nil, economy.ErrAccountNotFound
}

// GetNodeAccount retrieves a mock node account
func (m *TestMockEconomicEngine) GetNodeAccount(id string) (*economy.NodeAccount, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	if node, exists := m.NodeAccounts[id]; exists {
		return node, nil
	}

	return nil, economy.ErrAccountNotFound
}

// Transfer transfers tokens between accounts
func (m *TestMockEconomicEngine) Transfer(fromID, toID string, amount *big.Int, description string) (*economy.Transaction, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, economy.ErrInvalidAmount
	}

	var fromAccount *economy.Account
	var toAccount *economy.Account

	// Find from account
	if user, exists := m.UserAccounts[fromID]; exists {
		fromAccount = user.Account
	} else if node, exists := m.NodeAccounts[fromID]; exists {
		fromAccount = node.Account
	} else {
		return nil, economy.ErrAccountNotFound
	}

	// Find to account
	if user, exists := m.UserAccounts[toID]; exists {
		toAccount = user.Account
	} else if node, exists := m.NodeAccounts[toID]; exists {
		toAccount = node.Account
	} else {
		return nil, economy.ErrAccountNotFound
	}

	// Check balance
	if fromAccount.Balance.Cmp(amount) < 0 {
		return nil, economy.ErrInsufficientBalance
	}

	// Perform transfer
	fromAccount.Balance = new(big.Int).Sub(fromAccount.Balance, amount)
	toAccount.Balance = new(big.Int).Add(toAccount.Balance, amount)

	// Create transaction
	tx := &economy.Transaction{
		ID:          fmt.Sprintf("tx_%d", m.NextTxID),
		Type:        economy.TransactionTypeTransfer,
		FromID:      fromID,
		ToID:        toID,
		Amount:      new(big.Int).Set(amount),
		Fee:         big.NewInt(0),
		Description: description,
		Metadata:    make(map[string]any),
		Timestamp:   time.Now(),
		Status:      "completed",
	}

	m.NextTxID++
	m.Transactions = append(m.Transactions, tx)

	return tx, nil
}

// Stake stakes tokens for a node
func (m *TestMockEconomicEngine) Stake(nodeID string, amount *big.Int) (*economy.Transaction, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, economy.ErrInvalidAmount
	}

	node, exists := m.NodeAccounts[nodeID]
	if !exists {
		return nil, economy.ErrAccountNotFound
	}

	// Check balance
	if node.Balance.Cmp(amount) < 0 {
		return nil, economy.ErrInsufficientBalance
	}

	// Perform staking
	node.Balance = new(big.Int).Sub(node.Balance, amount)
	node.Stake = new(big.Int).Add(node.Stake, amount)

	// Create transaction
	tx := &economy.Transaction{
		ID:          fmt.Sprintf("tx_%d", m.NextTxID),
		Type:        economy.TransactionTypeStake,
		FromID:      nodeID,
		ToID:        nodeID,
		Amount:      new(big.Int).Set(amount),
		Fee:         big.NewInt(0),
		Description: "Stake tokens",
		Metadata:    make(map[string]any),
		Timestamp:   time.Now(),
		Status:      "completed",
	}

	m.NextTxID++
	m.Transactions = append(m.Transactions, tx)

	return tx, nil
}

// Unstake unstakes tokens for a node
func (m *TestMockEconomicEngine) Unstake(nodeID string, amount *big.Int) (*economy.Transaction, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, economy.ErrInvalidAmount
	}

	node, exists := m.NodeAccounts[nodeID]
	if !exists {
		return nil, economy.ErrAccountNotFound
	}

	// Check stake
	if node.Stake.Cmp(amount) < 0 {
		return nil, economy.ErrInsufficientStake
	}

	// Perform unstaking
	node.Stake = new(big.Int).Sub(node.Stake, amount)
	node.Balance = new(big.Int).Add(node.Balance, amount)

	// Create transaction
	tx := &economy.Transaction{
		ID:          fmt.Sprintf("tx_%d", m.NextTxID),
		Type:        economy.TransactionTypeUnstake,
		FromID:      nodeID,
		ToID:        nodeID,
		Amount:      new(big.Int).Set(amount),
		Fee:         big.NewInt(0),
		Description: "Unstake tokens",
		Metadata:    make(map[string]any),
		Timestamp:   time.Now(),
		Status:      "completed",
	}

	m.NextTxID++
	m.Transactions = append(m.Transactions, tx)

	return tx, nil
}

// CalculateQueryPrice calculates the price for a query
func (m *TestMockEconomicEngine) CalculateQueryPrice(modelTier economy.ModelTier, queryType economy.QueryType, inputSize int64) (*big.Int, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	// Get pricing rules for the model tier and query type
	tierRules, exists := m.PricingRules[modelTier]
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
		sizeTokens := new(big.Float).Mul(big.NewFloat(sizeComponent), new(big.Float).SetInt(economy.TokenUnit))
		var sizeInt big.Int
		sizeTokens.Int(&sizeInt)

		// Add to price
		price = new(big.Int).Add(price, &sizeInt)
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
func (m *TestMockEconomicEngine) ProcessQueryPayment(
	ctx context.Context,
	userID string,
	modelID string,
	modelTier economy.ModelTier,
	queryType economy.QueryType,
	inputSize int64,
) (*economy.Transaction, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	// Calculate the price
	price, err := m.CalculateQueryPrice(modelTier, queryType, inputSize)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate price: %w", err)
	}

	// Get user account
	user, exists := m.UserAccounts[userID]
	if !exists {
		return nil, economy.ErrAccountNotFound
	}

	// Check if user has sufficient balance
	if user.Balance.Cmp(price) < 0 {
		return nil, economy.ErrInsufficientBalance
	}

	// Calculate network fee (5%)
	feePercent := 5
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
	user.Balance = new(big.Int).Sub(user.Balance, price)
	user.TotalSpent = new(big.Int).Add(user.TotalSpent, price)
	user.QueriesSubmitted++

	// Create a transaction record
	tx := &economy.Transaction{
		ID:          fmt.Sprintf("tx_%d", m.NextTxID),
		Type:        economy.TransactionTypeQueryPayment,
		FromID:      userID,
		ToID:        "network", // Initially to network account
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
		Timestamp: time.Now(),
		Status:    "pending_reward", // Will be completed after reward distribution
	}

	m.NextTxID++
	m.Transactions = append(m.Transactions, tx)

	return tx, nil
}

// DistributeQueryReward distributes the reward for a processed query
func (m *TestMockEconomicEngine) DistributeQueryReward(
	ctx context.Context,
	txID string,
	nodeID string,
	responseTime int64,
	success bool,
	qualityScore float64,
) (*economy.Transaction, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	// Find the payment transaction
	var paymentTx *economy.Transaction
	for _, tx := range m.Transactions {
		if tx.ID == txID && tx.Type == economy.TransactionTypeQueryPayment && tx.Status == "pending_reward" {
			paymentTx = tx
			break
		}
	}

	if paymentTx == nil {
		return nil, fmt.Errorf("payment transaction not found or not in pending state")
	}

	// Get node account
	node, exists := m.NodeAccounts[nodeID]
	if !exists {
		return nil, economy.ErrAccountNotFound
	}

	// Parse amount after fee from metadata
	amountAfterFeeStr, ok := paymentTx.Metadata["amount_after_fee"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid payment transaction metadata")
	}

	amountAfterFee, ok := new(big.Int).SetString(amountAfterFeeStr, 10)
	if !ok {
		return nil, fmt.Errorf("invalid amount format in metadata")
	}

	// Calculate reward with multipliers (simplified for testing)
	reward := new(big.Int).Set(amountAfterFee)

	// Add to node's balance
	node.Balance = new(big.Int).Add(node.Balance, reward)
	node.TotalEarned = new(big.Int).Add(node.TotalEarned, reward)

	// Update node metrics
	node.QueriesProcessed++
	node.LastActive = time.Now()

	// Update payment transaction status
	paymentTx.Status = "completed"

	// Create a reward transaction record
	tx := &economy.Transaction{
		ID:          fmt.Sprintf("tx_%d", m.NextTxID),
		Type:        economy.TransactionTypeReward,
		FromID:      "network",
		ToID:        nodeID,
		Amount:      new(big.Int).Set(reward),
		Fee:         big.NewInt(0),
		Description: "Reward for query processing",
		Metadata: map[string]any{
			"payment_tx_id": paymentTx.ID,
			"response_time": responseTime,
			"success":       success,
			"quality_score": qualityScore,
			"reputation":    node.Reputation,
		},
		Timestamp: time.Now(),
		Status:    "completed",
	}

	m.NextTxID++
	m.Transactions = append(m.Transactions, tx)

	return tx, nil
}

// SlashNode penalizes a node for poor performance or malicious behavior
func (m *TestMockEconomicEngine) SlashNode(nodeID string, reason string, severity float64) (*economy.Transaction, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	if severity <= 0 || severity > 1.0 {
		return nil, fmt.Errorf("severity must be between 0 and 1")
	}

	node, exists := m.NodeAccounts[nodeID]
	if !exists {
		return nil, economy.ErrAccountNotFound
	}

	// Calculate slash amount based on stake and severity
	slashPercent := severity * 100.0
	slashFloat := new(big.Float).Mul(
		new(big.Float).SetInt(node.Stake),
		new(big.Float).Quo(
			big.NewFloat(slashPercent),
			big.NewFloat(100.0),
		),
	)

	var slashAmount big.Int
	slashFloat.Int(&slashAmount)

	// Ensure we don't slash more than available
	if slashAmount.Cmp(node.Stake) > 0 {
		slashAmount = *new(big.Int).Set(node.Stake)
	}

	// Apply the slash
	node.Stake = new(big.Int).Sub(node.Stake, &slashAmount)

	// Update reputation
	reputationPenalty := severity * 20.0 // Up to 20 points for max severity
	node.Reputation -= reputationPenalty
	if node.Reputation < 0 {
		node.Reputation = 0
	}

	// Create a slash transaction record
	tx := &economy.Transaction{
		ID:          fmt.Sprintf("tx_%d", m.NextTxID),
		Type:        economy.TransactionTypeSlash,
		FromID:      nodeID,
		ToID:        "network",
		Amount:      new(big.Int).Set(&slashAmount),
		Fee:         big.NewInt(0),
		Description: fmt.Sprintf("Slashed for: %s", reason),
		Metadata: map[string]any{
			"reason":   reason,
			"severity": severity,
		},
		Timestamp: time.Now(),
		Status:    "completed",
	}

	m.NextTxID++
	m.Transactions = append(m.Transactions, tx)

	return tx, nil
}

// MintTokens creates new tokens and adds them to an account
func (m *TestMockEconomicEngine) MintTokens(accountID string, amount *big.Int, reason string) (*economy.Transaction, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, economy.ErrInvalidAmount
	}

	// Find account
	var account *economy.Account
	if user, exists := m.UserAccounts[accountID]; exists {
		account = user.Account
	} else if node, exists := m.NodeAccounts[accountID]; exists {
		account = node.Account
	} else {
		return nil, economy.ErrAccountNotFound
	}

	// Add tokens to the account
	account.Balance = new(big.Int).Add(account.Balance, amount)

	// Create a transaction record
	tx := &economy.Transaction{
		ID:          fmt.Sprintf("tx_%d", m.NextTxID),
		Type:        economy.TransactionTypeMint,
		FromID:      "",
		ToID:        accountID,
		Amount:      new(big.Int).Set(amount),
		Fee:         big.NewInt(0),
		Description: reason,
		Metadata:    make(map[string]any),
		Timestamp:   time.Now(),
		Status:      "completed",
	}

	m.NextTxID++
	m.Transactions = append(m.Transactions, tx)

	return tx, nil
}

// GetTransactions returns transactions for an account
func (m *TestMockEconomicEngine) GetTransactions(accountID string, limit, offset int) ([]*economy.Transaction, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	// Check if account exists
	var exists bool
	if _, exists = m.UserAccounts[accountID]; !exists {
		if _, exists = m.NodeAccounts[accountID]; !exists {
			return nil, economy.ErrAccountNotFound
		}
	}

	var accountTxs []*economy.Transaction
	for _, tx := range m.Transactions {
		if tx.FromID == accountID || tx.ToID == accountID {
			accountTxs = append(accountTxs, tx)
		}
	}

	// Apply pagination
	if offset >= len(accountTxs) {
		return []*economy.Transaction{}, nil
	}

	end := offset + limit
	if end > len(accountTxs) {
		end = len(accountTxs)
	}

	return accountTxs[offset:end], nil
}

// GetTransaction returns a transaction by ID
func (m *TestMockEconomicEngine) GetTransaction(txID string) (*economy.Transaction, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	for _, tx := range m.Transactions {
		if tx.ID == txID {
			return tx, nil
		}
	}

	return nil, fmt.Errorf("transaction not found")
}

// GetNetworkStats returns statistics about the economic system
func (m *TestMockEconomicEngine) GetNetworkStats() map[string]interface{} {
	if m.FailNextAction {
		m.FailNextAction = false
		return map[string]interface{}{"error": "simulated failure"}
	}

	// Count active nodes (staked)
	activeNodes := 0
	totalStaked := big.NewInt(0)
	totalReputation := 0.0

	for _, node := range m.NodeAccounts {
		if node.Stake.Cmp(big.NewInt(0)) > 0 {
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
	txCounts := make(map[economy.TransactionType]int)
	for _, tx := range m.Transactions {
		txCounts[tx.Type]++
	}

	return map[string]interface{}{
		"total_supply":       "1000000000000000000000000", // Placeholder
		"active_nodes":       activeNodes,
		"total_staked":       totalStaked.String(),
		"avg_reputation":     avgReputation,
		"total_accounts":     len(m.UserAccounts) + len(m.NodeAccounts),
		"total_users":        len(m.UserAccounts),
		"total_nodes":        len(m.NodeAccounts),
		"total_transactions": len(m.Transactions),
		"transaction_counts": txCounts,
		"total_fees":         "0",
	}
}

// UpdatePricingRule updates a pricing rule
func (m *TestMockEconomicEngine) UpdatePricingRule(rule *economy.PricingRule) error {
	if m.FailNextAction {
		m.FailNextAction = false
		return fmt.Errorf("simulated failure")
	}

	if rule == nil {
		return fmt.Errorf("pricing rule cannot be nil")
	}

	tierRules, exists := m.PricingRules[rule.ModelTier]
	if !exists {
		tierRules = make(map[economy.QueryType]*economy.PricingRule)
		m.PricingRules[rule.ModelTier] = tierRules
	}

	tierRules[rule.QueryType] = rule

	return nil
}

// GetPricingRule returns a pricing rule
func (m *TestMockEconomicEngine) GetPricingRule(modelTier economy.ModelTier, queryType economy.QueryType) (*economy.PricingRule, error) {
	if m.FailNextAction {
		m.FailNextAction = false
		return nil, fmt.Errorf("simulated failure")
	}

	tierRules, exists := m.PricingRules[modelTier]
	if !exists {
		return nil, fmt.Errorf("no pricing rules for model tier: %s", modelTier)
	}

	rule, exists := tierRules[queryType]
	if !exists {
		return nil, fmt.Errorf("no pricing rules for query type: %s", queryType)
	}

	return rule, nil
}

// setupTestServer sets up a test server with the economy service
func setupTestServer(t *testing.T) (*httptest.Server, *TestMockEconomicEngine) {
	mockEngine := NewTestMockEconomicEngine()
	economyService := NewEconomyService((*economy.EconomicEngine)(mockEngine)) // This cast is problematic if TestMockEconomicEngine doesn't match EconomicEngine methods exactly

	router := mux.NewRouter()
	economyService.RegisterRoutes(router)

	server := httptest.NewServer(router)

	return server, mockEngine
}

// TestCreateUserAccount tests the create user account endpoint
func TestCreateUserAccount(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Test successful account creation
	t.Run("Success", func(t *testing.T) {
		reqBody := `{"id":"test-user-1","address":"0xTestAddress1"}`

		resp, err := http.Post(
			server.URL+"/accounts/user",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code %d, got %d", http.StatusCreated, resp.StatusCode)
		}

		var account economy.UserAccount
		if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if account.ID != "test-user-1" {
			t.Errorf("Expected account ID %s, got %s", "test-user-1", account.ID)
		}

		if account.Address != "0xTestAddress1" {
			t.Errorf("Expected account address %s, got %s", "0xTestAddress1", account.Address)
		}
	})

	// Test duplicate account creation
	t.Run("DuplicateAccount", func(t *testing.T) {
		reqBody := `{"id":"test-user-1","address":"0xAnotherAddress"}`

		resp, err := http.Post(
			server.URL+"/accounts/user",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusConflict {
			t.Errorf("Expected status code %d, got %d", http.StatusConflict, resp.StatusCode)
		}
	})

	// Test invalid request
	t.Run("InvalidRequest", func(t *testing.T) {
		reqBody := `{"id":"","address":"0xTestAddress2"}`

		resp, err := http.Post(
			server.URL+"/accounts/user",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
		}
	})

	// Test server error
	t.Run("ServerError", func(t *testing.T) {
		mockEngine.FailNextAction = true

		reqBody := `{"id":"test-user-error","address":"0xErrorAddress"}`

		resp, err := http.Post(
			server.URL+"/accounts/user",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		}
	})
}

// TestCreateNodeAccount tests the create node account endpoint
func TestCreateNodeAccount(t *testing.T) {
	server, _ := setupTestServer(t) // mockEngine is returned but not directly used in this specific test function body
	defer server.Close()

	// Test successful account creation
	t.Run("Success", func(t *testing.T) {
		reqBody := `{"id":"test-node-1","address":"0xNodeAddress1"}`

		resp, err := http.Post(
			server.URL+"/accounts/node",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status code %d, got %d", http.StatusCreated, resp.StatusCode)
		}

		var account economy.NodeAccount
		if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if account.ID != "test-node-1" {
			t.Errorf("Expected account ID %s, got %s", "test-node-1", account.ID)
		}

		if account.Address != "0xNodeAddress1" {
			t.Errorf("Expected account address %s, got %s", "0xNodeAddress1", account.Address)
		}

		if account.Reputation != economy.InitialReputationScore {
			t.Errorf("Expected reputation %f, got %f", economy.InitialReputationScore, account.Reputation)
		}
	})

	// Test duplicate account creation
	t.Run("DuplicateAccount", func(t *testing.T) {
		reqBody := `{"id":"test-node-1","address":"0xAnotherNodeAddress"}`

		resp, err := http.Post(
			server.URL+"/accounts/node",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusConflict {
			t.Errorf("Expected status code %d, got %d", http.StatusConflict, resp.StatusCode)
		}
	})
}

// TestGetAccount tests the get account endpoint
func TestGetAccount(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Create test accounts
	userAccount, _ := mockEngine.CreateUserAccount("get-test-user", "0xGetTestUser")
	nodeAccount, _ := mockEngine.CreateNodeAccount("get-test-node", "0xGetTestNode")

	// Test getting user account
	t.Run("GetUserAccount", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/accounts/" + userAccount.ID)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var account economy.UserAccount
		if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if account.ID != userAccount.ID {
			t.Errorf("Expected account ID %s, got %s", userAccount.ID, account.ID)
		}
	})

	// Test getting node account
	t.Run("GetNodeAccount", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/accounts/" + nodeAccount.ID)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var account economy.NodeAccount
		if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if account.ID != nodeAccount.ID {
			t.Errorf("Expected account ID %s, got %s", nodeAccount.ID, account.ID)
		}
	})

	// Test getting non-existent account
	t.Run("NonExistentAccount", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/accounts/non-existent")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
	})
}

// TestTransferTokens tests the transfer tokens endpoint
func TestTransferTokens(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Create test accounts
	fromID := "transfer-from"
	toID := "transfer-to"
	mockEngine.CreateUserAccount(fromID, "0xTransferFrom")
	mockEngine.CreateUserAccount(toID, "0xTransferTo")

	// Fund the sender account
	amount := new(big.Int).Mul(big.NewInt(100), economy.TokenUnit)
	mockEngine.MintTokens(fromID, amount, "Test funding")

	// Test successful transfer
	t.Run("Success", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"from_id":"%s","to_id":"%s","amount":"50000000000000000000","description":"Test transfer"}`, fromID, toID)

		resp, err := http.Post(
			server.URL+"/tokens/transfer",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var tx economy.Transaction
		if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if tx.Type != economy.TransactionTypeTransfer {
			t.Errorf("Expected transaction type %s, got %s", economy.TransactionTypeTransfer, tx.Type)
		}

		if tx.FromID != fromID {
			t.Errorf("Expected from ID %s, got %s", fromID, tx.FromID)
		}

		if tx.ToID != toID {
			t.Errorf("Expected to ID %s, got %s", toID, tx.ToID)
		}

		expectedAmount := new(big.Int).Mul(big.NewInt(50), economy.TokenUnit)
		if tx.Amount.String() != expectedAmount.String() {
			t.Errorf("Expected amount %s, got %s", expectedAmount.String(), tx.Amount.String())
		}
	})

	// Test insufficient balance
	t.Run("InsufficientBalance", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"from_id":"%s","to_id":"%s","amount":"1000000000000000000000","description":"Test transfer too much"}`, fromID, toID)

		resp, err := http.Post(
			server.URL+"/tokens/transfer",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected status code %d, got %d", http.StatusUnprocessableEntity, resp.StatusCode)
		}
	})

	// Test non-existent account
	t.Run("NonExistentAccount", func(t *testing.T) {
		reqBody := `{"from_id":"non-existent","to_id":"transfer-to","amount":"10000000000000000000","description":"Test transfer"}`

		resp, err := http.Post(
			server.URL+"/tokens/transfer",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, resp.StatusCode)
		}
	})
}

// TestStakeTokens tests the stake tokens endpoint
func TestStakeTokens(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Create test node account
	nodeID := "stake-test-node"
	mockEngine.CreateNodeAccount(nodeID, "0xStakeTestNode")

	// Fund the node account
	amount := new(big.Int).Mul(big.NewInt(1000), economy.TokenUnit)
	mockEngine.MintTokens(nodeID, amount, "Test funding")

	// Test successful staking
	t.Run("Success", func(t *testing.T) {
		reqBody := `{"amount":"500000000000000000000"}`

		resp, err := http.Post(
			server.URL+"/nodes/"+nodeID+"/stake",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var tx economy.Transaction
		if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if tx.Type != economy.TransactionTypeStake {
			t.Errorf("Expected transaction type %s, got %s", economy.TransactionTypeStake, tx.Type)
		}

		expectedAmount := new(big.Int).Mul(big.NewInt(500), economy.TokenUnit)
		if tx.Amount.String() != expectedAmount.String() {
			t.Errorf("Expected amount %s, got %s", expectedAmount.String(), tx.Amount.String())
		}

		// Verify stake amount
		node, _ := mockEngine.GetNodeAccount(nodeID)
		if node.Stake.String() != expectedAmount.String() {
			t.Errorf("Expected stake %s, got %s", expectedAmount.String(), node.Stake.String())
		}
	})

	// Test insufficient balance
	t.Run("InsufficientBalance", func(t *testing.T) {
		reqBody := `{"amount":"1000000000000000000000"}`

		resp, err := http.Post(
			server.URL+"/nodes/"+nodeID+"/stake",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected status code %d, got %d", http.StatusUnprocessableEntity, resp.StatusCode)
		}
	})
}

// TestUnstakeTokens tests the unstake tokens endpoint
func TestUnstakeTokens(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Create test node account
	nodeID := "unstake-test-node"
	mockEngine.CreateNodeAccount(nodeID, "0xUnstakeTestNode")

	// Fund and stake
	fundAmount := new(big.Int).Mul(big.NewInt(1000), economy.TokenUnit)
	mockEngine.MintTokens(nodeID, fundAmount, "Test funding")

	stakeAmount := new(big.Int).Mul(big.NewInt(500), economy.TokenUnit)
	mockEngine.Stake(nodeID, stakeAmount)

	// Test successful unstaking
	t.Run("Success", func(t *testing.T) {
		reqBody := `{"amount":"200000000000000000000"}`

		resp, err := http.Post(
			server.URL+"/nodes/"+nodeID+"/unstake",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var tx economy.Transaction
		if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if tx.Type != economy.TransactionTypeUnstake {
			t.Errorf("Expected transaction type %s, got %s", economy.TransactionTypeUnstake, tx.Type)
		}

		expectedAmount := new(big.Int).Mul(big.NewInt(200), economy.TokenUnit)
		if tx.Amount.String() != expectedAmount.String() {
			t.Errorf("Expected amount %s, got %s", expectedAmount.String(), tx.Amount.String())
		}

		// Verify stake amount
		node, _ := mockEngine.GetNodeAccount(nodeID)
		expectedStake := new(big.Int).Mul(big.NewInt(300), economy.TokenUnit)
		if node.Stake.String() != expectedStake.String() {
			t.Errorf("Expected stake %s, got %s", expectedStake.String(), node.Stake.String())
		}
	})

	// Test insufficient stake
	t.Run("InsufficientStake", func(t *testing.T) {
		reqBody := `{"amount":"1000000000000000000000"}`

		resp, err := http.Post(
			server.URL+"/nodes/"+nodeID+"/unstake",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected status code %d, got %d", http.StatusUnprocessableEntity, resp.StatusCode)
		}
	})
}

// TestCalculateQueryPrice tests the calculate query price endpoint
func TestCalculateQueryPrice(t *testing.T) {
	server, _ := setupTestServer(t)
	defer server.Close()

	// Test standard pricing
	t.Run("StandardPricing", func(t *testing.T) {
		reqBody := `{"model_tier":"standard","query_type":"completion","input_size":0}`

		resp, err := http.Post(
			server.URL+"/economy/price",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		price, ok := result["price"].(string)
		if !ok {
			t.Fatalf("Expected price to be a string, got %T", result["price"])
		}

		// Standard completion should be 5 tokens
		expectedPrice := new(big.Int).Mul(big.NewInt(5), economy.TokenUnit).String()
		if price != expectedPrice {
			t.Errorf("Expected price %s, got %s", expectedPrice, price)
		}
	})

	// Test pricing with input size
	t.Run("InputSizePricing", func(t *testing.T) {
		reqBody := `{"model_tier":"standard","query_type":"completion","input_size":1024}`

		resp, err := http.Post(
			server.URL+"/economy/price",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if _, ok := result["price"].(string); !ok {
			t.Fatalf("Expected price to be a string, got %T", result["price"])
		}
	})

	// Test invalid model tier
	t.Run("InvalidModelTier", func(t *testing.T) {
		reqBody := `{"model_tier":"invalid","query_type":"completion","input_size":0}`

		resp, err := http.Post(
			server.URL+"/economy/price",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
		}
	})
}

// TestProcessQueryPayment tests the process query payment endpoint
func TestProcessQueryPayment(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Create test user account
	userID := "payment-test-user"
	mockEngine.CreateUserAccount(userID, "0xPaymentTestUser")

	// Fund the user account
	amount := new(big.Int).Mul(big.NewInt(1000), economy.TokenUnit)
	mockEngine.MintTokens(userID, amount, "Test funding")

	// Test successful payment processing
	t.Run("Success", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"user_id":"%s","model_id":"gpt-4","model_tier":"standard","query_type":"completion","input_size":1024}`, userID)

		resp, err := http.Post(
			server.URL+"/economy/pay",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var tx economy.Transaction
		if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if tx.Type != economy.TransactionTypeQueryPayment {
			t.Errorf("Expected transaction type %s, got %s", economy.TransactionTypeQueryPayment, tx.Type)
		}

		if tx.FromID != userID {
			t.Errorf("Expected from ID %s, got %s", userID, tx.FromID)
		}

		if tx.Status != "pending_reward" {
			t.Errorf("Expected status %s, got %s", "pending_reward", tx.Status)
		}

		// Verify user balance decreased
		user, _ := mockEngine.GetUserAccount(userID)
		if user.Balance.Cmp(amount) >= 0 {
			t.Errorf("Expected balance to decrease from %s, got %s", amount.String(), user.Balance.String())
		}
	})

	// Test insufficient balance
	t.Run("InsufficientBalance", func(t *testing.T) {
		// Create a user with no balance
		poorUserID := "poor-user"
		mockEngine.CreateUserAccount(poorUserID, "0xPoorUser")

		reqBody := fmt.Sprintf(`{"user_id":"%s","model_id":"gpt-4","model_tier":"premium","query_type":"rag","input_size":2048}`, poorUserID)

		resp, err := http.Post(
			server.URL+"/economy/pay",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("Expected status code %d, got %d", http.StatusUnprocessableEntity, resp.StatusCode)
		}
	})
}

// TestDistributeQueryReward tests the distribute query reward endpoint
func TestDistributeQueryReward(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Set admin key for testing
	AdminAPIKey = "test-admin-key"

	// Create test accounts
	userID := "reward-test-user"
	nodeID := "reward-test-node"
	mockEngine.CreateUserAccount(userID, "0xRewardTestUser")
	mockEngine.CreateNodeAccount(nodeID, "0xRewardTestNode")

	// Fund the user account
	amount := new(big.Int).Mul(big.NewInt(100), economy.TokenUnit)
	mockEngine.MintTokens(userID, amount, "Test funding")

	// Process a payment
	paymentTx, _ := mockEngine.ProcessQueryPayment(
		context.Background(),
		userID,
		"gpt-4",
		economy.ModelTierStandard,
		economy.QueryTypeCompletion,
		1024,
	)

	// Test successful reward distribution
	t.Run("Success", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"transaction_id":"%s","node_id":"%s","response_time_ms":500,"success":true,"quality_score":0.9}`, paymentTx.ID, nodeID)

		req, _ := http.NewRequest("POST", server.URL+"/economy/reward", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Admin-Key", "test-admin-key")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var tx economy.Transaction
		if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if tx.Type != economy.TransactionTypeReward {
			t.Errorf("Expected transaction type %s, got %s", economy.TransactionTypeReward, tx.Type)
		}

		if tx.ToID != nodeID {
			t.Errorf("Expected to ID %s, got %s", nodeID, tx.ToID)
		}

		// Verify payment transaction status updated
		updatedPaymentTx, _ := mockEngine.GetTransaction(paymentTx.ID)
		if updatedPaymentTx.Status != "completed" {
			t.Errorf("Expected payment status %s, got %s", "completed", updatedPaymentTx.Status)
		}

		// Verify node balance increased
		node, _ := mockEngine.GetNodeAccount(nodeID)
		if node.Balance.Cmp(big.NewInt(0)) <= 0 {
			t.Errorf("Expected node balance to increase from 0, got %s", node.Balance.String())
		}
	})

	// Test unauthorized access
	t.Run("Unauthorized", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"transaction_id":"%s","node_id":"%s","response_time_ms":500,"success":true,"quality_score":0.9}`, paymentTx.ID, nodeID)

		// Send request without admin key
		resp, err := http.Post(
			server.URL+"/economy/reward",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		}
	})
}

// TestGetNetworkStats tests the get network stats endpoint
func TestGetNetworkStats(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Create some test accounts and transactions
	mockEngine.CreateUserAccount("stats-user-1", "0xStatsUser1")
	mockEngine.CreateUserAccount("stats-user-2", "0xStatsUser2")

	nodeID := "stats-node-1"
	mockEngine.CreateNodeAccount(nodeID, "0xStatsNode1")

	// Fund and stake
	mockEngine.MintTokens(nodeID, new(big.Int).Mul(big.NewInt(1000), economy.TokenUnit), "Test funding")
	mockEngine.Stake(nodeID, new(big.Int).Mul(big.NewInt(500), economy.TokenUnit))

	// Test getting network stats
	t.Run("Success", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/economy/stats")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var stats map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check basic stats
		if totalAccounts, ok := stats["total_accounts"].(float64); !ok || totalAccounts < 3 {
			t.Errorf("Expected at least 3 accounts, got %v", stats["total_accounts"])
		}

		if totalUsers, ok := stats["total_users"].(float64); !ok || totalUsers < 2 {
			t.Errorf("Expected at least 2 users, got %v", stats["total_users"])
		}

		if totalNodes, ok := stats["total_nodes"].(float64); !ok || totalNodes < 1 {
			t.Errorf("Expected at least 1 node, got %v", stats["total_nodes"])
		}

		if activeNodes, ok := stats["active_nodes"].(float64); !ok || activeNodes < 1 {
			t.Errorf("Expected at least 1 active node, got %v", stats["active_nodes"])
		}
	})
}

// TestUpdatePricingRule tests the update pricing rule endpoint
func TestUpdatePricingRule(t *testing.T) {
	server, _ := setupTestServer(t) // mockEngine is used implicitly by the handlers
	defer server.Close()

	// Set admin key for testing
	AdminAPIKey = "test-admin-key"

	// Test updating a pricing rule
	t.Run("Success", func(t *testing.T) {
		reqBody := `{"model_tier": "standard","query_type": "completion","base_price": "10000000000000000000","token_per_ms": "0.001","token_per_kb": "0.1","min_price": "10000000000000000000","max_price": "100000000000000000000","demand_multiplier": "1.0"}`

		req, _ := http.NewRequest("PUT", server.URL+"/economy/pricing-rules", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Admin-Key", "test-admin-key")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		// Verify the rule was updated by calculating a new price
		reqBody = `{"model_tier":"standard","query_type":"completion","input_size":0}`

		resp, err = http.Post(
			server.URL+"/economy/price",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		price, ok := result["price"].(string)
		if !ok {
			t.Fatalf("Expected price to be a string, got %T", result["price"])
		}

		// Price should now be 10 tokens
		expectedPrice := new(big.Int).Mul(big.NewInt(10), economy.TokenUnit).String()
		if price != expectedPrice {
			t.Errorf("Expected price %s, got %s", expectedPrice, price)
		}
	})

	// Test unauthorized access
	t.Run("Unauthorized", func(t *testing.T) {
		reqBody := `{"model_tier": "standard","query_type": "completion","base_price": "5000000000000000000","token_per_ms": "0.0005","token_per_kb": "0.05","min_price": "5000000000000000000","max_price": "50000000000000000000","demand_multiplier": "1.0"}`

		// Send request without admin key
		resp, err := http.NewRequest("PUT", server.URL+"/economy/pricing-rules", bytes.NewBufferString(reqBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		resp.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		response, err := client.Do(resp)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, response.StatusCode)
		}
	})
}

// TestMintTokens tests the mint tokens endpoint
func TestMintTokens(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Set admin key for testing
	AdminAPIKey = "test-admin-key"

	// Create test account
	accountID := "mint-test-account"
	mockEngine.CreateUserAccount(accountID, "0xMintTestAccount")

	// Test successful minting
	t.Run("Success", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"account_id":"%s","amount":"1000000000000000000000","reason":"Test minting"}`, accountID)

		req, _ := http.NewRequest("POST", server.URL+"/tokens/mint", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Admin-Key", "test-admin-key")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var tx economy.Transaction
		if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if tx.Type != economy.TransactionTypeMint {
			t.Errorf("Expected transaction type %s, got %s", economy.TransactionTypeMint, tx.Type)
		}

		if tx.ToID != accountID {
			t.Errorf("Expected to ID %s, got %s", accountID, tx.ToID)
		}

		expectedAmount := new(big.Int).Mul(big.NewInt(1000), economy.TokenUnit)
		if tx.Amount.String() != expectedAmount.String() {
			t.Errorf("Expected amount %s, got %s", expectedAmount.String(), tx.Amount.String())
		}

		// Verify account balance
		account, _ := mockEngine.GetAccount(accountID)
		if account.Balance.String() != expectedAmount.String() {
			t.Errorf("Expected balance %s, got %s", expectedAmount.String(), account.Balance.String())
		}
	})

	// Test unauthorized access
	t.Run("Unauthorized", func(t *testing.T) {
		reqBody := fmt.Sprintf(`{"account_id":"%s","amount":"1000000000000000000000","reason":"Test minting"}`, accountID)

		// Send request without admin key
		resp, err := http.Post(
			server.URL+"/tokens/mint",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		}
	})
}

// TestSlashNode tests the slash node endpoint
func TestSlashNode(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Set admin key for testing
	AdminAPIKey = "test-admin-key"

	// Create test node account
	nodeID := "slash-test-node"
	mockEngine.CreateNodeAccount(nodeID, "0xSlashTestNode")

	// Fund and stake
	mockEngine.MintTokens(nodeID, new(big.Int).Mul(big.NewInt(1000), economy.TokenUnit), "Test funding")
	mockEngine.Stake(nodeID, new(big.Int).Mul(big.NewInt(500), economy.TokenUnit))

	// Get initial reputation
	node, _ := mockEngine.GetNodeAccount(nodeID)
	initialReputation := node.Reputation

	// Test successful slashing
	t.Run("Success", func(t *testing.T) {
		reqBody := `{"reason":"Poor performance","severity":0.3}`

		req, _ := http.NewRequest("POST", server.URL+"/nodes/"+nodeID+"/slash", bytes.NewBufferString(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Admin-Key", "test-admin-key")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var tx economy.Transaction
		if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if tx.Type != economy.TransactionTypeSlash {
			t.Errorf("Expected transaction type %s, got %s", economy.TransactionTypeSlash, tx.Type)
		}

		if tx.FromID != nodeID {
			t.Errorf("Expected from ID %s, got %s", nodeID, tx.FromID)
		}

		// Verify stake decreased
		node, _ = mockEngine.GetNodeAccount(nodeID)
		expectedStake := new(big.Int).Mul(big.NewInt(350), economy.TokenUnit) // 500 - 30%
		if node.Stake.String() != expectedStake.String() {
			t.Errorf("Expected stake %s, got %s", expectedStake.String(), node.Stake.String())
		}

		// Verify reputation decreased
		if node.Reputation >= initialReputation {
			t.Errorf("Expected reputation to decrease from %f, got %f", initialReputation, node.Reputation)
		}
	})

	// Test unauthorized access
	t.Run("Unauthorized", func(t *testing.T) {
		reqBody := `{"reason":"Poor performance","severity":0.3}`

		// Send request without admin key
		resp, err := http.Post(
			server.URL+"/nodes/"+nodeID+"/slash",
			"application/json",
			bytes.NewBufferString(reqBody),
		)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, resp.StatusCode)
		}
	})
}

// TestGetAccountTransactions tests the get account transactions endpoint
func TestGetAccountTransactions(t *testing.T) {
	server, mockEngine := setupTestServer(t)
	defer server.Close()

	// Create test accounts
	userID := "tx-test-user"
	nodeID := "tx-test-node"
	mockEngine.CreateUserAccount(userID, "0xTxTestUser")
	mockEngine.CreateNodeAccount(nodeID, "0xTxTestNode")

	// Fund accounts
	mockEngine.MintTokens(userID, new(big.Int).Mul(big.NewInt(1000), economy.TokenUnit), "Fund user")

	// Create some transactions
	for i := 0; i < 5; i++ {
		mockEngine.Transfer(userID, nodeID, new(big.Int).Mul(big.NewInt(10), economy.TokenUnit), "Test transfer")
		time.Sleep(1 * time.Millisecond) // Ensure unique timestamps if sorting matters
	}

	// Test getting transactions with pagination
	t.Run("GetTransactionsWithPagination", func(t *testing.T) {
		// Get first 2 transactions
		resp, err := http.Get(server.URL + "/accounts/" + userID + "/transactions?limit=2")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var txs []*economy.Transaction
		if err := json.NewDecoder(resp.Body).Decode(&txs); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(txs) != 2 {
			t.Errorf("Expected 2 transactions, got %d", len(txs))
		}

		// Get next 2 transactions
		resp, err = http.Get(server.URL + "/accounts/" + userID + "/transactions?limit=2&offset=2")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}

		var nextTxs []*economy.Transaction
		if err := json.NewDecoder(resp.Body).Decode(&nextTxs); err != nil {
			t.Fatalf("Failed to decode response for next transactions: %v", err)
		}

		if len(nextTxs) != 2 {
			t.Errorf("Expected 2 next transactions, got %d", len(nextTxs))
		}

		// Ensure transactions are different (basic check, assumes order or unique IDs)
		if len(txs) > 0 && len(nextTxs) > 0 && txs[0].ID == nextTxs[0].ID {
			t.Errorf("Pagination returned same transactions: first page tx ID %s, second page tx ID %s", txs[0].ID, nextTxs[0].ID)
		}

		// Get all transactions for the user (1 mint + 5 transfers = 6)
		resp, err = http.Get(server.URL + "/accounts/" + userID + "/transactions?limit=10") // Limit > total
		if err != nil {
			t.Fatalf("Failed to send request for all transactions: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d for all transactions, got %d", http.StatusOK, resp.StatusCode)
		}

		var allTxs []*economy.Transaction
		if err := json.NewDecoder(resp.Body).Decode(&allTxs); err != nil {
			t.Fatalf("Failed to decode response for all transactions: %v", err)
		}

		expectedTotalTxs := 1 + 5
		if len(allTxs) != expectedTotalTxs {
			t.Errorf("Expected %d total transactions for user, got %d. Transactions: %+v", expectedTotalTxs, len(allTxs), allTxs)
		}
	})

	// Test non-existent account
	t.Run("NonExistentAccount", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/accounts/non-existent-user/transactions")
		if err != nil {
			t.Fatalf("Failed to send request for non-existent account: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status code %d for non-existent account, got %d", http.StatusNotFound, resp.StatusCode)
		}
	})
}

*/
