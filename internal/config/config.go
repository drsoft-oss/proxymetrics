// Package config loads, env-substitutes, and validates the YAML config.
package config

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   Server   `yaml:"server"`
	Storage  Storage  `yaml:"storage"`
	Events   Events   `yaml:"events"`
	Shutdown Shutdown `yaml:"shutdown"`
	Defaults Defaults `yaml:"defaults"`
	IPCheck  IPCheck  `yaml:"ipcheck"`
	Geo      Geo      `yaml:"geo"`
	Audit    Audit    `yaml:"audit"`
	Logging  Logging  `yaml:"logging"`
}

type Server struct {
	ProxyListen      string `yaml:"proxy_listen"`
	APIListen        string `yaml:"api_listen"`
	DeploymentSecret string `yaml:"deployment_secret"`
}

type Storage struct {
	DataDir string `yaml:"data_dir"`
}

type Events struct {
	ChannelSize   int           `yaml:"channel_size"`
	BatchSize     int           `yaml:"batch_size"`
	FlushInterval time.Duration `yaml:"flush_interval"`
}

type Shutdown struct {
	DrainTimeout time.Duration `yaml:"drain_timeout"`
}

type Defaults struct {
	Team    string `yaml:"team"`
	Project string `yaml:"project"`
}

type IPCheck struct {
	APIs     []string `yaml:"apis"`
	Rotation string   `yaml:"rotation"`
}

type Geo struct {
	IPAPI     GeoIPAPI     `yaml:"ipapi"`
	MaxMind   GeoMaxMind   `yaml:"maxmind"`
	Nominatim GeoNominatim `yaml:"nominatim"`
}

type GeoIPAPI struct {
	BaseURL string `yaml:"base_url"`
}

type GeoMaxMind struct {
	CityDB           string `yaml:"city_db"`
	ConnectionTypeDB string `yaml:"connection_type_db"`
	ISPDB            string `yaml:"isp_db"`
}

type GeoNominatim struct {
	BaseURL   string        `yaml:"base_url"`
	UserAgent string        `yaml:"user_agent"`
	Timeout   time.Duration `yaml:"timeout"`
}

type Audit struct {
	PerRequestTimeout time.Duration `yaml:"per_request_timeout"`
	MaxInFlight       int           `yaml:"max_in_flight"`
	MinRequestsPerRun int           `yaml:"min_requests_per_run"`
	MaxRequestsPerRun int           `yaml:"max_requests_per_run"`
}

type Logging struct {
	Level  string `yaml:"level"`  // debug|info|warn|error
	Format string `yaml:"format"` // auto|text|json
}

var envRE = regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)

// Load reads, env-substitutes, parses, and validates the YAML config at path.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	subbed, err := substituteEnv(string(raw))
	if err != nil {
		return Config{}, err
	}

	var c Config
	if err := yaml.Unmarshal([]byte(subbed), &c); err != nil {
		return Config{}, fmt.Errorf("parse yaml: %w", err)
	}
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func substituteEnv(in string) (string, error) {
	var missing []string
	out := envRE.ReplaceAllStringFunc(in, func(match string) string {
		name := match[2 : len(match)-1]
		v, ok := os.LookupEnv(name)
		if !ok {
			missing = append(missing, name)
			return match
		}
		return v
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("env var(s) not set: %v", missing)
	}
	return out, nil
}

func (c *Config) applyDefaults() {
	if c.Events.ChannelSize == 0 {
		c.Events.ChannelSize = 10000
	}
	if c.Events.BatchSize == 0 {
		c.Events.BatchSize = 500
	}
	if c.Events.FlushInterval == 0 {
		c.Events.FlushInterval = time.Second
	}
	if c.Shutdown.DrainTimeout == 0 {
		c.Shutdown.DrainTimeout = 30 * time.Second
	}
	if c.Defaults.Team == "" {
		c.Defaults.Team = "default"
	}
	if c.Defaults.Project == "" {
		c.Defaults.Project = "default"
	}
	if len(c.IPCheck.APIs) == 0 {
		c.IPCheck.APIs = []string{
			"https://api.ipify.org?format=json",
			"https://ifconfig.me/all.json",
			"https://icanhazip.com",
		}
	}
	if c.IPCheck.Rotation == "" {
		c.IPCheck.Rotation = "round_robin"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "auto"
	}
	if c.Geo.IPAPI.BaseURL == "" {
		c.Geo.IPAPI.BaseURL = "https://api.ipapi.is/"
	}
	if c.Geo.Nominatim.BaseURL == "" {
		c.Geo.Nominatim.BaseURL = "https://nominatim.openstreetmap.org/search"
	}
	if c.Geo.Nominatim.UserAgent == "" {
		c.Geo.Nominatim.UserAgent = "proxymetrics-audit/1.0"
	}
	if c.Geo.Nominatim.Timeout == 0 {
		c.Geo.Nominatim.Timeout = 10 * time.Second
	}
	if c.Audit.PerRequestTimeout == 0 {
		c.Audit.PerRequestTimeout = 30 * time.Second
	}
	if c.Audit.MaxInFlight == 0 {
		c.Audit.MaxInFlight = 5
	}
	if c.Audit.MinRequestsPerRun == 0 {
		c.Audit.MinRequestsPerRun = 100
	}
	if c.Audit.MaxRequestsPerRun == 0 {
		c.Audit.MaxRequestsPerRun = 1000
	}
}

func (c Config) validate() error {
	required := []struct {
		field, val string
	}{
		{"server.proxy_listen", c.Server.ProxyListen},
		{"server.api_listen", c.Server.APIListen},
		{"server.deployment_secret", c.Server.DeploymentSecret},
		{"storage.data_dir", c.Storage.DataDir},
	}
	for _, r := range required {
		if r.val == "" {
			return fmt.Errorf("required field missing: %s", r.field)
		}
	}
	if c.IPCheck.Rotation != "round_robin" {
		return fmt.Errorf("ipcheck.rotation: unsupported value %q (only \"round_robin\" is supported in v1)", c.IPCheck.Rotation)
	}
	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logging.level: invalid value %q", c.Logging.Level)
	}
	switch c.Logging.Format {
	case "auto", "text", "json":
	default:
		return fmt.Errorf("logging.format: invalid value %q", c.Logging.Format)
	}
	if c.Events.ChannelSize < 0 {
		return fmt.Errorf("events.channel_size: must be >= 0")
	}
	if c.Events.BatchSize <= 0 {
		return fmt.Errorf("events.batch_size: must be > 0")
	}
	if c.Events.FlushInterval <= 0 {
		return fmt.Errorf("events.flush_interval: must be > 0")
	}
	if c.Audit.MinRequestsPerRun < 1 {
		return fmt.Errorf("audit.min_requests_per_run: must be >= 1")
	}
	if c.Audit.MaxRequestsPerRun < c.Audit.MinRequestsPerRun {
		return fmt.Errorf("audit.max_requests_per_run: must be >= min_requests_per_run")
	}
	if c.Audit.MaxInFlight < 1 {
		return fmt.Errorf("audit.max_in_flight: must be >= 1")
	}
	if c.Audit.PerRequestTimeout <= 0 {
		return fmt.Errorf("audit.per_request_timeout: must be > 0")
	}
	return nil
}
