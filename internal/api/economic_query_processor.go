package api

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/loreum-org/cortex/internal/agenthub"
	"github.com/loreum-org/cortex/internal/economy"
	"github.com/loreum-org/cortex/internal/p2p"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/internal/consensus"
	"github.com/loreum-org/cortex/pkg/types"
)

// EconomicQueryProcessor handles query processing with full economic integration
type EconomicQueryProcessor struct {
	EconomicEngine *economy.EconomicEngine
	SolverAgent    *agenthub.SolverAgent
	RAGSystem      *rag.RAGSystem
	P2PNode        *p2p.P2PNode
	Metrics        *ServerMetrics
	// consensusBridge allows economic transactions to be routed through consensus
	ConsensusBridge *consensus.EconomicBridge
}

// EconomicQueryRequest represents a query request with economic parameters
type EconomicQueryRequest struct {
	Text         string            `json:"text"`
	Type         string            `json:"type"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Account      string            `json:"account"`
	ModelTier    string            `json:"model_tier,omitempty"`    // basic, standard, premium, enterprise
	QueryType    string            `json:"query_type,omitempty"`    // completion, chat, embedding, codegen, rag
	MaxPrice     string            `json:"max_price,omitempty"`     // Maximum price user is willing to pay
	MinQuality   float64           `json:"min_quality,omitempty"`   // Minimum quality score required
	Deadline     int64             `json:"deadline,omitempty"`      // Deadline in milliseconds
	Priority     int               `json:"priority,omitempty"`      // 1-10, higher is more priority
}

// EconomicQueryResponse represents a query response with economic information
type EconomicQueryResponse struct {
	QueryID       string            `json:"query_id"`
	Text          string            `json:"text"`
	Data          []byte            `json:"data,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Status        string            `json:"status"`
	Timestamp     int64             `json:"timestamp"`
	
	// Economic information
	PaymentTxID   string  `json:"payment_tx_id,omitempty"`
	RewardTxID    string  `json:"reward_tx_id,omitempty"`
	Price         string  `json:"price,omitempty"`
	ProcessorID   string  `json:"processor_id,omitempty"`
	ProcessTime   int64   `json:"process_time_ms"`
	QualityScore  float64 `json:"quality_score"`
	Success       bool    `json:"success"`
}

// SetConsensusBridge injects the consensus bridge dependency.
func (eqp *EconomicQueryProcessor) SetConsensusBridge(cb *consensus.EconomicBridge) {
	eqp.ConsensusBridge = cb
}

// ProcessEconomicQuery processes a query with full economic integration
func (eqp *EconomicQueryProcessor) ProcessEconomicQuery(ctx context.Context, req *EconomicQueryRequest) (*EconomicQueryResponse, error) {
	startTime := time.Now()
	
	// Validate request
	if err := eqp.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Determine model tier and query type
	modelTier, queryType := eqp.determineModelAndQueryType(req)
	
	// Calculate input size for pricing
	inputSize := int64(len(req.Text))
	
	// Step 1: Process payment
	paymentTx, err := eqp.processPayment(ctx, req, modelTier, queryType, inputSize)
	if err != nil {
		return &EconomicQueryResponse{
			QueryID:   generateEconomicQueryID(),
			Status:    "payment_failed",
			Timestamp: time.Now().Unix(),
		}, fmt.Errorf("payment failed: %w", err)
	}

	// Submit payment transaction to consensus, if bridge configured
	if eqp.ConsensusBridge != nil {
		if _, err := eqp.ConsensusBridge.ProcessEconomicTransaction(ctx, paymentTx); err != nil {
			log.Printf("Warning: failed to add payment transaction %s to consensus: %v", paymentTx.ID, err)
		} else {
			// Optionally wait a short period for finalization
			finalizeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			for {
				if eqp.ConsensusBridge.IsEconomicTransactionFinalized(paymentTx.ID) {
					break
				}
				select {
				case <-time.After(300 * time.Millisecond):
				case <-finalizeCtx.Done():
					// Timeout waiting for finality; continue anyway
					log.Printf("Info: payment transaction %s not finalized before timeout", paymentTx.ID)
					break
				}
				if finalizeCtx.Err() != nil {
					break
				}
			}
		}
	}

	// Step 2: Create query with economic context
	query := &types.EconomicQuery{
		Query: types.Query{
			ID:        paymentTx.ID, // Use payment transaction ID as query ID
			Text:      req.Text,
			Type:      req.Type,
			Metadata:  req.Metadata,
			Timestamp: time.Now().Unix(),
		},
		ModelTier:     types.ModelTier(modelTier),
		PaymentID:     paymentTx.ID,
		Priority:      req.Priority,
	}

	// Set max price if specified
	if req.MaxPrice != "" {
		if maxPrice, ok := new(big.Int).SetString(req.MaxPrice, 10); ok {
			query.MaxPrice = maxPrice
		}
	}

	query.MinReputation = req.MinQuality

	// Step 3: Broadcast query to P2P network
	if err := eqp.broadcastEconomicQuery(query); err != nil {
		log.Printf("Warning: Failed to broadcast economic query: %v", err)
	}

	// Step 4: Process query locally or select best node
	response, processorID, err := eqp.processQuery(ctx, query)
	if err != nil {
		// Handle failure and potentially refund
		return eqp.handleQueryFailure(ctx, paymentTx, query, err)
	}

	// Step 5: Calculate quality score
	qualityScore := eqp.calculateQualityScore(response, req)
	
	// Check if quality meets minimum requirements
	if req.MinQuality > 0 && qualityScore < req.MinQuality {
		return eqp.handleQualityFailure(ctx, paymentTx, query, qualityScore)
	}

	// Step 6: Distribute reward
	rewardTx, err := eqp.distributeReward(ctx, paymentTx, processorID, time.Since(startTime), qualityScore)
	if err != nil {
		log.Printf("Warning: Failed to distribute reward: %v", err)
	}

	// Step 7: Record metrics
	eqp.recordMetrics(query, response, time.Since(startTime), true)

	// Return economic response
	economicResponse := &EconomicQueryResponse{
		QueryID:      query.ID,
		Text:         response.Text,
		Data:         func() []byte {
			if response.Data != nil {
				if data, ok := response.Data.([]byte); ok {
					return data
				}
			}
			return nil
		}(),
		Metadata:     response.Metadata,
		Status:       "success",
		Timestamp:    response.Timestamp,
		PaymentTxID:  paymentTx.ID,
		Price:        paymentTx.Amount.String(),
		ProcessorID:  processorID,
		ProcessTime:  time.Since(startTime).Milliseconds(),
		QualityScore: qualityScore,
		Success:      true,
	}

	if rewardTx != nil {
		economicResponse.RewardTxID = rewardTx.ID
	}

	return economicResponse, nil
}

// validateRequest validates the economic query request
func (eqp *EconomicQueryProcessor) validateRequest(req *EconomicQueryRequest) error {
	if req.Text == "" {
		return fmt.Errorf("query text is required")
	}
	if req.Account == "" {
		return fmt.Errorf("user ID is required")
	}
	
	// Validate model tier if specified
	if req.ModelTier != "" {
		validTiers := map[string]bool{
			"basic": true, "standard": true, "premium": true, "enterprise": true,
		}
		if !validTiers[req.ModelTier] {
			return fmt.Errorf("invalid model tier: %s", req.ModelTier)
		}
	}
	
	// Validate query type if specified
	if req.QueryType != "" {
		validTypes := map[string]bool{
			"completion": true, "chat": true, "embedding": true, "codegen": true, "rag": true,
		}
		if !validTypes[req.QueryType] {
			return fmt.Errorf("invalid query type: %s", req.QueryType)
		}
	}
	
	return nil
}

// determineModelAndQueryType determines the appropriate model tier and query type
func (eqp *EconomicQueryProcessor) determineModelAndQueryType(req *EconomicQueryRequest) (economy.ModelTier, economy.QueryType) {
	// Determine model tier
	var modelTier economy.ModelTier
	if req.ModelTier != "" {
		modelTier = economy.ModelTier(req.ModelTier)
	} else {
		// Auto-determine based on query characteristics
		switch req.Type {
		case "code", "programming", "complex":
			modelTier = economy.ModelTierPremium
		case "simple", "basic":
			modelTier = economy.ModelTierBasic
		default:
			modelTier = economy.ModelTierStandard
		}
	}

	// Determine query type
	var queryType economy.QueryType
	if req.QueryType != "" {
		queryType = economy.QueryType(req.QueryType)
	} else {
		// Auto-determine based on request type
		switch req.Type {
		case "chat", "conversation":
			queryType = economy.QueryTypeChat
		case "code", "programming":
			queryType = economy.QueryTypeCodeGen
		case "embedding", "vector":
			queryType = economy.QueryTypeEmbedding
		case "rag", "knowledge":
			queryType = economy.QueryTypeRAG
		default:
			queryType = economy.QueryTypeCompletion
		}
	}

	return modelTier, queryType
}

// processPayment processes the payment for the query
func (eqp *EconomicQueryProcessor) processPayment(ctx context.Context, req *EconomicQueryRequest, modelTier economy.ModelTier, queryType economy.QueryType, inputSize int64) (*economy.Transaction, error) {
	// Get model ID
	modelID := "default"
	if req.Metadata != nil {
		if mid, exists := req.Metadata["model_id"]; exists {
			modelID = mid
		}
	}

	// Calculate expected price
	expectedPrice, err := eqp.EconomicEngine.CalculateQueryPrice(modelTier, queryType, inputSize)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate price: %w", err)
	}

	// Check if price exceeds user's maximum
	if req.MaxPrice != "" {
		if maxPrice, ok := new(big.Int).SetString(req.MaxPrice, 10); ok {
			if expectedPrice.Cmp(maxPrice) > 0 {
				return nil, fmt.Errorf("price %s exceeds maximum %s", expectedPrice.String(), maxPrice.String())
			}
		}
	}

	// Process payment
	return eqp.EconomicEngine.ProcessQueryPayment(ctx, req.Account, modelID, modelTier, queryType, inputSize)
}

// broadcastEconomicQuery broadcasts the query to the P2P network
func (eqp *EconomicQueryProcessor) broadcastEconomicQuery(query *types.EconomicQuery) error {
	// Convert to regular query for broadcasting
	regularQuery := &query.Query
	
	// Add economic metadata
	if regularQuery.Metadata == nil {
		regularQuery.Metadata = make(map[string]string)
	}
	regularQuery.Metadata["payment_id"] = query.PaymentID
	regularQuery.Metadata["model_tier"] = string(query.ModelTier)
	if query.MaxPrice != nil {
		regularQuery.Metadata["max_price"] = query.MaxPrice.String()
	}
	regularQuery.Metadata["min_reputation"] = fmt.Sprintf("%.2f", query.MinReputation)
	regularQuery.Metadata["priority"] = fmt.Sprintf("%d", query.Priority)

	return eqp.P2PNode.BroadcastQuery(regularQuery)
}

// processQuery processes the query using the appropriate system
func (eqp *EconomicQueryProcessor) processQuery(ctx context.Context, query *types.EconomicQuery) (*types.Response, string, error) {
	// For now, process locally using SolverAgent
	// In a real implementation, this would:
	// 1. Check if we should process locally or wait for network responses
	// 2. Consider node reputation and stake
	// 3. Select the best processor based on economic factors

	response, err := eqp.SolverAgent.Process(ctx, &query.Query)
	if err != nil {
		return nil, "", err
	}

	// Use this node's ID as processor
	processorID := eqp.P2PNode.Host.ID().String()

	return response, processorID, nil
}

// calculateQualityScore calculates a quality score for the response
func (eqp *EconomicQueryProcessor) calculateQualityScore(response *types.Response, req *EconomicQueryRequest) float64 {
	// Placeholder quality calculation
	// In a real implementation, this would use sophisticated quality metrics:
	// - Response relevance
	// - Accuracy validation
	// - User feedback
	// - Automated quality checks
	
	qualityScore := 1.0 // Default high quality
	
	// Simple length-based adjustment
	if len(response.Text) < 10 {
		qualityScore *= 0.5 // Very short responses get penalized
	}
	
	// Check for error indicators
	if response.Status == "error" || response.Status == "failed" {
		qualityScore = 0.0
	}
	
	return qualityScore
}

// distributeReward distributes the reward to the query processor
func (eqp *EconomicQueryProcessor) distributeReward(ctx context.Context, paymentTx *economy.Transaction, processorID string, responseTime time.Duration, qualityScore float64) (*economy.Transaction, error) {
	return eqp.EconomicEngine.DistributeQueryReward(
		ctx,
		paymentTx.ID,
		processorID,
		responseTime.Milliseconds(),
		true, // Success
		qualityScore,
	)
}

// handleQueryFailure handles query processing failures
func (eqp *EconomicQueryProcessor) handleQueryFailure(ctx context.Context, paymentTx *economy.Transaction, query *types.EconomicQuery, err error) (*EconomicQueryResponse, error) {
	log.Printf("Query %s failed: %v", query.ID, err)
	
	// Record failure metrics
	eqp.recordMetrics(query, nil, 0, false)
	
	// In a real implementation, we might:
	// 1. Attempt to refund the payment
	// 2. Try alternative processors
	// 3. Provide partial credit
	
	return &EconomicQueryResponse{
		QueryID:     query.ID,
		Status:      "failed",
		Timestamp:   time.Now().Unix(),
		PaymentTxID: paymentTx.ID,
		Price:       paymentTx.Amount.String(),
		Success:     false,
	}, fmt.Errorf("query processing failed: %w", err)
}

// handleQualityFailure handles cases where quality doesn't meet requirements
func (eqp *EconomicQueryProcessor) handleQualityFailure(ctx context.Context, paymentTx *economy.Transaction, query *types.EconomicQuery, qualityScore float64) (*EconomicQueryResponse, error) {
	log.Printf("Query %s quality %.2f below required %.2f", query.ID, qualityScore, query.MinReputation)
	
	// In a real implementation, we might:
	// 1. Try alternative processors
	// 2. Provide partial refund
	// 3. Allow user to accept lower quality at reduced price
	
	return &EconomicQueryResponse{
		QueryID:      query.ID,
		Status:       "quality_failure",
		Timestamp:    time.Now().Unix(),
		PaymentTxID:  paymentTx.ID,
		Price:        paymentTx.Amount.String(),
		QualityScore: qualityScore,
		Success:      false,
	}, fmt.Errorf("response quality %.2f below required %.2f", qualityScore, query.MinReputation)
}

// recordMetrics records query processing metrics
func (eqp *EconomicQueryProcessor) recordMetrics(query *types.EconomicQuery, response *types.Response, duration time.Duration, success bool) {
	if eqp.Metrics == nil {
		return
	}

	eqp.Metrics.mu.Lock()
	defer eqp.Metrics.mu.Unlock()

	eqp.Metrics.QueriesProcessed++
	
	if duration > 0 {
		eqp.Metrics.QueryLatencies = append(eqp.Metrics.QueryLatencies, duration)
		if len(eqp.Metrics.QueryLatencies) > 100 {
			eqp.Metrics.QueryLatencies = eqp.Metrics.QueryLatencies[1:]
		}
	}

	if success {
		eqp.Metrics.QuerySuccesses++
	} else {
		eqp.Metrics.QueryFailures++
	}
}

// generateEconomicQueryID generates a unique ID for economic queries
func generateEconomicQueryID() string {
	return fmt.Sprintf("eq_%d", time.Now().UnixNano())
}