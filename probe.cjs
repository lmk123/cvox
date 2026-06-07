#!/usr/bin/env node
// cvox hook probe — pure forensic logger for debugging which Claude Code hooks
// actually fire. Records only the event name + tool name (never tool_input /
// command / message) to avoid log self-pollution. Install/remove it via
// `npm run probe:install` / `npm run probe:uninstall`.
//
// Log path: $CVOX_PROBE_LOG, else <repo>/probe.log
const fs = require("fs");
const path = require("path");

const LOG = process.env.CVOX_PROBE_LOG || path.join(__dirname, "probe.log");

let data = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", (c) => (data += c));
process.stdin.on("end", () => {
  let ev = "?";
  let tool = "";
  try {
    const j = JSON.parse(data);
    ev = j.hook_event_name || "?";
    tool = j.tool_name || "";
  } catch {
    // non-JSON / empty stdin — still record that something fired
  }
  const ts = new Date().toISOString().slice(11, 19);
  fs.appendFileSync(LOG, `[${ts}] ${ev}${tool ? " " + tool : ""}\n`);
});
