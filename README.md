# DotCLI

A TUI-based dotfiles manager with intelligent package management and dependency resolution.

## Features

- **Interactive TUI** for module selection and management
- **Package Manager Integration** - automatically detects and uses brew/apt/pacman/yum/snap
- **Dependency Resolution** - handles module dependencies automatically
- **Package Verification** - skips already installed packages
- **Module Creation** - built-in forms for creating new modules
- **Force Reinstall** - option to reinstall existing packages

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
- `c` - Create new module
- `a` - Add dotfile to module
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
  brew: [myapp, helper-tool]
  apt: [myapp, helper-tool]
dotfiles:
  - source: dotfiles/.config
    destination: .config/myapp
```

## Templates

- **basic** - Empty module
- **shell** - Shell configuration (zsh)
- **editor** - Editor configuration (neovim)
- **cli-tool** - CLI tools configuration

## Requirements

- Go 1.21+
- One of: brew, apt, pacman, yum, snap