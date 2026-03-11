package templateparser

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempTemplate(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.mktemp")

	err := os.WriteFile(path, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to write temp template: %v", err)
	}

	return path
}

func TestParseTemplate(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		wantBlocks  int
		checkResult func(t *testing.T, tpl *Template)
	}{
		{
			name: "parses single dir block",
			content: `# dir
internal
internal/config
cmd/app
`,
			wantBlocks: 1,
			checkResult: func(t *testing.T, tpl *Template) {
				block := tpl.Blocks[0]
				if block.Kind != BlockDir || len(block.Dirs) != 3 {
					t.Fatalf("unexpected dir block: %+v", block)
				}
			},
		},
		{
			name: "parses content block with blank lines",
			content: `# content cmd/{{PN}}/main.go
package main

import "fmt"
`,
			wantBlocks: 1,
			checkResult: func(t *testing.T, tpl *Template) {
				block := tpl.Blocks[0]
				if block.Kind != BlockContent || len(block.Contents) != 1 {
					t.Fatalf("unexpected content block: %+v", block)
				}
			},
		},
		{
			name: "returns error for mixed block",
			content: `# dir
internal
# cmd
go test ./...
`,
			wantErr: true,
		},
		{
			name: "returns error for content without path",
			content: `# content
hello
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl, err := ParseTemplate(writeTempTemplate(t, tt.content))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tpl.Blocks) != tt.wantBlocks {
				t.Fatalf("expected %d blocks, got %d", tt.wantBlocks, len(tpl.Blocks))
			}
			if tt.checkResult != nil {
				tt.checkResult(t, tpl)
			}
		})
	}
}

func TestExecuteRunsCommandsInProjectRoot(t *testing.T) {
	templatePath := writeTempTemplate(t, `# dir
internal

# content cmd/{{PN}}/main.go
package main

func main() {}

# file
README

# cmd
pwd > cwd.txt
printf "{{PN}}" > project_name.txt
`)

	tpl, err := ParseTemplate(templatePath)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	projectRoot := filepath.Join(t.TempDir(), "demo-project")
	if err := tpl.Execute(ExecuteOptions{
		ProjectName: "demo-project",
		ProjectRoot: projectRoot,
		ModulePath:  "example.com/demo-project",
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	gotCwdBytes, err := os.ReadFile(filepath.Join(projectRoot, "cwd.txt"))
	if err != nil {
		t.Fatalf("failed to read cwd marker: %v", err)
	}
	gotCwd := strings.TrimSpace(string(gotCwdBytes))
	if gotCwd != projectRoot {
		t.Fatalf("expected command cwd %q, got %q", projectRoot, gotCwd)
	}
}

func TestExecuteWritesTemplateVars(t *testing.T) {
	templatePath := writeTempTemplate(t, `# content go.mod
module {{MODULE}}

# content cmd/{{PN}}/main.go
package main
`)

	tpl, err := ParseTemplate(templatePath)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	projectRoot := filepath.Join(t.TempDir(), "demo-project")
	if err := tpl.Execute(ExecuteOptions{
		ProjectName: "demo-project",
		ProjectRoot: projectRoot,
		ModulePath:  "example.com/demo-project",
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	goMod, err := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(goMod) != "module example.com/demo-project\n" {
		t.Fatalf("unexpected module file: %q", string(goMod))
	}
}

func TestExecuteDryRunOnlyPrintsPlan(t *testing.T) {
	templatePath := writeTempTemplate(t, `# dir
internal

# content go.mod
module {{MODULE}}

# cmd
echo ok
`)

	tpl, err := ParseTemplate(templatePath)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	projectRoot := filepath.Join(t.TempDir(), "demo-project")
	var out bytes.Buffer
	if err := tpl.Execute(ExecuteOptions{
		ProjectName: "demo-project",
		ProjectRoot: projectRoot,
		ModulePath:  "example.com/demo-project",
		DryRun:      true,
		Stdout:      &out,
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, err := os.Stat(projectRoot); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run not to create project root, stat err = %v", err)
	}
	if !strings.Contains(out.String(), "run    echo ok") {
		t.Fatalf("expected dry-run output, got %q", out.String())
	}
}

func TestExecuteRejectsPathsEscapingProjectRoot(t *testing.T) {
	tpl, err := ParseTemplate(writeTempTemplate(t, `# content ../outside.txt
blocked
`))
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	err = tpl.Execute(ExecuteOptions{
		ProjectName: "demo-project",
		ProjectRoot: filepath.Join(t.TempDir(), "demo-project"),
	})
	if err == nil {
		t.Fatal("expected Execute() to reject root-escaping path")
	}
}

func TestExecuteFailsClearlyWhenFileAlreadyExists(t *testing.T) {
	tpl, err := ParseTemplate(writeTempTemplate(t, `# content README
new
`))
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	projectRoot := filepath.Join(t.TempDir(), "demo-project")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "README"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err = tpl.Execute(ExecuteOptions{
		ProjectName: "demo-project",
		ProjectRoot: projectRoot,
	})
	if err == nil {
		t.Fatal("expected Execute() to fail when file already exists")
	}
}

func TestExecuteForceOverwritesExistingFile(t *testing.T) {
	tpl, err := ParseTemplate(writeTempTemplate(t, `# content README
new
`))
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	projectRoot := filepath.Join(t.TempDir(), "demo-project")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "README"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := tpl.Execute(ExecuteOptions{
		ProjectName: "demo-project",
		ProjectRoot: projectRoot,
		Force:       true,
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body, err := os.ReadFile(filepath.Join(projectRoot, "README"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(body) != "new" {
		t.Fatalf("expected force overwrite, got %q", string(body))
	}
}
