package consensus

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// ByzantineFaultTolerance implements Byzantine fault tolerant consensus
type ByzantineFaultTolerance struct {
	ConsensusService *ConsensusService
	VotingManager    *VotingManager
	NetworkSize      int     // Total number of nodes in the network
	FaultTolerance   int     // Maximum number of Byzantine nodes that can be tolerated (f)
	MinVoters        int     // Minimum number of voters required for consensus (2f + 1)
	SafetyThreshold  float64 // Threshold for safety (typically 2/3)
	LivenessTimeout  time.Duration
	mu               sync.RWMutex
	activePeers      map[string]*PeerInfo
	suspiciousPeers  map[string]*SuspicionInfo
}

// PeerInfo represents information about a peer node
type PeerInfo struct {
	NodeID       string    `json:"node_id"`
	PublicKey    []byte    `json:"public_key"`
	Reputation   float64   `json:"reputation"`
	LastSeen     time.Time `json:"last_seen"`
	MessageCount int64     `json:"message_count"`
	IsOnline     bool      `json:"is_online"`
	VoteHistory  []string  `json:"vote_history"` // Recent vote transaction IDs
}

// SuspicionInfo tracks suspicious behavior from peers
type SuspicionInfo struct {
	NodeID             string    `json:"node_id"`
	ConflictingVotes   int       `json:"conflicting_votes"`
	InvalidSignatures  int       `json:"invalid_signatures"`
	TimeoutViolations  int       `json:"timeout_violations"`
	FirstSuspicionTime time.Time `json:"first_suspicion_time"`
	LastSuspicionTime  time.Time `json:"last_suspicion_time"`
	SuspicionLevel     float64   `json:"suspicion_level"` // 0.0 to 1.0
}

// NewByzantineFaultTolerance creates a new Byzantine fault tolerance manager
func NewByzantineFaultTolerance(cs *ConsensusService, vm *VotingManager, networkSize int) *ByzantineFaultTolerance {
	// Calculate Byzantine fault tolerance parameters
	// In a network of n nodes, we can tolerate at most f = floor((n-1)/3) Byzantine nodes
	faultTolerance := (networkSize - 1) / 3
	minVoters := 2*faultTolerance + 1

	return &ByzantineFaultTolerance{
		ConsensusService: cs,
		VotingManager:    vm,
		NetworkSize:      networkSize,
		FaultTolerance:   faultTolerance,
		MinVoters:        minVoters,
		SafetyThreshold:  2.0 / 3.0, // Byzantine fault tolerant threshold
		LivenessTimeout:  30 * time.Second,
		activePeers:      make(map[string]*PeerInfo),
		suspiciousPeers:  make(map[string]*SuspicionInfo),
	}
}

// ProcessTransactionWithBFT processes a transaction with Byzantine fault tolerance
func (bft *ByzantineFaultTolerance) ProcessTransactionWithBFT(ctx context.Context, tx *types.Transaction, voters []string) error {
	log.Printf("Processing transaction %s with BFT consensus", tx.ID)

	// Step 1: Pre-vote phase - validate transaction
	if !bft.preVoteValidation(tx) {
		return fmt.Errorf("transaction %s failed pre-vote validation", tx.ID)
	}

	// Step 2: Add transaction to DAG (tentatively)
	bft.ConsensusService.lock.Lock()
	node := bft.ConsensusService.DAG.AddNode(tx)
	bft.ConsensusService.lock.Unlock()

	// Step 3: Initiate voting round with Byzantine fault tolerance
	return bft.runByzantineVotingRound(ctx, tx.ID, voters, node)
}

// preVoteValidation performs pre-vote validation to filter out obviously invalid transactions
func (bft *ByzantineFaultTolerance) preVoteValidation(tx *types.Transaction) bool {
	// Run existing validation rules
	for _, rule := range bft.ConsensusService.ValidationRules {
		if !rule(tx) {
			log.Printf("Transaction %s failed validation rule", tx.ID)
			return false
		}
	}

	// Additional Byzantine-specific validations
	return bft.validateTransactionStructure(tx) && bft.checkForConflicts(tx)
}

// validateTransactionStructure validates the basic structure of a transaction
func (bft *ByzantineFaultTolerance) validateTransactionStructure(tx *types.Transaction) bool {
	// Check required fields
	if tx.ID == "" || tx.Signature == nil || tx.Timestamp == 0 {
		log.Printf("Transaction %s has invalid structure", tx.ID)
		return false
	}

	// Check timestamp is reasonable (not too far in future or past)
	now := time.Now().Unix()
	if tx.Timestamp > now+300 || tx.Timestamp < now-3600 { // 5 min future, 1 hour past
		log.Printf("Transaction %s has invalid timestamp", tx.ID)
		return false
	}

	return true
}

// checkForConflicts checks if the transaction conflicts with existing transactions
func (bft *ByzantineFaultTolerance) checkForConflicts(tx *types.Transaction) bool {
	// For economic transactions, check for double spending
	if tx.Type == types.TransactionTypeEconomic && tx.EconomicData != nil {
		return bft.checkDoubleSpending(tx)
	}
	return true
}

// checkDoubleSpending checks for double spending in economic transactions
func (bft *ByzantineFaultTolerance) checkDoubleSpending(tx *types.Transaction) bool {
	bft.ConsensusService.lock.RLock()
	defer bft.ConsensusService.lock.RUnlock()

	fromAccount := tx.EconomicData.FromAccount
	if fromAccount == "" {
		return true // Minting transactions don't have double spending risk
	}

	// Check all pending transactions for conflicts
	for _, node := range bft.ConsensusService.DAG.Nodes {
		if node.Transaction.ID == tx.ID {
			continue
		}

		if !node.Transaction.Finalized &&
			node.Transaction.Type == types.TransactionTypeEconomic &&
			node.Transaction.EconomicData != nil &&
			node.Transaction.EconomicData.FromAccount == fromAccount {

			// Potential double spending - need to check amounts and account balance
			log.Printf("Potential double spending detected for account %s between transactions %s and %s",
				fromAccount, tx.ID, node.Transaction.ID)
			return false
		}
	}

	return true
}

// runByzantineVotingRound runs a Byzantine fault tolerant voting round
func (bft *ByzantineFaultTolerance) runByzantineVotingRound(ctx context.Context, transactionID string, voters []string, node *types.DAGNode) error {
	log.Printf("Starting Byzantine voting round for transaction %s with %d voters", transactionID, len(voters))

	// Validate we have enough voters for Byzantine fault tolerance
	if len(voters) < bft.MinVoters {
		return fmt.Errorf("insufficient voters for Byzantine consensus: need %d, got %d", bft.MinVoters, len(voters))
	}

	// Start voting round with timeout
	votingCtx, cancel := context.WithTimeout(ctx, bft.LivenessTimeout)
	defer cancel()

	// Monitor voting progress
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-votingCtx.Done():
			return bft.handleVotingTimeout(transactionID, startTime)

		case <-ticker.C:
			result := bft.VotingManager.GetVotingResult(transactionID)

			// Check if we have Byzantine fault tolerant consensus
			if bft.hasByzantineConsensus(result, voters) {
				return bft.finalizeTransaction(transactionID, result, node)
			}

			// Check for Byzantine faults
			bft.detectByzantineFaults(transactionID, voters)
		}
	}
}

// hasByzantineConsensus checks if the voting result achieves Byzantine fault tolerant consensus
func (bft *ByzantineFaultTolerance) hasByzantineConsensus(result *VotingResult, voters []string) bool {
	if result.TotalWeight == 0 {
		return false
	}

	// Calculate total possible weight from honest voters
	totalPossibleWeight := bft.calculateTotalVoterWeight(voters)

	// For Byzantine fault tolerance, we need more than 2/3 of total possible weight
	requiredWeight := totalPossibleWeight * bft.SafetyThreshold

	log.Printf("BFT consensus check: approval=%.2f, required=%.2f, total=%.2f",
		result.ApprovalWeight, requiredWeight, totalPossibleWeight)

	return result.ApprovalWeight > requiredWeight
}

// calculateTotalVoterWeight calculates the total possible weight from a list of voters
func (bft *ByzantineFaultTolerance) calculateTotalVoterWeight(voters []string) float64 {
	var totalWeight float64
	for _, voterID := range voters {
		reputation := bft.ConsensusService.ReputationManager.GetScore(voterID)
		if reputation <= 0 {
			reputation = 1.0 // Default reputation for new nodes
		}
		totalWeight += reputation
	}
	return totalWeight
}

// detectByzantineFaults detects potential Byzantine faults in voting behavior
func (bft *ByzantineFaultTolerance) detectByzantineFaults(transactionID string, voters []string) {
	votes := bft.VotingManager.GetVotes(transactionID)

	// Track voting patterns
	votesByVoter := make(map[string][]*Vote)
	for _, vote := range votes {
		votesByVoter[vote.VoterID] = append(votesByVoter[vote.VoterID], vote)
	}

	// Check for conflicting votes from the same voter
	for voterID, voterVotes := range votesByVoter {
		if len(voterVotes) > 1 {
			bft.reportSuspiciousBehavior(voterID, "multiple_votes",
				fmt.Sprintf("Multiple votes on transaction %s", transactionID))
		}

		// Check for invalid signatures
		for _, vote := range voterVotes {
			if !bft.VotingManager.validateVoteSignature(vote) {
				bft.reportSuspiciousBehavior(voterID, "invalid_signature",
					fmt.Sprintf("Invalid signature on transaction %s", transactionID))
			}
		}
	}

	// Check for voters who haven't voted within reasonable time
	bft.checkForNonResponsiveVoters(transactionID, voters)
}

// reportSuspiciousBehavior reports suspicious behavior from a peer
func (bft *ByzantineFaultTolerance) reportSuspiciousBehavior(nodeID, behaviorType, details string) {
	bft.mu.Lock()
	defer bft.mu.Unlock()

	suspicion, exists := bft.suspiciousPeers[nodeID]
	if !exists {
		suspicion = &SuspicionInfo{
			NodeID:             nodeID,
			FirstSuspicionTime: time.Now(),
		}
		bft.suspiciousPeers[nodeID] = suspicion
	}

	suspicion.LastSuspicionTime = time.Now()

	switch behaviorType {
	case "multiple_votes":
		suspicion.ConflictingVotes++
	case "invalid_signature":
		suspicion.InvalidSignatures++
	case "timeout":
		suspicion.TimeoutViolations++
	}

	// Calculate suspicion level
	suspicion.SuspicionLevel = bft.calculateSuspicionLevel(suspicion)

	log.Printf("Suspicious behavior detected: Node %s - %s (%s). Suspicion level: %.2f",
		nodeID, behaviorType, details, suspicion.SuspicionLevel)

	// If suspicion level is too high, reduce reputation
	if suspicion.SuspicionLevel > 0.7 {
		currentRep := bft.ConsensusService.ReputationManager.GetScore(nodeID)
		newRep := math.Max(0.1, currentRep*0.8) // Reduce by 20%, minimum 0.1
		bft.ConsensusService.ReputationManager.SetScore(nodeID, newRep)
		log.Printf("Reduced reputation for suspicious node %s: %.2f -> %.2f", nodeID, currentRep, newRep)
	}
}

// calculateSuspicionLevel calculates the suspicion level for a node
func (bft *ByzantineFaultTolerance) calculateSuspicionLevel(suspicion *SuspicionInfo) float64 {
	// Weight different types of suspicious behavior
	score := float64(suspicion.ConflictingVotes)*0.4 +
		float64(suspicion.InvalidSignatures)*0.3 +
		float64(suspicion.TimeoutViolations)*0.1

	// Time decay - older suspicions matter less
	daysSince := time.Since(suspicion.FirstSuspicionTime).Hours() / 24
	if daysSince > 1 {
		score *= math.Exp(-daysSince / 7) // Exponential decay over a week
	}

	return math.Min(1.0, score/10.0) // Normalize to 0-1 range
}

// checkForNonResponsiveVoters checks for voters who haven't responded in time
func (bft *ByzantineFaultTolerance) checkForNonResponsiveVoters(transactionID string, voters []string) {
	votes := bft.VotingManager.GetVotes(transactionID)
	votedNodes := make(map[string]bool)

	for _, vote := range votes {
		votedNodes[vote.VoterID] = true
	}

	for _, voterID := range voters {
		if !votedNodes[voterID] {
			// Check if this voter has been consistently non-responsive
			bft.trackNonResponsiveVoter(voterID, transactionID)
		}
	}
}

// trackNonResponsiveVoter tracks non-responsive voters
func (bft *ByzantineFaultTolerance) trackNonResponsiveVoter(voterID, transactionID string) {
	bft.mu.Lock()
	defer bft.mu.Unlock()

	peer, exists := bft.activePeers[voterID]
	if exists && time.Since(peer.LastSeen) > bft.LivenessTimeout/2 {
		bft.reportSuspiciousBehavior(voterID, "timeout",
			fmt.Sprintf("Non-responsive on transaction %s", transactionID))
	}
}

// finalizeTransaction finalizes a transaction after Byzantine consensus is reached
func (bft *ByzantineFaultTolerance) finalizeTransaction(transactionID string, result *VotingResult, node *types.DAGNode) error {
	bft.ConsensusService.lock.Lock()
	defer bft.ConsensusService.lock.Unlock()

	if result.Result == VoteTypeApprove {
		node.Transaction.Finalized = true
		node.Score = result.ApprovalWeight / result.TotalWeight

		log.Printf("Transaction %s finalized with Byzantine consensus (approval: %.2f%%, weight: %.2f)",
			transactionID, (result.ApprovalWeight/result.TotalWeight)*100, result.TotalWeight)

		// Reward honest voters who voted correctly
		bft.rewardHonestVoters(transactionID, result)

		return nil
	} else {
		// Transaction rejected by Byzantine consensus
		log.Printf("Transaction %s rejected by Byzantine consensus (rejection: %.2f%%)",
			transactionID, (result.RejectionWeight/result.TotalWeight)*100)

		// Remove from DAG
		delete(bft.ConsensusService.DAG.Nodes, transactionID)
		return fmt.Errorf("transaction rejected by Byzantine consensus")
	}
}

// rewardHonestVoters increases reputation of voters who voted with the consensus
func (bft *ByzantineFaultTolerance) rewardHonestVoters(transactionID string, result *VotingResult) {
	votes := bft.VotingManager.GetVotes(transactionID)

	for _, vote := range votes {
		if vote.VoteType == result.Result {
			// Increase reputation for honest voting
			currentRep := bft.ConsensusService.ReputationManager.GetScore(vote.VoterID)
			newRep := math.Min(10.0, currentRep*1.05) // Increase by 5%, max 10.0
			bft.ConsensusService.ReputationManager.SetScore(vote.VoterID, newRep)
		}
	}
}

// handleVotingTimeout handles the case when voting times out
func (bft *ByzantineFaultTolerance) handleVotingTimeout(transactionID string, startTime time.Time) error {
	duration := time.Since(startTime)
	log.Printf("Voting timeout for transaction %s after %v", transactionID, duration)

	// Check if we have any partial consensus
	result := bft.VotingManager.GetVotingResult(transactionID)

	if result.TotalWeight > 0 {
		// We have some votes, check if it's a clear rejection
		rejectionRatio := result.RejectionWeight / result.TotalWeight
		if rejectionRatio > bft.SafetyThreshold {
			log.Printf("Transaction %s rejected due to timeout with clear rejection (%.2f%%)",
				transactionID, rejectionRatio*100)

			// Remove from DAG
			bft.ConsensusService.lock.Lock()
			delete(bft.ConsensusService.DAG.Nodes, transactionID)
			bft.ConsensusService.lock.Unlock()

			return fmt.Errorf("transaction rejected due to timeout")
		}
	}

	// Timeout without clear consensus - keep transaction pending
	log.Printf("Transaction %s remains pending due to timeout without clear consensus", transactionID)
	return fmt.Errorf("voting timeout without consensus")
}

// UpdatePeerInfo updates information about an active peer
func (bft *ByzantineFaultTolerance) UpdatePeerInfo(nodeID string, publicKey []byte, isOnline bool) {
	bft.mu.Lock()
	defer bft.mu.Unlock()

	peer, exists := bft.activePeers[nodeID]
	if !exists {
		peer = &PeerInfo{
			NodeID:      nodeID,
			PublicKey:   publicKey,
			Reputation:  1.0, // Default reputation
			VoteHistory: make([]string, 0),
		}
		bft.activePeers[nodeID] = peer
	}

	peer.LastSeen = time.Now()
	peer.IsOnline = isOnline
	peer.MessageCount++
}

// GetActivePeers returns the list of active peers
func (bft *ByzantineFaultTolerance) GetActivePeers() map[string]*PeerInfo {
	bft.mu.RLock()
	defer bft.mu.RUnlock()

	peers := make(map[string]*PeerInfo)
	for id, peer := range bft.activePeers {
		peers[id] = peer
	}
	return peers
}

// GetSuspiciousPeers returns the list of suspicious peers
func (bft *ByzantineFaultTolerance) GetSuspiciousPeers() map[string]*SuspicionInfo {
	bft.mu.RLock()
	defer bft.mu.RUnlock()

	suspicious := make(map[string]*SuspicionInfo)
	for id, suspicion := range bft.suspiciousPeers {
		suspicious[id] = suspicion
	}
	return suspicious
}
