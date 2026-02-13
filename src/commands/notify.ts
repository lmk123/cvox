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
    Notification: "notification",
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

function escapeAppleScript(str: string): string {
  return str.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

function desktopNotify(title: string, message: string, config: CvoxConfig): void {
  if (!config.desktop.enabled) return;

  switch (process.platform) {
    case "darwin": {
      const script = `display notification "${escapeAppleScript(message)}" with title "${escapeAppleScript(title)}"`;
      execFile("osascript", ["-e", script], handleExecError);
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
