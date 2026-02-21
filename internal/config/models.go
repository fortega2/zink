package config

type Config struct {
	Server   serverConfig    `yaml:"server"`
	Services []serviceConfig `yaml:"services"`
}

type serverConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type serviceConfig struct {
	Name       string   `yaml:"name"`
	PathPrefix string   `yaml:"path_prefix"`
	Target     []string `yaml:"target"`
}
