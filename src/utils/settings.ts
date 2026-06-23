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

/**
 * Thrown when a settings file exists but cannot be safely read or parsed.
 * Callers MUST treat this as "do not write" — overwriting would clobber a
 * config we failed to understand (e.g. a hand-edited ~/.claude/settings.json
 * with a syntax error would lose all the user's unrelated Claude config).
 */
export class SettingsParseError extends Error {
  constructor(public readonly filePath: string, public readonly cause: unknown) {
    super(
      `Failed to read settings at ${filePath}: ${
        cause instanceof Error ? cause.message : String(cause)
      }`
    );
    this.name = "SettingsParseError";
  }
}

/**
 * Read a settings JSON file.
 *
 * A missing file is the normal first-run case and yields `{}`. But a file that
 * EXISTS yet cannot be read or JSON-parsed throws {@link SettingsParseError}
 * instead of silently returning `{}` — otherwise the caller's subsequent
 * writeSettings would overwrite (and destroy) a config we never understood.
 */
export function readSettings(filePath: string): Record<string, any> {
  let content: string;
  try {
    content = fs.readFileSync(filePath, "utf-8");
  } catch (err) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") {
      return {}; // File doesn't exist yet — safe to start fresh.
    }
    throw new SettingsParseError(filePath, err); // Exists but unreadable.
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(content);
  } catch (err) {
    throw new SettingsParseError(filePath, err);
  }
  // A settings file must be a JSON object; anything else (array, string, null)
  // is not a shape we can safely merge into and write back.
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new SettingsParseError(filePath, new Error("settings root is not a JSON object"));
  }
  return parsed as Record<string, any>;
}

export function writeSettings(
  filePath: string,
  settings: Record<string, any>
): void {
  const dir = path.dirname(filePath);
  fs.mkdirSync(dir, { recursive: true });
  // Write atomically: serialize to a temp file in the same dir, then rename
  // over the target. A crash mid-write can't leave settings.json truncated /
  // corrupted (which would in turn make the next readSettings throw).
  const tmpPath = `${filePath}.${process.pid}.tmp`;
  fs.writeFileSync(tmpPath, JSON.stringify(settings, null, 2) + "\n");
  fs.renameSync(tmpPath, filePath);
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
  // First strip every existing cvox matcher (matched by the "cvox notify"
  // command marker, regardless of event name) so a re-run cleans up hooks that
  // newer versions no longer mount — e.g. the legacy Notification hook.
  const result = removeHooks(settings);
  const merged = { ...(result.hooks || {}) };

  for (const [eventName, cvoxMatchers] of Object.entries(cvoxHooks.hooks)) {
    const existing: any[] = merged[eventName] || [];
    merged[eventName] = [...existing, ...cvoxMatchers];
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
