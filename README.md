# DotCLI

A TUI-based dotfiles manager with intelligent package management and dependency resolution.

## Features

- **Interactive TUI** for module selection and management
- **Package Manager Integration** - automatically detects and uses brew/apt/pacman/yum/snap
- **Dependency Resolution** - handles module dependencies automatically
- **Package Verification** - skips already installed packages
- **Module Creation** - built-in forms for creating new modules
- **Force Reinstall** - option to reinstall existing packages
- **Export Mode** - install only dotfiles without packages/commands

## Quick Start

```bash
# Build
go build -o bin/dotcli .

# Run
./bin/dotcli
```

## Usage

### Interactive Interface
- `↑/↓` - Navigate modules
- `Space` - Select/deselect modules
- `f` - Toggle force reinstall
- `x` - Toggle export mode (dotfiles only)
- `c` - Create new module
- `e` - Edit existing module
- `a` - Add dotfile to module
- `i` - Import existing dotfile
- `Enter` - Install selected modules
- `q` - Quit

### Module Structure
```
~/.dotfiles/modules/myapp/
├── config.yaml          # Module configuration
├── install.sh           # Custom setup script
└── dotfiles/            # Your dotfiles
    ├── .config/
    └── .myapprc
```

### Configuration Format
```yaml
name: myapp
description: "My application configuration"
dependencies:
  - shell
packages:
  common: [git, curl]
  specific:
    - name: myapp
      manager: brew
    - name: myapp
      manager: apt
commands:
  - command: "echo 'Custom setup command'"
    os: ""
dotfiles:
  - source: dotfiles/.config
    destination: ~/.config/myapp
```

## Working with Dotfiles

### Adding Dotfiles to Modules
1. **Add new dotfile mapping** (`a`): Define source and destination paths
2. **Import existing files** (`i`): Move existing config files into your dotfiles and create symlinks

### Import Process
When you import an existing file (e.g., `~/.bashrc`):
1. File is copied to your module's dotfiles directory
2. Original file is removed
3. Symlink is created from original location to module file
4. Configuration is updated automatically

### Export Mode
Perfect for when software is already installed and you just want to apply your configurations:
1. Press `x` to enable export mode
2. Select modules with `Space`
3. Press `Enter` to install only dotfiles (skips packages and commands)

### Path Expansion
The system supports various path formats:
- `~/.bashrc` - Home directory expansion
- `$HOME/.config/nvim` - Environment variable expansion
- `.bashrc` - Relative to home directory
- `/absolute/path` - Absolute paths

## Templates

- **basic** - Empty module
- **shell** - Shell configuration (zsh)
- **editor** - Editor configuration (neovim)
- **cli-tool** - CLI tools configuration

## Requirements

- Go 1.21+
- One of: brew, apt, pacman, yum, snap