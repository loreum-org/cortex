package consensus

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// CreateTransaction creates a new transaction with the given data and parents
func CreateTransaction(data string, parentIDs []string, privKey ed25519.PrivateKey) (*types.Transaction, error) {
	// Create a timestamp
	timestamp := time.Now().Unix()

	// Create a hash of the data and timestamp
	dataToHash := data + strconv.FormatInt(timestamp, 10) + strings.Join(parentIDs, "")
	hash := sha256.Sum256([]byte(dataToHash))

	// Sign the hash with the private key
	signature := ed25519.Sign(privKey, hash[:])

	// Create the transaction
	tx := &types.Transaction{
		ID:        hex.EncodeToString(hash[:]),
		Timestamp: timestamp,
		Data:      data,
		ParentIDs: parentIDs,
		Signature: signature,
		Finalized: false,
		Metadata:  make(map[string]string), // Initialize empty metadata
	}

	return tx, nil
}

// ValidateTransaction validates a transaction
func ValidateTransaction(tx *types.Transaction, pubKey ed25519.PublicKey) bool {
	// Recreate the hash
	dataToHash := tx.Data + strconv.FormatInt(tx.Timestamp, 10) + strings.Join(tx.ParentIDs, "")
	hash := sha256.Sum256([]byte(dataToHash))

	// Check if the hash matches the ID
	if hex.EncodeToString(hash[:]) != tx.ID {
		return false
	}

	// Verify the signature
	return ed25519.Verify(pubKey, hash[:], tx.Signature)
}

// SerializeTransaction serializes a transaction to a byte slice
func SerializeTransaction(tx *types.Transaction) ([]byte, error) {
	// Marshal metadata to JSON string
	metadataBytes, err := json.Marshal(tx.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// For simplicity, we'll use a string representation
	// In a real implementation, you might want to use a binary format like protobuf
	data := tx.ID + "," +
		strconv.FormatInt(tx.Timestamp, 10) + "," +
		tx.Data + "," +
		strings.Join(tx.ParentIDs, ":") + "," +
		hex.EncodeToString(tx.Signature) + "," +
		strconv.FormatBool(tx.Finalized) + "," +
		string(metadataBytes)

	return []byte(data), nil
}

// DeserializeTransaction deserializes a transaction from a byte slice
func DeserializeTransaction(data []byte) (*types.Transaction, error) {
	// Parse the string representation
	parts := strings.Split(string(data), ",")
	if len(parts) != 7 { // Now 7 parts including metadata
		return nil, fmt.Errorf("invalid transaction format: expected 7 parts, got %d", len(parts))
	}

	// Parse the timestamp
	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Parse the parent IDs
	var parentIDs []string
	if parts[3] != "" {
		parentIDs = strings.Split(parts[3], ":")
	}

	// Parse the signature
	signature, err := hex.DecodeString(parts[4])
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Parse the finalized flag
	finalized := parts[5] == "true"

	// Parse metadata
	var metadata map[string]string
	if err := json.Unmarshal([]byte(parts[6]), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Create the transaction
	tx := &types.Transaction{
		ID:        parts[0],
		Timestamp: timestamp,
		Data:      parts[2],
		ParentIDs: parentIDs,
		Signature: signature,
		Finalized: finalized,
		Metadata:  metadata,
	}

	return tx, nil
}
