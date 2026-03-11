package commands

import "fmt"

func helpHandler(cfg *cmdConfig) error {
	fmt.Printf("MK CLI TOOL USAGE: \nYou can skip -p/--path flag to create directory or file (extention is required to create a file)\n")
	fmt.Printf("All directories are created for provided path if they dont exist\n\n")

	fmt.Printf("%s", cfg.getVerboseHelpTable())

	for _, cmd := range cfg.cmdMap {
		if cmd.HelpIgnore {
			continue
		}
		fmt.Printf("%s", cmd.getHelpString())
	}
	return nil
}
