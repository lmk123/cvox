# cvox

Voice notifications for [Claude Code](https://claude.com/product/claude-code) **and the [Claude Desktop](https://claude.ai/download) app**. Get spoken alerts or desktop notifications when Claude needs permission or finishes a task — so you can step away from the screen.

**Upgrading from v2?** The v3 rewrite is a Go binary with no runtime dependencies. Run `cvox remove --global` to uninstall the old Node version, then follow the install steps below.

## Features

- Works with **both** the Claude Code CLI and the Claude Desktop app — one setup covers both
- Cross-platform TTS: macOS (`say`), Linux (`espeak`), Windows (SAPI via PowerShell)
- Cross-platform desktop notifications: macOS (`osascript`), Linux (`notify-send`), Windows (PowerShell NotifyIcon)
- Interactive setup: choose language and notification method (voice, desktop, or both); each offers an "Inherit" option to fall back to the parent layer (defaults → `~/.cvox.json`)
- Two hook events: permission prompt and task completion
- Three-layer config merging: defaults → `~/.cvox.json` → project `.cvox.json`
- Idempotent installation — safe to run multiple times
- `{project}` placeholder in messages, auto-detected from directory name
- **No runtime dependencies** — single Go binary (~3 MB)

## Quick Start

```bash
# Install via npm (requires Node.js 18+)
npm install -g cvox

# Or install the binary directly (macOS/Linux/Windows)
curl -fsSL https://raw.githubusercontent.com/lmk123/cvox/main/install.sh | sh

# Or download a binary from the Releases page:
# https://github.com/lmk123/cvox/releases

# Set up for your project (run inside the project directory)
cvox init

# Or enable it for every project at once
cvox init --global
```

cvox always installs its hooks into the **global** `~/.claude/settings.json`, so a
single `cvox init` covers every project and every git worktree on your machine.
Whether a given project actually speaks is controlled by a `.cvox.json` file:

- `cvox init` writes `.cvox.json` to the **project root** → only this project
  speaks. Because `.cvox.json` is committed to git, it travels with every git
  worktree of the project, so notifications keep working there too.
- `cvox init --global` writes `.cvox.json` to your **home directory** → every
  project speaks.

A project with no `.cvox.json` (neither in its root nor in your home dir) stays
silent — voice and desktop notifications both default to off.

## Uninstall

```bash
# Silence the current project (deletes its .cvox.json; hooks stay installed)
cvox remove

# Fully uninstall: remove the global hooks and ~/.cvox.json
cvox remove --global
```

## Configuration

Create a `.cvox.json` in your project root or home directory (`~/.cvox.json`) to customize behavior:

```json
{
  "project": "my app",
  "hooks": {
    "notification": {
      "enabled": true,
      "message": "Claude Code needs permission, from {project}"
    },
    "stop": {
      "enabled": true,
      "message": "Claude Code task completed, from {project}"
    }
  },
  "tts": {
    "enabled": true
  },
  "desktop": {
    "enabled": false
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `project` | string | directory name | Project name used in `{project}` placeholder |
| `hooks.notification.enabled` | boolean | `true` | Enable alert on permission prompts |
| `hooks.notification.message` | string | `"Claude Code needs permission, from {project}"` | Message for permission prompt |
| `hooks.stop.enabled` | boolean | `true` | Enable alert on task completion |
| `hooks.stop.message` | string | `"Claude Code task completed, from {project}"` | Message for task completion |
| `tts.enabled` | boolean | `false` | Enable/disable TTS voice globally |
| `desktop.enabled` | boolean | `false` | Enable/disable desktop notifications globally |

Config files are merged with deep merge — you only need to specify the fields you want to override. Both `tts.enabled` and `desktop.enabled` default to `false`, so a project only speaks once a `.cvox.json` turns one of them on (which `cvox init` does for you when you choose a concrete option). When running `cvox init`, both the language and notification method prompts offer an "Inherit" choice that omits the corresponding field from the `.cvox.json`, letting it fall back to the parent layer (defaults → `~/.cvox.json`).

## How It Works

1. `cvox init` injects hooks into the global `~/.claude/settings.json` shared by both the Claude Code CLI and the Claude Desktop app. Installing once covers every project and every git worktree.
2. When either surface triggers a hook event (permission prompt or stop), it pipes a JSON payload via stdin to `cvox notify`.
3. `cvox notify` reads the event, loads your config (defaults → `~/.cvox.json` → project `.cvox.json`), and calls the platform TTS engine and/or desktop notification. With no config file present, voice and desktop both default to off, so the project stays silent.

### Why the hooks live in the global settings file

Earlier versions wrote hooks to the project's `.claude/settings.local.json`. That file is git-ignored, and `git worktree` only checks out tracked files — so new worktrees lost the hook and went silent. Installing the hook globally (machine-level) avoids this entirely, while the committed `.cvox.json` travels with each worktree to decide whether that project speaks.

> **Upgrading?** Just run `cvox init` once to install the global hooks — all your existing projects (which already have a committed `.cvox.json`) start working immediately, worktrees included. If an old project's main checkout double-speaks, it still has a legacy hook in its `settings.local.json`; re-running `cvox init` in that project removes it.

## License

MIT
