package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
		wantErr bool
	}{
		{name: "older current", current: "v1.0.0", latest: "v1.0.1", want: -1},
		{name: "same version", current: "v1.0.1", latest: "v1.0.1", want: 0},
		{name: "newer current", current: "v1.2.0", latest: "v1.1.9", want: 1},
		{name: "prerelease ignored for ordering", current: "v1.0.0-beta.1", latest: "v1.0.0", want: 0},
		{name: "invalid current", current: "dev", latest: "v1.0.0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := compareSemver(tt.current, tt.latest)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("compareSemver() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("compareSemver() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: "dev"},
		{input: "(devel)", want: "dev"},
		{input: "dev", want: "dev"},
		{input: "1.0.0", want: "v1.0.0"},
		{input: "v1.0.1", want: "v1.0.1"},
	}

	for _, tt := range tests {
		if got := normalizeVersion(tt.input); got != tt.want {
			t.Fatalf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHandleCommandUpdateSkipsSetupWhenAlreadyCurrent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	prevLookup := lookupLatestVersion
	prevInstall := installLatest
	prevRemove := removeLegacyBinary
	t.Cleanup(func() {
		lookupLatestVersion = prevLookup
		installLatest = prevInstall
		removeLegacyBinary = prevRemove
	})

	lookupLatestVersion = func() (string, error) {
		return "v1.0.1", nil
	}
	installLatest = func() error {
		t.Fatal("installLatest should not be called when already current")
		return nil
	}
	removeLegacyBinary = func() (bool, error) {
		t.Fatal("removeLegacyBinary should not be called when already current")
		return false, nil
	}

	stdout, stderr := captureOutput(t, func() {
		if err := HandleCommand([]string{"update"}, "v1.0.1"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "already up to date") {
		t.Fatalf("unexpected update output: %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "mk")); !os.IsNotExist(err) {
		t.Fatalf("expected update command not to create config, stat err = %v", err)
	}
}

func TestHandleCommandUpdateInstallsLatestForDevBuild(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	prevLookup := lookupLatestVersion
	prevInstall := installLatest
	prevRemove := removeLegacyBinary
	t.Cleanup(func() {
		lookupLatestVersion = prevLookup
		installLatest = prevInstall
		removeLegacyBinary = prevRemove
	})

	lookupLatestVersion = func() (string, error) {
		return "v1.0.2", nil
	}

	installed := false
	installLatest = func() error {
		installed = true
		return nil
	}
	removeLegacyBinary = func() (bool, error) {
		return false, nil
	}

	stdout, stderr := captureOutput(t, func() {
		if err := HandleCommand([]string{"update"}, "dev"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !installed {
		t.Fatal("expected installLatest to be called")
	}
	if !strings.Contains(stdout, "Updated mk to v1.0.2") {
		t.Fatalf("unexpected update output: %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "mk")); !os.IsNotExist(err) {
		t.Fatalf("expected update command not to create config, stat err = %v", err)
	}
}

func TestHandleCommandUpdateRemovesLegacyBinary(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	prevLookup := lookupLatestVersion
	prevInstall := installLatest
	prevRemove := removeLegacyBinary
	t.Cleanup(func() {
		lookupLatestVersion = prevLookup
		installLatest = prevInstall
		removeLegacyBinary = prevRemove
	})

	lookupLatestVersion = func() (string, error) {
		return "v1.0.3", nil
	}
	installLatest = func() error {
		return nil
	}
	removeLegacyBinary = func() (bool, error) {
		return true, nil
	}

	stdout, stderr := captureOutput(t, func() {
		if err := HandleCommand([]string{"update"}, "v1.0.2"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "Removed legacy mk-cli binary") {
		t.Fatalf("expected legacy cleanup message, got %q", stdout)
	}
}

func TestRemoveLegacyInstalledBinary(t *testing.T) {
	binDir := t.TempDir()
	t.Setenv("GOBIN", binDir)

	legacyPath := filepath.Join(binDir, "mk-cli")
	if err := os.WriteFile(legacyPath, []byte("legacy"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	removed, err := removeLegacyInstalledBinary()
	if err != nil {
		t.Fatalf("removeLegacyInstalledBinary() error = %v", err)
	}
	if !removed {
		t.Fatal("expected legacy binary to be removed")
	}
	if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy binary to be removed, stat err = %v", err)
	}
}
