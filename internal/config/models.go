package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

type LoadBalancer string

const (
	LoadBalancerRoundRobin LoadBalancer = "round_robin"
)

type MiddlewareType string

const (
	MiddlewareRateLimit MiddlewareType = "rate_limit"
)

type Middleware struct {
	Type MiddlewareType `yaml:"type"`
}

type RateLimitMiddleware struct {
	Middleware `yaml:",inline"`

	Rate  float64 `yaml:"rate"`
	Burst int     `yaml:"burst"`
}

type MiddlewareConfig struct {
	Value any
}

func (m *MiddlewareConfig) UnmarshalYAML(value *yaml.Node) error {
	var base Middleware
	if err := value.Decode(&base); err != nil {
		return fmt.Errorf("failed to decode middleware type: %w", err)
	}

	switch base.Type {
	case MiddlewareRateLimit:
		var rl RateLimitMiddleware
		if err := value.Decode(&rl); err != nil {
			return fmt.Errorf("failed to decode rate_limit middleware: %w", err)
		}
		m.Value = rl
	default:
		return fmt.Errorf("unknown middleware type: %q", base.Type)
	}

	return nil
}

type Config struct {
	Server   ServerConfig    `yaml:"server"`
	Services []ServiceConfig `yaml:"services"`
}

type ServerConfig struct {
	Port         int           `yaml:"port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type ServiceConfig struct {
	Middlewares  []MiddlewareConfig `yaml:"middlewares"`
	Target       []string           `yaml:"target"`
	LoadBalancer LoadBalancer       `yaml:"load_balancer"`
	Name         string             `yaml:"name"`
	PathPrefix   string             `yaml:"path_prefix"`
}
