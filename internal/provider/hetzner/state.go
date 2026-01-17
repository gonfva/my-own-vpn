package hetzner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// ProvisionState stores the IDs of all provisioned resources for crash recovery.
// This state is persisted to disk so resources can be cleaned up even after app restart.
type ProvisionState struct {
	SessionID       string `json:"session_id"`
	ServerID        int64  `json:"server_id,omitempty"`
	FirewallID      int64  `json:"firewall_id,omitempty"`
	SSHKeyID        int64  `json:"ssh_key_id,omitempty"`
	ServerPublicIP  string `json:"server_public_ip,omitempty"`
	ServerPublicKey string `json:"server_public_key,omitempty"`
}

const (
	// stateFileName is the name of the state file
	stateFileName = "hetzner_provision_state.json"

	// labelKeyApplication is the label key for application identification
	labelKeyApplication = "application"

	// labelKeyManagedBy is the label key for management identification
	labelKeyManagedBy = "managed-by"

	// labelKeySessionID is the label key for session identification
	labelKeySessionID = "session-id"

	// labelValueApplication is the value for the Application label
	labelValueApplication = "my-own-vpn"
)

// generateSessionID creates a unique session identifier.
func generateSessionID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// getLabels returns the standard labels applied to all resources.
func (p *Provider) getLabels() map[string]string {
	labels := map[string]string{
		labelKeyApplication: labelValueApplication,
		labelKeyManagedBy:   labelValueApplication,
	}

	if p.sessionID != "" {
		labels[labelKeySessionID] = p.sessionID
	}

	return labels
}

// getStateFilePath returns the path to the state file.
// It uses the user's config directory to store the state.
func getStateFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	appDir := filepath.Clean(filepath.Join(configDir, "my-own-vpn"))
	if err := os.MkdirAll(appDir, 0o700); err != nil { // #nosec G301 - intentional directory permissions
		return "", fmt.Errorf("failed to create app directory: %w", err)
	}

	return filepath.Clean(filepath.Join(appDir, stateFileName)), nil
}

// saveState persists the current provision state to disk.
func (p *Provider) saveState() error {
	state := ProvisionState{
		SessionID:       p.sessionID,
		ServerID:        p.serverID,
		FirewallID:      p.firewallID,
		SSHKeyID:        p.sshKeyID,
		ServerPublicIP:  p.serverPublicIP,
		ServerPublicKey: p.serverPublicKey,
	}

	filePath, err := getStateFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil { // #nosec G306 - intentional file permissions
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// loadState loads the provision state from disk if it exists.
func loadState() (*ProvisionState, error) {
	filePath, err := getStateFilePath()
	if err != nil {
		return nil, err
	}

	// #nosec G304 - filePath is constructed from system config dir and a hardcoded constant
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No state file exists
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ProvisionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// clearState removes the state file from disk.
func clearState() error {
	filePath, err := getStateFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	return nil
}

// restoreFromState restores the provider's internal state from a ProvisionState.
func (p *Provider) restoreFromState(state *ProvisionState) {
	p.sessionID = state.SessionID
	p.serverID = state.ServerID
	p.firewallID = state.FirewallID
	p.sshKeyID = state.SSHKeyID
	p.serverPublicIP = state.ServerPublicIP
	p.serverPublicKey = state.ServerPublicKey
}

// findResourcesByLabel discovers existing resources created by this application
// using the managed-by label. This allows cleanup of orphaned resources.
func (p *Provider) findResourcesByLabel(ctx context.Context) error {
	labelSelector := fmt.Sprintf("%s=%s", labelKeyManagedBy, labelValueApplication)

	// Find Servers
	servers, _, err := p.client.Server.List(ctx, hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: labelSelector,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}
	if len(servers) > 0 {
		p.serverID = servers[0].ID
		if servers[0].PublicNet.IPv4.IP != nil {
			p.serverPublicIP = servers[0].PublicNet.IPv4.IP.String()
		}
		// Extract session ID from labels if available
		if sessionID, ok := servers[0].Labels[labelKeySessionID]; ok {
			p.sessionID = sessionID
		}
	}

	// Find Firewalls
	firewalls, _, err := p.client.Firewall.List(ctx, hcloud.FirewallListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: labelSelector,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list firewalls: %w", err)
	}
	if len(firewalls) > 0 {
		p.firewallID = firewalls[0].ID
	}

	// Find SSH Keys
	sshKeys, _, err := p.client.SSHKey.List(ctx, hcloud.SSHKeyListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: labelSelector,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list SSH keys: %w", err)
	}
	if len(sshKeys) > 0 {
		p.sshKeyID = sshKeys[0].ID
	}

	return nil
}

// HasProvisionedResources checks if there are any provisioned resources
// either in memory or persisted in the state file.
func (p *Provider) HasProvisionedResources() bool {
	return p.serverID != 0 || p.firewallID != 0 || p.sshKeyID != 0
}

// LoadExistingState attempts to load existing state from disk and restore it.
// Returns true if state was found and loaded.
func (p *Provider) LoadExistingState() (bool, error) {
	state, err := loadState()
	if err != nil {
		return false, err
	}
	if state == nil {
		return false, nil
	}

	p.restoreFromState(state)
	return true, nil
}
