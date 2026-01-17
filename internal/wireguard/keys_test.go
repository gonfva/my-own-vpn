package wireguard

import (
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	if kp.PrivateKeyString() == "" {
		t.Error("empty private key")
	}

	if kp.PublicKeyString() == "" {
		t.Error("empty public key")
	}

	// Verify public key derivation is consistent
	if kp.PublicKey != kp.PrivateKey.PublicKey() {
		t.Error("public key mismatch")
	}
}

func TestGenerateKeyPairUniqueness(t *testing.T) {
	kp1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("first key generation failed: %v", err)
	}

	kp2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("second key generation failed: %v", err)
	}

	if kp1.PrivateKeyString() == kp2.PrivateKeyString() {
		t.Error("generated keys should be unique")
	}

	if kp1.PublicKeyString() == kp2.PublicKeyString() {
		t.Error("generated public keys should be unique")
	}
}

func TestParsePrivateKey(t *testing.T) {
	// Generate a key pair first
	original, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	// Parse the private key string back into a KeyPair
	parsed, err := ParsePrivateKey(original.PrivateKeyString())
	if err != nil {
		t.Fatalf("failed to parse private key: %v", err)
	}

	// Verify the parsed key matches the original
	if parsed.PrivateKeyString() != original.PrivateKeyString() {
		t.Error("private keys don't match after parsing")
	}

	if parsed.PublicKeyString() != original.PublicKeyString() {
		t.Error("public keys don't match after parsing")
	}
}

func TestParsePrivateKeyInvalid(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"invalid base64", "not-valid-base64!@#$"},
		{"too short", "AAAA"},
		{"invalid characters", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAzz!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePrivateKey(tt.key)
			if err == nil {
				t.Error("expected error for invalid key")
			}
		})
	}
}

func TestKeyPairStringMethods(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("key generation failed: %v", err)
	}

	// WireGuard keys are 32 bytes, base64 encoded to 44 characters
	// (with padding character potentially at the end)
	privateStr := kp.PrivateKeyString()
	if len(privateStr) != 44 {
		t.Errorf("private key string length should be 44, got %d", len(privateStr))
	}

	publicStr := kp.PublicKeyString()
	if len(publicStr) != 44 {
		t.Errorf("public key string length should be 44, got %d", len(publicStr))
	}
}
