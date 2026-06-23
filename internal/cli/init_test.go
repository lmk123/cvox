package cli

import "testing"

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
