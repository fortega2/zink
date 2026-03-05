package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const issuerExample = "https://issuer.example.com"

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
		{
			name: "Valid configuration with rate_limit middleware",
			yamlContent: `
server:
  port: 8080
services:
  - name: "svc"
    path_prefix: "/api"
    target:
      - "http://backend"
    middlewares:
      - type: rate_limit
        rate: 10
        burst: 20
`,
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Services[0].Middlewares, 1)
				rl, ok := cfg.Services[0].Middlewares[0].Value.(RateLimitMiddleware)
				require.True(t, ok)
				assert.Equal(t, float64(10), rl.Rate)
				assert.Equal(t, 20, rl.Burst)
			},
		},
		{
			name: "Invalid rate_limit middleware: rate is zero",
			yamlContent: `
server:
  port: 8080
services:
  - name: "svc"
    path_prefix: "/api"
    target:
      - "http://backend"
    middlewares:
      - type: rate_limit
        rate: 0
        burst: 10
`,
			wantErr:     true,
			errContains: "rate_limit.rate must be greater than 0",
		},
		{
			name: "Invalid rate_limit middleware: burst is zero",
			yamlContent: `
server:
  port: 8080
services:
  - name: "svc"
    path_prefix: "/api"
    target:
      - "http://backend"
    middlewares:
      - type: rate_limit
        rate: 5
        burst: 0
`,
			wantErr:     true,
			errContains: "rate_limit.burst must be greater than 0",
		},
		{
			name: "Unknown middleware type",
			yamlContent: `
server:
  port: 8080
services:
  - name: "svc"
    path_prefix: "/api"
    target:
      - "http://backend"
    middlewares:
      - type: unknown_mw
`,
			wantErr:     true,
			errContains: "unknown middleware type",
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

func TestLoadAuthMiddlewareValidation(t *testing.T) {
	baseYAML := func(publicKeyPath, issuer, audience string) string {
		return "server:\n  port: 8080\nservices:\n  - name: \"svc\"\n    path_prefix: \"/api\"\n    target:\n      - \"http://backend\"\n    middlewares:\n      - type: auth\n        public_key_path: \"" + publicKeyPath + "\"\n        issuer: \"" + issuer + "\"\n        audience: \"" + audience + "\"\n"
	}

	writeConfig := func(t *testing.T, content string) string {
		t.Helper()
		dir := t.TempDir()
		p := filepath.Join(dir, "config.yml")
		require.NoError(t, os.WriteFile(p, []byte(content), 0600))
		return p
	}

	writePEM := func(t *testing.T) string {
		t.Helper()
		const testPublicKey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA5ncNa0BaoG3G1T9E6rYC
tQ88aS34BmhS050Uey5NXCjyDa+nInsV/kKR0DdZzsxKLPn8p60r3gXYnDJ0GME1
DUjiWEU5VEyiPQWScT/MS52xYqeZPrgT3XyxWqMGkYsCm7FCUlyaS7MQdSaLTIby
1xlT8bccajd/peQ3/LWql3yj8C2uy0Q3ulx0WCJsPNpZ7oIG49B4GvEMeeEh5++1
wE4EXh2XW2+y00blNIGdqPA7A++C9Ht0JXZX04dm5P5RZDRRM73xshOBpTkRyfei
HDXTbchKkKGEOr8i+uA2Omth0aoDwqJbO55LWS+Kohh+UW9pMHSPJ62KsutoIYMI
7QIDAQAB
-----END PUBLIC KEY-----
`
		dir := t.TempDir()
		p := filepath.Join(dir, "public.pem")
		require.NoError(t, os.WriteFile(p, []byte(testPublicKey), 0600))
		return p
	}

	t.Run("missing public_key_path", func(t *testing.T) {
		cfg, err := Load(writeConfig(t, baseYAML("", issuerExample, "zink")))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth.public_key_path is required")
		assert.Nil(t, cfg)
	})

	t.Run("public_key_path file not found", func(t *testing.T) {
		cfg, err := Load(writeConfig(t, baseYAML("/nonexistent/public.pem", issuerExample, "zink")))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth.public_key_path")
		assert.Nil(t, cfg)
	})

	t.Run("public_key_path invalid PEM content", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "bad.pem")
		require.NoError(t, os.WriteFile(p, []byte("not-a-pem"), 0600))
		cfg, err := Load(writeConfig(t, baseYAML(p, issuerExample, "zink")))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth.public_key_path is not a valid RSA public key PEM")
		assert.Nil(t, cfg)
	})

	t.Run("missing issuer", func(t *testing.T) {
		pemPath := writePEM(t)
		cfg, err := Load(writeConfig(t, baseYAML(pemPath, "", "zink")))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth.issuer is required")
		assert.Nil(t, cfg)
	})

	t.Run("missing audience", func(t *testing.T) {
		pemPath := writePEM(t)
		cfg, err := Load(writeConfig(t, baseYAML(pemPath, issuerExample, "")))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth.audience is required")
		assert.Nil(t, cfg)
	})

	t.Run("valid auth middleware config", func(t *testing.T) {
		pemPath := writePEM(t)
		cfg, err := Load(writeConfig(t, baseYAML(pemPath, issuerExample, "zink")))
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Len(t, cfg.Services[0].Middlewares, 1)
		auth, ok := cfg.Services[0].Middlewares[0].Value.(AuthMiddleware)
		require.True(t, ok)
		assert.Equal(t, pemPath, auth.PublicKeyPath)
		assert.Equal(t, issuerExample, auth.Issuer)
		assert.Equal(t, "zink", auth.Audience)
	})
}
