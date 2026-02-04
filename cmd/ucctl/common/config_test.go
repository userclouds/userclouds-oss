package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestGetConfigPath(t *testing.T) {
	// Save original env and restore after test
	originalEnv := os.Getenv(DefaultConfigPathVar)
	defer func() {
		if originalEnv != "" {
			os.Setenv(DefaultConfigPathVar, originalEnv)
		} else {
			os.Unsetenv(DefaultConfigPathVar)
		}
	}()

	tests := []struct {
		name         string
		explicitPath string
		envVar       string
		createLocal  bool
		setup        func(*testing.T) string
		validate     func(*testing.T, string)
	}{
		{
			name:         "explicit path has highest priority",
			explicitPath: "/explicit/path/config.yaml",
			envVar:       "/env/path/config.yaml",
			createLocal:  true,
			setup: func(t *testing.T) string {
				return ""
			},
			validate: func(t *testing.T, path string) {
				assert.Equal(t, "/explicit/path/config.yaml", path)
			},
		},
		{
			name:         "environment variable second priority",
			explicitPath: "",
			envVar:       "/env/path/config.yaml",
			createLocal:  true,
			setup: func(t *testing.T) string {
				return ""
			},
			validate: func(t *testing.T, path string) {
				assert.Equal(t, "/env/path/config.yaml", path)
			},
		},
		{
			name:         "local config file third priority",
			explicitPath: "",
			envVar:       "",
			createLocal:  true,
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				originalWd, _ := os.Getwd()
				os.Chdir(tmpDir)
				t.Cleanup(func() { os.Chdir(originalWd) })

				// Create local config file
				err := os.WriteFile(LocalConfigFile, []byte("test"), 0600)
				assert.NoError(t, err)
				return tmpDir
			},
			validate: func(t *testing.T, path string) {
				assert.Equal(t, LocalConfigFile, path)
			},
		},
		{
			name:         "default path when no others exist",
			explicitPath: "",
			envVar:       "",
			createLocal:  false,
			setup: func(t *testing.T) string {
				return ""
			},
			validate: func(t *testing.T, path string) {
				home, _ := os.UserHomeDir()
				expected := filepath.Join(home, DefaultConfigDir, DefaultConfigFile)
				assert.Equal(t, expected, path)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env
			os.Unsetenv(DefaultConfigPathVar)

			// Setup test environment
			tt.setup(t)

			// Set environment variable if needed
			if tt.envVar != "" {
				os.Setenv(DefaultConfigPathVar, tt.envVar)
				defer os.Unsetenv(DefaultConfigPathVar)
			}

			// Get config path
			path, err := GetConfigPath(tt.explicitPath)
			assert.NoError(t, err)

			// Validate result
			tt.validate(t, path)
		})
	}
}

func TestLoadFrom(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*testing.T) string
		validate func(*testing.T, *Config, error)
	}{
		{
			name: "load valid config",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "config.yaml")

				cfg := &Config{
					CurrentContext: "prod",
					Contexts: map[string]*Context{
						"prod": {
							URL:          "https://prod.example.com",
							ClientID:     "prod-client",
							ClientSecret: "prod-secret",
						},
						"staging": {
							URL:          "https://staging.example.com",
							ClientID:     "staging-client",
							ClientSecret: "staging-secret",
						},
					},
				}

				data, err := yaml.Marshal(cfg)
				assert.NoError(t, err)
				err = os.WriteFile(configPath, data, 0600)
				assert.NoError(t, err)

				return configPath
			},
			validate: func(t *testing.T, cfg *Config, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "prod", cfg.CurrentContext)
				assert.Len(t, cfg.Contexts, 2)
				assert.Equal(t, "https://prod.example.com", cfg.Contexts["prod"].URL)
				assert.Equal(t, "prod-client", cfg.Contexts["prod"].ClientID)
				assert.Equal(t, "prod-secret", cfg.Contexts["prod"].ClientSecret)
			},
		},
		{
			name: "file does not exist returns empty config",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "nonexistent.yaml")
			},
			validate: func(t *testing.T, cfg *Config, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.Empty(t, cfg.CurrentContext)
				assert.NotNil(t, cfg.Contexts)
				assert.Len(t, cfg.Contexts, 0)
			},
		},
		{
			name: "invalid YAML returns error",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "invalid.yaml")
				err := os.WriteFile(configPath, []byte("invalid: yaml: content: :"), 0600)
				assert.NoError(t, err)
				return configPath
			},
			validate: func(t *testing.T, cfg *Config, err error) {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			},
		},
		{
			name: "empty file returns empty config",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "empty.yaml")
				err := os.WriteFile(configPath, []byte(""), 0600)
				assert.NoError(t, err)
				return configPath
			},
			validate: func(t *testing.T, cfg *Config, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.NotNil(t, cfg.Contexts)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			cfg, err := LoadFrom(path)
			tt.validate(t, cfg, err)
		})
	}
}

func TestConfigSaveTo(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		validate func(*testing.T, string)
	}{
		{
			name: "save simple config",
			config: &Config{
				CurrentContext: "prod",
				Contexts: map[string]*Context{
					"prod": {
						URL:          "https://prod.example.com",
						ClientID:     "prod-client",
						ClientSecret: "prod-secret",
					},
				},
			},
			validate: func(t *testing.T, path string) {
				// Read back and verify
				cfg, err := LoadFrom(path)
				assert.NoError(t, err)
				assert.Equal(t, "prod", cfg.CurrentContext)
				assert.Len(t, cfg.Contexts, 1)
				assert.Equal(t, "https://prod.example.com", cfg.Contexts["prod"].URL)

				// Verify file permissions
				info, err := os.Stat(path)
				assert.NoError(t, err)
				assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
			},
		},
		{
			name: "save creates parent directory",
			config: &Config{
				CurrentContext: "test",
				Contexts: map[string]*Context{
					"test": {
						URL:          "https://test.example.com",
						ClientID:     "test-client",
						ClientSecret: "test-secret",
					},
				},
			},
			validate: func(t *testing.T, path string) {
				// Verify directory was created
				dir := filepath.Dir(path)
				info, err := os.Stat(dir)
				assert.NoError(t, err)
				assert.True(t, info.IsDir())

				// Verify file exists and is readable
				cfg, err := LoadFrom(path)
				assert.NoError(t, err)
				assert.Equal(t, "test", cfg.CurrentContext)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "subdir", "config.yaml")

			err := tt.config.SaveTo(configPath)
			assert.NoError(t, err)

			tt.validate(t, configPath)
		})
	}
}

func TestConfigGetContext(t *testing.T) {
	cfg := &Config{
		CurrentContext: "prod",
		Contexts: map[string]*Context{
			"prod": {
				URL:          "https://prod.example.com",
				ClientID:     "prod-client",
				ClientSecret: "prod-secret",
			},
			"staging": {
				URL:          "https://staging.example.com",
				ClientID:     "staging-client",
				ClientSecret: "staging-secret",
			},
		},
	}

	tests := []struct {
		name        string
		contextName string
		wantErr     bool
		validate    func(*testing.T, *Context)
	}{
		{
			name:        "get existing context",
			contextName: "prod",
			wantErr:     false,
			validate: func(t *testing.T, ctx *Context) {
				assert.Equal(t, "https://prod.example.com", ctx.URL)
				assert.Equal(t, "prod-client", ctx.ClientID)
			},
		},
		{
			name:        "get non-existent context",
			contextName: "nonexistent",
			wantErr:     true,
			validate:    func(t *testing.T, ctx *Context) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := cfg.GetContext(tt.contextName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ctx)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ctx)
				tt.validate(t, ctx)
			}
		})
	}
}

func TestConfigGetCurrentContext(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		wantErr  bool
		validate func(*testing.T, *Context)
	}{
		{
			name: "get current context when set",
			config: &Config{
				CurrentContext: "prod",
				Contexts: map[string]*Context{
					"prod": {
						URL:          "https://prod.example.com",
						ClientID:     "prod-client",
						ClientSecret: "prod-secret",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, ctx *Context) {
				assert.Equal(t, "https://prod.example.com", ctx.URL)
			},
		},
		{
			name: "error when no current context",
			config: &Config{
				CurrentContext: "",
				Contexts: map[string]*Context{
					"prod": {
						URL:          "https://prod.example.com",
						ClientID:     "prod-client",
						ClientSecret: "prod-secret",
					},
				},
			},
			wantErr:  true,
			validate: func(t *testing.T, ctx *Context) {},
		},
		{
			name: "error when current context not found",
			config: &Config{
				CurrentContext: "nonexistent",
				Contexts: map[string]*Context{
					"prod": {
						URL:          "https://prod.example.com",
						ClientID:     "prod-client",
						ClientSecret: "prod-secret",
					},
				},
			},
			wantErr:  true,
			validate: func(t *testing.T, ctx *Context) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := tt.config.GetCurrentContext()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ctx)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ctx)
				tt.validate(t, ctx)
			}
		})
	}
}

func TestConfigSetContext(t *testing.T) {
	tests := []struct {
		name     string
		initial  *Config
		ctxName  string
		ctx      *Context
		validate func(*testing.T, *Config)
	}{
		{
			name: "add new context",
			initial: &Config{
				Contexts: make(map[string]*Context),
			},
			ctxName: "new",
			ctx: &Context{
				URL:          "https://new.example.com",
				ClientID:     "new-client",
				ClientSecret: "new-secret",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Len(t, cfg.Contexts, 1)
				assert.Equal(t, "https://new.example.com", cfg.Contexts["new"].URL)
			},
		},
		{
			name: "update existing context",
			initial: &Config{
				Contexts: map[string]*Context{
					"prod": {
						URL:          "https://old.example.com",
						ClientID:     "old-client",
						ClientSecret: "old-secret",
					},
				},
			},
			ctxName: "prod",
			ctx: &Context{
				URL:          "https://new.example.com",
				ClientID:     "new-client",
				ClientSecret: "new-secret",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.Len(t, cfg.Contexts, 1)
				assert.Equal(t, "https://new.example.com", cfg.Contexts["prod"].URL)
			},
		},
		{
			name: "initialize contexts map if nil",
			initial: &Config{
				Contexts: nil,
			},
			ctxName: "new",
			ctx: &Context{
				URL:          "https://new.example.com",
				ClientID:     "new-client",
				ClientSecret: "new-secret",
			},
			validate: func(t *testing.T, cfg *Config) {
				assert.NotNil(t, cfg.Contexts)
				assert.Len(t, cfg.Contexts, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.SetContext(tt.ctxName, tt.ctx)
			tt.validate(t, tt.initial)
		})
	}
}

func TestConfigDeleteContext(t *testing.T) {
	tests := []struct {
		name     string
		initial  *Config
		ctxName  string
		wantErr  bool
		validate func(*testing.T, *Config)
	}{
		{
			name: "delete existing context",
			initial: &Config{
				CurrentContext: "staging",
				Contexts: map[string]*Context{
					"prod": {
						URL:          "https://prod.example.com",
						ClientID:     "prod-client",
						ClientSecret: "prod-secret",
					},
					"staging": {
						URL:          "https://staging.example.com",
						ClientID:     "staging-client",
						ClientSecret: "staging-secret",
					},
				},
			},
			ctxName: "prod",
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Len(t, cfg.Contexts, 1)
				assert.Nil(t, cfg.Contexts["prod"])
				assert.NotNil(t, cfg.Contexts["staging"])
				assert.Equal(t, "staging", cfg.CurrentContext)
			},
		},
		{
			name: "delete current context unsets it",
			initial: &Config{
				CurrentContext: "prod",
				Contexts: map[string]*Context{
					"prod": {
						URL:          "https://prod.example.com",
						ClientID:     "prod-client",
						ClientSecret: "prod-secret",
					},
				},
			},
			ctxName: "prod",
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Len(t, cfg.Contexts, 0)
				assert.Empty(t, cfg.CurrentContext)
			},
		},
		{
			name: "delete non-existent context returns error",
			initial: &Config{
				Contexts: map[string]*Context{
					"prod": {
						URL:          "https://prod.example.com",
						ClientID:     "prod-client",
						ClientSecret: "prod-secret",
					},
				},
			},
			ctxName: "nonexistent",
			wantErr: true,
			validate: func(t *testing.T, cfg *Config) {
				assert.Len(t, cfg.Contexts, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initial.DeleteContext(tt.ctxName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			tt.validate(t, tt.initial)
		})
	}
}

func TestConfigUseContext(t *testing.T) {
	tests := []struct {
		name     string
		initial  *Config
		ctxName  string
		wantErr  bool
		validate func(*testing.T, *Config)
	}{
		{
			name: "switch to existing context",
			initial: &Config{
				CurrentContext: "prod",
				Contexts: map[string]*Context{
					"prod": {
						URL:          "https://prod.example.com",
						ClientID:     "prod-client",
						ClientSecret: "prod-secret",
					},
					"staging": {
						URL:          "https://staging.example.com",
						ClientID:     "staging-client",
						ClientSecret: "staging-secret",
					},
				},
			},
			ctxName: "staging",
			wantErr: false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "staging", cfg.CurrentContext)
			},
		},
		{
			name: "switch to non-existent context returns error",
			initial: &Config{
				CurrentContext: "prod",
				Contexts: map[string]*Context{
					"prod": {
						URL:          "https://prod.example.com",
						ClientID:     "prod-client",
						ClientSecret: "prod-secret",
					},
				},
			},
			ctxName: "nonexistent",
			wantErr: true,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "prod", cfg.CurrentContext)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.initial.UseContext(tt.ctxName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			tt.validate(t, tt.initial)
		})
	}
}

func TestConfigRoundTrip(t *testing.T) {
	// Test that we can save and load a config without data loss
	original := &Config{
		CurrentContext: "prod",
		Contexts: map[string]*Context{
			"prod": {
				URL:          "https://prod.example.com",
				ClientID:     "prod-client",
				ClientSecret: "prod-secret",
			},
			"staging": {
				URL:          "https://staging.example.com",
				ClientID:     "staging-client",
				ClientSecret: "staging-secret",
			},
		},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Save
	err := original.SaveTo(configPath)
	assert.NoError(t, err)

	// Load
	loaded, err := LoadFrom(configPath)
	assert.NoError(t, err)

	// Compare
	assert.Equal(t, original.CurrentContext, loaded.CurrentContext)
	assert.Len(t, loaded.Contexts, len(original.Contexts))

	for name, ctx := range original.Contexts {
		loadedCtx := loaded.Contexts[name]
		assert.NotNil(t, loadedCtx)
		assert.Equal(t, ctx.URL, loadedCtx.URL)
		assert.Equal(t, ctx.ClientID, loadedCtx.ClientID)
		assert.Equal(t, ctx.ClientSecret, loadedCtx.ClientSecret)
	}
}
