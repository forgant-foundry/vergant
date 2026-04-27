package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Category identifies the classification of a version tag.
type Category string

const (
	Release   Category = "r"
	Candidate Category = "c"
	Dev       Category = "d"
)

// Prefixes maps each category to its tag prefix string.
type Prefixes struct {
	Release   string
	Candidate string
	Dev       string
}

// DefaultPrefixes returns the default prefix configuration: v for release, c for candidate, d for dev.
func DefaultPrefixes() Prefixes {
	return Prefixes{Release: "v", Candidate: "c", Dev: "d"}
}

func (p Prefixes) forCategory(cat Category) string {
	switch cat {
	case Release:
		return p.Release
	case Candidate:
		return p.Candidate
	default:
		return p.Dev
	}
}

// Version is an immutable semantic version with a category prefix.
type Version struct {
	major          int
	minor          int
	patch          int
	pre            []string // e.g. ["acme", "123", "0"] for d1.2.3-acme.123.0
	Category       Category
	renderedPrefix string // configured prefix to use in RenderCategorized; empty falls back to Category
}

var semverRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-(.+))?$`)

// Parse parses a categorized version string using the default prefix configuration (v/c/d).
// Returns nil, nil for an empty string.
// For custom prefixes, use ParseWithPrefixes.
func Parse(s string) (*Version, error) {
	return ParseWithPrefixes(s, DefaultPrefixes())
}

// ParseWithPrefixes parses a categorized version string using the given prefix configuration.
// Returns nil, nil for an empty string.
func ParseWithPrefixes(s string, p Prefixes) (*Version, error) {
	if s == "" {
		return nil, nil
	}
	for _, pair := range []struct {
		cat    Category
		prefix string
	}{
		{Release, p.Release},
		{Candidate, p.Candidate},
		{Dev, p.Dev},
	} {
		if strings.HasPrefix(s, pair.prefix) {
			v, err := parseSemver(s[len(pair.prefix):], pair.cat)
			if err != nil {
				return nil, err
			}
			v.renderedPrefix = pair.prefix
			return v, nil
		}
	}
	return nil, fmt.Errorf("found version without expected category prefix of %q, %q or %q for release, candidate or dev: %s",
		p.Release, p.Candidate, p.Dev, s)
}

func parseSemver(s string, cat Category) (*Version, error) {
	m := semverRe.FindStringSubmatch(s)
	if m == nil {
		return nil, fmt.Errorf("unable to parse semantic version from: %s", s)
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	var pre []string
	if m[4] != "" {
		pre = strings.Split(m[4], ".")
	}
	return &Version{major: major, minor: minor, patch: patch, pre: pre, Category: cat}, nil
}

// CoerceToCandidate accepts a release-prefixed, candidate-prefixed, or bare semver string
// and returns a candidate version using the default prefix configuration (v/c/d).
// For custom prefixes, use CoerceToCandidateWithPrefixes.
func CoerceToCandidate(s string) (*Version, error) {
	return CoerceToCandidateWithPrefixes(s, DefaultPrefixes())
}

// CoerceToCandidateWithPrefixes accepts a release-prefixed, candidate-prefixed, or bare semver
// string and returns a candidate version using the given prefix configuration.
func CoerceToCandidateWithPrefixes(s string, p Prefixes) (*Version, error) {
	for _, prefix := range []string{p.Release, p.Candidate} {
		if strings.HasPrefix(s, prefix) {
			v, err := parseSemver(s[len(prefix):], Candidate)
			if err != nil {
				return nil, err
			}
			v.renderedPrefix = p.Candidate
			return v, nil
		}
	}
	v, err := parseSemver(s, Candidate)
	if err != nil {
		return nil, err
	}
	v.renderedPrefix = p.Candidate
	return v, nil
}

// NewMajor returns a new major.0.0 version with the given category.
func NewMajor(major int, cat Category) *Version {
	return &Version{major: major, Category: cat}
}

// NewPreRelease returns a dev version targeting target with the given ticket identifier and build counter 0.
func NewPreRelease(target *Version, ticket string) *Version {
	parts := append(strings.Split(ticket, "."), "0")
	return &Version{major: target.major, minor: target.minor, patch: target.patch, pre: parts, Category: Dev}
}

func (v *Version) Major() int      { return v.major }
func (v *Version) Minor() int      { return v.minor }
func (v *Version) Patch() int      { return v.patch }
func (v *Version) PreRelease() []string { return v.pre }

// PreReleaseTicket returns the pre-release identifier without the trailing build counter.
func (v *Version) PreReleaseTicket() []string {
	if len(v.pre) == 0 {
		return nil
	}
	return v.pre[:len(v.pre)-1]
}

// EqualBase reports whether v and other share the same major.minor.patch, ignoring pre-release and category.
func (v *Version) EqualBase(other *Version) bool {
	return v.major == other.major && v.minor == other.minor && v.patch == other.patch
}

// WithCategory returns the version with a different category, numbers unchanged.
// Clears the rendered prefix; call WithRenderedPrefix to re-apply one.
func (v *Version) WithCategory(cat Category) *Version {
	n := *v
	n.Category = cat
	n.renderedPrefix = ""
	return &n
}

// WithRenderedPrefix returns the version with its rendered prefix set from p,
// so that RenderCategorized uses the configured prefix string.
func (v *Version) WithRenderedPrefix(p Prefixes) *Version {
	n := *v
	n.renderedPrefix = p.forCategory(v.Category)
	return &n
}

// IncrementMajor returns (major+1).0.0 with the given category.
// The rendered prefix is carried forward when the category is unchanged.
func (v *Version) IncrementMajor(cat Category) *Version {
	rp := v.renderedPrefixFor(cat)
	return &Version{major: v.major + 1, Category: cat, renderedPrefix: rp}
}

// IncrementMinor returns major.(minor+1).0 with the given category.
// The rendered prefix is carried forward when the category is unchanged.
func (v *Version) IncrementMinor(cat Category) *Version {
	rp := v.renderedPrefixFor(cat)
	return &Version{major: v.major, minor: v.minor + 1, Category: cat, renderedPrefix: rp}
}

// IncrementPatch returns major.minor.(patch+1) with the given category.
// The rendered prefix is carried forward when the category is unchanged.
func (v *Version) IncrementPatch(cat Category) *Version {
	rp := v.renderedPrefixFor(cat)
	return &Version{major: v.major, minor: v.minor, patch: v.patch + 1, Category: cat, renderedPrefix: rp}
}

// IncrementPreRelease returns the dev version with the trailing build counter incremented.
// The rendered prefix is carried forward.
func (v *Version) IncrementPreRelease() *Version {
	if len(v.pre) == 0 {
		return v
	}
	parts := make([]string, len(v.pre))
	copy(parts, v.pre)
	n, _ := strconv.Atoi(parts[len(parts)-1])
	parts[len(parts)-1] = strconv.Itoa(n + 1)
	return &Version{major: v.major, minor: v.minor, patch: v.patch, pre: parts, Category: Dev, renderedPrefix: v.renderedPrefix}
}

func (v *Version) renderedPrefixFor(cat Category) string {
	if cat == v.Category {
		return v.renderedPrefix
	}
	return ""
}

// RenderCategorized returns the full version string with the prefix.
// Uses the rendered prefix set by ParseWithPrefixes or WithRenderedPrefix; falls back to the
// internal Category identifier (r/c/d) when no rendered prefix has been set.
func (v *Version) RenderCategorized() string {
	if v.renderedPrefix != "" {
		return v.renderedPrefix + v.renderSemver()
	}
	return string(v.Category) + v.renderSemver()
}

// RenderUncategorized returns just the semver part, e.g. "1.2.3" or "1.2.3-acme.123.0".
func (v *Version) RenderUncategorized() string {
	return v.renderSemver()
}

func (v *Version) renderSemver() string {
	s := fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
	if len(v.pre) > 0 {
		s += "-" + strings.Join(v.pre, ".")
	}
	return s
}

// RenderMessage returns a human-readable annotation message for the tag.
func (v *Version) RenderMessage() string {
	switch v.Category {
	case Release:
		return "release " + v.RenderUncategorized()
	case Candidate:
		return "candidate " + v.RenderUncategorized()
	default:
		return "development " + v.RenderUncategorized()
	}
}
