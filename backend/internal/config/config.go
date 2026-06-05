package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Mode string

const (
	ModeDev  Mode = "dev"
	ModeProd Mode = "prod"
)

type Config struct {
	App   AppConfig   `yaml:"app"`
	MySQL MySQLConfig `yaml:"mysql"`
	JWT   JWTConfig   `yaml:"jwt"`
	Admin AdminConfig `yaml:"admin"`
	SMTP  SMTPConfig  `yaml:"smtp"`
	LLM   LLMConfig   `yaml:"llm"`
}

type AppConfig struct {
	Mode Mode   `yaml:"mode"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	Charset  string `yaml:"charset"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
}

type AdminConfig struct {
	Email    string `yaml:"email"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type LLMConfig struct {
	DefaultTimeoutSeconds int `yaml:"default_timeout_seconds"`
	PlannerTimeoutSeconds int `yaml:"planner_timeout_seconds"`
	WriterTimeoutSeconds  int `yaml:"writer_timeout_seconds"`
	SettlerTimeoutSeconds int `yaml:"settler_timeout_seconds"`
	AuditorTimeoutSeconds int `yaml:"auditor_timeout_seconds"`
	ReviserTimeoutSeconds int `yaml:"reviser_timeout_seconds"`
}

func Load(path string) *Config {
	cfg := &Config{
		App: AppConfig{
			Mode: ModeDev,
			Host: "0.0.0.0",
			Port: 8080,
		},
		MySQL: MySQLConfig{
			Host:     "127.0.0.1",
			Port:     3306,
			User:     "whwriter",
			Password: "whwriter123",
			Database: "whwriter",
			Charset:  "utf8mb4",
		},
		JWT: JWTConfig{
			Secret: "whwriter-jwt-secret-change-in-production",
		},
		Admin: AdminConfig{
			Email:    "admin@whwriter.com",
			Username: "admin",
			Password: "Admin123456",
		},
		SMTP: SMTPConfig{
			Port: 587,
		},
		LLM: LLMConfig{
			DefaultTimeoutSeconds: 120,
			PlannerTimeoutSeconds: 120,
			WriterTimeoutSeconds:  300,
			SettlerTimeoutSeconds: 180,
			AuditorTimeoutSeconds: 180,
			ReviserTimeoutSeconds: 180,
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: failed to read %s: %v, using defaults\n", path, err)
		return cfg
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "warn: failed to parse %s: %v\n", path, err)
	}

	return cfg
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.App.Host, c.App.Port)
}

func (c *Config) IsDev() bool {
	return c.App.Mode == ModeDev
}

func (c *Config) MySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.MySQL.User, c.MySQL.Password, c.MySQL.Host, c.MySQL.Port, c.MySQL.Database, c.MySQL.Charset)
}
