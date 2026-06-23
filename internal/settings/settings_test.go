package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmk123/cvox/internal/hooks"
	"github.com/tidwall/gjson"
)

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}

func TestReadMissingFile(t *testing.T) {
	// A missing file is the normal first-run case → empty object, no error.
	raw, err := Read("/nonexistent/path/settings.json")
	if err != nil {
		t.Fatalf("Read missing file: unexpected error %v", err)
	}
	if string(raw) != "{}" {
		t.Errorf("Read missing file: got %q, want {}", raw)
	}
}

func TestReadCorruptFile(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "settings.json")
	if err := writeFile(tmp, "{ this is not json"); err != nil {
		t.Fatal(err)
	}
	_, err := Read(tmp)
	if err == nil {
		t.Fatal("Read corrupt file: expected error, got nil")
	}
	if !IsParseError(err) {
		t.Errorf("Read corrupt file: expected ParseError, got %T", err)
	}
}

func TestReadNonObjectRoot(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "settings.json")
	if err := writeFile(tmp, `["array","not","object"]`); err != nil {
		t.Fatal(err)
	}
	_, err := Read(tmp)
	if !IsParseError(err) {
		t.Errorf("Read array root: expected ParseError, got %v", err)
	}
}

func TestMergeHooksPreservesUserConfig(t *testing.T) {
	// A settings file with unrelated keys and a user-owned hook.
	input := []byte(`{
  "model": "claude-opus-4",
  "permissions": {"allow": ["Bash"]},
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "echo hi"}]}
    ]
  }
}`)
	out, err := MergeHooks(input, hooks.Generate())
	if err != nil {
		t.Fatal(err)
	}

	// User's unrelated keys preserved.
	if gjson.GetBytes(out, "model").String() != "claude-opus-4" {
		t.Error("MergeHooks: model key was lost")
	}
	if gjson.GetBytes(out, "permissions.allow.0").String() != "Bash" {
		t.Error("MergeHooks: permissions key was lost")
	}
	// User's own hook preserved.
	if gjson.GetBytes(out, `hooks.PreToolUse.0.hooks.0.command`).String() != "echo hi" {
		t.Error("MergeHooks: user's PreToolUse hook was lost")
	}
	// cvox hooks added.
	if !gjson.GetBytes(out, "hooks.PermissionRequest").Exists() {
		t.Error("MergeHooks: PermissionRequest hook not added")
	}
	if !gjson.GetBytes(out, "hooks.Stop").Exists() {
		t.Error("MergeHooks: Stop hook not added")
	}
	if cmd := gjson.GetBytes(out, "hooks.PermissionRequest.0.hooks.0.command").String(); cmd != "cvox notify" {
		t.Errorf("MergeHooks: PermissionRequest command = %q, want cvox notify", cmd)
	}
}

func TestMergeHooksIdempotent(t *testing.T) {
	// Running merge twice should not duplicate cvox hooks.
	out1, err := MergeHooks([]byte(`{}`), hooks.Generate())
	if err != nil {
		t.Fatal(err)
	}
	out2, err := MergeHooks(out1, hooks.Generate())
	if err != nil {
		t.Fatal(err)
	}
	count := len(gjson.GetBytes(out2, "hooks.PermissionRequest").Array())
	if count != 1 {
		t.Errorf("MergeHooks idempotent: PermissionRequest has %d matchers, want 1", count)
	}
}

func TestMergeHooksStripsLegacy(t *testing.T) {
	// An old cvox hook under the legacy Notification event should be removed.
	input := []byte(`{
  "hooks": {
    "Notification": [
      {"matcher": "permission_prompt", "hooks": [{"type": "command", "command": "cvox notify", "async": true}]}
    ]
  }
}`)
	out, err := MergeHooks(input, hooks.Generate())
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "hooks.Notification").Exists() {
		t.Error("MergeHooks: legacy Notification hook should be stripped")
	}
}

func TestRemoveHooksOnlyTouchesCvox(t *testing.T) {
	input := []byte(`{
  "model": "x",
  "hooks": {
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [{"type": "command", "command": "echo hi"}]}
    ],
    "Stop": [
      {"matcher": "", "hooks": [{"type": "command", "command": "cvox notify", "async": true}]}
    ]
  }
}`)
	out, err := RemoveHooks(input)
	if err != nil {
		t.Fatal(err)
	}
	// cvox Stop hook removed (and the now-empty Stop event dropped).
	if gjson.GetBytes(out, "hooks.Stop").Exists() {
		t.Error("RemoveHooks: cvox Stop hook should be removed")
	}
	// User's PreToolUse hook preserved.
	if gjson.GetBytes(out, "hooks.PreToolUse.0.hooks.0.command").String() != "echo hi" {
		t.Error("RemoveHooks: user's hook should be preserved")
	}
	// Unrelated key preserved.
	if gjson.GetBytes(out, "model").String() != "x" {
		t.Error("RemoveHooks: model key was lost")
	}
}

func TestRemoveHooksDropsEmptyHooks(t *testing.T) {
	// When only cvox hooks exist, the entire "hooks" key should be dropped.
	input := []byte(`{
  "model": "x",
  "hooks": {
    "Stop": [
      {"matcher": "", "hooks": [{"type": "command", "command": "cvox notify"}]}
    ]
  }
}`)
	out, err := RemoveHooks(input)
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(out, "hooks").Exists() {
		t.Error("RemoveHooks: empty hooks key should be dropped")
	}
	if gjson.GetBytes(out, "model").String() != "x" {
		t.Error("RemoveHooks: model key was lost")
	}
}

func TestWriteAtomicAndFormatted(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "settings.json")
	if err := Write(tmp, []byte(`{"b":2,"a":1}`)); err != nil {
		t.Fatal(err)
	}
	content, err := readFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	// Key order preserved (not sorted): "b" comes before "a".
	if strings.Index(content, `"b"`) > strings.Index(content, `"a"`) {
		t.Error("Write: key order should be preserved (b before a)")
	}
	// 2-space indentation.
	if !strings.Contains(content, "\n  \"b\": 2") {
		t.Errorf("Write: expected 2-space indent, got:\n%s", content)
	}
	// Trailing newline.
	if !strings.HasSuffix(content, "\n") {
		t.Error("Write: expected trailing newline")
	}
}
