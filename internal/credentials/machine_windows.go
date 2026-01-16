//go:build windows

package credentials

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

// getMachineIdentifier returns a stable machine-specific identifier on Windows.
// It uses the MachineGuid from the registry, falling back to hostname+username if unavailable.
func getMachineIdentifier() string {
	// Try to read MachineGuid from registry
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err == nil {
		defer key.Close()
		guid, _, err := key.GetStringValue("MachineGuid")
		if err == nil && guid != "" {
			return guid
		}
	}

	// Fallback to hostname + username
	return getFallbackIdentifier()
}

// getFallbackIdentifier returns a fallback identifier using hostname and username
func getFallbackIdentifier() string {
	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")
	if username == "" {
		username = "unknown"
	}
	return hostname + ":" + username
}
