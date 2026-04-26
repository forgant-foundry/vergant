package tool

import (
	"fmt"

	"github.com/forgant-foundry/vergant/internal/branch"
	"github.com/forgant-foundry/vergant/internal/config"
	"github.com/forgant-foundry/vergant/internal/git"
	"github.com/forgant-foundry/vergant/internal/strategy"
	"github.com/forgant-foundry/vergant/internal/version"
)

// VersioningTool coordinates version tag operations for a repository.
type VersioningTool struct {
	config *config.Config
	git    git.Repository
}

// New returns a VersioningTool for the given config and git client.
func New(cfg *config.Config, g git.Repository) *VersioningTool {
	return &VersioningTool{config: cfg, git: g}
}

// FetchTags fetches all tags from the remote.
func (t *VersioningTool) FetchTags() error {
	return t.git.FetchTags()
}

// NewVersion calculates the next version tag without creating it.
func (t *VersioningTool) NewVersion() (*version.Version, error) {
	b, err := t.currentBranch()
	if err != nil {
		return nil, err
	}
	acquire := strategy.NewAcquireLastVersion(t.git).Resolve(b)
	last, err := acquire()
	if err != nil {
		return nil, err
	}
	var lastRelease *version.Version
	if b.Category == branch.Dev || b.Category == branch.Patch {
		tag, err := t.git.LastRelease()
		if err != nil {
			return nil, err
		}
		lastRelease, err = version.Parse(tag)
		if err != nil {
			return nil, err
		}
	}
	calculate := strategy.NewNewVersionCalculator(t.config).Resolve(b, lastRelease)
	return calculate(last)
}

// LastVersion returns the most recent release or candidate version, or nil.
func (t *VersioningTool) LastVersion() (*version.Version, error) {
	b, err := t.currentBranch()
	if err != nil {
		return nil, err
	}
	return strategy.NewAcquireLastVersion(t.git).Resolve(b)()
}

// LastRelease returns the most recent release version, or nil.
func (t *VersioningTool) LastRelease() (*version.Version, error) {
	tag, err := t.git.LastRelease()
	if err != nil {
		return nil, err
	}
	return version.Parse(tag)
}

// Promote returns the release version corresponding to the given candidate string.
// It does not create the tag; call Tag with the result to apply it.
func (t *VersioningTool) Promote(v string) (*version.Version, error) {
	candidate, err := version.CoerceToCandidate(v)
	if err != nil {
		return nil, err
	}
	release := candidate.WithCategory(version.Release)

	tags, err := t.git.CurrentTags()
	if err != nil {
		return nil, err
	}
	for _, tag := range tags {
		if tag == release.RenderCategorized() {
			return nil, fmt.Errorf("a release of that candidate already exists")
		}
	}
	for _, tag := range tags {
		if tag == candidate.RenderCategorized() {
			return release, nil
		}
	}
	return nil, fmt.Errorf("attempted to promote candidate %s, but current commit isn't tagged as that candidate",
		candidate.RenderCategorized())
}

// Tag creates an annotated tag and pushes it to the remote.
func (t *VersioningTool) Tag(v *version.Version) error {
	return t.git.TagVersion(v)
}

// ListTags returns all reachable tags sorted by date, newline-separated.
func (t *VersioningTool) ListTags() (string, error) {
	return t.git.ListTags()
}

func (t *VersioningTool) currentBranch() (*branch.Branch, error) {
	name, err := t.git.CurrentBranch()
	if err != nil {
		return nil, err
	}
	return branch.ForName(t.config, name)
}
