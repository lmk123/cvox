package notify

import (
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

// sapiScript drives the Windows Speech API; the message arrives as $args[0].
const sapiScript = `Add-Type -AssemblyName System.Speech; ` +
	`$s = New-Object System.Speech.Synthesis.SpeechSynthesizer; ` +
	`$s.Speak($args[0])`

// speakCmd builds (but does not start) the TTS command for message, or nil if
// the platform has no known engine.
func speakCmd(message string) *exec.Cmd {
	switch detectEngine() {
	case engineSay:
		return exec.Command("say", message)
	case engineEspeak:
		return exec.Command("espeak", message)
	case engineSAPI:
		return exec.Command("powershell", "-Command", sapiScript, message)
	default:
		return nil
	}
}
