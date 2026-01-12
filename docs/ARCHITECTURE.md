# My Own VPN - Architecture Document

## Overview

This document describes the architecture for the My Own VPN application - a Go-based system tray application that provides on-demand VPN infrastructure using WireGuard on AWS EC2 or Hetzner cloud providers.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        User's Machine                           │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    My Own VPN App                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │  │
│  │  │ System Tray │  │  Settings   │  │  Cost Tracker   │   │  │
│  │  │     UI      │  │   Window    │  │     Module      │   │  │
│  │  └──────┬──────┘  └──────┬──────┘  └────────┬────────┘   │  │
│  │         │                │                   │            │  │
│  │         └────────────────┼───────────────────┘            │  │
│  │                          ▼                                │  │
│  │              ┌───────────────────────┐                    │  │
│  │              │    Core Controller    │                    │  │
│  │              │  (State Management)   │                    │  │
│  │              └───────────┬───────────┘                    │  │
│  │                          │                                │  │
│  │         ┌────────────────┼────────────────┐               │  │
│  │         ▼                ▼                ▼               │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │  │
│  │  │ Credential  │  │  Provider   │  │  WireGuard  │       │  │
│  │  │  Manager    │  │  Interface  │  │   Client    │       │  │
│  │  └─────────────┘  └──────┬──────┘  └─────────────┘       │  │
│  │                          │                                │  │
│  │              ┌───────────┴───────────┐                    │  │
│  │              ▼                       ▼                    │  │
│  │       ┌─────────────┐         ┌─────────────┐            │  │
│  │       │ AWS Provider│         │   Hetzner   │            │  │
│  │       │             │         │  Provider   │            │  │
│  │       └─────────────┘         └─────────────┘            │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ WireGuard Tunnel
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Cloud Provider (AWS/Hetzner)                │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                          VPC/Network                      │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │                    Public Subnet                    │  │  │
│  │  │  ┌───────────────────────────────────────────────┐  │  │  │
│  │  │  │                 VM Instance                   │  │  │  │
│  │  │  │  ┌─────────────────────────────────────────┐  │  │  │  │
│  │  │  │  │            WireGuard Server             │  │  │  │  │
│  │  │  │  │  - Configured via cloud-init/SSH        │  │  │  │  │
│  │  │  │  │  - NAT for client traffic               │  │  │  │  │
│  │  │  │  └─────────────────────────────────────────┘  │  │  │  │
│  │  │  └───────────────────────────────────────────────┘  │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. System Tray UI

**Purpose**: Provide minimal, always-accessible interface for VPN control.

**Responsibilities**:
- Display connection status (disconnected, connecting, connected, disconnecting)
- Provide quick actions (connect, disconnect, settings)
- Show current region and provider
- Display cost information when connected
- Handle idle timeout notifications

**Library**: `github.com/getlantern/systray` or similar cross-platform library

**Menu Structure**:
```
[Icon] My Own VPN
├── Status: Disconnected
├── ─────────────
├── Connect
├── Settings...
├── ─────────────
├── Current Cost: $0.00
├── ─────────────
└── Quit
```

When connected:
```
[Icon] My Own VPN (Connected)
├── Status: Connected (us-east-1)
├── ─────────────
├── Disconnect
├── Settings...
├── ─────────────
├── Session Cost: $0.02
├── Session Time: 00:45:32
├── ─────────────
└── Quit
```

### 2. Settings Window

**Purpose**: Configure provider credentials, region selection, and preferences.

**Sections**:
1. **Provider Selection**: AWS or Hetzner radio buttons
2. **Credentials**: Provider-specific credential fields
3. **Region**: Dropdown populated based on selected provider
4. **Preferences**:
   - Idle timeout (enable/disable, duration)
   - Instance size selection
   - Auto-connect on startup (optional future feature)

**Library Options**:
- `fyne.io/fyne/v2` - Pure Go, cross-platform
- `github.com/webview/webview` - Lightweight HTML/CSS/JS UI

### 3. Core Controller

**Purpose**: Central state machine managing application lifecycle.

**States**:
```
DISCONNECTED ──Connect──▶ PROVISIONING ──Ready──▶ CONNECTING ──Success──▶ CONNECTED
     ▲                         │                       │                      │
     │                         │                       │                      │
     └─────────────────────────┴───────Fail/Cancel─────┴────Disconnect────────┘
                                                                    │
                                                                    ▼
                                                              DISCONNECTING
                                                                    │
                                                                    ▼
                                                         DEPROVISIONING ────▶ DISCONNECTED
```

**Responsibilities**:
- Manage state transitions
- Coordinate between providers and WireGuard client
- Handle errors and cleanup
- Track session metrics (time, estimated cost)
- Manage idle timeout

### 4. Credential Manager

**Purpose**: Securely store and retrieve cloud provider credentials.

**Storage Strategy**:
1. **Primary**: OS Keychain
   - macOS: Keychain Services (`github.com/keybase/go-keychain`)
   - Windows: Windows Credential Manager (`github.com/danieljoos/wincred`)
2. **Fallback**: Encrypted local file
   - Use `golang.org/x/crypto/nacl/secretbox`
   - Derive key from machine-specific identifier

**Stored Data**:
- AWS: Access Key ID, Secret Access Key
- Hetzner: API Token

**Security Considerations**:
- Never log credentials
- Clear credentials from memory after use
- Validate credentials before saving (test API call)

### 5. Provider Interface

**Purpose**: Abstract cloud provider operations behind common interface.

```go
type Provider interface {
    // ValidateCredentials tests if the stored credentials are valid
    ValidateCredentials(ctx context.Context) error

    // ListRegions returns available regions for this provider
    ListRegions(ctx context.Context) ([]Region, error)

    // Provision creates all infrastructure and returns server details
    Provision(ctx context.Context, config ProvisionConfig) (*ServerInfo, error)

    // Deprovision destroys all created infrastructure
    Deprovision(ctx context.Context) error

    // GetStatus returns current infrastructure status
    GetStatus(ctx context.Context) (*InfraStatus, error)

    // EstimateCost returns hourly cost estimate for the configuration
    EstimateCost(config ProvisionConfig) CostEstimate
}

type ServerInfo struct {
    PublicIP     string
    WireGuardPort int
    ServerPublicKey string
    // Other WireGuard config details
}
```

### 6. AWS Provider

**Purpose**: Implement provider interface for AWS EC2.

**Infrastructure Created**:
1. **VPC**: Dedicated VPC with CIDR (e.g., 10.0.0.0/16)
2. **Internet Gateway**: Attached to VPC
3. **Public Subnet**: Single subnet with auto-assign public IP
4. **Route Table**: Route 0.0.0.0/0 to IGW
5. **Security Group**:
   - Inbound: UDP 51820 (WireGuard) from anywhere
   - Outbound: All traffic
6. **Key Pair**: Generated for SSH access (for setup/debugging)
7. **EC2 Instance**:
   - Amazon Linux 2 or Ubuntu
   - t3.micro or t3.nano (configurable)
   - User data script to install/configure WireGuard

**Tagging Strategy**: All resources tagged with:
- `Application: my-own-vpn`
- `ManagedBy: my-own-vpn`
- `SessionID: <unique-id>` (for cleanup)

**Cleanup**: Use tags to find and delete all related resources.

**Library**: `github.com/aws/aws-sdk-go-v2`

### 7. Hetzner Provider

**Purpose**: Implement provider interface for Hetzner Cloud.

**Infrastructure Created**:
1. **SSH Key**: Upload public key for access
2. **Firewall**:
   - Inbound: UDP 51820 (WireGuard)
   - Outbound: All
3. **Server**:
   - CX11 or similar small instance
   - Ubuntu 22.04
   - Cloud-init to install WireGuard

**Library**: `github.com/hetznercloud/hcloud-go`

### 8. WireGuard Client

**Purpose**: Manage local WireGuard tunnel configuration.

**Responsibilities**:
- Generate client keypair
- Create WireGuard configuration
- Apply configuration (platform-specific)
- Monitor connection status
- Tear down tunnel

**Platform Considerations**:
- **Windows**: Use `wireguard.exe` CLI or WireGuard Windows service
- **macOS**: Use `wg-quick` or WireGuard app integration

**Configuration Template**:
```ini
[Interface]
PrivateKey = <client-private-key>
Address = 10.0.0.2/32
DNS = 1.1.1.1

[Peer]
PublicKey = <server-public-key>
AllowedIPs = 0.0.0.0/0
Endpoint = <server-ip>:51820
PersistentKeepalive = 25
```

### 9. Cost Tracker Module

**Purpose**: Track and display estimated costs.

**Features**:
- Calculate hourly rate based on instance type and region
- Track session duration
- Display running cost estimate
- (Future) Historical cost tracking

**Data Sources**:
- AWS: Embedded pricing data or AWS Price List API
- Hetzner: Embedded pricing data (simpler pricing model)

## Data Flow: Connect Sequence

```
1. User clicks "Connect"
2. Core Controller transitions to PROVISIONING
3. Provider.Provision() called:
   a. Create networking (VPC, subnet, etc.)
   b. Create security groups
   c. Launch instance with WireGuard setup script
   d. Wait for instance to be ready
   e. Retrieve server public key (via cloud-init output or SSH)
   f. Return ServerInfo
4. Core Controller transitions to CONNECTING
5. WireGuard Client:
   a. Generate client keypair (if not exists)
   b. Configure server with client public key (via SSH or API)
   c. Create local WireGuard config
   d. Activate tunnel
   e. Verify connectivity
6. Core Controller transitions to CONNECTED
7. Cost Tracker starts session timer
8. UI updates to show connected status
```

## Data Flow: Disconnect Sequence

```
1. User clicks "Disconnect" (or idle timeout triggered)
2. Core Controller transitions to DISCONNECTING
3. WireGuard Client:
   a. Deactivate tunnel
   b. Remove local configuration
4. Core Controller transitions to DEPROVISIONING
5. Provider.Deprovision() called:
   a. Terminate instance
   b. Delete security groups
   c. Delete networking resources
   d. Delete key pairs
6. Core Controller transitions to DISCONNECTED
7. Cost Tracker finalizes session cost
8. UI updates to show disconnected status
```

## Security Considerations

### Credential Storage
- Use OS keychain as primary storage
- Never store credentials in plain text
- Never log credentials
- Validate credentials on save

### WireGuard Keys
- Generate new client keypair for each session (or persist securely)
- Server keypair generated fresh each provision
- Keys never transmitted in plain text (generated on server)

### Cloud Infrastructure
- Minimal security group rules (only WireGuard port)
- No SSH access by default (use cloud-init for setup)
- All resources tagged for easy identification and cleanup
- Session IDs for tracking provisioned resources

### Network Security
- All traffic routed through VPN when connected
- DNS queries routed through VPN
- Kill switch consideration (future enhancement)

### Code Security (Open Source)
- No hardcoded secrets
- Dependency scanning in CI
- Regular security updates
- Clear documentation of security model

## Testing Strategy

### Unit Tests
- Provider logic (mocked API calls)
- State machine transitions
- Configuration parsing
- Cost calculations

### Integration Tests
- Actual API calls to cloud providers (with test credentials)
- WireGuard configuration generation
- Credential storage/retrieval

### End-to-End Tests
- Full connect/disconnect cycle
- Cleanup verification
- Cross-platform behavior

### Security Tests
- Credential storage verification
- No credential leakage in logs
- Proper cleanup of sensitive data

## CI/CD Pipeline

### Build
- Multi-platform builds (Windows, macOS)
- Cross-compilation from Linux CI
- Code signing (for distribution)

### Test
- Unit tests on every PR
- Integration tests on main branch
- Security scanning (gosec, dependency check)

### Release
- Semantic versioning
- GitHub Releases with binaries
- Checksums for verification
- (Future) Auto-update mechanism

## Directory Structure

```
my-own-vpn/
├── cmd/
│   └── my-own-vpn/
│       └── main.go           # Application entry point
├── internal/
│   ├── app/
│   │   └── app.go            # Core controller & state machine
│   ├── ui/
│   │   ├── systray.go        # System tray implementation
│   │   └── settings.go       # Settings window
│   ├── credentials/
│   │   ├── manager.go        # Credential manager interface
│   │   ├── keychain_darwin.go
│   │   ├── keychain_windows.go
│   │   └── fallback.go       # Encrypted file fallback
│   ├── provider/
│   │   ├── provider.go       # Provider interface
│   │   ├── aws/
│   │   │   └── aws.go        # AWS implementation
│   │   └── hetzner/
│   │       └── hetzner.go    # Hetzner implementation
│   ├── wireguard/
│   │   ├── client.go         # WireGuard client interface
│   │   ├── client_darwin.go
│   │   └── client_windows.go
│   └── cost/
│       └── tracker.go        # Cost tracking
├── scripts/
│   └── wireguard-setup.sh    # Server setup script (embedded)
├── docs/
│   └── ARCHITECTURE.md       # This document
├── .github/
│   └── workflows/
│       ├── ci.yml            # CI pipeline
│       └── release.yml       # Release pipeline
├── go.mod
├── go.sum
├── README.md
├── CLAUDE.md
└── LICENSE
```

## Development Phases

### Phase 1: Foundation
- Project setup with CI/CD
- Core application framework (system tray, basic UI)
- Credential management

### Phase 2: AWS Provider
- Full AWS infrastructure provisioning
- WireGuard server setup on AWS
- WireGuard client integration

### Phase 3: Core VPN Functionality
- Complete connect/disconnect flow
- Error handling and recovery
- Basic cost tracking

### Phase 4: Hetzner Provider
- Hetzner infrastructure provisioning
- Provider switching in UI

### Phase 5: Polish
- Idle timeout
- Enhanced cost tracking
- UI improvements
- Documentation

### Phase 6: Release
- Code signing
- Release automation
- Distribution
