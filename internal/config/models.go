package config

import "time"

type Config struct {
	Server   serverConfig    `yaml:"server"`
	Services []serviceConfig `yaml:"services"`
}

type serverConfig struct {
	Port         int           `yaml:"port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type serviceConfig struct {
	Name       string   `yaml:"name"`
	PathPrefix string   `yaml:"path_prefix"`
	Target     []string `yaml:"target"`
}
