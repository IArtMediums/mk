package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	helperfuncs "github.com/iartmediums/mk-cli/internal/helper_funcs"
)

func editHandler(cfg *cmdConfig) error {
	rawPath := cfg.args[1]
	if helperfuncs.PathLooksLikeDir(rawPath) {
		return fmt.Errorf("edit requires a file path, got directory path %q", rawPath)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error while getting current directory: %w", err)
	}

	fullPath, err := helperfuncs.ResolveWithinRoot(cwd, rawPath)
	if err != nil {
		return fmt.Errorf("invalid path %q: %w", rawPath, err)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		if err := helperfuncs.CreateFile(fullPath); err != nil {
			return fmt.Errorf("error while creating file: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error while checking file: %w", err)
	}

	if err := openInEditor(fullPath, cfg); err != nil {
		return err
	}

	return nil
}

func resolveEditor(cfg *cmdConfig) (string, error) {
	editor := ""
	if cfg != nil {
		editor = strings.TrimSpace(cfg.config.Editor)
	}
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("VISUAL"))
	}
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor != "" {
		return editor, nil
	}

	for _, candidate := range []string{"nvim", "vim", "nano", "vi"} {
		if _, err := exec.LookPath(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("set $VISUAL or $EDITOR to use `mk edit`")
}

func openInEditor(path string, cfg *cmdConfig) error {
	editor, err := resolveEditor(cfg)
	if err != nil {
		return err
	}

	cmd := exec.Command("sh", "-c", editor+" \"$1\"", "mk-edit", path)
	cmd.Dir = filepath.Dir(path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error while opening editor: %w", err)
	}

	return nil
}
