// Package branch categorizes git branch names into one of three versioning roles.
//
// # Categories
//
// [Default] — the primary integration branch (typically "main"). Drives minor
// or major version increments on the default line.
//
// [Support] — a maintenance branch for a previously released version (e.g.
// "support/1.4.x"). Drives patch increments in isolation from the default line.
// Matched by [Config.SupportBranchRegEx].
//
// [Dev] — an in-progress feature or fix branch (e.g. "feature/ACME-123").
// Drives development versions carrying build metadata. Matched by
// [Config.DevBranchRegEx].
//
// # Ticket extraction
//
// For Dev branches, [ForName] extracts a ticket identifier from the first
// capture group of devBranchRegEx. The raw match is coerced to lowercase with
// hyphens and underscores replaced by dots, producing a build-metadata-safe
// component:
//
//	"feature/ACME-123_fix" → capture "ACME-123" → coerce → "acme.123"
//
// This identifier is embedded in dev version tags as the pre-release identifier
// (e.g. d1.5.0-acme.123.0).
//
// # Failure behaviour
//
// [ForName] returns an error for branch names that match none of the four
// patterns. This is intentional — an unrecognised branch in a CD pipeline
// fails loudly rather than silently producing a wrong version.
package branch
