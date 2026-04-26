// Package version defines the [Version] type used throughout vergant.
//
// # Category prefix
//
// Every version carries a single-character category prefix encoding its
// lifecycle stage. Standard semver has no equivalent concept.
//
//   - r (release)     — stable and shipped
//   - c (candidate)   — proposed release under validation
//   - d (development) — in-flight work on a feature or fix branch
//
// The prefix is part of the git tag string only. Use [Version.RenderUncategorized]
// when a bare semver string is required (container image tags, deployment manifests).
//
// # Development versions and build metadata
//
// Dev versions encode a ticket identifier and a monotonic build counter in the
// semver build metadata field:
//
//	d1.4.0+acme.123.2
//
// The base version (1.4.0) is the last release or candidate reachable from the
// branch. The ticket identifier (acme.123) is derived from the branch name by
// the branch package. The build counter (2) increments with each successive tag
// on the same branch.
//
// Build metadata is ignored by semver precedence rules, so dev versions do not
// participate in release ordering. They are branch-local signals, not entries in
// the release timeline.
//
// # Parsing and construction
//
// [Parse] accepts a categorized string ("r1.2.3", "d1.4.0+acme.123.2") and
// returns a [Version]. An empty string returns nil, nil. An unrecognised prefix
// returns an error.
//
// [CoerceToCandidate] loosely accepts "r1.2.3", "c1.2.3", or "1.2.3" and
// returns a candidate version. Used when accepting user-supplied version strings
// for the promote command, where the caller may omit or vary the prefix.
//
// [NewMajor] and [NewBuild] are the two constructors used by the strategy layer
// when no prior version exists to increment from.
package version
