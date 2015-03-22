package config

import (
	"github.com/burntsushi/toml"
)

type Config struct {
	InstID string `toml:"instance_id"`
}

func LoadFile(path string) (*Config, error) {
	var config Config
	_, err := toml.DecodeFile(path, &config)
	return &config, err
}
