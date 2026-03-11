package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const moduleImportPath = "github.com/IArtMediums/mk"

var (
	lookupLatestVersion = fetchLatestModuleVersion
	installLatest       = installLatestModuleVersion
	removeLegacyBinary  = removeLegacyInstalledBinary
)

type latestModuleInfo struct {
	Version string `json:"Version"`
}

func updateHandler(currentVersion string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("mk update is not supported on Windows while the binary is running; use `go install %s@latest` manually", moduleImportPath)
	}

	latestVersion, err := lookupLatestVersion()
	if err != nil {
		return err
	}

	normalizedCurrent := normalizeVersion(currentVersion)
	normalizedLatest := normalizeVersion(latestVersion)

	if normalizedCurrent != "" && normalizedCurrent != "dev" {
		cmp, err := compareSemver(normalizedCurrent, normalizedLatest)
		if err != nil {
			return fmt.Errorf("unable to compare versions %q and %q: %w", normalizedCurrent, normalizedLatest, err)
		}
		if cmp >= 0 {
			fmt.Printf("mk %s is already up to date\n", normalizedCurrent)
			return nil
		}
		fmt.Printf("Updating mk from %s to %s\n", normalizedCurrent, normalizedLatest)
	} else {
		fmt.Printf("Current mk version is %q; installing latest release %s\n", displayVersion(currentVersion), normalizedLatest)
	}

	if err := installLatest(); err != nil {
		return err
	}

	removedLegacy, err := removeLegacyBinary()
	if err != nil {
		return err
	}

	fmt.Printf("Updated mk to %s\n", normalizedLatest)
	if removedLegacy {
		fmt.Println("Removed legacy mk-cli binary")
	}
	return nil
}

func fetchLatestModuleVersion() (string, error) {
	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("go is required for mk update: %w", err)
	}

	cmd := exec.Command("go", "list", "-m", "-json", moduleImportPath+"@latest")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("unable to check latest mk version: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	var info latestModuleInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return "", fmt.Errorf("unable to parse latest mk version: %w", err)
	}
	if info.Version == "" {
		return "", fmt.Errorf("go did not return a latest version for %s", moduleImportPath)
	}

	return info.Version, nil
}

func installLatestModuleVersion() error {
	cmd := exec.Command("go", "install", moduleImportPath+"@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to install latest mk release: %w", err)
	}
	return nil
}

func removeLegacyInstalledBinary() (bool, error) {
	legacyName := "mk-cli"
	if runtime.GOOS == "windows" {
		legacyName += ".exe"
	}

	paths, err := legacyBinaryCandidates(legacyName)
	if err != nil {
		return false, err
	}

	for _, candidate := range paths {
		info, err := os.Stat(candidate)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("unable to inspect legacy binary %s: %w", candidate, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := os.Remove(candidate); err != nil {
			return false, fmt.Errorf("unable to remove legacy binary %s: %w", candidate, err)
		}
		return true, nil
	}

	return false, nil
}

func legacyBinaryCandidates(legacyName string) ([]string, error) {
	seen := map[string]struct{}{}
	var candidates []string

	add := func(path string) {
		if path == "" {
			return
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			return
		}
		seen[clean] = struct{}{}
		candidates = append(candidates, clean)
	}

	if executablePath, err := os.Executable(); err == nil {
		add(filepath.Join(filepath.Dir(executablePath), legacyName))
	}

	if gobin := strings.TrimSpace(os.Getenv("GOBIN")); gobin != "" {
		add(filepath.Join(gobin, legacyName))
	}

	goBinFromEnv, err := goBinDir()
	if err != nil {
		return nil, err
	}
	add(filepath.Join(goBinFromEnv, legacyName))

	return candidates, nil
}

func goBinDir() (string, error) {
	if gobin := strings.TrimSpace(os.Getenv("GOBIN")); gobin != "" {
		return gobin, nil
	}

	if _, err := exec.LookPath("go"); err != nil {
		return "", fmt.Errorf("go is required to resolve install directory: %w", err)
	}

	cmd := exec.Command("go", "env", "GOBIN", "GOPATH")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("unable to resolve Go bin directory: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) != 2 {
		return "", fmt.Errorf("unexpected go env output while resolving Go bin directory")
	}

	gobin := strings.TrimSpace(lines[0])
	if gobin != "" {
		return gobin, nil
	}

	gopath := strings.TrimSpace(lines[1])
	if gopath == "" {
		return "", fmt.Errorf("Go did not report GOPATH")
	}

	return filepath.Join(gopath, "bin"), nil
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" || version == "(devel)" {
		return "dev"
	}
	if version == "dev" {
		return version
	}
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func displayVersion(version string) string {
	normalized := normalizeVersion(version)
	if normalized == "" {
		return "dev"
	}
	return normalized
}

func compareSemver(current, latest string) (int, error) {
	currentParts, err := parseSemver(current)
	if err != nil {
		return 0, err
	}
	latestParts, err := parseSemver(latest)
	if err != nil {
		return 0, err
	}

	for i := 0; i < len(currentParts); i++ {
		switch {
		case currentParts[i] < latestParts[i]:
			return -1, nil
		case currentParts[i] > latestParts[i]:
			return 1, nil
		}
	}

	return 0, nil
}

func parseSemver(version string) ([3]int, error) {
	var parts [3]int

	if !strings.HasPrefix(version, "v") {
		return parts, fmt.Errorf("version must start with v")
	}

	core := strings.TrimPrefix(version, "v")
	core = strings.SplitN(core, "-", 2)[0]
	core = strings.SplitN(core, "+", 2)[0]

	segments := strings.Split(core, ".")
	if len(segments) != 3 {
		return parts, fmt.Errorf("version must have major, minor, and patch components")
	}

	for i, segment := range segments {
		value, err := strconv.Atoi(segment)
		if err != nil {
			return parts, fmt.Errorf("invalid semantic version component %q", segment)
		}
		parts[i] = value
	}

	return parts, nil
}
