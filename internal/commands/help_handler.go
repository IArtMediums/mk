package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
)

func printHelpSection(title string, keys []string, help map[string]CommandHelp) error {
	fmt.Printf("%s:\n", title)
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	for _, key := range keys {
		item := help[key]
		fmt.Fprintf(writer, "  %s\t%s\n", item.Usage, item.Description)
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	fmt.Println()
	return nil
}

func helpHandler(cfg *cmdConfig) error {
	fmt.Println("mk - smart file, directory, and project template creation tool")
	fmt.Println()

	help := commandHelp()
	if err := printHelpSection("Basic Usage", []string{"path", "path-edit"}, help); err != nil {
		return err
	}
	if err := printHelpSection("Template Commands", []string{"template-init", "template-new", "template-edit", "template-remove", "template-list", "template-verify", "template-help"}, help); err != nil {
		return err
	}
	if err := printHelpSection("Configuration Commands", []string{"config-edit", "config-setup", "config-help"}, help); err != nil {
		return err
	}
	if err := printHelpSection("General Help", []string{"help", "version"}, help); err != nil {
		return err
	}

	fmt.Println("Reserved commands: tmpl, config, help")
	fmt.Println(`Use "./name" if you want to create a path with one of those names.`)
	return nil
}

func versionHandler(version string) error {
	if version == "" {
		version = "dev"
	}
	fmt.Printf("mk %s\n", version)
	return nil
}

func templateHelpHandler(cfg *cmdConfig) error {
	fmt.Println("mk tmpl help")
	fmt.Println()

	help := commandHelp()
	if err := printHelpSection("Template Commands", []string{"template-init", "template-new", "template-edit", "template-remove", "template-list", "template-verify", "template-help"}, help); err != nil {
		return err
	}

	fmt.Println("Template Syntax:")
	fmt.Println("  # dir               one relative path per line, creates directories")
	fmt.Println("  # file              one relative path per line, creates empty files")
	fmt.Println("  # content <path>    writes multiline file content into <path>")
	fmt.Println("  # cmd               one shell command per line, run in project root")
	fmt.Println()
	fmt.Println("Rules:")
	fmt.Println("  - Separate blocks with an empty line.")
	fmt.Println("  - Paths must stay inside the project root.")
	fmt.Println("  - {{PN}} is replaced with the project name in paths, content, and commands.")
	fmt.Println("  - {{MODULE}} is replaced with the resolved module path during template init.")
	fmt.Println("  - Empty # dir, # file, and # cmd blocks are allowed.")
	return nil
}

func configHelpHandler(cfg *cmdConfig) error {
	fmt.Println("mk config help")
	fmt.Println()

	help := commandHelp()
	if err := printHelpSection("Configuration Commands", []string{"config-edit", "config-setup", "config-help"}, help); err != nil {
		return err
	}

	fmt.Println("Configuration File:")
	fmt.Println("  Location: ~/.config/mk/config.json")
	fmt.Println()
	fmt.Println("Fields:")
	fmt.Println("  editor")
	fmt.Println("    Command used to open files in the editor.")
	fmt.Println("  module")
	fmt.Println(`    Default module path template. "{{.PN}}" is replaced with the project name.`)
	fmt.Println("  disableGoDefaultTemp")
	fmt.Println("    Prevents creation of the default go.mktmpl template during setup.")
	fmt.Println("  editOnTmplCreation")
	fmt.Println("    Opens a newly created template in the editor automatically.")
	return nil
}
