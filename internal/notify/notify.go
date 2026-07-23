// Package notify processes a single Claude Code hook event from stdin and fires
// the configured voice and/or desktop notification.
package notify

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lmk123/cvox/internal/config"
)

// startable wraps an exec.Cmd so it can be started, waited on, and errors can be
// collected without holding the cmd itself in the closure.
type startable struct {
	cmd *exec.Cmd
}

func (s *startable) run() {
	if err := s.cmd.Run(); err != nil {
		os.Stderr.WriteString("cvox: " + err.Error() + "\n")
	}
}

// hookInput is the subset of the hook stdin payload cvox cares about.
type hookInput struct {
	HookEventName string `json:"hook_event_name"`
	Cwd           string `json:"cwd"`
	ToolName      string `json:"tool_name"`
}

// eventKey maps a hook event name to the config section it drives, or "" if the
// event is one cvox ignores.
//
// PermissionRequest fires on permission prompts in both the CLI and the Claude
// Desktop app; it maps to the "notification" config. Stop maps to "stop".
func eventKey(hookEventName string) string {
	switch hookEventName {
	case "PermissionRequest":
		return "notification"
	case "Stop":
		return "stop"
	default:
		return ""
	}
}

// Run reads a hook event from r and dispatches notifications. cwdFallback is
// used when the payload omits cwd (typically the process working directory).
// stdinIsTTY short-circuits to a no-op when stdin is an interactive terminal,
// matching the TS guard that avoids hanging on a manual `cvox notify`.
func Run(r io.Reader, cwdFallback string, stdinIsTTY bool) error {
	if stdinIsTTY {
		return nil
	}

	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return nil
	}

	var input hookInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil // malformed payload: stay silent, like the TS try/catch
	}
	if input.HookEventName == "" {
		return nil
	}

	cwd := input.Cwd
	if cwd == "" {
		cwd = cwdFallback
	}
	cfg := config.Load(cwd)

	// In debug mode, log every received event (including ones cvox ignores or
	// mutes below) so users can see which hook fired for a given prompt.
	if cfg.Debug {
		writeDebugLog(cwd, input)
	}

	key := eventKey(input.HookEventName)
	if key == "" {
		return nil
	}

	// Some tools (e.g. Claude Desktop's Preview tools) fire a PermissionRequest
	// hook but are auto-approved without ever prompting the user. Stay silent
	// for those to avoid spurious notification sounds.
	if key == "notification" && input.ToolName != "" && IsToolMuted(input.ToolName) {
		return nil
	}

	var event config.HookEvent
	if key == "notification" {
		event = cfg.Hooks.Notification
	} else {
		event = cfg.Hooks.Stop
	}
	if !event.Enabled {
		return nil
	}

	message := strings.ReplaceAll(event.Message, "{project}", cfg.Project)

	dispatch(message, cfg)
	return nil
}

// writeDebugLog appends a single line describing the received event to
// <cwd>/.cvox.log. It records only the fields cvox parses (hook_event_name,
// tool_name, cwd), mirroring probe.cjs's compact mode. Best-effort: any error
// is written to stderr and never aborts notification handling.
func writeDebugLog(cwd string, input hookInput) {
	path := filepath.Join(cwd, ".cvox.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		os.Stderr.WriteString("cvox: " + err.Error() + "\n")
		return
	}
	defer f.Close()

	line := "[" + time.Now().Format("15:04:05") + "] " +
		input.HookEventName + " tool=" + input.ToolName + " cwd=" + cwd + "\n"
	if _, err := f.WriteString(line); err != nil {
		os.Stderr.WriteString("cvox: " + err.Error() + "\n")
	}
}

// dispatch fires the voice and desktop notifications concurrently (so the
// blocking `say` doesn't delay the desktop balloon) and waits for both. Any
// command error is written to stderr but never aborts the other.
func dispatch(message string, cfg config.Config) {
	var cmds []*startable
	if cfg.TTS.Enabled {
		if c := speakCmd(message); c != nil {
			cmds = append(cmds, &startable{cmd: c})
		}
	}
	if cfg.Desktop.Enabled {
		if c := desktopCmd("cvox", message); c != nil {
			cmds = append(cmds, &startable{cmd: c})
		}
	}

	var wg sync.WaitGroup
	for _, s := range cmds {
		wg.Add(1)
		go func(s *startable) {
			defer wg.Done()
			s.run()
		}(s)
	}
	wg.Wait()
}
