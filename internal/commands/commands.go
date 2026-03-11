// Package commands handles cli commands
package commands

import (
	"fmt"
	"strings"
)

type cmdConfig struct {
	cmdString string
	args      []string
	cmdMap    commandMap
}

type Command struct {
	Name          string
	AltName       string
	Description   string
	Usage         string
	OptionalFlags []string
	NumOfArgs     int
	HelpIgnore    bool
	Handler       func(*cmdConfig) error
}

type commandMap map[string]Command

func HandleCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("expected at least 1 argument, received None")
	}
	cfg := NewConfig(args)

	err := cfg.handleCommand()
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}

func NewConfig(args []string) *cmdConfig {
	var cfg cmdConfig

	cfg.cmdString = args[0]

	if strings.HasPrefix(cfg.cmdString, "-") {
		if len(args) > 1 {
			cfg.args = args[1:]
		} else {
			cfg.args = []string{}
		}
	} else {
		cfg.args = []string{cfg.cmdString}
		cfg.cmdString = "--path"
	}

	cfg.cmdMap = commandMap{}

	cfg.addCommand("--help", "-h", "Show available commands", "mk --help", 0, helpHandler)
	cfg.addCommand("--path", "-p", "Creates all directories and/or file. In order to create file extention must be provided. -w flag will write to file as well", "mk --path <path>", 1, mkBasicHandler)
	cfg.addCommand("--init", "", "Creates project template from <template_name>.mktemp in ~/.confing/mk/templates using <project_name>", "mk --init <tpl_name> <proj_name>", 2, initHandler)

	return &cfg
}

func (cfg *cmdConfig) addCommand(name, altName, description, usage string, numofArgs int, handler func(*cmdConfig) error) {
	cmd := Command{
		Name:          name,
		AltName:       altName,
		Description:   description,
		Usage:         usage,
		NumOfArgs:     numofArgs,
		OptionalFlags: []string{},
		Handler:       handler,
		HelpIgnore:    false,
	}
	cfg.cmdMap[name] = cmd

	if altName != "" {
		cmd.HelpIgnore = true
		cmd.Name = altName
		cmd.AltName = name
		cfg.cmdMap[altName] = cmd
	}
}

func (cfg *cmdConfig) handleCommand() error {
	cmd, ok := cfg.cmdMap[cfg.cmdString]

	if !ok && strings.HasPrefix(cfg.cmdString, "-") {
		return fmt.Errorf("'%s' is invalid command. Use mk -h / mk --help to see the usage", cfg.cmdString)
	}
	if !ok {
		err := mkBasicHandler(cfg)
		return err
	}

	if cmd.NumOfArgs != len(cfg.args) {
		fmt.Printf("Wrong number of arguments, expected: %d, received %d\n", cmd.NumOfArgs, len(cfg.args))
		return fmt.Errorf("%v", cmd.getHelpString())
	}

	err := cmd.Handler(cfg)
	if err != nil {
		return fmt.Errorf("error while handling command: %v", err)
	}

	return nil
}

func (cfg *cmdConfig) getVerboseHelpTable() string {
	return "NAME						USAGE				DESCRIPTION\n"
}

func (cmd Command) getHelpString() string {
	helpString := fmt.Sprintf("%s					%s			%s\n", cmd.Name, cmd.Usage, cmd.Description)
	if cmd.AltName != "" {
		helpString = fmt.Sprintf("%s, %s					%s			%s\n", cmd.Name, cmd.AltName, cmd.Usage, cmd.Description)
	}
	return helpString
}
