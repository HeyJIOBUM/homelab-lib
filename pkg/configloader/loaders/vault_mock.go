package loaders

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

type VaultClientInterface interface {
	AppRoleLogin(ctx context.Context, request schema.AppRoleLoginRequest, options ...vault.RequestOption) (*vault.Response[map[string]any], error)
	SetToken(token string) error
	Read(ctx context.Context, path string, options ...vault.RequestOption) (*vault.Response[map[string]any], error)
}

type VaultClientAdapter struct {
	client *vault.Client
}

func NewVaultClientAdapter(client *vault.Client) VaultClientInterface {
	return &VaultClientAdapter{client: client}
}

func (a *VaultClientAdapter) AppRoleLogin(ctx context.Context, request schema.AppRoleLoginRequest, options ...vault.RequestOption) (*vault.Response[map[string]any], error) {
	return a.client.Auth.AppRoleLogin(ctx, request, options...)
}

func (a *VaultClientAdapter) SetToken(token string) error {
	return a.client.SetToken(token)
}

func (a *VaultClientAdapter) Read(ctx context.Context, path string, options ...vault.RequestOption) (*vault.Response[map[string]any], error) {
	return a.client.Read(ctx, path, options...)
}

type MockVaultClient struct {
	secrets        map[string]map[string]any
	token          string
	authError      error
	readError      error
	setTokenErr    error
	appRoleCalled  bool
	readCalled     bool
	setTokenCalled bool
}

func NewMockVaultClient() *MockVaultClient {
	return &MockVaultClient{
		secrets: make(map[string]map[string]any),
	}
}

func (m *MockVaultClient) WithSecret(path string, data map[string]any) *MockVaultClient {
	m.secrets[path] = data
	return m
}

func (m *MockVaultClient) WithAuthError(err error) *MockVaultClient {
	m.authError = err
	return m
}

func (m *MockVaultClient) WithReadError(err error) *MockVaultClient {
	m.readError = err
	return m
}

func (m *MockVaultClient) WithSetTokenError(err error) *MockVaultClient {
	m.setTokenErr = err
	return m
}

func (m *MockVaultClient) AppRoleLogin(ctx context.Context, request schema.AppRoleLoginRequest, options ...vault.RequestOption) (*vault.Response[map[string]any], error) {
	m.appRoleCalled = true
	if m.authError != nil {
		return nil, m.authError
	}
	return &vault.Response[map[string]any]{
		Auth: &vault.ResponseAuth{
			ClientToken: "mock-token-123",
		},
	}, nil
}

func (m *MockVaultClient) SetToken(token string) error {
	m.setTokenCalled = true
	if m.setTokenErr != nil {
		return m.setTokenErr
	}
	m.token = token
	return nil
}

func (m *MockVaultClient) Read(ctx context.Context, path string, options ...vault.RequestOption) (*vault.Response[map[string]any], error) {
	m.readCalled = true
	if m.readError != nil {
		return nil, m.readError
	}
	if secret, ok := m.secrets[path]; ok {
		return &vault.Response[map[string]any]{
			Data: map[string]any{
				"data": secret,
			},
		}, nil
	}
	return nil, errors.New("secret not found")
}

func (m *MockVaultClient) AssertLoginCalled(t *testing.T) {
	t.Helper()
	if !m.appRoleCalled {
		t.Error("AppRoleLogin was not called")
	}
}

func (m *MockVaultClient) AssertReadCalled(t *testing.T) {
	t.Helper()
	if !m.readCalled {
		t.Error("Read was not called")
	}
}

func (m *MockVaultClient) AssertSetTokenCalled(t *testing.T) {
	t.Helper()
	if !m.setTokenCalled {
		t.Error("SetToken was not called")
	}
}

func (m *MockVaultClient) AssertSecretExists(t *testing.T, path string) {
	t.Helper()
	if _, ok := m.secrets[path]; !ok {
		t.Errorf("Secret at path %s was not found in mock", path)
	}
}
