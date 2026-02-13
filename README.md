# cvox

[![npm version](https://img.shields.io/npm/v/cvox.svg)](https://www.npmjs.com/package/cvox)

Voice notifications for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) hooks. Get spoken alerts when Claude needs permission or finishes a task — so you can step away from the screen.

## Features

- Cross-platform TTS: macOS (`say`), Linux (`espeak`), Windows (SAPI via PowerShell)
- Two hook events: permission prompt and task completion
- Three-layer config merging: defaults → `~/.cvox.json` → project `.cvox.json`
- Idempotent installation — safe to run multiple times
- `{project}` placeholder in messages, auto-detected from directory name

## Quick Start

```bash
# One-liner: install hooks into your project
npx cvox init

# Or install globally
npm install -g cvox
cvox init

# Install hooks globally (applies to all projects)
cvox init --global
```

That's it. Claude Code will now speak to you when it needs attention.

## Configuration

Create a `.cvox.json` in your project root or home directory (`~/.cvox.json`) to customize behavior:

```json
{
  "project": "my-app",
  "hooks": {
    "notification": {
      "enabled": true,
      "message": "{project} needs permission"
    },
    "stop": {
      "enabled": true,
      "message": "{project} task complete"
    }
  },
  "tts": {
    "enabled": true
  }
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `project` | string | directory name | Project name used in `{project}` placeholder |
| `hooks.notification.enabled` | boolean | `true` | Enable voice alert on permission prompts |
| `hooks.notification.message` | string | `"{project} needs permission"` | Message spoken on permission prompt |
| `hooks.stop.enabled` | boolean | `true` | Enable voice alert on task completion |
| `hooks.stop.message` | string | `"{project} task completed"` | Message spoken on task completion |
| `tts.enabled` | boolean | `true` | Enable/disable TTS globally |

Config files are merged with deep merge — you only need to specify the fields you want to override.

## How It Works

1. `cvox init` injects hooks into Claude Code's `settings.json`
2. When Claude Code triggers a hook event (permission prompt or stop), it pipes a JSON payload via stdin to `cvox notify`
3. `cvox notify` reads the event, loads your config, and calls the platform TTS engine to speak the message

## License

MIT
