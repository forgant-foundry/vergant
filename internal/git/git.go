package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/forgant-foundry/vergant/internal/version"
)

// Repository is the interface for git operations consumed by the versioning logic.
type Repository interface {
	FetchTags() error
	CurrentBranch() (string, error)
	LastVersion() (string, error)
	LastRelease() (string, error)
	LastVersionForDevelopment(buildTicket string) (string, error)
	CurrentTags() ([]string, error)
	ListTags() (string, error)
	TagVersion(v *version.Version) error
}

// Git runs git operations in a specific directory.
type Git struct {
	dir string
}

// New returns a Git client rooted at dir.
func New(dir string) *Git {
	return &Git{dir: dir}
}

func (g *Git) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimRight(string(ee.Stderr), "\r\n"))
		}
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// FetchTags fetches all tags from the remote origin.
func (g *Git) FetchTags() error {
	_, err := g.run("fetch", "--tags")
	return err
}

// CurrentBranch returns the name of the currently checked-out branch.
func (g *Git) CurrentBranch() (string, error) {
	return g.run("rev-parse", "--abbrev-ref", "HEAD")
}

// FindTag searches annotated tags (sorted descending by tag date, including merged ancestors)
// and returns the first tag matching pattern, or "" if none found.
func (g *Git) FindTag(pattern string) (string, error) {
	out, err := g.run("tag", "--list", "--sort=-taggerdate", "--merged")
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid tag pattern %q: %w", pattern, err)
	}
	for _, tag := range strings.Split(out, "\n") {
		if re.MatchString(tag) {
			return tag, nil
		}
	}
	return "", nil
}

// LastRelease returns the most recent release tag (e.g. "r1.2.3"), or "".
func (g *Git) LastRelease() (string, error) {
	return g.FindTag(`r\d+\.\d+\.\d+$`)
}

// LastVersion returns the most recent release or candidate tag, or "".
func (g *Git) LastVersion() (string, error) {
	return g.FindTag(`[cr]\d+\.\d+\.\d+$`)
}

// LastVersionForDevelopment returns the most recent dev tag for buildTicket, or "" if none exists.
func (g *Git) LastVersionForDevelopment(buildTicket string) (string, error) {
	pattern := fmt.Sprintf(`d\d+\.\d+\.\d+-%s\.\d+$`, regexp.QuoteMeta(buildTicket))
	return g.FindTag(pattern)
}

// CurrentTags returns all tags pointing at HEAD.
func (g *Git) CurrentTags() ([]string, error) {
	out, err := g.run("tag", "--points-at", "HEAD")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Fields(out), nil
}

// ListTags returns all merged tags sorted by tagger date (descending), newline-separated.
func (g *Git) ListTags() (string, error) {
	cmd := exec.Command("git", "tag", "--list", "--sort=-taggerdate", "--merged")
	cmd.Dir = g.dir
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", strings.TrimRight(string(ee.Stderr), "\r\n"))
		}
		return "", err
	}
	return string(out), nil
}

// TagVersion creates an annotated tag and pushes it to origin.
func (g *Git) TagVersion(v *version.Version) error {
	if _, err := g.run("tag", "-a", v.RenderCategorized(), "-m", v.RenderMessage()); err != nil {
		return err
	}
	_, err := g.run("push", "--tags", "origin", v.RenderCategorized())
	return err
}
