package config

import (
	_ "github.com/netrack/netrack/config/environment"
)

type Config struct {
	ID string
}

func LoadFile(path string) (*Config, error) {
	// var config Config
	// _, err := toml.DecodeFile(path, &config)
	// return &config, err
	panic("PANIC PANIC PANIC")
}
