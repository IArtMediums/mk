package commands

import (
	"fmt"
	"os"

	helperfuncs "github.com/iartmediums.com/cmd_create_file/internal/helper_funcs"
)

func mkBasicHandler(cfg *cmdConfig) error {
	pathToCreate := cfg.args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error while getting current directory: %v", err)
	}

	fullPath, err := helperfuncs.ResolveWithinRoot(cwd, pathToCreate)
	if err != nil {
		return fmt.Errorf("%s\n not a valid path: %v", pathToCreate, err)
	}

	return helperfuncs.CreatePath(fullPath)
}
