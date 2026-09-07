package configloader

import (
	"os"
	"testing"
)

func TestLoadHomelabAppConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		wantCfg HomelabAppConfig
	}{
		{
			name: "all required env vars set",
			env: map[string]string{
				"APP_NAME":              "myservice",
				"APP_ENV":               "prod",
				"APP_PORT":              "8080",
				"VAULT_ENV_CONFIG_PATH": "/etc/vault/creds.env",
				"CONFIG_FILE_PATH":      "config.yaml",
			},
			wantErr: false,
			wantCfg: HomelabAppConfig{
				AppName:            "myservice",
				AppEnv:             "prod",
				AppPort:            "8080",
				VaultEnvConfigPath: "/etc/vault/creds.env",
				ConfigFilePath:     "config.yaml",
			},
		},
		{
			name: "missing APP_NAME",
			env: map[string]string{
				"APP_ENV":               "prod",
				"APP_PORT":              "8080",
				"VAULT_ENV_CONFIG_PATH": "/etc/vault/creds.env",
			},
			wantErr: true,
		},
		{
			name: "missing APP_ENV",
			env: map[string]string{
				"APP_NAME":              "myservice",
				"APP_PORT":              "8080",
				"VAULT_ENV_CONFIG_PATH": "/etc/vault/creds.env",
			},
			wantErr: true,
		},
		{
			name: "missing APP_PORT",
			env: map[string]string{
				"APP_NAME":              "myservice",
				"APP_ENV":               "prod",
				"VAULT_ENV_CONFIG_PATH": "/etc/vault/creds.env",
			},
			wantErr: true,
		},
		{
			name: "missing VAULT_ENV_CONFIG_PATH",
			env: map[string]string{
				"APP_NAME": "myservice",
				"APP_ENV":  "prod",
				"APP_PORT": "8080",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := LoadHomelabAppConfig()
			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if !tt.wantErr && cfg != tt.wantCfg {
				t.Errorf("Expected %+v, got %+v", tt.wantCfg, cfg)
			}
		})
	}
}

func TestLoadVaultConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		envFile string
		wantErr bool
	}{
		{
			name: "all vault env vars set via env file",
			envFile: `
VAULT_ADDR=http://vault:8200
VAULT_ROLE_ID=role-123
VAULT_SECRET_ID=secret-456
VAULT_SECRET_PATH=secret/data/dev/myapp
`,
			wantErr: false,
		},
		{
			name: "missing VAULT_ADDR",
			envFile: `
VAULT_ROLE_ID=role-123
VAULT_SECRET_ID=secret-456
VAULT_SECRET_PATH=secret/data/dev/myapp
`,
			wantErr: true,
		},
		{
			name: "missing VAULT_ROLE_ID",
			envFile: `
VAULT_ADDR=http://vault:8200
VAULT_SECRET_ID=secret-456
VAULT_SECRET_PATH=secret/data/dev/myapp
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			var paths []string

			if tt.envFile != "" {
				tmpFile, err := os.CreateTemp("", "test_*.env")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(tmpFile.Name())

				if _, err := tmpFile.Write([]byte(tt.envFile)); err != nil {
					t.Fatal(err)
				}
				tmpFile.Close()
				paths = []string{tmpFile.Name()}
			}

			cfg, err := LoadVaultConfig(paths...)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if !tt.wantErr {
				if cfg.AppRoleMount == "" {
					t.Error("AppRoleMount should have default value")
				}
				if cfg.Timeout <= 0 {
					t.Error("Timeout should have default value")
				}
				if cfg.MaxRetries <= 0 {
					t.Error("MaxRetries should have default value")
				}
				if cfg.RetryDelay <= 0 {
					t.Error("RetryDelay should have default value")
				}
			}
		})
	}
}

func TestLoadEnvFilesWithOverride(t *testing.T) {
	tests := []struct {
		name    string
		envFile string
		system  map[string]string
		want    map[string]string
	}{
		{
			name: "system overrides env file",
			envFile: `
FOO=bar
BAZ=qux
`,
			system: map[string]string{
				"FOO": "override",
			},
			want: map[string]string{
				"FOO": "override",
				"BAZ": "qux",
			},
		},
		{
			name: "env file only",
			envFile: `
FOO=bar
BAZ=qux
`,
			system: map[string]string{},
			want: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "test_*.env")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())

			if tt.envFile != "" {
				if _, err := tmpFile.Write([]byte(tt.envFile)); err != nil {
					t.Fatal(err)
				}
			}
			tmpFile.Close()

			for k, v := range tt.system {
				t.Setenv(k, v)
			}

			result, err := LoadEnvFilesWithOverride(tmpFile.Name())
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			for k, v := range tt.want {
				if result[k] != v {
					t.Errorf("Expected %s=%s, got %s=%s", k, v, k, result[k])
				}
			}
		})
	}
}

func TestLoadEnvFilesWithOverride_MultipleFiles(t *testing.T) {
	file1, err := os.CreateTemp("", "test1_*.env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file1.Name())
	if _, err := file1.Write([]byte("FOO=from_file1\nBAR=bar")); err != nil {
		t.Fatal(err)
	}
	file1.Close()

	file2, err := os.CreateTemp("", "test2_*.env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file2.Name())
	if _, err := file2.Write([]byte("FOO=from_file2\nBAZ=baz")); err != nil {
		t.Fatal(err)
	}
	file2.Close()

	t.Setenv("FOO", "from_system")

	result, err := LoadEnvFilesWithOverride(file1.Name(), file2.Name())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := map[string]string{
		"FOO": "from_system",
		"BAR": "bar",
		"BAZ": "baz",
	}

	for k, v := range expected {
		if result[k] != v {
			t.Errorf("Expected %s=%s, got %s=%s", k, v, k, result[k])
		}
	}
}

func TestNewHomelabConfigLoader(t *testing.T) {
	envContent := `
VAULT_ADDR=http://vault:8200
VAULT_ROLE_ID=role-123
VAULT_SECRET_ID=secret-456
VAULT_SECRET_PATH=secret/data/dev/myapp
`
	tmpFile, err := os.CreateTemp("", "test_*.env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write([]byte(envContent)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	appCfg := HomelabAppConfig{
		AppName:            "test-service",
		AppEnv:             "dev",
		AppPort:            "8080",
		VaultEnvConfigPath: tmpFile.Name(),
		ConfigFilePath:     "",
	}

	loader, err := NewHomelabConfigLoader(appCfg)
	if err != nil {
		t.Fatalf("NewHomelabConfigLoader failed: %v", err)
	}

	if loader == nil {
		t.Error("Expected non-nil loader")
	}
}
