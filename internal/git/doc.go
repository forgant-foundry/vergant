// Package git defines the [Repository] interface and provides [Git], its native
// implementation using os/exec.
//
// # Reachable tags
//
// All tag queries are built on:
//
//	git tag --list --sort=-taggerdate --merged
//
// The --merged flag restricts results to tags on commits that are ancestors of
// the current HEAD. This is the mechanism that makes branch-aware versioning
// correct without coordination:
//
//   - A dev branch sees only tags that existed on main at branch-cut time, plus
//     its own tags. Tags added to main after branching are invisible.
//   - Two parallel dev branches are opaque to each other regardless of creation
//     order.
//   - A patch branch cannot see releases added to main after it was cut.
//   - Once a branch is merged, its tags become reachable from the merge target.
//
// "What is the last version?" is therefore always answered relative to the
// current branch's ancestry, not the repository as a whole.
//
// # Repository interface
//
// [Repository] is the interface consumed by the strategy and tool layers.
// Accepting an interface rather than a concrete type keeps those layers
// independent of the git implementation and testable with [testutil.StubRepository].
//
// # FindTag
//
// [Git.FindTag] accepts a regexp pattern and returns the first matching tag from
// the --merged list. All higher-level queries — [Git.LastVersion],
// [Git.LastRelease], [Git.LastVersionForDevelopment] — are built on FindTag.
//
// # Tag creation
//
// [Git.TagVersion] creates an annotated tag (preserving taggerdate for correct
// sort ordering) and pushes it to origin. Annotated tags are required; lightweight
// tags do not carry a taggerdate and will sort incorrectly under --sort=-taggerdate.
package git
