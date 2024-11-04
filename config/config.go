package config

import (
	"github.com/pelletier/go-toml/v2"
	"os"
)

func ParseConfig(filePath string) (*Config, error) {
	var cfg Config

	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
