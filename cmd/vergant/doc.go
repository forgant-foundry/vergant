/*
Package main implements vergant, a semantic versioning tool driven by reachable
git tags.

# Continuous Delivery Context

Continuous delivery pipelines require versioning to be automatic, unambiguous,
and correct across parallel branches. Most existing tools fail at least one of
these requirements in practice.

Commit-message-convention tools (semantic-release, release-please, and their
derivatives) delegate version intent to individual commit messages. This creates
three problems in CD environments: the convention must be enforced uniformly
across every contributor and every merge strategy; squash merges silently discard
the signal; and a single malformed message in a long branch history produces a
wrong version with no diagnostic. These are human coordination failures that
appear as pipeline failures.

In any disciplined CD practice, the branch is already the coordination signal.
The name of the branch — default, patch, or feature — expresses the intent of
the work. vergant reads that signal directly from git rather than requiring it to
be restated in commit messages.

The practical consequences for a CD pipeline are:

  - No commit convention to enforce, train, or lint.
  - Versioning runs correctly on squash merges, rebases, and merge commits alike.
  - Feature branches running in parallel each calculate their own version in
    isolation without locks or cross-branch awareness.
  - Every build artifact carries a version that encodes its base release, its
    originating ticket, and its build sequence (e.g. d1.4.0+acme.123.2),
    providing provenance without an external lookup.
  - Hotfix branches on released versions calculate patch increments in isolation
    from main-line work, supporting concurrent maintenance of multiple release
    lines.
  - The tool is a single binary with no runtime dependencies beyond git, making
    it straightforward to install and pin in any pipeline environment.

# Reachable Tags

The central mechanism of vergant is git's ancestry-scoped tag query:

	git tag --list --sort=-taggerdate --merged

The --merged flag restricts results to tags that are reachable from the current
HEAD — that is, tags on commits that are direct ancestors of the current commit.
This single property is what makes branch-aware versioning correct without any
external state or cross-branch communication.

The consequences are precise:

  - A dev branch sees only tags that existed on main at the point it was cut,
    plus any tags created on the dev branch itself. Tags added to main after
    branching are invisible.
  - Two parallel dev branches are opaque to each other. Neither can observe the
    other's tags regardless of when they were created.
  - A patch branch (e.g. support/1.1.x) sees only the ancestry it shares with
    main up to its branch point, plus its own tags. Tags on main added after the
    patch branch was cut are invisible.
  - Once a branch is merged, its tags become reachable from the merge target and
    participate in future version calculations there.

This means "what is the last version?" is always answered relative to the current
branch's history, not the repository as a whole. Version calculation requires no
knowledge of other branches, no locks, and no coordination.

# VCS Tags as the System of Record

Rather than maintaining a changelog, manifest file, or external database, vergant
treats the repository's annotated tag history as the sole source of truth for
versions. A version exists if and only if a matching tag exists on an ancestor
commit. This has several consequences:

  - Version history is co-located with source history. Moving or cloning the
    repository transfers the complete version record automatically.
  - Creating a version and recording it are a single atomic git operation — there
    is no window where a version is "applied" but not yet persisted.
  - Deleting or force-pushing a tag corrupts the version record without any
    warning from this tool. Tag protection rules at the remote level are the
    appropriate guardrail.

# Version Lifecycle: r, c, and d Prefixes

Standard semver carries no information about where in a release lifecycle a version
sits. vergant addresses this by prefixing every tag with a single character that
encodes its stage:

  - r (release):     stable, shipped. Produced on the default or a patch branch.
  - c (candidate):   proposed release, under validation. Produced on the default
                     or patch branch in candidate mode.
  - d (development): in-flight work on a feature or fix branch. Carries build
                     metadata encoding the ticket identifier and an incrementing
                     build counter, e.g. d1.4.0+acme.123.2.

This means the same semantic version number can legally appear at more than one
lifecycle stage. r1.4.0 and c1.4.0 coexist without ambiguity; c1.4.0 promotes
to r1.4.0 by changing only the prefix and creating a new tag. No renaming,
re-tagging, or changelog entry is required.

The prefix is part of the git tag only. Consumers that need a bare semver string
(container image tags, package manifests) should use RenderUncategorized.

# Development Versions and Build Metadata

Dev versions embed the ticket identifier and a monotonic build counter in the
semver build metadata field, e.g. d1.4.0+acme.123.0. The base version (1.4.0)
matches the last release or candidate visible from the branch, so a dev version
always reads as "this ticket's work on top of version X". Each successive tag on
the same branch increments only the trailing counter.

Because build metadata is ignored by semver precedence rules, dev versions do not
participate in the release ordering. They are branch-local signals, not entries in
the release timeline.

# Branch-Driven Strategy

The versioning strategy — which component to increment, and which lifecycle stage
to target — is determined entirely by the name of the current branch matched
against patterns in .vergant.yml. This means:

  - No commit message convention (Conventional Commits, etc.) is required.
  - Increment type is a property of the branch, not of individual commits.
  - Default branch → minor increment.
  - Patch branch (e.g. support/1.4.x) → patch increment.
  - Dev branch (e.g. feature/ACME-123) → build metadata version.

# Concerns

Tag timestamp ordering: vergant sorts candidates by taggerdate, the timestamp
embedded in the annotated tag object. This ordering can be wrong if tags are
created out of sequence (e.g. a tag on an older commit is created later in wall
time). CI pipelines running in strict sequence are not affected; parallel pipelines
tagging the same branch could be.

Concurrent tagging: two processes tagging the same branch at the same moment can
calculate the same next version independently and both succeed, producing a
duplicate. A distributed lock or serialized pipeline stage is the correct
mitigation; vergant does not attempt to solve this.

Shallow clones: the --merged traversal requires a clone depth that reaches all
relevant tags. Shallow clones in CI must explicitly fetch tags with sufficient
depth.
*/
package main
