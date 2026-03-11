package commands

import (
	"fmt"
	"os"
	"path/filepath"

	helperfuncs "github.com/IArtMediums/mk/internal/helper_funcs"
	parser "github.com/IArtMediums/mk/internal/template_parser"
)

func initHandler(cfg *cmdConfig) error {
	templateName := cfg.args[0]
	projectName := cfg.args[1]

	templatePath, err := getTemplatePath(templateName)
	if err != nil {
		return err
	}

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

	if cfg.verbose {
		fmt.Printf("Scaffolding %s from template %s\n", filepath.Base(projectName), templateName)
		fmt.Printf("Target: %s\n", projectRoot)
	}

	modulePath := cfg.modulePath
	if modulePath == "" {
		modulePath = buildDefaultModulePath(cfg.config, filepath.Base(projectName))
	}

	err = tpl.Execute(parser.ExecuteOptions{
		ProjectName: filepath.Base(projectName),
		ProjectRoot: projectRoot,
		ModulePath:  modulePath,
		Verbose:     cfg.verbose,
		Force:       cfg.force,
	})
	if err != nil {
		return fmt.Errorf("error during execution of template: %v", err)
	}

	if cfg.verbose {
		fmt.Printf("Created project: %s\n", projectRoot)
	}

	return nil
}
