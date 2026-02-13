#!/usr/bin/env node
import { readFileSync } from "fs";
import { fileURLToPath } from "url";
import { dirname, join } from "path";
import { Command } from "commander";
import { initCommand } from "./commands/init.js";
import { notifyCommand } from "./commands/notify.js";
import { removeCommand } from "./commands/remove.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const pkg = JSON.parse(readFileSync(join(__dirname, "..", "package.json"), "utf-8"));

const program = new Command();

program
  .name("cvox")
  .description("Claude Voice Notifications")
  .version(pkg.version);

program
  .command("init")
  .description("Set up cvox for your project")
  .option("--global", "Write to global ~/.claude/settings.json")
  .action(initCommand);

program
  .command("notify")
  .description("Process Claude Code events and play notifications")
  .action(notifyCommand);

program
  .command("remove")
  .description("Remove cvox hooks from Claude Code settings")
  .option("--global", "Remove from global ~/.claude/settings.json")
  .action(removeCommand);

program.parse();
