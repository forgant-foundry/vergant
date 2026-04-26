package version_test

import (
	"testing"

	"github.com/forgant-foundry/vergant/internal/version"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input    string
		wantNil  bool
		wantCat  version.Category
		wantRaw  string
		wantErr  bool
	}{
		{input: "", wantNil: true},
		{input: "r1.2.3", wantCat: version.Release, wantRaw: "r1.2.3"},
		{input: "c1.2.3", wantCat: version.Candidate, wantRaw: "c1.2.3"},
		{input: "d1.2.3+asdf.123.0", wantCat: version.Dev, wantRaw: "d1.2.3+asdf.123.0"},
		{input: "v1.2.3", wantErr: true},
		{input: "rnot-a-version", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := version.Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil {
				if v != nil {
					t.Fatalf("expected nil, got %v", v.RenderCategorized())
				}
				return
			}
			if v.Category != tt.wantCat {
				t.Errorf("category: got %q, want %q", v.Category, tt.wantCat)
			}
			if v.RenderCategorized() != tt.wantRaw {
				t.Errorf("render: got %q, want %q", v.RenderCategorized(), tt.wantRaw)
			}
		})
	}
}

func TestParseFields(t *testing.T) {
	v := mustParse(t, "r1.2.3")
	if v.Major() != 1 || v.Minor() != 2 || v.Patch() != 3 {
		t.Errorf("got %d.%d.%d, want 1.2.3", v.Major(), v.Minor(), v.Patch())
	}

	d := mustParse(t, "d1.2.3+asdf.123.0")
	if got := d.RenderUncategorized(); got != "1.2.3+asdf.123.0" {
		t.Errorf("uncategorized: got %q", got)
	}
	if ticket := d.BuildTicket(); len(ticket) != 2 || ticket[0] != "asdf" || ticket[1] != "123" {
		t.Errorf("build ticket: got %v", ticket)
	}
}

func TestCoerceToCandidate(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "r1.2.3", want: "c1.2.3"},
		{input: "c1.2.3", want: "c1.2.3"},
		{input: "1.2.3", want: "c1.2.3"},
		{input: "asdf", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := version.CoerceToCandidate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
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

func TestNewMajor(t *testing.T) {
	v := version.NewMajor(4, version.Candidate)
	if got := v.RenderCategorized(); got != "c4.0.0" {
		t.Errorf("got %q, want c4.0.0", got)
	}
}

func TestNewBuild(t *testing.T) {
	base := mustParse(t, "r1.2.3")
	v := version.NewBuild(base, "asdf.123")
	if got := v.RenderCategorized(); got != "d1.2.3+asdf.123.0" {
		t.Errorf("got %q, want d1.2.3+asdf.123.0", got)
	}
}

func TestIncrements(t *testing.T) {
	r := mustParse(t, "r1.2.3")
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"major", r.IncrementMajor(version.Release).RenderCategorized(), "r2.0.0"},
		{"minor", r.IncrementMinor(version.Release).RenderCategorized(), "r1.3.0"},
		{"patch", r.IncrementPatch(version.Release).RenderCategorized(), "r1.2.4"},
		{"category", r.WithCategory(version.Candidate).RenderCategorized(), "c1.2.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestIncrementBuild(t *testing.T) {
	v := mustParse(t, "d1.2.3+asdf.123.1")
	if got := v.IncrementBuild().RenderCategorized(); got != "d1.2.3+asdf.123.2" {
		t.Errorf("got %q, want d1.2.3+asdf.123.2", got)
	}
}

func TestRenderMessage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"r1.2.3", "release 1.2.3"},
		{"c1.2.3", "candidate 1.2.3"},
		{"d1.2.3+asdf.123.2", "development 1.2.3+asdf.123.2"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v := mustParse(t, tt.input)
			if got := v.RenderMessage(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func mustParse(t *testing.T, s string) *version.Version {
	t.Helper()
	v, err := version.Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q): %v", s, err)
	}
	return v
}
