//go:build windows

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
	tunnelName     = "my-own-vpn"
	wireguardExe   = "wireguard.exe"
	wgExe          = "wg.exe"
	configFileName = tunnelName + ".conf"
)

// windowsClient implements the Client interface for Windows using wireguard.exe CLI.
type windowsClient struct {
	mu         sync.RWMutex
	configPath string
	keyPair    *KeyPair
	serverIP   string
	connected  bool
}

// NewClient creates a new WireGuard client for Windows.
func NewClient() (Client, error) {
	return &windowsClient{}, nil
}

// Connect establishes the VPN tunnel using wireguard.exe.
// It generates the configuration file and installs the tunnel service.
func (c *windowsClient) Connect(ctx context.Context, serverInfo *provider.ServerInfo, clientKeyPair *KeyPair) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	// Get config directory
	configDir, err := c.getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	c.configPath = filepath.Join(configDir, configFileName)

	// Generate and write configuration
	config := DefaultConfig()
	config.PrivateKey = clientKeyPair.PrivateKeyString()
	config.ServerPublicKey = serverInfo.ServerPublicKey
	config.ServerEndpoint = fmt.Sprintf("%s:%d", serverInfo.PublicIP, serverInfo.WireGuardPort)

	if err := config.WriteToFile(c.configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Install tunnel service using wireguard.exe
	// #nosec G204 - configPath is constructed from known safe components
	cmd := exec.CommandContext(ctx, wireguardExe, "/installtunnelservice", c.configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up config file on failure
		_ = os.Remove(c.configPath)
		return fmt.Errorf("failed to install tunnel service: %w, output: %s", err, string(output))
	}

	c.keyPair = clientKeyPair
	c.serverIP = serverInfo.PublicIP
	c.connected = true

	return nil
}

// Disconnect tears down the VPN tunnel by uninstalling the tunnel service.
func (c *windowsClient) Disconnect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	// Uninstall tunnel service
	// #nosec G204 - tunnelName is a constant
	cmd := exec.CommandContext(ctx, wireguardExe, "/uninstalltunnelservice", tunnelName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to uninstall tunnel service: %w, output: %s", err, string(output))
	}

	// Clean up config file
	if c.configPath != "" {
		_ = os.Remove(c.configPath)
	}

	c.connected = false
	c.serverIP = ""

	return nil
}

// Status returns the current connection status by querying wg.exe.
func (c *windowsClient) Status() ConnectionStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := ConnectionStatus{
		Connected: c.connected,
		ServerIP:  c.serverIP,
	}

	if !c.connected {
		return status
	}

	// Query wg.exe for detailed status
	// #nosec G204 - tunnelName is a constant
	cmd := exec.Command(wgExe, "show", tunnelName)
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
func (c *windowsClient) GetPublicKey() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.keyPair == nil {
		return ""
	}
	return c.keyPair.PublicKeyString()
}

// getConfigDir returns the directory where WireGuard configs should be stored.
func (c *windowsClient) getConfigDir() (string, error) {
	// On Windows, WireGuard looks for configs in %LOCALAPPDATA%\WireGuard\Configurations
	// or we can specify the full path when installing the tunnel
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		return "", fmt.Errorf("LOCALAPPDATA environment variable not set")
	}

	configDir := filepath.Join(localAppData, "WireGuard", "Configurations")

	// Create directory if it doesn't exist
	// #nosec G301 - Configuration directory needs to be accessible
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}
