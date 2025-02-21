package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
)

// GenerateKeys creates a new key pair for signing transactions.
func GenerateKeys() (ed25519.PublicKey, ed25519.PrivateKey) {
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	return publicKey, privateKey
}
