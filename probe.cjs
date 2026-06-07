#!/usr/bin/env node
// cvox hook probe — forensic logger for debugging which Claude Code hooks
// actually fire. Default mode records only the event name + tool name (never
// tool_input / command / message) to avoid log self-pollution. Pass --full to
// dump the ENTIRE stdin JSON instead — use when you need to see every field a
// real event carries (commands, paths, etc; don't commit that log). Install /
// remove via `npm run probe:install` / `probe:install:full` / `probe:uninstall`.
//
// Log path: $CVOX_PROBE_LOG, else <repo>/probe.log (--full: <repo>/probe-full.log)
const fs = require("fs");
const path = require("path");

const FULL = process.argv.includes("--full");
const LOG =
  process.env.CVOX_PROBE_LOG ||
  path.join(__dirname, FULL ? "probe-full.log" : "probe.log");

let data = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", (c) => (data += c));
process.stdin.on("end", () => {
  const ts = new Date().toISOString().slice(11, 19);
  if (FULL) {
    let pretty = data;
    try {
      pretty = JSON.stringify(JSON.parse(data), null, 2);
    } catch {
      // non-JSON / empty stdin — leave raw
    }
    fs.appendFileSync(LOG, `\n===== [${ts}] =====\n${pretty}\n`);
    return;
  }
  let ev = "?";
  let tool = "";
  try {
    const j = JSON.parse(data);
    ev = j.hook_event_name || "?";
    tool = j.tool_name || "";
  } catch {
    // non-JSON / empty stdin — still record that something fired
  }
  fs.appendFileSync(LOG, `[${ts}] ${ev}${tool ? " " + tool : ""}\n`);
});
