package loaders

import (
	"context"
	"errors"
	"reflect"
	"testing"
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
			name:      "get string secret from service path",
			tagValue:  "test_string",
			fieldType: reflect.TypeFor[string](),
			expected:  "hello_world",
			wantOk:    true,
			wantErr:   false,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/dev/services/sync", map[string]any{
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
				m.WithSecret("secret/data/dev/services/sync", map[string]any{
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
				m.WithSecret("secret/data/dev/services/sync", map[string]any{
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
				m.WithSecret("secret/data/dev/services/sync", map[string]any{
					"test_float": 3.14,
				})
			},
		},
		{
			name:      "get secret from relative subpath",
			tagValue:  "s3:access_key",
			fieldType: reflect.TypeFor[string](),
			expected:  "GK123",
			wantOk:    true,
			wantErr:   false,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/dev/services/sync/s3", map[string]any{
					"access_key": "GK123",
				})
			},
		},
		{
			name:      "get secret from absolute path",
			tagValue:  "/shared/notifications:token",
			fieldType: reflect.TypeFor[string](),
			expected:  "notify-token",
			wantOk:    true,
			wantErr:   false,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/dev/shared/notifications", map[string]any{
					"token": "notify-token",
				})
			},
		},
		{
			name:      "empty path with colon uses service path",
			tagValue:  ":db_password",
			fieldType: reflect.TypeFor[string](),
			expected:  "db-secret",
			wantOk:    true,
			wantErr:   false,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/dev/services/sync", map[string]any{
					"db_password": "db-secret",
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
				m.WithSecret("secret/data/dev/services/sync", map[string]any{
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
				m.WithReadError(errors.New("vault connection error"))
			},
		},
		{
			name:      "secret not found at path",
			tagValue:  "test_string",
			fieldType: reflect.TypeFor[string](),
			expected:  nil,
			wantOk:    false,
			wantErr:   true,
			mockSetup: func(m *MockVaultClient) {
				m.WithSecret("secret/data/dev/services/sync", map[string]any{
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
					SecretMountPath:   "secret/data/dev",
					SecretServicePath: "services/sync",
					MaxRetries:        1,
					RoleID:            "test-role",
					SecretID:          "test-secret",
					AppRoleMount:      "approle",
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

func TestVaultLoader_resolvePath(t *testing.T) {
	tests := []struct {
		name     string
		tagValue string
		wantPath string
		wantKey  string
	}{
		{
			name:     "plain key",
			tagValue: "db_password",
			wantPath: "services/sync",
			wantKey:  "db_password",
		},
		{
			name:     "relative subpath",
			tagValue: "s3:access_key",
			wantPath: "services/sync/s3",
			wantKey:  "access_key",
		},
		{
			name:     "absolute path",
			tagValue: "/shared/notifications:token",
			wantPath: "shared/notifications",
			wantKey:  "token",
		},
		{
			name:     "empty path with colon",
			tagValue: ":token",
			wantPath: "services/sync",
			wantKey:  "token",
		},
		{
			name:     "nested relative path",
			tagValue: "a/b/c:key",
			wantPath: "services/sync/a/b/c",
			wantKey:  "key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := &VaultLoader{
				vaultConfig: VaultConfig{
					SecretServicePath: "services/sync",
				},
			}

			gotPath, gotKey := loader.resolvePath(tt.tagValue)

			if gotPath != tt.wantPath {
				t.Errorf("path: expected %q, got %q", tt.wantPath, gotPath)
			}
			if gotKey != tt.wantKey {
				t.Errorf("key: expected %q, got %q", tt.wantKey, gotKey)
			}
		})
	}
}

func TestVaultLoader_fullPath(t *testing.T) {
	tests := []struct {
		name      string
		mountPath string
		relative  string
		want      string
	}{
		{
			name:      "typical path",
			mountPath: "secret/data/dev",
			relative:  "services/sync",
			want:      "secret/data/dev/services/sync",
		},
		{
			name:      "mount path with trailing slash",
			mountPath: "secret/data/dev/",
			relative:  "services/sync",
			want:      "secret/data/dev/services/sync",
		},
		{
			name:      "relative with leading slash",
			mountPath: "secret/data/dev",
			relative:  "/services/sync",
			want:      "secret/data/dev/services/sync",
		},
		{
			name:      "empty relative",
			mountPath: "secret/data/dev",
			relative:  "",
			want:      "secret/data/dev",
		},
		{
			name:      "kv v1 style (no data)",
			mountPath: "secret/dev",
			relative:  "services/sync",
			want:      "secret/dev/services/sync",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := &VaultLoader{
				vaultConfig: VaultConfig{
					SecretMountPath: tt.mountPath,
				},
			}

			got := loader.fullPath(tt.relative)

			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
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
			name:      "successful refresh",
			mockSetup: func(m *MockVaultClient) {},
			wantErr:   false,
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
					Address:           "http://localhost:8200",
					RoleID:            "test-role",
					SecretID:          "test-secret",
					AppRoleMount:      "approle",
					SecretMountPath:   "secret/data/dev",
					SecretServicePath: "services/sync",
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
				Address:           "http://localhost:8200",
				RoleID:            "test-role",
				SecretID:          "test-secret",
				AppRoleMount:      "approle",
				SecretMountPath:   "secret/data/dev",
				SecretServicePath: "services/sync",
				Timeout:           30,
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
