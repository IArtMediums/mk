package commands

import (
	"fmt"
	"os"

	helperfuncs "github.com/IArtMediums/mk-cli/internal/helper_funcs"
)

func pathHandler(cfg *cmdConfig) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error while getting current directory: %w", err)
	}

	createdFiles := []string{}

	for _, pathToCreate := range cfg.args {
		fullPath, err := helperfuncs.ResolveWithinRoot(cwd, pathToCreate)
		if err != nil {
			return fmt.Errorf("%s\n not a valid path: %v", pathToCreate, err)
		}

		if helperfuncs.PathLooksLikeDir(pathToCreate) {
			if err := helperfuncs.CreateDir(fullPath); err != nil {
				return err
			}
			continue
		}

		if err := helperfuncs.CreateFile(fullPath); err != nil {
			return err
		}
		createdFiles = append(createdFiles, fullPath)
	}

	if cfg.edit {
		for _, file := range createdFiles {
			if err := openInEditor(file, cfg); err != nil {
				return err
			}
		}
	}

	return nil
}
