// Package config resolves cvox's effective configuration by merging three
// layers — built-in defaults, the global ~/.cvox.json, and the project
// .cvox.json — and reads/writes the project config file.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
	"github.com/tidwall/sjson"
)

// HookEvent is the resolved per-event configuration.
type HookEvent struct {
	Enabled bool
	Message string
}

// Toggle is a simple on/off section (tts, desktop).
type Toggle struct {
	Enabled bool
}

// Config is the fully resolved configuration after merging all layers.
type Config struct {
	Project string
	Hooks   struct {
		Notification HookEvent
		Stop         HookEvent
	}
	TTS     Toggle
	Desktop Toggle
}

// LocaleMessages maps a locale code to its notification/stop message templates.
type LocaleMessages struct {
	Notification string
	Stop         string
}

// Locales holds the built-in message templates for each supported language.
var Locales = map[string]LocaleMessages{
	"en": {Notification: "Claude Code needs permission, from {project}", Stop: "Claude Code task completed, from {project}"},
	"zh": {Notification: "Claude Code 需要权限，来自 {project}", Stop: "Claude Code 任务完成，来自 {project}"},
	"ja": {Notification: "Claude Code 権限が必要です、{project} より", Stop: "Claude Code タスクが完了しました、{project} より"},
	"ko": {Notification: "Claude Code 권한이 필요합니다, {project} 에서", Stop: "Claude Code 작업이 완료되었습니다, {project} 에서"},
}

// defaults returns a fresh Config with the built-in defaults. tts/desktop both
// default to OFF: this is the opt-in mechanism — a project with no .cvox.json
// merges down to silence.
func defaults() Config {
	var c Config
	c.Hooks.Notification = HookEvent{Enabled: true, Message: Locales["en"].Notification}
	c.Hooks.Stop = HookEvent{Enabled: true, Message: Locales["en"].Stop}
	c.TTS = Toggle{Enabled: false}
	c.Desktop = Toggle{Enabled: false}
	return c
}

// partial mirrors the on-disk .cvox.json schema with pointer fields, so a field
// that is absent from the file is nil and leaves the merged value untouched
// (replicating the TS deep-merge, where source keys override and missing keys
// are inherited).
type partial struct {
	Project *string `json:"project"`
	Hooks   *struct {
		Notification *partialEvent `json:"notification"`
		Stop         *partialEvent `json:"stop"`
	} `json:"hooks"`
	TTS     *partialToggle `json:"tts"`
	Desktop *partialToggle `json:"desktop"`
}

// Partial mirrors the on-disk .cvox.json schema with pointer fields.
// Exported for use by cli package to read existing config for default values.
type Partial = partial

// PartialHooks is the exported version of the nested Hooks struct in partial.
type PartialHooks = struct {
	Notification *PartialEvent `json:"notification"`
	Stop         *PartialEvent `json:"stop"`
}

type partialEvent struct {
	Enabled *bool   `json:"enabled"`
	Message *string `json:"message"`
}

// PartialEvent is the exported version of partialEvent.
type PartialEvent = partialEvent

type partialToggle struct {
	Enabled *bool `json:"enabled"`
}

// PartialToggle is the exported version of partialToggle.
type PartialToggle = partialToggle

// readPartial reads and parses a .cvox.json layer. Like the TS tryReadJson it is
// lenient: a missing or unparseable file yields nil (no error), so a corrupt
// .cvox.json simply doesn't contribute rather than aborting.
func readPartial(path string) *partial {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var p partial
	if err := json.Unmarshal(content, &p); err != nil {
		return nil
	}
	return &p
}

// ReadPartial reads and parses a .cvox.json layer, returning a Partial.
// It is lenient: a missing or unparseable file yields nil.
func ReadPartial(path string) *Partial {
	return readPartial(path)
}

func applyEvent(dst *HookEvent, src *partialEvent) {
	if src == nil {
		return
	}
	if src.Enabled != nil {
		dst.Enabled = *src.Enabled
	}
	if src.Message != nil {
		dst.Message = *src.Message
	}
}

func apply(c *Config, p *partial) {
	if p == nil {
		return
	}
	if p.Project != nil {
		c.Project = *p.Project
	}
	if p.Hooks != nil {
		applyEvent(&c.Hooks.Notification, p.Hooks.Notification)
		applyEvent(&c.Hooks.Stop, p.Hooks.Stop)
	}
	if p.TTS != nil && p.TTS.Enabled != nil {
		c.TTS.Enabled = *p.TTS.Enabled
	}
	if p.Desktop != nil && p.Desktop.Enabled != nil {
		c.Desktop.Enabled = *p.Desktop.Enabled
	}
}

// Load resolves the effective config for cwd: defaults → ~/.cvox.json → project
// .cvox.json. When project is still empty afterward it falls back to the
// directory name.
func Load(cwd string) Config {
	c := defaults()

	if home, err := os.UserHomeDir(); err == nil {
		apply(&c, readPartial(filepath.Join(home, ".cvox.json")))
	}
	apply(&c, readPartial(filepath.Join(cwd, ".cvox.json")))

	if c.Project == "" {
		c.Project = filepath.Base(cwd)
	}
	return c
}

// LoadForInherit resolves the config layers that an "inherit" choice in
// `cvox init` would fall back to — i.e. everything ABOVE the file init is about
// to write. For a project init that's defaults → ~/.cvox.json (the project
// .cvox.json is the file being written, so it's excluded). For --global it's
// just the defaults (the ~/.cvox.json being written is itself the global layer).
// This is what the "Inherit (X)" prompt label must reflect, not Load — Load
// includes the very file init is overwriting, which would show a stale value.
func LoadForInherit(global bool) Config {
	c := defaults()
	if !global {
		if home, err := os.UserHomeDir(); err == nil {
			apply(&c, readPartial(filepath.Join(home, ".cvox.json")))
		}
	}
	return c
}

// ProjectInput is what WriteProject persists into a .cvox.json.
// Nil fields mean "inherit" — they are not written to the file.
type ProjectInput struct {
	Project         string
	NotificationMsg *string
	StopMsg         *string
	TTSEnabled      *bool
	DesktopEnabled  *bool
}

// WriteProject writes (or updates) dir/.cvox.json with the given values. It sets
// only the specific leaf fields onto any existing file content — preserving
// unknown keys and key order, matching the TS deep-merge-onto-existing — then
// pretty-prints with 2-space indentation and a trailing newline.
//
// Nil fields in ProjectInput mean "inherit": any existing value for that field
// is deleted so it falls back to the parent layers (defaults → ~/.cvox.json),
// rather than being left behind. Parent objects that become empty as a result
// (hooks.notification, hooks.stop, hooks, tts, desktop) are pruned so re-running
// init with "Inherit" leaves a clean file instead of empty `{}` husks.
//
// Note the written shape deliberately omits hooks.*.enabled (the messages are
// the only per-event field init writes), mirroring the original.
func WriteProject(dir string, in ProjectInput) error {
	path := filepath.Join(dir, ".cvox.json")

	raw := []byte("{}")
	if existing, err := os.ReadFile(path); err == nil && json.Valid(existing) {
		raw = existing
	}

	var err error
	set := func(p string, v any) {
		if err != nil {
			return
		}
		raw, err = sjson.SetBytes(raw, p, v)
	}
	// del removes a leaf; non-existent paths are a no-op (sjson returns the
	// input unchanged), so it is safe to call unconditionally for nil fields.
	del := func(p string) {
		if err != nil {
			return
		}
		raw, err = sjson.DeleteBytes(raw, p)
	}
	// pruneIfEmpty deletes p only when it holds an empty object, so we don't
	// disturb unknown sibling keys the user may have added.
	pruneIfEmpty := func(p string) {
		if err != nil {
			return
		}
		res := gjson.GetBytes(raw, p)
		if res.Exists() && res.IsObject() && len(res.Map()) == 0 {
			raw, err = sjson.DeleteBytes(raw, p)
		}
	}

	set("project", in.Project)
	if in.NotificationMsg != nil {
		set("hooks.notification.message", *in.NotificationMsg)
	} else {
		del("hooks.notification.message")
	}
	if in.StopMsg != nil {
		set("hooks.stop.message", *in.StopMsg)
	} else {
		del("hooks.stop.message")
	}
	if in.TTSEnabled != nil {
		set("tts.enabled", *in.TTSEnabled)
	} else {
		del("tts.enabled")
	}
	if in.DesktopEnabled != nil {
		set("desktop.enabled", *in.DesktopEnabled)
	} else {
		del("desktop.enabled")
	}
	// Prune containers emptied by the deletions above (children before parent).
	pruneIfEmpty("hooks.notification")
	pruneIfEmpty("hooks.stop")
	pruneIfEmpty("hooks")
	pruneIfEmpty("tts")
	pruneIfEmpty("desktop")
	if err != nil {
		return err
	}

	formatted := pretty.PrettyOptions(raw, &pretty.Options{Indent: "  ", SortKeys: false})
	return os.WriteFile(path, formatted, 0o644)
}

// RemoveProject deletes dir/.cvox.json, returning true if a file was removed.
func RemoveProject(dir string) bool {
	return os.Remove(filepath.Join(dir, ".cvox.json")) == nil
}
