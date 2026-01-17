//go:build darwin

package wireguard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/gonfva/my-own-vpn/internal/provider"
)

const (
	tunnelNameDarwin = "my-own-vpn"
	wgQuickCmd       = "wg-quick"
	wgCmd            = "wg"
	configDirDarwin  = "/etc/wireguard"
)

// darwinClient implements the Client interface for macOS using wg-quick.
type darwinClient struct {
	mu         sync.RWMutex
	configPath string
	keyPair    *KeyPair
	serverIP   string
	connected  bool
}

// NewClient creates a new WireGuard client for macOS.
func NewClient() (Client, error) {
	return &darwinClient{}, nil
}

// Connect establishes the VPN tunnel using wg-quick.
// Note: This requires sudo privileges or the user to be in the appropriate group.
func (c *darwinClient) Connect(ctx context.Context, serverInfo *provider.ServerInfo, clientKeyPair *KeyPair) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	c.configPath = filepath.Join(configDirDarwin, tunnelNameDarwin+".conf")

	// Generate configuration
	config := DefaultConfig()
	config.PrivateKey = clientKeyPair.PrivateKeyString()
	config.ServerPublicKey = serverInfo.ServerPublicKey
	config.ServerEndpoint = fmt.Sprintf("%s:%d", serverInfo.PublicIP, serverInfo.WireGuardPort)

	// Write config file - this may require elevated privileges
	if err := config.WriteToFile(c.configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Bring up the tunnel using wg-quick
	// #nosec G204 - tunnelNameDarwin is a constant
	cmd := exec.CommandContext(ctx, "sudo", wgQuickCmd, "up", tunnelNameDarwin)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up config file on failure
		_ = os.Remove(c.configPath)
		return fmt.Errorf("failed to bring up tunnel: %w, output: %s", err, string(output))
	}

	c.keyPair = clientKeyPair
	c.serverIP = serverInfo.PublicIP
	c.connected = true

	return nil
}

// Disconnect tears down the VPN tunnel using wg-quick down.
func (c *darwinClient) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	// Bring down the tunnel
	// #nosec G204 - tunnelNameDarwin is a constant
	cmd := exec.CommandContext(ctx, "sudo", wgQuickCmd, "down", tunnelNameDarwin)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to bring down tunnel: %w, output: %s", err, string(output))
	}

	// Clean up config file
	if c.configPath != "" {
		_ = os.Remove(c.configPath)
	}

	c.connected = false
	c.serverIP = ""

	return nil
}

// Status returns the current connection status by querying wg show.
func (c *darwinClient) Status() ConnectionStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := ConnectionStatus{
		Connected: c.connected,
		ServerIP:  c.serverIP,
	}

	if !c.connected {
		return status
	}

	// Query wg for detailed status
	// #nosec G204 - tunnelNameDarwin is a constant
	cmd := exec.Command("sudo", wgCmd, "show", tunnelNameDarwin)
	output, err := cmd.Output()
	if err != nil {
		// If wg show fails, assume we're still connected but can't get stats
		return status
	}

	// Parse wg show output
	status.BytesSent, status.BytesReceived, status.LastHandshake = parseWgShowOutput(string(output))

	return status
}

// GetPublicKey returns the client's public key.
func (c *darwinClient) GetPublicKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.keyPair == nil {
		return ""
	}
	return c.keyPair.PublicKeyString()
}
