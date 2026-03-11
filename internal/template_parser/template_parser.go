package templateparser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	helperfuncs "github.com/IArtMediums/mk/internal/helper_funcs"
)

const (
	dirDeclaration     = "# dir"
	fileDeclaration    = "# file"
	cmdDeclaration     = "# cmd"
	contentHeader      = "# content"
	contentDeclaration = "# content "
	projectNameVar     = "{{PN}}"
	modulePathVar      = "{{MODULE}}"
)

type parseTempState int

const (
	noneState parseTempState = iota
	parseCmd
	parseDir
	parseFile
	parseContent
)

type DirPath string
type FilePath string

type Command struct {
	Raw string
}

type ContentFile struct {
	Path string
	Body string
}

type BlockKind int

const (
	BlockUnknown BlockKind = iota
	BlockCommand
	BlockDir
	BlockFile
	BlockContent
)

type Block struct {
	Kind     BlockKind
	Dirs     []DirPath
	Files    []FilePath
	Cmds     []Command
	Contents []ContentFile
}

type Template struct {
	Blocks []Block
}

type ExecuteOptions struct {
	ProjectName string
	ProjectRoot string
	ModulePath  string
	Verbose     bool
	DryRun      bool
	Force       bool
	Stdout      io.Writer
}

func ParseTemplate(tempPath string) (*Template, error) {
	file, err := os.Open(tempPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var currentBlock Block
	var template Template
	var contentPath string
	var contentLines []string

	state := noneState
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if state == parseContent {
			if isBlockHeader(line) {
				if err := finalizeContentBlock(&template, &currentBlock, contentPath, contentLines); err != nil {
					return nil, err
				}
				currentBlock = Block{}
				contentPath = ""
				contentLines = nil
				state = noneState
			} else {
				contentLines = append(contentLines, line)
				continue
			}
		}

		switch {
		case line == dirDeclaration:
			if hasBlockEntries(currentBlock) {
				return nil, fmt.Errorf("blocks must be separated by empty line")
			}
			if err := template.addBlock(currentBlock); err != nil {
				return nil, err
			}
			currentBlock = Block{}
			state = parseDir
			continue
		case line == fileDeclaration:
			if hasBlockEntries(currentBlock) {
				return nil, fmt.Errorf("blocks must be separated by empty line")
			}
			if err := template.addBlock(currentBlock); err != nil {
				return nil, err
			}
			currentBlock = Block{}
			state = parseFile
			continue
		case line == cmdDeclaration:
			if hasBlockEntries(currentBlock) {
				return nil, fmt.Errorf("blocks must be separated by empty line")
			}
			if err := template.addBlock(currentBlock); err != nil {
				return nil, err
			}
			currentBlock = Block{}
			state = parseCmd
			continue
		case line == contentHeader:
			return nil, fmt.Errorf("content block requires a target path")
		case strings.HasPrefix(line, contentDeclaration):
			if hasBlockEntries(currentBlock) {
				return nil, fmt.Errorf("blocks must be separated by empty line")
			}
			if err := template.addBlock(currentBlock); err != nil {
				return nil, err
			}
			currentBlock = Block{Kind: BlockContent}
			contentPath = strings.TrimSpace(strings.TrimPrefix(line, contentDeclaration))
			if contentPath == "" {
				return nil, fmt.Errorf("content block requires a target path")
			}
			contentLines = nil
			state = parseContent
			continue
		case line == "":
			if err := template.addBlock(currentBlock); err != nil {
				return nil, err
			}
			currentBlock = Block{}
			state = noneState
			continue
		}

		switch state {
		case parseDir:
			currentBlock.addDir(line)
		case parseFile:
			currentBlock.addFile(line)
		case parseCmd:
			currentBlock.addCmd(line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if state == parseContent {
		if err := finalizeContentBlock(&template, &currentBlock, contentPath, contentLines); err != nil {
			return nil, err
		}
		return &template, nil
	}

	if err := template.addBlock(currentBlock); err != nil {
		return nil, err
	}

	return &template, nil
}

func hasBlockEntries(block Block) bool {
	return len(block.Cmds) > 0 || len(block.Dirs) > 0 || len(block.Files) > 0 || len(block.Contents) > 0
}

func finalizeContentBlock(template *Template, block *Block, path string, lines []string) error {
	block.addContent(path, strings.Join(lines, "\n"))
	return template.addBlock(*block)
}

func isBlockHeader(line string) bool {
	return line == dirDeclaration ||
		line == fileDeclaration ||
		line == cmdDeclaration ||
		line == contentHeader ||
		strings.HasPrefix(line, contentDeclaration)
}

func renderTemplateValue(raw, projectName, modulePath string) string {
	replaced := strings.ReplaceAll(raw, projectNameVar, projectName)
	return strings.ReplaceAll(replaced, modulePathVar, modulePath)
}

func outputf(writer io.Writer, enabled bool, format string, args ...any) {
	if enabled && writer != nil {
		fmt.Fprintf(writer, format, args...)
	}
}

func (b *Block) executeCmd(opts ExecuteOptions) error {
	if len(b.Cmds) == 0 {
		return fmt.Errorf("unable to execute empty command block")
	}

	for _, cmd := range b.Cmds {
		raw := renderTemplateValue(cmd.Raw, opts.ProjectName, opts.ModulePath)
		outputf(opts.Stdout, opts.Verbose || opts.DryRun, "run    %s\n", raw)
		if opts.DryRun {
			continue
		}
		if err := runCommand(opts.ProjectRoot, raw, opts.Verbose); err != nil {
			return err
		}
	}
	return nil
}

func runCommand(projectRoot, raw string, verbose bool) error {
	execCmd := exec.Command("sh", "-c", raw)
	execCmd.Dir = projectRoot
	if verbose {
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		if err := execCmd.Run(); err != nil {
			return fmt.Errorf("error while executing command %q: %w", raw, err)
		}
		return nil
	}

	output, err := execCmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("error while executing command %q: %w\n%s", raw, err, strings.TrimSpace(string(output)))
		}
		return fmt.Errorf("error while executing command %q: %w", raw, err)
	}
	return nil
}

func (b *Block) executeDir(opts ExecuteOptions) error {
	if len(b.Dirs) == 0 {
		return fmt.Errorf("unable to create dir on empty dir block")
	}

	for _, path := range b.Dirs {
		renderedPath := renderTemplateValue(string(path), opts.ProjectName, opts.ModulePath)
		fullPath, err := helperfuncs.ResolveWithinRoot(opts.ProjectRoot, renderedPath)
		if err != nil {
			return err
		}
		outputf(opts.Stdout, opts.Verbose || opts.DryRun, "dir    %s\n", renderedPath)
		if opts.DryRun {
			continue
		}
		if err := helperfuncs.CreateDir(fullPath); err != nil {
			return fmt.Errorf("unable to create dir: %w", err)
		}
	}
	return nil
}

func (b *Block) executeFile(opts ExecuteOptions) error {
	if len(b.Files) == 0 {
		return fmt.Errorf("unable to create file on empty file block")
	}

	for _, path := range b.Files {
		renderedPath := renderTemplateValue(string(path), opts.ProjectName, opts.ModulePath)
		fullPath, err := helperfuncs.ResolveWithinRoot(opts.ProjectRoot, renderedPath)
		if err != nil {
			return err
		}
		outputf(opts.Stdout, opts.Verbose || opts.DryRun, "file   %s\n", renderedPath)
		if opts.DryRun {
			continue
		}
		if err := helperfuncs.CreateFileWithMode(fullPath, opts.Force); err != nil {
			return fmt.Errorf("unable to create file: %w", err)
		}
	}
	return nil
}

func (b *Block) executeContent(opts ExecuteOptions) error {
	if len(b.Contents) == 0 {
		return fmt.Errorf("unable to write empty content block")
	}

	for _, content := range b.Contents {
		renderedPath := renderTemplateValue(content.Path, opts.ProjectName, opts.ModulePath)
		fullPath, err := helperfuncs.ResolveWithinRoot(opts.ProjectRoot, renderedPath)
		if err != nil {
			return err
		}

		renderedBody := renderTemplateValue(content.Body, opts.ProjectName, opts.ModulePath)
		outputf(opts.Stdout, opts.Verbose || opts.DryRun, "write  %s\n", renderedPath)
		if opts.DryRun {
			continue
		}
		if err := helperfuncs.WriteFileWithMode(fullPath, renderedBody, opts.Force); err != nil {
			return fmt.Errorf("unable to write file content: %w", err)
		}
	}
	return nil
}

func (b *Block) IsBlockCommand() bool {
	return b.Kind == BlockCommand
}

func (b *Block) addCmd(cmdString string) {
	if strings.TrimSpace(cmdString) == "" {
		return
	}
	b.Cmds = append(b.Cmds, Command{Raw: cmdString})
}

func (b *Block) addDir(dir string) {
	b.Dirs = append(b.Dirs, DirPath(dir))
}

func (b *Block) addFile(path string) {
	b.Files = append(b.Files, FilePath(path))
}

func (b *Block) addContent(path, body string) {
	b.Contents = append(b.Contents, ContentFile{
		Path: path,
		Body: body,
	})
}

func (t *Template) addBlock(block Block) error {
	if len(block.Cmds) == 0 && len(block.Dirs) == 0 && len(block.Files) == 0 && len(block.Contents) == 0 {
		return nil
	}

	kinds := 0
	if len(block.Cmds) > 0 {
		block.Kind = BlockCommand
		kinds++
	}
	if len(block.Dirs) > 0 {
		block.Kind = BlockDir
		kinds++
	}
	if len(block.Files) > 0 {
		block.Kind = BlockFile
		kinds++
	}
	if len(block.Contents) > 0 {
		block.Kind = BlockContent
		kinds++
	}
	if kinds > 1 {
		return fmt.Errorf("blocks must be separated by empty line")
	}

	t.Blocks = append(t.Blocks, block)
	return nil
}

func (t *Template) Execute(opts ExecuteOptions) error {
	if len(t.Blocks) == 0 {
		return fmt.Errorf("unable to execute on empty template")
	}

	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	opts.ProjectRoot = filepath.Clean(opts.ProjectRoot)
	if opts.ModulePath == "" {
		opts.ModulePath = opts.ProjectName
	}

	if opts.DryRun {
		outputf(opts.Stdout, true, "plan   %s\n", opts.ProjectRoot)
	} else {
		if err := helperfuncs.CreateDir(opts.ProjectRoot); err != nil {
			return fmt.Errorf("error while creating project root: %w", err)
		}
	}

	for _, kind := range []BlockKind{BlockDir, BlockFile, BlockContent, BlockCommand} {
		for _, b := range t.Blocks {
			if b.Kind != kind {
				continue
			}

			switch b.Kind {
			case BlockCommand:
				if err := b.executeCmd(opts); err != nil {
					return fmt.Errorf("error while executing commands: %w", err)
				}
			case BlockDir:
				if err := b.executeDir(opts); err != nil {
					return fmt.Errorf("error while creating dirs: %w", err)
				}
			case BlockFile:
				if err := b.executeFile(opts); err != nil {
					return fmt.Errorf("error while creating files: %w", err)
				}
			case BlockContent:
				if err := b.executeContent(opts); err != nil {
					return fmt.Errorf("error while writing file contents: %w", err)
				}
			}
		}
	}

	return nil
}
