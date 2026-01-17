package wireguard

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// Config represents a WireGuard client configuration.
type Config struct {
	// PrivateKey is the client's private key (base64 encoded)
	PrivateKey string
	// Address is the client's IP address in CIDR notation (e.g., "10.0.0.2/32")
	Address string
	// DNS is a list of DNS servers to use when connected
	DNS []string
	// ServerPublicKey is the server's public key (base64 encoded)
	ServerPublicKey string
	// ServerEndpoint is the server's address in IP:Port format
	ServerEndpoint string
	// AllowedIPs specifies which traffic should be routed through the VPN
	AllowedIPs []string
}

const configTemplate = `[Interface]
PrivateKey = {{.PrivateKey}}
Address = {{.Address}}
DNS = {{range $i, $dns := .DNS}}{{if $i}}, {{end}}{{$dns}}{{end}}

[Peer]
PublicKey = {{.ServerPublicKey}}
AllowedIPs = {{range $i, $ip := .AllowedIPs}}{{if $i}}, {{end}}{{$ip}}{{end}}
Endpoint = {{.ServerEndpoint}}
PersistentKeepalive = 25
`

// Generate creates the WireGuard configuration file content from the template.
func (c *Config) Generate() (string, error) {
	tmpl, err := template.New("wg").Parse(configTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse config template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, c); err != nil {
		return "", fmt.Errorf("failed to execute config template: %w", err)
	}

	return buf.String(), nil
}

// WriteToFile writes the configuration to a file with secure permissions (0600).
// It creates any necessary parent directories.
func (c *Config) WriteToFile(path string) error {
	content, err := c.Generate()
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	// #nosec G301 - Configuration directory needs to be accessible
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write with secure permissions (owner read/write only)
	// #nosec G306 - Intentionally restrictive permissions for sensitive config
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultConfig returns a Config with sensible defaults for a full-tunnel VPN.
// The PrivateKey, ServerPublicKey, and ServerEndpoint must still be set.
func DefaultConfig() *Config {
	return &Config{
		Address:    "10.0.0.2/32",
		DNS:        []string{"1.1.1.1", "1.0.0.1"},
		AllowedIPs: []string{"0.0.0.0/0", "::/0"},
	}
}
