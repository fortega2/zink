package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
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

	return &cfg, nil
}

func validate(cfg *Config) error {
	if cfg.Server.Port <= 0 {
		return errors.New("server port must be greater than 0")
	}

	if len(cfg.Services) == 0 {
		return errors.New("at least one service must be defined (services)")
	}

	for i, svc := range cfg.Services {
		if svc.Name == "" {
			return fmt.Errorf("service at index %d is missing name", i)
		}
		if svc.PathPrefix == "" {
			return fmt.Errorf("service '%s' is missing path_prefix", svc.Name)
		}
		if len(svc.Target) == 0 {
			return fmt.Errorf("service '%s' must have at least one target", svc.Name)
		}

		for j, targetURL := range svc.Target {
			if _, err := url.ParseRequestURI(targetURL); err != nil {
				return fmt.Errorf("invalid target URL '%s' in service '%s' (index %d): %w", targetURL, svc.Name, j, err)
			}
		}
	}

	return nil
}
