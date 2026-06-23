package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/lmk123/cvox/internal/config"
	"github.com/lmk123/cvox/internal/settings"
)

// Remove uninstalls cvox. By default it silences the current project (deletes
// its .cvox.json; the machine-wide hooks stay). With --global it fully
// uninstalls: drops the global hooks and ~/.cvox.json.
func Remove(args []string) error {
	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	global := fs.Bool("global", false, "Remove from global ~/.claude/settings.json")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *global {
		return removeGlobal()
	}
	return removeProject()
}

func removeGlobal() error {
	// Full uninstall: drop the machine-level hooks and the global config.
	settingsPath, err := settings.GetSettingsPath(true, "")
	if err != nil {
		return err
	}
	raw, err := settings.Read(settingsPath)
	if err != nil {
		if settings.IsParseError(err) {
			fmt.Fprintln(os.Stderr, "cvox:", err)
			fmt.Fprintln(os.Stderr, "cvox: Refusing to overwrite it. Fix the JSON (or remove the file) and re-run, or delete the cvox hooks by hand.")
			os.Exit(1)
		}
		return err
	}
	cleaned, err := settings.RemoveHooks(raw)
	if err != nil {
		return err
	}
	if err := settings.Write(settingsPath, cleaned); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configRemoved := config.RemoveProject(home)

	fmt.Println("cvox: Hooks removed from global settings →", settingsPath)
	if configRemoved {
		fmt.Printf("cvox: Config file removed → %s/.cvox.json\n", home)
	}
	return nil
}

func removeProject() error {
	// Project-level opt-out: deleting .cvox.json silences this project (hooks
	// still fire machine-wide, but with no config tts/desktop default to false).
	// Also strip any legacy cvox hook left in this project's settings.local.json.
	// Best-effort: if local settings are corrupt, skip cleanup and warn.
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	localSettingsPath, err := settings.GetSettingsPath(false, cwd)
	if err != nil {
		return err
	}
	localRaw, err := settings.Read(localSettingsPath)
	if err == nil {
		cleaned, err := settings.RemoveHooks(localRaw)
		if err == nil {
			if string(cleaned) != string(localRaw) {
				if err := settings.Write(localSettingsPath, cleaned); err != nil {
					return err
				}
				fmt.Println("cvox: Removed legacy hooks from", localSettingsPath)
			}
		} else if settings.IsParseError(err) {
			fmt.Fprintf(os.Stderr, "cvox: Skipped legacy-hook cleanup — %s is not valid JSON.\n", localSettingsPath)
		} else {
			return err
		}
	}

	if config.RemoveProject(cwd) {
		fmt.Printf("cvox: Config file removed → %s/.cvox.json\n", cwd)
		// The project override is gone, but a global ~/.cvox.json may still turn
		// notifications on (merge order: defaults → ~/.cvox.json → project).
		// Re-check the effective config so we don't falsely claim silence.
		effective := config.Load(cwd)
		if effective.TTS.Enabled || effective.Desktop.Enabled {
			fmt.Println("cvox: Note: ~/.cvox.json still enables notifications globally, so this project will keep notifying.")
		} else {
			fmt.Println("cvox: This project is now silent.")
		}
	} else {
		fmt.Println("cvox: No project .cvox.json found.")
	}
	fmt.Println("cvox: To uninstall the global hooks entirely, run: cvox remove --global")
	return nil
}
