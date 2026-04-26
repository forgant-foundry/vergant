# vergant

Branch-driven semantic versioning for continuous delivery pipelines.

## Motivation

Continuous delivery pipelines require versioning to be automatic, unambiguous, and correct across parallel branches simultaneously. The dominant class of tools — semantic-release, release-please, and similar — delegates version intent to commit message conventions. This approach has three failure modes in CD environments: the convention must be enforced uniformly across every contributor and merge strategy; squash merges silently discard the signal; and a single malformed message produces a wrong version with no diagnostic. These are human coordination failures that manifest as pipeline failures.

In a disciplined CD practice, the branch is already the coordination signal. The branch name expresses intent — the default branch for regular releases, support branches for patches, feature branches for in-progress work. vergant reads that signal directly from the git branch rather than requiring it to be restated in commit messages.

The secondary problem is parallel branch isolation. A team running multiple feature branches simultaneously cannot use tools that reason about "the latest tag in the repository" — concurrent branches will collide or produce versions contaminated by each other's history. vergant scopes every version query to the ancestry of the current commit via `git tag --list --merged`, giving each branch a correct, isolated view of version history without locks or cross-branch communication.

## How It Works

### Reachable Tags

All version queries are built on:

```
git tag --list --sort=-taggerdate --merged
```

The `--merged` flag restricts results to tags on commits that are ancestors of HEAD. This means:

- A dev branch sees only tags that existed on main at branch-cut time, plus its own tags. Tags added to main after branching are invisible.
- Two parallel dev branches are opaque to each other regardless of creation order.
- A patch branch cannot see releases added to main after it was cut.
- Once a branch is merged, its tags become reachable from the merge target.

"What is the last version?" is always answered relative to the current branch's ancestry, not the repository as a whole.

### Version Lifecycle

Versions carry a single-letter category prefix:

| Prefix | Meaning | Example |
|--------|---------|---------|
| `r` | Release — shipped | `r1.4.0` |
| `c` | Candidate — proposed release | `c1.4.0` |
| `d` | Development — in-progress build | `d1.5.0-acme.123.2` |

The same semver number can exist at multiple lifecycle stages. Promotion from candidate to release changes only the prefix — the version number itself does not change.

Development versions use semver pre-release identifiers (`-`) to embed a ticket identifier and a monotonic build counter. Unlike build metadata (`+`), pre-release identifiers are ordered by semver precedence rules, making dev versions sortable by tools that consume semver (npm, cargo, etc.). The base version in a dev tag is the **target** next release — the version this branch will produce when merged — rather than the last shipped release.

### Branch-Driven Strategy

Branch name determines increment type. No commit message convention is required.

| Branch | Matched by | Produces |
|--------|-----------|---------|
| Default (`main`) | `defaultBranch` config | Minor increment; major increment when `majorVersion` config advances |
| Support (`support/1.4.x`) | `supportBranchRegEx` | Patch increment |
| Dev (`dev/ACME-123`) | `devBranchRegEx` | Pre-release targeting next minor (or next major) |
| Patch (`patch/ACME-456`) | `patchBranchRegEx` | Pre-release targeting next patch |

The branch prefix encodes intent. `dev/*` branches target a minor (or major) release; `patch/*` branches target a patch release and are merged into a `support/*` branch. No flags are required — the branch name is the sole signal.

An unrecognised branch name is an error — a CD pipeline should fail loudly rather than silently produce a wrong version.

## Installation

```bash
go install github.com/forgant-foundry/vergant/cmd/vergant@latest
```

Or build from source:

```bash
git clone https://github.com/forgant-foundry/vergant
cd vergant
go build -o vergant ./cmd/vergant
```

## Usage

```
vergant <command> [flags]

Commands:
  last-version   Print the most recent release or candidate tag reachable from HEAD
  last-release   Print the most recent release tag reachable from HEAD
  new            Calculate and apply the next version tag
  promote        Promote a candidate tag to a release tag
  list           List all reachable tags

Global flags:
  --config string   config file (default: .vergant.yml)
  --no-fetch        skip fetching tags from remote before querying

Flags for new and promote:
  --dry-run   print the version without creating or pushing the tag
```

### Examples

```bash
# See what the next version would be without tagging
vergant new --dry-run

# Tag and push the next version
vergant new

# Promote a candidate to release (HEAD must carry the candidate tag)
vergant promote c1.4.0

# Diagnose what tags are visible from the current commit
vergant list
```

## Configuration

Create `.vergant.yml` in the project root. All fields are optional — the defaults work for most projects.

```yaml
majorVersion: 1
```

Full configuration with all fields:

```yaml
majorVersion: 1
defaultBranch: main
supportBranchRegEx: ^support\/.*
devBranchRegEx: ^dev\/(.+)$
patchBranchRegEx: ^patch\/(.+)$
mode: release
```

The file is flat key-value YAML — one field per line, no nesting. Lines beginning with `#` are comments.

### Fields

**`majorVersion`** — The current major version. When you are ready for a breaking change, increment this value and run `vergant new` on the default branch. That single config change is the only manual step required for a major version bump.

**`defaultBranch`** — The trunk branch name. Default: `main`.

**`supportBranchRegEx`** — Regular expression identifying support branches. Default: `^support\/.*`. A common alternative is `^release\/.*`. By convention, name the support branch after the version it patches: `support/1.4.0`.

**`devBranchRegEx`** — Regular expression identifying `dev/*` branches. Must contain exactly one capture group whose match becomes the ticket identifier in the pre-release tag. Default: `^dev\/(.+)$`. The captured value is coerced to lowercase with hyphens and underscores replaced by dots: `ACME-123` → `acme.123`.

**`patchBranchRegEx`** — Regular expression identifying `patch/*` branches, which produce pre-releases targeting a patch increment. Same capture group requirement as `devBranchRegEx`. Default: `^patch\/(.+)$`.

**`mode`** — Workflow mode. `release` (default) creates release tags directly on the default branch. `candidate` creates candidate tags that must be explicitly promoted. Use `candidate` for pipelines where a build is exercised across multiple stages before shipping.

## Branching Approach

vergant has opinions about branching that shed unnecessary GitFlow complexity.

**Default branch** — The trunk. Applying vergant's approach renders a separate `dev` or `develop` branch unnecessary. Releases are identified by version tags, not branch names.

**`dev/*` branches** — For features and changes targeting a minor (or major) release. The captured suffix becomes the ticket identifier in the pre-release tag (`dev/ACME-123` → `acme.123`).

**`patch/*` branches** — For fixes targeting a patch release. Merge into the relevant `support/*` branch. Create from the `support/*` branch being patched, not from main.

**`support/*` branches** — For maintaining a previously released version. Create from the release tag being patched (e.g. `support/1.4.0`). `patch/*` branches merge here; `vergant new` on `support/*` after each merge produces the patch release.

## Example Scenarios

### Initial version

No prior tags exist. `majorVersion` is `1`.

```
branch: main
previous: (none)
result:   r1.0.0
```

### Major version bump

`majorVersion` in config has been advanced to `2`.

```
branch:   main
previous: r1.4.0
result:   r2.0.0
```

### Minor version increment

Normal release on the default branch.

```
branch:   main
previous: r1.4.0
result:   r1.5.0
```

### Patch increment

On a patch branch cut from `r1.4.0`.

```
branch:   support/1.4.0
previous: r1.4.0
result:   r1.4.1
```

### First dev build on a branch

`dev/ACME-123` was cut after `r1.4.0`. The `dev/` prefix signals a minor-increment target.

```
branch:       dev/ACME-123
last release: r1.4.0
result:       d1.5.0-acme.123.0
```

The base version `1.5.0` is the target next release — what this branch will produce when merged to main.

### Subsequent dev builds

Each `vergant new` on the same branch increments the build counter.

```
branch:   dev/ACME-123
last dev: d1.5.0-acme.123.2
result:   d1.5.0-acme.123.3
```

### Dev branch rebased on a newer base

When a dev branch is rebased on a newer main, the target advances and the counter resets.

```
branch:       dev/ACME-123
last release: r1.6.0
last dev:     d1.5.0-acme.123.3
result:       d1.7.0-acme.123.0
```

### Patch dev build

`patch/ACME-456` was cut from `support/1.4.0` after `r1.4.2`. The `patch/` prefix signals a patch-increment target.

```
branch:       patch/ACME-456
last release: r1.4.2
result:       d1.4.3-acme.456.0
```

### Candidate to release (candidate mode)

`vergant promote` is called on the commit that carries `c1.5.0`.

```
branch:   main
HEAD has: c1.5.0
result:   r1.5.0
```

## GitHub Actions

vergant is a CLI, which makes CI integration a thin wrapper: install the tool, run it. A single workflow covers the full lifecycle — versioning on every branch push, and release when the result is an `r*` tag.

A single workflow is required because GitHub Actions will not trigger a second workflow from a tag pushed by `GITHUB_TOKEN`. The release steps run as conditional steps in the same job, not as a separate workflow triggered by the tag.

`fetch-depth: 0` is required. Without full commit history, `git tag --list --merged` cannot correctly scope tags to branch ancestry, which is the mechanism that makes parallel branch isolation work.

```yaml
# .github/workflows/delivery.yml
name: delivery

on:
  push:
    branches:
      - '**'

jobs:
  version:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Install vergant
        run: go install github.com/forgant-foundry/vergant/cmd/vergant@latest

      - name: Configure git identity
        run: |
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git config user.name "github-actions[bot]"

      - name: Apply version
        id: version
        run: echo "tag=$(vergant new)" >> $GITHUB_OUTPUT

      - name: Build release binaries
        if: startsWith(steps.version.outputs.tag, 'r')
        run: |
          GOOS=linux   GOARCH=amd64 go build -o dist/vergant-linux-amd64        ./cmd/vergant
          GOOS=darwin  GOARCH=amd64 go build -o dist/vergant-darwin-amd64       ./cmd/vergant
          GOOS=darwin  GOARCH=arm64 go build -o dist/vergant-darwin-arm64       ./cmd/vergant
          GOOS=windows GOARCH=amd64 go build -o dist/vergant-windows-amd64.exe  ./cmd/vergant

      - name: Create release
        if: startsWith(steps.version.outputs.tag, 'r')
        run: |
          VERSION="${{ steps.version.outputs.tag }}"
          gh release create "$VERSION" \
            --title "${VERSION#r}" \
            --generate-notes \
            dist/*
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

The `branches: ['**']` trigger fires on branch pushes only — GitHub Actions excludes tag pushes from this filter, preventing a loop where vergant's tag triggers another version run.

`permissions: contents: write` authorises `GITHUB_TOKEN` to push tags and create releases. No additional secrets or tokens are required.

## Protecting Tags

Version tags are the system of record for your release history. Protect them. Your git hosting platform likely provides tag protection rules — use them to restrict who can create or delete tags matching the `r*`, `c*`, and `d*` patterns. A deleted or forged version tag is a corrupted release history.
