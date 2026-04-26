package strategy

import (
	"fmt"

	"github.com/forgant-foundry/vergant/internal/branch"
	"github.com/forgant-foundry/vergant/internal/config"
	"github.com/forgant-foundry/vergant/internal/git"
	"github.com/forgant-foundry/vergant/internal/version"
)

// AcquireLastVersion fetches the last known version from git tags.
type AcquireLastVersion struct {
	git git.Repository
}

func NewAcquireLastVersion(g git.Repository) *AcquireLastVersion {
	return &AcquireLastVersion{git: g}
}

// Resolve returns the relevant last version for b, or nil if no version exists yet.
func (a *AcquireLastVersion) Resolve(b *branch.Branch) (*version.Version, error) {
	if b.Category == branch.Dev {
		return a.lastVersionForDevelopment(b.BuildTicket)
	}
	return a.lastVersion()
}

func (a *AcquireLastVersion) lastVersion() (*version.Version, error) {
	tag, err := a.git.LastVersion()
	if err != nil {
		return nil, err
	}
	return version.Parse(tag)
}

func (a *AcquireLastVersion) lastVersionForDevelopment(buildTicket string) (*version.Version, error) {
	tag, err := a.git.LastVersionForDevelopment(buildTicket)
	if err != nil {
		return nil, err
	}
	return version.Parse(tag)
}

// NewVersionCalculator computes the next version given the current branch and last version.
type NewVersionCalculator struct {
	config *config.Config
}

func NewNewVersionCalculator(cfg *config.Config) *NewVersionCalculator {
	return &NewVersionCalculator{config: cfg}
}

func (c *NewVersionCalculator) defaultCategory() version.Category {
	if c.config.Mode == config.CandidateToRelease {
		return version.Candidate
	}
	return version.Release
}

// Resolve calculates the next version for b given the last known version.
func (c *NewVersionCalculator) Resolve(b *branch.Branch, last *version.Version) (*version.Version, error) {
	switch b.Category {
	case branch.Default:
		return c.onDefault(last)
	case branch.Patch:
		return c.onPatch(last)
	case branch.Dev:
		return c.onDev(b, last)
	default:
		return nil, fmt.Errorf("unsupported branch category")
	}
}

func (c *NewVersionCalculator) onDefault(last *version.Version) (*version.Version, error) {
	cat := c.defaultCategory()
	if last == nil || last.Major() < c.config.MajorVersion {
		return version.NewMajor(c.config.MajorVersion, cat), nil
	}
	if last.Major() > c.config.MajorVersion {
		return nil, fmt.Errorf("trying to use major version %d from config, but found existing previous version: %s",
			c.config.MajorVersion, last.RenderCategorized())
	}
	return last.IncrementMinor(cat), nil
}

func (c *NewVersionCalculator) onPatch(last *version.Version) (*version.Version, error) {
	if last == nil {
		return nil, fmt.Errorf("on a patch branch, previous version not acquired from tags")
	}
	return last.IncrementPatch(c.defaultCategory()), nil
}

func (c *NewVersionCalculator) onDev(b *branch.Branch, last *version.Version) (*version.Version, error) {
	if last == nil {
		base := version.NewMajor(c.config.MajorVersion, c.defaultCategory())
		return version.NewBuild(base, b.BuildTicket), nil
	}
	if len(last.Build()) == 0 {
		return version.NewBuild(last, b.BuildTicket), nil
	}
	return last.IncrementBuild(), nil
}
