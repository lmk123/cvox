package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// No config files: defaults only, tts/desktop off, project from cwd.
	cwd, _ := os.Getwd()
	cfg := Load(cwd)
	if cfg.Project != filepath.Base(cwd) {
		t.Errorf("Load: project = %q, want %q", cfg.Project, filepath.Base(cwd))
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
	if err := WriteProject(tmpDir, ProjectInput{
		Project:         "testproj",
		NotificationMsg: "test notification",
		StopMsg:         "test stop",
		TTSEnabled:      true,
		DesktopEnabled:  false,
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
	if err := WriteProject(globalDir, ProjectInput{
		Project:         "", // not set: should fall back to cwd
		NotificationMsg: "global notification",
		StopMsg:         "global stop",
		TTSEnabled:      false,
		DesktopEnabled:  true,
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
	if err := WriteProject(globalDir, ProjectInput{
		Project:         "globalproj",
		NotificationMsg: "global notification",
		StopMsg:         "global stop",
		TTSEnabled:      false,
		DesktopEnabled:  true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := WriteProject(tmpDir, ProjectInput{
		Project:         "proj",
		NotificationMsg: "proj notification",
		StopMsg:         "proj stop",
		TTSEnabled:      true,
		DesktopEnabled:  false,
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
	if err := WriteProject(tmpDir, ProjectInput{
		Project:         "new",
		NotificationMsg: "new notification",
		StopMsg:         "new stop",
		TTSEnabled:      true,
		DesktopEnabled:  false,
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
