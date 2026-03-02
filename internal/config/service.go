package config

type LoadBalancer string

const (
	LoadBalancerRoundRobin LoadBalancer = "round_robin"
)

type ServiceConfig struct {
	Middlewares  []MiddlewareConfig `yaml:"middlewares"`
	Target       []string           `yaml:"target"`
	LoadBalancer LoadBalancer       `yaml:"load_balancer"`
	Name         string             `yaml:"name"`
	PathPrefix   string             `yaml:"path_prefix"`
}
