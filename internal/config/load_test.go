package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		errContains string
		checkConfig func(*testing.T, *Config)
	}{
		{
			name: "Valid configuration with default timeouts",
			yamlContent: `
server:
  port: 8080
  host: "127.0.0.1"
services:
  - name: "test-service"
    path_prefix: "/api/test"
    target:
      - "http://localhost:8081"
`,
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 8080, cfg.Server.Port)
				assert.Equal(t, "127.0.0.1", cfg.Server.Host)
				assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
				assert.Equal(t, 15*time.Second, cfg.Server.WriteTimeout)
				assert.Equal(t, 60*time.Second, cfg.Server.IdleTimeout)
				require.Len(t, cfg.Services, 1)
				assert.Equal(t, "test-service", cfg.Services[0].Name)
				assert.Equal(t, "/api/test", cfg.Services[0].PathPrefix)
				assert.Equal(t, []string{"http://localhost:8081"}, cfg.Services[0].Target)
			},
		},
		{
			name: "Valid configuration with custom timeouts and empty host",
			yamlContent: `
server:
  port: 9000
  read_timeout: "30s"
  write_timeout: "45s"
  idle_timeout: "120s"
services:
  - name: "svc"
    path_prefix: "/test"
    target:
      - "http://backend"
`,
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 9000, cfg.Server.Port)
				assert.Equal(t, "0.0.0.0", cfg.Server.Host)
				assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
				assert.Equal(t, 45*time.Second, cfg.Server.WriteTimeout)
				assert.Equal(t, 120*time.Second, cfg.Server.IdleTimeout)
			},
		},
		{
			name: "Invalid port (missing/zero)",
			yamlContent: `
server:
  host: "127.0.0.1"
services:
  - name: "svc"
    path_prefix: "/test"
    target: ["http://backend"]
`,
			wantErr:     true,
			errContains: "server port must be greater than 0",
		},
		{
			name: "Missing services",
			yamlContent: `
server:
  port: 8080
`,
			wantErr:     true,
			errContains: "at least one service must be defined",
		},
		{
			name: "Missing service name",
			yamlContent: `
server:
  port: 8080
services:
  - path_prefix: "/api"
    target: ["http://backend"]
`,
			wantErr:     true,
			errContains: "missing name",
		},
		{
			name: "Missing service path_prefix",
			yamlContent: `
server:
  port: 8080
services:
  - name: "svc"
    target: ["http://backend"]
`,
			wantErr:     true,
			errContains: "missing path_prefix",
		},
		{
			name: "Missing service targets",
			yamlContent: `
server:
  port: 8080
services:
  - name: "svc"
    path_prefix: "/api"
`,
			wantErr:     true,
			errContains: "must have at least one target",
		},
		{
			name: "Invalid target URL",
			yamlContent: `
server:
  port: 8080
services:
  - name: "svc"
    path_prefix: "/api"
    target: ["not a valid url"]
`,
			wantErr:     true,
			errContains: "invalid target URL",
		},
		{
			name:        "Invalid YAML syntax",
			yamlContent: "server:\n  port: 8080\n services:",
			wantErr:     true,
			errContains: "failed to decode YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tempFile := filepath.Join(tempDir, "config.yml")
			err := os.WriteFile(tempFile, []byte(tt.yamlContent), 0600)
			require.NoError(t, err)

			cfg, err := Load(tempFile)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				if tt.checkConfig != nil {
					tt.checkConfig(t, cfg)
				}
			}
		})
	}
}

func TestLoadFileNotFound(t *testing.T) {
	cfg, err := Load("non_existent_file.yml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
	assert.Nil(t, cfg)
}
