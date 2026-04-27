package tool_test

import (
	"strings"
	"testing"

	"github.com/forgant-foundry/vergant/internal/config"
	"github.com/forgant-foundry/vergant/internal/testutil"
	"github.com/forgant-foundry/vergant/internal/tool"
	"github.com/forgant-foundry/vergant/internal/version"
)

func defaultConfig() *config.Config {
	return &config.Config{
		MajorVersion:        1,
		DefaultBranch:       "main",
		SupportBranchRegEx:    `^support\/.*`,
		DevBranchRegEx:      `^dev\/(.+)$`,
		PatchBranchRegEx: `^patch\/(.+)$`,
		Mode:                config.ReleaseOnly,
	}
}

func TestNewVersion(t *testing.T) {
	stub := &testutil.StubRepository{
		CurrentBranchVal: "main",
		LastVersionVal:   "v1.2.0",
	}
	v, err := tool.New(defaultConfig(), stub).NewVersion()
	if err != nil {
		t.Fatal(err)
	}
	if got := v.RenderCategorized(); got != "v1.3.0" {
		t.Errorf("got %q, want v1.3.0", got)
	}
}

func TestNewVersionDevBranch(t *testing.T) {
	stub := &testutil.StubRepository{
		CurrentBranchVal:     "dev/acme-123",
		LastVersionForDevVal: "d1.3.0-acme.123.0",
		LastReleaseVal:       "v1.2.0",
	}
	v, err := tool.New(defaultConfig(), stub).NewVersion()
	if err != nil {
		t.Fatal(err)
	}
	// last release v1.2.0 + minor = v1.3.0; last dev base matches target → increment counter
	if got := v.RenderCategorized(); got != "d1.3.0-acme.123.1" {
		t.Errorf("got %q, want d1.3.0-acme.123.1", got)
	}
}

func TestNewVersionPatchDevBranch(t *testing.T) {
	stub := &testutil.StubRepository{
		CurrentBranchVal:     "patch/acme-456",
		LastVersionForDevVal: "",
		LastReleaseVal:       "v1.2.0",
	}
	v, err := tool.New(defaultConfig(), stub).NewVersion()
	if err != nil {
		t.Fatal(err)
	}
	// last release v1.2.0 + patch = v1.2.1; no existing pre-release → start at 0
	if got := v.RenderCategorized(); got != "d1.2.1-acme.456.0" {
		t.Errorf("got %q, want d1.2.1-acme.456.0", got)
	}
}

func TestLastRelease(t *testing.T) {
	stub := &testutil.StubRepository{LastReleaseVal: "v1.1.0"}
	v, err := tool.New(defaultConfig(), stub).LastRelease()
	if err != nil {
		t.Fatal(err)
	}
	if got := v.RenderCategorized(); got != "v1.1.0" {
		t.Errorf("got %q, want v1.1.0", got)
	}
}

func TestPromote(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		currentTags []string
		want        string
		wantErr     string
	}{
		{
			name:        "candidate present",
			input:       "c1.2.0",
			currentTags: []string{"c1.2.0"},
			want:        "v1.2.0",
		},
		{
			name:        "bare version coerced",
			input:       "1.2.0",
			currentTags: []string{"c1.2.0"},
			want:        "v1.2.0",
		},
		{
			name:        "release already exists",
			input:       "c1.2.0",
			currentTags: []string{"c1.2.0", "v1.2.0"},
			wantErr:     "already exists",
		},
		{
			name:        "candidate absent",
			input:       "c1.3.0",
			currentTags: []string{"c1.2.0"},
			wantErr:     "isn't tagged as that candidate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &testutil.StubRepository{CurrentTagsVal: tt.currentTags}
			v, err := tool.New(defaultConfig(), stub).Promote(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if got := err.Error(); !strings.Contains(got, tt.wantErr) {
					t.Errorf("error %q does not contain %q", got, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := v.RenderCategorized(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTag(t *testing.T) {
	stub := &testutil.StubRepository{}
	v, _ := version.Parse("v1.3.0")
	if err := tool.New(defaultConfig(), stub).Tag(v); err != nil {
		t.Fatal(err)
	}
	if len(stub.Tagged) != 1 || stub.Tagged[0].RenderCategorized() != "v1.3.0" {
		t.Errorf("tagged versions: %v", stub.Tagged)
	}
}

func TestListTags(t *testing.T) {
	stub := &testutil.StubRepository{ListTagsVal: "v1.1.0\nv1.0.0\n"}
	got, err := tool.New(defaultConfig(), stub).ListTags()
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.1.0\nv1.0.0\n" {
		t.Errorf("got %q", got)
	}
}
