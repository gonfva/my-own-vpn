package credentials

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"

	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/scrypt"
)

const (
	// Storage version for future crypto upgrades
	storageVersion = 1
	// Salt file name
	saltFileName = "salt"
	// Credentials file name
	credentialsFileName = "credentials.enc"
	// Directory name within user config
	configDirName = "my-own-vpn"
	// Nonce size for NaCl secretbox
	nonceSize = 24
	// Key size for NaCl secretbox
	keySize = 32
	// Salt size
	saltSize = 32
)

// encryptedStore represents the structure stored in the encrypted file
type encryptedStore struct {
	Version int    `json:"version"`
	AWS     []byte `json:"aws,omitempty"`
	Hetzner []byte `json:"hetzner,omitempty"`
}

// fallbackManager implements Manager using encrypted file storage
type fallbackManager struct {
	key  [keySize]byte
	path string
}

// newFallbackManager creates a new fallbackManager instance
func newFallbackManager() (*fallbackManager, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	// Ensure config directory exists with restrictive permissions
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return nil, err
	}

	key, err := deriveKey(configDir)
	if err != nil {
		return nil, err
	}

	credPath := filepath.Join(configDir, credentialsFileName)

	return &fallbackManager{key: key, path: credPath}, nil
}

// getConfigDir returns the configuration directory path
func getConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, configDirName), nil
}

// deriveKey derives an encryption key from the machine identifier and a stored salt
func deriveKey(configDir string) ([keySize]byte, error) {
	var key [keySize]byte

	saltPath := filepath.Join(configDir, saltFileName)
	// #nosec G304 -- saltPath is constructed from os.UserConfigDir() and a constant filename
	salt, err := os.ReadFile(saltPath)
	if err != nil {
		// Create new random salt
		salt = make([]byte, saltSize)
		if _, err := rand.Read(salt); err != nil {
			return key, err
		}
		// Write salt with restrictive permissions
		if err := os.WriteFile(saltPath, salt, 0o600); err != nil {
			return key, err
		}
	}

	// Get machine identifier
	machineID := getMachineIdentifier()

	// Derive key using scrypt (memory-hard key derivation)
	// Parameters: N=32768, r=8, p=1 (recommended for interactive logins)
	derived, err := scrypt.Key([]byte(machineID), salt, 32768, 8, 1, keySize)
	if err != nil {
		return key, err
	}

	copy(key[:], derived)
	return key, nil
}

// encrypt encrypts data using NaCl secretbox
func (m *fallbackManager) encrypt(data []byte) ([]byte, error) {
	var nonce [nonceSize]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}

	// secretbox.Seal appends the encrypted data to the first argument
	encrypted := secretbox.Seal(nonce[:], data, &nonce, &m.key)
	return encrypted, nil
}

// decrypt decrypts data using NaCl secretbox
func (m *fallbackManager) decrypt(encrypted []byte) ([]byte, error) {
	if len(encrypted) < nonceSize {
		return nil, ErrInvalidData
	}

	var nonce [nonceSize]byte
	copy(nonce[:], encrypted[:nonceSize])

	decrypted, ok := secretbox.Open(nil, encrypted[nonceSize:], &nonce, &m.key)
	if !ok {
		return nil, ErrDecryptionFailed
	}
	return decrypted, nil
}

// loadStore reads and decrypts the credential store from disk
func (m *fallbackManager) loadStore() (*encryptedStore, error) {
	// #nosec G304 -- m.path is constructed from os.UserConfigDir() and a constant filename
	data, err := os.ReadFile(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty store if file doesn't exist
			return &encryptedStore{Version: storageVersion}, nil
		}
		return nil, err
	}

	decrypted, err := m.decrypt(data)
	if err != nil {
		return nil, err
	}

	var store encryptedStore
	if err := json.Unmarshal(decrypted, &store); err != nil {
		return nil, err
	}

	return &store, nil
}

// saveStore encrypts and writes the credential store to disk
func (m *fallbackManager) saveStore(store *encryptedStore) error {
	store.Version = storageVersion

	data, err := json.Marshal(store)
	if err != nil {
		return err
	}

	encrypted, err := m.encrypt(data)
	if err != nil {
		return err
	}

	return os.WriteFile(m.path, encrypted, 0o600)
}

func (m *fallbackManager) SaveAWS(_ context.Context, creds AWSCredentials) error {
	store, err := m.loadStore()
	if err != nil {
		return err
	}

	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	store.AWS = data
	return m.saveStore(store)
}

func (m *fallbackManager) LoadAWS(_ context.Context) (*AWSCredentials, error) {
	store, err := m.loadStore()
	if err != nil {
		return nil, err
	}

	if len(store.AWS) == 0 {
		return nil, ErrNotFound
	}

	var creds AWSCredentials
	if err := json.Unmarshal(store.AWS, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func (m *fallbackManager) DeleteAWS(_ context.Context) error {
	store, err := m.loadStore()
	if err != nil {
		return err
	}

	store.AWS = nil
	return m.saveStore(store)
}

func (m *fallbackManager) SaveHetzner(_ context.Context, creds HetznerCredentials) error {
	store, err := m.loadStore()
	if err != nil {
		return err
	}

	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}

	store.Hetzner = data
	return m.saveStore(store)
}

func (m *fallbackManager) LoadHetzner(_ context.Context) (*HetznerCredentials, error) {
	store, err := m.loadStore()
	if err != nil {
		return nil, err
	}

	if len(store.Hetzner) == 0 {
		return nil, ErrNotFound
	}

	var creds HetznerCredentials
	if err := json.Unmarshal(store.Hetzner, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func (m *fallbackManager) DeleteHetzner(_ context.Context) error {
	store, err := m.loadStore()
	if err != nil {
		return err
	}

	store.Hetzner = nil
	return m.saveStore(store)
}

func (m *fallbackManager) HasCredentials(ctx context.Context, provider string) bool {
	switch provider {
	case ProviderAWS:
		creds, err := m.LoadAWS(ctx)
		return err == nil && creds != nil && !creds.IsEmpty()
	case ProviderHetzner:
		creds, err := m.LoadHetzner(ctx)
		return err == nil && creds != nil && !creds.IsEmpty()
	default:
		return false
	}
}
