import {
  getSettingsPath,
  readSettings,
  writeSettings,
  removeHooks,
  SettingsParseError,
} from "../utils/settings.js";
import { removeProjectConfig, loadConfig } from "../utils/config.js";
import * as os from "os";

export function removeCommand(options: { global?: boolean }): void {
  const isGlobal = options.global ?? false;

  if (isGlobal) {
    // Full uninstall: drop the machine-level hooks and the global config.
    const settingsPath = getSettingsPath(true);
    let settings: Record<string, any>;
    try {
      settings = readSettings(settingsPath);
    } catch (err) {
      if (err instanceof SettingsParseError) {
        console.error(`cvox: ${err.message}`);
        console.error(
          "cvox: Refusing to overwrite it. Fix the JSON (or remove the file) and re-run, or delete the cvox hooks by hand."
        );
        process.exitCode = 1;
        return;
      }
      throw err;
    }
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
  // best-effort：local settings 损坏时跳过清理并警告，不阻断后续的 .cvox.json 删除。
  const cwd = process.cwd();
  const localSettingsPath = getSettingsPath(false, cwd);
  try {
    const localSettings = readSettings(localSettingsPath);
    const localCleaned = removeHooks(localSettings);
    if (JSON.stringify(localCleaned) !== JSON.stringify(localSettings)) {
      writeSettings(localSettingsPath, localCleaned);
      console.log(`cvox: Removed legacy hooks from ${localSettingsPath}`);
    }
  } catch (err) {
    if (err instanceof SettingsParseError) {
      console.error(
        `cvox: Skipped legacy-hook cleanup — ${localSettingsPath} is not valid JSON.`
      );
    } else {
      throw err;
    }
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
