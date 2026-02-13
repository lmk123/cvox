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
  .description("向 Claude Code settings 注入 hooks")
  .option("--global", "写入全局 ~/.claude/settings.json")
  .action(initCommand);

program
  .command("notify")
  .description("被 hooks 调用，读取 stdin 播放语音通知")
  .action(notifyCommand);

program.parse();
