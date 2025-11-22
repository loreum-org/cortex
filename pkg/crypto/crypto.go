package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/scrypt"
)

// KeySize defines the size of encryption keys in bytes
const KeySize = 32

// NonceSize defines the size of nonces for encryption
const NonceSize = 12

// SaltSize defines the size of salts for key derivation
const SaltSize = 16

// KeyPair represents an Ed25519 key pair
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// EncryptedData represents encrypted data with its nonce
type EncryptedData struct {
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// GenerateKeyPair generates a new Ed25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return &KeyPair{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// Sign signs a message using the private key
func (kp *KeyPair) Sign(message []byte) []byte {
	return ed25519.Sign(kp.PrivateKey, message)
}

// Verify verifies a signature using the public key
func (kp *KeyPair) Verify(message, signature []byte) bool {
	return ed25519.Verify(kp.PublicKey, message, signature)
}

// PublicKeyFromBytes creates a public key from bytes
func PublicKeyFromBytes(data []byte) (ed25519.PublicKey, error) {
	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d, expected %d", len(data), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(data), nil
}

// PrivateKeyFromBytes creates a private key from bytes
func PrivateKeyFromBytes(data []byte) (ed25519.PrivateKey, error) {
	if len(data) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: %d, expected %d", len(data), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(data), nil
}

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// GenerateKey generates a random encryption key
func GenerateKey() ([]byte, error) {
	return GenerateRandomBytes(KeySize)
}

// DeriveKeyFromPassword derives an encryption key from a password using Argon2
func DeriveKeyFromPassword(password string, salt []byte) []byte {
	// Argon2id parameters
	time := uint32(1)
	memory := uint32(64 * 1024) // 64 MB
	threads := uint8(4)

	return argon2.IDKey([]byte(password), salt, time, memory, threads, KeySize)
}

// DeriveKeyFromPasswordScrypt derives an encryption key from a password using scrypt
func DeriveKeyFromPasswordScrypt(password string, salt []byte) ([]byte, error) {
	// scrypt parameters
	N := 32768 // CPU/memory cost
	r := 8     // block size
	p := 1     // parallelization

	key, err := scrypt.Key([]byte(password), salt, N, r, p, KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key with scrypt: %w", err)
	}
	return key, nil
}

// GenerateSalt generates a random salt for key derivation
func GenerateSalt() ([]byte, error) {
	return GenerateRandomBytes(SaltSize)
}

// Encrypt encrypts data using AES-GCM with the provided key
func Encrypt(data, key []byte) (*EncryptedData, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: %d, expected %d", len(key), KeySize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := GenerateRandomBytes(NonceSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nil, nonce, data, nil)

	return &EncryptedData{
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}, nil
}

// Decrypt decrypts data using AES-GCM with the provided key
func Decrypt(encData *EncryptedData, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: %d, expected %d", len(key), KeySize)
	}

	if len(encData.Nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce size: %d, expected %d", len(encData.Nonce), NonceSize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := aesGCM.Open(nil, encData.Nonce, encData.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptWithPassword encrypts data with a password-derived key
func EncryptWithPassword(data []byte, password string) (*EncryptedData, []byte, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key := DeriveKeyFromPassword(password, salt)
	encData, err := Encrypt(data, key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	return encData, salt, nil
}

// DecryptWithPassword decrypts data with a password-derived key
func DecryptWithPassword(encData *EncryptedData, password string, salt []byte) ([]byte, error) {
	key := DeriveKeyFromPassword(password, salt)
	return Decrypt(encData, key)
}

// Hash computes SHA-256 hash of data
func Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashString computes SHA-256 hash of a string and returns hex encoding
func HashString(data string) string {
	hash := Hash([]byte(data))
	return hex.EncodeToString(hash)
}

// SecureCompare performs constant-time comparison of two byte slices
func SecureCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// EncodeBase64 encodes bytes to base64 string
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 string to bytes
func DecodeBase64(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// EncodeHex encodes bytes to hex string
func EncodeHex(data []byte) string {
	return hex.EncodeToString(data)
}

// DecodeHex decodes hex string to bytes
func DecodeHex(data string) ([]byte, error) {
	return hex.DecodeString(data)
}

// HMAC computes HMAC-SHA256 of data with the given key
func HMAC(key, data []byte) []byte {
	h := sha256.New()
	h.Write(key)
	h.Write(data)
	return h.Sum(nil)
}

// VerifyHMAC verifies HMAC-SHA256 signature
func VerifyHMAC(key, data, signature []byte) bool {
	expected := HMAC(key, data)
	return SecureCompare(expected, signature)
}

// SecureRandom generates a cryptographically secure random number in the range [0, max)
func SecureRandom(max int64) (int64, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max must be positive")
	}

	// Calculate the number of bytes needed
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return 0, fmt.Errorf("failed to read random bytes: %w", err)
	}

	// Convert bytes to int64
	var result int64
	for i, b := range bytes {
		result |= int64(b) << (8 * i)
	}

	// Make it positive and within range
	if result < 0 {
		result = -result
	}
	return result % max, nil
}

// KeyStretch stretches a key using PBKDF2-like iteration
func KeyStretch(key []byte, iterations int) []byte {
	if iterations <= 0 {
		iterations = 10000
	}

	stretched := Hash(key)

	for i := 1; i < iterations; i++ {
		stretched = Hash(stretched)
	}

	return stretched
}

// ZeroBytes securely zeros out a byte slice
func ZeroBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

// SecureWipe securely wipes sensitive data from memory
func SecureWipe(data []byte) {
	// First pass: zero out
	ZeroBytes(data)

	// Second pass: random data
	rand.Read(data)

	// Third pass: zero out again
	ZeroBytes(data)
}

// GenerateCSRF generates a CSRF token
func GenerateCSRF() (string, error) {
	bytes, err := GenerateRandomBytes(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate CSRF token: %w", err)
	}
	return EncodeBase64(bytes), nil
}

// GenerateSessionID generates a secure session ID
func GenerateSessionID() (string, error) {
	bytes, err := GenerateRandomBytes(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return EncodeHex(bytes), nil
}

// ValidateSignature validates an Ed25519 signature
func ValidateSignature(publicKey ed25519.PublicKey, message, signature []byte) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size")
	}

	if len(signature) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature size")
	}

	if !ed25519.Verify(publicKey, message, signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// SecureReader wraps an io.Reader to ensure it reads the exact number of bytes
type SecureReader struct {
	reader io.Reader
}

// NewSecureReader creates a new SecureReader
func NewSecureReader(reader io.Reader) *SecureReader {
	return &SecureReader{reader: reader}
}

// ReadExact reads exactly n bytes or returns an error
func (sr *SecureReader) ReadExact(n int) ([]byte, error) {
	data := make([]byte, n)
	_, err := io.ReadFull(sr.reader, data)
	if err != nil {
		return nil, fmt.Errorf("failed to read exact bytes: %w", err)
	}
	return data, nil
}

// PasswordStrength represents the strength of a password
type PasswordStrength int

const (
	PasswordWeak PasswordStrength = iota
	PasswordFair
	PasswordGood
	PasswordStrong
	PasswordVeryStrong
)

// String returns the string representation of password strength
func (ps PasswordStrength) String() string {
	switch ps {
	case PasswordWeak:
		return "weak"
	case PasswordFair:
		return "fair"
	case PasswordGood:
		return "good"
	case PasswordStrong:
		return "strong"
	case PasswordVeryStrong:
		return "very strong"
	default:
		return "unknown"
	}
}

// CheckPasswordStrength evaluates password strength
func CheckPasswordStrength(password string) PasswordStrength {
	if len(password) < 6 {
		return PasswordWeak
	}

	score := 0

	// Length bonus
	if len(password) >= 8 {
		score++
	}
	if len(password) >= 12 {
		score++
	}

	// Character variety
	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= '0' && char <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	if hasLower {
		score++
	}
	if hasUpper {
		score++
	}
	if hasDigit {
		score++
	}
	if hasSpecial {
		score++
	}

	switch {
	case score <= 2:
		return PasswordWeak
	case score == 3:
		return PasswordFair
	case score == 4:
		return PasswordGood
	case score == 5:
		return PasswordStrong
	default:
		return PasswordVeryStrong
	}
}
