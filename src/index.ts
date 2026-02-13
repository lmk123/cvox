#!/usr/bin/env node
import { Command } from "commander";
import { initCommand } from "./commands/init.js";
import { notifyCommand } from "./commands/notify.js";

const program = new Command();

program
  .name("cvox")
  .description("Claude Voice Notifications")
  .version("1.0.0");

program
  .command("init")
  .description("Inject hooks into Claude Code settings")
  .option("--global", "Write to global ~/.claude/settings.json")
  .action(initCommand);

program
  .command("notify")
  .description("Called by hooks, reads stdin and plays voice notifications")
  .action(notifyCommand);

program.parse();
