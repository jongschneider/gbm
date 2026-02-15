# GBM - Git Branch Manager

A CLI tool for managing Git repository branches and worktrees based on configuration.

## Installation

```bash
go install ./cmd
# Or use: just install
```

## Quick Start

```bash
# Initialize a new repository with worktree structure
gbm init

# Clone an existing repository
gbm clone git@github.com:user/repo.git

# List worktrees
gbm wt list

# Add a new worktree
gbm wt add feature-branch

# Switch to a worktree
gbm wt switch feature-branch
```

## Config TUI

The `gbm config` command launches an interactive terminal interface for managing your `.gbm/config.yaml` file.

### Usage

```bash
# Launch the config TUI
gbm config

# Get help
gbm config --help
```

### Sections

The config TUI provides four main sections accessible via the sidebar:

1. **Basics** - Core settings:
   - Default Branch: Branch to use when creating new worktrees (e.g., `main`, `develop`)
   - Worktrees Directory: Where worktrees are stored (e.g., `./worktrees`)

2. **JIRA** - JIRA integration settings:
   - Enable/disable toggle
   - Server: Host URL, username, API token
   - Filters: Status, priority, type filters
   - Attachments: Enable/disable, max size, directory
   - Markdown: Export settings for JIRA issues

3. **FileCopy** - File copy rules for new worktrees:
   - Source worktree: Which worktree to copy files from
   - Files: List of files to copy

4. **Worktrees** - Pre-defined worktree configurations:
   - Name, branch, merge target, description

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| **Navigation** | |
| `↑`/`↓` | Move up/down |
| `Enter` | Select/confirm |
| `Tab` | Next field |
| `Shift+Tab` | Previous field |
| `Esc` | Cancel/go back |
| **Sidebar** | |
| `s` | Save all changes |
| `r` | Reset from file |
| `q` | Quit |
| `?` | Show help |
| **Table Forms** | |
| `a` | Add new entry |
| `e` | Edit selected entry |
| `d` | Delete selected entry |
| **Modals** | |
| `y`/`Y` | Confirm (yes) |
| `n`/`N` | Cancel (no) |
| **FilePicker** | |
| `b` | Browse files |
| `→` | Open directory |
| `←` | Parent directory |
| `Space` | Select file |

### Examples

#### Edit Default Branch
1. Run `gbm config`
2. Select "Basics" from the sidebar
3. Edit the "Default Branch" field
4. Press `s` to save

#### Configure JIRA Integration
1. Run `gbm config`
2. Select "JIRA" from the sidebar
3. Toggle "Enable JIRA Integration" to Yes
4. Fill in server details (host, username, API token)
5. Press `s` to save

#### Add a File Copy Rule
1. Run `gbm config`
2. Select "FileCopy" from the sidebar
3. Press `a` to add a new rule
4. Enter source worktree and files
5. Press `Enter` to confirm, then `s` to save

## Troubleshooting

### Config TUI Validation Errors

#### Basics Section

| Error | Cause | Fix |
|-------|-------|-----|
| `default_branch is required` | Empty branch name | Enter a valid branch name (e.g., `main`) |
| `branch name contains invalid characters` | Special characters in branch name | Use only alphanumeric, `-`, `/`, `_` |
| `worktrees_dir is required` | Empty directory path | Enter a valid directory path (e.g., `./worktrees`) |

#### JIRA Section

| Error | Cause | Fix |
|-------|-------|-----|
| `JIRA host is required` | Empty host field | Enter your JIRA server URL |
| `JIRA host must start with http:// or https://` | Missing protocol | Add `https://` prefix (e.g., `https://jira.company.com`) |
| `invalid URL format` | Malformed URL | Check URL for typos |
| `username is required` | Empty username | Enter your JIRA username or email |
| `API token is required` | Empty API token | Generate and enter a JIRA API token |

#### Worktrees Section

| Error | Cause | Fix |
|-------|-------|-----|
| `Name is required` | Empty worktree name | Enter a name for the worktree |
| `Invalid name: use only alphanumeric, -, _` | Special characters in name | Remove spaces and special characters |
| `Branch is required` | Empty branch field | Enter the branch name |
| `Worktree '<name>' already exists` | Duplicate name | Choose a unique worktree name |

### Common Issues

#### TUI fails with `/dev/tty` error
The TUI requires an interactive terminal. Run `gbm config` directly in your terminal, not through pipes or redirects.

#### Changes not saving
- Ensure you press `s` to save before quitting
- Check the `[modified]` indicator in the footer - if visible, you have unsaved changes
- Verify write permissions on `.gbm/config.yaml`

#### Config file not found
```bash
# Create the config directory and file
mkdir -p .gbm
gbm init-config
```

## Configuration File

GBM stores configuration in `.gbm/config.yaml`. See `config.example.yaml` for a complete reference.

## Commands

| Command | Description |
|---------|-------------|
| `gbm config` | Interactive configuration editor |
| `gbm init` | Initialize a new repository |
| `gbm clone <url>` | Clone with worktree structure |
| `gbm wt list` | List worktrees |
| `gbm wt add <branch>` | Add a new worktree |
| `gbm wt switch <name>` | Switch to a worktree |
| `gbm wt remove <name>` | Remove a worktree |
| `gbm sync` | Synchronize worktrees with config |

Run `gbm --help` or `gbm <command> --help` for more details.

## Development

```bash
just build     # Build binary
just validate  # Run all checks
just storybook # View TUI component storybook
```

See `CLAUDE.md` for detailed development guidelines.
