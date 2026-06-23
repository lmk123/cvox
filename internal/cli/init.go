// Package cli implements the three cvox commands: init, notify, and remove.
package cli

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmk123/cvox/internal/config"
	"github.com/lmk123/cvox/internal/hooks"
	"github.com/lmk123/cvox/internal/settings"
)

// localeOption pairs a locale code with its display label and numeric key.
// An empty code means "inherit" (don't set in .cvox.json).
type localeOption struct {
	key   string
	code  string
	label string
}

var localeOptions = []localeOption{
	{"1", "en", "English"},
	{"2", "zh", "中文"},
	{"3", "ja", "日本語"},
	{"4", "ko", "한국어"},
	{"5", "", "Inherit"},
}

// notifyMethodOption pairs a method key with its display label and the
// tts/desktop flags it sets. Nil flags mean "inherit" (don't set in .cvox.json).
type notifyMethodOption struct {
	key     string
	label   string
	tts     *bool
	desktop *bool
}

var notifyMethodOptions = []notifyMethodOption{
	{"1", "Voice only", ptr(true), ptr(false)},
	{"2", "Desktop notification only", ptr(false), ptr(true)},
	{"3", "Both voice and desktop", ptr(true), ptr(true)},
	{"4", "Inherit", nil, nil},
}

// ptr returns a pointer to b.
func ptr(b bool) *bool {
	return &b
}

// Init sets up cvox for the current project or globally.
func Init(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	global := fs.Bool("global", false, "Write .cvox.json to the home directory (all projects speak)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	defaultName := filepath.Base(cwd)

	// Read the global settings up front (before prompting) so a corrupt
	// ~/.claude/settings.json aborts immediately rather than after the user has
	// answered every question. Crucially we NEVER write on a parse error — that
	// would overwrite (and destroy) the user's unrelated Claude config.
	settingsPath, err := settings.GetSettingsPath(true, "")
	if err != nil {
		return err
	}
	raw, err := settings.Read(settingsPath)
	if err != nil {
		if settings.IsParseError(err) {
			fmt.Fprintln(os.Stderr, "cvox:", err)
			fmt.Fprintln(os.Stderr, "cvox: Refusing to overwrite it. Please fix the JSON (or remove the file) and re-run cvox init.")
			os.Exit(1)
		}
		return err
	}

	// Interactive prompts.
	r := bufio.NewReader(os.Stdin)

	// Question 1: project name.
	fmt.Printf("Project name (default: %s): ", defaultName)
	nameAnswer, _ := r.ReadString('\n')
	projectName := strings.TrimSpace(nameAnswer)
	if projectName == "" {
		projectName = defaultName
	}

	// Question 2: language.
	// Load the parent layers (everything above the file we're about to write) so
	// the "Inherit (X)" label shows the value an inherit choice would fall back
	// to — not the value in the file we're overwriting.
	inheritConfig := config.LoadForInherit(*global)
	inheritedLocale := "en" // default
	if inheritConfig.Hooks.Notification.Message != "" {
		// Detect the inherited locale by matching its notification message
		// against the built-in templates (best-effort: a hand-edited custom
		// message won't match and falls back to English for display only).
		for code, msgs := range config.Locales {
			if inheritConfig.Hooks.Notification.Message == msgs.Notification {
				inheritedLocale = code
				break
			}
		}
	}
	inheritedLabel := ""
	for _, opt := range localeOptions {
		if opt.code == inheritedLocale {
			inheritedLabel = opt.label
			break
		}
	}

	fmt.Println("Voice language:")
	for _, opt := range localeOptions {
		defaultMark := ""
		if opt.key == "1" {
			defaultMark = " (default)"
		}
		label := opt.label
		if opt.code == "" && inheritedLabel != "" {
			label = fmt.Sprintf("%s (%s)", opt.label, inheritedLabel)
		}
		fmt.Printf("  %s. %s%s\n", opt.key, label, defaultMark)
	}
	fmt.Print("Select language [1]: ")
	localeAnswer, _ := r.ReadString('\n')
	selected := localeOptions[0]
	for _, opt := range localeOptions {
		if strings.TrimSpace(localeAnswer) == opt.key {
			selected = opt
			break
		}
	}
	// messages is only consumed when a concrete locale is chosen; for Inherit
	// (empty code) the message fields are left nil and never read.
	messages := config.Locales[selected.code]

	// Question 3: notification method.
	// Show the method an inherit choice would fall back to.
	inheritedMethodLabel := "None (silent)"
	if inheritConfig.TTS.Enabled && inheritConfig.Desktop.Enabled {
		inheritedMethodLabel = "Both voice and desktop"
	} else if inheritConfig.Desktop.Enabled {
		inheritedMethodLabel = "Desktop notification only"
	} else if inheritConfig.TTS.Enabled {
		inheritedMethodLabel = "Voice only"
	}

	fmt.Println("Notification method:")
	for _, opt := range notifyMethodOptions {
		defaultMark := ""
		if opt.key == "1" {
			defaultMark = " (default)"
		}
		label := opt.label
		if opt.tts == nil && opt.desktop == nil {
			label = fmt.Sprintf("%s (%s)", opt.label, inheritedMethodLabel)
		}
		fmt.Printf("  %s. %s%s\n", opt.key, label, defaultMark)
	}
	fmt.Print("Select method [1]: ")
	methodAnswer, _ := r.ReadString('\n')
	selectedMethod := notifyMethodOptions[0]
	for _, opt := range notifyMethodOptions {
		if strings.TrimSpace(methodAnswer) == opt.key {
			selectedMethod = opt
			break
		}
	}

	// Inject hooks. Hooks are always written to the global ~/.claude/settings.json
	// because it is machine-level; a single install covers every project and every
	// git worktree (worktrees don't check out .gitignored settings.local.json).
	cvoxHooks := hooks.Generate()
	merged, err := settings.MergeHooks(raw, cvoxHooks)
	if err != nil {
		return err
	}
	if err := settings.Write(settingsPath, merged); err != nil {
		return err
	}

	// Lazy cleanup: early versions wrote hooks to the project's
	// .claude/settings.local.json. If this project still has a legacy hook, the
	// main checkout would "global + local" double-speak (worktrees are fine
	// because they don't have the file). Strip the project's cvox hooks by marker
	// (only cvox's own hooks are touched). Best-effort: if local settings are
	// corrupt we skip cleanup and warn, but the global install is already done.
	localSettingsPath, err := settings.GetSettingsPath(false, cwd)
	if err != nil {
		return err
	}
	var hadLocalHook bool
	localRaw, err := settings.Read(localSettingsPath)
	if err == nil {
		cleaned, err := settings.RemoveHooks(localRaw)
		if err == nil {
			hadLocalHook = string(cleaned) != string(localRaw)
			if hadLocalHook {
				if err := settings.Write(localSettingsPath, cleaned); err != nil {
					return err
				}
			}
		} else if settings.IsParseError(err) {
			fmt.Fprintf(os.Stderr, "cvox: Skipped legacy-hook cleanup — %s is not valid JSON. Fix it manually if this project double-speaks.\n", localSettingsPath)
		} else {
			return err
		}
	}

	// Generate .cvox.json. --global writes to the home directory (all projects
	// speak), otherwise to the project root (only this project speaks, and it
	// travels with worktrees because .cvox.json is committed).
	configDir := cwd
	if *global {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configDir = home
	}

	// Convert selected values to pointers for ProjectInput.
	// Nil means "inherit" (don't write to .cvox.json).
	var notificationMsg, stopMsg *string
	if selected.code != "" {
		notificationMsg = &messages.Notification
		stopMsg = &messages.Stop
	}
	if err := config.WriteProject(configDir, config.ProjectInput{
		Project:         projectName,
		NotificationMsg: notificationMsg,
		StopMsg:         stopMsg,
		TTSEnabled:      selectedMethod.tts,
		DesktopEnabled:  selectedMethod.desktop,
	}); err != nil {
		return err
	}

	scope := "all projects"
	if !*global {
		scope = projectName
	}
	fmt.Println("cvox: Hooks installed globally →", settingsPath)
	fmt.Println("  - Notify on permission prompt")
	fmt.Println("  - Notify on task completion")
	if hadLocalHook {
		fmt.Println("cvox: Removed legacy hooks from", localSettingsPath)
	}
	fmt.Printf("cvox: Generated .cvox.json for %s → %s/.cvox.json (%s, %s)\n",
		scope, configDir, selected.label, selectedMethod.label)
	return nil
}
