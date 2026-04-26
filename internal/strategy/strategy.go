package strategy

import (
	"fmt"

	"github.com/forgant-foundry/vergant/internal/branch"
	"github.com/forgant-foundry/vergant/internal/config"
	"github.com/forgant-foundry/vergant/internal/git"
	"github.com/forgant-foundry/vergant/internal/version"
)

// Acquire retrieves the relevant last version for a branch.
type Acquire func() (*version.Version, error)

// Calculate computes the next version given the last known version for the branch.
type Calculate func(last *version.Version) (*version.Version, error)

// AcquireLastVersion fetches the last known version from git tags.
type AcquireLastVersion struct {
	git git.Repository
}

func NewAcquireLastVersion(g git.Repository) *AcquireLastVersion {
	return &AcquireLastVersion{git: g}
}

// Resolve returns an Acquire function appropriate for b.
func (a *AcquireLastVersion) Resolve(b *branch.Branch) Acquire {
	switch b.Category {
	case branch.Dev, branch.Patch:
		return a.acquireLastVersionForDevelopment(b.BuildTicket)
	default:
		return a.acquireLastVersion
	}
}

func (a *AcquireLastVersion) acquireLastVersion() (*version.Version, error) {
	tag, err := a.git.LastVersion()
	if err != nil {
		return nil, err
	}
	return version.Parse(tag)
}

func (a *AcquireLastVersion) acquireLastVersionForDevelopment(buildTicket string) Acquire {
	return func() (*version.Version, error) {
		tag, err := a.git.LastVersionForDevelopment(buildTicket)
		if err != nil {
			return nil, err
		}
		return version.Parse(tag)
	}
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

// Resolve returns a Calculate function appropriate for b.
// lastRelease is captured in the closure for Dev and Patch branches.
func (c *NewVersionCalculator) Resolve(b *branch.Branch, lastRelease *version.Version) Calculate {
	switch b.Category {
	case branch.Default:
		return c.onDefault
	case branch.Support:
		return c.onSupport
	case branch.Dev:
		return c.onDev(b, lastRelease)
	case branch.Patch:
		return c.onPatch(b, lastRelease)
	default:
		return func(_ *version.Version) (*version.Version, error) {
			return nil, fmt.Errorf("unsupported branch category")
		}
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

func (c *NewVersionCalculator) onSupport(last *version.Version) (*version.Version, error) {
	if last == nil {
		return nil, fmt.Errorf("on a support branch, previous version not acquired from tags")
	}
	return last.IncrementPatch(c.defaultCategory()), nil
}

func (c *NewVersionCalculator) onDev(b *branch.Branch, lastRelease *version.Version) Calculate {
	return func(lastDev *version.Version) (*version.Version, error) {
		target, err := c.minorTarget(lastRelease)
		if err != nil {
			return nil, err
		}
		if lastDev != nil && sameBase(lastDev, target) {
			return lastDev.IncrementPreRelease(), nil
		}
		return version.NewPreRelease(target, b.BuildTicket), nil
	}
}

func (c *NewVersionCalculator) onPatch(b *branch.Branch, lastRelease *version.Version) Calculate {
	return func(lastDev *version.Version) (*version.Version, error) {
		target, err := c.patchTarget(lastRelease)
		if err != nil {
			return nil, err
		}
		if lastDev != nil && sameBase(lastDev, target) {
			return lastDev.IncrementPreRelease(), nil
		}
		return version.NewPreRelease(target, b.BuildTicket), nil
	}
}

func (c *NewVersionCalculator) minorTarget(lastRelease *version.Version) (*version.Version, error) {
	if lastRelease == nil || lastRelease.Major() < c.config.MajorVersion {
		return version.NewMajor(c.config.MajorVersion, version.Release), nil
	}
	if lastRelease.Major() > c.config.MajorVersion {
		return nil, fmt.Errorf("config majorVersion %d is below existing release %s",
			c.config.MajorVersion, lastRelease.RenderCategorized())
	}
	return lastRelease.IncrementMinor(version.Release), nil
}

func (c *NewVersionCalculator) patchTarget(lastRelease *version.Version) (*version.Version, error) {
	if lastRelease == nil {
		return nil, fmt.Errorf("on a patch dev branch, no release found to patch")
	}
	return lastRelease.IncrementPatch(version.Release), nil
}

func sameBase(v, target *version.Version) bool {
	return v.Major() == target.Major() && v.Minor() == target.Minor() && v.Patch() == target.Patch()
}
