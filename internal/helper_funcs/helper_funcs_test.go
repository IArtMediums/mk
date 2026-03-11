package helperfuncs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWithinRoot(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "relative path", input: "cmd/app/main.go"},
		{name: "clean nested path", input: "cmd/../cmd/app"},
		{name: "escape root", input: "../outside", wantErr: true},
		{name: "absolute path", input: filepath.Join(root, "abs"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveWithinRoot(root, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil with path %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if filepath.Dir(got) == "" {
				t.Fatalf("expected resolved path, got %q", got)
			}
		})
	}
}

func TestCreateFileDoesNotOverwriteExistingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "README")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}

	if err := CreateFile(path); err == nil {
		t.Fatal("expected CreateFile to fail when file already exists")
	}
}

func TestCreateFileWithForceOverwritesExistingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "README")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}

	if err := CreateFileWithMode(path, true); err != nil {
		t.Fatalf("CreateFileWithMode() error = %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(body) != "" {
		t.Fatalf("expected truncated file, got %q", string(body))
	}
}

func TestPathLooksLikeDir(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "dir/", want: true},
		{path: "dir/file", want: false},
		{path: "dir/file.go", want: false},
	}

	for _, tt := range tests {
		if got := PathLooksLikeDir(tt.path); got != tt.want {
			t.Fatalf("PathLooksLikeDir(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
