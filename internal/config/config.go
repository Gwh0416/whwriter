package config

import "os"

type Mode string

const (
	ModeDev  Mode = "dev"
	ModeProd Mode = "prod"
)

type Config struct {
	Mode Mode
	Host string
	Port string
}

func Load() *Config {
	return &Config{
		Mode: Mode(getEnv("APP_MODE", "dev")),
		Host: getEnv("APP_HOST", "0.0.0.0"),
		Port: getEnv("APP_PORT", "8080"),
	}
}

func (c *Config) Addr() string {
	return c.Host + ":" + c.Port
}

func (c *Config) IsDev() bool {
	return c.Mode == ModeDev
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
