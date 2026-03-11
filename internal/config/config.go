package config

import (
	"github.com/spf13/viper"
)

// Config holds global configuration for lopa.
type Config struct {
	HTTP struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"http"`

	Log struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`

	Reflector struct {
		Enabled  bool   `mapstructure:"enabled"`
		Addr     string `mapstructure:"addr"`
		TwampAddr string `mapstructure:"twamp_addr"` // optional, e.g. ":862" for TWAMP-light reflector
	} `mapstructure:"reflector"`
}

var global Config

// Load initializes configuration using viper.
func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("LOPA")
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("http.addr", ":8080")
	v.SetDefault("log.level", "info")
	v.SetDefault("reflector.enabled", true)
	v.SetDefault("reflector.addr", ":8081")
	v.SetDefault("reflector.twamp_addr", ":862") // TWAMP-light reflector; set empty to disable

	if err := v.Unmarshal(&global); err != nil {
		return nil, err
	}

	return &global, nil
}

// Global returns the loaded global configuration.
func Global() *Config {
	return &global
}

