package config

import "time"

type LoadBalancer string

const (
	LoadBalancerRoundRobin LoadBalancer = "round_robin"
)

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

type RateLimitConfig struct {
	Rate    float64 `yaml:"rate"`
	Burst   int     `yaml:"burst"`
	Enabled bool    `yaml:"enabled"`
}

type ServiceConfig struct {
	LoadBalancer LoadBalancer    `yaml:"load_balancer"`
	RateLimit    RateLimitConfig `yaml:"rate_limit"`
	Target       []string        `yaml:"target"`
	Name         string          `yaml:"name"`
	PathPrefix   string          `yaml:"path_prefix"`
}
