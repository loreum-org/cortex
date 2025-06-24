package types

import (
	"math/big"
	"time"
)

// Transaction represents a transaction in the DAG
type Transaction struct {
	ID        string            `json:"id"`
	Data      string            `json:"data"`
	ParentIDs []string          `json:"parent_ids"`
	Signature []byte            `json:"signature"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Finalized bool              `json:"finalized"`
	Timestamp int64             `json:"timestamp"`
}

// NewTransaction creates a new transaction
func NewTransaction(id, data string, parentIDs []string) *Transaction {
	return &Transaction{
		ID:        id,
		Data:      data,
		ParentIDs: parentIDs,
		Signature: nil,
		Metadata:  make(map[string]string),
		Finalized: false,
		Timestamp: time.Now().Unix(),
	}
}

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

// EconomicQuery extends Query with economic metadata
type EconomicQuery struct {
	Query
	MaxPrice      *big.Int  `json:"max_price"`
	MinReputation float64   `json:"min_reputation"`
	ModelTier     ModelTier `json:"model_tier"`
	Priority      int       `json:"priority"`
	PaymentID     string    `json:"payment_id,omitempty"`
}

// EconomicResponse extends Response with economic information
type EconomicResponse struct {
	Response
	Price         *big.Int `json:"price"`
	Reward        *big.Int `json:"reward"`
	TransactionID string   `json:"transaction_id"`
	ProcessorID   string   `json:"processor_id"`
	ProcessTime   int64    `json:"process_time_ms"`
}

// NodeEconomicStatus represents the economic status of a node
type NodeEconomicStatus struct {
	NodeID        string    `json:"node_id"`
	Balance       *big.Int  `json:"balance"`
	Stake         *big.Int  `json:"stake"`
	Reputation    float64   `json:"reputation"`
	TotalEarned   *big.Int  `json:"total_earned"`
	LastActive    time.Time `json:"last_active"`
	SuccessRate   float64   `json:"success_rate"`
	ResponseTime  int64     `json:"avg_response_time_ms"`
	QueriesServed int64     `json:"queries_served"`
	StakeRatio    float64   `json:"stake_ratio"`
}

// PaymentRequest represents a request to process a payment
type PaymentRequest struct {
	UserID      string    `json:"user_id"`
	ModelID     string    `json:"model_id"`
	ModelTier   ModelTier `json:"model_tier"`
	QueryType   QueryType `json:"query_type"`
	InputSize   int64     `json:"input_size"`
	MaxPrice    *big.Int  `json:"max_price"`
	Description string    `json:"description"`
}

// RewardDistribution represents a reward distribution
type RewardDistribution struct {
	NodeID        string         `json:"node_id"`
	Amount        *big.Int       `json:"amount"`
	TransactionID string         `json:"transaction_id"`
	ResponseTime  int64          `json:"response_time_ms"`
	Success       bool           `json:"success"`
	QualityScore  float64        `json:"quality_score"`
	Timestamp     time.Time      `json:"timestamp"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// EconomicMetrics represents economic performance metrics
type EconomicMetrics struct {
	TotalSupply       *big.Int                 `json:"total_supply"`
	ActiveNodes       int                      `json:"active_nodes"`
	TotalStaked       *big.Int                 `json:"total_staked"`
	AvgReputation     float64                  `json:"avg_reputation"`
	TotalTransactions int                      `json:"total_transactions"`
	TransactionCounts map[string]int           `json:"transaction_counts"`
	TotalFees         *big.Int                 `json:"total_fees"`
	DemandFactors     map[ModelTier]*big.Float `json:"demand_factors"`
	PriceHistory      map[ModelTier][]*big.Int `json:"price_history"`
	RevenueByModel    map[string]*big.Int      `json:"revenue_by_model"`
	RevenueByType     map[QueryType]*big.Int   `json:"revenue_by_type"`
	NodeEarnings      map[string]*big.Int      `json:"node_earnings"`
	UpdatedAt         time.Time                `json:"updated_at"`
}
