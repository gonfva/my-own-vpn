package wireguard

import (
	"context"
	"time"

	"github.com/gonfva/my-own-vpn/internal/provider"
)

// Client defines the interface for WireGuard VPN operations.
// Platform-specific implementations will be provided in separate files.
type Client interface {
	// Connect establishes the VPN tunnel to the server.
	// The serverInfo contains the server's public IP, port, and public key.
	// The clientKeyPair is used to authenticate with the server.
	Connect(ctx context.Context, serverInfo *provider.ServerInfo, clientKeyPair *KeyPair) error

	// Disconnect tears down the VPN tunnel.
	Disconnect(ctx context.Context) error

	// Status returns the current connection status.
	Status() ConnectionStatus

	// GetPublicKey returns the client's public key.
	// This is needed to configure the server to accept this client.
	GetPublicKey() string
}

// ConnectionStatus represents the current state of the VPN connection.
type ConnectionStatus struct {
	// Connected indicates whether the VPN tunnel is active
	Connected bool
	// ServerIP is the public IP of the connected server
	ServerIP string
	// BytesSent is the total bytes sent through the tunnel
	BytesSent uint64
	// BytesReceived is the total bytes received through the tunnel
	BytesReceived uint64
	// LastHandshake is the time of the last successful handshake
	LastHandshake time.Time
}

// IsActive returns true if the connection is active and recent.
// A connection is considered active if it's connected and the last handshake
// was within the last 3 minutes (WireGuard considers a peer stale after this).
func (s ConnectionStatus) IsActive() bool {
	if !s.Connected {
		return false
	}
	if s.LastHandshake.IsZero() {
		return false
	}
	return time.Since(s.LastHandshake) < 3*time.Minute
}
