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

type ServiceConfig struct {
	Name         string       `yaml:"name"`
	PathPrefix   string       `yaml:"path_prefix"`
	Target       []string     `yaml:"target"`
	LoadBalancer LoadBalancer `yaml:"load_balancer"`
}
