package commands

import (
	"fmt"
	"os"
	"path/filepath"

	helperfuncs "github.com/iartmediums.com/cmd_create_file/internal/helper_funcs"
	parser "github.com/iartmediums.com/cmd_create_file/internal/template_parser"
)

func initHandler(cfg *cmdConfig) error {
	templateName := cfg.args[0]
	projectName := cfg.args[1]

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error while getting user home directory: %v", err)
	}

	templatePath := filepath.Join(home, ".config", "mk", "templates", templateName+".mktemp")

	tpl, err := parser.ParseTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("error during template parsing: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error while getting current directory: %v", err)
	}

	projectRoot, err := helperfuncs.ResolveWithinRoot(cwd, projectName)
	if err != nil {
		return fmt.Errorf("invalid project path: %v", err)
	}

	err = tpl.Execute(filepath.Base(projectName), projectRoot)
	if err != nil {
		return fmt.Errorf("error during execution of template: %v", err)
	}

	return nil
}
