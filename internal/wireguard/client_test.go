package wireguard

import (
	"testing"
	"time"
)

func TestConnectionStatusIsActive(t *testing.T) {
	tests := []struct {
		name     string
		status   ConnectionStatus
		expected bool
	}{
		{
			name: "active connection with recent handshake",
			status: ConnectionStatus{
				Connected:     true,
				LastHandshake: time.Now().Add(-1 * time.Minute),
			},
			expected: true,
		},
		{
			name: "not connected",
			status: ConnectionStatus{
				Connected:     false,
				LastHandshake: time.Now(),
			},
			expected: false,
		},
		{
			name: "connected but stale handshake",
			status: ConnectionStatus{
				Connected:     true,
				LastHandshake: time.Now().Add(-5 * time.Minute),
			},
			expected: false,
		},
		{
			name: "connected but no handshake yet",
			status: ConnectionStatus{
				Connected:     true,
				LastHandshake: time.Time{}, // zero value
			},
			expected: false,
		},
		{
			name: "connected with handshake exactly at 3 minute threshold",
			status: ConnectionStatus{
				Connected:     true,
				LastHandshake: time.Now().Add(-3*time.Minute - time.Second),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsActive(); got != tt.expected {
				t.Errorf("IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}
