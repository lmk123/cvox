import {
  getSettingsPath,
  readSettings,
  writeSettings,
  removeHooks,
} from "../utils/settings.js";
import { removeProjectConfig } from "../utils/config.js";
import * as os from "os";

export function removeCommand(options: { global?: boolean }): void {
  const isGlobal = options.global ?? false;
  const settingsPath = getSettingsPath(isGlobal);
  const settings = readSettings(settingsPath);
  const cleaned = removeHooks(settings);
  writeSettings(settingsPath, cleaned);

  const configDir = isGlobal ? os.homedir() : process.cwd();
  const configRemoved = removeProjectConfig(configDir);

  const target = isGlobal ? "global" : "project";
  console.log(`cvox: Hooks removed from ${target} settings → ${settingsPath}`);
  if (configRemoved) {
    console.log(`cvox: Config file removed → ${configDir}/.cvox.json`);
  }
}
