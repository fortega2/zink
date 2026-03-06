package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type MiddlewareType string

const (
	MiddlewareRateLimit MiddlewareType = "rate_limit"
	MiddlewareAuth      MiddlewareType = "auth"
)

type Middleware struct {
	Type MiddlewareType `yaml:"type"`
}

// MiddlewareConfig holds a decoded middleware value together with its type identifier.
// TypeName is populated during YAML unmarshalling and can be used directly without
// a type switch on Value.
type MiddlewareConfig struct {
	TypeName MiddlewareType
	Value    any
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
		m.TypeName = MiddlewareRateLimit
		m.Value = rl
	case MiddlewareAuth:
		var auth AuthMiddleware
		if err := value.Decode(&auth); err != nil {
			return fmt.Errorf("failed to decode auth middleware: %w", err)
		}
		m.TypeName = MiddlewareAuth
		m.Value = auth
	default:
		return fmt.Errorf("unknown middleware type: %q", base.Type)
	}

	return nil
}
