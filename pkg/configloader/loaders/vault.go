package loaders

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"go.uber.org/multierr"
)

type VaultLoader struct {
	name         string
	priority     int
	supportedTag string
	vaultConfig  VaultConfig
	ctx          context.Context
	client       *vault.Client
}

type VaultConfig struct {
	Address      string
	RoleId       string
	SecretId     string
	SecretPath   string
	AppRoleMount string // ?
	Timeout      int    // ?
	MaxRetries   int
	RetryDelay   time.Duration
}

func NewVaultLoader(vaultConfig VaultConfig) *VaultLoader {
	return NewVaultLoaderWithOptions("VaultLoader", 0, "vault", vaultConfig)
}

func NewVaultLoaderWithOptions(name string, priority int, supportedTag string, vaultConfig VaultConfig) *VaultLoader {
	ctx := context.Background()

	vaultLoader := &VaultLoader{
		name:         name,
		priority:     priority,
		supportedTag: supportedTag,
		vaultConfig:  vaultConfig,
		ctx:          ctx,
	}

	return vaultLoader
}

func (vl *VaultLoader) Name() string         { return vl.name }
func (vl *VaultLoader) Priority() int        { return vl.priority }
func (vl *VaultLoader) SupportedTag() string { return vl.supportedTag }

func (vl *VaultLoader) LoadValue(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
	if vl.client == nil {
		return false, fmt.Errorf("connection is not initialized")
	}

	var retryErrors error

	for attempt := 0; attempt < vl.vaultConfig.MaxRetries; attempt++ {

		serviceSecrets, err := vl.client.Read(vl.ctx, vl.vaultConfig.SecretPath)
		if err == nil {
			data, ok := serviceSecrets.Data["data"].(map[string]any)
			if !ok {
				return false, fmt.Errorf("invalid secret data format")
			}

			secret, ok := data[tagValue]
			if !ok {
				return false, nil
			}

			if err := setFieldValueFromInterface(field, secret); err != nil {
				return false, err
			}
			return true, nil
		}

		if shouldRetry(err) {
			return false, err
		}

		if refreshErr := vl.RefreshToken(); refreshErr != nil {
			return false, fmt.Errorf("refresh token failed: %w", refreshErr)
		}

		delay := vl.vaultConfig.RetryDelay * time.Duration(1<<attempt)
		select {
		case <-time.After(delay):
		case <-vl.ctx.Done():
			return false, vl.ctx.Err()
		}

		retryErrors = multierr.Append(retryErrors, err)
	}

	return false, fmt.Errorf("max retries exceeded: %w", retryErrors)
}

func (vl *VaultLoader) Connect() error {
	if vl.vaultConfig.Address == "" {
		return fmt.Errorf("vault address is required")
	}

	client, err := vault.New(
		vault.WithAddress(vl.vaultConfig.Address),
		vault.WithRequestTimeout(time.Duration(vl.vaultConfig.Timeout)*time.Second),
	)
	if err != nil {
		return fmt.Errorf("create vault client: %w", err)
	}

	vl.client = client
	vl.RefreshToken()

	return nil
}

func (vl *VaultLoader) RefreshToken() error {
	if vl.vaultConfig.RoleId == "" || vl.vaultConfig.SecretId == "" {
		return fmt.Errorf("vault role_id and secret_id are required")
	}

	if vl.vaultConfig.AppRoleMount == "" {
		return fmt.Errorf("auth mount path is required")
	}

	resp, err := vl.client.Auth.AppRoleLogin(
		vl.ctx,
		schema.AppRoleLoginRequest{
			RoleId:   vl.vaultConfig.RoleId,
			SecretId: vl.vaultConfig.SecretId,
		},
		// vault.WithMountPath(vl.vaultConfig.AppRoleMount),
	)
	if err != nil {
		return fmt.Errorf("approle login: %w", err)
	}

	if resp == nil || resp.Auth == nil {
		return fmt.Errorf("no auth response from Vault")
	}

	vl.client.SetToken(resp.Auth.ClientToken)

	return nil
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	if vault.IsErrorStatus(err, http.StatusForbidden) {
		return true
	}

	if vault.IsErrorStatus(err, http.StatusTooManyRequests) {
		return true
	}

	if vault.IsErrorStatus(err, http.StatusInternalServerError) {
		return true
	}

	return false
}
