package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// No config files: defaults only, tts/desktop off, project from cwd.
	tmpDir := t.TempDir()
	// Mock home dir to avoid reading the user's actual ~/.cvox.json.
	// Note: os.UserHomeDir() reads HOME on Unix and USERPROFILE on Windows,
	// so this only isolates on Unix. Cross-platform isolation would require
	// a more elaborate setup (e.g. testutil.Chdir + env override).
	t.Setenv("HOME", tmpDir)

	cfg := Load(tmpDir)
	if cfg.Project != filepath.Base(tmpDir) {
		t.Errorf("Load: project = %q, want %q", cfg.Project, filepath.Base(tmpDir))
	}
	if cfg.TTS.Enabled {
		t.Error("Load: tts.enabled should be false by default")
	}
	if cfg.Desktop.Enabled {
		t.Error("Load: desktop.enabled should be false by default")
	}
	if !cfg.Hooks.Notification.Enabled {
		t.Error("Load: hooks.notification.enabled should be true by default")
	}
	if !cfg.Hooks.Stop.Enabled {
		t.Error("Load: hooks.stop.enabled should be true by default")
	}
}

func TestLoadWithProjectConfig(t *testing.T) {
	tmpDir := t.TempDir()
	notificationMsg := "test notification"
	stopMsg := "test stop"
	ttsEnabled := true
	desktopEnabled := false
	if err := WriteProject(tmpDir, ProjectInput{
		Project:         "testproj",
		NotificationMsg: &notificationMsg,
		StopMsg:         &stopMsg,
		TTSEnabled:      &ttsEnabled,
		DesktopEnabled:  &desktopEnabled,
	}); err != nil {
		t.Fatal(err)
	}
	cfg := Load(tmpDir)
	if cfg.Project != "testproj" {
		t.Errorf("Load: project = %q, want testproj", cfg.Project)
	}
	if !cfg.TTS.Enabled {
		t.Error("Load: tts.enabled should be true from project config")
	}
	if cfg.Desktop.Enabled {
		t.Error("Load: desktop.enabled should be false from project config")
	}
	if cfg.Hooks.Notification.Message != "test notification" {
		t.Errorf("Load: notification message = %q, want test notification", cfg.Hooks.Notification.Message)
	}
	if cfg.Hooks.Stop.Message != "test stop" {
		t.Errorf("Load: stop message = %q, want test stop", cfg.Hooks.Stop.Message)
	}
}

func TestLoadWithGlobalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := t.TempDir()
	notificationMsg := "global notification"
	stopMsg := "global stop"
	ttsEnabled := false
	desktopEnabled := true
	if err := WriteProject(globalDir, ProjectInput{
		Project:         "", // not set: should fall back to cwd
		NotificationMsg: &notificationMsg,
		StopMsg:         &stopMsg,
		TTSEnabled:      &ttsEnabled,
		DesktopEnabled:  &desktopEnabled,
	}); err != nil {
		t.Fatal(err)
	}
	// Mock home dir by temporarily setting HOME to globalDir.
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", globalDir)
	defer os.Setenv("HOME", oldHome)

	cfg := Load(tmpDir)
	if cfg.Project != filepath.Base(tmpDir) {
		// project name comes from cwd when not set in global config
		t.Errorf("Load: project = %q, want %q", cfg.Project, filepath.Base(tmpDir))
	}
	if cfg.TTS.Enabled {
		t.Error("Load: tts.enabled should be false from global config")
	}
	if !cfg.Desktop.Enabled {
		t.Error("Load: desktop.enabled should be true from global config")
	}
	if cfg.Hooks.Notification.Message != "global notification" {
		t.Errorf("Load: notification message = %q, want global notification", cfg.Hooks.Notification.Message)
	}
}

func TestLoadProjectOverridesGlobal(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := t.TempDir()
	globalNotificationMsg := "global notification"
	globalStopMsg := "global stop"
	globalTtsEnabled := false
	globalDesktopEnabled := true
	if err := WriteProject(globalDir, ProjectInput{
		Project:         "globalproj",
		NotificationMsg: &globalNotificationMsg,
		StopMsg:         &globalStopMsg,
		TTSEnabled:      &globalTtsEnabled,
		DesktopEnabled:  &globalDesktopEnabled,
	}); err != nil {
		t.Fatal(err)
	}
	projNotificationMsg := "proj notification"
	projStopMsg := "proj stop"
	projTtsEnabled := true
	projDesktopEnabled := false
	if err := WriteProject(tmpDir, ProjectInput{
		Project:         "proj",
		NotificationMsg: &projNotificationMsg,
		StopMsg:         &projStopMsg,
		TTSEnabled:      &projTtsEnabled,
		DesktopEnabled:  &projDesktopEnabled,
	}); err != nil {
		t.Fatal(err)
	}
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", globalDir)
	defer os.Setenv("HOME", oldHome)

	cfg := Load(tmpDir)
	if cfg.Project != "proj" {
		t.Errorf("Load: project = %q, want proj", cfg.Project)
	}
	if !cfg.TTS.Enabled {
		t.Error("Load: tts.enabled should be true from project config (overrides global)")
	}
	if cfg.Desktop.Enabled {
		t.Error("Load: desktop.enabled should be false from project config (overrides global)")
	}
	if cfg.Hooks.Notification.Message != "proj notification" {
		t.Errorf("Load: notification message = %q, want proj notification", cfg.Hooks.Notification.Message)
	}
}

func TestWriteProjectPreservesUnknownKeys(t *testing.T) {
	tmpDir := t.TempDir()
	// Write a config with an unknown key.
	initial := []byte(`{"project":"old","unknownKey":"value"}`)
	if err := os.WriteFile(filepath.Join(tmpDir, ".cvox.json"), initial, 0o644); err != nil {
		t.Fatal(err)
	}
	notificationMsg := "new notification"
	stopMsg := "new stop"
	ttsEnabled := true
	desktopEnabled := false
	if err := WriteProject(tmpDir, ProjectInput{
		Project:         "new",
		NotificationMsg: &notificationMsg,
		StopMsg:         &stopMsg,
		TTSEnabled:      &ttsEnabled,
		DesktopEnabled:  &desktopEnabled,
	}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, ".cvox.json"))
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, `"unknownKey": "value"`) {
		t.Error("WriteProject: unknown key should be preserved")
	}
	if !strings.Contains(contentStr, `"project": "new"`) {
		t.Error("WriteProject: project should be updated")
	}
}

func TestRemoveProject(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".cvox.json")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !RemoveProject(tmpDir) {
		t.Error("RemoveProject: should return true when file existed")
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Error("RemoveProject: file should be deleted")
	}
	if RemoveProject(tmpDir) {
		t.Error("RemoveProject: should return false when file didn't exist")
	}
}

func TestWriteProjectInheritsNilFields(t *testing.T) {
	tmpDir := t.TempDir()
	// Write with nil fields (inherit).
	if err := WriteProject(tmpDir, ProjectInput{
		Project: "testproj",
		// All other fields are nil → inherit.
	}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, ".cvox.json"))
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)
	// Only project should be written.
	if !strings.Contains(contentStr, `"project": "testproj"`) {
		t.Error("WriteProject: project should be written")
	}
	if strings.Contains(contentStr, `"hooks"`) {
		t.Error("WriteProject: hooks should not be written when nil")
	}
	if strings.Contains(contentStr, `"tts"`) {
		t.Error("WriteProject: tts should not be written when nil")
	}
	if strings.Contains(contentStr, `"desktop"`) {
		t.Error("WriteProject: desktop should not be written when nil")
	}
}

func TestWriteProjectDeletesInheritedFields(t *testing.T) {
	tmpDir := t.TempDir()
	// Start with a full config.
	notificationMsg := "old notification"
	stopMsg := "old stop"
	ttsEnabled := true
	desktopEnabled := true
	if err := WriteProject(tmpDir, ProjectInput{
		Project:         "testproj",
		NotificationMsg: &notificationMsg,
		StopMsg:         &stopMsg,
		TTSEnabled:      &ttsEnabled,
		DesktopEnabled:  &desktopEnabled,
	}); err != nil {
		t.Fatal(err)
	}
	// Re-run with nil fields (inherit) → should delete the explicit values.
	if err := WriteProject(tmpDir, ProjectInput{
		Project: "testproj",
		// All other fields nil → delete existing keys.
	}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(tmpDir, ".cvox.json"))
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, `"project": "testproj"`) {
		t.Error("WriteProject: project should remain")
	}
	if strings.Contains(contentStr, `"hooks"`) {
		t.Error("WriteProject: hooks should be deleted when nil (empty parent pruned)")
	}
	if strings.Contains(contentStr, `"tts"`) {
		t.Error("WriteProject: tts should be deleted when nil (empty parent pruned)")
	}
	if strings.Contains(contentStr, `"desktop"`) {
		t.Error("WriteProject: desktop should be deleted when nil (empty parent pruned)")
	}
}
