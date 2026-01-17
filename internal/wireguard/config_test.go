package wireguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigGenerate(t *testing.T) {
	cfg := &Config{
		PrivateKey:      "cGFzc3dvcmQxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ=",
		Address:         "10.0.0.2/32",
		DNS:             []string{"1.1.1.1"},
		ServerPublicKey: "c2VydmVya2V5MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
		ServerEndpoint:  "1.2.3.4:51820",
		AllowedIPs:      []string{"0.0.0.0/0"},
	}

	output, err := cfg.Generate()
	if err != nil {
		t.Fatalf("config generation failed: %v", err)
	}

	// Verify output contains expected values
	checks := []string{
		"[Interface]",
		"PrivateKey = cGFzc3dvcmQxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ=",
		"Address = 10.0.0.2/32",
		"DNS = 1.1.1.1",
		"[Peer]",
		"PublicKey = c2VydmVya2V5MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
		"AllowedIPs = 0.0.0.0/0",
		"Endpoint = 1.2.3.4:51820",
		"PersistentKeepalive = 25",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output missing expected content: %s", check)
		}
	}
}

func TestConfigGenerateMultipleDNS(t *testing.T) {
	cfg := &Config{
		PrivateKey:      "cGFzc3dvcmQxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ=",
		Address:         "10.0.0.2/32",
		DNS:             []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"},
		ServerPublicKey: "c2VydmVya2V5MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
		ServerEndpoint:  "1.2.3.4:51820",
		AllowedIPs:      []string{"0.0.0.0/0"},
	}

	output, err := cfg.Generate()
	if err != nil {
		t.Fatalf("config generation failed: %v", err)
	}

	if !strings.Contains(output, "DNS = 1.1.1.1, 8.8.8.8, 9.9.9.9") {
		t.Error("multiple DNS servers not formatted correctly")
	}
}

func TestConfigGenerateMultipleAllowedIPs(t *testing.T) {
	cfg := &Config{
		PrivateKey:      "cGFzc3dvcmQxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ=",
		Address:         "10.0.0.2/32",
		DNS:             []string{"1.1.1.1"},
		ServerPublicKey: "c2VydmVya2V5MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
		ServerEndpoint:  "1.2.3.4:51820",
		AllowedIPs:      []string{"0.0.0.0/0", "::/0"},
	}

	output, err := cfg.Generate()
	if err != nil {
		t.Fatalf("config generation failed: %v", err)
	}

	if !strings.Contains(output, "AllowedIPs = 0.0.0.0/0, ::/0") {
		t.Error("multiple AllowedIPs not formatted correctly")
	}
}

func TestConfigWriteToFile(t *testing.T) {
	cfg := &Config{
		PrivateKey:      "cGFzc3dvcmQxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ=",
		Address:         "10.0.0.2/32",
		DNS:             []string{"1.1.1.1"},
		ServerPublicKey: "c2VydmVya2V5MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM=",
		ServerEndpoint:  "1.2.3.4:51820",
		AllowedIPs:      []string{"0.0.0.0/0"},
	}

	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "wireguard-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "subdir", "wg0.conf")

	err = cfg.WriteToFile(configPath)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Verify the file exists
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	// Verify permissions are 0600
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
	}

	// Verify the content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	if !strings.Contains(string(content), "[Interface]") {
		t.Error("config file missing expected content")
	}
}

func TestConfigWriteToFileCreatesParentDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wireguard-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a nested path that doesn't exist
	configPath := filepath.Join(tmpDir, "deep", "nested", "path", "wg0.conf")

	cfg := DefaultConfig()
	cfg.PrivateKey = "cGFzc3dvcmQxMjM0NTY3ODkwMTIzNDU2Nzg5MDEyMzQ="
	cfg.ServerPublicKey = "c2VydmVya2V5MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM="
	cfg.ServerEndpoint = "1.2.3.4:51820"

	err = cfg.WriteToFile(configPath)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Verify parent directory was created
	parentDir := filepath.Dir(configPath)
	info, err := os.Stat(parentDir)
	if err != nil {
		t.Fatalf("parent directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("parent path is not a directory")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Address != "10.0.0.2/32" {
		t.Errorf("unexpected default address: %s", cfg.Address)
	}

	if len(cfg.DNS) != 2 {
		t.Errorf("expected 2 DNS servers, got %d", len(cfg.DNS))
	}

	if cfg.DNS[0] != "1.1.1.1" || cfg.DNS[1] != "1.0.0.1" {
		t.Error("unexpected default DNS servers")
	}

	if len(cfg.AllowedIPs) != 2 {
		t.Errorf("expected 2 allowed IPs, got %d", len(cfg.AllowedIPs))
	}

	// Check it includes both IPv4 and IPv6 catch-all
	foundIPv4 := false
	foundIPv6 := false
	for _, ip := range cfg.AllowedIPs {
		if ip == "0.0.0.0/0" {
			foundIPv4 = true
		}
		if ip == "::/0" {
			foundIPv6 = true
		}
	}

	if !foundIPv4 || !foundIPv6 {
		t.Error("default AllowedIPs should include both IPv4 and IPv6 catch-all")
	}
}
