package commands

import (
	"fmt"
	"os"
	"path/filepath"

	helperfuncs "github.com/iartmediums/mk-cli/internal/helper_funcs"
)

const templateSkeleton = `# dir

# file

# cmd
`

func createTemplateHandler(cfg *cmdConfig) error {
	templateName := cfg.args[2]

	templatePath, err := getTemplatePath(templateName)
	if err != nil {
		return err
	}

	if err := helperfuncs.CreateDir(filepath.Dir(templatePath)); err != nil {
		return fmt.Errorf("error while creating template directory: %w", err)
	}

	if err := helperfuncs.WriteFile(templatePath, templateSkeleton); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("template %q already exists at %s", templateName, templatePath)
		}
		return fmt.Errorf("error while creating template: %w", err)
	}

	fmt.Printf("Created template: %s\n", templatePath)
	if cfg.config.EditOnTmplCreation {
		return openInEditor(templatePath, cfg)
	}
	return nil
}
