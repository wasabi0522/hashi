# hashi Command Reference

hashi is a tool that manages **git branches / worktrees / tmux sessions and windows** together.
It automatically creates an isolated working directory (worktree) and tmux window for each branch,
making branch switching seamless.

See the [README](../README.md) for an overview and installation instructions.

## Prerequisites

- **git 2.31+**: Must be run inside a git repository
- **tmux**: Must be installed

## Resource Mapping

hashi maintains the following 1:1 mapping:

```
git branch ─── git worktree ─── tmux window
                                    │
                              tmux session (one per repository)
```

- **tmux session**: One per repository. Named in `hs/org/repo` format
- **tmux window**: One per branch. Opens in the worktree directory
- **git worktree**: One per branch. Located at `.worktrees/<branch>/` (the default branch uses the repository root)

If any resource is missing, hashi automatically creates it.

---

## Global Options

| Option | Description |
|--------|-------------|
| `--verbose`, `-v` | Enable verbose log output |
| `--version` | Show version |

---

## Command List

| Command | Alias | Purpose |
|---------|-------|---------|
| [`hashi new`](#hashi-new) | `n` | Start working on a new branch |
| [`hashi switch`](#hashi-switch) | `sw` | Switch to an existing branch |
| [`hashi rename`](#hashi-rename) | `mv` | Rename a branch |
| [`hashi remove`](#hashi-remove) | `rm` | Delete a branch and its associated resources |
| [`hashi list`](#hashi-list) | `ls` | List managed resources |
| [`hashi init`](#hashi-init) | - | Generate a `.hashi.yaml` configuration template |
| [`hashi completion`](#hashi-completion) | - | Output shell completion script |

---

## Branch Name Constraints

All commands require branch names to follow these rules:

| Rule | Bad Example | Error Message |
|------|-------------|---------------|
| Must not be empty | | `branch name must not be empty` |
| Must not contain whitespace (spaces, tabs) | `feature login` | `branch name contains whitespace` |
| Must not contain control characters | | `branch name contains control character` |
| Must not contain `~` `^` `*` `?` `[` `\` | `feature[1]` | `branch name contains invalid character` |
| Must not contain `:` | `fix:bug` | `branch name contains ':'` |
| Must not contain `..` | `feature..login` | `branch name contains '..'` |
| Must not contain `@{` | `feature@{0}` | `branch name contains '@{'` |
| Must not start with `-` | `-feature` | `branch name must not start with '-'` |
| Must not start with `.` | `.feature` | `branch name must not start with '.'` |
| Must not end with `.` | `feature.` | `branch name must not end with '.'` |
| Must not end with `/` | `feature/` | `branch name must not end with '/'` |
| Must not end with `.lock` | `feature.lock` | `branch name must not end with '.lock'` |
| Must not contain `//` | `feat//login` | `branch name contains '//'` |

These rules conform to git's branch naming conventions.

---

## hashi new

```
hashi new <branch> [base]
```
Alias: `hashi n`

**Start working on a new branch.** Creates a branch, worktree, and tmux window together, then [connects](#tmux-connection-behavior) to the branch's workspace.

### Basic Usage

```bash
# Create feature-login from the default branch and start working
hashi new feature-login

# Create from the develop branch
hashi new feature-login develop
```

### Detailed Behavior

#### When the branch does not exist (typical case)

1. Create a new branch from `base` (defaults to the default branch if unspecified)
2. Create a worktree at `.worktrees/<branch>/`
3. Set up a tmux window (creates a session too if one doesn't exist)
4. Run [hooks](#hook-execution-order-and-timing) (`copy_files` then `post_new`, if configured)
5. [Connect](#tmux-connection-behavior) to the tmux window

#### When the branch already exists

Specifying `base` results in an error (you cannot change the base of an existing branch).
Without `base`, missing resources (worktree / tmux window) are automatically created and connected.
If a new worktree is created, [hooks](#hook-execution-order-and-timing) are also executed.

> If you just want to switch to an existing branch, [`hashi switch`](#hashi-switch) expresses that intent more clearly.

> When the default branch is specified, the repository root itself is used as the worktree.

### Errors

| Condition | Message |
|-----------|---------|
| Specified `base` for an existing branch | `cannot specify base branch for existing branch '<branch>'` |
| `base` branch does not exist | `branch '<base>' does not exist` |

### Failure Behavior

Rollback is performed on a best-effort basis.

| Failure Point | Behavior |
|---------------|----------|
| Worktree creation | Nothing remains (branch is also not created) |
| tmux creation | Worktree and branch are automatically deleted |
| Hook execution | Resources are left intact (can be inspected manually) |

---

## hashi switch

```
hashi switch <branch>
```
Alias: `hashi sw`

**Switch to an existing branch.** The branch must already exist. If a worktree or tmux window is missing, it is automatically created.

### Basic Usage

```bash
# Switch to the feature-login branch
hashi switch feature-login
```

### Detailed Behavior

1. Verify the branch exists (error if not found)
2. Create a worktree if missing
3. Set up a tmux window (creates a session too if one doesn't exist)
4. Run [hooks](#hook-execution-order-and-timing) only if a new worktree was created (`copy_files` then `post_new`)
5. [Connect](#tmux-connection-behavior) to the tmux window

> When the default branch is specified, the repository root itself is used as the worktree.

### Difference from `new`

| | `new` | `switch` |
|---|---|---|
| Branch does not exist | Creates it | Error |
| `base` argument | Available | Not available |
| Primary use case | Start new work | Return to existing work |

Both commands auto-create missing worktrees and tmux windows.

### Errors

| Condition | Message |
|-----------|---------|
| Branch does not exist | `branch '<branch>' does not exist` |

### Failure Behavior

No rollback is performed. Already-created resources are in a valid state and will be completed on the next run.

---

## hashi rename

```
hashi rename <old> <new>
```
Alias: `hashi mv`

**Rename a branch.** Updates the branch, worktree, and tmux window names together.

### Basic Usage

```bash
# Rename a branch
hashi rename feature-login feature-auth
```

### Detailed Behavior

1. Precondition checks (see error conditions below)
2. Rename the branch with `git branch -m`
3. Worktree handling:
   - **Exists**: Move the directory and run `git worktree repair` to fix consistency
   - **Does not exist**: Create a worktree with the new name and run [hooks](#hook-execution-order-and-timing) (`copy_files` then `post_new`)
4. tmux window handling:
   - **Exists**: Rename the window and update its directory
   - **Does not exist**: Create a window with the new name
   - **No session**: Skip tmux operations

### Errors

| Condition | Message |
|-----------|---------|
| Renaming the default branch | `cannot rename default branch` |
| Old branch does not exist | `branch '<old>' does not exist` |
| New name is already in use (including the default branch) | `branch '<new>' already exists` |

### Failure Behavior

Rollback is performed on a best-effort basis.

| Failure Point | Behavior |
|---------------|----------|
| Worktree move | Reverts the branch name |
| Worktree repair | Reverts the worktree and branch name |
| tmux operations | Branch and worktree remain with the new name |

---

## hashi remove

```
hashi remove [-f] <branch...>
```
Alias: `hashi rm`

**Delete a branch and its associated resources.** Multiple branches can be specified at once.

### Basic Usage

```bash
# Delete with confirmation prompt
hashi remove feature-login

# Force delete (no confirmation)
hashi remove -f feature-login

# Delete multiple branches at once
hashi remove feature-login feature-signup
```

### Detailed Behavior

#### When the branch exists

1. Show a confirmation prompt unless `-f` is specified (with warnings for uncommitted changes or unmerged branches)
2. If deleting the currently active window, switch to the default branch first
3. Delete existing resources in order: **worktree → branch → window**
4. If it was the last window in the session, delete the session too

The window is deleted last to prevent process interruption from the window's termination signal (SIGHUP) when running `hashi remove` from the active window.

#### When the branch does not exist but orphaned resources remain

If only a worktree or tmux window remains (e.g., after directly deleting a branch with `git branch -d`), hashi cleans up those orphaned resources after confirmation. Use `-f` to skip confirmation.

```
$ hashi remove orphan-branch
Remove 'orphan-branch'? (worktree) y/N [N] y
Removed 'orphan-branch'
```

#### When neither the branch nor orphaned resources exist

Results in an error.

### Confirmation Prompt Behavior

- **Declined**: Skips that branch and proceeds to the next one
- **Error**: Aborts processing (remaining branches are not processed)

When stdin is closed (e.g., running from a script), the confirmation is treated as declined (N). Use `-f` to run without confirmation.

### Errors

| Condition | Message |
|-----------|---------|
| Deleting the default branch | `cannot remove default branch` |
| Neither branch nor orphaned resources exist | `branch '<branch>' does not exist` |

### Failure Behavior

No rollback is performed. Already-deleted resources remain deleted, and remaining resources can be cleaned up by running `hashi remove` again.

---

## hashi list

```
hashi list [--json]
```
Alias: `hashi ls`

**List the status of managed resources.** Shows warnings and remediation steps for any inconsistencies. This command is read-only and does not modify any resources.

### Basic Usage

```bash
# Display as a table
hashi list

# Output as JSON
hashi list --json
```

### Table Output Example

Normal state:

```
   BRANCH          WORKTREE                                    STATUS
 * feature/login   /home/user/repo/.worktrees/feature/login
   fix/typo        /home/user/repo/.worktrees/fix/typo
   main            /home/user/repo
```

`*` indicates the currently active tmux window.

With inconsistencies:

```
   BRANCH          WORKTREE              STATUS
   feature/login   (worktree missing)    ⚠ Run 'hashi new feature/login'
   orphan-x        (orphaned window)     ⚠ Run 'hashi remove orphan-x'
   main            /home/user/repo
```

### State Classification

| State | Meaning | Display |
|-------|---------|---------|
| Worktree exists + window exists | Normal | Standard display |
| Worktree exists + no window | Normal (window is auto-recreated on next operation) | Standard display |
| Worktree exists + no branch | Orphaned worktree (e.g., branch deleted directly with `git branch -d`) | Suggests `hashi remove <name>` |
| No worktree + window exists + branch exists | Missing worktree (e.g., worktree manually deleted) | Suggests `hashi new <branch>` |
| No worktree + window exists + no branch | Orphaned window (e.g., branch and worktree manually deleted) | Suggests `hashi remove <name>` |
| No worktree + no window | Not managed by hashi | Not shown |

### JSON Output Format

When `--json` is specified, output follows this format:

```json
[
  {
    "branch": "main",
    "worktree": "/path/to/repo",
    "window": true,
    "active": true,
    "is_default": true,
    "status": "ok"
  },
  {
    "branch": "feature-login",
    "worktree": "/path/to/repo/.worktrees/feature-login",
    "window": true,
    "active": false,
    "is_default": false,
    "status": "ok"
  },
  {
    "branch": "orphan-x",
    "window": true,
    "active": false,
    "is_default": false,
    "status": "orphaned_window"
  }
]
```

| Field | Type | Description |
|-------|------|-------------|
| `branch` | string | Branch name |
| `worktree` | string | Worktree path (omitted if it doesn't exist) |
| `window` | bool | Whether a tmux window exists |
| `active` | bool | Whether this is the currently active window |
| `is_default` | bool | Whether this is the default branch |
| `status` | string | `"ok"`, `"worktree_missing"`, `"orphaned_window"`, `"orphaned_worktree"` |

### Notes

- The default branch is always shown since it is tied to the repository's main worktree
- Branches created directly with `git branch` that have neither a worktree nor a window are not shown (not managed by hashi)

---

## hashi init

```
hashi init
```

**Generate a `.hashi.yaml` configuration template.** Creates a configuration file with default values in the project root.

```bash
hashi init
# => Created /path/to/repo/.hashi.yaml
```

### Errors

| Condition | Message |
|-----------|---------|
| `.hashi.yaml` already exists | `.hashi.yaml already exists` |

---

## hashi completion

```
hashi completion <bash|zsh|fish>
```

**Output a shell completion script.** Enables tab completion for branch names and other arguments.

### Setup Examples

```bash
# Bash
hashi completion bash > /usr/local/etc/bash_completion.d/hashi

# Zsh
hashi completion zsh > "${fpath[1]}/_hashi"

# Fish
hashi completion fish > ~/.config/fish/completions/hashi.fish
```

Restart your shell after running the command to activate completions.

### Errors

| Condition | Message |
|-----------|---------|
| Unsupported shell specified | `unsupported shell: <name>` |

---

## Configuration File (`.hashi.yaml`)

Place a `.hashi.yaml` file in the project root to customize the worktree directory and hook behavior.
Use [`hashi init`](#hashi-init) to generate a template.

```yaml
# Worktree directory (relative path from the repository root)
worktree_dir: .worktrees

hooks:
  # Files/directories to copy from the repository root when a worktree is created
  copy_files:
    - .env
    - .claude

  # Shell commands to run after worktree creation (executed sequentially)
  post_new:
    - npm install
    - cp .env.example .env
```

### Environment Variable Overrides

Settings can be overridden using environment variables with the `HASHI_` prefix.
Environment variables take precedence over the configuration file.

| Environment Variable | Corresponding Setting |
|---------------------|----------------------|
| `HASHI_WORKTREE_DIR` | `worktree_dir` |

```bash
# Change the worktree directory via environment variable
HASHI_WORKTREE_DIR=.wt hashi new feature-login
```

### worktree_dir

Specifies the directory where worktrees are placed. Defaults to `.worktrees`.

- Only relative paths are allowed (absolute paths cause an error)
- Paths containing `..` are not allowed
- `.` (directly under the repository root) is not allowed

### hooks.copy_files

A list of files and directories to **copy from the repository root to the worktree** when a new worktree is created.
Useful for files excluded by `.gitignore` (such as `.env` or editor configuration) that you want available in worktrees.

```yaml
hooks:
  copy_files:
    - .env            # Copy a file
    - .claude         # Copy a directory recursively
    - CLAUDE.md       # Copy a file
```

- Both files and directories can be specified (directories are copied recursively)
- Entries that don't exist in the repository root are silently ignored (no error)
- File permissions are preserved

### hooks.post_new

A list of shell commands to run after a new worktree is created.
Commands are executed in order; if any command fails, subsequent commands are skipped (chained with `&&`).
When `post_new` is configured, the user's shell (`$SHELL`, or `sh` if unset) is launched after hooks complete (or fail).

```yaml
hooks:
  post_new:
    - npm install
    - make setup
```

### Hook Execution Order and Timing

When a new worktree is created, processing occurs in the following order:

```
worktree creation → copy_files → post_new → tmux connection
```

When an existing worktree is reused, neither `copy_files` nor `post_new` is executed.

| Command | Hooks Executed |
|---------|---------------|
| `new` (new branch) | Always |
| `new` (existing branch, no worktree) | Yes |
| `new` (existing branch, worktree exists) | No |
| `switch` (no worktree) | Yes |
| `switch` (worktree exists) | No |
| `rename` (no worktree) | Yes |
| `rename` (worktree exists) | No |
| `remove` | No |

---

## tmux Connection Behavior

hashi automatically selects the tmux connection method based on the execution environment.

| Environment | Connection Method |
|-------------|-------------------|
| Inside a tmux session (`$TMUX` is set) | `switch-client` to the target window |
| Outside a tmux session (`$TMUX` is not set) | `attach-session` to the target window |
