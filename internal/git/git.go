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
	dir      string
	prefixes version.Prefixes
	path     string
}

// NewGitRepository returns a Git client rooted at dir using the given tag prefix configuration.
func NewGitRepository(dir string, p version.Prefixes) *Git {
	return &Git{dir: dir, prefixes: p}
}

// WithPath returns a copy of the Git client scoped to a version sequence path.
// All tag queries and writes are restricted to tags prefixed with path + "/".
// An empty path restores default single-sequence behaviour.
func (g *Git) WithPath(path string) *Git {
	n := *g
	n.path = path
	return &n
}

// fullTagName prepends the path prefix to a version tag name for use in git commands.
func (g *Git) fullTagName(tag string) string {
	if g.path == "" {
		return tag
	}
	return g.path + "/" + tag
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
// When a path is set, only tags under that path are searched and the path prefix is stripped
// from the returned tag name.
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
	prefix := g.path + "/"
	for _, tag := range strings.Split(out, "\n") {
		if g.path != "" {
			if !strings.HasPrefix(tag, prefix) {
				continue
			}
			tag = tag[len(prefix):]
		}
		if re.MatchString(tag) {
			return tag, nil
		}
	}
	return "", nil
}

// LastRelease returns the most recent release tag (e.g. "v1.2.3"), or "".
func (g *Git) LastRelease() (string, error) {
	pattern := regexp.QuoteMeta(g.prefixes.Release) + `\d+\.\d+\.\d+$`
	return g.FindTag(pattern)
}

// LastVersion returns the most recent release or candidate tag, or "".
func (g *Git) LastVersion() (string, error) {
	rp := regexp.QuoteMeta(g.prefixes.Release)
	cp := regexp.QuoteMeta(g.prefixes.Candidate)
	pattern := fmt.Sprintf(`(?:%s|%s)\d+\.\d+\.\d+$`, rp, cp)
	return g.FindTag(pattern)
}

// LastVersionForDevelopment returns the most recent dev tag for buildTicket, or "" if none exists.
func (g *Git) LastVersionForDevelopment(buildTicket string) (string, error) {
	dp := regexp.QuoteMeta(g.prefixes.Dev)
	pattern := fmt.Sprintf(`%s\d+\.\d+\.\d+-%s\.\d+$`, dp, regexp.QuoteMeta(buildTicket))
	return g.FindTag(pattern)
}

// CurrentTags returns all tags pointing at HEAD scoped to the configured path.
// The path prefix is stripped from returned tag names.
func (g *Git) CurrentTags() ([]string, error) {
	out, err := g.run("tag", "--points-at", "HEAD")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	if g.path == "" {
		return strings.Fields(out), nil
	}
	prefix := g.path + "/"
	var filtered []string
	for _, tag := range strings.Fields(out) {
		if strings.HasPrefix(tag, prefix) {
			filtered = append(filtered, tag[len(prefix):])
		}
	}
	return filtered, nil
}

// ListTags returns all merged tags sorted by tagger date (descending), newline-separated.
// When a path is set, only tags under that path are listed and the path prefix is stripped.
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
	if g.path == "" {
		return string(out), nil
	}
	prefix := g.path + "/"
	var sb strings.Builder
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, prefix) {
			sb.WriteString(line[len(prefix):])
			sb.WriteByte('\n')
		}
	}
	return sb.String(), nil
}

// TagVersion creates an annotated tag and pushes it to origin.
// When a path is set, the tag name is prefixed with path + "/".
func (g *Git) TagVersion(v *version.Version) error {
	tagName := g.fullTagName(v.RenderCategorized())
	if _, err := g.run("tag", "-a", tagName, "-m", v.RenderMessage()); err != nil {
		return err
	}
	_, err := g.run("push", "--tags", "origin", tagName)
	return err
}
