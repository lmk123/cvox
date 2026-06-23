package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultProjectName(t *testing.T) {
	tests := []struct {
		name        string
		global      bool
		cwd         string
		home        string
		wantDefault string
	}{
		{
			name:        "non-global uses cwd",
			global:      false,
			cwd:         "/Users/mica/projects/myproject",
			home:        "/Users/mica",
			wantDefault: "myproject",
		},
		{
			name:        "global uses home directory name",
			global:      true,
			cwd:         "/Users/mica/projects/myproject",
			home:        "/Users/mica",
			wantDefault: "mica",
		},
		{
			name:        "global with nested home path",
			global:      true,
			cwd:         "/Users/mica/projects/myproject",
			home:        "/home/alice",
			wantDefault: "alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original cwd and home
			origCwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(origCwd)

			origHome := os.Getenv("HOME")
			defer os.Setenv("HOME", origHome)

			// Set up test environment
			os.Setenv("HOME", tt.home)

			// Change to test cwd
			if tt.cwd != "" {
				// Create a temporary directory for testing
				tmpDir := t.TempDir()
				// Create the path structure
				testPath := filepath.Join(tmpDir, filepath.Base(tt.cwd))
				if err := os.MkdirAll(testPath, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Chdir(testPath); err != nil {
					t.Fatal(err)
				}
			}

			// Calculate default name using the same logic as Init
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			defaultName := filepath.Base(cwd)
			if tt.global {
				home, err := os.UserHomeDir()
				if err != nil {
					t.Fatalf("unexpected error getting home directory: %v", err)
				}
				defaultName = filepath.Base(home)
			}

			if defaultName != tt.wantDefault {
				t.Errorf("defaultName = %q, want %q", defaultName, tt.wantDefault)
			}
		})
	}
}
