#!/usr/bin/env node
// Install / uninstall the cvox hook probe into project Claude settings.
//
//   node scripts/probe-toggle.cjs install     (or: npm run probe:install)
//   node scripts/probe-toggle.cjs uninstall   (or: npm run probe:uninstall)
//   node scripts/probe-toggle.cjs status
//
// The probe is a command hook that runs probe.cjs. It is identified by the
// marker substring "probe.cjs" in the command, so uninstall removes ONLY the
// probe and leaves cvox hooks (marker "cvox notify") and any user hooks intact.
//
// Settings file: .claude/settings.local.json under the repo root (not committed,
// per .gitignore convention for local settings). Override with CVOX_PROBE_SETTINGS.

const fs = require("fs");
const path = require("path");

const REPO = path.resolve(__dirname, "..");
const PROBE = path.join(REPO, "probe.cjs");
const PROBE_CMD = `node ${PROBE}`;
const MARKER = "probe.cjs"; // substring that identifies a probe hook
const SETTINGS =
  process.env.CVOX_PROBE_SETTINGS ||
  path.join(REPO, ".claude", "settings.local.json");

// Hook events the probe attaches to. All matchers are "" (match everything) so
// the log captures the full pipeline regardless of tool/notification type.
const EVENTS = [
  "PreToolUse",
  "PostToolUse",
  "Notification",
  "PermissionRequest",
  "UserPromptSubmit",
  "Stop",
  "SubagentStop",
  "SessionStart",
  "SessionEnd",
  "PreCompact",
];

function readSettings() {
  try {
    return JSON.parse(fs.readFileSync(SETTINGS, "utf8"));
  } catch {
    return {};
  }
}

function writeSettings(obj) {
  fs.mkdirSync(path.dirname(SETTINGS), { recursive: true });
  fs.writeFileSync(SETTINGS, JSON.stringify(obj, null, 2) + "\n");
}

function isProbeHook(h) {
  return h && typeof h.command === "string" && h.command.includes(MARKER);
}

function probeHook() {
  return { type: "command", command: PROBE_CMD, async: true };
}

// Remove every probe hook from settings.hooks, pruning emptied matchers/events.
// Returns the number of probe hooks removed.
function stripProbe(settings) {
  let removed = 0;
  const hooks = settings.hooks;
  if (!hooks || typeof hooks !== "object") return 0;

  for (const event of Object.keys(hooks)) {
    const matchers = hooks[event];
    if (!Array.isArray(matchers)) continue;

    for (const m of matchers) {
      if (!m || !Array.isArray(m.hooks)) continue;
      const before = m.hooks.length;
      m.hooks = m.hooks.filter((h) => !isProbeHook(h));
      removed += before - m.hooks.length;
    }
    // Drop matcher entries that now have no hooks.
    hooks[event] = matchers.filter(
      (m) => m && Array.isArray(m.hooks) && m.hooks.length > 0
    );
    // Drop the whole event if it has no matchers left.
    if (hooks[event].length === 0) delete hooks[event];
  }
  if (Object.keys(hooks).length === 0) delete settings.hooks;
  return removed;
}

// Append a probe hook to the all-matcher ("") entry of each event, creating the
// event / matcher entry as needed. Idempotent: skips events that already have
// a probe hook on the "" matcher.
function addProbe(settings) {
  settings.hooks = settings.hooks || {};
  let added = 0;

  for (const event of EVENTS) {
    const matchers = settings.hooks[event] || (settings.hooks[event] = []);
    let entry = matchers.find((m) => m && m.matcher === "");
    if (!entry) {
      entry = { matcher: "", hooks: [] };
      matchers.push(entry);
    }
    entry.hooks = entry.hooks || [];
    if (entry.hooks.some(isProbeHook)) continue; // already installed
    entry.hooks.push(probeHook());
    added++;
  }
  return added;
}

function countProbe(settings) {
  let n = 0;
  const hooks = settings.hooks || {};
  for (const event of Object.keys(hooks)) {
    const matchers = hooks[event];
    if (!Array.isArray(matchers)) continue;
    for (const m of matchers) {
      if (m && Array.isArray(m.hooks)) n += m.hooks.filter(isProbeHook).length;
    }
  }
  return n;
}

function main() {
  const cmd = process.argv[2];

  if (!fs.existsSync(PROBE)) {
    console.error(`cvox probe: missing ${PROBE}`);
    process.exit(1);
  }

  if (cmd === "install") {
    const settings = readSettings();
    stripProbe(settings); // clear stale probe hooks first (idempotent reinstall)
    const added = addProbe(settings);
    writeSettings(settings);
    console.log(`cvox probe: installed on ${added} event(s) -> ${SETTINGS}`);
    console.log(`cvox probe: log file -> ${path.join(REPO, "probe.log")}`);
    console.log(
      "cvox probe: restart Claude Desktop / start a new session for hooks to load."
    );
  } else if (cmd === "uninstall") {
    const settings = readSettings();
    const removed = stripProbe(settings);
    writeSettings(settings);
    console.log(`cvox probe: removed ${removed} probe hook(s) from ${SETTINGS}`);
  } else if (cmd === "status") {
    const settings = readSettings();
    const n = countProbe(settings);
    console.log(`cvox probe: ${n} probe hook(s) currently installed in ${SETTINGS}`);
  } else {
    console.error("usage: probe-toggle.cjs <install|uninstall|status>");
    process.exit(1);
  }
}

main();
