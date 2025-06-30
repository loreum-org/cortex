package services

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// ServiceRouter handles intelligent routing of queries to appropriate nodes
type ServiceRouter struct {
	registry       *types.ServiceRegistry
	routingPolicy  *types.RoutingPolicy
	nodeMetrics    map[string]*NodeMetrics
	loadBalancer   *LoadBalancer
}

// NodeMetrics tracks performance metrics for each node
type NodeMetrics struct {
	NodeID           string        `json:"node_id"`
	AverageResponse  time.Duration `json:"average_response"`
	SuccessRate      float64       `json:"success_rate"`
	CurrentLoad      int           `json:"current_load"`
	MaxLoad          int           `json:"max_load"`
	ReputationScore  float64       `json:"reputation_score"`
	PriceCompetitive float64       `json:"price_competitive"`
	LastUpdated      time.Time     `json:"last_updated"`
}

// LoadBalancer manages load distribution across nodes
type LoadBalancer struct {
	nodeLoads map[string]int
	maxLoads  map[string]int
}

// RouteResult contains the result of query routing
type RouteResult struct {
	SelectedNodeID string                     `json:"selected_node_id"`
	ServiceID      string                     `json:"service_id"`
	EstimatedCost  string                     `json:"estimated_cost"`
	EstimatedTime  time.Duration              `json:"estimated_time"`
	Confidence     float64                    `json:"confidence"`
	Alternatives   []string                   `json:"alternatives"`
	Reasoning      string                     `json:"reasoning"`
}

// NewServiceRouter creates a new service router
func NewServiceRouter(registry *types.ServiceRegistry) *ServiceRouter {
	return &ServiceRouter{
		registry: registry,
		routingPolicy: &types.RoutingPolicy{
			Strategy:           types.RoutingStrategyWeighted,
			ReputationWeight:   0.3,
			PriceWeight:        0.25,
			ResponseTimeWeight: 0.25,
			LoadWeight:         0.2,
		},
		nodeMetrics: make(map[string]*NodeMetrics),
		loadBalancer: &LoadBalancer{
			nodeLoads: make(map[string]int),
			maxLoads:  make(map[string]int),
		},
	}
}

// RouteQuery routes a query to the best available node
func (sr *ServiceRouter) RouteQuery(query *types.ServiceQuery) (string, error) {
	// Find all nodes that can handle the query
	candidateNodes := sr.findCandidateNodes(query)
	if len(candidateNodes) == 0 {
		return "", fmt.Errorf("no nodes available to handle query requirements")
	}

	// Apply routing strategy
	selectedNode, err := sr.selectBestNode(candidateNodes, query)
	if err != nil {
		return "", fmt.Errorf("failed to select best node: %w", err)
	}

	// Update load balancer
	sr.loadBalancer.incrementLoad(selectedNode)

	return selectedNode, nil
}

// RouteQueryWithDetails routes a query and returns detailed routing information
func (sr *ServiceRouter) RouteQueryWithDetails(query *types.ServiceQuery) (*RouteResult, error) {
	candidateNodes := sr.findCandidateNodes(query)
	if len(candidateNodes) == 0 {
		return nil, fmt.Errorf("no nodes available to handle query requirements")
	}

	// Score all candidates
	scoredNodes := sr.scoreNodes(candidateNodes, query)
	
	// Sort by score (highest first)
	sort.Slice(scoredNodes, func(i, j int) bool {
		return scoredNodes[i].Score > scoredNodes[j].Score
	})

	if len(scoredNodes) == 0 {
		return nil, fmt.Errorf("no suitable nodes found")
	}

	best := scoredNodes[0]
	
	// Create alternatives list
	alternatives := make([]string, 0, min(3, len(scoredNodes)-1))
	for i := 1; i < len(scoredNodes) && i <= 3; i++ {
		alternatives = append(alternatives, scoredNodes[i].NodeID)
	}

	result := &RouteResult{
		SelectedNodeID: best.NodeID,
		ServiceID:      best.ServiceID,
		EstimatedCost:  sr.estimateCost(best.NodeID, query),
		EstimatedTime:  sr.estimateResponseTime(best.NodeID),
		Confidence:     best.Score,
		Alternatives:   alternatives,
		Reasoning:      sr.generateReasoning(best, query),
	}

	// Update load balancer
	sr.loadBalancer.incrementLoad(best.NodeID)

	return result, nil
}

// findCandidateNodes finds all nodes that can handle the query requirements
func (sr *ServiceRouter) findCandidateNodes(query *types.ServiceQuery) []CandidateNode {
	var candidates []CandidateNode

	for _, requirement := range query.RequiredServices {
		// Find services of the required type
		serviceIDs := sr.registry.TypeIndex[requirement.Type]
		
		for _, serviceID := range serviceIDs {
			service := sr.registry.Services[serviceID]
			if service == nil || service.Status != types.ServiceStatusActive {
				continue
			}

			// Check if service meets requirements
			if sr.serviceMatchesRequirement(service, requirement) {
				// Check if node is not overloaded
				if sr.loadBalancer.canHandleMore(service.NodeID) {
					candidates = append(candidates, CandidateNode{
						NodeID:    service.NodeID,
						ServiceID: service.ID,
						Service:   service,
					})
				}
			}
		}
	}

	return sr.deduplicateByNode(candidates)
}

// CandidateNode represents a node that can potentially handle a query
type CandidateNode struct {
	NodeID    string                    `json:"node_id"`
	ServiceID string                    `json:"service_id"`
	Service   *types.ServiceOffering    `json:"service"`
	Score     float64                   `json:"score"`
}

// selectBestNode selects the best node based on the routing policy
func (sr *ServiceRouter) selectBestNode(candidates []CandidateNode, query *types.ServiceQuery) (string, error) {
	switch sr.routingPolicy.Strategy {
	case types.RoutingStrategyRoundRobin:
		return sr.roundRobinSelection(candidates), nil
	case types.RoutingStrategyLowestPrice:
		return sr.lowestPriceSelection(candidates), nil
	case types.RoutingStrategyHighestRep:
		return sr.highestReputationSelection(candidates), nil
	case types.RoutingStrategyFastestResponse:
		return sr.fastestResponseSelection(candidates), nil
	case types.RoutingStrategyWeighted:
		return sr.weightedSelection(candidates, query)
	default:
		return sr.weightedSelection(candidates, query)
	}
}

// scoreNodes scores all candidate nodes based on multiple factors
func (sr *ServiceRouter) scoreNodes(candidates []CandidateNode, query *types.ServiceQuery) []CandidateNode {
	for i := range candidates {
		candidates[i].Score = sr.calculateNodeScore(candidates[i], query)
	}
	return candidates
}

// calculateNodeScore calculates a comprehensive score for a node
func (sr *ServiceRouter) calculateNodeScore(candidate CandidateNode, query *types.ServiceQuery) float64 {
	metrics := sr.getNodeMetrics(candidate.NodeID)
	
	// Reputation score (0-1)
	reputationScore := metrics.ReputationScore / 10.0 // Assuming reputation is 0-10
	
	// Price score (inverse, lower price = higher score)
	priceScore := sr.calculatePriceScore(candidate.Service, query)
	
	// Response time score (inverse, faster = higher score)
	responseScore := sr.calculateResponseTimeScore(metrics)
	
	// Load score (inverse, lower load = higher score)
	loadScore := sr.calculateLoadScore(metrics)
	
	// Weighted combination
	totalScore := 
		reputationScore * sr.routingPolicy.ReputationWeight +
		priceScore * sr.routingPolicy.PriceWeight +
		responseScore * sr.routingPolicy.ResponseTimeWeight +
		loadScore * sr.routingPolicy.LoadWeight

	return totalScore
}

// Routing strategy implementations

func (sr *ServiceRouter) roundRobinSelection(candidates []CandidateNode) string {
	if len(candidates) == 0 {
		return ""
	}
	// Simple round-robin (could be improved with proper state tracking)
	index := rand.Intn(len(candidates))
	return candidates[index].NodeID
}

func (sr *ServiceRouter) lowestPriceSelection(candidates []CandidateNode) string {
	if len(candidates) == 0 {
		return ""
	}
	
	bestNode := candidates[0]
	bestPrice := sr.parsePrice(bestNode.Service.Pricing.BasePrice)
	
	for _, candidate := range candidates[1:] {
		price := sr.parsePrice(candidate.Service.Pricing.BasePrice)
		if price < bestPrice {
			bestPrice = price
			bestNode = candidate
		}
	}
	
	return bestNode.NodeID
}

func (sr *ServiceRouter) highestReputationSelection(candidates []CandidateNode) string {
	if len(candidates) == 0 {
		return ""
	}
	
	bestNode := candidates[0]
	bestReputation := sr.getNodeMetrics(bestNode.NodeID).ReputationScore
	
	for _, candidate := range candidates[1:] {
		reputation := sr.getNodeMetrics(candidate.NodeID).ReputationScore
		if reputation > bestReputation {
			bestReputation = reputation
			bestNode = candidate
		}
	}
	
	return bestNode.NodeID
}

func (sr *ServiceRouter) fastestResponseSelection(candidates []CandidateNode) string {
	if len(candidates) == 0 {
		return ""
	}
	
	bestNode := candidates[0]
	bestResponse := sr.getNodeMetrics(bestNode.NodeID).AverageResponse
	
	for _, candidate := range candidates[1:] {
		response := sr.getNodeMetrics(candidate.NodeID).AverageResponse
		if response < bestResponse {
			bestResponse = response
			bestNode = candidate
		}
	}
	
	return bestNode.NodeID
}

func (sr *ServiceRouter) weightedSelection(candidates []CandidateNode, query *types.ServiceQuery) (string, error) {
	scoredNodes := sr.scoreNodes(candidates, query)
	
	if len(scoredNodes) == 0 {
		return "", fmt.Errorf("no nodes available after scoring")
	}
	
	// Sort by score and return the best
	sort.Slice(scoredNodes, func(i, j int) bool {
		return scoredNodes[i].Score > scoredNodes[j].Score
	})
	
	return scoredNodes[0].NodeID, nil
}

// Helper methods

func (sr *ServiceRouter) serviceMatchesRequirement(service *types.ServiceOffering, requirement types.ServiceRequirement) bool {
	// Check type
	if service.Type != requirement.Type {
		return false
	}

	// Check capabilities
	if len(requirement.Capabilities) > 0 {
		for _, reqCap := range requirement.Capabilities {
			found := false
			for _, serviceCap := range service.Capabilities {
				if serviceCap.Name == reqCap {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check version if specified
	if requirement.MinVersion != "" {
		// Version comparison logic would go here
		// For now, assume compatible
	}

	return true
}

func (sr *ServiceRouter) deduplicateByNode(candidates []CandidateNode) []CandidateNode {
	seen := make(map[string]bool)
	var result []CandidateNode
	
	for _, candidate := range candidates {
		if !seen[candidate.NodeID] {
			seen[candidate.NodeID] = true
			result = append(result, candidate)
		}
	}
	
	return result
}

func (sr *ServiceRouter) getNodeMetrics(nodeID string) *NodeMetrics {
	if metrics, exists := sr.nodeMetrics[nodeID]; exists {
		return metrics
	}
	
	// Return default metrics if not found
	return &NodeMetrics{
		NodeID:          nodeID,
		AverageResponse: 1 * time.Second,
		SuccessRate:     0.95,
		CurrentLoad:     0,
		MaxLoad:         100,
		ReputationScore: 5.0,
		LastUpdated:     time.Now(),
	}
}

func (sr *ServiceRouter) calculatePriceScore(service *types.ServiceOffering, query *types.ServiceQuery) float64 {
	if service.Pricing == nil {
		return 0.5 // Default neutral score
	}
	
	basePrice := sr.parsePrice(service.Pricing.BasePrice)
	if basePrice == 0 {
		return 1.0 // Free service gets highest price score
	}
	
	// Normalize price score (this is simplified)
	// In reality, you'd want to compare against market rates
	maxPrice := sr.parsePrice(query.MaxPrice)
	if maxPrice > 0 && basePrice <= maxPrice {
		return 1.0 - (basePrice / maxPrice)
	}
	
	return 0.5 // Default if no max price specified
}

func (sr *ServiceRouter) calculateResponseTimeScore(metrics *NodeMetrics) float64 {
	// Convert response time to score (inverse relationship)
	// Faster response = higher score
	responseMs := float64(metrics.AverageResponse.Milliseconds())
	if responseMs <= 100 {
		return 1.0
	} else if responseMs <= 1000 {
		return 1.0 - (responseMs-100)/900*0.5
	} else {
		return math.Max(0.1, 0.5 - (responseMs-1000)/10000*0.4)
	}
}

func (sr *ServiceRouter) calculateLoadScore(metrics *NodeMetrics) float64 {
	if metrics.MaxLoad == 0 {
		return 1.0
	}
	
	loadRatio := float64(metrics.CurrentLoad) / float64(metrics.MaxLoad)
	return math.Max(0.1, 1.0 - loadRatio)
}

func (sr *ServiceRouter) parsePrice(priceStr string) float64 {
	// Simple price parsing - in reality would handle big.Int
	// For now, just return a mock value
	if priceStr == "" || priceStr == "0" {
		return 0
	}
	return 100.0 // Mock price
}

func (sr *ServiceRouter) estimateCost(nodeID string, query *types.ServiceQuery) string {
	// Simplified cost estimation
	return "1000000000000000000" // 1 token
}

func (sr *ServiceRouter) estimateResponseTime(nodeID string) time.Duration {
	metrics := sr.getNodeMetrics(nodeID)
	return metrics.AverageResponse
}

func (sr *ServiceRouter) generateReasoning(selected CandidateNode, query *types.ServiceQuery) string {
	return fmt.Sprintf("Selected node %s based on weighted scoring: reputation %.2f, load %d/%d, estimated response time %v",
		selected.NodeID, 
		sr.getNodeMetrics(selected.NodeID).ReputationScore,
		sr.getNodeMetrics(selected.NodeID).CurrentLoad,
		sr.getNodeMetrics(selected.NodeID).MaxLoad,
		sr.getNodeMetrics(selected.NodeID).AverageResponse,
	)
}

// LoadBalancer methods

func (lb *LoadBalancer) canHandleMore(nodeID string) bool {
	currentLoad := lb.nodeLoads[nodeID]
	maxLoad := lb.maxLoads[nodeID]
	if maxLoad == 0 {
		maxLoad = 100 // Default max load
	}
	return currentLoad < maxLoad
}

func (lb *LoadBalancer) incrementLoad(nodeID string) {
	lb.nodeLoads[nodeID]++
}

func (lb *LoadBalancer) decrementLoad(nodeID string) {
	if lb.nodeLoads[nodeID] > 0 {
		lb.nodeLoads[nodeID]--
	}
}

func (lb *LoadBalancer) setMaxLoad(nodeID string, maxLoad int) {
	lb.maxLoads[nodeID] = maxLoad
}

// UpdateNodeMetrics updates performance metrics for a node
func (sr *ServiceRouter) UpdateNodeMetrics(nodeID string, metrics *NodeMetrics) {
	sr.nodeMetrics[nodeID] = metrics
}

// SetRoutingPolicy updates the routing policy
func (sr *ServiceRouter) SetRoutingPolicy(policy *types.RoutingPolicy) {
	sr.routingPolicy = policy
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}