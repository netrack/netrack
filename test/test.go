//+build test

package test

import (
	"github.com/netrack/netrack/config"
)

var (
	initialied    bool
	configuration *config.Config
)

func Config() (c *config.Config, err error) {
	if !initialied {
		configuration, err = config.LoadFile("../config/config.toml")
	}

	initialied = true
	return configuration, err
}
