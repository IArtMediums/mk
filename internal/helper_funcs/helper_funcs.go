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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return file.Close()
}

func CreatePath(path string) error {
	dirsToCreate := filepath.Dir(path)

	err := os.MkdirAll(dirsToCreate, 0o755)
	if err != nil {
		return err
	}

	lastPathItem, _ := strings.CutPrefix(path, dirsToCreate)

	ext := filepath.Ext(lastPathItem)

	if ext == "" {
		return CreateDir(path)
	}
	return CreateFile(path)
}
