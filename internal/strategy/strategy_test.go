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

	defaultBranch := &branch.Branch{Category: branch.Default, Name: "main"}
	patchBranch := &branch.Branch{Category: branch.Patch, Name: "support/2.1.0"}
	devBranch := &branch.Branch{Category: branch.Dev, Name: "feature/acme-123", BuildTicket: "acme.123"}

	tests := []struct {
		name    string
		cfg     *config.Config
		b       *branch.Branch
		last    string
		want    string
		wantErr bool
	}{
		// default branch — release mode
		{"no prior version", releaseCfg, defaultBranch, "", "r2.0.0", false},
		{"prior major below config", releaseCfg, defaultBranch, "r1.0.0", "r2.0.0", false},
		{"prior major equals config", releaseCfg, defaultBranch, "r2.0.0", "r2.1.0", false},
		{"prior major above config", releaseCfg, defaultBranch, "r3.0.0", "", true},

		// default branch — candidate mode
		{"candidate mode, no prior", candidateCfg, defaultBranch, "", "c2.0.0", false},
		{"candidate mode, increment", candidateCfg, defaultBranch, "c2.1.0", "c2.2.0", false},

		// patch branch
		{"patch, no prior", releaseCfg, patchBranch, "", "", true},
		{"patch, has prior", releaseCfg, patchBranch, "r2.1.0", "r2.1.1", false},

		// dev branch
		{"dev, no prior", releaseCfg, devBranch, "", "d2.0.0+acme.123.0", false},
		{"dev, prior release", releaseCfg, devBranch, "r2.1.0", "d2.1.0+acme.123.0", false},
		{"dev, prior dev same ticket", releaseCfg, devBranch, "d2.1.0+acme.123.0", "d2.1.0+acme.123.1", false},
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
			calc := strategy.NewNewVersionCalculator(tt.cfg)
			got, err := calc.Resolve(tt.b, last)
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
			name: "default branch returns last version",
			stub: &testutil.StubRepository{LastVersionVal: "r1.4.0"},
			b:    &branch.Branch{Category: branch.Default, Name: "main"},
			want: "r1.4.0",
		},
		{
			name: "patch branch returns last version",
			stub: &testutil.StubRepository{LastVersionVal: "r1.4.0"},
			b:    &branch.Branch{Category: branch.Patch, Name: "support/1.4.x"},
			want: "r1.4.0",
		},
		{
			name:   "dev branch calls LastVersionForDevelopment with ticket",
			stub:   &testutil.StubRepository{LastVersionForDevVal: "d1.4.0+acme.123.2"},
			b:      &branch.Branch{Category: branch.Dev, Name: "feature/acme-123", BuildTicket: "acme.123"},
			want:   "d1.4.0+acme.123.2",
			ticket: "acme.123",
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
