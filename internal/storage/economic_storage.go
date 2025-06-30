package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/loreum-org/cortex/internal/economy"
	"github.com/loreum-org/cortex/pkg/types"
)

// EconomicStorage provides persistent storage for economic state
type EconomicStorage struct {
	sqlStorage   *SQLStorage
	redisStorage *RedisStorage
	ctx          context.Context
}

// NewEconomicStorage creates a new economic storage service
func NewEconomicStorage(sqlConfig, redisConfig *types.StorageConfig) (*EconomicStorage, error) {
	// Initialize SQL storage
	sqlStorage, err := NewSQLStorage(sqlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SQL storage: %w", err)
	}

	// Initialize Redis storage
	redisStorage, err := NewRedisStorage(redisConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis storage: %w", err)
	}

	storage := &EconomicStorage{
		sqlStorage:   sqlStorage,
		redisStorage: redisStorage,
		ctx:          context.Background(),
	}

	// Initialize database schema
	if err := storage.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return storage, nil
}

// initializeSchema creates the necessary database tables for economic data
func (es *EconomicStorage) initializeSchema() error {
	// Create accounts table
	_, err := es.sqlStorage.DB.Exec(`
		CREATE TABLE IF NOT EXISTS accounts (
			id VARCHAR(255) PRIMARY KEY,
			address VARCHAR(255) NOT NULL,
			balance DECIMAL(78, 0) NOT NULL DEFAULT 0,
			stake DECIMAL(78, 0) NOT NULL DEFAULT 0,
			account_type ENUM('user', 'node', 'network') NOT NULL DEFAULT 'user',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_address (address),
			INDEX idx_type (account_type)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create accounts table: %w", err)
	}

	// Create node_accounts table for node-specific data
	_, err = es.sqlStorage.DB.Exec(`
		CREATE TABLE IF NOT EXISTS node_accounts (
			id VARCHAR(255) PRIMARY KEY,
			reputation DECIMAL(10, 6) NOT NULL DEFAULT 50.0,
			total_earned DECIMAL(78, 0) NOT NULL DEFAULT 0,
			queries_processed BIGINT NOT NULL DEFAULT 0,
			success_rate DECIMAL(5, 4) NOT NULL DEFAULT 1.0,
			avg_response_time BIGINT NOT NULL DEFAULT 0,
			last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			capabilities JSON,
			performance JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (id) REFERENCES accounts(id) ON DELETE CASCADE,
			INDEX idx_reputation (reputation),
			INDEX idx_last_active (last_active)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create node_accounts table: %w", err)
	}

	// Create user_accounts table for user-specific data
	_, err = es.sqlStorage.DB.Exec(`
		CREATE TABLE IF NOT EXISTS user_accounts (
			id VARCHAR(255) PRIMARY KEY,
			total_spent DECIMAL(78, 0) NOT NULL DEFAULT 0,
			queries_submitted BIGINT NOT NULL DEFAULT 0,
			usage_by_model JSON,
			usage_by_type JSON,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (id) REFERENCES accounts(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_accounts table: %w", err)
	}

	// Create transactions table
	_, err = es.sqlStorage.DB.Exec(`
		CREATE TABLE IF NOT EXISTS economic_transactions (
			id VARCHAR(255) PRIMARY KEY,
			type ENUM('query_payment', 'reward', 'stake', 'unstake', 'transfer', 'network_fee', 'slash', 'mint') NOT NULL,
			from_id VARCHAR(255),
			to_id VARCHAR(255),
			amount DECIMAL(78, 0) NOT NULL,
			fee DECIMAL(78, 0) NOT NULL DEFAULT 0,
			description TEXT,
			metadata JSON,
			timestamp TIMESTAMP NOT NULL,
			status ENUM('pending', 'completed', 'failed', 'pending_reward', 'consensus_finalized', 'consensus_failed') NOT NULL DEFAULT 'pending',
			consensus_tx_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_type (type),
			INDEX idx_from_id (from_id),
			INDEX idx_to_id (to_id),
			INDEX idx_timestamp (timestamp),
			INDEX idx_status (status),
			INDEX idx_consensus_tx_id (consensus_tx_id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create economic_transactions table: %w", err)
	}

	// Create pricing_rules table
	_, err = es.sqlStorage.DB.Exec(`
		CREATE TABLE IF NOT EXISTS pricing_rules (
			id INT AUTO_INCREMENT PRIMARY KEY,
			model_tier ENUM('basic', 'standard', 'premium', 'enterprise') NOT NULL,
			query_type ENUM('completion', 'chat', 'embedding', 'codegen', 'rag') NOT NULL,
			base_price DECIMAL(78, 0) NOT NULL,
			token_per_ms DECIMAL(20, 10) NOT NULL,
			token_per_kb DECIMAL(20, 10) NOT NULL,
			min_price DECIMAL(78, 0) NOT NULL,
			max_price DECIMAL(78, 0) NOT NULL,
			demand_multiplier DECIMAL(10, 6) NOT NULL DEFAULT 1.0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY unique_tier_type (model_tier, query_type)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create pricing_rules table: %w", err)
	}

	log.Println("Economic storage schema initialized successfully")
	return nil
}

// StoreAccount stores an account in the database
func (es *EconomicStorage) StoreAccount(account *economy.Account) error {
	// Convert big.Int to string for storage
	balanceStr := account.Balance.String()
	stakeStr := account.Stake.String()

	// Determine account type
	accountType := "user" // default
	if account.ID == "network" {
		accountType = "network"
	}

	_, err := es.sqlStorage.DB.Exec(`
		INSERT INTO accounts (id, address, balance, stake, account_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			address = VALUES(address),
			balance = VALUES(balance),
			stake = VALUES(stake),
			updated_at = VALUES(updated_at)
	`, account.ID, account.Address, balanceStr, stakeStr, accountType, account.CreatedAt, account.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to store account %s: %w", account.ID, err)
	}

	// Cache in Redis for fast access
	accountData, _ := json.Marshal(account)
	es.redisStorage.Store(fmt.Sprintf("account:%s", account.ID), accountData)

	return nil
}

// LoadAccount loads an account from the database
func (es *EconomicStorage) LoadAccount(accountID string) (*economy.Account, error) {
	// Try Redis cache first
	if cached, err := es.redisStorage.Retrieve(fmt.Sprintf("account:%s", accountID)); err == nil && cached != nil {
		var account economy.Account
		if json.Unmarshal(cached, &account) == nil {
			return &account, nil
		}
	}

	// Load from SQL
	var balanceStr, stakeStr string
	var account economy.Account

	err := es.sqlStorage.DB.QueryRow(`
		SELECT id, address, balance, stake, created_at, updated_at
		FROM accounts
		WHERE id = ?
	`, accountID).Scan(&account.ID, &account.Address, &balanceStr, &stakeStr, &account.CreatedAt, &account.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, economy.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to load account %s: %w", accountID, err)
	}

	// Convert string back to big.Int
	account.Balance, _ = new(big.Int).SetString(balanceStr, 10)
	account.Stake, _ = new(big.Int).SetString(stakeStr, 10)

	// Cache in Redis
	accountData, _ := json.Marshal(&account)
	es.redisStorage.Store(fmt.Sprintf("account:%s", accountID), accountData)

	return &account, nil
}

// StoreNodeAccount stores a node account in the database
func (es *EconomicStorage) StoreNodeAccount(nodeAccount *economy.NodeAccount) error {
	// First store the base account
	if err := es.StoreAccount(nodeAccount.Account); err != nil {
		return err
	}

	// Convert capabilities and performance to JSON
	capabilitiesJSON, _ := json.Marshal(nodeAccount.Capabilities)
	performanceJSON, _ := json.Marshal(nodeAccount.Performance)
	totalEarnedStr := nodeAccount.TotalEarned.String()

	_, err := es.sqlStorage.DB.Exec(`
		INSERT INTO node_accounts (id, reputation, total_earned, queries_processed, success_rate, avg_response_time, last_active, capabilities, performance, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			reputation = VALUES(reputation),
			total_earned = VALUES(total_earned),
			queries_processed = VALUES(queries_processed),
			success_rate = VALUES(success_rate),
			avg_response_time = VALUES(avg_response_time),
			last_active = VALUES(last_active),
			capabilities = VALUES(capabilities),
			performance = VALUES(performance),
			updated_at = VALUES(updated_at)
	`, nodeAccount.ID, nodeAccount.Reputation, totalEarnedStr, nodeAccount.QueriesProcessed,
		nodeAccount.SuccessRate, nodeAccount.AvgResponseTime, nodeAccount.LastActive,
		capabilitiesJSON, performanceJSON, nodeAccount.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to store node account %s: %w", nodeAccount.ID, err)
	}

	// Cache in Redis
	nodeData, _ := json.Marshal(nodeAccount)
	es.redisStorage.Store(fmt.Sprintf("node:%s", nodeAccount.ID), nodeData)

	return nil
}

// LoadNodeAccount loads a node account from the database
func (es *EconomicStorage) LoadNodeAccount(nodeID string) (*economy.NodeAccount, error) {
	// Try Redis cache first
	if cached, err := es.redisStorage.Retrieve(fmt.Sprintf("node:%s", nodeID)); err == nil && cached != nil {
		var nodeAccount economy.NodeAccount
		if json.Unmarshal(cached, &nodeAccount) == nil {
			return &nodeAccount, nil
		}
	}

	// Load base account first
	baseAccount, err := es.LoadAccount(nodeID)
	if err != nil {
		return nil, err
	}

	// Load node-specific data
	var totalEarnedStr string
	var capabilitiesJSON, performanceJSON []byte
	nodeAccount := &economy.NodeAccount{Account: baseAccount}

	err = es.sqlStorage.DB.QueryRow(`
		SELECT reputation, total_earned, queries_processed, success_rate, avg_response_time, last_active, capabilities, performance
		FROM node_accounts
		WHERE id = ?
	`, nodeID).Scan(&nodeAccount.Reputation, &totalEarnedStr, &nodeAccount.QueriesProcessed,
		&nodeAccount.SuccessRate, &nodeAccount.AvgResponseTime, &nodeAccount.LastActive,
		&capabilitiesJSON, &performanceJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, economy.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to load node account %s: %w", nodeID, err)
	}

	// Convert string back to big.Int
	nodeAccount.TotalEarned, _ = new(big.Int).SetString(totalEarnedStr, 10)

	// Unmarshal JSON fields
	json.Unmarshal(capabilitiesJSON, &nodeAccount.Capabilities)
	json.Unmarshal(performanceJSON, &nodeAccount.Performance)

	// Cache in Redis
	nodeData, _ := json.Marshal(nodeAccount)
	es.redisStorage.Store(fmt.Sprintf("node:%s", nodeID), nodeData)

	return nodeAccount, nil
}

// StoreUserAccount stores a user account in the database
func (es *EconomicStorage) StoreUserAccount(userAccount *economy.UserAccount) error {
	// First store the base account
	if err := es.StoreAccount(userAccount.Account); err != nil {
		return err
	}

	// Convert usage maps to JSON
	usageByModelJSON, _ := json.Marshal(userAccount.UsageByModel)
	usageByTypeJSON, _ := json.Marshal(userAccount.UsageByType)
	totalSpentStr := userAccount.TotalSpent.String()

	_, err := es.sqlStorage.DB.Exec(`
		INSERT INTO user_accounts (id, total_spent, queries_submitted, usage_by_model, usage_by_type, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			total_spent = VALUES(total_spent),
			queries_submitted = VALUES(queries_submitted),
			usage_by_model = VALUES(usage_by_model),
			usage_by_type = VALUES(usage_by_type),
			updated_at = VALUES(updated_at)
	`, userAccount.ID, totalSpentStr, userAccount.QueriesSubmitted,
		usageByModelJSON, usageByTypeJSON, userAccount.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to store user account %s: %w", userAccount.ID, err)
	}

	// Cache in Redis
	userData, _ := json.Marshal(userAccount)
	es.redisStorage.Store(fmt.Sprintf("user:%s", userAccount.ID), userData)

	return nil
}

// LoadUserAccount loads a user account from the database
func (es *EconomicStorage) LoadUserAccount(userID string) (*economy.UserAccount, error) {
	// Try Redis cache first
	if cached, err := es.redisStorage.Retrieve(fmt.Sprintf("user:%s", userID)); err == nil && cached != nil {
		var userAccount economy.UserAccount
		if json.Unmarshal(cached, &userAccount) == nil {
			return &userAccount, nil
		}
	}

	// Load base account first
	baseAccount, err := es.LoadAccount(userID)
	if err != nil {
		return nil, err
	}

	// Load user-specific data
	var totalSpentStr string
	var usageByModelJSON, usageByTypeJSON []byte
	userAccount := &economy.UserAccount{Account: baseAccount}

	err = es.sqlStorage.DB.QueryRow(`
		SELECT total_spent, queries_submitted, usage_by_model, usage_by_type
		FROM user_accounts
		WHERE id = ?
	`, userID).Scan(&totalSpentStr, &userAccount.QueriesSubmitted, &usageByModelJSON, &usageByTypeJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, economy.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to load user account %s: %w", userID, err)
	}

	// Convert string back to big.Int
	userAccount.TotalSpent, _ = new(big.Int).SetString(totalSpentStr, 10)

	// Unmarshal JSON fields
	json.Unmarshal(usageByModelJSON, &userAccount.UsageByModel)
	json.Unmarshal(usageByTypeJSON, &userAccount.UsageByType)

	// Cache in Redis
	userData, _ := json.Marshal(userAccount)
	es.redisStorage.Store(fmt.Sprintf("user:%s", userID), userData)

	return userAccount, nil
}

// StoreTransaction stores a transaction in the database
func (es *EconomicStorage) StoreTransaction(tx *economy.Transaction) error {
	// Convert metadata to JSON
	metadataJSON, _ := json.Marshal(tx.Metadata)
	amountStr := tx.Amount.String()
	feeStr := tx.Fee.String()

	_, err := es.sqlStorage.DB.Exec(`
		INSERT INTO economic_transactions (id, type, from_id, to_id, amount, fee, description, metadata, timestamp, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			status = VALUES(status),
			metadata = VALUES(metadata)
	`, tx.ID, string(tx.Type), tx.FromID, tx.ToID, amountStr, feeStr,
		tx.Description, metadataJSON, tx.Timestamp, tx.Status)

	if err != nil {
		return fmt.Errorf("failed to store transaction %s: %w", tx.ID, err)
	}

	// Cache recent transactions in Redis
	txData, _ := json.Marshal(tx)
	es.redisStorage.Store(fmt.Sprintf("tx:%s", tx.ID), txData)

	return nil
}

// LoadTransaction loads a transaction from the database
func (es *EconomicStorage) LoadTransaction(txID string) (*economy.Transaction, error) {
	// Try Redis cache first
	if cached, err := es.redisStorage.Retrieve(fmt.Sprintf("tx:%s", txID)); err == nil && cached != nil {
		var tx economy.Transaction
		if json.Unmarshal(cached, &tx) == nil {
			return &tx, nil
		}
	}

	// Load from SQL
	var amountStr, feeStr string
	var metadataJSON []byte
	var tx economy.Transaction

	err := es.sqlStorage.DB.QueryRow(`
		SELECT id, type, from_id, to_id, amount, fee, description, metadata, timestamp, status
		FROM economic_transactions
		WHERE id = ?
	`, txID).Scan(&tx.ID, &tx.Type, &tx.FromID, &tx.ToID, &amountStr, &feeStr,
		&tx.Description, &metadataJSON, &tx.Timestamp, &tx.Status)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to load transaction %s: %w", txID, err)
	}

	// Convert strings back to big.Int
	tx.Amount, _ = new(big.Int).SetString(amountStr, 10)
	tx.Fee, _ = new(big.Int).SetString(feeStr, 10)

	// Unmarshal metadata
	json.Unmarshal(metadataJSON, &tx.Metadata)

	// Cache in Redis
	txData, _ := json.Marshal(&tx)
	es.redisStorage.Store(fmt.Sprintf("tx:%s", txID), txData)

	return &tx, nil
}

// GetTransactionsByAccount retrieves transactions for an account
func (es *EconomicStorage) GetTransactionsByAccount(accountID string, limit, offset int) ([]*economy.Transaction, error) {
	rows, err := es.sqlStorage.DB.Query(`
		SELECT id, type, from_id, to_id, amount, fee, description, metadata, timestamp, status
		FROM economic_transactions
		WHERE from_id = ? OR to_id = ?
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, accountID, accountID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to query transactions for account %s: %w", accountID, err)
	}
	defer rows.Close()

	var transactions []*economy.Transaction
	for rows.Next() {
		var tx economy.Transaction
		var amountStr, feeStr string
		var metadataJSON []byte

		err := rows.Scan(&tx.ID, &tx.Type, &tx.FromID, &tx.ToID, &amountStr, &feeStr,
			&tx.Description, &metadataJSON, &tx.Timestamp, &tx.Status)
		if err != nil {
			continue
		}

		// Convert strings back to big.Int
		tx.Amount, _ = new(big.Int).SetString(amountStr, 10)
		tx.Fee, _ = new(big.Int).SetString(feeStr, 10)

		// Unmarshal metadata
		json.Unmarshal(metadataJSON, &tx.Metadata)

		transactions = append(transactions, &tx)
	}

	return transactions, nil
}

// StorePricingRule stores a pricing rule in the database
func (es *EconomicStorage) StorePricingRule(rule *economy.PricingRule) error {
	basePriceStr := rule.BasePrice.String()
	minPriceStr := rule.MinPrice.String()
	maxPriceStr := rule.MaxPrice.String()
	tokenPerMS, _ := rule.TokenPerMS.Float64()
	tokenPerKB, _ := rule.TokenPerKB.Float64()
	demandMultiplier, _ := rule.DemandMultiplier.Float64()

	_, err := es.sqlStorage.DB.Exec(`
		INSERT INTO pricing_rules (model_tier, query_type, base_price, token_per_ms, token_per_kb, min_price, max_price, demand_multiplier)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			base_price = VALUES(base_price),
			token_per_ms = VALUES(token_per_ms),
			token_per_kb = VALUES(token_per_kb),
			min_price = VALUES(min_price),
			max_price = VALUES(max_price),
			demand_multiplier = VALUES(demand_multiplier),
			updated_at = CURRENT_TIMESTAMP
	`, string(rule.ModelTier), string(rule.QueryType), basePriceStr, tokenPerMS, tokenPerKB,
		minPriceStr, maxPriceStr, demandMultiplier)

	if err != nil {
		return fmt.Errorf("failed to store pricing rule: %w", err)
	}

	// Cache in Redis
	ruleKey := fmt.Sprintf("pricing:%s:%s", rule.ModelTier, rule.QueryType)
	ruleData, _ := json.Marshal(rule)
	es.redisStorage.Store(ruleKey, ruleData)

	return nil
}

// InvalidateCache removes cached data for an account
func (es *EconomicStorage) InvalidateCache(accountID string) {
	es.redisStorage.Delete(fmt.Sprintf("account:%s", accountID))
	es.redisStorage.Delete(fmt.Sprintf("user:%s", accountID))
	es.redisStorage.Delete(fmt.Sprintf("node:%s", accountID))
}

// Close closes the storage connections
func (es *EconomicStorage) Close() error {
	if es.sqlStorage != nil {
		es.sqlStorage.Close()
	}
	if es.redisStorage != nil {
		es.redisStorage.Close()
	}
	return nil
}
