# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

cvox (Claude Voice Notifications) — a Go CLI tool that provides voice alerts and desktop notifications for Claude Code via the hooks system. When a permission prompt appears or a task completes, cvox speaks and/or pops up a desktop notification.

## Build

```bash
go build -o cvox .          # Build the binary
go test ./...               # Run all tests
bash test.sh                # Smoke test the notify command
```

## Hook 探针（调试工具）

排查「某个权限框/事件到底触发了哪个 hook」时用，避免靠耳朵猜（agent 沙箱挡音频、改 hook 没重启等假象）。

```bash
node scripts/probe-toggle.cjs install          # 把探针注入项目 .claude/settings.local.json（10 个 hook 事件，全 matcher），默认精简模式
node scripts/probe-toggle.cjs install --full   # 同上，但完整模式：dump 整个 stdin JSON（含 tool_input）到 probe-full.log
node scripts/probe-toggle.cjs uninstall        # 只移除探针，保留 cvox 和用户自定义 hook
node scripts/probe-toggle.cjs status           # 查看当前装了几个探针 hook
```

- `probe.cjs` — 探针本体。默认模式纯记录 `hook_event_name` + `tool_name` 到 `probe.log`（行首 `[HH:MM:SS] Event Tool`），**不记录 tool_input/command/message** 以避免日志自污染；带 `--full`（即 `probe:install:full`）则 dump 整个 stdin JSON 到 `probe-full.log`，排查「事件到底携带哪些字段」时用。日志路径可用 `CVOX_PROBE_LOG` 覆盖。
- `scripts/probe-toggle.cjs` — install/uninstall/status 逻辑。marker 用命令里含子串 `probe.cjs` 识别（完整模式命令 `probe.cjs --full` 同样匹配），故一次 uninstall 即可清掉任意模式的探针（与 cvox 的 marker `cvox notify` 互不干扰）；install 幂等，且会先清旧探针再装，故 full/simple 间可直接 re-install 切换。settings 路径可用 `CVOX_PROBE_SETTINGS` 覆盖（测试用）。
- `probe.log` / `probe-full.log` 已加入 `.gitignore`。
- ⚠️ **改完 settings 需重启 Claude Desktop / 新开会话才生效**；在 agent 的 Bash 沙箱里跑 install 会 EPERM（settings.local.json 受保护），需在普通终端执行。

## Architecture

CLI is a single Go binary with three core commands:

- `cvox init [--global]` — Injects hooks into the global `~/.claude/settings.json` (always global, see key design below), interactively selects language and notification method; `--global` only decides where `.cvox.json` is written (home directory vs project root)
- `cvox notify` — Called by hooks, reads stdin JSON event and triggers TTS voice and/or desktop notification
- `cvox remove [--global]` — By default only deletes the project `.cvox.json` to silence this project (hooks remain global); `--global` fully uninstalls (deletes global hooks + `~/.cvox.json`)

### Source structure

- `main.go` — CLI entry point: subcommand dispatch and --version
- `internal/cli/` — Command implementations
  - `init.go` — Hook installation logic: hooks always written to global `~/.claude/settings.json`, deep merge to avoid overwriting existing config; `--global` only determines `.cvox.json` location (home vs project root); lazy cleanup of legacy project-level `settings.local.json` hooks; interactive selection of notification method (voice/desktop/both)
  - `notify.go` — Event handling, TTS invocation, desktop notification invocation; includes built-in mute list `MUTED_NOTIFICATION_TOOLS` (see key design below). opt-in is implemented via config defaults, `notify` itself contains no "file exists" check
  - `remove.go` — Removal logic: default (project-level) deletes project `.cvox.json` + cleans up legacy project cvox hooks, does not touch global hooks; `--global` deletes global hooks + `~/.cvox.json`
- `internal/hooks/` — Hook definitions (`hooks.go`): permission prompt only mounts PermissionRequest (matcher `""`) — both CLI and Desktop permission dialogs trigger PermissionRequest; CLI additionally triggers Notification (matcher `permission_prompt`) which Desktop does not, so it's been dropped to avoid CLI double-speak; stop corresponds to task completion
- `internal/config/` — Three-layer config merging (`config.go`): defaults → `~/.cvox.json` → project `.cvox.json`; `tts.enabled`/`desktop.enabled` default to `false` (see key design "opt-in via defaults")
- `internal/settings/` — Claude settings.json read/write (`settings.go`): `Read` distinguishes "file missing" (ENOENT → returns `{}`, normal first-run) vs "file exists but unreadable/JSON corrupt/root not object" (throws `SettingsParseError`, never returns `{}`) — this prevents "empty object overwriting user config" at the source; `Write` is atomic (write `.tmp` then `rename`) to prevent half-corrupted files (see key design "never overwrite corrupt settings.json")
- `internal/notify/` — Platform-specific notification implementations
  - `tts.go` — TTS engine detection and command building (`say`/`espeak`/SAPI PowerShell)
  - `desktop.go` — Desktop notification command building (`osascript`/`notify-send`/PowerShell NotifyIcon)
  - `mute.go` — Built-in mute list for the notification path (`MUTED_NOTIFICATION_TOOLS`), glob pattern matching (only `*` special, `!` prefix negates, last match wins)
  - `notify.go` — Event processing orchestration: stdin read, event mapping, mute check, concurrent dispatch of voice+desktop notifications

### Key design decisions

- **Hooks are always global**: Hooks are always written to `~/.claude/settings.json` (machine-level), not the project's `.claude/settings.local.json`. Reason: `settings.local.json` is git-ignored, and `git worktree` only checks out tracked files, so hooks written there would be lost in new worktrees. Global installation once covers all projects and all worktrees; "which project speaks" is controlled by the committed `.cvox.json` (which travels with worktrees). `init` lazily cleans up legacy project-level hooks to avoid "global + local" double-speak in the main checkout.
- **Never overwrite corrupt settings.json**: After upgrading hooks to global installation, the write target changed from project-level `settings.local.json` to machine-level shared `~/.claude/settings.json` (which contains the user's permissions/env/model/other hooks). All write paths do `Read` then `Write` as a whole, so if `Read` swallowed "file exists but JSON corrupt" as `{}`, it would overwrite the user's entire global config with an object containing only cvox hooks. Defense: `Read` only returns `{}` for ENOENT; corrupt/unreadable/non-object root all throw `SettingsParseError`. Commands catch this — `init` reads global settings **before** interactive prompts, and on corruption prints the path + message and exits with code 1 (no interaction, no write); `remove --global` aborts on corruption too. Project-level `settings.local.json` lazy cleanup is best-effort: corruption only skips + warns, doesn't block the main flow. Combined with atomic `Write`, this ensures the user's settings.json is never damaged.
- **Opt-in via defaults**: `DEFAULT_CONFIG` has `tts.enabled`/`desktop.enabled` both default to `false`. A project with no `.cvox.json` merges down to both false, and the early returns in `speak`/`desktopNotify` (via `dispatch`) → silence. `init` always explicitly writes `tts.enabled` (Voice/Both=true, Desktop only=false), so projects that have run `init` speak normally. Note: a hand-written, existing but `tts`-less minimal `.cvox.json` will be silent (no write = no sound).
- **Cross-platform TTS**: macOS `say` / Linux `espeak` / Windows SAPI via PowerShell
- **Cross-platform desktop notifications**: macOS `osascript` / Linux `notify-send` / Windows PowerShell NotifyIcon
- **Config messages support `{project}` placeholder**
- **Built-in mute list** (`mute.go`): The `PermissionRequest` hook fires when a tool "enters the permission flow", before the actual dialog appears. Claude Desktop's Preview tools are auto-approved and never show a dialog, but the hook still fires, causing spurious voice. The hook input has no field to distinguish "real confirmation vs auto-approval" (only `tool_name`/`tool_input`/`permission_suggestions`, and `permission_suggestions` presence/absence doesn't correlate with dialog appearance, and can even be inverted), so we mute by `tool_name` matching a list. Syntax: single array, only `*` wildcard, `!` prefix negates, later items override earlier ones (mirrors Claude Code permission rules). Currently covers `mcp__Claude_Preview__*` with negation for `!mcp__Claude_Preview__preview_start` (that tool does show a confirmation dialog in Claude Desktop and was incorrectly silenced by the wildcard). If other Preview tools also show dialogs, add `"!mcp__Claude_Preview__preview_xxx"` to allow them. Only affects the notification (permission) path, not stop. User-transparent, not exposed as a config option.
- **Hook installation uses marker substring for idempotency**
- **Go-specific**: The settings read/write uses `github.com/tidwall/gjson`/`sjson`/`pretty` to preserve the user's settings.json key order and unknown fields byte-for-byte, only editing the `hooks` key. This avoids the standard library `map[string]any` behavior of reordering keys alphabetically on JSON serialization, which would produce noisy diffs.

### Language convention

User-facing content (CLI output, help text, README.md) is in English.

### Workflow convention

After modifying code, confirm whether CLAUDE.md and README.md need updates.
