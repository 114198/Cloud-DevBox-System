# DevBox CLI

Command-line interface for managing Cloud DevBox development environments.

## Installation

### macOS (Homebrew)

```bash
brew install cloud-devbox/tap/devbox
```

### Windows (Scoop)

```powershell
scoop bucket add devbox https://github.com/your-org/scoop-bucket
scoop install devbox
```

### Linux (Debian/Ubuntu)

```bash
curl -LO https://github.com/your-org/cloud-devbox/releases/latest/download/devbox-cli_amd64.deb
sudo dpkg -i devbox-cli_amd64.deb
```

### From Source

```bash
cargo install --path .
```

## Quick Start

```bash
# Login to Cloud DevBox
devbox login

# List available templates
devbox template list

# Create a new environment
devbox create my-project --template nodejs

# List your environments
devbox list

# SSH into an environment
devbox ssh my-project

# Stop an environment
devbox stop my-project

# Delete an environment
devbox delete my-project
```

## Commands

### Authentication

```bash
devbox login              # Login to Cloud DevBox
devbox logout             # Logout from Cloud DevBox
devbox whoami             # Show current user info
```

### Environment Management

```bash
devbox env create <name> --template <template>  # Create environment
devbox env list [--status <status>]             # List environments
devbox env info <name>                          # Show environment details
devbox env start <name>                         # Start environment
devbox env stop <name>                          # Stop environment
devbox env delete <name> [--force]              # Delete environment
devbox env ssh <name>                           # SSH into environment
devbox env ssh-config <name>                    # Generate SSH config
```

Shortcuts are available:
```bash
devbox create <name> --template <template>
devbox list
devbox start <name>
devbox stop <name>
devbox delete <name>
devbox ssh <name>
```

### Template Management

```bash
devbox template list [--category <category>]    # List templates
devbox template show <name>                     # Show template details
```

### Configuration

```bash
devbox config show                              # Show all config
devbox config list                              # List config keys
devbox config get <key>                         # Get config value
devbox config set <key> <value>                 # Set config value
devbox config reset                             # Reset to defaults
```

Available configuration keys:
- `api_endpoint` - API server URL
- `default_template` - Default template for new environments
- `output_format` - Output format (table, json, yaml)
- `color` - Enable/disable colored output
- `default_cpu` - Default CPU allocation
- `default_memory` - Default memory allocation
- `default_storage` - Default storage allocation

### Shell Completions

```bash
# Bash
devbox completions bash > ~/.local/share/bash-completion/completions/devbox

# Zsh
devbox completions zsh > ~/.zfunc/_devbox

# Fish
devbox completions fish > ~/.config/fish/completions/devbox.fish

# PowerShell
devbox completions powershell >> $PROFILE
```

## Output Formats

Use `--format` or `-f` to change output format:

```bash
devbox list --format json
devbox list --format yaml
devbox list --format table  # default
```

## Verbose Mode

Use `--verbose` or `-v` for debug output:

```bash
devbox -v list
```

## Configuration File

Configuration is stored in:
- Linux/macOS: `~/.config/devbox/config.json`
- Windows: `%APPDATA%\devbox\config.json`

## Credentials

API tokens are stored securely using the system keychain:
- macOS: Keychain
- Linux: Secret Service (GNOME Keyring, KWallet)
- Windows: Credential Manager

## License

MIT License - see LICENSE file for details.
