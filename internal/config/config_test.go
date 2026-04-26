package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/forgant-foundry/vergant/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	c, err := config.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.MajorVersion != 0 {
		t.Errorf("MajorVersion: got %d, want 0", c.MajorVersion)
	}
	if c.DefaultBranch != "main" {
		t.Errorf("DefaultBranch: got %q, want main", c.DefaultBranch)
	}
	if c.Mode != config.ReleaseOnly {
		t.Errorf("Mode: got %v, want ReleaseOnly", c.Mode)
	}
}

func TestLoadMissingFile(t *testing.T) {
	c, err := config.Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatal(err)
	}
	if c.DefaultBranch != "main" {
		t.Errorf("expected defaults, got DefaultBranch=%q", c.DefaultBranch)
	}
}

func TestLoadFromFile(t *testing.T) {
	tests := []struct {
		name    string
		json    map[string]any
		check   func(*testing.T, *config.Config)
	}{
		{
			name: "majorVersion",
			json: map[string]any{"majorVersion": 2},
			check: func(t *testing.T, c *config.Config) {
				if c.MajorVersion != 2 {
					t.Errorf("MajorVersion: got %d, want 2", c.MajorVersion)
				}
				if c.DefaultBranch != "main" {
					t.Errorf("defaults preserved: DefaultBranch=%q", c.DefaultBranch)
				}
			},
		},
		{
			name: "candidateMode",
			json: map[string]any{"majorVersion": 1, "mode": "candidate"},
			check: func(t *testing.T, c *config.Config) {
				if c.Mode != config.CandidateToRelease {
					t.Errorf("Mode: got %v, want CandidateToRelease", c.Mode)
				}
			},
		},
		{
			name: "releaseMode",
			json: map[string]any{"majorVersion": 1, "mode": "release"},
			check: func(t *testing.T, c *config.Config) {
				if c.Mode != config.ReleaseOnly {
					t.Errorf("Mode: got %v, want ReleaseOnly", c.Mode)
				}
			},
		},
		{
			name: "overrides",
			json: map[string]any{
				"majorVersion":     1,
				"defaultBranch":    "dev",
				"patchBranchRegEx": `^release\/.*`,
			},
			check: func(t *testing.T, c *config.Config) {
				if c.DefaultBranch != "dev" {
					t.Errorf("DefaultBranch: got %q", c.DefaultBranch)
				}
				if c.PatchBranchRegEx != `^release\/.*` {
					t.Errorf("PatchBranchRegEx: got %q", c.PatchBranchRegEx)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeJSON(t, tt.json)
			c, err := config.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, c)
		})
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(path); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func writeJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
