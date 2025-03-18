package consensus

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/loreum-org/cortex/pkg/types"
)

// CreateTransaction creates a new transaction with the given data and parents
func CreateTransaction(data string, parents []string, privKey ed25519.PrivateKey) (*types.Transaction, error) {
	// Create a timestamp
	timestamp := time.Now()

	// Create a hash of the data and timestamp
	dataToHash := data + timestamp.String() + strings.Join(parents, "")
	hash := sha256.Sum256([]byte(dataToHash))

	// Sign the hash with the private key
	signature := ed25519.Sign(privKey, hash[:])

	// Create the transaction
	tx := &types.Transaction{
		ID:        hex.EncodeToString(hash[:]),
		Timestamp: timestamp,
		Data:      data,
		Parents:   parents,
		Signature: signature,
		Finalized: false,
	}

	return tx, nil
}

// ValidateTransaction validates a transaction
func ValidateTransaction(tx *types.Transaction, pubKey ed25519.PublicKey) bool {
	// Recreate the hash
	dataToHash := tx.Data + tx.Timestamp.String() + strings.Join(tx.Parents, "")
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
	// For simplicity, we'll use a string representation
	// In a real implementation, you might want to use a binary format like protobuf
	data := tx.ID + "," +
		tx.Timestamp.Format(time.RFC3339) + "," +
		tx.Data + "," +
		strings.Join(tx.Parents, ":") + "," +
		hex.EncodeToString(tx.Signature) + "," +
		strconv.FormatBool(tx.Finalized)

	return []byte(data), nil
}

// DeserializeTransaction deserializes a transaction from a byte slice
func DeserializeTransaction(data []byte) (*types.Transaction, error) {
	// Parse the string representation
	parts := strings.Split(string(data), ",")
	if len(parts) != 6 {
		return nil, nil
	}

	// Parse the timestamp
	timestamp, err := time.Parse(time.RFC3339, parts[1])
	if err != nil {
		return nil, err
	}

	// Parse the parents
	var parents []string
	if parts[3] != "" {
		parents = strings.Split(parts[3], ":")
	}

	// Parse the signature
	signature, err := hex.DecodeString(parts[4])
	if err != nil {
		return nil, err
	}

	// Parse the finalized flag
	finalized := parts[5] == "true"

	// Create the transaction
	tx := &types.Transaction{
		ID:        parts[0],
		Timestamp: timestamp,
		Data:      parts[2],
		Parents:   parents,
		Signature: signature,
		Finalized: finalized,
	}

	return tx, nil
}
