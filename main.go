package main

import (
	"fmt"
	"os"

	"github.com/iartmediums.com/cmd_create_file/internal/commands"
)

func main() {
	args := os.Args
	err := commands.HandleCommand(args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
