package consensus_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/loreum-org/cortex/internal/consensus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to generate a new Ed25519 key pair for testing
func generateTestKeys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err, "Failed to generate test key pair")
	return pubKey, privKey
}

func TestCreateTransaction(t *testing.T) {
	_, privKey := generateTestKeys(t)
	data := "test transaction data"
	parentIDs := []string{"parent1", "parent2"}

	t.Run("SuccessfulCreationWithParents", func(t *testing.T) {
		before := time.Now().Unix()
		tx, err := consensus.CreateTransaction(data, parentIDs, privKey)
		after := time.Now().Unix()

		require.NoError(t, err)
		require.NotNil(t, tx)

		assert.NotEmpty(t, tx.ID, "Transaction ID should not be empty")
		assert.True(t, tx.Timestamp >= before && tx.Timestamp <= after, "Timestamp out of range")
		assert.Equal(t, data, tx.Data)
		assert.Equal(t, parentIDs, tx.ParentIDs)
		assert.NotEmpty(t, tx.Signature, "Signature should not be empty")
		assert.False(t, tx.Finalized, "New transaction should not be finalized")
		assert.NotNil(t, tx.Metadata, "Metadata should be initialized")
		assert.Empty(t, tx.Metadata, "Metadata should be empty initially")
	})

	t.Run("SuccessfulCreationWithEmptyParents", func(t *testing.T) {
		emptyParentIDs := []string{}
		tx, err := consensus.CreateTransaction(data, emptyParentIDs, privKey)
		require.NoError(t, err)
		require.NotNil(t, tx)
		assert.Equal(t, emptyParentIDs, tx.ParentIDs)
		assert.Empty(t, tx.ParentIDs) // Double check for empty specifically
	})

	t.Run("SuccessfulCreationWithNilParents", func(t *testing.T) {
		var nilParentIDs []string = nil
		tx, err := consensus.CreateTransaction(data, nilParentIDs, privKey)
		require.NoError(t, err)
		require.NotNil(t, tx)
		assert.Equal(t, nilParentIDs, tx.ParentIDs) // Should be nil
		assert.Nil(t, tx.ParentIDs)                 // Double check for nil specifically
	})
}

func TestValidateTransaction(t *testing.T) {
	pubKey, privKey := generateTestKeys(t)
	data := "valid transaction data"
	parentIDs := []string{"p1", "p2"}

	validTx, err := consensus.CreateTransaction(data, parentIDs, privKey)
	require.NoError(t, err)
	require.NotNil(t, validTx)

	t.Run("ValidTransaction", func(t *testing.T) {
		isValid := consensus.ValidateTransaction(validTx, pubKey)
		assert.True(t, isValid, "Valid transaction should pass validation")
	})

	t.Run("InvalidSignatureWrongKey", func(t *testing.T) {
		otherPubKey, _ := generateTestKeys(t)
		isValid := consensus.ValidateTransaction(validTx, otherPubKey)
		assert.False(t, isValid, "Transaction signed with a different key should fail validation")
	})

	t.Run("TamperedData", func(t *testing.T) {
		tamperedTx := *validTx // Create a copy
		tamperedTx.Data = "tampered data"
		// Note: ID will now mismatch the hash of tampered data + original timestamp + original parents.
		// ValidateTransaction checks ID against hash first.
		isValid := consensus.ValidateTransaction(&tamperedTx, pubKey)
		assert.False(t, isValid, "Transaction with tampered data should fail validation")
	})

	t.Run("TamperedTimestamp", func(t *testing.T) {
		tamperedTx := *validTx
		tamperedTx.Timestamp = time.Now().Unix() + 1000
		// ID will mismatch
		isValid := consensus.ValidateTransaction(&tamperedTx, pubKey)
		assert.False(t, isValid, "Transaction with tampered timestamp should fail validation")
	})

	t.Run("TamperedParentIDs", func(t *testing.T) {
		tamperedTx := *validTx
		tamperedTx.ParentIDs = []string{"tamperedP1"}
		// ID will mismatch
		isValid := consensus.ValidateTransaction(&tamperedTx, pubKey)
		assert.False(t, isValid, "Transaction with tampered parent IDs should fail validation")
	})

	t.Run("TamperedID", func(t *testing.T) {
		tamperedTx := *validTx
		tamperedTx.ID = "invalidID123"
		isValid := consensus.ValidateTransaction(&tamperedTx, pubKey)
		assert.False(t, isValid, "Transaction with tampered ID should fail validation")
	})

	t.Run("TamperedSignature", func(t *testing.T) {
		tamperedTx := *validTx // Copy the valid transaction
		// Keep ID, Data, Timestamp, ParentIDs the same as validTx so hash matches.
		// Only corrupt the signature.
		corruptedSig := make([]byte, len(tamperedTx.Signature))
		copy(corruptedSig, tamperedTx.Signature)
		if len(corruptedSig) > 0 {
			corruptedSig[0] ^= 0xff // Flip some bits
		} else {
			corruptedSig = []byte("corrupted") // If original signature was empty (should not happen for valid tx)
		}
		tamperedTx.Signature = corruptedSig

		isValid := consensus.ValidateTransaction(&tamperedTx, pubKey)
		assert.False(t, isValid, "Transaction with tampered signature should fail validation")
	})

	t.Run("NilTransaction", func(t *testing.T) {
		// This should ideally not happen, but good to see it doesn't panic.
		// The function expects a non-nil tx. Behavior for nil is undefined by signature.
		// It will likely panic due to nil pointer dereference.
		// Go tests typically don't assert panics unless specifically designed for it.
		// We can skip this or expect a panic if that's the desired contract.
		// For now, assume valid inputs.
	})
}

func TestSerializeDeserializeTransaction(t *testing.T) {
	_, privKey := generateTestKeys(t)
	originalTx, err := consensus.CreateTransaction("serialize me", []string{"parentA", "parentB"}, privKey)
	require.NoError(t, err)
	originalTx.Finalized = true
	originalTx.Metadata = map[string]string{"key1": "value1", "key2": "value2"}

	t.Run("SuccessfulRoundTrip", func(t *testing.T) {
		serializedData, err := consensus.SerializeTransaction(originalTx)
		require.NoError(t, err)
		require.NotNil(t, serializedData)

		deserializedTx, err := consensus.DeserializeTransaction(serializedData)
		require.NoError(t, err)
		require.NotNil(t, deserializedTx)

		assert.Equal(t, originalTx.ID, deserializedTx.ID)
		assert.Equal(t, originalTx.Timestamp, deserializedTx.Timestamp)
		assert.Equal(t, originalTx.Data, deserializedTx.Data)
		assert.Equal(t, originalTx.ParentIDs, deserializedTx.ParentIDs)
		assert.Equal(t, originalTx.Signature, deserializedTx.Signature)
		assert.Equal(t, originalTx.Finalized, deserializedTx.Finalized)
		assert.Equal(t, originalTx.Metadata, deserializedTx.Metadata)
	})

	t.Run("RoundTripWithEmptyParents", func(t *testing.T) {
		txWithEmptyParents, _ := consensus.CreateTransaction("empty parents", []string{}, privKey)
		txWithEmptyParents.Metadata = map[string]string{"meta": "data"}

		serializedData, err := consensus.SerializeTransaction(txWithEmptyParents)
		require.NoError(t, err)
		deserializedTx, err := consensus.DeserializeTransaction(serializedData)
		require.NoError(t, err)
		assert.Empty(t, deserializedTx.ParentIDs)
		assert.NotNil(t, deserializedTx.ParentIDs, "ParentIDs should be an empty slice, not nil, if originally empty")
	})

	t.Run("RoundTripWithNilParents", func(t *testing.T) {
		txWithNilParents, _ := consensus.CreateTransaction("nil parents", nil, privKey)
		txWithNilParents.Metadata = map[string]string{"meta": "data"}

		serializedData, err := consensus.SerializeTransaction(txWithNilParents)
		require.NoError(t, err)
		deserializedTx, err := consensus.DeserializeTransaction(serializedData)
		require.NoError(t, err)
		// The current serialization joins ParentIDs with ":". If ParentIDs is nil, strings.Join(nil, ":") is "".
		// Then, strings.Split("", ":") results in a slice with one empty string: `[""]`.
		// This is a known behavior of strings.Split.
		// If strict nil preservation is needed, serialization/deserialization logic needs adjustment.
		// For now, test current behavior.
		if len(deserializedTx.ParentIDs) == 1 && deserializedTx.ParentIDs[0] == "" {
			// This is the current behavior if original ParentIDs was nil or empty.
			// To distinguish nil from empty []string{}, serialization would need a different scheme.
			// For many use cases, an empty slice is equivalent to nil for ParentIDs.
			// If the distinction is critical, the serialization format needs to handle it.
			// The current `DeserializeTransaction` will produce `[]string{}` if `parts[3]` is empty.
			// Let's re-evaluate:
			// `strings.Join(nil, ":")` -> `""`
			// `strings.Split("", ":")` -> `[]string{""}`. This is not ideal.
			// The `DeserializeTransaction` has `if parts[3] != "" { parentIDs = strings.Split(parts[3], ":") }`
			// So if `parts[3]` is `""`, `parentIDs` remains `nil` (its zero value as declared `var parentIDs []string`).
			assert.Nil(t, deserializedTx.ParentIDs, "ParentIDs should be nil if originally nil and serialized as empty string part")
		} else {
			assert.Nil(t, deserializedTx.ParentIDs, "ParentIDs should be nil if originally nil")
		}
	})

	t.Run("DeserializeInvalidFormatTooFewParts", func(t *testing.T) {
		invalidData := []byte("id,timestamp,data,parents,signature,finalized") // Missing metadata
		_, err := consensus.DeserializeTransaction(invalidData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid transaction format")
	})

	t.Run("DeserializeInvalidFormatTooManyParts", func(t *testing.T) {
		invalidData := []byte("id,timestamp,data,parents,signature,finalized,meta,extra")
		_, err := consensus.DeserializeTransaction(invalidData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid transaction format")
	})

	t.Run("DeserializeInvalidTimestamp", func(t *testing.T) {
		// Construct a valid-looking string but with bad timestamp
		metadataBytes, _ := json.Marshal(map[string]string{"k": "v"})
		invalidDataStr := originalTx.ID + "," +
			"not_a_timestamp" + "," +
			originalTx.Data + "," +
			strings.Join(originalTx.ParentIDs, ":") + "," +
			hex.EncodeToString(originalTx.Signature) + "," +
			strconv.FormatBool(originalTx.Finalized) + "," +
			string(metadataBytes)

		_, err := consensus.DeserializeTransaction([]byte(invalidDataStr))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse timestamp")
	})

	t.Run("DeserializeInvalidSignatureHex", func(t *testing.T) {
		metadataBytes, _ := json.Marshal(map[string]string{"k": "v"})
		invalidDataStr := originalTx.ID + "," +
			strconv.FormatInt(originalTx.Timestamp, 10) + "," +
			originalTx.Data + "," +
			strings.Join(originalTx.ParentIDs, ":") + "," +
			"not_hex_signature" + "," +
			strconv.FormatBool(originalTx.Finalized) + "," +
			string(metadataBytes)

		_, err := consensus.DeserializeTransaction([]byte(invalidDataStr))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode signature")
	})

	t.Run("DeserializeInvalidMetadataJSON", func(t *testing.T) {
		invalidDataStr := originalTx.ID + "," +
			strconv.FormatInt(originalTx.Timestamp, 10) + "," +
			originalTx.Data + "," +
			strings.Join(originalTx.ParentIDs, ":") + "," +
			hex.EncodeToString(originalTx.Signature) + "," +
			strconv.FormatBool(originalTx.Finalized) + "," +
			"{not_valid_json"

		_, err := consensus.DeserializeTransaction([]byte(invalidDataStr))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal metadata")
	})

	t.Run("DeserializeWithEmptyMetadataField", func(t *testing.T) {
		// This scenario implies the metadata part of the string is `"{}"` or an empty map's JSON.
		txNoMeta, _ := consensus.CreateTransaction("no meta", nil, privKey)
		// txNoMeta.Metadata is already make(map[string]string)
		serializedData, _ := consensus.SerializeTransaction(txNoMeta)
		deserializedTx, err := consensus.DeserializeTransaction(serializedData)
		require.NoError(t, err)
		assert.NotNil(t, deserializedTx.Metadata)
		assert.Empty(t, deserializedTx.Metadata)
	})

	t.Run("SerializeNilTransaction", func(t *testing.T) {
		// Similar to TestValidateTransaction/NilTransaction, this might panic.
		// The function expects a non-nil tx.
		// _, err := consensus.SerializeTransaction(nil)
		// assert.Error(t, err) // Or expect panic.
		// For now, assume valid inputs.
	})

	t.Run("DeserializeEmptyData", func(t *testing.T) {
		_, err := consensus.DeserializeTransaction([]byte(""))
		assert.Error(t, err) // Should fail due to wrong number of parts
	})
}

// Test for the specific case of ParentIDs serialization/deserialization
// to confirm behavior around nil vs empty slice.
func TestParentIDSerialization(t *testing.T) {
	_, privKey := generateTestKeys(t)

	// Case 1: ParentIDs is nil
	txNilParents, _ := consensus.CreateTransaction("data", nil, privKey)
	serializedNil, err := consensus.SerializeTransaction(txNilParents)
	require.NoError(t, err)
	deserializedNil, err := consensus.DeserializeTransaction(serializedNil)
	require.NoError(t, err)
	assert.Nil(t, deserializedNil.ParentIDs, "ParentIDs deserialized from nil original should be nil")

	// Case 2: ParentIDs is an empty slice
	txEmptyParents, _ := consensus.CreateTransaction("data", []string{}, privKey)
	serializedEmpty, err := consensus.SerializeTransaction(txEmptyParents)
	require.NoError(t, err)
	deserializedEmpty, err := consensus.DeserializeTransaction(serializedEmpty)
	require.NoError(t, err)
	// Current DeserializeTransaction logic for an empty `parts[3]` (from `strings.Join([]string{}, ":") == ""`)
	// results in `parentIDs` (which is `var parentIDs []string`) remaining `nil`.
	assert.Nil(t, deserializedEmpty.ParentIDs, "ParentIDs deserialized from empty slice original results in nil with current logic")
	// If strict empty slice `[]string{}` preservation is needed, DeserializeTransaction would need:
	// if parts[3] == "" { parentIDs = []string{} } else { parentIDs = strings.Split(parts[3], ":") }
	// However, for many practical purposes, a nil slice and an empty slice of strings are treated similarly.
}

func TestTransactionMetadata(t *testing.T) {
	_, privKey := generateTestKeys(t)
	tx, _ := consensus.CreateTransaction("metadata test", nil, privKey)

	// Default metadata
	assert.NotNil(t, tx.Metadata)
	assert.Empty(t, tx.Metadata)

	// Set metadata
	tx.Metadata["type"] = "test_tx"
	tx.Metadata["version"] = "1.0"

	serialized, err := consensus.SerializeTransaction(tx)
	require.NoError(t, err)

	deserialized, err := consensus.DeserializeTransaction(serialized)
	require.NoError(t, err)

	assert.NotNil(t, deserialized.Metadata)
	assert.Equal(t, "test_tx", deserialized.Metadata["type"])
	assert.Equal(t, "1.0", deserialized.Metadata["version"])
	assert.Len(t, deserialized.Metadata, 2)
}
