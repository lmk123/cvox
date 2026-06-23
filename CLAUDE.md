# CLAUDE.md

此文件为 Claude Code (claude.ai/code) 在此代码库中工作时提供指导。

## 项目

cvox (Claude Voice Notifications) — 一个 Go CLI 工具，通过 Claude Code hooks 系统提供语音提醒和桌面通知。当出现权限提示或任务完成时，cvox 会语音播报和/或弹出桌面通知。

## 构建

```bash
make build    # 编译到 bin/cvox-bin（与 npm 安装路径一致）
make install  # 安装到 ~/go/bin（本地开发用，使用当前工作目录代码）
make test     # 运行所有测试
make clean    # 清理编译产物

# 或者直接使用 go 命令
go build -o bin/cvox-bin .   # 编译二进制
go test ./...               # 运行所有测试
bash test.sh                # 冒烟测试 notify 命令
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

## 架构

CLI 是单个 Go 二进制，有三个核心命令：

- `cvox init [--global]` — 将 hooks 注入到全局 `~/.claude/settings.json`（始终是全局的，见关键设计），交互式选择语言和通知方式；`--global` 仅决定 `.cvox.json` 的写入位置（家目录 vs 项目根目录）
- `cvox notify` — 由 hooks 调用，读取 stdin JSON 事件并触发 TTS 语音和/或桌面通知
- `cvox remove [--global]` — 默认删除项目的 `.cvox.json`；如果用户用的是项目级 `cvox init`（无全局配置），这会让项目静音；如果用户用的是 `cvox init --global`（有全局 `~/.cvox.json`），删除项目配置不会让项目静音（全局配置仍生效）。`--global` 完全卸载（删除全局 hooks + `~/.cvox.json`）

### 源码结构

- `main.go` — CLI 入口：子命令分发和 --version
- `internal/cli/` — 命令实现
  - `init.go` — Hook 安装逻辑：hooks 始终写入全局 `~/.claude/settings.json`，深度合并以避免覆盖已有配置；`--global` 仅决定 `.cvox.json` 位置（家目录 vs 项目根目录）；懒清理旧的项目级 `settings.local.json` hooks；交互式选择语言和通知方式（语音/桌面/两者）。两个提示都额外提供「Inherit」选项（列在末尾、非默认）：选中即不写入对应字段，回退到父层。提示里 `Inherit (X)` 的 X 由 `config.LoadForInherit(global)` 计算 —— 只看父层（非 global=默认值+`~/.cvox.json`，global=仅默认值），不含正要被覆盖的目标文件
  - `notify.go` — 事件处理、TTS 调用、桌面通知调用；包含内置静音名单 `MUTED_NOTIFICATION_TOOLS`（见关键设计）。opt-in 通过配置默认值实现，`notify` 本身不包含"文件存在"检查
  - `remove.go` — 移除逻辑：默认删除项目 `.cvox.json` + 清理旧的项目 cvox hooks，不触碰全局 hooks；`--global` 删除全局 hooks + `~/.cvox.json`
- `internal/hooks/` — Hook 定义（`hooks.go`）：权限提示只挂载 PermissionRequest（matcher `""`）— CLI 和 Desktop 权限对话框都触发 PermissionRequest；CLI 额外触发 Notification（matcher `permission_prompt`）而 Desktop 不触发，所以已弃用以避免 CLI 双响；stop 对应任务完成
- `internal/config/` — 三层配置合并（`config.go`）：默认值 → `~/.cvox.json` → 项目 `.cvox.json`；`tts.enabled`/`desktop.enabled` 默认为 `false`（见关键设计"通过默认值 opt-in"）。`Load` 合并全部三层；`LoadForInherit(global)` 只合并父层（非 global=默认值+`~/.cvox.json`，global=仅默认值），供 init 的「Inherit (X)」提示展示继承值用。`WriteProject` 的 `ProjectInput` 字段是指针类型，nil 表示「继承」—— 会删除已有 key 并清理空对象
- `internal/settings/` — Claude settings.json 读写（`settings.go`）：`Read` 区分"文件缺失"（ENOENT → 返回 `{}`，正常首次运行）vs"文件存在但不可读/JSON 损坏/根不是对象"（抛出 `SettingsParseError`，从不返回 `{}`）— 这防止"空对象覆盖用户配置"；`Write` 是原子的（写 `.tmp` 然后 `rename`）以防止半损坏文件（见关键设计"永不覆盖损坏的 settings.json"）
- `internal/notify/` — 平台特定的通知实现
  - `tts.go` — TTS 引擎检测和命令构建（`say`/`espeak`/SAPI PowerShell）
  - `desktop.go` — 桌面通知命令构建（`osascript`/`notify-send`/PowerShell NotifyIcon）
  - `mute.go` — 通知路径的内置静音名单（`MUTED_NOTIFICATION_TOOLS`），glob 模式匹配（仅 `*` 特殊，`!` 前缀否定，后匹配项胜出）
  - `notify.go` — 事件处理编排：stdin 读取、事件映射、静音检查、语音+桌面通知的并发分发

### 关键设计决策

- **Hooks 始终是全局的**：Hooks 始终写入 `~/.claude/settings.json`（机器级别），而不是项目的 `.claude/settings.local.json`。原因：`settings.local.json` 被 git 忽略，而 `git worktree` 只检出跟踪的文件，所以写入那里的 hooks 会在新 worktree 中丢失。全局安装一次即可覆盖所有项目和所有 worktree；"哪个项目说话"由提交的 `.cvox.json` 控制（它会随 worktree 一起移动）。`init` 懒清理旧的项目级 hooks 以避免主检出中的"全局 + 本地"双响。
- **永不覆盖损坏的 settings.json**：升级 hooks 到全局安装后，写入目标从项目级 `settings.local.json` 改为机器级共享的 `~/.claude/settings.json`（包含用户的权限/环境/模型/其他 hooks）。所有写入路径都先 `Read` 再整体 `Write`，所以如果 `Read` 把"文件存在但 JSON 损坏"吞成 `{}`，就会用只包含 cvox hooks 的对象覆盖用户的整个全局配置。防御：`Read` 只对 ENOENT 返回 `{}`；损坏/不可读/根不是对象都抛出 `SettingsParseError`。命令捕获此错误 — `init` 在交互提示前读取全局设置，损坏时打印路径 + 消息并以退出码 1 退出（不交互，不写入）；`remove --global` 在损坏时也中止。项目级 `settings.local.json` 懒清理是尽力而为：损坏只跳过 + 警告，不阻塞主流程。结合原子 `Write`，确保用户的 settings.json 永不被损坏。
- **通过默认值 opt-in**：`DEFAULT_CONFIG` 中 `tts.enabled`/`desktop.enabled` 都默认为 `false`。没有 `.cvox.json` 的项目合并后两者都为 false，`speak`/`desktopNotify` 中的早期返回（通过 `dispatch`）→ 静音。`init` 的默认选项（English + Voice only）会显式写入 `tts.enabled: true`，所以一路回车的项目正常说话。`init` 另提供「Inherit」选项（非默认）：选中即不写入对应字段，回退到父层。`WriteProject` 的 `ProjectInput` 字段是指针类型（`*string`/`*bool`），nil 表示「继承」—— 此时 `WriteProject` 会删除文件中已有的该字段，并清理因删除而变空的父对象（`hooks.notification`/`hooks.stop`/`hooks`/`tts`/`desktop`），确保重跑 init 选 Inherit 能真正生效。注意：手写的、存在但缺少 `tts` 的最小 `.cvox.json` 会静音（无写入 = 无声音）。
- **跨平台 TTS**：macOS `say` / Linux `espeak` / Windows SAPI via PowerShell
- **跨平台桌面通知**：macOS `osascript` / Linux `notify-send` / Windows PowerShell NotifyIcon
- **配置消息支持 `{project}` 占位符**
- **内置静音名单**（`mute.go`）：`PermissionRequest` hook 在工具"进入权限流程"时触发，早于实际对话框出现。Claude Desktop 的 Preview 工具会被自动批准、从不显示对话框，但 hook 仍然触发，导致多余语音。hook 输入没有字段能区分"真实确认 vs 自动批准"（只有 `tool_name`/`tool_input`/`permission_suggestions`，且 `permission_suggestions` 存在与否与对话框出现不相关，甚至会反转），所以通过 `tool_name` 匹配名单静音。语法：单数组，仅 `*` 通配符，`!` 前缀否定，后项覆盖前项（镜像 Claude Code 权限规则）。当前覆盖 `mcp__Claude_Preview__*` 并否定 `!mcp__Claude_Preview__preview_start`（该工具在 Claude Desktop 确实显示确认对话框，被通配符错误静音）。如果其他 Preview 工具也显示对话框，添加 `"!mcp__Claude_Preview__preview_xxx"` 允许它们。仅影响通知（权限）路径，不影响 stop。对用户透明，不暴露为配置选项。
- **Hook 安装使用 marker 子字符串实现幂等性**
- **Go 特定**：settings 读写使用 `github.com/tidwall/gjson`/`sjson`/`pretty` 来逐字节保留用户的 settings.json 键顺序和未知字段，只编辑 `hooks` 键。这避免了标准库 `map[string]any` 在 JSON 序列化时按字母顺序重排键的行为，这会产生嘈杂的 diff。

### 语言规范

用户可见内容（CLI 输出、help 文本、README.md）使用英文。

### 工作流规范

修改代码后，确认 CLAUDE.md 和 README.md 是否需要更新。
