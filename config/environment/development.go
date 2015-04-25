//+build development

package environment

import (
	// Register http api
	_ "github.com/netrack/netrack/httprest/v1"

	// Register modules
	_ "github.com/netrack/netrack/netutil/drivers"
	_ "github.com/netrack/netrack/netutil/ip.v4"
	_ "github.com/netrack/netrack/netutil/ofp.v13"
)

const Env = "development"
