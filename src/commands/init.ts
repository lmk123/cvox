import { generateHooksConfig } from "../hooks/config.js";
import {
  getSettingsPath,
  readSettings,
  writeSettings,
  mergeHooks,
} from "../utils/settings.js";

export function initCommand(options: { global?: boolean }): void {
  const isGlobal = options.global ?? false;
  const settingsPath = getSettingsPath(isGlobal);
  const settings = readSettings(settingsPath);
  const cvoxHooks = generateHooksConfig();
  const merged = mergeHooks(settings, cvoxHooks);

  writeSettings(settingsPath, merged);

  const target = isGlobal ? "全局" : "项目";
  console.log(`cvox: 已配置 ${target} hooks → ${settingsPath}`);
  console.log("  - Notification (permission_prompt)");
  console.log("  - Stop");
}
