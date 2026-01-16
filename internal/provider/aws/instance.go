package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Instance constants
const (
	// defaultInstanceType is the default EC2 instance type
	defaultInstanceType = "t3.micro"

	// instanceWaitTimeout is the maximum time to wait for instance operations
	instanceWaitTimeout = 10 * time.Minute

	// wireGuardReadyTimeout is the maximum time to wait for WireGuard to be ready
	wireGuardReadyTimeout = 5 * time.Minute

	// wireGuardPollInterval is how often to check if WireGuard is ready
	wireGuardPollInterval = 10 * time.Second
)

// generateUserData returns the cloud-init script to install and configure WireGuard.
// The script generates keys on the instance and outputs the public key to a log file.
func generateUserData() string {
	script := `#!/bin/bash
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

# Create WireGuard config
cat > /etc/wireguard/wg0.conf << 'WGEOF'
[Interface]
Address = 10.0.0.1/24
ListenPort = 51820
PrivateKey = PRIVATE_KEY_PLACEHOLDER
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o ens5 -j MASQUERADE; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o ens5 -j MASQUERADE; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
WGEOF

# Replace placeholder with actual private key
sed -i "s|PRIVATE_KEY_PLACEHOLDER|$SERVER_PRIVATE|g" /etc/wireguard/wg0.conf
chmod 600 /etc/wireguard/wg0.conf

# Enable IP forwarding
echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
sysctl -p

# Start WireGuard
systemctl enable wg-quick@wg0
systemctl start wg-quick@wg0

# Output public key to a file and as an instance tag
SERVER_PUBKEY=$(cat /etc/wireguard/server_public.key)
echo "WIREGUARD_PUBKEY=$SERVER_PUBKEY" >> /var/log/wireguard-setup.log
echo "WIREGUARD_READY=true" >> /var/log/wireguard-setup.log
echo "WireGuard setup complete!"
`
	return script
}

// getLatestUbuntuAMI finds the latest Ubuntu 22.04 LTS AMI for the current region.
func (p *Provider) getLatestUbuntuAMI(ctx context.Context) (string, error) {
	// Search for official Ubuntu 22.04 LTS AMIs from Canonical
	// Canonical's AWS account ID: 099720109477
	result, err := p.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: []string{"099720109477"}, // Canonical
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{"ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"},
			},
			{
				Name:   aws.String("architecture"),
				Values: []string{"x86_64"},
			},
			{
				Name:   aws.String("virtualization-type"),
				Values: []string{"hvm"},
			},
			{
				Name:   aws.String("state"),
				Values: []string{"available"},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe images: %w", err)
	}

	if len(result.Images) == 0 {
		return "", fmt.Errorf("no Ubuntu 22.04 AMI found in region %s", p.region)
	}

	// Find the most recent image by creation date
	var latestImage *types.Image
	for i := range result.Images {
		img := &result.Images[i]
		if latestImage == nil || (img.CreationDate != nil && latestImage.CreationDate != nil &&
			*img.CreationDate > *latestImage.CreationDate) {
			latestImage = img
		}
	}

	if latestImage == nil || latestImage.ImageId == nil {
		return "", fmt.Errorf("no valid Ubuntu AMI found")
	}

	return *latestImage.ImageId, nil
}

// createKeyPair creates an EC2 key pair for SSH access to the instance.
// Returns the key pair name. The private key is not stored as we use
// user-data for configuration instead of SSH.
func (p *Provider) createKeyPair(ctx context.Context) (string, error) {
	keyName := fmt.Sprintf("my-own-vpn-%d", time.Now().Unix())

	_, err := p.ec2Client.CreateKeyPair(ctx, &ec2.CreateKeyPairInput{
		KeyName:           aws.String(keyName),
		TagSpecifications: p.getTagSpecifications(types.ResourceTypeKeyPair),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create key pair: %w", err)
	}

	p.keyPairName = keyName
	return keyName, nil
}

// deleteKeyPair deletes the EC2 key pair if it exists.
func (p *Provider) deleteKeyPair(ctx context.Context) error {
	if p.keyPairName == "" {
		return nil
	}

	_, err := p.ec2Client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyName: aws.String(p.keyPairName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete key pair: %w", err)
	}

	p.keyPairName = ""
	return nil
}

// launchInstance launches an EC2 instance with WireGuard configured via user-data.
func (p *Provider) launchInstance(ctx context.Context, instanceType string) error {
	if p.subnetID == "" {
		return fmt.Errorf("subnet must be created before launching instance")
	}
	if p.securityGroupID == "" {
		return fmt.Errorf("security group must be created before launching instance")
	}

	// Get the latest Ubuntu AMI
	amiID, err := p.getLatestUbuntuAMI(ctx)
	if err != nil {
		return err
	}

	// Create key pair
	keyName, err := p.createKeyPair(ctx)
	if err != nil {
		return err
	}

	// Use default instance type if not specified
	if instanceType == "" {
		instanceType = defaultInstanceType
	}

	// Generate and encode user data
	userData := base64.StdEncoding.EncodeToString([]byte(generateUserData()))

	// Launch the instance
	result, err := p.ec2Client.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId:      aws.String(amiID),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		KeyName:      aws.String(keyName),
		NetworkInterfaces: []types.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int32(0),
				SubnetId:                 aws.String(p.subnetID),
				Groups:                   []string{p.securityGroupID},
				AssociatePublicIpAddress: aws.Bool(true),
			},
		},
		UserData:          aws.String(userData),
		TagSpecifications: p.getTagSpecifications(types.ResourceTypeInstance),
	})
	if err != nil {
		return fmt.Errorf("failed to launch instance: %w", err)
	}

	if len(result.Instances) == 0 {
		return fmt.Errorf("no instance was created")
	}

	p.instanceID = *result.Instances[0].InstanceId

	return nil
}

// waitForInstanceRunning waits for the instance to reach the running state.
func (p *Provider) waitForInstanceRunning(ctx context.Context) error {
	if p.instanceID == "" {
		return fmt.Errorf("no instance to wait for")
	}

	waiter := ec2.NewInstanceRunningWaiter(p.ec2Client)
	err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{p.instanceID},
	}, instanceWaitTimeout)
	if err != nil {
		return fmt.Errorf("failed waiting for instance to be running: %w", err)
	}

	return nil
}

// getInstancePublicIP retrieves the public IP address of the instance.
func (p *Provider) getInstancePublicIP(ctx context.Context) (string, error) {
	if p.instanceID == "" {
		return "", fmt.Errorf("no instance ID")
	}

	result, err := p.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{p.instanceID},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return "", fmt.Errorf("instance not found")
	}

	instance := result.Reservations[0].Instances[0]
	if instance.PublicIpAddress == nil {
		return "", fmt.Errorf("instance has no public IP address")
	}

	return *instance.PublicIpAddress, nil
}

// waitForWireGuardReady polls the instance console output to check if WireGuard is ready.
// This approach avoids the need for SSH access.
func (p *Provider) waitForWireGuardReady(ctx context.Context) (string, error) {
	if p.instanceID == "" {
		return "", fmt.Errorf("no instance ID")
	}

	deadline := time.Now().Add(wireGuardReadyTimeout)

	for time.Now().Before(deadline) {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Get console output
		output, err := p.ec2Client.GetConsoleOutput(ctx, &ec2.GetConsoleOutputInput{
			InstanceId: aws.String(p.instanceID),
		})
		if err != nil {
			// Console output may not be available immediately
			time.Sleep(wireGuardPollInterval)
			continue
		}

		if output.Output == nil {
			time.Sleep(wireGuardPollInterval)
			continue
		}

		// Decode the console output (it's base64 encoded)
		decoded, err := base64.StdEncoding.DecodeString(*output.Output)
		if err != nil {
			time.Sleep(wireGuardPollInterval)
			continue
		}

		consoleOutput := string(decoded)

		// Check if WireGuard is ready
		if strings.Contains(consoleOutput, "WIREGUARD_READY=true") {
			// Extract the public key
			pubKey := extractWireGuardPubKey(consoleOutput)
			if pubKey != "" {
				return pubKey, nil
			}
		}

		time.Sleep(wireGuardPollInterval)
	}

	return "", fmt.Errorf("timeout waiting for WireGuard to be ready")
}

// extractWireGuardPubKey extracts the WireGuard public key from console output.
func extractWireGuardPubKey(output string) string {
	// Look for the line containing the public key
	const prefix = "WIREGUARD_PUBKEY="
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, prefix) {
			// Extract the key value
			idx := strings.Index(line, prefix)
			if idx >= 0 {
				key := strings.TrimSpace(line[idx+len(prefix):])
				// WireGuard public keys are 44 characters (base64 encoded 32 bytes)
				if len(key) >= 44 {
					return key[:44]
				}
			}
		}
	}
	return ""
}

// terminateInstance terminates the EC2 instance if it exists.
func (p *Provider) terminateInstance(ctx context.Context) error {
	if p.instanceID == "" {
		return nil
	}

	_, err := p.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{p.instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	// Wait for the instance to be terminated
	waiter := ec2.NewInstanceTerminatedWaiter(p.ec2Client)
	err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{p.instanceID},
	}, instanceWaitTimeout)
	if err != nil {
		return fmt.Errorf("failed waiting for instance to terminate: %w", err)
	}

	p.instanceID = ""
	return nil
}

// getInstanceState retrieves the current state of the EC2 instance.
func (p *Provider) getInstanceState(ctx context.Context) (types.InstanceStateName, error) {
	if p.instanceID == "" {
		return "", ErrNotProvisioned
	}

	result, err := p.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{p.instanceID},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return "", fmt.Errorf("instance not found")
	}

	instance := result.Reservations[0].Instances[0]
	if instance.State == nil {
		return "", fmt.Errorf("instance state is nil")
	}

	return instance.State.Name, nil
}

// InstanceInfo contains information about the provisioned EC2 instance.
type InstanceInfo struct {
	InstanceID         string
	PublicIP           string
	State              types.InstanceStateName
	WireGuardPublicKey string
	KeyPairName        string
}

// GetInstanceInfo returns information about the provisioned EC2 instance.
func (p *Provider) GetInstanceInfo() InstanceInfo {
	return InstanceInfo{
		InstanceID:  p.instanceID,
		KeyPairName: p.keyPairName,
	}
}
