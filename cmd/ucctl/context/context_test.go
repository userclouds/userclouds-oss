package context

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"userclouds.com/cmd/ucctl/common"
)

func TestSetCommand_ValidateSecrets(t *testing.T) {
	tests := []struct {
		name         string
		clientSecret string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "plain secret is wrapped as dev-literal",
			clientSecret: "mysupersecret123",
			expectError:  false,
		},
		{
			name:         "dev-literal secret is valid",
			clientSecret: "dev-literal://mysecret",
			expectError:  false,
		},
		{
			name:         "aws secret without params is valid",
			clientSecret: "aws://secrets/my-secret",
			expectError:  false,
		},
		{
			name:         "aws secret with profile param is valid",
			clientSecret: "aws://secrets/my-secret?profile=production",
			expectError:  false,
		},
		{
			name:         "aws secret with region param is valid",
			clientSecret: "aws://secrets/my-secret?region=us-east-1",
			expectError:  false,
		},
		{
			name:         "aws secret with both profile and region is valid",
			clientSecret: "aws://secrets/my-secret?profile=prod&region=us-west-2",
			expectError:  false,
		},
		{
			name:         "aws secret with invalid param fails",
			clientSecret: "aws://secrets/my-secret?invalid=param",
			expectError:  true,
			errorMsg:     "unsupported query parameter \"invalid\" for AWS provider",
		},
		{
			name:         "aws secret with mixed valid and invalid params fails",
			clientSecret: "aws://secrets/my-secret?profile=prod&invalid=param",
			expectError:  true,
			errorMsg:     "unsupported query parameter \"invalid\" for AWS provider",
		},
		{
			name:         "env secret is valid",
			clientSecret: "env://MY_SECRET",
			expectError:  false,
		},
		{
			name:         "env secret with query params fails",
			clientSecret: "env://MY_SECRET?profile=prod",
			expectError:  true,
			errorMsg:     "env provider does not support query parameters",
		},
		{
			name:         "kube secret is valid",
			clientSecret: "kube://secrets/my-secret",
			expectError:  false,
		},
		{
			name:         "kube secret with query params fails",
			clientSecret: "kube://secrets/my-secret?namespace=default",
			expectError:  true,
			errorMsg:     "kubernetes provider does not support query parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file for this test
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			cmd := &SetCommand{
				URL:          "https://example.com",
				ClientID:     "test-client",
				ClientSecret: tt.clientSecret,
			}

			// Save the config path in an env var for the test
			t.Setenv("UC_CONTEXT", configPath)

			err := cmd.RunE(nil, []string{"test-context"})

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)

				// Verify the context was saved correctly
				cfg, err := common.LoadFrom(configPath)
				require.NoError(t, err)

				ctx, err := cfg.GetContext("test-context")
				require.NoError(t, err)
				assert.Equal(t, "https://example.com", ctx.URL)
				assert.Equal(t, "test-client", ctx.ClientID)

				// Verify the secret validates
				assert.NoError(t, ctx.ClientSecret.Validate())
			}
		})
	}
}

func TestLoadConfig_ValidateSecrets(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with plain secret",
			configYAML: `current_context: test
contexts:
  test:
    url: https://example.com
    client_id: test-client
    client_secret: mysupersecret`,
			expectError: false,
		},
		{
			name: "valid config with aws secret and region",
			configYAML: `current_context: test
contexts:
  test:
    url: https://example.com
    client_id: test-client
    client_secret: aws://secrets/my-secret?profile=prod&region=us-east-1`,
			expectError: false,
		},
		{
			name: "invalid config with unsupported aws param",
			configYAML: `current_context: test
contexts:
  test:
    url: https://example.com
    client_id: test-client
    client_secret: aws://secrets/my-secret?invalid=param`,
			expectError: true,
			errorMsg:    "invalid client secret in context \"test\"",
		},
		{
			name: "invalid config with env query params",
			configYAML: `current_context: test
contexts:
  test:
    url: https://example.com
    client_id: test-client
    client_secret: env://MY_SECRET?profile=prod`,
			expectError: true,
			errorMsg:    "env provider does not support query parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			err := os.WriteFile(configPath, []byte(tt.configYAML), 0600)
			require.NoError(t, err)

			cfg, err := common.LoadFrom(configPath)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}
