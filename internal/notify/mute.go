package notify

import (
	"regexp"
	"strings"
)

// mutedNotificationTools is the built-in mute list for the notification
// (PermissionRequest) path. Claude Desktop's Preview tools are auto-approved
// without ever showing a permission dialog, so a PermissionRequest hook fires
// (and cvox would speak) even though the user is never actually prompted. Mute
// the whole Preview namespace to kill that spurious sound.
//
// Patterns: only `*` is special (matches any run of chars, incl. empty). A
// leading `!` negates (un-mutes). Patterns are evaluated in order; the last one
// that matches a tool name wins — so add e.g.
// "!mcp__Claude_Preview__preview_eval" after the wildcard to re-enable the
// sound for a tool that turns out to really prompt.
var mutedNotificationTools = []string{
	"mcp__Claude_Preview__*",
	"!mcp__Claude_Preview__preview_start",
	"mcp__Claude_Browser__*",
	"!mcp__Claude_Browser__preview_start",
}

// globToRegexp escapes every regex metacharacter except `*`, then turns the
// literal `*` into `.*`, anchored to match the whole string.
func globToRegexp(pattern string) *regexp.Regexp {
	// regexp.QuoteMeta escapes `*` too, so split on `*` and quote each segment.
	parts := strings.Split(pattern, "*")
	for i, p := range parts {
		parts[i] = regexp.QuoteMeta(p)
	}
	return regexp.MustCompile("^" + strings.Join(parts, ".*") + "$")
}

// IsToolMuted reports whether toolName is muted on the notification path. The
// last matching pattern wins; a `!`-prefixed pattern un-mutes.
func IsToolMuted(toolName string) bool {
	muted := false
	for _, pattern := range mutedNotificationTools {
		negated := strings.HasPrefix(pattern, "!")
		body := pattern
		if negated {
			body = pattern[1:]
		}
		if globToRegexp(body).MatchString(toolName) {
			muted = !negated // last matching pattern wins
		}
	}
	return muted
}
