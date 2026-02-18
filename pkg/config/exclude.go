package config

import (
	"go.yaml.in/yaml/v4"
)

type Exclude struct {
	EventTypeID string `yaml:"event_type_id"`
}

type ExcludeConfig struct {
	Excludes []Exclude `yaml:"excludes"`
}

func EncodeRoots(e *[]Exclude) (string, error) {
	cc := ExcludeConfig{
		Excludes: *e,
	}

	buf, err := yaml.Marshal(&cc)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func DefaultExcludeConfig() *ExcludeConfig {
	roots := []Exclude{}

	return &ExcludeConfig{
		Excludes: roots,
	}
}
