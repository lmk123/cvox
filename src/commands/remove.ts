import {
  getSettingsPath,
  readSettings,
  writeSettings,
  removeHooks,
} from "../utils/settings.js";

export function removeCommand(options: { global?: boolean }): void {
  const isGlobal = options.global ?? false;
  const settingsPath = getSettingsPath(isGlobal);
  const settings = readSettings(settingsPath);
  const cleaned = removeHooks(settings);
  writeSettings(settingsPath, cleaned);

  const target = isGlobal ? "global" : "project";
  console.log(`cvox: Hooks removed from ${target} settings → ${settingsPath}`);
}
