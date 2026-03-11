# mk

`mk` is a command-line tool for creating files, directories, and project
structures with minimal typing.

It combines three workflows:

- smart path creation
- optional editor handoff for a newly created file
- reusable project templates

## What It Does

`mk` creates missing parent directories automatically.

Path rules are simple:

- if a path ends with `/`, `mk` creates a directory
- otherwise, `mk` creates a file

It can also scaffold projects from templates stored under:

```text
~/.config/mk/templates
```

## Reserved Commands

The following words are reserved for command handling:

- `tmpl`
- `config`
- `help`

If you want to create a file or directory with one of those names, prefix it
with `./`.

Example:

```sh
mk ./tmpl ./config ./help
```

## Installation
## Platform Support

`mk` is currently documented and tested as a Go CLI for:

- Linux
- macOS
- Windows

Requirements on every platform:

- Go 1.25 or newer installed
- a shell environment for running the binary

Optional but recommended:

- a configured editor via `editor` in `~/.config/mk/config.json`
- or `VISUAL` / `EDITOR` environment variables for `mk -e`, `mk tmpl edit`, and `mk config edit`

## Install Prerequisites

### Linux

Any modern Linux distribution should work as long as Go is installed and your user can write to:

```text
~/.config/mk
```

Typical Go installation approaches by distro:

- Ubuntu / Debian: install Go from the distro package manager or from the official Go tarball
- Fedora: install Go from `dnf` or from the official Go tarball
- Arch: install Go from `pacman`

After installing Go, verify:

```sh
go version
```

### macOS

Install Go with either:

- Homebrew: `brew install go`
- the official Go installer package

Then verify:

```sh
go version
```

### Windows

Install Go using the official Windows installer.

Then verify in PowerShell:

```powershell
go version
```

Notes for Windows users:

- `mk` uses the same Go-based build/install flow as Linux and macOS
- the README examples use Unix-style paths; adapt `PATH` updates to PowerShell or Windows Settings
- the config examples below use `~/.config/mk`; if your environment resolves `HOME`, `mk` will use that home directory
- editor commands should be set to something available from your shell, for example `code`, `notepad`, or `nvim`

### Option 1: Build From Source

From the repository root:

```sh
go build -o mk .
```

On Windows, use:

```powershell
go build -o mk.exe .
```

Move the binary somewhere on your `PATH`, for example:

```sh
mv mk ~/.local/bin/mk
```

Make sure `~/.local/bin` is on your `PATH`.

Linux/macOS example:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Windows PowerShell example for the current session:

```powershell
$env:Path = "$HOME\AppData\Local\Programs;$env:Path"
```

### Option 2: Install With Go

If you want to install it directly with Go, run:

```sh
go install github.com/iartmediums/mk-cli@latest
```

This places the binary in your Go bin directory, usually:

```text
$(go env GOPATH)/bin
```

Ensure that directory is on your `PATH`.

You can inspect the exact bin directory with:

```sh
go env GOPATH
```

On many systems the binary ends up in one of these locations:

- Linux/macOS: `$(go env GOPATH)/bin`
- Windows: `%USERPROFILE%\\go\\bin`

The repository/module name is `mk-cli`, but the installed executable is still
named `mk`.

### Verify Installation

```sh
mk help
mk --version
```

If `mk` is not found, the binary is installed but not yet on your `PATH`.
If installed from a tagged release with `go install ...@latest`, `mk --version`
shows that module version. Local source builds may report `dev`.

## First Release

For the first public shipment, create and push a semver tag before telling users
to install with `@latest`.

Example:

```sh
git tag v1.0.0
git push origin v1.0.0
```

After that, users can install either:

```sh
go install github.com/iartmediums/mk-cli@v1.0.0
```

or:

```sh
go install github.com/iartmediums/mk-cli@latest
```

## First Run

On first use, `mk` creates its config directory and default files under:

```text
~/.config/mk
```

Created files may include:

- `~/.config/mk/config.json`
- `~/.config/mk/templates/go.mktmpl`

This means users do not need to manually create configuration files before the
first command.

To restore or repair missing setup files later, run:

```sh
mk config setup
```

## Basic Usage

### Create a File or Directory

```sh
mk <path>
```

Examples:

```sh
mk main.go
mk internal/server/server.go
mk internal/server/
```

### Create Multiple Paths

```sh
mk <path1> <path2> <path3> ...
```

Example:

```sh
mk internal/server/server.go internal/config/ cmd/app/main.go
```

### Create and Open a File

```sh
mk -e <path>
```

`-e` only accepts a single path.

- if the path resolves to a file, `mk` opens it in the configured editor
- if the path resolves to a directory, `mk` creates the directory and skips the editor

Example:

```sh
mk -e main.go
```

## Template Commands

Templates are stored in:

```text
~/.config/mk/templates
```

Template files use the `.mktmpl` extension.

### Initialize a Project From a Template

```sh
mk tmpl init <template_name> <project_name> [-v] [-f] [-m <module_path>]
```

Flags:

- `-v` shows verbose execution output
- `-f` overwrites generated files if they already exist
- `-m` sets the module path used by the template

Module resolution order:

1. `-m <module_path>`
2. `module` from `~/.config/mk/config.json`
3. `<project_name>`

Examples:

```sh
mk tmpl init go myproject
mk tmpl init go myproject -v
mk tmpl init go myproject -m github.com/user/myproject
```

### Create a New Template

```sh
mk tmpl new <name>
```

Creates:

```text
~/.config/mk/templates/<name>.mktmpl
```

If `editOnTmplCreation` is enabled in config, the new template is opened in
your editor automatically.

### Edit a Template

```sh
mk tmpl edit <name>
```

### Remove a Template

```sh
mk tmpl remove <name>
```

### List Templates

```sh
mk tmpl list
```

### Verify a Template

```sh
mk tmpl verify <name>
```

This validates the template in dry-run mode without writing files.

### Template Help

```sh
mk tmpl help
```

This shows both template command usage and template syntax rules.

## Template Syntax

Supported template blocks:

```text
# dir
# file
# content <path>
# cmd
```

Rules:

- blocks must be separated by an empty line
- paths must stay inside the target project root
- `{{PN}}` is replaced with the project name
- `{{MODULE}}` is replaced with the resolved module path
- empty `# dir`, `# file`, and `# cmd` blocks are allowed

Example:

```text
# dir
cmd
cmd/{{PN}}
internal

# content cmd/{{PN}}/main.go
package main

func main() {}

# cmd
go mod init {{MODULE}}
go fmt ./...
```

## Configuration Commands

### Edit Config

```sh
mk config edit
```

### Restore Config and Default Templates

```sh
mk config setup
```

This command:

- creates missing config files
- restores missing config fields in `config.json`
- restores the default `go.mktmpl` unless disabled

### Show Config Help

```sh
mk config help
```

## Configuration File

Location:

```text
~/.config/mk/config.json
```

Example:

```json
{
  "editor": "nvim",
  "module": "github.com/<username>/{{.PN}}",
  "disableGoDefaultTemp": false,
  "editOnTmplCreation": true
}
```

Fields:

- `editor`
  Command used to open files and templates.
- `module`
  Default module path template. `{{.PN}}` is replaced with the project name.
- `disableGoDefaultTemp`
  Prevents creation of the default `go.mktmpl` during setup.
- `editOnTmplCreation`
  Opens a newly created template in the editor automatically.

## Help

```sh
mk help
mk --version
mk tmpl help
mk config help
```

## Shipping Notes

Before sharing this project with users, assume the following:

- users need Go installed unless you publish prebuilt binaries
- users need the installed `mk` binary available on their `PATH`
- users who want editor integration should configure `editor`, `VISUAL`, or `EDITOR`

At the moment this repository does not provide:

- distro-native packages such as `.deb`, `.rpm`, or `pacman` packages
- Homebrew, Scoop, or Chocolatey formulas
- prebuilt binaries attached to releases

So the supported install path today is source-based Go installation.
