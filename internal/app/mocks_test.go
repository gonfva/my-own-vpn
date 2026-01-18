package app

import (
	"context"
	"sync"

	"github.com/gonfva/my-own-vpn/internal/credentials"
	"github.com/gonfva/my-own-vpn/internal/provider"
	"github.com/gonfva/my-own-vpn/internal/wireguard"
)

// MockCredentialsManager is a mock implementation of credentials.Manager for testing
type MockCredentialsManager struct {
	mu sync.Mutex

	// AWS credentials
	awsCreds    *credentials.AWSCredentials
	awsLoadErr  error
	awsSaveErr  error
	hasAWSCreds bool

	// Hetzner credentials
	hetznerCreds    *credentials.HetznerCredentials
	hetznerLoadErr  error
	hetznerSaveErr  error
	hasHetznerCreds bool
}

func (m *MockCredentialsManager) SaveAWS(_ context.Context, creds credentials.AWSCredentials) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.awsSaveErr != nil {
		return m.awsSaveErr
	}
	m.awsCreds = &creds
	m.hasAWSCreds = true
	return nil
}

func (m *MockCredentialsManager) LoadAWS(_ context.Context) (*credentials.AWSCredentials, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.awsLoadErr != nil {
		return nil, m.awsLoadErr
	}
	return m.awsCreds, nil
}

func (m *MockCredentialsManager) DeleteAWS(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.awsCreds = nil
	m.hasAWSCreds = false
	return nil
}

func (m *MockCredentialsManager) SaveHetzner(_ context.Context, creds credentials.HetznerCredentials) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hetznerSaveErr != nil {
		return m.hetznerSaveErr
	}
	m.hetznerCreds = &creds
	m.hasHetznerCreds = true
	return nil
}

func (m *MockCredentialsManager) LoadHetzner(_ context.Context) (*credentials.HetznerCredentials, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hetznerLoadErr != nil {
		return nil, m.hetznerLoadErr
	}
	return m.hetznerCreds, nil
}

func (m *MockCredentialsManager) DeleteHetzner(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hetznerCreds = nil
	m.hasHetznerCreds = false
	return nil
}

func (m *MockCredentialsManager) HasCredentials(_ context.Context, providerType string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch providerType {
	case "aws":
		return m.hasAWSCreds
	case "hetzner":
		return m.hasHetznerCreds
	default:
		return false
	}
}

// Ensure MockCredentialsManager implements credentials.Manager
var _ credentials.Manager = (*MockCredentialsManager)(nil)

// MockProvider is a mock implementation of provider.Provider for testing
type MockProvider struct {
	mu sync.Mutex

	// Control behavior
	validateErr    error
	provisionErr   error
	deprovisionErr error
	serverInfo     *provider.ServerInfo

	// Track calls
	ValidateCalled    bool
	ProvisionCalled   bool
	DeprovisionCalled bool
	ProvisionConfig   provider.ProvisionConfig
}

func (m *MockProvider) ValidateCredentials(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ValidateCalled = true
	return m.validateErr
}

func (m *MockProvider) ListRegions(_ context.Context) ([]provider.Region, error) {
	return []provider.Region{{ID: "test-region", Name: "Test Region"}}, nil
}

func (m *MockProvider) Provision(_ context.Context, cfg provider.ProvisionConfig) (*provider.ServerInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ProvisionCalled = true
	m.ProvisionConfig = cfg
	if m.provisionErr != nil {
		return nil, m.provisionErr
	}
	if m.serverInfo != nil {
		return m.serverInfo, nil
	}
	return &provider.ServerInfo{
		PublicIP:        "1.2.3.4",
		WireGuardPort:   51820,
		ServerPublicKey: "test-server-public-key",
	}, nil
}

func (m *MockProvider) Deprovision(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DeprovisionCalled = true
	return m.deprovisionErr
}

func (m *MockProvider) GetStatus(_ context.Context) (*provider.InfraStatus, error) {
	return &provider.InfraStatus{State: provider.StateRunning, Message: "Running"}, nil
}

func (m *MockProvider) EstimateCost(_ provider.ProvisionConfig) provider.CostEstimate {
	return provider.CostEstimate{HourlyRate: 0.01, Currency: "USD"}
}

// SetProvisionError sets the error to return from Provision
func (m *MockProvider) SetProvisionError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.provisionErr = err
}

// SetValidateError sets the error to return from ValidateCredentials
func (m *MockProvider) SetValidateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validateErr = err
}

// SetDeprovisionError sets the error to return from Deprovision
func (m *MockProvider) SetDeprovisionError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deprovisionErr = err
}

// Ensure MockProvider implements provider.Provider
var _ provider.Provider = (*MockProvider)(nil)

// MockWireGuardClient is a mock implementation of wireguard.Client for testing
type MockWireGuardClient struct {
	mu sync.Mutex

	// Control behavior
	connectErr    error
	disconnectErr error
	connected     bool

	// Track calls
	ConnectCalled    bool
	DisconnectCalled bool
	ServerInfo       *provider.ServerInfo
	ClientKeyPair    *wireguard.KeyPair
}

func (m *MockWireGuardClient) Connect(_ context.Context, serverInfo *provider.ServerInfo, clientKeyPair *wireguard.KeyPair) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConnectCalled = true
	m.ServerInfo = serverInfo
	m.ClientKeyPair = clientKeyPair
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *MockWireGuardClient) Disconnect(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DisconnectCalled = true
	if m.disconnectErr != nil {
		return m.disconnectErr
	}
	m.connected = false
	return nil
}

func (m *MockWireGuardClient) Status() wireguard.ConnectionStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return wireguard.ConnectionStatus{Connected: m.connected}
}

func (m *MockWireGuardClient) GetPublicKey() string {
	return "mock-client-public-key"
}

// SetConnectError sets the error to return from Connect
func (m *MockWireGuardClient) SetConnectError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectErr = err
}

// SetDisconnectError sets the error to return from Disconnect
func (m *MockWireGuardClient) SetDisconnectError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disconnectErr = err
}

// SetConnected sets the connected state
func (m *MockWireGuardClient) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = connected
}

// Ensure MockWireGuardClient implements wireguard.Client
var _ wireguard.Client = (*MockWireGuardClient)(nil)
