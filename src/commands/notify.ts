import { execFile } from "child_process";
import * as path from "path";
import { loadConfig, CvoxConfig } from "../utils/config.js";

interface HookInput {
  hook_event_name?: string;
  cwd?: string;
}

function readStdin(): Promise<string> {
  return new Promise((resolve) => {
    let data = "";
    process.stdin.setEncoding("utf-8");
    process.stdin.on("data", (chunk) => (data += chunk));
    process.stdin.on("end", () => resolve(data));
    if (process.stdin.isTTY) {
      resolve("");
    }
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
      execFile("say", [message], () => {});
      break;
    }
    case "espeak": {
      execFile("espeak", [message], () => {});
      break;
    }
    case "sapi": {
      const ps = `Add-Type -AssemblyName System.Speech; ` +
        `$s = New-Object System.Speech.Synthesis.SpeechSynthesizer; ` +
        `$s.Speak('${message.replace(/'/g, "''")}')`;
      execFile("powershell", ["-Command", ps], () => {});
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
      execFile("osascript", ["-e", script], () => {});
      break;
    }
    case "win32": {
      const ps =
        `Add-Type -AssemblyName System.Windows.Forms; ` +
        `$n = New-Object System.Windows.Forms.NotifyIcon; ` +
        `$n.Icon = [System.Drawing.SystemIcons]::Information; ` +
        `$n.Visible = $true; ` +
        `$n.ShowBalloonTip(5000, '${title.replace(/'/g, "''")}', '${message.replace(/'/g, "''")}', 'Info')`;
      execFile("powershell", ["-Command", ps], () => {});
      break;
    }
    default: {
      execFile("notify-send", [title, message], () => {});
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
