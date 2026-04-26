package branch_test

import (
	"testing"

	"github.com/forgant-foundry/vergant/internal/branch"
	"github.com/forgant-foundry/vergant/internal/config"
)

var defaultCfg = &config.Config{
	MajorVersion:     1,
	DefaultBranch:    "main",
	PatchBranchRegEx: `^support\/.*`,
	DevBranchRegEx:   `^.*?\/*(\w+-\d+)\D*`,
}

func TestDefaultBranch(t *testing.T) {
	b, err := branch.ForName(defaultCfg, "main")
	if err != nil {
		t.Fatal(err)
	}
	if b.Category != branch.Default {
		t.Errorf("category: got %v, want Default", b.Category)
	}
}

func TestDefaultBranchOverride(t *testing.T) {
	cfg := &config.Config{
		DefaultBranch:    "dev",
		PatchBranchRegEx: defaultCfg.PatchBranchRegEx,
		DevBranchRegEx:   defaultCfg.DevBranchRegEx,
	}
	b, err := branch.ForName(cfg, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if b.Category != branch.Default {
		t.Errorf("category: got %v, want Default", b.Category)
	}
}

func TestPatchBranch(t *testing.T) {
	tests := []string{"support/1.0.0", "support/2.3.x"}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := branch.ForName(defaultCfg, name)
			if err != nil {
				t.Fatal(err)
			}
			if b.Category != branch.Patch {
				t.Errorf("category: got %v, want Patch", b.Category)
			}
		})
	}
}

func TestDevBranch(t *testing.T) {
	tests := []struct {
		name   string
		ticket string
	}{
		{"feature/ACME-123_some_branch", "acme.123"},
		{"feature/ACME-123", "acme.123"},
		{"ACME-123", "acme.123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := branch.ForName(defaultCfg, tt.name)
			if err != nil {
				t.Fatal(err)
			}
			if b.Category != branch.Dev {
				t.Errorf("category: got %v, want Dev", b.Category)
			}
			if b.BuildTicket != tt.ticket {
				t.Errorf("ticket: got %q, want %q", b.BuildTicket, tt.ticket)
			}
		})
	}
}

func TestUnsupportedBranch(t *testing.T) {
	if _, err := branch.ForName(defaultCfg, "some_unsupported_branch"); err == nil {
		t.Fatal("expected error for unsupported branch")
	}
}
