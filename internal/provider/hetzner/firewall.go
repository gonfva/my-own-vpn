package hetzner

import (
	"context"
	"fmt"
	"net"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// createFirewall creates a Hetzner Cloud firewall for the VPN server.
// The firewall only allows inbound UDP traffic on the WireGuard port.
func (p *Provider) createFirewall(ctx context.Context) error {
	// Create firewall rules
	// Allow inbound UDP 51820 (WireGuard)
	rules := []hcloud.FirewallRule{
		{
			Direction: hcloud.FirewallRuleDirectionIn,
			Protocol:  hcloud.FirewallRuleProtocolUDP,
			Port:      hcloud.Ptr("51820"),
			SourceIPs: []net.IPNet{
				{IP: net.ParseIP("0.0.0.0"), Mask: net.CIDRMask(0, 32)}, // IPv4 any
				{IP: net.ParseIP("::"), Mask: net.CIDRMask(0, 128)},     // IPv6 any
			},
			Description: hcloud.Ptr("Allow WireGuard UDP"),
		},
		// Allow inbound SSH for initial setup (retrieve WireGuard public key)
		{
			Direction: hcloud.FirewallRuleDirectionIn,
			Protocol:  hcloud.FirewallRuleProtocolTCP,
			Port:      hcloud.Ptr("22"),
			SourceIPs: []net.IPNet{
				{IP: net.ParseIP("0.0.0.0"), Mask: net.CIDRMask(0, 32)}, // IPv4 any
				{IP: net.ParseIP("::"), Mask: net.CIDRMask(0, 128)},     // IPv6 any
			},
			Description: hcloud.Ptr("Allow SSH for setup"),
		},
	}

	// Create the firewall
	result, _, err := p.client.Firewall.Create(ctx, hcloud.FirewallCreateOpts{
		Name:   fmt.Sprintf("my-own-vpn-%s", p.sessionID),
		Rules:  rules,
		Labels: p.getLabels(),
	})
	if err != nil {
		return fmt.Errorf("failed to create firewall: %w", err)
	}

	p.firewallID = result.Firewall.ID

	// Wait for actions to complete
	if len(result.Actions) > 0 {
		if err := p.client.Action.WaitForFunc(ctx, nil, result.Actions...); err != nil {
			return fmt.Errorf("failed waiting for firewall creation: %w", err)
		}
	}

	return nil
}

// deleteFirewall deletes the Hetzner Cloud firewall if it exists.
func (p *Provider) deleteFirewall(ctx context.Context) error {
	if p.firewallID == 0 {
		return nil
	}

	_, err := p.client.Firewall.Delete(ctx, &hcloud.Firewall{ID: p.firewallID})
	if err != nil {
		// Check if already deleted
		if hcloud.IsError(err, hcloud.ErrorCodeNotFound) {
			p.firewallID = 0
			return nil
		}
		return fmt.Errorf("failed to delete firewall: %w", err)
	}

	p.firewallID = 0
	return nil
}
