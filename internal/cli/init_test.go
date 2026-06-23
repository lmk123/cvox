package cli

import (
	"testing"

	"github.com/lmk123/cvox/internal/config"
)

// strPtr returns a pointer to s.
func strPtr(s string) *string {
	return &s
}

func TestDefaultProjectName(t *testing.T) {
	tests := []struct {
		name   string
		cwd    string
		home   string
		global bool
		want   string
	}{
		{
			name:   "non-global uses cwd base",
			cwd:    "/Users/mica/projects/myproject",
			home:   "/Users/mica",
			global: false,
			want:   "myproject",
		},
		{
			name:   "global uses home base",
			cwd:    "/Users/mica/projects/myproject",
			home:   "/Users/mica",
			global: true,
			want:   "mica",
		},
		{
			name:   "global with nested home path",
			cwd:    "/Users/mica/projects/myproject",
			home:   "/home/alice",
			global: true,
			want:   "alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultProjectName(tt.cwd, tt.home, tt.global)
			if got != tt.want {
				t.Errorf("defaultProjectName(%q, %q, %v) = %q, want %q",
					tt.cwd, tt.home, tt.global, got, tt.want)
			}
		})
	}
}

func TestBoolPtrEqual(t *testing.T) {
	tests := []struct {
		a, b *bool
		want bool
	}{
		{nil, nil, true},
		{ptr(true), ptr(true), true},
		{ptr(false), ptr(false), true},
		{ptr(true), ptr(false), false},
		{nil, ptr(true), false},
		{ptr(true), nil, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := boolPtrEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("boolPtrEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDefaultLocaleKey(t *testing.T) {
	tests := []struct {
		name string
		p    *config.Partial
		want string
	}{
		{
			name: "nil returns default",
			p:    nil,
			want: "1",
		},
		{
			name: "missing hooks returns default",
			p:    &config.Partial{},
			want: "1",
		},
		{
			name: "English message returns English key",
			p: &config.Partial{
				Hooks: &config.PartialHooks{
					Notification: &config.PartialEvent{
						Message: strPtr("Claude Code needs permission, from {project}"),
					},
				},
			},
			want: "1",
		},
		{
			name: "Chinese message returns Chinese key",
			p: &config.Partial{
				Hooks: &config.PartialHooks{
					Notification: &config.PartialEvent{
						Message: strPtr("Claude Code 需要权限，来自 {project}"),
					},
				},
			},
			want: "2",
		},
		{
			name: "Japanese message returns Japanese key",
			p: &config.Partial{
				Hooks: &config.PartialHooks{
					Notification: &config.PartialEvent{
						Message: strPtr("Claude Code 権限が必要です、{project} より"),
					},
				},
			},
			want: "3",
		},
		{
			name: "Korean message returns Korean key",
			p: &config.Partial{
				Hooks: &config.PartialHooks{
					Notification: &config.PartialEvent{
						Message: strPtr("Claude Code 권한이 필요합니다, {project} 에서"),
					},
				},
			},
			want: "4",
		},
		{
			name: "custom message returns default",
			p: &config.Partial{
				Hooks: &config.PartialHooks{
					Notification: &config.PartialEvent{
						Message: strPtr("custom message"),
					},
				},
			},
			want: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultLocaleKey(tt.p)
			if got != tt.want {
				t.Errorf("defaultLocaleKey(%v) = %q, want %q", tt.p, got, tt.want)
			}
		})
	}
}

func TestDefaultMethodKey(t *testing.T) {
	tests := []struct {
		name string
		p    *config.Partial
		want string
	}{
		{
			name: "nil returns default",
			p:    nil,
			want: "1",
		},
		{
			name: "missing fields returns default",
			p:    &config.Partial{},
			want: "1",
		},
		{
			name: "Voice only returns Voice only key",
			p: &config.Partial{
				TTS:     &config.PartialToggle{Enabled: ptr(true)},
				Desktop: &config.PartialToggle{Enabled: ptr(false)},
			},
			want: "1",
		},
		{
			name: "Desktop only returns Desktop only key",
			p: &config.Partial{
				TTS:     &config.PartialToggle{Enabled: ptr(false)},
				Desktop: &config.PartialToggle{Enabled: ptr(true)},
			},
			want: "2",
		},
		{
			name: "Both returns Both key",
			p: &config.Partial{
				TTS:     &config.PartialToggle{Enabled: ptr(true)},
				Desktop: &config.PartialToggle{Enabled: ptr(true)},
			},
			want: "3",
		},
		{
			name: "TTS nil, Desktop false returns default (no match)",
			p: &config.Partial{
				TTS:     &config.PartialToggle{Enabled: nil},
				Desktop: &config.PartialToggle{Enabled: ptr(false)},
			},
			want: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultMethodKey(tt.p)
			if got != tt.want {
				t.Errorf("defaultMethodKey(%v) = %q, want %q", tt.p, got, tt.want)
			}
		})
	}
}
