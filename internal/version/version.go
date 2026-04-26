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

// Version is an immutable semantic version with a category prefix.
type Version struct {
	major    int
	minor    int
	patch    int
	pre      []string // e.g. ["acme", "123", "0"] for d1.2.3-acme.123.0
	Category Category
}

var semverRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-(.+))?$`)

// Parse parses a categorized version string such as "r1.2.3" or "d1.2.3-acme.123.0".
// Returns nil, nil for an empty string.
func Parse(s string) (*Version, error) {
	if s == "" {
		return nil, nil
	}
	cat := Category(s[0:1])
	switch cat {
	case Release, Candidate, Dev:
	default:
		return nil, fmt.Errorf("found version without expected category prefix of 'r', 'c' or 'd' for release, candidate or dev: %s", s)
	}
	return parseSemver(s[1:], cat)
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

// CoerceToCandidate accepts "cX.X.X", "rX.X.X", or "X.X.X" and returns a candidate version.
func CoerceToCandidate(s string) (*Version, error) {
	if len(s) > 0 {
		cat := Category(s[0:1])
		if cat == Release || cat == Candidate {
			return parseSemver(s[1:], Candidate)
		}
	}
	return parseSemver(s, Candidate)
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

// WithCategory returns the version with a different category, numbers unchanged.
func (v *Version) WithCategory(cat Category) *Version {
	n := *v
	n.Category = cat
	return &n
}

// IncrementMajor returns (major+1).0.0 with the given category.
func (v *Version) IncrementMajor(cat Category) *Version {
	return &Version{major: v.major + 1, Category: cat}
}

// IncrementMinor returns major.(minor+1).0 with the given category.
func (v *Version) IncrementMinor(cat Category) *Version {
	return &Version{major: v.major, minor: v.minor + 1, Category: cat}
}

// IncrementPatch returns major.minor.(patch+1) with the given category.
func (v *Version) IncrementPatch(cat Category) *Version {
	return &Version{major: v.major, minor: v.minor, patch: v.patch + 1, Category: cat}
}

// IncrementPreRelease returns the dev version with the trailing build counter incremented.
func (v *Version) IncrementPreRelease() *Version {
	if len(v.pre) == 0 {
		return v
	}
	parts := make([]string, len(v.pre))
	copy(parts, v.pre)
	n, _ := strconv.Atoi(parts[len(parts)-1])
	parts[len(parts)-1] = strconv.Itoa(n + 1)
	return &Version{major: v.major, minor: v.minor, patch: v.patch, pre: parts, Category: Dev}
}

// RenderCategorized returns the full version string with category prefix, e.g. "r1.2.3".
func (v *Version) RenderCategorized() string {
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
