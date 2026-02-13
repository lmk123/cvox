# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

cvox (Claude Voice Notifications) — 一个 CLI 工具，通过 Claude Code hooks 系统提供语音提醒和桌面通知。当出现权限提示或任务完成时，通过 TTS 语音和/或系统原生桌面弹窗通知用户。

## Build

```bash
npm run build    # tsc 编译 TypeScript → dist/
npm link         # 全局安装用于本地开发
```

无测试框架、无 linter 配置。

## Architecture

CLI 基于 Commander.js，两个核心命令：

- `cvox init [--global]` — 将 hooks 注入 Claude Code settings（项目级或全局），交互选择语言和通知方式
- `cvox notify` — 由 hooks 调用，读取 stdin JSON 事件并触发 TTS 语音和/或桌面通知

### 源码结构

- `src/index.ts` — CLI 入口
- `src/commands/init.ts` — hook 安装逻辑，支持 deep merge 避免覆盖已有配置，交互选择通知方式（语音/桌面/两者）
- `src/commands/notify.ts` — 事件处理、TTS 调用与桌面通知调用
- `src/hooks/config.ts` — hook 定义（notification 对应 permission_prompt，stop 对应任务完成）
- `src/utils/config.ts` — 三层配置合并：默认值 → `~/.cvox.json` → 项目 `.cvox.json`
- `src/utils/settings.ts` — Claude settings.json 读写

### 关键设计

- 跨平台 TTS：macOS `say` / Linux `espeak` / Windows SAPI PowerShell
- 跨平台桌面通知：macOS `osascript` / Linux `notify-send` / Windows PowerShell NotifyIcon
- 配置消息支持 `{project}` 占位符
- hook 安装使用 marker 标记实现幂等性
- TypeScript strict mode，CommonJS 输出，target ES2016

### 语言规范

用户可见内容（CLI 输出、help 信息、README.md 等）一律使用英文

### 工作流规范

修改代码之后，需要确认 CLAUDE.md 和 README.md 是否要更新
