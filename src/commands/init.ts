import * as readline from "readline";
import * as path from "path";
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

    rl.close();

    // 注入 hooks 到 Claude Code settings
    const isGlobal = options.global ?? false;
    const settingsPath = getSettingsPath(isGlobal);
    const settings = readSettings(settingsPath);
    const cvoxHooks = generateHooksConfig();
    const merged = mergeHooks(settings, cvoxHooks);
    writeSettings(settingsPath, merged);

    // 生成 .cvox.json
    writeProjectConfig(cwd, {
      project: projectName,
      hooks: {
        notification: { message: messages.notification },
        stop: { message: messages.stop },
      },
    });

    const target = isGlobal ? "global" : "project";
    console.log(`cvox: Configured ${target} hooks → ${settingsPath}`);
    console.log("  - Notification (permission_prompt)");
    console.log("  - Stop");
    console.log(`cvox: Generated .cvox.json (${selected.label})`);
  } finally {
    rl.close();
  }
}
