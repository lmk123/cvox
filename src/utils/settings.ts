import * as fs from "fs";
import * as path from "path";
import * as os from "os";
import { CvoxHooksConfig } from "../hooks/config.js";

const CVOX_MARKER = "cvox notify";

export function getSettingsPath(global: boolean, cwd?: string): string {
  if (global) {
    return path.join(os.homedir(), ".claude", "settings.json");
  }
  return path.join(cwd || process.cwd(), ".claude", "settings.local.json");
}

export function readSettings(filePath: string): Record<string, any> {
  try {
    const content = fs.readFileSync(filePath, "utf-8");
    return JSON.parse(content);
  } catch {
    return {};
  }
}

export function writeSettings(
  filePath: string,
  settings: Record<string, any>
): void {
  const dir = path.dirname(filePath);
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(filePath, JSON.stringify(settings, null, 2) + "\n");
}

function isCvoxHook(hook: any): boolean {
  return (
    hook &&
    typeof hook.command === "string" &&
    hook.command.includes(CVOX_MARKER)
  );
}

function isCvoxMatcher(matcher: any): boolean {
  return (
    matcher &&
    Array.isArray(matcher.hooks) &&
    matcher.hooks.some(isCvoxHook)
  );
}

export function mergeHooks(
  settings: Record<string, any>,
  cvoxHooks: CvoxHooksConfig
): Record<string, any> {
  const result = { ...settings };
  const existingHooks = result.hooks || {};
  const merged = { ...existingHooks };

  for (const [eventName, cvoxMatchers] of Object.entries(cvoxHooks.hooks)) {
    const existing: any[] = merged[eventName] || [];
    const filtered = existing.filter((m: any) => !isCvoxMatcher(m));
    merged[eventName] = [...filtered, ...cvoxMatchers];
  }

  result.hooks = merged;
  return result;
}

export function removeHooks(settings: Record<string, any>): Record<string, any> {
  const result = { ...settings };
  if (!result.hooks) return result;

  const hooks = { ...result.hooks };
  for (const eventName of Object.keys(hooks)) {
    if (Array.isArray(hooks[eventName])) {
      hooks[eventName] = hooks[eventName].filter((m: any) => !isCvoxMatcher(m));
      if (hooks[eventName].length === 0) {
        delete hooks[eventName];
      }
    }
  }

  if (Object.keys(hooks).length === 0) {
    delete result.hooks;
  } else {
    result.hooks = hooks;
  }

  return result;
}
