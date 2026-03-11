package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/IArtMediums/mk-cli/internal/commands"
)

var version = "dev"

func main() {
	if buildInfo, ok := debug.ReadBuildInfo(); ok && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		version = buildInfo.Main.Version
	}

	args := os.Args
	err := commands.HandleCommand(args[1:], version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
