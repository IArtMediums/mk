package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	prevOut := os.Stdout
	prevErr := os.Stderr
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() stdout error = %v", err)
	}
	errReader, errWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe() stderr error = %v", err)
	}

	os.Stdout = outWriter
	os.Stderr = errWriter
	defer func() {
		os.Stdout = prevOut
		os.Stderr = prevErr
	}()

	fn()

	_ = outWriter.Close()
	_ = errWriter.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	_, _ = io.Copy(&stdout, outReader)
	_, _ = io.Copy(&stderr, errReader)
	return stdout.String(), stderr.String()
}

func prepareIsolatedHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".config", "mk")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, firstRunNoticeName), []byte("shown\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return home
}

func chdirTemp(t *testing.T) string {
	t.Helper()

	workdir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	if err := os.Chdir(workdir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	return workdir
}

func writeTemplate(t *testing.T, home, name, body string) {
	t.Helper()

	templatesDir := filepath.Join(home, ".config", "mk", "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, name+templateExt), []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func TestHandleCommandCreatesMultiplePaths(t *testing.T) {
	prepareIsolatedHome(t)
	workdir := chdirTemp(t)

	if err := HandleCommand([]string{"internal/server/server.go", "internal/config/", "cmd/app/main.go"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(workdir, "internal", "server", "server.go")); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	info, err := os.Stat(filepath.Join(workdir, "internal", "config"))
	if err != nil || !info.IsDir() {
		t.Fatalf("expected directory to exist, err=%v", err)
	}
}

func TestHandleCommandEditFlagOpensSingleCreatedFile(t *testing.T) {
	prepareIsolatedHome(t)
	workdir := chdirTemp(t)
	t.Setenv("EDITOR", "true")

	if err := HandleCommand([]string{"-e", "main.go"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(workdir, "main.go")); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestHandleCommandEditFlagRejectsMultiplePaths(t *testing.T) {
	prepareIsolatedHome(t)

	err := HandleCommand([]string{"-e", "main.go", "other.go"}, "test-version")
	if err == nil {
		t.Fatal("expected -e with multiple paths to fail")
	}
}

func TestHandleCommandEditFlagIgnoresDirectoryPath(t *testing.T) {
	prepareIsolatedHome(t)
	workdir := chdirTemp(t)

	if err := HandleCommand([]string{"-e", "internal/app/"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(workdir, "internal", "app"))
	if err != nil || !info.IsDir() {
		t.Fatalf("expected directory to exist, err=%v", err)
	}
}

func TestHandleCommandAllowsReservedNamesAsPathsWithDotSlash(t *testing.T) {
	prepareIsolatedHome(t)
	workdir := chdirTemp(t)

	if err := HandleCommand([]string{"./tmpl", "./config", "./help"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}

	for _, name := range []string{"tmpl", "config", "help"} {
		if _, err := os.Stat(filepath.Join(workdir, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}
}

func TestHandleCommandTmplInitFromManualSyntax(t *testing.T) {
	home := prepareIsolatedHome(t)
	workdir := chdirTemp(t)

	writeTemplate(t, home, "basic", `# dir
internal

# content README.md
# {{PN}}
`)

	stdout, stderr := captureOutput(t, func() {
		if err := HandleCommand([]string{"tmpl", "init", "basic", "demo"}, "test-version"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if stdout != "" || stderr != "" {
		t.Fatalf("expected quiet init, stdout=%q stderr=%q", stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(workdir, "demo", "README.md")); err != nil {
		t.Fatalf("expected template file to exist: %v", err)
	}
}

func TestHandleCommandTmplInitVerbose(t *testing.T) {
	home := prepareIsolatedHome(t)
	chdirTemp(t)

	writeTemplate(t, home, "basic", `# cmd
printf 'done'
`)

	stdout, _ := captureOutput(t, func() {
		if err := HandleCommand([]string{"tmpl", "init", "basic", "demo", "-v"}, "test-version"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if !strings.Contains(stdout, "Scaffolding demo from template basic") {
		t.Fatalf("expected verbose output, got %q", stdout)
	}
}

func TestHandleCommandTmplInitForce(t *testing.T) {
	home := prepareIsolatedHome(t)
	workdir := chdirTemp(t)

	writeTemplate(t, home, "basic", `# content README.md
new
`)

	projectRoot := filepath.Join(workdir, "demo")
	_ = os.MkdirAll(projectRoot, 0o755)
	_ = os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("old"), 0o644)

	if err := HandleCommand([]string{"tmpl", "init", "basic", "demo", "-f"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}

	body, err := os.ReadFile(filepath.Join(projectRoot, "README.md"))
	if err != nil || string(body) != "new" {
		t.Fatalf("expected forced overwrite, body=%q err=%v", string(body), err)
	}
}

func TestHandleCommandTmplInitUsesConfigModuleTemplate(t *testing.T) {
	home := prepareIsolatedHome(t)
	chdirTemp(t)

	configPath := filepath.Join(home, ".config", "mk", "config.json")
	configBody := "{\n  \"editor\": \"\",\n  \"module\": \"github.com/user/{{.PN}}\",\n  \"disableGoDefaultTemp\": false,\n  \"editOnTmplCreation\": true\n}\n"
	if err := os.WriteFile(configPath, []byte(configBody), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	writeTemplate(t, home, "basic", `# content go.mod
module {{MODULE}}
`)

	if err := HandleCommand([]string{"tmpl", "init", "basic", "demo"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}

	body, err := os.ReadFile(filepath.Join("demo", "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(body) != "module github.com/user/demo" {
		t.Fatalf("unexpected module content: %q", string(body))
	}
}

func TestHandleCommandTmplNew(t *testing.T) {
	home := prepareIsolatedHome(t)
	t.Setenv("EDITOR", "true")

	if err := HandleCommand([]string{"tmpl", "new", "test"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}

	path := filepath.Join(home, ".config", "mk", "templates", "test"+templateExt)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != templateSkeleton {
		t.Fatalf("unexpected template content:\n%s", string(content))
	}
}

func TestHandleCommandTmplEdit(t *testing.T) {
	home := prepareIsolatedHome(t)
	t.Setenv("EDITOR", "true")
	writeTemplate(t, home, "test", templateSkeleton)

	if err := HandleCommand([]string{"tmpl", "edit", "test"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}
}

func TestHandleCommandTmplRemove(t *testing.T) {
	home := prepareIsolatedHome(t)
	writeTemplate(t, home, "test", templateSkeleton)

	if err := HandleCommand([]string{"tmpl", "remove", "test"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".config", "mk", "templates", "test"+templateExt)); !os.IsNotExist(err) {
		t.Fatalf("expected template to be removed, stat err = %v", err)
	}
}

func TestHandleCommandTmplList(t *testing.T) {
	home := prepareIsolatedHome(t)
	writeTemplate(t, home, "zeta", "# dir\ninternal\n")
	writeTemplate(t, home, "alpha", "# dir\ninternal\n")

	stdout, _ := captureOutput(t, func() {
		if err := HandleCommand([]string{"tmpl", "list"}, "test-version"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if strings.TrimSpace(stdout) != "alpha\ngo\nzeta" {
		t.Fatalf("unexpected template list output: %q", stdout)
	}
}

func TestHandleCommandTmplVerify(t *testing.T) {
	home := prepareIsolatedHome(t)
	writeTemplate(t, home, "basic", `# content go.mod
module {{MODULE}}
`)

	stdout, _ := captureOutput(t, func() {
		if err := HandleCommand([]string{"tmpl", "verify", "basic"}, "test-version"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if !strings.Contains(stdout, `template "basic" verified successfully`) {
		t.Fatalf("unexpected verify output: %q", stdout)
	}
}

func TestHandleCommandTmplHelp(t *testing.T) {
	prepareIsolatedHome(t)

	stdout, _ := captureOutput(t, func() {
		if err := HandleCommand([]string{"tmpl", "help"}, "test-version"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if !strings.Contains(stdout, "Template Syntax:") {
		t.Fatalf("expected template syntax in help, got %q", stdout)
	}
}

func TestHandleCommandConfigEdit(t *testing.T) {
	home := prepareIsolatedHome(t)
	t.Setenv("EDITOR", "true")

	if err := HandleCommand([]string{"config", "edit"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(home, ".config", "mk", "config.json")); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
}

func TestHandleCommandConfigSetupCreatesMissingDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	stdout, stderr := captureOutput(t, func() {
		if err := HandleCommand([]string{"config", "setup"}, "test-version"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if !strings.Contains(stdout, "mk config setup:") {
		t.Fatalf("expected setup output, got %q", stdout)
	}
	if !strings.Contains(stderr, "Manual steps for full capability") {
		t.Fatalf("expected setup guidance, got %q", stderr)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "mk", "templates", "go"+templateExt)); err != nil {
		t.Fatalf("expected go template to exist: %v", err)
	}
}

func TestHandleCommandConfigSetupRestoresMissingConfigFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "mk")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, []byte("{\n  \"editor\": \"nvim\"\n}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := HandleCommand([]string{"config", "setup"}, "test-version"); err != nil {
		t.Fatalf("HandleCommand() error = %v", err)
	}

	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(body)
	if !strings.Contains(content, `"editor": "nvim"`) {
		t.Fatalf("expected existing editor value to remain, got %q", content)
	}
	if !strings.Contains(content, `"module": ""`) {
		t.Fatalf("expected missing module field to be restored, got %q", content)
	}
	if !strings.Contains(content, `"disableGoDefaultTemp": false`) {
		t.Fatalf("expected missing disableGoDefaultTemp field to be restored, got %q", content)
	}
	if !strings.Contains(content, `"editOnTmplCreation": true`) {
		t.Fatalf("expected missing editOnTmplCreation field to be restored, got %q", content)
	}
}

func TestHandleCommandConfigHelp(t *testing.T) {
	prepareIsolatedHome(t)

	stdout, _ := captureOutput(t, func() {
		if err := HandleCommand([]string{"config", "help"}, "test-version"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if !strings.Contains(stdout, "Configuration File:") {
		t.Fatalf("expected config help output, got %q", stdout)
	}
}

func TestHandleCommandHelp(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	chdirTemp(t)

	stdout, stderr := captureOutput(t, func() {
		if err := HandleCommand([]string{"help"}, "test-version"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if !strings.Contains(stdout, "Basic Usage:") {
		t.Fatalf("expected top-level help sections, got %q", stdout)
	}
	if !strings.Contains(stderr, "Manual steps for full capability") {
		t.Fatalf("expected first-run notice, got %q", stderr)
	}
}

func TestHandleCommandVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	stdout, stderr := captureOutput(t, func() {
		if err := HandleCommand([]string{"--version"}, "1.0.0"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.TrimSpace(stdout) != "mk 1.0.0" {
		t.Fatalf("unexpected version output: %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "mk")); !os.IsNotExist(err) {
		t.Fatalf("expected version command not to create config, stat err = %v", err)
	}
}

func TestHandleCommandVersionAlias(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	stdout, stderr := captureOutput(t, func() {
		if err := HandleCommand([]string{"version"}, "1.0.0"); err != nil {
			t.Fatalf("HandleCommand() error = %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if strings.TrimSpace(stdout) != "mk 1.0.0" {
		t.Fatalf("unexpected version output: %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "mk")); !os.IsNotExist(err) {
		t.Fatalf("expected version command not to create config, stat err = %v", err)
	}
}
