package hetzner

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/gonfva/my-own-vpn/internal/provider"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"golang.org/x/crypto/ssh"
)

// Server constants
const (
	// defaultServerType is the default Hetzner server type
	defaultServerType = "cx11"

	// defaultImage is the Ubuntu image to use
	defaultImage = "ubuntu-22.04"

	// wireGuardPort is the UDP port for WireGuard
	wireGuardPort = 51820

	// wireGuardReadyTimeout is the maximum time to wait for WireGuard to be ready
	wireGuardReadyTimeout = 5 * time.Minute

	// wireGuardPollInterval is how often to check if WireGuard is ready
	wireGuardPollInterval = 10 * time.Second

	// serverReadyTimeout is the maximum time to wait for server to be running
	serverReadyTimeout = 5 * time.Minute

	// serverPollInterval is how often to check server status
	serverPollInterval = 5 * time.Second
)

// generateUserData returns the cloud-init script to install and configure WireGuard.
func generateUserData() string {
	return `#!/bin/bash
set -e

# Log output for debugging
exec > >(tee /var/log/wireguard-setup.log) 2>&1
echo "Starting WireGuard setup..."

# Install WireGuard
apt-get update
apt-get install -y wireguard

# Generate server keys
wg genkey | tee /etc/wireguard/server_private.key | wg pubkey > /etc/wireguard/server_public.key
chmod 600 /etc/wireguard/server_private.key

# Get server private key
SERVER_PRIVATE=$(cat /etc/wireguard/server_private.key)

# Get the primary network interface
PRIMARY_IFACE=$(ip route | grep default | awk '{print $5}' | head -n1)

# Create WireGuard config
cat > /etc/wireguard/wg0.conf << WGEOF
[Interface]
Address = 10.0.0.1/24
ListenPort = 51820
PrivateKey = $SERVER_PRIVATE
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o $PRIMARY_IFACE -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o $PRIMARY_IFACE -j MASQUERADE
WGEOF

chmod 600 /etc/wireguard/wg0.conf

# Enable IP forwarding
echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
sysctl -p

# Start WireGuard
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0

# Output public key for retrieval
SERVER_PUBKEY=$(cat /etc/wireguard/server_public.key)
echo "WIREGUARD_PUBKEY=$SERVER_PUBKEY"
echo "WIREGUARD_READY=true"
echo "WireGuard setup complete!"
`
}

// sshKeyPair holds the generated SSH key pair
type sshKeyPair struct {
	privateKey []byte // PEM-encoded private key
	publicKey  string // OpenSSH format public key
	signer     ssh.Signer
}

// generateSSHKeyPair generates an Ed25519 SSH key pair for server access.
func generateSSHKeyPair() (*sshKeyPair, error) {
	// Generate Ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}

	// Convert to SSH public key
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH public key: %w", err)
	}
	publicKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey)))

	// Convert private key to PEM format
	privKeyBytes, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	privateKeyPEM := pem.EncodeToMemory(privKeyBytes)

	// Create SSH signer
	signer, err := ssh.NewSignerFromKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH signer: %w", err)
	}

	return &sshKeyPair{
		privateKey: privateKeyPEM,
		publicKey:  publicKey,
		signer:     signer,
	}, nil
}

// createSSHKey creates an SSH key in Hetzner Cloud for server access.
// The private key is stored temporarily for retrieving the WireGuard public key.
func (p *Provider) createSSHKey(ctx context.Context) error {
	keyPair, err := generateSSHKeyPair()
	if err != nil {
		return err
	}

	keyName := fmt.Sprintf("my-own-vpn-%s", p.sessionID)

	result, _, err := p.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      keyName,
		PublicKey: keyPair.publicKey,
		Labels:    p.getLabels(),
	})
	if err != nil {
		return fmt.Errorf("failed to create SSH key: %w", err)
	}

	p.sshKeyID = result.ID
	p.sshKey = keyPair
	return nil
}

// deleteSSHKey deletes the SSH key if it exists.
func (p *Provider) deleteSSHKey(ctx context.Context) error {
	if p.sshKeyID == 0 {
		return nil
	}

	_, err := p.client.SSHKey.Delete(ctx, &hcloud.SSHKey{ID: p.sshKeyID})
	if err != nil {
		// Check if already deleted
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			p.sshKeyID = 0
			return nil
		}
		return fmt.Errorf("failed to delete SSH key: %w", err)
	}

	p.sshKeyID = 0
	return nil
}

// createServer creates a Hetzner server with WireGuard configured via cloud-init.
func (p *Provider) createServer(ctx context.Context, cfg provider.ProvisionConfig) error {
	if p.sshKeyID == 0 {
		return fmt.Errorf("SSH key must be created before creating server")
	}

	// Use default server type if not specified
	serverType := cfg.InstanceType
	if serverType == "" {
		serverType = defaultServerType
	}

	// Use default location if not specified
	location := cfg.Region
	if location == "" {
		location = "fsn1" // Falkenstein, Germany
	}

	// Get the server type
	serverTypeObj, _, err := p.client.ServerType.GetByName(ctx, serverType)
	if err != nil {
		return fmt.Errorf("failed to get server type: %w", err)
	}
	if serverTypeObj == nil {
		return fmt.Errorf("server type %s not found", serverType)
	}

	// Get the image
	image, _, err := p.client.Image.GetByNameAndArchitecture(ctx, defaultImage, serverTypeObj.Architecture)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}
	if image == nil {
		return fmt.Errorf("image %s not found for architecture %s", defaultImage, serverTypeObj.Architecture)
	}

	// Get the location
	loc, _, err := p.client.Location.GetByName(ctx, location)
	if err != nil {
		return fmt.Errorf("failed to get location: %w", err)
	}
	if loc == nil {
		return fmt.Errorf("location %s not found", location)
	}

	// Get the SSH key
	sshKey, _, err := p.client.SSHKey.GetByID(ctx, p.sshKeyID)
	if err != nil {
		return fmt.Errorf("failed to get SSH key: %w", err)
	}

	// Get the firewall
	var firewalls []*hcloud.ServerCreateFirewall
	if p.firewallID != 0 {
		firewalls = []*hcloud.ServerCreateFirewall{
			{Firewall: hcloud.Firewall{ID: p.firewallID}},
		}
	}

	// Create server
	result, _, err := p.client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       fmt.Sprintf("my-own-vpn-%s", p.sessionID),
		ServerType: serverTypeObj,
		Image:      image,
		Location:   loc,
		SSHKeys:    []*hcloud.SSHKey{sshKey},
		Firewalls:  firewalls,
		UserData:   generateUserData(),
		Labels:     p.getLabels(),
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	p.serverID = result.Server.ID

	// Wait for server action to complete
	if result.Action != nil {
		if err := p.client.Action.WaitForFunc(ctx, nil, result.Action); err != nil {
			return fmt.Errorf("failed waiting for server creation: %w", err)
		}
	}

	// Wait for server to be running
	if err := p.waitForServerRunning(ctx); err != nil {
		return err
	}

	// Get the server's public IP
	server, _, err := p.client.Server.GetByID(ctx, p.serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	if server.PublicNet.IPv4.IP != nil {
		p.serverPublicIP = server.PublicNet.IPv4.IP.String()
	} else {
		return fmt.Errorf("server has no public IPv4 address")
	}

	return nil
}

// waitForServerRunning waits for the server to reach the running state.
func (p *Provider) waitForServerRunning(ctx context.Context) error {
	if p.serverID == 0 {
		return fmt.Errorf("no server to wait for")
	}

	deadline := time.Now().Add(serverReadyTimeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		server, _, err := p.client.Server.GetByID(ctx, p.serverID)
		if err != nil {
			time.Sleep(serverPollInterval)
			continue
		}

		if server != nil && server.Status == hcloud.ServerStatusRunning {
			return nil
		}

		time.Sleep(serverPollInterval)
	}

	return fmt.Errorf("timeout waiting for server to be running")
}

// waitForWireGuardReady waits for WireGuard to be ready by polling the server via SSH.
// Since Hetzner doesn't provide a console output API like AWS, we use SSH to check.
func (p *Provider) waitForWireGuardReady(ctx context.Context) (string, error) {
	if p.serverID == 0 {
		return "", fmt.Errorf("no server ID")
	}

	if p.serverPublicIP == "" {
		return "", fmt.Errorf("server IP not available")
	}

	deadline := time.Now().Add(wireGuardReadyTimeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Try to retrieve the WireGuard public key via SSH
		pubKey, err := p.retrievePublicKeyViaSSH(ctx)
		if err == nil && pubKey != "" {
			return pubKey, nil
		}

		time.Sleep(wireGuardPollInterval)
	}

	return "", fmt.Errorf("timeout waiting for WireGuard to be ready")
}

// retrievePublicKeyViaSSH connects to the server via SSH to retrieve the WireGuard public key.
func (p *Provider) retrievePublicKeyViaSSH(ctx context.Context) (string, error) {
	if p.sshKey == nil || p.sshKey.signer == nil {
		return "", fmt.Errorf("SSH key not available")
	}

	if p.serverPublicIP == "" {
		return "", fmt.Errorf("server IP not available")
	}

	// SSH configuration
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(p.sshKey.signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // #nosec G106 - acceptable for ephemeral VPN servers
		Timeout:         30 * time.Second,
	}

	// Connect to server
	addr := fmt.Sprintf("%s:22", p.serverPublicIP)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("failed to dial SSH: %w", err)
	}
	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Run the command to get the WireGuard public key
	output, err := session.CombinedOutput("cat /etc/wireguard/server_public.key 2>/dev/null")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve WireGuard public key: %w", err)
	}

	pubKey := strings.TrimSpace(string(output))
	if len(pubKey) != 44 { // WireGuard public keys are 44 characters (base64 encoded 32 bytes)
		return "", fmt.Errorf("invalid WireGuard public key length: %d", len(pubKey))
	}

	return pubKey, nil
}

// deleteServer deletes the Hetzner server if it exists.
func (p *Provider) deleteServer(ctx context.Context) error {
	if p.serverID == 0 {
		return nil
	}

	result, _, err := p.client.Server.DeleteWithResult(ctx, &hcloud.Server{ID: p.serverID})
	if err != nil {
		// Check if already deleted
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			p.serverID = 0
			p.serverPublicIP = ""
			return nil
		}
		return fmt.Errorf("failed to delete server: %w", err)
	}

	// Wait for deletion to complete
	if result.Action != nil {
		if err := p.client.Action.WaitForFunc(ctx, nil, result.Action); err != nil {
			return fmt.Errorf("failed waiting for server deletion: %w", err)
		}
	}

	p.serverID = 0
	p.serverPublicIP = ""
	return nil
}
