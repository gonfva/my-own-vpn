//go:build windows

package credentials

import (
	"context"
	"encoding/json"

	"github.com/danieljoos/wincred"
)

// wincredManager implements Manager using Windows Credential Manager
type wincredManager struct{}

// newWincredManager creates a new wincredManager instance
func newWincredManager() (*wincredManager, error) {
	return &wincredManager{}, nil
}

func (m *wincredManager) SaveAWS(_ context.Context, creds AWSCredentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return m.saveItem(AWSAccount, data)
}

func (m *wincredManager) LoadAWS(_ context.Context) (*AWSCredentials, error) {
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

func (m *wincredManager) DeleteAWS(_ context.Context) error {
	return m.deleteItem(AWSAccount)
}

func (m *wincredManager) SaveHetzner(_ context.Context, creds HetznerCredentials) error {
	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return m.saveItem(HetznerAccount, data)
}

func (m *wincredManager) LoadHetzner(_ context.Context) (*HetznerCredentials, error) {
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

func (m *wincredManager) DeleteHetzner(_ context.Context) error {
	return m.deleteItem(HetznerAccount)
}

func (m *wincredManager) HasCredentials(ctx context.Context, provider string) bool {
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

// targetName returns the Windows Credential Manager target name for the given account
func (m *wincredManager) targetName(account string) string {
	return ServiceName + ":" + account
}

// saveItem saves data to Windows Credential Manager for the given account
func (m *wincredManager) saveItem(account string, data []byte) error {
	cred := wincred.NewGenericCredential(m.targetName(account))
	cred.CredentialBlob = data
	return cred.Write()
}

// loadItem retrieves data from Windows Credential Manager for the given account
func (m *wincredManager) loadItem(account string) ([]byte, error) {
	cred, err := wincred.GetGenericCredential(m.targetName(account))
	if err != nil {
		// Windows returns an error when credential is not found
		return nil, ErrNotFound
	}

	return cred.CredentialBlob, nil
}

// deleteItem removes an item from Windows Credential Manager for the given account
func (m *wincredManager) deleteItem(account string) error {
	cred, err := wincred.GetGenericCredential(m.targetName(account))
	if err != nil {
		// Don't report error if credential doesn't exist
		return nil
	}
	return cred.Delete()
}
