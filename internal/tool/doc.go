// Package tool provides [VersioningTool], the facade that assembles the config,
// branch, git, and strategy packages into the operations exposed by the CLI.
//
// # Separation of calculation and tagging
//
// [VersioningTool.NewVersion] resolves the current branch, acquires the last
// known version, and calculates the next one — but does not create the tag.
// [VersioningTool.Tag] applies it. The CLI's --dry-run flag uses this split:
// it calls NewVersion, prints the result, and skips Tag. No logic is duplicated.
//
// The same separation applies to [VersioningTool.Promote]: it validates and
// returns the release version without creating the tag.
//
// # Promote validation
//
// [VersioningTool.Promote] enforces two preconditions before returning a release
// version:
//
//  1. The corresponding release tag must not already exist on the current commit.
//  2. The candidate tag must exist on the current commit.
//
// These checks use [Repository.CurrentTags], which queries only tags pointing
// at HEAD — not the full reachable set. This ensures promotion is tied to the
// exact commit being shipped, not just any commit that was once tagged as that
// candidate.
package tool
