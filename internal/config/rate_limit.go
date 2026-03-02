package config

type RateLimitMiddleware struct {
	Middleware `yaml:",inline"`

	Rate  float64 `yaml:"rate"`
	Burst int     `yaml:"burst"`
}
