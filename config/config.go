package config

import (
	"github.com/burntsushi/toml"

	"github.com/netrack/netrack/service/flowvisor"
	"github.com/netrack/netrack/service/metadata"
	"github.com/netrack/netrack/service/neighbor"
)

type Config struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`

	Neighbor  neighbor.Config  `toml:"neighbor"`
	Metadata  metadata.Config  `toml:"metadata"`
	Flowvisor flowvisor.Config `toml:"flowvisor"`
}

func LoadFile(path string) (*Config, error) {
	var config Config
	_, err := toml.DecodeFile(path, &config)
	return &config, err
}
