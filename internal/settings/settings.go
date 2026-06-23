// Package settings reads and writes Claude's settings.json while preserving the
// user's unrelated configuration. cvox only ever touches the "hooks" key; every
// other top-level key (permissions, env, model, …) is passed through verbatim,
// byte for byte, so installing/removing cvox never reorders or mangles a config
// we don't own.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lmk123/cvox/internal/hooks"
	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
	"github.com/tidwall/sjson"
)

// cvoxMarker identifies a hook command as belonging to cvox. Any matcher whose
// hooks include a command containing this substring is considered ours and is
// stripped on re-install / remove.
const cvoxMarker = "cvox notify"

// GetSettingsPath returns the path to the Claude settings file. When global is
// true it is the machine-level ~/.claude/settings.json; otherwise it is the
// project-level .claude/settings.local.json under cwd (or the process cwd when
// cwd is empty).
func GetSettingsPath(global bool, cwd string) (string, error) {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude", "settings.json"), nil
	}
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(cwd, ".claude", "settings.local.json"), nil
}

// ParseError is returned when a settings file exists but cannot be safely read
// or parsed. Callers MUST treat this as "do not write" — overwriting would
// clobber a config we failed to understand (e.g. a hand-edited
// ~/.claude/settings.json with a syntax error would lose all the user's
// unrelated Claude config).
type ParseError struct {
	Path string
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("Failed to read settings at %s: %v", e.Path, e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

// IsParseError reports whether err is (or wraps) a *ParseError.
func IsParseError(err error) bool {
	var pe *ParseError
	return errors.As(err, &pe)
}

// Read reads a settings JSON file and returns its raw bytes.
//
// A missing file is the normal first-run case and yields an empty object `{}`.
// But a file that EXISTS yet cannot be read or JSON-parsed returns a
// *ParseError instead of silently returning `{}` — otherwise a subsequent Write
// would overwrite (and destroy) a config we never understood. The root must be
// a JSON object; an array/string/number/null is rejected the same way.
func Read(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []byte("{}"), nil // doesn't exist yet — safe to start fresh
		}
		return nil, &ParseError{Path: path, Err: err} // exists but unreadable
	}

	if !gjson.ValidBytes(content) {
		return nil, &ParseError{Path: path, Err: errors.New("invalid JSON")}
	}
	if !gjson.ParseBytes(content).IsObject() {
		return nil, &ParseError{Path: path, Err: errors.New("settings root is not a JSON object")}
	}
	return content, nil
}

// Write atomically writes raw settings bytes to path. The content is
// pretty-printed with 2-space indentation (key order preserved) plus a trailing
// newline, then written to a temp file in the same dir and renamed over the
// target — a crash mid-write can't leave settings.json truncated / corrupted.
func Write(path string, raw []byte) error {
	formatted := pretty.PrettyOptions(raw, &pretty.Options{Indent: "  ", SortKeys: false})
	// pretty.Pretty already appends a trailing newline.

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmpPath := path + "." + strconv.Itoa(os.Getpid()) + ".tmp"
	if err := os.WriteFile(tmpPath, formatted, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// escapeKey escapes sjson/gjson path metacharacters so an arbitrary JSON object
// key (e.g. a user-defined event name) is treated as a literal path segment.
// Escapes: \ . * ? # @ |
func escapeKey(key string) string {
	repl := strings.NewReplacer(
		`\`, `\\`,
		`.`, `\.`,
		`*`, `\*`,
		`?`, `\?`,
		`#`, `\#`,
		`@`, `\@`,
		`|`, `\|`,
	)
	return repl.Replace(key)
}

// rawArray joins raw JSON values into a single raw JSON array.
func rawArray(items []string) string {
	return "[" + strings.Join(items, ",") + "]"
}

// matcherIsCvox reports whether a matcher contains any hook whose command
// includes the cvox marker.
func matcherIsCvox(matcher gjson.Result) bool {
	hooksArr := matcher.Get("hooks")
	if !hooksArr.IsArray() {
		return false
	}
	found := false
	hooksArr.ForEach(func(_, hook gjson.Result) bool {
		cmd := hook.Get("command")
		if cmd.Type == gjson.String && strings.Contains(cmd.String(), cvoxMarker) {
			found = true
			return false // stop iterating
		}
		return true
	})
	return found
}

// MergeHooks strips every existing cvox matcher (so a re-run cleans up hooks
// newer versions no longer mount — e.g. the legacy Notification hook), then
// appends cvox's hooks for each event. All non-hook keys and any user-owned
// matchers are preserved byte-for-byte.
func MergeHooks(raw []byte, cfg hooks.Config) ([]byte, error) {
	out, err := RemoveHooks(raw)
	if err != nil {
		return nil, err
	}

	for _, ev := range hooks.EventOrder {
		path := "hooks." + escapeKey(ev)

		var items []string
		if existing := gjson.GetBytes(out, path); existing.IsArray() {
			existing.ForEach(func(_, m gjson.Result) bool {
				items = append(items, m.Raw)
				return true
			})
		}
		for _, m := range cfg.Hooks[ev] {
			b, err := json.Marshal(m)
			if err != nil {
				return nil, err
			}
			items = append(items, string(b))
		}

		out, err = sjson.SetRawBytes(out, path, []byte(rawArray(items)))
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// RemoveHooks removes every cvox matcher (matched by the marker, regardless of
// event name) from settings.hooks, pruning emptied events and dropping the
// whole "hooks" key when nothing is left. Non-cvox content is untouched.
func RemoveHooks(raw []byte) ([]byte, error) {
	hooksVal := gjson.GetBytes(raw, "hooks")
	if !hooksVal.IsObject() {
		return raw, nil
	}

	out := raw
	remainingEvents := 0
	var iterErr error
	hooksVal.ForEach(func(evKey, matchers gjson.Result) bool {
		path := "hooks." + escapeKey(evKey.String())
		if !matchers.IsArray() {
			remainingEvents++
			return true
		}

		var kept []string
		matchers.ForEach(func(_, m gjson.Result) bool {
			if !matcherIsCvox(m) {
				kept = append(kept, m.Raw)
			}
			return true
		})

		if len(kept) == 0 {
			out, iterErr = sjson.DeleteBytes(out, path)
		} else {
			out, iterErr = sjson.SetRawBytes(out, path, []byte(rawArray(kept)))
			remainingEvents++
		}
		return iterErr == nil
	})
	if iterErr != nil {
		return nil, iterErr
	}

	if remainingEvents == 0 {
		out, _ = sjson.DeleteBytes(out, "hooks")
	}
	return out, nil
}
