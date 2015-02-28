package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	version string

	flVersion = flag.Bool("version", false, "Print version information and quit")
	flHelp    = flag.Bool("help", false, "Pring usage")
)

func main() {
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
