package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	c, err := config.Load(filepath.Join(t.TempDir(), "does-not-exist.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if c.DefaultBranch != "main" {
		t.Errorf("expected defaults, got DefaultBranch=%q", c.DefaultBranch)
	}
}

func TestLoadFromFile(t *testing.T) {
	tests := []struct {
		name   string
		yaml   map[string]any
		check  func(*testing.T, *config.Config)
	}{
		{
			name: "majorVersion",
			yaml: map[string]any{"majorVersion": 2},
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
			yaml: map[string]any{"majorVersion": 1, "mode": "candidate"},
			check: func(t *testing.T, c *config.Config) {
				if c.Mode != config.CandidateToRelease {
					t.Errorf("Mode: got %v, want CandidateToRelease", c.Mode)
				}
			},
		},
		{
			name: "releaseMode",
			yaml: map[string]any{"majorVersion": 1, "mode": "release"},
			check: func(t *testing.T, c *config.Config) {
				if c.Mode != config.ReleaseOnly {
					t.Errorf("Mode: got %v, want ReleaseOnly", c.Mode)
				}
			},
		},
		{
			name: "overrides",
			yaml: map[string]any{
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
			path := writeYAML(t, tt.yaml)
			c, err := config.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, c)
		})
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yml")
	if err := os.WriteFile(path, []byte("not valid yaml\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(path); err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func writeYAML(t *testing.T, fields map[string]any) string {
	t.Helper()
	var sb strings.Builder
	for k, v := range fields {
		sb.WriteString(fmt.Sprintf("%s: %v\n", k, v))
	}
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
