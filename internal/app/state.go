package app

// State represents the current state of the VPN connection lifecycle
type State int

const (
	// StateDisconnected indicates no active connection or provisioning
	StateDisconnected State = iota
	// StateProvisioning indicates cloud infrastructure is being created
	StateProvisioning
	// StateConnecting indicates WireGuard connection is being established
	StateConnecting
	// StateConnected indicates VPN connection is active
	StateConnected
	// StateDisconnecting indicates VPN connection is being torn down
	StateDisconnecting
	// StateDeprovisioning indicates cloud infrastructure is being destroyed
	StateDeprovisioning
	// StateError indicates an error occurred during operation
	StateError
)

// String returns a human-readable string representation of the State
func (s State) String() string {
	switch s {
	case StateDisconnected:
		return "Disconnected"
	case StateProvisioning:
		return "Provisioning"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateDisconnecting:
		return "Disconnecting"
	case StateDeprovisioning:
		return "Deprovisioning"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}
