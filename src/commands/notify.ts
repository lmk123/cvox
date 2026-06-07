import { execFile, ExecFileException } from "child_process";
import * as path from "path";
import { loadConfig, CvoxConfig } from "../utils/config.js";

function handleExecError(err: ExecFileException | null): void {
  if (err) {
    process.stderr.write(`cvox: ${err.message}\n`);
  }
}

interface HookInput {
  hook_event_name?: string;
  cwd?: string;
  tool_name?: string;
}

// Built-in mute list for the notification (PermissionRequest) path.
// Claude Desktop's Preview tools are auto-approved without ever showing a
// permission dialog, so a PermissionRequest hook fires (and cvox would speak)
// even though the user is never actually prompted. Mute the whole Preview
// namespace to kill that spurious sound.
//
// Patterns: only `*` is special (matches any run of chars, incl. empty). A
// leading `!` negates (un-mutes). Patterns are evaluated in order; the last
// one that matches a tool name wins — so add e.g.
// "!mcp__Claude_Preview__preview_eval" after the wildcard to re-enable the
// sound for a tool that turns out to really prompt.
const MUTED_NOTIFICATION_TOOLS: string[] = [
  "mcp__Claude_Preview__*",
];

function globToRegExp(pattern: string): RegExp {
  // Escape regex metachars (not `*`), then turn the literal `*` into `.*`.
  const escaped = pattern
    .replace(/[.+?^${}()|[\]\\]/g, "\\$&")
    .replace(/\*/g, ".*");
  return new RegExp(`^${escaped}$`);
}

function isToolMuted(toolName: string): boolean {
  let muted = false;
  for (const pattern of MUTED_NOTIFICATION_TOOLS) {
    const negated = pattern.startsWith("!");
    const body = negated ? pattern.slice(1) : pattern;
    if (globToRegExp(body).test(toolName)) {
      muted = !negated; // last matching pattern wins
    }
  }
  return muted;
}

function readStdin(): Promise<string> {
  return new Promise((resolve) => {
    if (process.stdin.isTTY) {
      resolve("");
      return;
    }
    let data = "";
    process.stdin.setEncoding("utf-8");
    process.stdin.on("data", (chunk) => (data += chunk));
    process.stdin.on("end", () => resolve(data));
  });
}

type EventKey = "notification" | "stop";

function mapEventName(hookEventName: string): EventKey | null {
  const map: Record<string, EventKey> = {
    // PermissionRequest fires on permission prompts in both the CLI and the
    // Claude Desktop app; it maps to the "notification" config.
    PermissionRequest: "notification",
    Stop: "stop",
  };
  return map[hookEventName] ?? null;
}

function speak(message: string, config: CvoxConfig): void {
  const { tts } = config;
  if (!tts.enabled) return;

  const engine = detectEngine();

  switch (engine) {
    case "say": {
      execFile("say", [message], handleExecError);
      break;
    }
    case "espeak": {
      execFile("espeak", [message], handleExecError);
      break;
    }
    case "sapi": {
      const ps = `Add-Type -AssemblyName System.Speech; ` +
        `$s = New-Object System.Speech.Synthesis.SpeechSynthesizer; ` +
        `$s.Speak($args[0])`;
      execFile("powershell", ["-Command", ps, message], handleExecError);
      break;
    }
  }
}

function desktopNotify(title: string, message: string, config: CvoxConfig): void {
  if (!config.desktop.enabled) return;

  switch (process.platform) {
    case "darwin": {
      execFile("osascript", [
        "-e", "on run argv",
        "-e", "display notification (item 2 of argv) with title (item 1 of argv)",
        "-e", "end run",
        "--", title, message,
      ], handleExecError);
      break;
    }
    case "win32": {
      const ps =
        `Add-Type -AssemblyName System.Windows.Forms; ` +
        `$n = New-Object System.Windows.Forms.NotifyIcon; ` +
        `$n.Icon = [System.Drawing.SystemIcons]::Information; ` +
        `$n.Visible = $true; ` +
        `$n.ShowBalloonTip(5000, $args[0], $args[1], 'Info')`;
      execFile("powershell", ["-Command", ps, title, message], handleExecError);
      break;
    }
    default: {
      execFile("notify-send", [title, message], handleExecError);
      break;
    }
  }
}

function detectEngine(): "say" | "espeak" | "sapi" {
  switch (process.platform) {
    case "darwin":
      return "say";
    case "win32":
      return "sapi";
    default:
      return "espeak";
  }
}

export async function notifyCommand(): Promise<void> {
  const raw = await readStdin();
  if (!raw.trim()) return;

  let input: HookInput;
  try {
    input = JSON.parse(raw);
  } catch {
    return;
  }

  const eventName = input.hook_event_name;
  if (!eventName) return;

  const eventKey = mapEventName(eventName);
  if (!eventKey) return;

  // Some tools (e.g. Claude Desktop's Preview tools) fire a PermissionRequest
  // hook but are auto-approved without ever prompting the user. Stay silent
  // for those to avoid spurious notification sounds.
  if (eventKey === "notification" && input.tool_name && isToolMuted(input.tool_name)) {
    return;
  }

  const cwd = input.cwd || process.cwd();
  const config = loadConfig(cwd);
  const hookConfig = config.hooks[eventKey];

  if (!hookConfig.enabled) return;

  const message = hookConfig.message.replace(
    /\{project\}/g,
    config.project
  );

  speak(message, config);
  desktopNotify("cvox", message, config);
}
