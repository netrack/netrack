package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/netrack/netrack/controller"
	_ "github.com/netrack/netrack/httprest/v1"
	_ "github.com/netrack/netrack/netutil/drivers"
	_ "github.com/netrack/netrack/netutil/ip.v4"
	_ "github.com/netrack/netrack/netutil/ofp.v13"
)

var (
	version string

	flVersion = flag.Bool("version", false, "Print version information and quit")
	flHelp    = flag.Bool("help", false, "Pring usage")
)

func Main() {
	flag.Parse()

	if *flVersion {
		flDoVersion()
		return
	}

	if *flHelp {
		flDoHelp()
		return
	}
}

func flDoHelp() {
	fmt.Fprintf(os.Stdout, "Usage: netrack [OPTIONS] COMMAND [args...]\n\n")
	fmt.Fprintf(os.Stdout, "Commands:\n")
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		fmt.Fprintf(os.Stdout, "    --%-10.10s%s\n", f.Name, f.Usage)
	})

	fmt.Fprintf(os.Stdout, "\n")
}

func flDoVersion() {
	fmt.Fprintf(os.Stdout, "%s\n", version)
}

func main() {
	c := controller.C{Addr: "192.168.0.100:6633"}
	c.ListenAndServe()
}
