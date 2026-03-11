package templateparser

import (
	"os"
	"path/filepath"
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
				t.Helper()

				block := tpl.Blocks[0]
				if block.Kind != BlockDir {
					t.Fatalf("expected dir block, got %v", block.Kind)
				}

				if len(block.Dirs) != 3 {
					t.Fatalf("expected 3 dirs, got %d", len(block.Dirs))
				}

				if block.Dirs[0] != DirPath("internal") {
					t.Fatalf("expected first dir to be %q, got %q", "internal", block.Dirs[0])
				}
			},
		},
		{
			name: "parses single command block",
			content: `# cmd
go mod init github.com/iartmediums/test
go fmt ./...
`,
			wantBlocks: 1,
			checkResult: func(t *testing.T, tpl *Template) {
				t.Helper()

				block := tpl.Blocks[0]
				if block.Kind != BlockCommand {
					t.Fatalf("expected command block, got %v", block.Kind)
				}

				if len(block.Cmds) != 2 {
					t.Fatalf("expected 2 commands, got %d", len(block.Cmds))
				}

				if block.Cmds[0].Raw != "go mod init github.com/iartmediums/test" {
					t.Fatalf("unexpected raw command: %q", block.Cmds[0].Raw)
				}
			},
		},
		{
			name: "parses file block",
			content: `# file
README
cmd/app/main.go
`,
			wantBlocks: 1,
			checkResult: func(t *testing.T, tpl *Template) {
				t.Helper()

				block := tpl.Blocks[0]
				if block.Kind != BlockFile {
					t.Fatalf("expected file block, got %v", block.Kind)
				}

				if len(block.Files) != 2 {
					t.Fatalf("expected 2 files, got %d", len(block.Files))
				}
			},
		},
		{
			name: "parses multiple blocks",
			content: `# dir
internal
pkg

# cmd
go test ./...
`,
			wantBlocks: 2,
			checkResult: func(t *testing.T, tpl *Template) {
				t.Helper()

				if tpl.Blocks[0].Kind != BlockDir {
					t.Fatalf("expected first block to be dir block")
				}

				if tpl.Blocks[1].Kind != BlockCommand {
					t.Fatalf("expected second block to be command block")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempTemplate(t, tt.content)

			tpl, err := ParseTemplate(path)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tpl == nil {
				t.Fatal("expected template, got nil")
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

# cmd
pwd > cwd.txt
printf "{{PN}}" > project_name.txt

# file
README
`)

	tpl, err := ParseTemplate(templatePath)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	parent := t.TempDir()
	projectRoot := filepath.Join(parent, "demo-project")

	if err := tpl.Execute("demo-project", projectRoot); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	gotCwdBytes, err := os.ReadFile(filepath.Join(projectRoot, "cwd.txt"))
	if err != nil {
		t.Fatalf("failed to read cwd marker: %v", err)
	}

	gotCwd := string(gotCwdBytes)
	if gotCwd != projectRoot+"\n" && gotCwd != projectRoot {
		t.Fatalf("expected command to run in %q, got %q", projectRoot, gotCwd)
	}

	projectNameBytes, err := os.ReadFile(filepath.Join(projectRoot, "project_name.txt"))
	if err != nil {
		t.Fatalf("failed to read project name marker: %v", err)
	}
	if string(projectNameBytes) != "demo-project" {
		t.Fatalf("expected project name substitution, got %q", string(projectNameBytes))
	}

	if _, err := os.Stat(filepath.Join(projectRoot, "internal")); err != nil {
		t.Fatalf("expected internal dir to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "README")); err != nil {
		t.Fatalf("expected README file to exist: %v", err)
	}
}

func TestExecuteRejectsPathsEscapingProjectRoot(t *testing.T) {
	templatePath := writeTempTemplate(t, `# file
../outside.txt
`)

	tpl, err := ParseTemplate(templatePath)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	projectRoot := filepath.Join(t.TempDir(), "demo-project")
	if err := tpl.Execute("demo-project", projectRoot); err == nil {
		t.Fatal("expected Execute() to reject root-escaping path")
	}
}

func TestExecuteFailsClearlyWhenFileAlreadyExists(t *testing.T) {
	templatePath := writeTempTemplate(t, `# file
README
`)

	tpl, err := ParseTemplate(templatePath)
	if err != nil {
		t.Fatalf("ParseTemplate() error = %v", err)
	}

	projectRoot := filepath.Join(t.TempDir(), "demo-project")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("failed to create project root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "README"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("failed to seed README: %v", err)
	}

	if err := tpl.Execute("demo-project", projectRoot); err == nil {
		t.Fatal("expected Execute() to fail when file already exists")
	}
}
