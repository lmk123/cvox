import * as readline from "readline";
import * as path from "path";
import * as os from "os";
import { generateHooksConfig } from "../hooks/config.js";
import {
  getSettingsPath,
  readSettings,
  writeSettings,
  mergeHooks,
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

    // 注入 hooks 到 Claude Code settings
    const isGlobal = options.global ?? false;
    const settingsPath = getSettingsPath(isGlobal);
    const settings = readSettings(settingsPath);
    const cvoxHooks = generateHooksConfig();
    const merged = mergeHooks(settings, cvoxHooks);
    writeSettings(settingsPath, merged);

    // 生成 .cvox.json
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

    const target = isGlobal ? "global" : "project";
    console.log(`cvox: Setup complete (${target}) → ${settingsPath}`);
    console.log("  - Notify on permission prompt");
    console.log("  - Notify on task completion");
    console.log(`cvox: Generated .cvox.json (${selected.label}, ${selectedMethod.label})`);
  } finally {
    rl.close();
  }
}
