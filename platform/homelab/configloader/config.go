package configloader

import (
	"fmt"
	"os"

	"github.com/HeyJIOBUM/homelab-lib/pkg/configloader"
	"github.com/HeyJIOBUM/homelab-lib/pkg/configloader/loaders"
	"github.com/joho/godotenv"
	"go.uber.org/multierr"
)

type HomelabAppConfig struct {
	AppName            string
	AppEnv             string
	AppPort            string
	VaultEnvConfigPath string
	ConfigFilePath     string
}

func LoadHomelabAppConfig() (HomelabAppConfig, error) {
	cfg := HomelabAppConfig{
		AppName:            os.Getenv("APP_NAME"),
		AppEnv:             os.Getenv("APP_ENV"),
		AppPort:            os.Getenv("APP_PORT"),
		VaultEnvConfigPath: os.Getenv("VAULT_ENV_CONFIG_PATH"),
		ConfigFilePath:     os.Getenv("CONFIG_FILE_PATH"),
	}

	var errors error
	if cfg.AppName == "" {
		errors = multierr.Append(errors, fmt.Errorf("APP_NAME is required"))
	}
	if cfg.AppEnv == "" {
		errors = multierr.Append(errors, fmt.Errorf("APP_ENV is required"))
	}
	if cfg.AppPort == "" {
		errors = multierr.Append(errors, fmt.Errorf("APP_PORT is required"))
	}
	if cfg.VaultEnvConfigPath == "" {
		errors = multierr.Append(errors, fmt.Errorf("VAULT_ENV_CONFIG_PATH is required"))
	}

	if errors != nil {
		return HomelabAppConfig{}, fmt.Errorf("missing required environment variables: %w", errors)
	}

	return cfg, nil
}

func NewHomelabConfigLoader(appCfg HomelabAppConfig) (*configloader.Loader, error) {
	envPaths := []string{appCfg.VaultEnvConfigPath}

	vaultCfg, err := LoadVaultConfig(envPaths...)
	if err != nil {
		return nil, fmt.Errorf("load vault config: %w", err)
	}

	loadersList := []configloader.LoaderInterface{
		loaders.NewDefaultLoader(),
		loaders.NewEnvLoader(),
	}

	if appCfg.ConfigFilePath != "" {
		yamlLoader, err := loaders.NewYamlLoader(appCfg.ConfigFilePath)
		if err != nil {
			return nil, fmt.Errorf("create yaml loader: %w", err)
		}
		loadersList = append(loadersList, yamlLoader)
	}

	vaultLoader, err := loaders.NewVaultLoader(vaultCfg)
	if err != nil {
		return nil, fmt.Errorf("create vault loader: %w", err)
	}
	loadersList = append(loadersList, vaultLoader)

	return configloader.NewLoader(loadersList, true), nil
}

func LoadVaultConfig(envPaths ...string) (loaders.VaultConfig, error) {
	envMap, err := LoadEnvFilesWithOverride(envPaths...)
	if err != nil {
		return loaders.VaultConfig{}, fmt.Errorf("load env file: %w", err)
	}

	cfg := loaders.DefaultVaultConfig()

	cfg.Address = envMap["VAULT_ADDR"]
	cfg.RoleID = envMap["VAULT_ROLE_ID"]
	cfg.SecretID = envMap["VAULT_SECRET_ID"]
	cfg.SecretPath = envMap["VAULT_SECRET_PATH"]

	var errors error
	if cfg.Address == "" {
		errors = multierr.Append(errors, fmt.Errorf("VAULT_ADDR is required"))
	}
	if cfg.RoleID == "" {
		errors = multierr.Append(errors, fmt.Errorf("VAULT_ROLE_ID is required"))
	}
	if cfg.SecretID == "" {
		errors = multierr.Append(errors, fmt.Errorf("VAULT_SECRET_ID is required"))
	}
	if cfg.SecretPath == "" {
		errors = multierr.Append(errors, fmt.Errorf("VAULT_SECRET_PATH is required"))
	}

	if errors != nil {
		return loaders.VaultConfig{}, fmt.Errorf("missing required Vault environment variables: %w", errors)
	}

	return cfg, nil
}

func LoadEnvFilesWithOverride(paths ...string) (map[string]string, error) {
	if len(paths) == 0 {
		return make(map[string]string), nil
	}

	envMap, err := godotenv.Read(paths...)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("read env file: %w", err)
	}

	for key := range envMap {
		if val := os.Getenv(key); val != "" {
			envMap[key] = val
		}
	}

	return envMap, nil
}
