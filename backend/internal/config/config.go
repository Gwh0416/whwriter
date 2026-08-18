package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Mode string

const (
	ModeDev  Mode = "dev"
	ModeProd Mode = "prod"
)

type Config struct {
	App     AppConfig     `yaml:"app"`
	SQLite  SQLiteConfig  `yaml:"sqlite"`
	LLM     LLMConfig     `yaml:"llm"`
	Browser BrowserConfig `yaml:"browser"`
	Radar   RadarConfig   `yaml:"radar"`
}

type AppConfig struct {
	Mode Mode   `yaml:"mode"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type LLMConfig struct {
	DefaultTimeoutSeconds int `yaml:"default_timeout_seconds"`
	PlannerTimeoutSeconds int `yaml:"planner_timeout_seconds"`
	WriterTimeoutSeconds  int `yaml:"writer_timeout_seconds"`
	SettlerTimeoutSeconds int `yaml:"settler_timeout_seconds"`
	AuditorTimeoutSeconds int `yaml:"auditor_timeout_seconds"`
	ReviserTimeoutSeconds int `yaml:"reviser_timeout_seconds"`
}

type BrowserConfig struct {
	CDPURL                     string `yaml:"cdp_url"`
	ChapterFetchTimeoutSeconds int    `yaml:"chapter_fetch_timeout_seconds"`
	AutoLaunch                 bool   `yaml:"auto_launch"`
	ChromeAppName              string `yaml:"chrome_app_name"`
	UserDataDir                string `yaml:"user_data_dir"`
}

type RadarConfig struct {
	FanqieContentAPIURL            string `yaml:"fanqie_content_api_url"`
	FanqieContentAPITimeoutSeconds int    `yaml:"fanqie_content_api_timeout_seconds"`
}

func Load(path string) *Config {
	cfg := &Config{
		App: AppConfig{
			Mode: ModeDev,
			Host: "0.0.0.0",
			Port: 8080,
		},
		SQLite: SQLiteConfig{
			Path: "data/whwriter.db",
		},
		LLM: LLMConfig{
			DefaultTimeoutSeconds: 120,
			PlannerTimeoutSeconds: 120,
			WriterTimeoutSeconds:  300,
			SettlerTimeoutSeconds: 180,
			AuditorTimeoutSeconds: 180,
			ReviserTimeoutSeconds: 180,
		},
		Browser: BrowserConfig{
			CDPURL:                     "http://127.0.0.1:9222",
			ChapterFetchTimeoutSeconds: 120,
			AutoLaunch:                 true,
			ChromeAppName:              "Google Chrome",
			UserDataDir:                "$HOME/.whwriter-chrome",
		},
		Radar: RadarConfig{
			FanqieContentAPIURL:            "http://101.35.133.34:5000/api/raw_full?item_id={item_id}",
			FanqieContentAPITimeoutSeconds: 8,
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

func (c *Config) SQLitePath() string {
	if c.SQLite.Path == "" {
		return "data/whwriter.db"
	}
	return c.SQLite.Path
}

func (c *Config) BrowserCDPURL() string {
	if v := os.Getenv("WHWRITER_BROWSER_CDP_URL"); v != "" {
		return v
	}
	if strings.TrimSpace(c.Browser.CDPURL) == "" {
		return "http://127.0.0.1:9222"
	}
	return c.Browser.CDPURL
}

func (c *Config) BrowserChapterFetchTimeoutSeconds() int {
	if c.Browser.ChapterFetchTimeoutSeconds <= 0 {
		return 120
	}
	return c.Browser.ChapterFetchTimeoutSeconds
}

func (c *Config) BrowserAutoLaunch() bool {
	return c.Browser.AutoLaunch
}

func (c *Config) BrowserChromeAppName() string {
	if strings.TrimSpace(c.Browser.ChromeAppName) == "" {
		return "Google Chrome"
	}
	return strings.TrimSpace(c.Browser.ChromeAppName)
}

func (c *Config) BrowserUserDataDir() string {
	if strings.TrimSpace(c.Browser.UserDataDir) == "" {
		return os.ExpandEnv("$HOME/.whwriter-chrome")
	}
	return os.ExpandEnv(strings.TrimSpace(c.Browser.UserDataDir))
}

func (c *Config) FanqieContentAPIURL() string {
	if v := os.Getenv("WHWRITER_FANQIE_CONTENT_API_URL"); strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(c.Radar.FanqieContentAPIURL)
}

func (c *Config) FanqieContentAPITimeoutSeconds() int {
	if c.Radar.FanqieContentAPITimeoutSeconds <= 0 {
		return 8
	}
	return c.Radar.FanqieContentAPITimeoutSeconds
}
