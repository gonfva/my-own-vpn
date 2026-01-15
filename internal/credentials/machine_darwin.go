//go:build darwin

package credentials

import (
	"os"
	"os/exec"
	"strings"
)

// getMachineIdentifier returns a stable machine-specific identifier on macOS.
// It uses the IOPlatformUUID from IOKit, falling back to hostname+username if unavailable.
func getMachineIdentifier() string {
	// Try to get IOPlatformUUID using ioreg
	cmd := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	output, err := cmd.Output()
	if err == nil {
		// Parse the output to find IOPlatformUUID
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "IOPlatformUUID") {
				// Format: "IOPlatformUUID" = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
				parts := strings.Split(line, "=")
				if len(parts) >= 2 {
					uuid := strings.TrimSpace(parts[1])
					uuid = strings.Trim(uuid, "\"")
					if uuid != "" {
						return uuid
					}
				}
			}
		}
	}

	// Fallback to hostname + username
	return getFallbackIdentifier()
}

// getFallbackIdentifier returns a fallback identifier using hostname and username
func getFallbackIdentifier() string {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = "unknown"
	}
	return hostname + ":" + username
}
