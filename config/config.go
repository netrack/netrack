package config

import (
	"fmt"

	"github.com/BurntSushi/toml"

	"github.com/netrack/netrack/config/environment"
)

// Config represents global configuration.
type Config struct {
	ID string `toml:"instance_id"`

	Database map[string]DatabaseConfig `toml:"database"`
}

func (c *Config) ConnString() string {
	dbconfig := c.Database[environment.Env]
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s",
		dbconfig.User, dbconfig.Password, dbconfig.DBName, dbconfig.SSLMode)
}

// Database configuration placeholder.
type DatabaseConfig struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	DBName   string `toml:"dbname"`
	SSLMode  string `toml:"sslmode"`
}

func LoadFile(path string) (*Config, error) {
	var config Config
	_, err := toml.DecodeFile(path, &config)
	return &config, err
}
