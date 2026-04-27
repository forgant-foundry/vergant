# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Is

Vergant is a Go CLI tool for branch-driven semantic versioning in CD pipelines. It uses reachable git tags as the system of record for version state — no database, no shared service, no commit message conventions. Branch name alone determines the increment type; the full version history is derived from tags reachable via `git tag --list --sort=-taggerdate --merged`.

Vergant is managed by forglet (it carries `.forglet.yml`) and versions forglet (forglet is a live release target of vergant's branching strategy). See the **Evolution** section below.

## Commands

```bash
# Build
go build -o vergant ./cmd/vergant

# Test all packages
go test ./...

# Run integration tests (real git)
go test ./internal/git/...

# Run unit tests (stub repository)
go test ./internal/strategy/...
go test ./internal/tool/...

# Inspect temp repos during test development
FORGLET_KEEP_TEMP=1 go test -v ./internal/git/...
```

### CLI Subcommands

```
vergant last-version    # most recent release or candidate reachable from HEAD
vergant last-release    # most recent release (r-prefix) reachable from HEAD
vergant new             # calculate and tag the next version; --dry-run to skip tag
vergant promote <ver>   # promote a candidate to release; --dry-run to skip tag
vergant list            # list all reachable tags
```

Global flags: `--config .vergant.yml`, `--no-fetch`, `--path <sequence>`

## Architecture

### Reachable Tags — the Core Mechanism

All tag queries are built on:

```
git tag --list --sort=-taggerdate --merged
```

`--merged` restricts results to tags on commits that are ancestors of HEAD. This is the mechanism that makes branch-aware versioning correct without coordination:

- A dev branch sees only tags that existed on main at branch-cut time, plus its own tags. Tags added to main after branching are invisible.
- Two parallel dev branches are opaque to each other regardless of creation order.
- A patch branch cannot see releases added to main after it was cut.
- Once a branch is merged, its tags become reachable from the merge target.

"What is the last version?" is always answered relative to the current branch's ancestry, not the repository as a whole.

### Version Lifecycle

Versions carry a single-letter category prefix:

| Prefix | Meaning | Example |
|--------|---------|---------|
| `r` | Release — shipped, immutable | `r1.4.0` |
| `c` | Candidate — proposed release | `c1.4.0` |
| `d` | Development — in-progress build | `d1.5.0-acme.123.2` |

When a path is configured (see [Multi-Path Versioning](#multi-path-versioning)), these tags are stored in git with a path prefix — e.g. `sockets/r1.4.0` — but vergant always prints and accepts the short form without the prefix.

Promote changes only the prefix (`c` → `r`) on the same commit — it does not recalculate the version number. `CurrentTags` (HEAD-only, not the full reachable set) enforces that promotion is tied to the exact commit being shipped.

### Branch-Driven Strategy

Branch name determines increment type; no commit message convention is required.

| Branch type | Matched by | Increment |
|-------------|-----------|-----------|
| Default (`main`) | `Config.DefaultBranch` | minor (or major if behind `majorVersion`) |
| Patch (`support/*`) | `Config.SupportBranchRegEx` | patch release |
| Dev (`dev/*`) | `Config.DevBranchRegEx` | pre-release targeting next minor (or major) |
| PatchDev (`patch/*`) | `Config.PatchBranchRegEx` | pre-release targeting next patch |

The branch prefix encodes the increment intent — no flag required. Dev versions use semver pre-release identifiers (`-`) so they are ordered by semver-aware tools. The base version is the **target** next release. The ticket identifier is extracted from the first capture group of the branch regex and coerced: `dev/ACME-123` → `acme.123` → `d1.5.0-acme.123.0`.

Unrecognised branch names are an error — a CD pipeline should fail loudly rather than silently produce a wrong version.

### Configuration (`.vergant.yml`)

```yaml
majorVersion: 0
defaultBranch: main
patchBranchRegEx: ^support\/.*
devBranchRegEx: ^dev\/(.+)$
patchDevBranchRegEx: ^patch\/(.+)$
mode: release
path: ""        # omit or leave empty for single-sequence repos
```

Missing file or empty path returns defaults. `Load("")` is valid.

### Multi-Path Versioning

A single git log can carry independent version sequences for multiple libraries by setting a `path` on the `Git` client. The path becomes the first segment of every git tag name:

```
sockets/r1.4.0    http/r1.2.0    ftp/r3.2.2
```

All three sequences share the same `--merged` reachability guarantee — each library's branch sees only its own ancestors' tags.

**Configuring a path** (two equivalent ways, CLI overrides config):

```yaml
# .vergant.yml for the sockets library
path: sockets
```

```bash
vergant new --path sockets --dry-run
```

**How it works at the git layer.** `git.Git` holds an optional `path string`. When set:

- `FindTag` / `ListTags` list all merged tags and filter in Go to those starting with `path + "/"`. The prefix is stripped before the regex pattern is applied and before the tag name is returned, so all callers above the git layer see short names like `r1.4.0`.
- `CurrentTags` (used by `promote`) similarly filters and strips.
- `TagVersion` prepends `path + "/"` to the git tag name before creating it.

The `Repository` interface, strategy, tool, and branch layers are entirely unaware of paths. Path is an infrastructure concern fully encapsulated in `git.Git`.

**Integration test coverage** for path scoping lives in `internal/git/git_test.go`:

- `TestPath_VersionsScopedToPath`
- `TestPath_LastVersionForDevelopmentScoped`
- `TestPath_CurrentTagsFilteredToPath`
- `TestPath_ListTagsScopedToPath`
- `TestPath_ReachabilityRespected`

### Separation of Calculation and Tagging

`VersioningTool.NewVersion(increment string)` resolves and calculates the next version but does not create the tag. `VersioningTool.Tag` applies it. `-dry-run` calls `NewVersion`, prints the result, and skips `Tag`. The same split applies to `Promote`. No logic is duplicated.

### `git.Repository` Interface

The strategy and tool layers accept the `git.Repository` interface, not the concrete `git.Git` type. This keeps them independent of git implementation and fully exercisable in unit tests via `testutil.StubRepository`.

```go
type Repository interface {
    FetchTags() error
    CurrentBranch() (string, error)
    LastVersion() (string, error)
    LastRelease() (string, error)
    LastVersionForDevelopment(buildTicket string) (string, error)
    CurrentTags() ([]string, error)
    ListTags() (string, error)
    TagVersion(v *version.Version) error
}
```

### Module Structure

Single `go.mod` at the root (`github.com/forgant-foundry/vergant`):

- `cmd/vergant/` — Cobra CLI; `main.go` wires subcommands; `doc.go` covers CD rationale
- `internal/version/` — `Version` type, category prefix, parse/render, increment methods
- `internal/config/` — JSON config loading with defaults
- `internal/branch/` — branch categorisation, ticket coercion, explicit failure on unrecognised branch
- `internal/git/` — `Repository` interface and `Git` implementation using `os/exec`
- `internal/strategy/` — `AcquireLastVersion` and `NewVersionCalculator`
- `internal/tool/` — `VersioningTool` facade assembling the layers
- `internal/testutil/` — `StubRepository` (unit test seam) and `RepoHelper` (integration test helper with `GIT_COMMITTER_DATE` injection)

## Testing Conventions

Two distinct test helpers serve different purposes:

**`StubRepository`** — configurable `git.Repository` implementation. Set `Val` fields for the common case; set `Fn` fields for dynamic or stateful behaviour. `Tagged` accumulates every version passed to `TagVersion`. Use in unit tests for the strategy and tool layers where git behaviour is irrelevant.

**`RepoHelper`** — creates a real temporary git repo via `t.TempDir()` (auto-cleanup). Tag timestamps are controlled via `GIT_COMMITTER_DATE` set to a fixed base time incremented by one second per operation. This guarantees deterministic `--sort=-taggerdate` ordering without sleeping between tag creations. Use in tests that must exercise real git behaviour: reachability boundaries, branch isolation, and tag ordering.

Integration test coverage for `--merged` reachability lives in `internal/git/git_test.go`:

- `TestReachability_AncestorTagsVisible`
- `TestReachability_DevBranchIsolatedFromPostBranchMainTags`
- `TestReachability_ParallelDevBranchesIsolated`
- `TestReachability_PatchBranchIsolatedFromNewerMainTags`
- `TestReachability_PatchTagNotVisibleFromMain`

Multi-path scoping tests (also in `internal/git/git_test.go`): see [Multi-Path Versioning](#multi-path-versioning).

Annotated tags are required (`git tag -a`). Lightweight tags do not carry a `taggerdate` and will sort incorrectly.

## Evolution

### Mutual Client Relationship with forglet

Vergant and forglet are mutual clients of each other. This relationship is not incidental — it is the primary integration test for both projects.

**Forglet is a client of vergant.** Forglet uses vergant for its own release versioning. Forglet's branch-driven release lifecycle — dev builds on feature branches, candidates on integration, releases on main — is managed by vergant. Every forglet release is tagged by vergant. When vergant's branching strategy or tagging behaviour changes, forglet's release pipeline is the live target.

**Vergant is a client of forglet.** Vergant carries `.forglet.yml` and its scaffold is synthesized by forglet's Go domain. When the Go synthesizer or any plugin changes, re-synthesizing vergant is the live integration test. Vergant's `.forglet.yml` is the sharpest feedback on whether forglet's Go support serves real projects well.

### Implications

- Changes to vergant's core versioning logic should be validated by running vergant against the forglet repo before shipping.
- Changes to the forglet Go synthesizer should be validated by running `forglet synth` in the vergant repo before shipping.
- Features each project needs from the other are the highest-signal driver of evolution: if versioning forglet exposes a gap in vergant's strategy logic, that gap is real. If managing vergant exposes a gap in forglet's Go support, that gap is real.
- Future work: a forglet plugin that wires vergant versioning into managed projects — so that `forglet new go myproject` bootstraps both the project scaffold and its release pipeline in a single operation.

### First Release

Vergant's repository is currently private. Bootstrap steps before the first public release:

1. Manually apply the first tag (`r0.1.0` or `r1.0.0`) to establish the version history anchor.
2. Wire goreleaser (or equivalent) to consume vergant's own tagging for release artifact builds.
3. At that point, vergant versions itself.
