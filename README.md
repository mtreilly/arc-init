# arc-init

Initialize Arc components.

## Features

- **system** - Initialize global arc configuration (~/.config/arc/)
- **project** - Initialize project-local configuration (.arc/config.yaml)
- **shell** - Initialize shell completions (bash, zsh, fish, PowerShell)

## Installation

```bash
go install github.com/mtreilly/arc-init@latest
```

## Usage

```bash
# Initialize system configuration interactively
arc-init system --interactive

# Initialize project configuration
arc-init project --interactive

# Initialize with scaffolding and gitignore
arc-init project --scaffold --gitignore

# Set up shell completions
arc-init shell
```

## License

MIT
