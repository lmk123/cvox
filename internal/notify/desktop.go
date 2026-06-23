package notify

import (
	"os"
	"os/exec"
	"runtime"
)

// notifyScript is the Windows balloon-notification PowerShell snippet; title and
// message arrive via CVOX_TITLE and CVOX_MSG environment variables. Using env
// vars instead of $args because PowerShell's -Command does not bind subsequent
// arguments to $args.
const notifyScript = `Add-Type -AssemblyName System.Windows.Forms; ` +
	`$n = New-Object System.Windows.Forms.NotifyIcon; ` +
	`$n.Icon = [System.Drawing.SystemIcons]::Information; ` +
	`$n.Visible = $true; ` +
	`$n.ShowBalloonTip(5000, $env:CVOX_TITLE, $env:CVOX_MSG, 'Info')`

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
		cmd := exec.Command("powershell", "-Command", notifyScript)
		cmd.Env = append(os.Environ(), "CVOX_TITLE="+title, "CVOX_MSG="+message)
		return cmd
	default:
		return exec.Command("notify-send", title, message)
	}
}
