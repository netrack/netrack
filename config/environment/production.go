//+build production

package environment

import (
	_ "github.com/netrack/netrack/netutil/drivers"
	_ "github.com/netrack/netrack/netutil/ip.v4"
	_ "github.com/netrack/netrack/netutil/ofp.v13"
)

const Env = "production"
