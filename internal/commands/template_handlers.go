package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	parser "github.com/IArtMediums/mk/internal/template_parser"
)

func templateListHandler(cfg *cmdConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error while getting user home directory: %w", err)
	}

	templatesDir := filepath.Join(home, ".config", "mk", "templates")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return fmt.Errorf("error while reading templates directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != templateExt {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), templateExt))
	}
	sort.Strings(names)

	if len(names) == 0 {
		fmt.Println("No templates installed")
		return nil
	}

	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}

func templateEditHandler(cfg *cmdConfig) error {
	templateName := cfg.args[2]
	templatePath, err := getTemplatePath(templateName)
	if err != nil {
		return err
	}

	if _, err := os.Stat(templatePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("template %q does not exist at %s", templateName, templatePath)
		}
		return fmt.Errorf("error while checking template %q: %w", templateName, err)
	}

	return openInEditor(templatePath, cfg)
}

func templateRemoveHandler(cfg *cmdConfig) error {
	templateName := cfg.args[2]
	templatePath, err := getTemplatePath(templateName)
	if err != nil {
		return err
	}

	if err := os.Remove(templatePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("template %q does not exist at %s", templateName, templatePath)
		}
		return fmt.Errorf("error while removing template %q: %w", templateName, err)
	}

	fmt.Printf("Removed template: %s\n", templatePath)
	return nil
}

func templateVerifyHandler(cfg *cmdConfig) error {
	templateName := cfg.args[2]
	templatePath, err := getTemplatePath(templateName)
	if err != nil {
		return err
	}

	tpl, err := parser.ParseTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("template %q is invalid: %w", templateName, err)
	}

	var out bytes.Buffer
	err = tpl.Execute(parser.ExecuteOptions{
		ProjectName: "example-project",
		ProjectRoot: filepath.Join(os.TempDir(), "mk-template-validate"),
		ModulePath:  buildDefaultModulePath(cfg.config, "example-project"),
		DryRun:      true,
		Stdout:      &out,
	})
	if err != nil {
		return fmt.Errorf("template %q failed validation: %w", templateName, err)
	}

	fmt.Printf("template %q verified successfully\n", templateName)
	return nil
}

func configEditHandler(cfg *cmdConfig) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}
	return openInEditor(configPath, cfg)
}
