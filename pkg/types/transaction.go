package types

import (
	"math/big"
	"time"
)

// TransactionType represents the type of transaction
type TransactionType string

const (
	// TransactionTypeData represents a generic data transaction
	TransactionTypeData TransactionType = "data"

	// TransactionTypeEconomic represents an economic transaction
	TransactionTypeEconomic TransactionType = "economic"

	// TransactionTypeQuery represents a query processing transaction
	TransactionTypeQuery TransactionType = "query"

	// TransactionTypeReward represents a reward distribution transaction
	TransactionTypeReward TransactionType = "reward"

	// TransactionTypeAGIUpdate represents an AGI domain knowledge update
	TransactionTypeAGIUpdate TransactionType = "agi_update"

	// TransactionTypeAGIEvolution represents an AGI evolution event
	TransactionTypeAGIEvolution TransactionType = "agi_evolution"
)

// Transaction represents a transaction in the DAG
type Transaction struct {
	ID        string            `json:"id"`
	Type      TransactionType   `json:"type"`
	Data      string            `json:"data"`
	ParentIDs []string          `json:"parent_ids"`
	Signature []byte            `json:"signature"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Finalized bool              `json:"finalized"`
	Timestamp int64             `json:"timestamp"`

	// Economic-specific fields
	EconomicData *EconomicTransactionData `json:"economic_data,omitempty"`

	// AGI-specific fields
	AGIData *AGITransactionData `json:"agi_data,omitempty"`
}

// EconomicTransactionData contains economic-specific transaction data
type EconomicTransactionData struct {
	FromAccount  string   `json:"from_account"`
	ToAccount    string   `json:"to_account"`
	Amount       *big.Int `json:"amount"`
	Fee          *big.Int `json:"fee"`
	EconomicType string   `json:"economic_type"` // payment, reward, transfer, stake, etc.
	QueryID      string   `json:"query_id,omitempty"`
	ModelTier    string   `json:"model_tier,omitempty"`
	QualityScore float64  `json:"quality_score,omitempty"`
	ResponseTime int64    `json:"response_time,omitempty"`
}

// AGITransactionData contains AGI-specific transaction data
type AGITransactionData struct {
	NodeID           string                    `json:"node_id"`
	AGIVersion       int                       `json:"agi_version"`
	IntelligenceLevel float64                  `json:"intelligence_level"`
	DomainKnowledge  map[string]float64        `json:"domain_knowledge"` // domain -> expertise level
	LearningMetrics  *AGILearningMetrics       `json:"learning_metrics,omitempty"`
	EvolutionEvent   *AGIEvolutionEvent        `json:"evolution_event,omitempty"`
	Capabilities     map[string]float64        `json:"capabilities,omitempty"`
	RecentInsights   []string                  `json:"recent_insights,omitempty"`
	UpdateType       string                    `json:"update_type"` // "knowledge_update", "evolution", "capability_growth"
	Signature        []byte                    `json:"signature,omitempty"` // Node signature for authenticity
}

// AGILearningMetrics represents learning performance metrics
type AGILearningMetrics struct {
	ConceptsLearned     int                `json:"concepts_learned"`
	PatternsIdentified  int                `json:"patterns_identified"`
	LearningRate        float64            `json:"learning_rate"`
	KnowledgeGrowth     map[string]float64 `json:"knowledge_growth"` // domain -> growth rate
	LastLearningEvent   int64              `json:"last_learning_event"`
	TotalInteractions   int64              `json:"total_interactions"`
}

// AGIEvolutionEvent represents a significant AGI evolution
type AGIEvolutionEvent struct {
	TriggerType         string             `json:"trigger_type"`
	IntelligenceGain    float64            `json:"intelligence_gain"`
	DomainsAffected     []string           `json:"domains_affected"`
	CapabilitiesGained  []string           `json:"capabilities_gained"`
	EvolutionDescription string            `json:"evolution_description"`
	Impact              float64            `json:"impact"` // 0.0 to 1.0
}

// NetworkAGISnapshot represents AGI state of all nodes at a point in time
type NetworkAGISnapshot struct {
	Timestamp        int64                        `json:"timestamp"`
	BlockHash        string                       `json:"block_hash"`
	TotalNodes       int                          `json:"total_nodes"`
	NodeAGIStates    map[string]*NodeAGIState     `json:"node_agi_states"`
	NetworkMetrics   *NetworkAGIMetrics           `json:"network_metrics"`
	DomainLeaders    map[string]string            `json:"domain_leaders"` // domain -> node_id with highest expertise
}

// NodeAGIState represents the AGI state of a specific node
type NodeAGIState struct {
	NodeID           string             `json:"node_id"`
	IntelligenceLevel float64           `json:"intelligence_level"`
	DomainKnowledge  map[string]float64 `json:"domain_knowledge"`
	LastUpdate       int64              `json:"last_update"`
	Version          int                `json:"version"`
	Reputation       float64            `json:"reputation"`
	TotalQueries     int64              `json:"total_queries"`
	SuccessRate      float64            `json:"success_rate"`
}

// NetworkAGIMetrics represents network-wide AGI metrics
type NetworkAGIMetrics struct {
	AverageIntelligence    float64            `json:"average_intelligence"`
	TotalKnowledgeDomains  int                `json:"total_knowledge_domains"`
	DomainDistribution     map[string]int     `json:"domain_distribution"` // domain -> count of nodes with expertise
	NetworkLearningRate    float64            `json:"network_learning_rate"`
	CollectiveIntelligence float64            `json:"collective_intelligence"`
	KnowledgeSpecialization map[string]float64 `json:"knowledge_specialization"` // domain -> specialization score
}

// NewTransaction creates a new transaction
func NewTransaction(id, data string, parentIDs []string) *Transaction {
	return &Transaction{
		ID:        id,
		Type:      TransactionTypeData,
		Data:      data,
		ParentIDs: parentIDs,
		Signature: nil,
		Metadata:  make(map[string]string),
		Finalized: false,
		Timestamp: time.Now().Unix(),
	}
}

// NewEconomicTransaction creates a new economic transaction
func NewEconomicTransaction(id string, economicData *EconomicTransactionData, parentIDs []string) *Transaction {
	return &Transaction{
		ID:           id,
		Type:         TransactionTypeEconomic,
		Data:         economicData.EconomicType,
		ParentIDs:    parentIDs,
		Signature:    nil,
		Metadata:     make(map[string]string),
		Finalized:    false,
		Timestamp:    time.Now().Unix(),
		EconomicData: economicData,
	}
}

// NewAGITransaction creates a new AGI transaction
func NewAGITransaction(id string, agiData *AGITransactionData, parentIDs []string) *Transaction {
	var txType TransactionType
	if agiData.EvolutionEvent != nil {
		txType = TransactionTypeAGIEvolution
	} else {
		txType = TransactionTypeAGIUpdate
	}

	return &Transaction{
		ID:        id,
		Type:      txType,
		Data:      agiData.UpdateType,
		ParentIDs: parentIDs,
		Signature: agiData.Signature,
		Metadata:  make(map[string]string),
		Finalized: false,
		Timestamp: time.Now().Unix(),
		AGIData:   agiData,
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
	Account     string    `json:"account"`
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
