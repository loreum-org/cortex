package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/loreum-org/cortex/internal/economy"
)

// AdminAPIKey is the API key required for administrative operations
var AdminAPIKey = os.Getenv("LOREUM_ADMIN_KEY")

// EconomyService provides API handlers for the economic engine
type EconomyService struct {
	Engine *economy.EconomicEngine
}

// NewEconomyService creates a new EconomyService
func NewEconomyService(engine *economy.EconomicEngine) *EconomyService {
	return &EconomyService{
		Engine: engine,
	}
}

// RegisterRoutes registers the economic API routes
func (s *EconomyService) RegisterRoutes(router *mux.Router) {
	// Account Management
	router.HandleFunc("/accounts/user", s.createUserAccountHandler).Methods("POST")
	router.HandleFunc("/accounts/node", s.createNodeAccountHandler).Methods("POST")
	router.HandleFunc("/accounts/{id}", s.getAccountHandler).Methods("GET")
	router.HandleFunc("/accounts/{id}/transactions", s.getAccountTransactionsHandler).Methods("GET")

	// Token Operations
	router.HandleFunc("/tokens/transfer", s.transferTokensHandler).Methods("POST")
	router.HandleFunc("/nodes/{id}/stake", s.stakeTokensHandler).Methods("POST")
	router.HandleFunc("/nodes/{id}/unstake", s.unstakeTokensHandler).Methods("POST")
	router.HandleFunc("/economy/stats", s.getNetworkStatsHandler).Methods("GET")

	// Pricing & Payments
	router.HandleFunc("/economy/price", s.calculateQueryPriceHandler).Methods("POST")
	router.HandleFunc("/economy/pay", s.processQueryPaymentHandler).Methods("POST")
	router.HandleFunc("/economy/reward", s.adminAuthMiddleware(s.distributeQueryRewardHandler)).Methods("POST")

	// Economic Metrics (some already covered by /economy/stats)
	router.HandleFunc("/nodes/{id}/performance", s.getNodePerformanceHandler).Methods("GET")
	router.HandleFunc("/users/{id}/spending", s.getUserSpendingHandler).Methods("GET")

	// Administrative Functions
	router.HandleFunc("/economy/pricing-rules", s.adminAuthMiddleware(s.updatePricingRuleHandler)).Methods("PUT")
	router.HandleFunc("/nodes/{id}/slash", s.adminAuthMiddleware(s.slashNodeHandler)).Methods("POST")
	router.HandleFunc("/tokens/mint", s.adminAuthMiddleware(s.mintTokensHandler)).Methods("POST")
}

// adminAuthMiddleware checks for a valid admin API key
func (s *EconomyService) adminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if AdminAPIKey == "" {
			http.Error(w, "Admin API key not configured", http.StatusInternalServerError)
			return
		}
		if r.Header.Get("X-Admin-Key") != AdminAPIKey {
			http.Error(w, "Unauthorized: Invalid Admin Key", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// writeJSONResponse writes a JSON response to the http.ResponseWriter
func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error writing JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeErrorResponse writes an error response to the http.ResponseWriter
func writeErrorResponse(w http.ResponseWriter, status int, message string, err error) {
	log.Printf("API Error: %s - %v", message, err)
	writeJSONResponse(w, status, map[string]string{"error": message, "details": err.Error()})
}

// createUserAccountRequest represents the request body for creating a user account
type createUserAccountRequest struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// createUserAccountHandler handles requests to create a new user account
// @Summary Create a new user account
// @Description Creates a new user account in the Loreum Network economy.
// @Tags Accounts
// @Accept json
// @Produce json
// @Param account body createUserAccountRequest true "User account details"
// @Success 201 {object} economy.UserAccount
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 409 {object} map[string]string "Account already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /accounts/user [post]
func (s *EconomyService) createUserAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req createUserAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.ID == "" || req.Address == "" {
		writeErrorResponse(w, http.StatusBadRequest, "ID and Address are required", fmt.Errorf("missing fields"))
		return
	}

	account, err := s.Engine.CreateUserAccount(req.ID, req.Address)
	if err != nil {
		if err.Error() == fmt.Sprintf("account with ID %s already exists", req.ID) {
			writeErrorResponse(w, http.StatusConflict, "Account already exists", err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to create user account", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusCreated, account)
}

// createNodeAccountRequest represents the request body for creating a node account
type createNodeAccountRequest struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// createNodeAccountHandler handles requests to create a new node account
// @Summary Create a new node account
// @Description Creates a new node account in the Loreum Network economy.
// @Tags Accounts
// @Accept json
// @Produce json
// @Param account body createNodeAccountRequest true "Node account details"
// @Success 201 {object} economy.NodeAccount
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 409 {object} map[string]string "Account already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /accounts/node [post]
func (s *EconomyService) createNodeAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req createNodeAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.ID == "" || req.Address == "" {
		writeErrorResponse(w, http.StatusBadRequest, "ID and Address are required", fmt.Errorf("missing fields"))
		return
	}

	account, err := s.Engine.CreateNodeAccount(req.ID, req.Address)
	if err != nil {
		if err.Error() == fmt.Sprintf("account with ID %s already exists", req.ID) {
			writeErrorResponse(w, http.StatusConflict, "Account already exists", err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to create node account", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusCreated, account)
}

// getAccountHandler handles requests to get account details
// @Summary Get account details
// @Description Retrieves details for an account by ID.
// @Tags Accounts
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} economy.Account
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /accounts/{id} [get]
func (s *EconomyService) getAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	account, err := s.Engine.GetAccount(id)
	if err != nil {
		if err == economy.ErrAccountNotFound {
			writeErrorResponse(w, http.StatusNotFound, "Account not found", err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to get account", err)
		}
		return
	}

	// Check if it's a node account
	nodeAccount, err := s.Engine.GetNodeAccount(id)
	if err == nil {
		writeJSONResponse(w, http.StatusOK, nodeAccount)
		return
	}

	// Check if it's a user account
	userAccount, err := s.Engine.GetUserAccount(id)
	if err == nil {
		writeJSONResponse(w, http.StatusOK, userAccount)
		return
	}

	// Return basic account if not a specialized type
	writeJSONResponse(w, http.StatusOK, account)
}

// getAccountTransactionsHandler handles requests to get account transactions
// @Summary Get account transactions
// @Description Retrieves transaction history for an account.
// @Tags Accounts
// @Produce json
// @Param id path string true "Account ID"
// @Param limit query int false "Maximum number of transactions to return" default(10)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} economy.Transaction
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /accounts/{id}/transactions [get]
func (s *EconomyService) getAccountTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Parse pagination parameters
	limit := 10
	offset := 0
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		fmt.Sscanf(limitParam, "%d", &limit)
	}
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		fmt.Sscanf(offsetParam, "%d", &offset)
	}

	transactions, err := s.Engine.GetTransactions(id, limit, offset)
	if err != nil {
		if err == economy.ErrAccountNotFound {
			writeErrorResponse(w, http.StatusNotFound, "Account not found", err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to get transactions", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, transactions)
}

// transferTokensRequest represents the request body for transferring tokens
type transferTokensRequest struct {
	FromID      string `json:"from_id"`
	ToID        string `json:"to_id"`
	Amount      string `json:"amount"`
	Description string `json:"description"`
}

// transferTokensHandler handles requests to transfer tokens between accounts
// @Summary Transfer tokens
// @Description Transfers tokens from one account to another.
// @Tags Tokens
// @Accept json
// @Produce json
// @Param transfer body transferTokensRequest true "Transfer details"
// @Success 200 {object} economy.Transaction
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 422 {object} map[string]string "Insufficient balance"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tokens/transfer [post]
func (s *EconomyService) transferTokensHandler(w http.ResponseWriter, r *http.Request) {
	var req transferTokensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.FromID == "" || req.ToID == "" || req.Amount == "" {
		writeErrorResponse(w, http.StatusBadRequest, "FromID, ToID, and Amount are required", fmt.Errorf("missing fields"))
		return
	}

	// Parse amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid amount format", fmt.Errorf("amount must be a valid integer"))
		return
	}

	// Perform transfer
	tx, err := s.Engine.Transfer(req.FromID, req.ToID, amount, req.Description)
	if err != nil {
		switch err {
		case economy.ErrAccountNotFound:
			writeErrorResponse(w, http.StatusNotFound, "Account not found", err)
		case economy.ErrInsufficientBalance:
			writeErrorResponse(w, http.StatusUnprocessableEntity, "Insufficient balance", err)
		default:
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to transfer tokens", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, tx)
}

// stakeTokensRequest represents the request body for staking tokens
type stakeTokensRequest struct {
	Amount string `json:"amount"`
}

// stakeTokensHandler handles requests to stake tokens for a node
// @Summary Stake tokens
// @Description Stakes tokens for a node.
// @Tags Nodes
// @Accept json
// @Produce json
// @Param id path string true "Node ID"
// @Param stake body stakeTokensRequest true "Stake details"
// @Success 200 {object} economy.Transaction
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 404 {object} map[string]string "Node not found"
// @Failure 422 {object} map[string]string "Insufficient balance"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /nodes/{id}/stake [post]
func (s *EconomyService) stakeTokensHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	var req stakeTokensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.Amount == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Amount is required", fmt.Errorf("missing field"))
		return
	}

	// Parse amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid amount format", fmt.Errorf("amount must be a valid integer"))
		return
	}

	// Perform staking
	tx, err := s.Engine.Stake(nodeID, amount)
	if err != nil {
		switch err {
		case economy.ErrAccountNotFound:
			writeErrorResponse(w, http.StatusNotFound, "Node not found", err)
		case economy.ErrInsufficientBalance:
			writeErrorResponse(w, http.StatusUnprocessableEntity, "Insufficient balance", err)
		default:
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to stake tokens", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, tx)
}

// unstakeTokensRequest represents the request body for unstaking tokens
type unstakeTokensRequest struct {
	Amount string `json:"amount"`
}

// unstakeTokensHandler handles requests to unstake tokens for a node
// @Summary Unstake tokens
// @Description Unstakes tokens for a node.
// @Tags Nodes
// @Accept json
// @Produce json
// @Param id path string true "Node ID"
// @Param unstake body unstakeTokensRequest true "Unstake details"
// @Success 200 {object} economy.Transaction
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 404 {object} map[string]string "Node not found"
// @Failure 422 {object} map[string]string "Insufficient stake"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /nodes/{id}/unstake [post]
func (s *EconomyService) unstakeTokensHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	var req unstakeTokensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.Amount == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Amount is required", fmt.Errorf("missing field"))
		return
	}

	// Parse amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid amount format", fmt.Errorf("amount must be a valid integer"))
		return
	}

	// Perform unstaking
	tx, err := s.Engine.Unstake(nodeID, amount)
	if err != nil {
		switch err {
		case economy.ErrAccountNotFound:
			writeErrorResponse(w, http.StatusNotFound, "Node not found", err)
		case economy.ErrInsufficientStake:
			writeErrorResponse(w, http.StatusUnprocessableEntity, "Insufficient stake", err)
		default:
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to unstake tokens", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, tx)
}

// getNetworkStatsHandler handles requests to get network economic statistics
// @Summary Get network statistics
// @Description Retrieves economic statistics for the Loreum Network.
// @Tags Economy
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /economy/stats [get]
func (s *EconomyService) getNetworkStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := s.Engine.GetNetworkStats()
	writeJSONResponse(w, http.StatusOK, stats)
}

// calculateQueryPriceRequest represents the request body for calculating query price
type calculateQueryPriceRequest struct {
	ModelTier string `json:"model_tier"`
	QueryType string `json:"query_type"`
	InputSize int64  `json:"input_size"`
}

// calculateQueryPriceResponse represents the response for a price calculation
type calculateQueryPriceResponse struct {
	Price     string `json:"price"`
	ModelTier string `json:"model_tier"`
	QueryType string `json:"query_type"`
	InputSize int64  `json:"input_size"`
}

// calculateQueryPriceHandler handles requests to calculate query price
// @Summary Calculate query price
// @Description Calculates the price for a query based on model tier, query type, and input size.
// @Tags Economy
// @Accept json
// @Produce json
// @Param price body calculateQueryPriceRequest true "Price calculation parameters"
// @Success 200 {object} calculateQueryPriceResponse
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /economy/price [post]
func (s *EconomyService) calculateQueryPriceHandler(w http.ResponseWriter, r *http.Request) {
	var req calculateQueryPriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.ModelTier == "" || req.QueryType == "" {
		writeErrorResponse(w, http.StatusBadRequest, "ModelTier and QueryType are required", fmt.Errorf("missing fields"))
		return
	}

	// Calculate price
	price, err := s.Engine.CalculateQueryPrice(
		economy.ModelTier(req.ModelTier),
		economy.QueryType(req.QueryType),
		req.InputSize,
	)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to calculate price", err)
		return
	}

	response := calculateQueryPriceResponse{
		Price:     price.String(),
		ModelTier: req.ModelTier,
		QueryType: req.QueryType,
		InputSize: req.InputSize,
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// processQueryPaymentRequest represents the request body for processing a query payment
type processQueryPaymentRequest struct {
	UserID    string `json:"user_id"`
	ModelID   string `json:"model_id"`
	ModelTier string `json:"model_tier"`
	QueryType string `json:"query_type"`
	InputSize int64  `json:"input_size"`
}

// processQueryPaymentHandler handles requests to process a query payment
// @Summary Process query payment
// @Description Processes a payment for a query.
// @Tags Economy
// @Accept json
// @Produce json
// @Param payment body processQueryPaymentRequest true "Payment details"
// @Success 200 {object} economy.Transaction
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 422 {object} map[string]string "Insufficient balance"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /economy/pay [post]
func (s *EconomyService) processQueryPaymentHandler(w http.ResponseWriter, r *http.Request) {
	var req processQueryPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.UserID == "" || req.ModelID == "" || req.ModelTier == "" || req.QueryType == "" {
		writeErrorResponse(w, http.StatusBadRequest, "UserID, ModelID, ModelTier, and QueryType are required", fmt.Errorf("missing fields"))
		return
	}

	// Process payment
	tx, err := s.Engine.ProcessQueryPayment(
		r.Context(),
		req.UserID,
		req.ModelID,
		economy.ModelTier(req.ModelTier),
		economy.QueryType(req.QueryType),
		req.InputSize,
	)
	if err != nil {
		switch err {
		case economy.ErrAccountNotFound:
			writeErrorResponse(w, http.StatusNotFound, "User not found", err)
		case economy.ErrInsufficientBalance:
			writeErrorResponse(w, http.StatusUnprocessableEntity, "Insufficient balance", err)
		default:
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to process payment", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, tx)
}

// distributeQueryRewardRequest represents the request body for distributing a query reward
type distributeQueryRewardRequest struct {
	TransactionID string  `json:"transaction_id"`
	NodeID        string  `json:"node_id"`
	ResponseTime  int64   `json:"response_time_ms"`
	Success       bool    `json:"success"`
	QualityScore  float64 `json:"quality_score"`
}

// distributeQueryRewardHandler handles requests to distribute a query reward
// @Summary Distribute query reward
// @Description Distributes a reward for a processed query.
// @Tags Economy
// @Accept json
// @Produce json
// @Param X-Admin-Key header string true "Admin API Key"
// @Param reward body distributeQueryRewardRequest true "Reward details"
// @Success 200 {object} economy.Transaction
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Transaction or node not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /economy/reward [post]
func (s *EconomyService) distributeQueryRewardHandler(w http.ResponseWriter, r *http.Request) {
	var req distributeQueryRewardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.TransactionID == "" || req.NodeID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "TransactionID and NodeID are required", fmt.Errorf("missing fields"))
		return
	}

	// Distribute reward
	tx, err := s.Engine.DistributeQueryReward(
		r.Context(),
		req.TransactionID,
		req.NodeID,
		req.ResponseTime,
		req.Success,
		req.QualityScore,
	)
	if err != nil {
		if err == economy.ErrAccountNotFound {
			writeErrorResponse(w, http.StatusNotFound, "Node not found", err)
		} else if err.Error() == "payment transaction not found or not in pending state" {
			writeErrorResponse(w, http.StatusNotFound, "Transaction not found", err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to distribute reward", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, tx)
}

// getNodePerformanceHandler handles requests to get node performance and earnings
// @Summary Get node performance
// @Description Retrieves performance metrics and earnings for a node.
// @Tags Nodes
// @Produce json
// @Param id path string true "Node ID"
// @Success 200 {object} economy.NodeAccount
// @Failure 404 {object} map[string]string "Node not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /nodes/{id}/performance [get]
func (s *EconomyService) getNodePerformanceHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	nodeAccount, err := s.Engine.GetNodeAccount(nodeID)
	if err != nil {
		if err == economy.ErrAccountNotFound {
			writeErrorResponse(w, http.StatusNotFound, "Node not found", err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to get node performance", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, nodeAccount)
}

// getUserSpendingHandler handles requests to get user spending patterns
// @Summary Get user spending
// @Description Retrieves spending patterns for a user.
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} economy.UserAccount
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /users/{id}/spending [get]
func (s *EconomyService) getUserSpendingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]

	userAccount, err := s.Engine.GetUserAccount(userID)
	if err != nil {
		if err == economy.ErrAccountNotFound {
			writeErrorResponse(w, http.StatusNotFound, "User not found", err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to get user spending", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, userAccount)
}

// updatePricingRuleRequest represents the request body for updating a pricing rule
type updatePricingRuleRequest struct {
	ModelTier        string `json:"model_tier"`
	QueryType        string `json:"query_type"`
	BasePrice        string `json:"base_price"`
	TokenPerMS       string `json:"token_per_ms"`
	TokenPerKB       string `json:"token_per_kb"`
	MinPrice         string `json:"min_price"`
	MaxPrice         string `json:"max_price"`
	DemandMultiplier string `json:"demand_multiplier"`
}

// updatePricingRuleHandler handles requests to update a pricing rule
// @Summary Update pricing rule
// @Description Updates a pricing rule for a model tier and query type.
// @Tags Economy
// @Accept json
// @Produce json
// @Param X-Admin-Key header string true "Admin API Key"
// @Param rule body updatePricingRuleRequest true "Pricing rule details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /economy/pricing-rules [put]
func (s *EconomyService) updatePricingRuleHandler(w http.ResponseWriter, r *http.Request) {
	var req updatePricingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.ModelTier == "" || req.QueryType == "" || req.BasePrice == "" {
		writeErrorResponse(w, http.StatusBadRequest, "ModelTier, QueryType, and BasePrice are required", fmt.Errorf("missing fields"))
		return
	}

	// Parse big.Int and big.Float values
	basePrice, ok := new(big.Int).SetString(req.BasePrice, 10)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid BasePrice format", fmt.Errorf("must be a valid integer"))
		return
	}

	minPrice, ok := new(big.Int).SetString(req.MinPrice, 10)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid MinPrice format", fmt.Errorf("must be a valid integer"))
		return
	}

	maxPrice, ok := new(big.Int).SetString(req.MaxPrice, 10)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid MaxPrice format", fmt.Errorf("must be a valid integer"))
		return
	}

	tokenPerMS, ok := new(big.Float).SetString(req.TokenPerMS)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid TokenPerMS format", fmt.Errorf("must be a valid float"))
		return
	}

	tokenPerKB, ok := new(big.Float).SetString(req.TokenPerKB)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid TokenPerKB format", fmt.Errorf("must be a valid float"))
		return
	}

	demandMultiplier, ok := new(big.Float).SetString(req.DemandMultiplier)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid DemandMultiplier format", fmt.Errorf("must be a valid float"))
		return
	}

	// Create pricing rule
	rule := &economy.PricingRule{
		ModelTier:        economy.ModelTier(req.ModelTier),
		QueryType:        economy.QueryType(req.QueryType),
		BasePrice:        basePrice,
		TokenPerMS:       tokenPerMS,
		TokenPerKB:       tokenPerKB,
		MinPrice:         minPrice,
		MaxPrice:         maxPrice,
		DemandMultiplier: demandMultiplier,
	}

	// Update pricing rule
	err := s.Engine.UpdatePricingRule(rule)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update pricing rule", err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"status": "success"})
}

// slashNodeRequest represents the request body for slashing a node
type slashNodeRequest struct {
	Reason   string  `json:"reason"`
	Severity float64 `json:"severity"`
}

// slashNodeHandler handles requests to slash a node
// @Summary Slash node
// @Description Penalizes a node for poor performance or malicious behavior.
// @Tags Nodes
// @Accept json
// @Produce json
// @Param X-Admin-Key header string true "Admin API Key"
// @Param id path string true "Node ID"
// @Param slash body slashNodeRequest true "Slash details"
// @Success 200 {object} economy.Transaction
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Node not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /nodes/{id}/slash [post]
func (s *EconomyService) slashNodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeID := vars["id"]

	var req slashNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.Reason == "" || req.Severity <= 0 || req.Severity > 1.0 {
		writeErrorResponse(w, http.StatusBadRequest, "Reason is required and Severity must be between 0 and 1", fmt.Errorf("invalid fields"))
		return
	}

	// Slash node
	tx, err := s.Engine.SlashNode(nodeID, req.Reason, req.Severity)
	if err != nil {
		if err == economy.ErrAccountNotFound {
			writeErrorResponse(w, http.StatusNotFound, "Node not found", err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to slash node", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, tx)
}

// mintTokensRequest represents the request body for minting tokens
type mintTokensRequest struct {
	AccountID string `json:"account_id"`
	Amount    string `json:"amount"`
	Reason    string `json:"reason"`
}

// mintTokensHandler handles requests to mint new tokens
// @Summary Mint tokens
// @Description Creates new tokens and adds them to an account.
// @Tags Tokens
// @Accept json
// @Produce json
// @Param X-Admin-Key header string true "Admin API Key"
// @Param mint body mintTokensRequest true "Mint details"
// @Success 200 {object} economy.Transaction
// @Failure 400 {object} map[string]string "Invalid request payload"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Account not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tokens/mint [post]
func (s *EconomyService) mintTokensHandler(w http.ResponseWriter, r *http.Request) {
	var req mintTokensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.AccountID == "" || req.Amount == "" || req.Reason == "" {
		writeErrorResponse(w, http.StatusBadRequest, "AccountID, Amount, and Reason are required", fmt.Errorf("missing fields"))
		return
	}

	// Parse amount
	amount, ok := new(big.Int).SetString(req.Amount, 10)
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid amount format", fmt.Errorf("amount must be a valid integer"))
		return
	}

	// Mint tokens
	tx, err := s.Engine.MintTokens(req.AccountID, amount, req.Reason)
	if err != nil {
		if err == economy.ErrAccountNotFound {
			writeErrorResponse(w, http.StatusNotFound, "Account not found", err)
		} else {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to mint tokens", err)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, tx)
}
