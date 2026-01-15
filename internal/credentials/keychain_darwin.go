//go:build darwin

package credentials

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/keybase/go-keychain"
)

// keychainManager implements Manager using macOS Keychain
type keychainManager struct{}

// newKeychainManager creates a new keychainManager instance
func newKeychainManager() (*keychainManager, error) {
	return &keychainManager{}, nil
}

func (m *keychainManager) SaveAWS(_ context.Context, creds AWSCredentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return m.saveItem(AWSAccount, data)
}

func (m *keychainManager) LoadAWS(_ context.Context) (*AWSCredentials, error) {
	data, err := m.loadItem(AWSAccount)
	if err != nil {
		return nil, err
	}

	var creds AWSCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func (m *keychainManager) DeleteAWS(_ context.Context) error {
	return m.deleteItem(AWSAccount)
}

func (m *keychainManager) SaveHetzner(_ context.Context, creds HetznerCredentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return m.saveItem(HetznerAccount, data)
}

func (m *keychainManager) LoadHetzner(_ context.Context) (*HetznerCredentials, error) {
	data, err := m.loadItem(HetznerAccount)
	if err != nil {
		return nil, err
	}

	var creds HetznerCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

func (m *keychainManager) DeleteHetzner(_ context.Context) error {
	return m.deleteItem(HetznerAccount)
}

func (m *keychainManager) HasCredentials(ctx context.Context, provider string) bool {
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

// saveItem saves data to the keychain for the given account
func (m *keychainManager) saveItem(account string, data []byte) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(ServiceName)
	item.SetAccount(account)
	item.SetData(data)
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)

	// Delete existing item first (to handle update case)
	_ = m.deleteItem(account)

	return keychain.AddItem(item)
}

// loadItem retrieves data from the keychain for the given account
func (m *keychainManager) loadItem(account string) ([]byte, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(ServiceName)
	query.SetAccount(account)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		// Convert keychain "not found" error to our ErrNotFound
		if errors.Is(err, keychain.ErrorItemNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrNotFound
	}

	return results[0].Data, nil
}

// deleteItem removes an item from the keychain for the given account
func (m *keychainManager) deleteItem(account string) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(ServiceName)
	item.SetAccount(account)

	err := keychain.DeleteItem(item)
	if err != nil {
		// Don't report error if item wasn't found
		if errors.Is(err, keychain.ErrorItemNotFound) {
			return nil
		}
		return err
	}
	return nil
}
