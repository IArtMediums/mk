package commands

import (
	"fmt"
	"path/filepath"
	"strings"
)

type cmdConfig struct {
	args       []string
	verbose    bool
	force      bool
	modulePath string
	edit       bool
	config     mkConfig
}

type CommandHelp struct {
	Usage       string
	Description string
}

func HandleCommand(args []string, version string) error {
	if len(args) == 1 && (args[0] == "version" || args[0] == "--version") {
		return versionHandler(version)
	}

	printNotice := true
	if len(args) > 1 && args[0] == "config" && args[1] == "setup" {
		printNotice = false
	}

	result, err := ensureSetup(printNotice)
	if err != nil {
		return err
	}

	cfg := &cmdConfig{
		args:   args,
		config: result.Config,
	}
	return cfg.handleCommand(version)
}

func (cfg *cmdConfig) handleCommand(version string) error {
	if len(cfg.args) == 0 {
		return helpHandler(cfg)
	}

	switch cfg.args[0] {
	case "help":
		if len(cfg.args) != 1 {
			return fmt.Errorf("usage: mk help")
		}
		return helpHandler(cfg)
	case "version", "--version":
		if len(cfg.args) != 1 {
			return fmt.Errorf("usage: mk --version")
		}
		return versionHandler(version)
	case "tmpl":
		return cfg.handleTemplateCommand()
	case "config":
		return cfg.handleConfigCommand()
	}

	if cfg.args[0] == "-e" {
		cfg.edit = true
		cfg.args = cfg.args[1:]
		if len(cfg.args) != 1 {
			return fmt.Errorf("usage: mk -e <path>")
		}
	}

	return pathHandler(cfg)
}

func (cfg *cmdConfig) handleTemplateCommand() error {
	if len(cfg.args) < 2 {
		return fmt.Errorf("usage: mk tmpl <init|new|edit|remove|list|verify|help>")
	}

	switch cfg.args[1] {
	case "init":
		if len(cfg.args) < 4 {
			return fmt.Errorf("usage: mk tmpl init <template_name> <project_name> [-v] [-f] [-m <module_path>]")
		}
		cfg.args = cfg.args[2:]
		if err := cfg.parseInitFlags(cfg.args[2:]); err != nil {
			return err
		}
		cfg.args = cfg.args[:2]
		return initHandler(cfg)
	case "new":
		if len(cfg.args) != 3 {
			return fmt.Errorf("usage: mk tmpl new <name>")
		}
		return createTemplateHandler(cfg)
	case "edit":
		if len(cfg.args) != 3 {
			return fmt.Errorf("usage: mk tmpl edit <name>")
		}
		return templateEditHandler(cfg)
	case "remove":
		if len(cfg.args) != 3 {
			return fmt.Errorf("usage: mk tmpl remove <name>")
		}
		return templateRemoveHandler(cfg)
	case "list":
		if len(cfg.args) != 2 {
			return fmt.Errorf("usage: mk tmpl list")
		}
		return templateListHandler(cfg)
	case "verify":
		if len(cfg.args) != 3 {
			return fmt.Errorf("usage: mk tmpl verify <name>")
		}
		return templateVerifyHandler(cfg)
	case "help":
		if len(cfg.args) != 2 {
			return fmt.Errorf("usage: mk tmpl help")
		}
		return templateHelpHandler(cfg)
	default:
		return fmt.Errorf("usage: mk tmpl <init|new|edit|remove|list|verify|help>")
	}
}

func (cfg *cmdConfig) handleConfigCommand() error {
	if len(cfg.args) < 2 {
		return fmt.Errorf("usage: mk config <edit|setup|help>")
	}

	switch cfg.args[1] {
	case "edit":
		if len(cfg.args) != 2 {
			return fmt.Errorf("usage: mk config edit")
		}
		return configEditHandler(cfg)
	case "setup":
		if len(cfg.args) != 2 {
			return fmt.Errorf("usage: mk config setup")
		}
		return configSetupHandler(cfg)
	case "help":
		if len(cfg.args) != 2 {
			return fmt.Errorf("usage: mk config help")
		}
		return configHelpHandler(cfg)
	default:
		return fmt.Errorf("usage: mk config <edit|setup|help>")
	}
}

func (cfg *cmdConfig) parseInitFlags(flags []string) error {
	for i := 0; i < len(flags); i++ {
		switch flags[i] {
		case "-v":
			cfg.verbose = true
		case "-f":
			cfg.force = true
		case "-m":
			if i+1 >= len(flags) {
				return fmt.Errorf("usage: mk tmpl init <template_name> <project_name> [-v] [-f] [-m <module_path>]")
			}
			i++
			cfg.modulePath = strings.TrimSpace(flags[i])
			if cfg.modulePath == "" {
				return fmt.Errorf("module path cannot be empty")
			}
		default:
			return fmt.Errorf("unknown argument %q. Use `mk help` to see available commands", flags[i])
		}
	}
	return nil
}

func getTemplatePath(templateName string) (string, error) {
	if templateName == "" {
		return "", fmt.Errorf("template name cannot be empty")
	}
	if filepath.Base(templateName) != templateName {
		return "", fmt.Errorf("template name must not contain path separators")
	}

	templatesDir, err := getTemplatesDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(templatesDir, templateName+templateExt), nil
}

func commandHelp() map[string]CommandHelp {
	return map[string]CommandHelp{
		"path": {
			Usage:       "mk <path1> <path2> ...",
			Description: "Create one or more smart paths. A trailing slash creates a directory; otherwise mk creates a file.",
		},
		"path-edit": {
			Usage:       "mk -e <path>",
			Description: "Create a single smart path and open it in the configured editor only if the path resolves to a file.",
		},
		"template-init": {
			Usage:       "mk tmpl init <template_name> <project_name> [-v] [-f] [-m <module_path>]",
			Description: "Initialize <project_name> from a template stored under ~/.config/mk/templates.",
		},
		"template-new": {
			Usage:       "mk tmpl new <name>",
			Description: "Create a new template file named ~/.config/mk/templates/<name>.mktmpl.",
		},
		"template-edit": {
			Usage:       "mk tmpl edit <name>",
			Description: "Open an existing template file in the configured editor.",
		},
		"template-remove": {
			Usage:       "mk tmpl remove <name>",
			Description: "Delete an existing template file from ~/.config/mk/templates.",
		},
		"template-list": {
			Usage:       "mk tmpl list",
			Description: "Display all available template names.",
		},
		"template-verify": {
			Usage:       "mk tmpl verify <name>",
			Description: "Validate a template using dry-run execution without creating files.",
		},
		"template-help": {
			Usage:       "mk tmpl help",
			Description: "Show template command usage and template file syntax.",
		},
		"config-edit": {
			Usage:       "mk config edit",
			Description: "Open ~/.config/mk/config.json in the configured editor.",
		},
		"config-setup": {
			Usage:       "mk config setup",
			Description: "Create missing config files and restore default templates under ~/.config/mk.",
		},
		"config-help": {
			Usage:       "mk config help",
			Description: "Show configuration commands and explain the config.json fields.",
		},
		"help": {
			Usage:       "mk help",
			Description: "Display the mk usage guide grouped by purpose.",
		},
		"version": {
			Usage:       "mk --version",
			Description: "Display the installed mk version.",
		},
	}
}
