package config

import (
	"context"
	"os"

	"github.com/9506hqwy/vmomi-event-source/pkg/flag"
	"go.yaml.in/yaml/v4"
)

type Config struct {
	ExcludeConfig `yaml:",omitempty,inline"`
}

func DecodeConfig(config []byte) (*Config, error) {
	var c Config
	err := yaml.Unmarshal(config, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func EncodeConfig(c *Config) (string, error) {
	buf, err := yaml.Marshal(&c)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func DefaultConfig() *Config {
	return &Config{
		ExcludeConfig: *DefaultExcludeConfig(),
	}
}

func GetConfig(ctx context.Context) (*Config, error) {
	filePath, ok := ctx.Value(flag.LokiConfigKey{}).(string)
	if !ok || filePath == "" {
		return DefaultConfig(), nil
	}

	config, err := LoadFileConfig(filePath)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func LoadFileConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	config, err := DecodeConfig(data)
	if err != nil {
		return nil, err
	}

	return config, nil
}
