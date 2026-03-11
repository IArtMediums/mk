package helperfuncs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveWithinRoot(root, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	root = filepath.Clean(root)
	clean := filepath.Clean(input)
	if clean == "." {
		return "", fmt.Errorf("path cannot resolve to current directory")
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("absolute paths are not allowed: %s", input)
	}

	fullPath := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, fullPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root: %s", input)
	}

	return fullPath, nil
}

func CreateDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func CreateFile(path string) error {
	return CreateFileWithMode(path, false)
}

func CreateFileWithMode(path string, force bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	flags := os.O_CREATE | os.O_WRONLY
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}

	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	return file.Close()
}

func WriteFile(path, body string) error {
	return WriteFileWithMode(path, body, false)
}

func WriteFileWithMode(path, body string, force bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	flags := os.O_CREATE | os.O_WRONLY
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}

	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(body); err != nil {
		return err
	}
	return nil
}

func CreatePath(path string) error {
	if PathLooksLikeDir(path) {
		return CreateDir(path)
	}

	return CreateFile(path)
}

func PathLooksLikeDir(path string) bool {
	return strings.HasSuffix(path, "/") || strings.HasSuffix(path, string(filepath.Separator))
}
