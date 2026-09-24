package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const (
	defaultEndpoint = "http://127.0.0.1:8000"
	defaultFormat   = "table"
)

type settings struct {
	APIKey   string
	Endpoint string
	Team     string
	Format   string
}

func (s settings) validate() error {
	if s.Format != "table" && s.Format != "json" {
		return fmt.Errorf(`format must be "table" or "json"`)
	}

	return nil
}

// resolveSettings builds the effective settings. The config file is the base,
// then FORJA_* environment variables override it, then explicit flags. A
// missing config file is not an error; a malformed one is.
func resolveSettings(cfgFile string, overrides map[string]*string) (settings, error) {
	v := viper.New()
	v.SetEnvPrefix("FORJA")
	v.AutomaticEnv()
	v.SetDefault("endpoint", defaultEndpoint)
	v.SetDefault("format", defaultFormat)

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(home)
		v.SetConfigType("yaml")
		v.SetConfigName(".forja")
	}

	var missing viper.ConfigFileNotFoundError
	if err := v.ReadInConfig(); err != nil && !errors.As(err, &missing) && !errors.Is(err, os.ErrNotExist) {
		return settings{}, fmt.Errorf("reading config: %w", err)
	}

	s := settings{
		APIKey:   v.GetString("api_key"),
		Endpoint: v.GetString("endpoint"),
		Team:     v.GetString("team"),
		Format:   v.GetString("format"),
	}

	for key, value := range overrides {
		if value == nil || *value == "" {
			continue
		}

		switch key {
		case "api_key":
			s.APIKey = *value
		case "endpoint":
			s.Endpoint = *value
		case "team":
			s.Team = *value
		case "format":
			s.Format = *value
		}
	}

	s.Endpoint = strings.TrimRight(s.Endpoint, "/")

	return s, s.validate()
}
