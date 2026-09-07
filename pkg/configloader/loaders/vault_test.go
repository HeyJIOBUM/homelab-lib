package loaders

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/hashicorp/vault-client-go/schema"
)

func TestVaultLoader_LoadValue_Unit(t *testing.T) {
	tests := []struct {
		name      string
		tagValue  string
		fieldType reflect.Type
		expected  any
		wantOk    bool
		wantErr   bool
		mockSetup func(*MockVaultClient)
	}{
		{
			name:      "get string secret",
			tagValue:  "test_string",
			fieldType: reflect.TypeFor[string](),
			expected:  "hello_world",
			wantOk:    true,
			wantErr:   false,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/test", map[string]any{
					"test_string": "hello_world",
				})
			},
		},
		{
			name:      "get int secret",
			tagValue:  "test_int",
			fieldType: reflect.TypeFor[int](),
			expected:  12345,
			wantOk:    true,
			wantErr:   false,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/test", map[string]any{
					"test_int": 12345,
				})
			},
		},
		{
			name:      "get bool secret",
			tagValue:  "test_bool",
			fieldType: reflect.TypeFor[bool](),
			expected:  true,
			wantOk:    true,
			wantErr:   false,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/test", map[string]any{
					"test_bool": true,
				})
			},
		},
		{
			name:      "get float secret",
			tagValue:  "test_float",
			fieldType: reflect.TypeFor[float64](),
			expected:  3.14,
			wantOk:    true,
			wantErr:   false,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/test", map[string]any{
					"test_float": 3.14,
				})
			},
		},
		{
			name:      "non-existent key",
			tagValue:  "non_existent",
			fieldType: reflect.TypeFor[string](),
			expected:  nil,
			wantOk:    false,
			wantErr:   false,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/test", map[string]any{
					"test_string": "hello_world",
				})
			},
		},
		{
			name:      "empty tag value",
			tagValue:  "",
			fieldType: reflect.TypeFor[string](),
			expected:  nil,
			wantOk:    false,
			wantErr:   false,
			mockSetup: nil,
		},
		{
			name:      "vault returns error",
			tagValue:  "test_string",
			fieldType: reflect.TypeFor[string](),
			expected:  nil,
			wantOk:    false,
			wantErr:   true,
			mockSetup: func(m *MockVaultClient) {
				m.WithAuthError(errors.New("vault connection error"))
			},
		},
		{
			name:      "secret not found",
			tagValue:  "test_string",
			fieldType: reflect.TypeFor[string](),
			expected:  nil,
			wantOk:    false,
			wantErr:   false,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/test", map[string]any{
					"different_secret": "hello_world",
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockVaultClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mockClient)
			}

			loader := &VaultLoader{
				name:         "VaultLoader",
				priority:     0,
				supportedTag: "vault",
				vaultConfig: VaultConfig{
					SecretPath:   "secret/data/test",
					MaxRetries:   1,
					RoleID:       "test-role",
					SecretID:     "test-secret",
					AppRoleMount: "approle",
				},
				client: mockClient,
				ctx:    context.Background(),
			}

			field := reflect.New(tt.fieldType).Elem()
			structField := reflect.StructField{
				Name: "TestField",
				Type: tt.fieldType,
			}

			ok, err := loader.LoadValue(tt.tagValue, field, structField)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if ok != tt.wantOk {
				t.Errorf("Expected ok = %v, got %v", tt.wantOk, ok)
				return
			}

			if tt.wantOk {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Expected %v, got %v", tt.expected, got)
				}
			}
		})
	}
}

func TestVaultLoader_RefreshToken_Unit(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(*MockVaultClient)
		wantErr   bool
	}{
		{
			name: "successful refresh",
			mockSetup: func(m *MockVaultClient) {
			},
			wantErr: false,
		},
		{
			name: "auth error",
			mockSetup: func(m *MockVaultClient) {
				m.WithAuthError(errors.New("auth failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockVaultClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mockClient)
			}

			loader := &VaultLoader{
				name:         "VaultLoader",
				priority:     0,
				supportedTag: "vault",
				vaultConfig: VaultConfig{
					Address:      "http://localhost:8200",
					RoleID:       "test-role",
					SecretID:     "test-secret",
					AppRoleMount: "approle",
					SecretPath:   "secret/data/test",
				},
				client: mockClient,
				ctx:    context.Background(),
			}

			err := loader.RefreshToken()

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestVaultLoader_Connect_Unit(t *testing.T) {
	tests := []struct {
		name      string
		config    VaultConfig
		mockSetup func(*MockVaultClient)
		wantErr   bool
	}{
		{
			name: "successful connect",
			config: VaultConfig{
				Address:      "http://localhost:8200",
				RoleID:       "test-role",
				SecretID:     "test-secret",
				AppRoleMount: "approle",
				SecretPath:   "secret/data/test",
				Timeout:      30,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewMockVaultClient()

			loader := &VaultLoader{
				name:         "VaultLoader",
				priority:     0,
				supportedTag: "vault",
				vaultConfig:  tt.config,
				client:       mockClient,
				ctx:          context.Background(),
			}

			err := loader.Connect()

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestMockVaultClient_Chaining(t *testing.T) {
	mock := NewMockVaultClient().
		WithSecret("secret/data/test", map[string]any{
			"key1": "value1",
			"key2": "value2",
		})

	data, err := mock.Read(context.Background(), "secret/data/test")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if data == nil {
		t.Error("Expected data, got nil")
	}

	if secret, ok := data.Data["data"].(map[string]any); ok {
		if secret["key1"] != "value1" {
			t.Errorf("Expected key1 = value1, got %v", secret["key1"])
		}
		if secret["key2"] != "value2" {
			t.Errorf("Expected key2 = value2, got %v", secret["key2"])
		}
	} else {
		t.Error("Expected data.data to be map[string]any")
	}
}

func TestMockVaultClient_Assertions(t *testing.T) {
	mock := NewMockVaultClient().
		WithSecret("secret/data/test", map[string]any{
			"test_string": "hello_world",
		})

	ctx := context.Background()

	mock.AppRoleLogin(ctx, schema.AppRoleLoginRequest{})
	mock.Read(ctx, "secret/data/test")
	mock.SetToken("test-token")

	mock.AssertLoginCalled(t)
	mock.AssertReadCalled(t)
	mock.AssertSetTokenCalled(t)
	mock.AssertSecretExists(t, "secret/data/test")
}
