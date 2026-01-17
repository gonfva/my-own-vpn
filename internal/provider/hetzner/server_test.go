package hetzner

import (
	"strings"
	"testing"
)

func TestGenerateSSHKeyPair(t *testing.T) {
	keyPair, err := generateSSHKeyPair()
	if err != nil {
		t.Fatalf("generateSSHKeyPair() returned error: %v", err)
	}

	if keyPair == nil {
		t.Fatal("generateSSHKeyPair() returned nil")
	}

	// Check private key is PEM encoded
	if len(keyPair.privateKey) == 0 {
		t.Error("private key is empty")
	}
	if !strings.Contains(string(keyPair.privateKey), "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Error("private key is not in PEM format")
	}

	// Check public key is in OpenSSH format
	if len(keyPair.publicKey) == 0 {
		t.Error("public key is empty")
	}
	if !strings.HasPrefix(keyPair.publicKey, "ssh-ed25519 ") {
		t.Error("public key is not in OpenSSH format")
	}

	// Check signer is available
	if keyPair.signer == nil {
		t.Error("signer is nil")
	}
}

func TestGenerateSSHKeyPairUniqueness(t *testing.T) {
	keyPair1, err := generateSSHKeyPair()
	if err != nil {
		t.Fatalf("generateSSHKeyPair() returned error: %v", err)
	}

	keyPair2, err := generateSSHKeyPair()
	if err != nil {
		t.Fatalf("generateSSHKeyPair() returned error: %v", err)
	}

	if keyPair1.publicKey == keyPair2.publicKey {
		t.Error("expected different public keys")
	}

	if string(keyPair1.privateKey) == string(keyPair2.privateKey) {
		t.Error("expected different private keys")
	}
}

func TestGenerateUserData(t *testing.T) {
	userData := generateUserData()

	if len(userData) == 0 {
		t.Error("user data is empty")
	}

	// Check for expected content
	expectedContents := []string{
		"#!/bin/bash",
		"apt-get install -y wireguard",
		"wg genkey",
		"/etc/wireguard/wg0.conf",
		"Address = 10.0.0.1/24",
		"ListenPort = 51820",
		"WIREGUARD_PUBKEY=",
		"WIREGUARD_READY=true",
		"systemctl enable wg-quick@wg0",
		"systemctl start wg-quick@wg0",
		"net.ipv4.ip_forward=1",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(userData, expected) {
			t.Errorf("user data missing expected content: %s", expected)
		}
	}
}
