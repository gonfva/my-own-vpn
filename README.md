# My Own VPN

A Go application that provides on-demand VPN infrastructure. The app runs in the system tray (Windows and macOS) and dynamically provisions cloud VMs with WireGuard when you need a VPN connection.

## Core Concept

When connecting:
1. The app provisions a VM (AWS EC2 or Hetzner) in your chosen region
2. Sets up necessary networking (VPC, subnet, IGW, Security Groups, keypair for AWS)
3. Installs and configures WireGuard on the VM
4. Establishes a VPN connection from your machine to the VM

When disconnecting:
1. Tears down the VPN connection
2. Decommissions all cloud infrastructure to avoid ongoing costs

## Features

- **System Tray Application** - Runs in notification area (Windows & macOS)
- **Multi-Provider Support** - AWS EC2 or Hetzner (one active configuration at a time)
- **Region Selection** - Choose deployment region based on provider's available regions
- **Secure Credential Storage** - Uses OS keychain where available, encrypted local storage as fallback
- **Cost Visibility** - Shows estimated running costs and usage metrics
- **Idle Timeout** - Optional auto-disconnect after period of inactivity

## Target Platforms

- Windows
- macOS

(Linux support may be added in future versions)

## Technology Stack

- **Language**: Go
- **System Tray**: github.com/getlantern/systray (or similar)
- **VPN Protocol**: WireGuard
- **Cloud Providers**: AWS, Hetzner
