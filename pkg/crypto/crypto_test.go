package crypto

import (
	"bytes"
	"crypto/ed25519"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Errorf("GenerateKeyPair() unexpected error: %v", err)
		return
	}

	if kp == nil {
		t.Errorf("GenerateKeyPair() returned nil key pair")
		return
	}

	if len(kp.PublicKey) != ed25519.PublicKeySize {
		t.Errorf("Public key size = %d, want %d", len(kp.PublicKey), ed25519.PublicKeySize)
	}

	if len(kp.PrivateKey) != ed25519.PrivateKeySize {
		t.Errorf("Private key size = %d, want %d", len(kp.PrivateKey), ed25519.PrivateKeySize)
	}
}

func TestKeyPair_SignVerify(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	message := []byte("test message for signing")
	signature := kp.Sign(message)

	if len(signature) != ed25519.SignatureSize {
		t.Errorf("Signature size = %d, want %d", len(signature), ed25519.SignatureSize)
	}

	// Test valid signature
	if !kp.Verify(message, signature) {
		t.Errorf("Verify() failed for valid signature")
	}

	// Test invalid signature
	wrongMessage := []byte("wrong message")
	if kp.Verify(wrongMessage, signature) {
		t.Errorf("Verify() succeeded for invalid signature")
	}

	// Test with tampered signature
	tamperedSignature := make([]byte, len(signature))
	copy(tamperedSignature, signature)
	tamperedSignature[0] ^= 1 // flip a bit

	if kp.Verify(message, tamperedSignature) {
		t.Errorf("Verify() succeeded for tampered signature")
	}
}

func TestPublicPrivateKeyFromBytes(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Test public key from bytes
	pubKey, err := PublicKeyFromBytes(kp.PublicKey)
	if err != nil {
		t.Errorf("PublicKeyFromBytes() unexpected error: %v", err)
	}

	if !bytes.Equal(pubKey, kp.PublicKey) {
		t.Errorf("PublicKeyFromBytes() result mismatch")
	}

	// Test invalid public key size
	_, err = PublicKeyFromBytes([]byte("invalid"))
	if err == nil {
		t.Errorf("PublicKeyFromBytes() expected error for invalid size")
	}

	// Test private key from bytes
	privKey, err := PrivateKeyFromBytes(kp.PrivateKey)
	if err != nil {
		t.Errorf("PrivateKeyFromBytes() unexpected error: %v", err)
	}

	if !bytes.Equal(privKey, kp.PrivateKey) {
		t.Errorf("PrivateKeyFromBytes() result mismatch")
	}

	// Test invalid private key size
	_, err = PrivateKeyFromBytes([]byte("invalid"))
	if err == nil {
		t.Errorf("PrivateKeyFromBytes() expected error for invalid size")
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	sizes := []int{16, 32, 64, 128}

	for _, size := range sizes {
		t.Run(string(rune(size)), func(t *testing.T) {
			bytes1, err := GenerateRandomBytes(size)
			if err != nil {
				t.Errorf("GenerateRandomBytes(%d) unexpected error: %v", size, err)
				return
			}

			if len(bytes1) != size {
				t.Errorf("GenerateRandomBytes(%d) length = %d, want %d", size, len(bytes1), size)
			}

			// Generate another set and ensure they're different
			bytes2, err := GenerateRandomBytes(size)
			if err != nil {
				t.Errorf("GenerateRandomBytes(%d) unexpected error: %v", size, err)
				return
			}

			if bytes.Equal(bytes1, bytes2) {
				t.Errorf("GenerateRandomBytes(%d) generated identical bytes", size)
			}
		})
	}
}

func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Errorf("GenerateKey() unexpected error: %v", err)
		return
	}

	if len(key) != KeySize {
		t.Errorf("GenerateKey() length = %d, want %d", len(key), KeySize)
	}
}

func TestDeriveKeyFromPassword(t *testing.T) {
	password := "test-password"
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	key1 := DeriveKeyFromPassword(password, salt)
	key2 := DeriveKeyFromPassword(password, salt)

	if len(key1) != KeySize {
		t.Errorf("DeriveKeyFromPassword() length = %d, want %d", len(key1), KeySize)
	}

	// Same password and salt should produce same key
	if !bytes.Equal(key1, key2) {
		t.Errorf("DeriveKeyFromPassword() inconsistent results")
	}

	// Different salt should produce different key
	salt2, _ := GenerateSalt()
	key3 := DeriveKeyFromPassword(password, salt2)
	if bytes.Equal(key1, key3) {
		t.Errorf("DeriveKeyFromPassword() same key for different salts")
	}
}

func TestDeriveKeyFromPasswordScrypt(t *testing.T) {
	password := "test-password"
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	key, err := DeriveKeyFromPasswordScrypt(password, salt)
	if err != nil {
		t.Errorf("DeriveKeyFromPasswordScrypt() unexpected error: %v", err)
		return
	}

	if len(key) != KeySize {
		t.Errorf("DeriveKeyFromPasswordScrypt() length = %d, want %d", len(key), KeySize)
	}
}

func TestGenerateSalt(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Errorf("GenerateSalt() unexpected error: %v", err)
		return
	}

	if len(salt) != SaltSize {
		t.Errorf("GenerateSalt() length = %d, want %d", len(salt), SaltSize)
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	testData := [][]byte{
		[]byte("hello world"),
		[]byte(""),
		[]byte("a very long message that spans multiple blocks and tests the encryption implementation thoroughly"),
		{0x00, 0x01, 0x02, 0xFF, 0xFE}, // binary data
	}

	for i, data := range testData {
		t.Run(string(rune(i)), func(t *testing.T) {
			// Encrypt
			encData, err := Encrypt(data, key)
			if err != nil {
				t.Errorf("Encrypt() unexpected error: %v", err)
				return
			}

			if len(encData.Nonce) != NonceSize {
				t.Errorf("Encrypt() nonce size = %d, want %d", len(encData.Nonce), NonceSize)
			}

			// Decrypt
			decrypted, err := Decrypt(encData, key)
			if err != nil {
				t.Errorf("Decrypt() unexpected error: %v", err)
				return
			}

			if !bytes.Equal(data, decrypted) {
				t.Errorf("Decrypt() result mismatch. Original: %v, Decrypted: %v", data, decrypted)
			}
		})
	}
}

func TestEncryptDecrypt_InvalidKey(t *testing.T) {
	data := []byte("test data")

	// Test invalid key size for encryption
	invalidKey := []byte("short")
	_, err := Encrypt(data, invalidKey)
	if err == nil {
		t.Errorf("Encrypt() expected error for invalid key size")
	}

	// Test invalid key size for decryption
	validKey, _ := GenerateKey()
	encData, _ := Encrypt(data, validKey)
	_, err = Decrypt(encData, invalidKey)
	if err == nil {
		t.Errorf("Decrypt() expected error for invalid key size")
	}
}

func TestEncryptWithPassword(t *testing.T) {
	password := "secure-password-123"
	data := []byte("secret data to encrypt")

	encData, salt, err := EncryptWithPassword(data, password)
	if err != nil {
		t.Errorf("EncryptWithPassword() unexpected error: %v", err)
		return
	}

	if len(salt) != SaltSize {
		t.Errorf("EncryptWithPassword() salt size = %d, want %d", len(salt), SaltSize)
	}

	// Decrypt
	decrypted, err := DecryptWithPassword(encData, password, salt)
	if err != nil {
		t.Errorf("DecryptWithPassword() unexpected error: %v", err)
		return
	}

	if !bytes.Equal(data, decrypted) {
		t.Errorf("DecryptWithPassword() result mismatch")
	}

	// Test wrong password
	_, err = DecryptWithPassword(encData, "wrong-password", salt)
	if err == nil {
		t.Errorf("DecryptWithPassword() expected error for wrong password")
	}
}

func TestHash(t *testing.T) {
	data := []byte("test data")
	hash1 := Hash(data)
	hash2 := Hash(data)

	if len(hash1) != 32 { // SHA-256 output size
		t.Errorf("Hash() length = %d, want 32", len(hash1))
	}

	// Same data should produce same hash
	if !bytes.Equal(hash1, hash2) {
		t.Errorf("Hash() inconsistent results")
	}

	// Different data should produce different hash
	hash3 := Hash([]byte("different data"))
	if bytes.Equal(hash1, hash3) {
		t.Errorf("Hash() same result for different data")
	}
}

func TestHashString(t *testing.T) {
	data := "test string"
	hash := HashString(data)

	if len(hash) != 64 { // 32 bytes * 2 (hex encoding)
		t.Errorf("HashString() length = %d, want 64", len(hash))
	}

	// Should be valid hex
	for _, char := range hash {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			t.Errorf("HashString() contains invalid hex character: %c", char)
		}
	}
}

func TestSecureCompare(t *testing.T) {
	data1 := []byte("same data")
	data2 := []byte("same data")
	data3 := []byte("different")

	if !SecureCompare(data1, data2) {
		t.Errorf("SecureCompare() failed for equal data")
	}

	if SecureCompare(data1, data3) {
		t.Errorf("SecureCompare() succeeded for different data")
	}

	if SecureCompare(data1, []byte("")) {
		t.Errorf("SecureCompare() succeeded for different length data")
	}
}

func TestBase64Encoding(t *testing.T) {
	data := []byte("test data for base64 encoding")
	encoded := EncodeBase64(data)
	decoded, err := DecodeBase64(encoded)

	if err != nil {
		t.Errorf("DecodeBase64() unexpected error: %v", err)
		return
	}

	if !bytes.Equal(data, decoded) {
		t.Errorf("Base64 encoding/decoding mismatch")
	}

	// Test invalid base64
	_, err = DecodeBase64("invalid base64!")
	if err == nil {
		t.Errorf("DecodeBase64() expected error for invalid input")
	}
}

func TestHexEncoding(t *testing.T) {
	data := []byte("test data for hex encoding")
	encoded := EncodeHex(data)
	decoded, err := DecodeHex(encoded)

	if err != nil {
		t.Errorf("DecodeHex() unexpected error: %v", err)
		return
	}

	if !bytes.Equal(data, decoded) {
		t.Errorf("Hex encoding/decoding mismatch")
	}

	// Test invalid hex
	_, err = DecodeHex("invalid hex!")
	if err == nil {
		t.Errorf("DecodeHex() expected error for invalid input")
	}
}

func TestHMAC(t *testing.T) {
	key := []byte("secret key")
	data := []byte("data to authenticate")

	mac1 := HMAC(key, data)
	mac2 := HMAC(key, data)

	if len(mac1) != 32 { // SHA-256 output size
		t.Errorf("HMAC() length = %d, want 32", len(mac1))
	}

	// Same key and data should produce same HMAC
	if !bytes.Equal(mac1, mac2) {
		t.Errorf("HMAC() inconsistent results")
	}

	// Verify HMAC
	if !VerifyHMAC(key, data, mac1) {
		t.Errorf("VerifyHMAC() failed for valid HMAC")
	}

	// Wrong key should fail verification
	wrongKey := []byte("wrong key")
	if VerifyHMAC(wrongKey, data, mac1) {
		t.Errorf("VerifyHMAC() succeeded with wrong key")
	}
}

func TestSecureRandom(t *testing.T) {
	max := int64(100)

	for i := 0; i < 10; i++ {
		num, err := SecureRandom(max)
		if err != nil {
			t.Errorf("SecureRandom() unexpected error: %v", err)
			continue
		}

		if num < 0 || num >= max {
			t.Errorf("SecureRandom(%d) = %d, out of range", max, num)
		}
	}

	// Test invalid max
	_, err := SecureRandom(0)
	if err == nil {
		t.Errorf("SecureRandom(0) expected error")
	}

	_, err = SecureRandom(-1)
	if err == nil {
		t.Errorf("SecureRandom(-1) expected error")
	}
}

func TestKeyStretch(t *testing.T) {
	key := []byte("original key")

	stretched1 := KeyStretch(key, 1000)
	stretched2 := KeyStretch(key, 1000)

	if len(stretched1) != 32 { // SHA256 always returns 32 bytes
		t.Errorf("KeyStretch() unexpected length: got %d, want 32", len(stretched1))
	}

	// Same key and iterations should produce same result
	if !bytes.Equal(stretched1, stretched2) {
		t.Errorf("KeyStretch() inconsistent results")
	}

	// Different iterations should produce different results
	stretched3 := KeyStretch(key, 2000)
	if bytes.Equal(stretched1, stretched3) {
		t.Errorf("KeyStretch() same result for different iterations")
	}
}

func TestZeroBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	ZeroBytes(data)

	for i, b := range data {
		if b != 0 {
			t.Errorf("ZeroBytes() data[%d] = %d, want 0", i, b)
		}
	}
}

func TestGenerateCSRF(t *testing.T) {
	token1, err := GenerateCSRF()
	if err != nil {
		t.Errorf("GenerateCSRF() unexpected error: %v", err)
		return
	}

	token2, err := GenerateCSRF()
	if err != nil {
		t.Errorf("GenerateCSRF() unexpected error: %v", err)
		return
	}

	if token1 == token2 {
		t.Errorf("GenerateCSRF() generated identical tokens")
	}

	// Should be valid base64
	_, err = DecodeBase64(token1)
	if err != nil {
		t.Errorf("GenerateCSRF() produced invalid base64: %v", err)
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := GenerateSessionID()
	if err != nil {
		t.Errorf("GenerateSessionID() unexpected error: %v", err)
		return
	}

	id2, err := GenerateSessionID()
	if err != nil {
		t.Errorf("GenerateSessionID() unexpected error: %v", err)
		return
	}

	if id1 == id2 {
		t.Errorf("GenerateSessionID() generated identical IDs")
	}

	// Should be valid hex
	_, err = DecodeHex(id1)
	if err != nil {
		t.Errorf("GenerateSessionID() produced invalid hex: %v", err)
	}
}

func TestValidateSignature(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	message := []byte("test message")
	signature := kp.Sign(message)

	// Valid signature
	err = ValidateSignature(kp.PublicKey, message, signature)
	if err != nil {
		t.Errorf("ValidateSignature() unexpected error for valid signature: %v", err)
	}

	// Invalid public key size
	err = ValidateSignature([]byte("short"), message, signature)
	if err == nil {
		t.Errorf("ValidateSignature() expected error for invalid public key size")
	}

	// Invalid signature size
	err = ValidateSignature(kp.PublicKey, message, []byte("short"))
	if err == nil {
		t.Errorf("ValidateSignature() expected error for invalid signature size")
	}

	// Invalid signature
	wrongMessage := []byte("wrong message")
	err = ValidateSignature(kp.PublicKey, wrongMessage, signature)
	if err == nil {
		t.Errorf("ValidateSignature() expected error for invalid signature")
	}
}

func TestCheckPasswordStrength(t *testing.T) {
	tests := []struct {
		password string
		expected PasswordStrength
	}{
		{"", PasswordWeak},
		{"123", PasswordWeak},
		{"password", PasswordWeak},
		{"Password", PasswordFair},
		{"Password1", PasswordGood},
		{"Password1!", PasswordStrong},
		{"VerySecurePassword123!", PasswordVeryStrong},
	}

	for _, tt := range tests {
		t.Run(tt.password, func(t *testing.T) {
			result := CheckPasswordStrength(tt.password)
			if result != tt.expected {
				t.Errorf("CheckPasswordStrength(%q) = %s, want %s", tt.password, result, tt.expected)
			}
		})
	}
}

func TestPasswordStrength_String(t *testing.T) {
	tests := []struct {
		strength PasswordStrength
		expected string
	}{
		{PasswordWeak, "weak"},
		{PasswordFair, "fair"},
		{PasswordGood, "good"},
		{PasswordStrong, "strong"},
		{PasswordVeryStrong, "very strong"},
	}

	for _, tt := range tests {
		result := tt.strength.String()
		if result != tt.expected {
			t.Errorf("PasswordStrength.String() = %s, want %s", result, tt.expected)
		}
	}
}

func TestSecureReader(t *testing.T) {
	data := []byte("test data for secure reading")
	reader := NewSecureReader(bytes.NewReader(data))

	// Read exact amount
	result, err := reader.ReadExact(9) // "test data"
	if err != nil {
		t.Errorf("ReadExact() unexpected error: %v", err)
		return
	}

	expected := []byte("test data")
	if !bytes.Equal(result, expected) {
		t.Errorf("ReadExact() = %v, want %v", result, expected)
	}

	// Try to read more than available
	_, err = reader.ReadExact(100)
	if err == nil {
		t.Errorf("ReadExact() expected error when reading more than available")
	}
}
