package templateparser

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	helperfuncs "github.com/iartmediums.com/cmd_create_file/internal/helper_funcs"
)

const (
	dirDeclaration  = "# dir"
	fileDeclaration = "# file"
	cmdDeclaration  = "# cmd"
	projectNameVar  = "{{PN}}"
)

type parseTempState int

const (
	noneState parseTempState = iota
	parseCmd
	parseDir
	parseFile
)

type DirPath string
type FilePath string

type Command struct {
	Raw string
}

type BlockKind int

const (
	BlockUnknown BlockKind = iota
	BlockCommand
	BlockDir
	BlockFile
)

type Block struct {
	Kind  BlockKind
	Dirs  []DirPath
	Files []FilePath
	Cmds  []Command
}

type Template struct {
	Blocks []Block
}

func ParseTemplate(tempPath string) (*Template, error) {
	file, err := os.Open(tempPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var currentBlock Block
	var template Template

	state := noneState
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		switch line {
		case dirDeclaration:
			state = parseDir
			continue
		case fileDeclaration:
			state = parseFile
			continue
		case cmdDeclaration:
			state = parseCmd
			continue
		case "":
			state = noneState
			if err := template.addBlock(currentBlock); err != nil {
				return nil, err
			}
			currentBlock = Block{}
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

	if err := template.addBlock(currentBlock); err != nil {
		return nil, err
	}

	return &template, nil
}

func (b *Block) executeCmd(projectName, projectRoot string) error {
	if len(b.Cmds) == 0 {
		return fmt.Errorf("unable to execute empty command block")
	}

	for _, cmd := range b.Cmds {
		raw := strings.ReplaceAll(cmd.Raw, projectNameVar, projectName)
		fmt.Printf("Executing command in %s: %s\n", projectRoot, raw)
		if err := runCommand(projectRoot, raw); err != nil {
			return err
		}
	}
	fmt.Println("Finished executing commands")
	return nil
}

func runCommand(projectRoot, raw string) error {
	execCmd := exec.Command("sh", "-c", raw)
	execCmd.Dir = projectRoot
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	if err := execCmd.Run(); err != nil {
		return fmt.Errorf("error while executing command %q: %w", raw, err)
	}
	return nil
}

func (b *Block) executeDir(projectRoot string) error {
	if len(b.Dirs) == 0 {
		return fmt.Errorf("unable to create dir on empty dir block")
	}

	for _, path := range b.Dirs {
		fullPath, err := helperfuncs.ResolveWithinRoot(projectRoot, string(path))
		if err != nil {
			return err
		}
		fmt.Printf("Creating dir: %s\n", fullPath)
		if err := helperfuncs.CreateDir(fullPath); err != nil {
			return fmt.Errorf("unable to create dir: %w", err)
		}
	}
	fmt.Println("Finished creating dirs")
	return nil
}

func (b *Block) executeFile(projectRoot string) error {
	if len(b.Files) == 0 {
		return fmt.Errorf("unable to create file on empty file block")
	}

	for _, path := range b.Files {
		fullPath, err := helperfuncs.ResolveWithinRoot(projectRoot, string(path))
		if err != nil {
			return err
		}
		fmt.Printf("Creating file: %s\n", fullPath)
		if err := helperfuncs.CreateFile(fullPath); err != nil {
			return fmt.Errorf("unable to create file: %w", err)
		}
	}
	fmt.Println("Finished creating files")
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

func (t *Template) addBlock(block Block) error {
	if len(block.Cmds) == 0 && len(block.Dirs) == 0 && len(block.Files) == 0 {
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
	if kinds > 1 {
		return fmt.Errorf("blocks must be separated by empty line")
	}

	t.Blocks = append(t.Blocks, block)
	return nil
}

func (t *Template) Execute(projectName, projectRoot string) error {
	if len(t.Blocks) == 0 {
		return fmt.Errorf("unable to execute on empty template")
	}

	projectRoot = filepath.Clean(projectRoot)
	if err := helperfuncs.CreateDir(projectRoot); err != nil {
		return fmt.Errorf("error while creating project root: %w", err)
	}

	for _, b := range t.Blocks {
		switch b.Kind {
		case BlockCommand:
			if err := b.executeCmd(projectName, projectRoot); err != nil {
				return fmt.Errorf("error while executing commands: %w", err)
			}
		case BlockDir:
			if err := b.executeDir(projectRoot); err != nil {
				return fmt.Errorf("error while creating dirs: %w", err)
			}
		case BlockFile:
			if err := b.executeFile(projectRoot); err != nil {
				return fmt.Errorf("error while creating files: %w", err)
			}
		default:
			return fmt.Errorf("unsupported block kind")
		}
	}

	return nil
}
