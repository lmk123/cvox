# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

cvox (Claude Voice Notifications) — 一个 CLI 工具，通过 Claude Code hooks 系统提供语音提醒和桌面通知。当出现权限提示或任务完成时，通过 TTS 语音和/或系统原生桌面弹窗通知用户。

## Build

```bash
npm run build    # tsc 编译 TypeScript → dist/
npm run lint     # Typescript 类型检查
npm link         # 全局安装用于本地开发
```

无测试框架、无 linter 配置。

## Hook 探针（调试工具）

排查「某个权限框/事件到底触发了哪个 hook」时用，避免靠耳朵猜（agent 沙箱挡音频、改 hook 没重启等假象）。

```bash
npm run probe:install      # 把探针注入项目 .claude/settings.local.json（10 个 hook 事件，全 matcher），默认精简模式
npm run probe:install:full # 同上，但完整模式：dump 整个 stdin JSON（含 tool_input）到 probe-full.log
npm run probe:status       # 查看当前装了几个探针 hook
npm run probe:uninstall    # 只移除探针，保留 cvox 和用户自定义 hook
```

- `probe.cjs` — 探针本体。默认模式纯记录 `hook_event_name` + `tool_name` 到 `probe.log`（行首 `[HH:MM:SS] Event Tool`），**不记录 tool_input/command/message** 以避免日志自污染；带 `--full`（即 `probe:install:full`）则 dump 整个 stdin JSON 到 `probe-full.log`，排查「事件到底携带哪些字段」时用。日志路径可用 `CVOX_PROBE_LOG` 覆盖。
- `scripts/probe-toggle.cjs` — install/uninstall/status 逻辑。marker 用命令里含子串 `probe.cjs` 识别（完整模式命令 `probe.cjs --full` 同样匹配），故一次 uninstall 即可清掉任意模式的探针（与 cvox 的 marker `cvox notify` 互不干扰）；install 幂等，且会先清旧探针再装，故 full/simple 间可直接 re-install 切换。settings 路径可用 `CVOX_PROBE_SETTINGS` 覆盖（测试用）。
- `probe.log` / `probe-full.log` 已加入 `.gitignore`。
- ⚠️ **改完 settings 需重启 Claude Desktop / 新开会话才生效**；在 agent 的 Bash 沙箱里跑 install 会 EPERM（settings.local.json 受保护），需在普通终端执行。

## Architecture

CLI 基于 Commander.js，三个核心命令：

- `cvox init [--global]` — 将 hooks 注入 Claude Code settings（项目级或全局），交互选择语言和通知方式
- `cvox notify` — 由 hooks 调用，读取 stdin JSON 事件并触发 TTS 语音和/或桌面通知
- `cvox remove [--global]` — 从 Claude Code settings 中移除 cvox hooks

### 源码结构

- `src/index.ts` — CLI 入口
- `src/commands/init.ts` — hook 安装逻辑，支持 deep merge 避免覆盖已有配置，交互选择通知方式（语音/桌面/两者）
- `src/commands/notify.ts` — 事件处理、TTS 调用与桌面通知调用；含内置静音名单 `MUTED_NOTIFICATION_TOOLS`（见关键设计）
- `src/commands/remove.ts` — hook 移除逻辑
- `src/hooks/config.ts` — hook 定义（权限提示只挂 PermissionRequest(matcher `""`)：CLI 和 Desktop 权限框都触发 PermissionRequest；CLI 额外触发的 Notification(matcher `permission_prompt`) Desktop 不触发，故已弃用以避免 CLI 双响；stop 对应任务完成）
- `src/utils/config.ts` — 三层配置合并：默认值 → `~/.cvox.json` → 项目 `.cvox.json`
- `src/utils/settings.ts` — Claude settings.json 读写

### 关键设计

- 跨平台 TTS：macOS `say` / Linux `espeak` / Windows SAPI PowerShell
- 跨平台桌面通知：macOS `osascript` / Linux `notify-send` / Windows PowerShell NotifyIcon
- 配置消息支持 `{project}` 占位符
- **内置静音名单**（`notify.ts` 的 `MUTED_NOTIFICATION_TOOLS`）：`PermissionRequest` hook 在工具「进入权限流程」时就触发，早于「弹框」动作；Claude Desktop 的 Preview 类工具会被自动放行、根本不弹框，但 hook 照样触发导致多余语音。hook 输入无任何字段能区分「真要确认 vs 自动放行」（只有 `tool_name`/`tool_input`/`permission_suggestions`，且 `permission_suggestions` 有无与是否弹框无关、会判反），故按 `tool_name` 匹配名单静音。语法：单数组、仅 `*` 通配、`!` 前缀反排除、后项覆盖前项（贴近 Claude Code 权限规则）。当前罩 `mcp__Claude_Preview__*` 并反排除 `!mcp__Claude_Preview__preview_start`（该工具在 Claude Desktop 确会弹确认框，被通配符误伤）；其它 Preview 工具若也确会弹框，照此追加 `"!mcp__Claude_Preview__preview_xxx"` 放行。仅作用于 notification（权限）路径，不影响 stop。对用户无感知，不暴露为配置项。
- hook 安装使用 marker 标记实现幂等性
- TypeScript strict mode，ESM 输出（`"module": "node16"`），target ES2020

### 语言规范

用户可见内容（CLI 输出、help 信息、README.md 等）一律使用英文

### 工作流规范

修改代码之后，需要确认 CLAUDE.md 和 README.md 是否要更新
