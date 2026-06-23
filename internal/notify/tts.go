package notify

import (
	"os"
	"os/exec"
	"runtime"
)

// engine identifies the platform TTS backend.
type engine int

const (
	engineNone   engine = iota
	engineSay           // macOS `say`
	engineEspeak        // Linux `espeak`
	engineSAPI          // Windows SAPI via PowerShell
)

func detectEngine() engine {
	switch runtime.GOOS {
	case "darwin":
		return engineSay
	case "windows":
		return engineSAPI
	default:
		return engineEspeak
	}
}

// sapiScript drives the Windows Speech API; the message arrives via the
// CVOX_MSG environment variable. Using an env var instead of $args because
// PowerShell's -Command does not bind subsequent arguments to $args.
const sapiScript = `Add-Type -AssemblyName System.Speech; ` +
	`$s = New-Object System.Speech.Synthesis.SpeechSynthesizer; ` +
	`$s.Speak($env:CVOX_MSG)`

// speakCmd builds (but does not start) the TTS command for message, or nil if
// the platform has no known engine.
func speakCmd(message string) *exec.Cmd {
	switch detectEngine() {
	case engineSay:
		return exec.Command("say", message)
	case engineEspeak:
		return exec.Command("espeak", message)
	case engineSAPI:
		cmd := exec.Command("powershell", "-Command", sapiScript)
		cmd.Env = append(os.Environ(), "CVOX_MSG="+message)
		return cmd
	default:
		return nil
	}
}
