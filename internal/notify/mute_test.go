package notify

import "testing"

func TestIsToolMuted(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		want     bool
	}{
		// wildcard mutes everything in the Preview namespace
		{"preview wildcard mutes preview_eval", "mcp__Claude_Preview__preview_eval", true},
		{"preview wildcard mutes preview_click", "mcp__Claude_Preview__preview_click", true},
		{"preview wildcard mutes preview_inspect", "mcp__Claude_Preview__preview_inspect", true},
		// but preview_start is explicitly un-muted by the ! pattern
		{"preview_start is un-muted", "mcp__Claude_Preview__preview_start", false},
		// tools outside the Preview namespace are not muted
		{"non-preview tool not muted", "mcp__other__foo", false},
		{"bash not muted", "bash", false},
		// edge cases
		{"empty string", "", false},
		{"partial match not muted", "mcp__Claude_Preview", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsToolMuted(tt.toolName); got != tt.want {
				t.Errorf("IsToolMuted(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}
