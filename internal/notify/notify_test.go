package notify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDebugLog(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, ".cvox.log")

	writeDebugLog(tmpDir, hookInput{
		HookEventName: "PermissionRequest",
		Cwd:           tmpDir,
		ToolName:      "Bash",
	})

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("writeDebugLog: log file not created: %v", err)
	}
	got := string(content)
	for _, want := range []string{"PermissionRequest", "tool=Bash", "cwd=" + tmpDir} {
		if !strings.Contains(got, want) {
			t.Errorf("writeDebugLog: log %q missing %q", got, want)
		}
	}

	// A second event should append, not overwrite.
	writeDebugLog(tmpDir, hookInput{HookEventName: "Stop", Cwd: tmpDir})
	content, err = os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	got = string(content)
	if !strings.Contains(got, "PermissionRequest") || !strings.Contains(got, "Stop") {
		t.Errorf("writeDebugLog: expected both events appended, got %q", got)
	}
	if lines := strings.Count(strings.TrimSpace(got), "\n") + 1; lines != 2 {
		t.Errorf("writeDebugLog: expected 2 log lines, got %d: %q", lines, got)
	}
}
