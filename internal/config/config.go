package config

import "os"

type Mode string

const (
	ModeDev  Mode = "dev"
	ModeProd Mode = "prod"
)

type Config struct {
	Mode      Mode
	Host      string
	Port      string
	MySQLDSN  string
	JWTSecret string
}

func Load() *Config {
	return &Config{
		Mode:      Mode(getEnv("APP_MODE", "dev")),
		Host:      getEnv("APP_HOST", "0.0.0.0"),
		Port:      getEnv("APP_PORT", "8080"),
		MySQLDSN:  getEnv("MYSQL_DSN", "whwriter:whwriter123@tcp(127.0.0.1:3306)/whwriter?charset=utf8mb4&parseTime=True&loc=Local"),
		JWTSecret: getEnv("JWT_SECRET", "whwriter-jwt-secret-change-in-production"),
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
