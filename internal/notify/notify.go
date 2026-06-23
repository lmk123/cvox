// Package notify processes a single Claude Code hook event from stdin and fires
// the configured voice and/or desktop notification.
package notify

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

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

	cwd := input.Cwd
	if cwd == "" {
		cwd = cwdFallback
	}
	cfg := config.Load(cwd)

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
