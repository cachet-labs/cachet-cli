package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Provider    string       `mapstructure:"provider"`
	APIKey      string       `mapstructure:"apiKey"`
	Model       string       `mapstructure:"model"`
	Temperature float64      `mapstructure:"temperature"`
	Redact      RedactConfig `mapstructure:"redact"`
	Dev         DevConfig    `mapstructure:"dev"`
}

type RedactConfig struct {
	Headers  []string `mapstructure:"headers"`
	Patterns []string `mapstructure:"patterns"`
}

// DevConfig holds settings for `cachet dev` — the combined dev-server + proxy supervisor.
type DevConfig struct {
	Command   string `mapstructure:"command"`   // shell command to start the dev server (e.g. "bun run dev")
	Port      int    `mapstructure:"port"`      // dev server port (default 3000)
	ProxyPort int    `mapstructure:"proxyPort"` // cachet proxy port (default 8080)
	MinStatus int    `mapstructure:"minStatus"` // lowest status code to capture (default 400)
}

// Load reads cachet.config.json from the current directory, applies env overrides.
// Returns an empty Config (not an error) when the config file is absent.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("cachet.config")
	v.SetConfigType("json")
	v.AddConfigPath(".")

	v.SetDefault("model", "claude-sonnet-4-6")
	v.SetDefault("temperature", 0.2)

	v.SetEnvPrefix("CACHET")
	_ = v.BindEnv("apiKey", "CACHET_API_KEY")
	_ = v.BindEnv("provider", "CACHET_PROVIDER")
	_ = v.BindEnv("model", "CACHET_MODEL")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
