package strategy_test

import (
	"testing"

	"github.com/forgant-foundry/vergant/internal/branch"
	"github.com/forgant-foundry/vergant/internal/config"
	"github.com/forgant-foundry/vergant/internal/strategy"
	"github.com/forgant-foundry/vergant/internal/testutil"
	"github.com/forgant-foundry/vergant/internal/version"
)

// --- NewVersionCalculator ---

func TestNewVersionCalculator(t *testing.T) {
	releaseCfg := &config.Config{
		MajorVersion:  2,
		DefaultBranch: "main",
		Mode:          config.ReleaseOnly,
	}
	candidateCfg := &config.Config{
		MajorVersion:  2,
		DefaultBranch: "main",
		Mode:          config.CandidateToRelease,
	}

	mainBranch     := &branch.Branch{Category: branch.Default, Name: "main"}
	supportBranch  := &branch.Branch{Category: branch.Support, Name: "support/2.1.0"}
	devBranch      := &branch.Branch{Category: branch.Dev, Name: "dev/acme-123", BuildTicket: "acme.123"}
	patchDevBranch := &branch.Branch{Category: branch.Patch, Name: "patch/acme-456", BuildTicket: "acme.456"}

	tests := []struct {
		name        string
		cfg         *config.Config
		b           *branch.Branch
		last        string // last dev tag (dev/patchDev branches) or last release/candidate (others)
		lastRelease string // last release tag, used by dev and patchDev branches
		want        string
		wantErr     bool
	}{
		// Scenario 1: main branch
		// majorVersion > last major → new major release
		{"main: no prior tags", releaseCfg, mainBranch, "", "", "r2.0.0", false},
		{"main: prior major below config", releaseCfg, mainBranch, "r1.0.0", "", "r2.0.0", false},
		// majorVersion == last major → increment minor
		{"main: prior major equals config", releaseCfg, mainBranch, "r2.0.0", "", "r2.1.0", false},
		{"main: candidate mode", candidateCfg, mainBranch, "c2.1.0", "", "c2.2.0", false},
		// prior major > config → error
		{"main: prior major above config", releaseCfg, mainBranch, "r3.0.0", "", "", true},

		// Scenario 2: support/* branch — patch increment only, major never changes
		{"support: no prior", releaseCfg, supportBranch, "", "", "", true},
		{"support: increment patch", releaseCfg, supportBranch, "r2.1.0", "", "r2.1.1", false},
		{"support: consecutive patches", releaseCfg, supportBranch, "r2.1.3", "", "r2.1.4", false},

		// Scenario 3: dev/* branch — pre-release targeting next minor (or major if config advances)
		// No prior release: use config majorVersion as target
		{"dev: no prior release", releaseCfg, devBranch, "", "", "d2.0.0-acme.123.0", false},
		// Config majorVersion > lastRelease major: pre-release of new major
		{"dev: config major advances", &config.Config{MajorVersion: 2, DefaultBranch: "main"}, devBranch, "", "r1.4.0", "d2.0.0-acme.123.0", false},
		// Normal case: pre-release of next minor
		{"dev: first pre-release", releaseCfg, devBranch, "", "r2.1.0", "d2.2.0-acme.123.0", false},
		// Existing pre-release matches target: increment counter
		{"dev: increment counter", releaseCfg, devBranch, "d2.2.0-acme.123.2", "r2.1.0", "d2.2.0-acme.123.3", false},
		// Existing pre-release base differs from new target (e.g. after rebase): reset counter
		{"dev: target changes on rebase", releaseCfg, devBranch, "d2.1.0-acme.123.5", "r2.1.0", "d2.2.0-acme.123.0", false},
		// Config major above lastRelease major: error
		{"dev: config major below release", &config.Config{MajorVersion: 1, DefaultBranch: "main"}, devBranch, "", "r2.0.0", "", true},

		// Scenario 4: patch/* branch — pre-release targeting next patch
		// No prior release: error (nothing to patch)
		{"patchdev: no prior release", releaseCfg, patchDevBranch, "", "", "", true},
		// Normal case: pre-release of next patch
		{"patchdev: first pre-release", releaseCfg, patchDevBranch, "", "r2.1.0", "d2.1.1-acme.456.0", false},
		// Existing pre-release matches target: increment counter
		{"patchdev: increment counter", releaseCfg, patchDevBranch, "d2.1.1-acme.456.1", "r2.1.0", "d2.1.1-acme.456.2", false},
		// Existing pre-release base differs from target: reset counter
		{"patchdev: target changes", releaseCfg, patchDevBranch, "d2.0.1-acme.456.3", "r2.1.0", "d2.1.1-acme.456.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var last *version.Version
			if tt.last != "" {
				v, err := version.Parse(tt.last)
				if err != nil {
					t.Fatal(err)
				}
				last = v
			}
			var lastRelease *version.Version
			if tt.lastRelease != "" {
				v, err := version.Parse(tt.lastRelease)
				if err != nil {
					t.Fatal(err)
				}
				lastRelease = v
			}
			calc := strategy.NewNewVersionCalculator(tt.cfg)
			got, err := calc.Resolve(tt.b, last, lastRelease)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.RenderCategorized() != tt.want {
				t.Errorf("got %q, want %q", got.RenderCategorized(), tt.want)
			}
		})
	}
}

// --- AcquireLastVersion ---

func TestAcquireLastVersion(t *testing.T) {
	tests := []struct {
		name   string
		stub   *testutil.StubRepository
		b      *branch.Branch
		want   string // empty means expect nil
		ticket string // non-empty: assert LastVersionForDevelopment was called with this value
	}{
		{
			name: "main branch returns last version",
			stub: &testutil.StubRepository{LastVersionVal: "r1.4.0"},
			b:    &branch.Branch{Category: branch.Default, Name: "main"},
			want: "r1.4.0",
		},
		{
			name: "support branch returns last version",
			stub: &testutil.StubRepository{LastVersionVal: "r1.4.0"},
			b:    &branch.Branch{Category: branch.Support, Name: "support/1.4.x"},
			want: "r1.4.0",
		},
		{
			name:   "dev branch calls LastVersionForDevelopment with ticket",
			stub:   &testutil.StubRepository{LastVersionForDevVal: "d1.5.0-acme.123.2"},
			b:      &branch.Branch{Category: branch.Dev, Name: "dev/acme-123", BuildTicket: "acme.123"},
			want:   "d1.5.0-acme.123.2",
			ticket: "acme.123",
		},
		{
			name:   "dev branch, no dev tag returns nil",
			stub:   &testutil.StubRepository{LastVersionForDevVal: ""},
			b:      &branch.Branch{Category: branch.Dev, Name: "dev/acme-123", BuildTicket: "acme.123"},
			want:   "",
			ticket: "acme.123",
		},
		{
			name:   "patch dev branch calls LastVersionForDevelopment with ticket",
			stub:   &testutil.StubRepository{LastVersionForDevVal: "d1.4.1-acme.456.0"},
			b:      &branch.Branch{Category: branch.Patch, Name: "patch/acme-456", BuildTicket: "acme.456"},
			want:   "d1.4.1-acme.456.0",
			ticket: "acme.456",
		},
		{
			name: "no tags returns nil",
			stub: &testutil.StubRepository{},
			b:    &branch.Branch{Category: branch.Default, Name: "main"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotTicket string
			if tt.ticket != "" {
				tt.stub.LastVersionForDevFn = func(ticket string) (string, error) {
					gotTicket = ticket
					return tt.stub.LastVersionForDevVal, nil
				}
			}

			v, err := strategy.NewAcquireLastVersion(tt.stub).Resolve(tt.b)
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == "" {
				if v != nil {
					t.Errorf("expected nil, got %q", v.RenderCategorized())
				}
				return
			}
			if v.RenderCategorized() != tt.want {
				t.Errorf("got %q, want %q", v.RenderCategorized(), tt.want)
			}
			if tt.ticket != "" && gotTicket != tt.ticket {
				t.Errorf("LastVersionForDevelopment called with ticket %q, want %q", gotTicket, tt.ticket)
			}
		})
	}
}
