// cvox is a CLI tool that provides voice and desktop notifications for Claude Code.
// It installs hooks into Claude's settings.json and processes hook events to speak
// or pop up notifications when Claude needs permission or finishes a task.
package main

import (
	"fmt"
	"os"

	"github.com/lmk123/cvox/internal/cli"
)

// version is injected at build time by goreleaser (or a manual -ldflags).
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		if err := cli.Init(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "cvox:", err)
			os.Exit(1)
		}
	case "notify":
		// notify is called by hooks; errors are logged to stderr by the
		// implementation but we don't exit on failure (best-effort).
		cli.Notify(os.Args[2:])
	case "remove":
		if err := cli.Remove(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "cvox:", err)
			os.Exit(1)
		}
	case "-v", "--version", "version":
		fmt.Println("cvox", version)
	default:
		if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help" {
			printUsage()
		} else {
			fmt.Fprintln(os.Stderr, "cvox: unknown command:", os.Args[1])
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Println("cvox - Claude Voice Notifications")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cvox init [--global]       Set up cvox for your project")
	fmt.Println("  cvox notify                 Process Claude Code events (called by hooks)")
	fmt.Println("  cvox remove [--global]     Remove cvox hooks from Claude Code settings")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --global                    Write config to ~/.cvox.json (all projects speak)")
	fmt.Println()
	fmt.Println("See https://github.com/lmk123/cvox for details.")
}
