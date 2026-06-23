import {
  getSettingsPath,
  readSettings,
  writeSettings,
  removeHooks,
} from "../utils/settings.js";
import { removeProjectConfig, loadConfig } from "../utils/config.js";
import * as os from "os";

export function removeCommand(options: { global?: boolean }): void {
  const isGlobal = options.global ?? false;

  if (isGlobal) {
    // Full uninstall: drop the machine-level hooks and the global config.
    const settingsPath = getSettingsPath(true);
    const settings = readSettings(settingsPath);
    const cleaned = removeHooks(settings);
    writeSettings(settingsPath, cleaned);

    const configRemoved = removeProjectConfig(os.homedir());

    console.log(`cvox: Hooks removed from global settings → ${settingsPath}`);
    if (configRemoved) {
      console.log(`cvox: Config file removed → ${os.homedir()}/.cvox.json`);
    }
    return;
  }

  // Project-level opt-out: deleting .cvox.json silences this project (hooks
  // still fire machine-wide, but with no config tts/desktop default to false).
  // Also strip any legacy cvox hook left in this project's settings.local.json.
  const cwd = process.cwd();
  const localSettingsPath = getSettingsPath(false, cwd);
  const localSettings = readSettings(localSettingsPath);
  const localCleaned = removeHooks(localSettings);
  if (JSON.stringify(localCleaned) !== JSON.stringify(localSettings)) {
    writeSettings(localSettingsPath, localCleaned);
    console.log(`cvox: Removed legacy hooks from ${localSettingsPath}`);
  }

  const configRemoved = removeProjectConfig(cwd);
  if (configRemoved) {
    console.log(`cvox: Config file removed → ${cwd}/.cvox.json`);
    // The project override is gone, but a global ~/.cvox.json may still turn
    // notifications on (config merge order: defaults → ~/.cvox.json → project).
    // Re-check the effective config so we don't falsely claim silence.
    const effective = loadConfig(cwd);
    if (effective.tts.enabled || effective.desktop.enabled) {
      console.log(
        "cvox: Note: ~/.cvox.json still enables notifications globally, so this project will keep notifying."
      );
    } else {
      console.log("cvox: This project is now silent.");
    }
  } else {
    console.log("cvox: No project .cvox.json found.");
  }
  console.log(
    "cvox: To uninstall the global hooks entirely, run: cvox remove --global"
  );
}
