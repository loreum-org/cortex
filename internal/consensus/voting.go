package consensus

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log"
	"sync"
	"time"
)

// Vote represents a vote on a transaction
type Vote struct {
	TransactionID string            `json:"transaction_id"`
	VoterID       string            `json:"voter_id"`
	VoterPubKey   ed25519.PublicKey `json:"voter_pub_key"`
	VoteType      VoteType          `json:"vote_type"`
	Signature     []byte            `json:"signature"`
	Timestamp     time.Time         `json:"timestamp"`
	Reputation    float64           `json:"reputation"`
}

// VoteType represents the type of vote
type VoteType string

const (
	VoteTypeApprove VoteType = "approve"
	VoteTypeReject  VoteType = "reject"
	VoteTypeAbstain VoteType = "abstain"
)

// VotingManager manages the voting process for transactions
type VotingManager struct {
	votes           map[string][]*Vote // TransactionID -> list of votes
	votingThreshold float64            // Threshold for finalization (e.g., 0.67 for 2/3 majority)
	votingTimeout   time.Duration      // How long to wait for votes
	reputationMgr   *ReputationManager
	mu              sync.RWMutex
}

// VotingResult represents the result of voting on a transaction
type VotingResult struct {
	TransactionID    string    `json:"transaction_id"`
	TotalWeight      float64   `json:"total_weight"`
	ApprovalWeight   float64   `json:"approval_weight"`
	RejectionWeight  float64   `json:"rejection_weight"`
	AbstentionWeight float64   `json:"abstention_weight"`
	IsFinalized      bool      `json:"is_finalized"`
	Result           VoteType  `json:"result"`
	Timestamp        time.Time `json:"timestamp"`
}

// NewVotingManager creates a new voting manager
func NewVotingManager(reputationMgr *ReputationManager) *VotingManager {
	return &VotingManager{
		votes:           make(map[string][]*Vote),
		votingThreshold: 0.67, // 2/3 majority
		votingTimeout:   30 * time.Second,
		reputationMgr:   reputationMgr,
	}
}

// SubmitVote submits a vote for a transaction
func (vm *VotingManager) SubmitVote(vote *Vote) error {
	// Validate vote signature
	if !vm.validateVoteSignature(vote) {
		return fmt.Errorf("invalid vote signature")
	}

	// Get voter reputation
	reputation := vm.reputationMgr.GetScore(vote.VoterID)
	if reputation <= 0 {
		return fmt.Errorf("voter has no reputation")
	}
	vote.Reputation = reputation

	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Check if voter has already voted
	if vm.hasVoted(vote.TransactionID, vote.VoterID) {
		return fmt.Errorf("voter has already voted on this transaction")
	}

	// Add the vote
	vm.votes[vote.TransactionID] = append(vm.votes[vote.TransactionID], vote)

	log.Printf("Vote submitted: %s voted %s on transaction %s (reputation: %.2f)",
		vote.VoterID, vote.VoteType, vote.TransactionID, vote.Reputation)

	return nil
}

// hasVoted checks if a voter has already voted on a transaction
func (vm *VotingManager) hasVoted(transactionID, voterID string) bool {
	votes := vm.votes[transactionID]
	for _, vote := range votes {
		if vote.VoterID == voterID {
			return true
		}
	}
	return false
}

// GetVotingResult calculates the voting result for a transaction
func (vm *VotingManager) GetVotingResult(transactionID string) *VotingResult {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	votes := vm.votes[transactionID]
	if len(votes) == 0 {
		return &VotingResult{
			TransactionID: transactionID,
			Result:        VoteTypeAbstain,
			Timestamp:     time.Now(),
		}
	}

	var totalWeight, approvalWeight, rejectionWeight, abstentionWeight float64

	for _, vote := range votes {
		weight := vote.Reputation
		totalWeight += weight

		switch vote.VoteType {
		case VoteTypeApprove:
			approvalWeight += weight
		case VoteTypeReject:
			rejectionWeight += weight
		case VoteTypeAbstain:
			abstentionWeight += weight
		}
	}

	// Determine result
	var result VoteType
	var isFinalized bool

	if totalWeight > 0 {
		approvalRatio := approvalWeight / totalWeight
		rejectionRatio := rejectionWeight / totalWeight

		if approvalRatio >= vm.votingThreshold {
			result = VoteTypeApprove
			isFinalized = true
		} else if rejectionRatio >= vm.votingThreshold {
			result = VoteTypeReject
			isFinalized = true
		} else {
			result = VoteTypeAbstain
			isFinalized = false
		}
	}

	return &VotingResult{
		TransactionID:    transactionID,
		TotalWeight:      totalWeight,
		ApprovalWeight:   approvalWeight,
		RejectionWeight:  rejectionWeight,
		AbstentionWeight: abstentionWeight,
		IsFinalized:      isFinalized,
		Result:           result,
		Timestamp:        time.Now(),
	}
}

// IsTransactionFinalized checks if a transaction has enough votes to be finalized
func (vm *VotingManager) IsTransactionFinalized(transactionID string) bool {
	result := vm.GetVotingResult(transactionID)
	return result.IsFinalized
}

// CreateVote creates and signs a vote
func (vm *VotingManager) CreateVote(transactionID, voterID string, voterPubKey ed25519.PublicKey,
	voterPrivKey ed25519.PrivateKey, voteType VoteType) (*Vote, error) {

	vote := &Vote{
		TransactionID: transactionID,
		VoterID:       voterID,
		VoterPubKey:   voterPubKey,
		VoteType:      voteType,
		Timestamp:     time.Now(),
	}

	// Create signature
	voteData := fmt.Sprintf("%s:%s:%s:%d", transactionID, voterID, voteType, vote.Timestamp.Unix())
	signature := ed25519.Sign(voterPrivKey, []byte(voteData))
	vote.Signature = signature

	return vote, nil
}

// validateVoteSignature validates a vote's signature
func (vm *VotingManager) validateVoteSignature(vote *Vote) bool {
	voteData := fmt.Sprintf("%s:%s:%s:%d", vote.TransactionID, vote.VoterID,
		vote.VoteType, vote.Timestamp.Unix())
	return ed25519.Verify(vote.VoterPubKey, []byte(voteData), vote.Signature)
}

// StartVotingRound starts a voting round for a transaction
func (vm *VotingManager) StartVotingRound(ctx context.Context, transactionID string,
	voters []string, broadcaster VoteBroadcaster) {

	// Wait for votes with timeout
	timeout := time.NewTimer(vm.votingTimeout)
	defer timeout.Stop()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout.C:
			log.Printf("Voting timeout for transaction %s", transactionID)
			return
		case <-ticker.C:
			result := vm.GetVotingResult(transactionID)
			if result.IsFinalized {
				log.Printf("Transaction %s finalized with result: %s (approval: %.2f%%)",
					transactionID, result.Result, (result.ApprovalWeight/result.TotalWeight)*100)

				// Broadcast finalization
				if broadcaster != nil {
					broadcaster.BroadcastFinalization(transactionID, result)
				}
				return
			}
		}
	}
}

// VoteBroadcaster interface for broadcasting votes and finalization
type VoteBroadcaster interface {
	BroadcastVote(vote *Vote) error
	BroadcastFinalization(transactionID string, result *VotingResult) error
}

// GetVotes returns all votes for a transaction
func (vm *VotingManager) GetVotes(transactionID string) []*Vote {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	votes := vm.votes[transactionID]
	result := make([]*Vote, len(votes))
	copy(result, votes)
	return result
}

// SetVotingThreshold sets the voting threshold
func (vm *VotingManager) SetVotingThreshold(threshold float64) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.votingThreshold = threshold
}

// SetVotingTimeout sets the voting timeout
func (vm *VotingManager) SetVotingTimeout(timeout time.Duration) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.votingTimeout = timeout
}

// CleanupOldVotes removes votes for finalized transactions older than the specified duration
func (vm *VotingManager) CleanupOldVotes(olderThan time.Duration) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	for transactionID, votes := range vm.votes {
		if len(votes) > 0 && votes[0].Timestamp.Before(cutoff) {
			// Check if transaction is finalized
			result := vm.GetVotingResult(transactionID)
			if result.IsFinalized {
				delete(vm.votes, transactionID)
				log.Printf("Cleaned up votes for finalized transaction %s", transactionID)
			}
		}
	}
}
