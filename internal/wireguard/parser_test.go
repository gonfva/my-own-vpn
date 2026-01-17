package wireguard

import (
	"testing"
	"time"
)

func TestParseBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint64
	}{
		{
			name:     "bytes",
			input:    "100 B",
			expected: 100,
		},
		{
			name:     "kibibytes",
			input:    "1 KiB",
			expected: 1024,
		},
		{
			name:     "mebibytes",
			input:    "1 MiB",
			expected: 1024 * 1024,
		},
		{
			name:     "gibibytes",
			input:    "1 GiB",
			expected: 1024 * 1024 * 1024,
		},
		{
			name:     "tebibytes",
			input:    "1 TiB",
			expected: 1024 * 1024 * 1024 * 1024,
		},
		{
			name:     "fractional kibibytes",
			input:    "1.5 KiB",
			expected: 1536,
		},
		{
			name:     "fractional mebibytes",
			input:    "2.5 MiB",
			expected: 2621440,
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: 0,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "unknown unit",
			input:    "100 XB",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBytes(tt.input)
			if result != tt.expected {
				t.Errorf("parseBytes(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{
			name:     "seconds only",
			input:    "30 seconds",
			expected: 30 * time.Second,
		},
		{
			name:     "single second",
			input:    "1 second",
			expected: 1 * time.Second,
		},
		{
			name:     "minutes only",
			input:    "5 minutes",
			expected: 5 * time.Minute,
		},
		{
			name:     "single minute",
			input:    "1 minute",
			expected: 1 * time.Minute,
		},
		{
			name:     "hours only",
			input:    "2 hours",
			expected: 2 * time.Hour,
		},
		{
			name:     "single hour",
			input:    "1 hour",
			expected: 1 * time.Hour,
		},
		{
			name:     "days only",
			input:    "3 days",
			expected: 3 * 24 * time.Hour,
		},
		{
			name:     "single day",
			input:    "1 day",
			expected: 24 * time.Hour,
		},
		{
			name:     "minutes and seconds",
			input:    "1 minute, 5 seconds",
			expected: 1*time.Minute + 5*time.Second,
		},
		{
			name:     "hours minutes and seconds",
			input:    "2 hours, 30 minutes, 45 seconds",
			expected: 2*time.Hour + 30*time.Minute + 45*time.Second,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "invalid format",
			input:    "invalid",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDuration(tt.input)
			if result != tt.expected {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseTransferLine(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedSent     uint64
		expectedReceived uint64
	}{
		{
			name:             "normal transfer line",
			input:            "transfer: 1.23 MiB received, 456.78 KiB sent",
			expectedSent:     467742,  // 456.78 * 1024
			expectedReceived: 1289748, // 1.23 * 1024 * 1024
		},
		{
			name:             "small transfer",
			input:            "transfer: 100 B received, 50 B sent",
			expectedSent:     50,
			expectedReceived: 100,
		},
		{
			name:             "large transfer",
			input:            "transfer: 1 GiB received, 500 MiB sent",
			expectedSent:     500 * 1024 * 1024,
			expectedReceived: 1024 * 1024 * 1024,
		},
		{
			name:             "empty line",
			input:            "transfer:",
			expectedSent:     0,
			expectedReceived: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sent, received := parseTransferLine(tt.input)
			if sent != tt.expectedSent {
				t.Errorf("parseTransferLine(%q) sent = %d, want %d", tt.input, sent, tt.expectedSent)
			}
			if received != tt.expectedReceived {
				t.Errorf("parseTransferLine(%q) received = %d, want %d", tt.input, received, tt.expectedReceived)
			}
		})
	}
}

func TestParseHandshakeLine(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectZero  bool
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{
			name:        "recent handshake",
			input:       "latest handshake: 30 seconds ago",
			expectZero:  false,
			minDuration: 29 * time.Second,
			maxDuration: 31 * time.Second,
		},
		{
			name:        "minute handshake",
			input:       "latest handshake: 1 minute, 5 seconds ago",
			expectZero:  false,
			minDuration: 64 * time.Second,
			maxDuration: 66 * time.Second,
		},
		{
			name:       "no handshake",
			input:      "latest handshake: (none)",
			expectZero: true,
		},
		{
			name:       "empty handshake",
			input:      "latest handshake:",
			expectZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHandshakeLine(tt.input)
			if tt.expectZero {
				if !result.IsZero() {
					t.Errorf("parseHandshakeLine(%q) = %v, want zero time", tt.input, result)
				}
			} else {
				if result.IsZero() {
					t.Errorf("parseHandshakeLine(%q) = zero time, want non-zero", tt.input)
					return
				}
				elapsed := time.Since(result)
				if elapsed < tt.minDuration || elapsed > tt.maxDuration {
					t.Errorf("parseHandshakeLine(%q) elapsed = %v, want between %v and %v",
						tt.input, elapsed, tt.minDuration, tt.maxDuration)
				}
			}
		})
	}
}

func TestParseWgShowOutput(t *testing.T) {
	sampleOutput := `interface: my-own-vpn
  public key: somepublickey123456789012345678901234567890==
  private key: (hidden)
  listening port: 51820

peer: serverpublickey1234567890123456789012345678901234==
  endpoint: 192.168.1.100:51820
  allowed ips: 0.0.0.0/0, ::/0
  latest handshake: 1 minute, 30 seconds ago
  transfer: 10 MiB received, 5 MiB sent
`

	sent, received, handshake := parseWgShowOutput(sampleOutput)

	expectedSent := uint64(5 * 1024 * 1024)
	expectedReceived := uint64(10 * 1024 * 1024)

	if sent != expectedSent {
		t.Errorf("parseWgShowOutput sent = %d, want %d", sent, expectedSent)
	}
	if received != expectedReceived {
		t.Errorf("parseWgShowOutput received = %d, want %d", received, expectedReceived)
	}
	if handshake.IsZero() {
		t.Error("parseWgShowOutput handshake is zero, want non-zero")
	}

	// Handshake should be approximately 1 minute 30 seconds ago
	elapsed := time.Since(handshake)
	if elapsed < 89*time.Second || elapsed > 91*time.Second {
		t.Errorf("parseWgShowOutput handshake elapsed = %v, want ~90 seconds", elapsed)
	}
}

func TestParseWgShowOutputEmpty(t *testing.T) {
	sent, received, handshake := parseWgShowOutput("")

	if sent != 0 {
		t.Errorf("parseWgShowOutput empty sent = %d, want 0", sent)
	}
	if received != 0 {
		t.Errorf("parseWgShowOutput empty received = %d, want 0", received)
	}
	if !handshake.IsZero() {
		t.Errorf("parseWgShowOutput empty handshake = %v, want zero", handshake)
	}
}

func TestParseWgShowOutputNoHandshake(t *testing.T) {
	sampleOutput := `interface: my-own-vpn
  public key: somepublickey123456789012345678901234567890==
  private key: (hidden)
  listening port: 51820

peer: serverpublickey1234567890123456789012345678901234==
  endpoint: 192.168.1.100:51820
  allowed ips: 0.0.0.0/0, ::/0
  latest handshake: (none)
  transfer: 0 B received, 0 B sent
`

	sent, received, handshake := parseWgShowOutput(sampleOutput)

	if sent != 0 {
		t.Errorf("parseWgShowOutput no handshake sent = %d, want 0", sent)
	}
	if received != 0 {
		t.Errorf("parseWgShowOutput no handshake received = %d, want 0", received)
	}
	if !handshake.IsZero() {
		t.Errorf("parseWgShowOutput no handshake handshake = %v, want zero", handshake)
	}
}
