// Package testutil provides test infrastructure shared across vergant's internal
// packages.
//
// # StubRepository
//
// [StubRepository] implements [git.Repository] with configurable return values.
// Set the Val fields for the common case; set the Fn fields when a test needs
// dynamic or stateful behaviour. The Tagged field accumulates every version
// passed to TagVersion, enabling assertions on what was tagged without a real
// repository.
//
// Use StubRepository in unit tests for the strategy and tool layers — tests
// that exercise version calculation logic where the git behaviour is irrelevant.
//
// # RepoHelper
//
// [RepoHelper] creates and manages a temporary git repository backed by
// t.TempDir(), which is cleaned up automatically when the test ends. No manual
// teardown is required.
//
// Tag timestamps are controlled via GIT_COMMITTER_DATE, set to a fixed base
// time incremented by one second per operation. This guarantees deterministic
// --sort=-taggerdate ordering without sleeping between tag creations — the root
// cause of slow tests in naive git integration test helpers.
//
// Use RepoHelper in tests that must exercise real git behaviour: the --merged
// reachability boundary, branch isolation, and tag ordering.
package testutil
