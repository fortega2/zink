package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultReadTimeout  = 15 * time.Second
	defaultWriteTimeout = 15 * time.Second
	defaultIdleTimeout  = 60 * time.Second
)

func Load(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}

	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = defaultReadTimeout
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = defaultWriteTimeout
	}
	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = defaultIdleTimeout
	}

	return &cfg, nil
}
