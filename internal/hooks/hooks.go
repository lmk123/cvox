// Package hooks defines the Claude Code hook entries cvox installs and the
// marker used to recognize them on re-install / removal.
package hooks

// Hook is a single command hook entry in Claude's settings.json.
type Hook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Async   bool   `json:"async"`
}

// Matcher pairs a tool matcher pattern with the hooks that fire for it.
type Matcher struct {
	Matcher string `json:"matcher"`
	Hooks   []Hook `json:"hooks"`
}

// Config is the set of cvox matchers keyed by hook event name.
type Config struct {
	Hooks map[string][]Matcher
}

// EventOrder is the deterministic order in which events are written into
// settings.json. Map iteration in Go is randomized, so we keep an explicit
// slice to make output stable.
var EventOrder = []string{"PermissionRequest", "Stop"}

// Generate returns the cvox hook configuration.
//
// Permission prompts fire PermissionRequest on both the Claude Code CLI and the
// Claude Desktop app, so this single hook covers both. (The CLI also fires a
// legacy Notification hook that Desktop does not; cvox no longer uses it, since
// PermissionRequest alone is enough.) Stop fires on task completion.
func Generate() Config {
	notifyHook := Hook{Type: "command", Command: "cvox notify", Async: true}
	return Config{
		Hooks: map[string][]Matcher{
			// empty matcher matches all events of that type
			"PermissionRequest": {{Matcher: "", Hooks: []Hook{notifyHook}}},
			"Stop":              {{Matcher: "", Hooks: []Hook{notifyHook}}},
		},
	}
}
