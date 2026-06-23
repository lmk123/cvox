import * as readline from "readline";
import * as path from "path";
import * as os from "os";
import { generateHooksConfig } from "../hooks/config.js";
import {
  getSettingsPath,
  readSettings,
  writeSettings,
  mergeHooks,
  removeHooks,
  SettingsParseError,
} from "../utils/settings.js";
import { LOCALE_MESSAGES, writeProjectConfig } from "../utils/config.js";

function prompt(rl: readline.Interface, question: string): Promise<string> {
  return new Promise((resolve) => {
    rl.question(question, (answer) => resolve(answer.trim()));
  });
}

const LOCALE_OPTIONS = [
  { key: "1", code: "en", label: "English" },
  { key: "2", code: "zh", label: "中文" },
  { key: "3", code: "ja", label: "日本語" },
  { key: "4", code: "ko", label: "한국어" },
];

const NOTIFY_METHOD_OPTIONS = [
  { key: "1", label: "Voice only", tts: true, desktop: false },
  { key: "2", label: "Desktop notification only", tts: false, desktop: true },
  { key: "3", label: "Both voice and desktop", tts: true, desktop: true },
];

export async function initCommand(options: { global?: boolean }): Promise<void> {
  const cwd = process.cwd();
  const defaultName = path.basename(cwd);

  // Read the global settings up front (before prompting) so a corrupt
  // ~/.claude/settings.json aborts immediately rather than after the user has
  // answered every question. Crucially we NEVER write on a parse error — that
  // would overwrite (and destroy) the user's unrelated Claude config.
  const settingsPath = getSettingsPath(true);
  let settings: Record<string, any>;
  try {
    settings = readSettings(settingsPath);
  } catch (err) {
    if (err instanceof SettingsParseError) {
      console.error(`cvox: ${err.message}`);
      console.error(
        "cvox: Refusing to overwrite it. Please fix the JSON (or remove the file) and re-run cvox init."
      );
      process.exitCode = 1;
      return;
    }
    throw err;
  }

  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  try {
    // 问题 1：项目名称
    const nameAnswer = await prompt(rl, `Project name (default: ${defaultName}): `);
    const projectName = nameAnswer || defaultName;

    // 问题 2：语种选择
    console.log("Voice language:");
    for (const opt of LOCALE_OPTIONS) {
      const defaultMark = opt.key === "1" ? " (default)" : "";
      console.log(`  ${opt.key}. ${opt.label}${defaultMark}`);
    }
    const localeAnswer = await prompt(rl, "Select language [1]: ");
    const selected = LOCALE_OPTIONS.find((o) => o.key === localeAnswer) ?? LOCALE_OPTIONS[0];
    const messages = LOCALE_MESSAGES[selected.code];

    // 问题 3：通知方式
    console.log("Notification method:");
    for (const opt of NOTIFY_METHOD_OPTIONS) {
      const defaultMark = opt.key === "1" ? " (default)" : "";
      console.log(`  ${opt.key}. ${opt.label}${defaultMark}`);
    }
    const methodAnswer = await prompt(rl, "Select method [1]: ");
    const selectedMethod = NOTIFY_METHOD_OPTIONS.find((o) => o.key === methodAnswer) ?? NOTIFY_METHOD_OPTIONS[0];

    // 注入 hooks。hook 始终写进全局 ~/.claude/settings.json：它是机器级配置，
    // 一次安装即覆盖所有项目与所有 git worktree（worktree 不会 checkout 被 git
    // 忽略的 settings.local.json，hook 若写在那里会在 worktree 中丢失）。
    const isGlobal = options.global ?? false;
    const cvoxHooks = generateHooksConfig();
    const merged = mergeHooks(settings, cvoxHooks);
    writeSettings(settingsPath, merged);

    // 懒清理：早期版本把 hook 写进项目 .claude/settings.local.json。若本项目仍有
    // 残留，主 checkout 会「全局 + 本地」双触发导致语音念两遍（worktree 中无此文件
    // 故不受影响）。这里顺手剥掉本项目残留的 cvox hook（按 marker，只动 cvox 自己的）。
    // best-effort：local settings 损坏时跳过清理并警告，不影响已完成的全局安装。
    const localSettingsPath = getSettingsPath(false, cwd);
    let hadLocalHook = false;
    try {
      const localSettings = readSettings(localSettingsPath);
      const localCleaned = removeHooks(localSettings);
      hadLocalHook =
        JSON.stringify(localCleaned) !== JSON.stringify(localSettings);
      if (hadLocalHook) {
        writeSettings(localSettingsPath, localCleaned);
      }
    } catch (err) {
      if (err instanceof SettingsParseError) {
        console.error(
          `cvox: Skipped legacy-hook cleanup — ${localSettingsPath} is not valid JSON. Fix it manually if this project double-speaks.`
        );
      } else {
        throw err;
      }
    }

    // 生成 .cvox.json：--global 写到家目录（所有项目都响），否则写到项目根
    // （仅本项目响，且因 .cvox.json 入库而随 worktree 走）。
    const configDir = isGlobal ? os.homedir() : cwd;
    writeProjectConfig(configDir, {
      project: projectName,
      hooks: {
        notification: { message: messages.notification },
        stop: { message: messages.stop },
      },
      tts: { enabled: selectedMethod.tts },
      desktop: { enabled: selectedMethod.desktop },
    });

    const scope = isGlobal ? "all projects" : projectName;
    console.log(`cvox: Hooks installed globally → ${settingsPath}`);
    console.log("  - Notify on permission prompt");
    console.log("  - Notify on task completion");
    if (hadLocalHook) {
      console.log(`cvox: Removed legacy hooks from ${localSettingsPath}`);
    }
    console.log(
      `cvox: Generated .cvox.json for ${scope} → ${configDir}/.cvox.json (${selected.label}, ${selectedMethod.label})`
    );
  } finally {
    rl.close();
  }
}
