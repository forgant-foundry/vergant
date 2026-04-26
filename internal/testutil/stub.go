package testutil

import (
	"github.com/forgant-foundry/vergant/internal/git"
	"github.com/forgant-foundry/vergant/internal/version"
)

// compile-time assertion
var _ git.Repository = (*StubRepository)(nil)

// StubRepository is a configurable git.Repository for unit tests.
// Each method delegates to its Fn field if set; otherwise returns the corresponding Val field.
type StubRepository struct {
	CurrentBranchVal            string
	LastVersionVal              string
	LastReleaseVal              string
	LastVersionForDevVal        string
	CurrentTagsVal              []string
	ListTagsVal                 string
	FetchTagsErr                error
	Tagged                      []*version.Version

	CurrentBranchFn            func() (string, error)
	LastVersionFn              func() (string, error)
	LastReleaseFn              func() (string, error)
	LastVersionForDevFn        func(string) (string, error)
	CurrentTagsFn              func() ([]string, error)
}

func (s *StubRepository) FetchTags() error { return s.FetchTagsErr }

func (s *StubRepository) CurrentBranch() (string, error) {
	if s.CurrentBranchFn != nil {
		return s.CurrentBranchFn()
	}
	return s.CurrentBranchVal, nil
}

func (s *StubRepository) LastVersion() (string, error) {
	if s.LastVersionFn != nil {
		return s.LastVersionFn()
	}
	return s.LastVersionVal, nil
}

func (s *StubRepository) LastRelease() (string, error) {
	if s.LastReleaseFn != nil {
		return s.LastReleaseFn()
	}
	return s.LastReleaseVal, nil
}

func (s *StubRepository) LastVersionForDevelopment(buildTicket string) (string, error) {
	if s.LastVersionForDevFn != nil {
		return s.LastVersionForDevFn(buildTicket)
	}
	return s.LastVersionForDevVal, nil
}

func (s *StubRepository) CurrentTags() ([]string, error) {
	if s.CurrentTagsFn != nil {
		return s.CurrentTagsFn()
	}
	return s.CurrentTagsVal, nil
}

func (s *StubRepository) ListTags() (string, error) {
	return s.ListTagsVal, nil
}

func (s *StubRepository) TagVersion(v *version.Version) error {
	s.Tagged = append(s.Tagged, v)
	return nil
}
