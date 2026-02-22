<div align="center">

# 🥢 hashi (箸)

**Git worktree and tmux session manager**

[![CI](https://github.com/wasabi0522/hashi/actions/workflows/ci.yaml/badge.svg)](https://github.com/wasabi0522/hashi/actions/workflows/ci.yaml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
![git 2.31+](https://img.shields.io/badge/git-2.31%2B-green)
![tmux](https://img.shields.io/badge/tmux-required-green)

</div>

Ever work on a feature branch, then need to switch to another branch for a hotfix? You stash your changes, switch, open a new terminal, forget where you were...

hashi gives each branch its own directory and its own tmux window. Switch branches by switching windows. No stashing, no context lost.

## Quick Start

> **Install:** `brew install wasabi0522/tap/hashi` ([other methods](#installation))

Create a branch — its directory and tmux window appear automatically:

```bash
hashi new feature/login
```

Start a second branch:

```bash
hashi new fix/typo
```

Switch back — your tmux window jumps to feature/login:

```bash
hashi switch feature/login
```

See everything at a glance:

```bash
hashi list
```

```
   BRANCH          WORKTREE                                    STATUS
 * feature/login   /home/user/repo/.worktrees/feature/login
   fix/typo        /home/user/repo/.worktrees/fix/typo
   main            /home/user/repo
```

Done with a branch? Remove everything together:

```bash
hashi remove fix/typo
```

```
Remove 'fix/typo'? [y/N] y
Removed 'fix/typo'
```

## Commands

| Command | Description |
|---------|-------------|
| `hashi new <branch> [base]` | Create a branch with its worktree and tmux window |
| `hashi switch <branch>` | Switch to an existing branch and its tmux window |
| `hashi list [--json]` | List all managed branches, worktrees, and windows |
| `hashi rename <old> <new>` | Rename a branch, worktree, and window together |
| `hashi remove [-f] <branch...>` | Remove branches, worktrees, and windows together |
| `hashi init` | Generate a `.hashi.yaml` config template |
| `hashi completion <shell>` | Output shell completion script (bash/zsh/fish) |

hashi manages local resources only — it never runs `git push`, `git pull`, or modifies remote branches.

## How It Works

### Worktree layout

Each branch gets its own directory under `.worktrees/` in the repository root:

```
~/repo/
├── .worktrees/
│   ├── feature/login/      ← hashi new feature/login
│   └── fix/typo/           ← hashi new fix/typo
├── .git/
├── src/
└── ...
```

The main branch uses the original clone directory — it is never duplicated under `.worktrees/`.

### State detection and self-repair

If resources get out of sync (e.g. a tmux window was closed manually), `hashi list` shows what went wrong and how to fix it:

```
   BRANCH          WORKTREE              STATUS
   feature/login   (worktree missing)    ⚠ Run 'hashi new feature/login'
   orphan-x        (orphaned window)     ⚠ Run 'hashi remove orphan-x'
```

`hashi switch` and `hashi new` automatically repair missing resources — if a window exists without a worktree (or vice versa), the missing piece is created.

<details>
<summary>Session naming, rollback, and tmux behavior</summary>

#### Session and window naming

hashi derives the tmux session name from the git remote URL:

```
git@github.com:user/repo.git  →  tmux session: user/repo
```

Each branch becomes a tmux window within that session. Your tmux status bar becomes a branch list.

#### Rollback on failure

`hashi new` performs multiple steps (create branch, create worktree, create tmux window). If a step fails partway through, hashi rolls back already-created resources on a best-effort basis.

#### Works inside and outside tmux

- **Inside tmux**: `hashi new` / `hashi switch` use `switch-client` to jump to the target window.
- **Outside tmux**: hashi creates the session if needed and attaches to it.

</details>

## Installation

> [!NOTE]
> Requires **git 2.31+** and **tmux**.

```bash
brew install wasabi0522/tap/hashi
```

<details>
<summary>Other installation methods</summary>

**Go install:**

```bash
go install github.com/wasabi0522/hashi@latest
```

**GitHub Releases:**

Download a binary from [Releases](https://github.com/wasabi0522/hashi/releases) and place it somewhere in your `PATH`.

</details>

## Configuration

No config file is needed — defaults work out of the box. To customize, place a `.hashi.yaml` at the repository root:

```yaml
# Change worktree directory (default: .worktrees)
worktree_dir: .wt

# Commands to run after creating a worktree
hooks:
  post_new:
    - npm install
```

Environment variables (`HASHI_` prefix) override the config file. For example, `HASHI_WORKTREE_DIR=.wt` overrides `worktree_dir`.

Run `hashi init` to generate a commented-out template.

## License

[MIT](LICENSE)
