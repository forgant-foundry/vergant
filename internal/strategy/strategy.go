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
// For Dev and PatchDev branches this is the most recent dev tag for the ticket, or nil if none.
func (a *AcquireLastVersion) Resolve(b *branch.Branch) (*version.Version, error) {
	switch b.Category {
	case branch.Dev, branch.Patch:
		return a.lastVersionForDevelopment(b.BuildTicket)
	default:
		return a.lastVersion()
	}
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
// lastRelease is used only by Dev and PatchDev branches to determine the target version.
func (c *NewVersionCalculator) Resolve(b *branch.Branch, last *version.Version, lastRelease *version.Version) (*version.Version, error) {
	switch b.Category {
	case branch.Default:
		return c.onDefault(last)
	case branch.Support:
		return c.onPatch(last)
	case branch.Dev:
		return c.onDev(b, last, lastRelease)
	case branch.Patch:
		return c.onPatchDev(b, last, lastRelease)
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

// onDev handles dev/* branches: pre-release targeting the next minor (or major) release.
func (c *NewVersionCalculator) onDev(b *branch.Branch, lastDev *version.Version, lastRelease *version.Version) (*version.Version, error) {
	target, err := c.minorTarget(lastRelease)
	if err != nil {
		return nil, err
	}
	if lastDev != nil && sameBase(lastDev, target) {
		return lastDev.IncrementPreRelease(), nil
	}
	return version.NewPreRelease(target, b.BuildTicket), nil
}

// onPatchDev handles patch/* branches: pre-release targeting the next patch release.
func (c *NewVersionCalculator) onPatchDev(b *branch.Branch, lastDev *version.Version, lastRelease *version.Version) (*version.Version, error) {
	target, err := c.patchTarget(lastRelease)
	if err != nil {
		return nil, err
	}
	if lastDev != nil && sameBase(lastDev, target) {
		return lastDev.IncrementPreRelease(), nil
	}
	return version.NewPreRelease(target, b.BuildTicket), nil
}

// minorTarget computes the target version for a dev/* branch: next minor, or next major if config advances.
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

// patchTarget computes the target version for a patch/* branch: next patch of the last release.
func (c *NewVersionCalculator) patchTarget(lastRelease *version.Version) (*version.Version, error) {
	if lastRelease == nil {
		return nil, fmt.Errorf("on a patch dev branch, no release found to patch")
	}
	return lastRelease.IncrementPatch(version.Release), nil
}

func sameBase(v, target *version.Version) bool {
	return v.Major() == target.Major() && v.Minor() == target.Minor() && v.Patch() == target.Patch()
}
