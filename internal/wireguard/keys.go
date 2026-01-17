// Package wireguard provides WireGuard client functionality including
// key generation and configuration management.
package wireguard

import (
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// KeyPair represents a WireGuard key pair consisting of a private key
// and its corresponding public key.
type KeyPair struct {
	PrivateKey wgtypes.Key
	PublicKey  wgtypes.Key
}

// GenerateKeyPair generates a new WireGuard key pair.
// The private key is randomly generated using cryptographically secure random bytes,
// and the public key is derived from it.
func GenerateKeyPair() (*KeyPair, error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey(),
	}, nil
}

// PrivateKeyString returns the base64-encoded private key string.
// WARNING: This value is sensitive and should never be logged.
func (kp *KeyPair) PrivateKeyString() string {
	return kp.PrivateKey.String()
}

// PublicKeyString returns the base64-encoded public key string.
func (kp *KeyPair) PublicKeyString() string {
	return kp.PublicKey.String()
}

// ParsePrivateKey parses a base64-encoded private key string and returns
// a KeyPair with the corresponding public key derived from it.
func ParsePrivateKey(privateKeyStr string) (*KeyPair, error) {
	privateKey, err := wgtypes.ParseKey(privateKeyStr)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey(),
	}, nil
}
