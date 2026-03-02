package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultRedTimeout   = 15 * time.Second
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
		cfg.Server.ReadTimeout = defaultRedTimeout
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = defaultWriteTimeout
	}
	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = defaultIdleTimeout
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
		if err := validateService(i, svc); err != nil {
			return err
		}
	}

	return nil
}

func validateService(i int, svc ServiceConfig) error {
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
		if err := validateTargetURL(svc.Name, j, targetURL); err != nil {
			return err
		}
	}

	if err := validateRateLimit(svc.Name, svc.RateLimit); err != nil {
		return err
	}

	return nil
}

func validateRateLimit(svcName string, rl RateLimitConfig) error {
	if !rl.Enabled {
		return nil
	}
	if rl.Rate <= 0 {
		return fmt.Errorf("service '%s': rate_limit.rate must be greater than 0 when enabled", svcName)
	}
	if rl.Burst <= 0 {
		return fmt.Errorf("service '%s': rate_limit.burst must be greater than 0 when enabled", svcName)
	}
	return nil
}

func validateTargetURL(svcName string, j int, targetURL string) error {
	parsed, err := url.ParseRequestURI(targetURL)
	if err != nil {
		return fmt.Errorf("invalid target URL '%s' in service '%s' (index %d): %w", targetURL, svcName, j, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("target URL '%s' in service '%s' (index %d) has unsupported scheme '%s': only http and https are allowed",
			targetURL, svcName, j, parsed.Scheme)
	}

	return nil
}
