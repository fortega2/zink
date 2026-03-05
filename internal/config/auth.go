package config

type AuthMiddleware struct {
	Middleware `yaml:",inline"`

	PublicKeyPath string `yaml:"public_key_path"`
	Issuer        string `yaml:"issuer"`
	Audience      string `yaml:"audience"`
}
