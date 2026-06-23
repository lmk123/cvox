package notify

import (
	"os/exec"
	"runtime"
)

// notifyScript is the Windows balloon-notification PowerShell snippet; title and
// message arrive as $args[0] and $args[1].
const notifyScript = `Add-Type -AssemblyName System.Windows.Forms; ` +
	`$n = New-Object System.Windows.Forms.NotifyIcon; ` +
	`$n.Icon = [System.Drawing.SystemIcons]::Information; ` +
	`$n.Visible = $true; ` +
	`$n.ShowBalloonTip(5000, $args[0], $args[1], 'Info')`

// desktopCmd builds (but does not start) the desktop-notification command for
// the platform.
func desktopCmd(title, message string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("osascript",
			"-e", "on run argv",
			"-e", "display notification (item 2 of argv) with title (item 1 of argv)",
			"-e", "end run",
			"--", title, message,
		)
	case "windows":
		return exec.Command("powershell", "-Command", notifyScript, title, message)
	default:
		return exec.Command("notify-send", title, message)
	}
}
