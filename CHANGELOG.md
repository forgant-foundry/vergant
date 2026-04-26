# Changelog

All notable changes to this project will be documented here.
Entries are moved from Unreleased to a versioned section at release time.

---

## Background

Continuous delivery pipelines require versioning to be automatic, unambiguous,
and correct across parallel branches simultaneously. The dominant class of tools
— semantic-release, release-please, and similar — delegates version intent to
commit message conventions. This approach has three failure modes in CD
environments: the convention must be enforced uniformly across every contributor
and merge strategy; squash merges silently discard the signal; and a single
malformed message produces a wrong version with no diagnostic. These are human
coordination failures that manifest as pipeline failures.

In a disciplined CD practice, the branch is already the coordination signal. The
branch name expresses intent — default branch for regular releases, support
branches for patches, feature branches for in-progress work. vergant reads that
signal directly from the git branch rather than requiring it to be restated in
commit messages.

The secondary problem is parallel branch isolation. A team running multiple
feature branches simultaneously cannot use tools that reason about "the latest
tag in the repository" — concurrent branches will collide or produce versions
contaminated by each other's history. vergant scopes every version query to the
ancestry of the current commit via `git tag --list --merged`, giving each branch
a correct, isolated view of version history without locks or cross-branch
communication.

---

## [Unreleased]

### Added

- Initial Go implementation of the versioning tool.
- Reachable-tags query (`git tag --list --sort=-taggerdate --merged`) as the
  foundational mechanism for branch-aware version calculation. Tags are scoped
  to the ancestry of the current commit, giving each branch an isolated view of
  version history without coordination.
- Version lifecycle prefix scheme: `r` (release), `c` (candidate), `d`
  (development). The same semver number can exist at multiple stages; promotion
  changes only the prefix.
- Development versions encode ticket identifier and monotonic build counter in
  semver build metadata (e.g. `d1.4.0+acme.123.2`).
- Branch-driven strategy: branch name determines increment type and lifecycle
  stage — no commit message convention required.
- Subcommand CLI (`last-version`, `last-release`, `new`, `promote`, `list`)
  with `-config`, `-no-fetch`, and `-dry-run` flags implemented with stdlib
  `flag` — no external dependencies.
- `.vergant.yml` for project-level configuration (major version, branch
  regex patterns, workflow mode). Flat key-value YAML parsed without a library.
- `git.Repository` interface decoupling business logic from the git
  implementation, enabling fast unit tests with a stub.
- `testutil.RepoHelper` using `GIT_COMMITTER_DATE` injection for deterministic
  tag ordering in integration tests — no sleeping required.
- Reachability integration tests covering all isolation boundaries: ancestor
  visibility, dev branch isolation from post-branch main tags, parallel dev
  branch isolation, and patch branch isolation from newer main releases.
- GitHub Actions workflows: `version.yml` applies a version tag on every branch
  push; `release.yml` builds multi-platform binaries and publishes a GitHub
  Release when an `r*` tag is pushed. The two workflows share a clean handoff —
  the version tag is both vergant's version record and the release trigger.
- `README.md` covering motivation, reachable tags mechanism, version lifecycle,
  branch-driven strategy, configuration reference, CLI usage, example scenarios,
  branching guidance, GitHub Actions integration, and tag protection.
- `.gitignore` synthesized by forglet's GitPlugin via `git: true` in
  `.forglet.yml`. vergant is now a managed forglet project.

### Changed

- CLI redesigned from a flat flag interface (`--lastVersion`, `--new`) to
  subcommands with per-command flags and built-in help.
- Cobra dependency removed; CLI rewritten with stdlib `flag` package, reducing
  binary size and eliminating all external dependencies.
- Configuration file format changed from JSON (`vergant.config.json`) to flat
  key-value YAML (`.vergant.yml`), parsed without a library.
- Dev version format changed from build metadata to semver pre-release:
  `d1.4.0+acme.123.0` → `d1.5.0-acme.123.0`. Pre-release identifiers are
  ordered by semver precedence rules, making dev versions sortable by
  ecosystem tooling (npm, cargo, etc.). The base version now reflects the
  target next release rather than the last shipped release.
- Branch prefix encodes increment intent: `dev/*` targets the next minor (or
  major) release; `patch/*` targets the next patch release. No `-increment`
  flag required — the branch name is the sole signal.
- New `patchDevBranchRegEx` config field (default `^patch\/(.+)$`) identifies
  `patch/*` branches that produce patch pre-releases.
- Default `devBranchRegEx` changed from a JIRA-capture pattern to `^dev\/(.+)$`
  to match the `dev/*` naming convention.
- `AcquireLastVersion` strategy tests consolidated into a single table-driven
  test; added patch branch case and assertion that `LastVersionForDevelopment`
  is called with the correct ticket identifier.

### Removed

- GitLab CI environment variable detection (`CI_API_V4_URL`, `CI_PROJECT_ID`,
  `CI_COMMIT_SHORT_SHA`, `CI_COMMIT_BRANCH`, `APOVER_TAGGER_TOKEN`). All
  behaviour is now determined by explicit command line arguments.
- GitLab API tag creation path. Only native git is used.
