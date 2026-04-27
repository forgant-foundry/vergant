package git_test

import (
	"testing"

	"github.com/forgant-foundry/vergant/internal/git"
	"github.com/forgant-foundry/vergant/internal/testutil"
	"github.com/forgant-foundry/vergant/internal/version"
)

func newRepo(t *testing.T) (*testutil.RepoHelper, *git.Git) {
	t.Helper()
	r := testutil.NewRepo(t)
	return r, git.NewGitRepository(r.Dir, version.DefaultPrefixes())
}

func newPathRepo(t *testing.T, path string) (*testutil.RepoHelper, *git.Git) {
	t.Helper()
	r := testutil.NewRepo(t)
	return r, git.NewGitRepository(r.Dir, version.DefaultPrefixes()).WithPath(path)
}

// --- Basic operations ---

func TestLastVersionNoTags(t *testing.T) {
	r, g := newRepo(t)
	r.Commit()

	tag, err := g.LastVersion()
	if err != nil {
		t.Fatal(err)
	}
	if tag != "" {
		t.Errorf("expected empty, got %q", tag)
	}
}

func TestLastVersionAndRelease(t *testing.T) {
	r, g := newRepo(t)
	r.TaggedCommit("v1.0.0")
	r.TaggedCommit("c1.1.0")
	r.TaggedCommit("v1.1.0")
	r.TaggedCommit("c1.2.0")

	if got, _ := g.LastVersion(); got != "c1.2.0" {
		t.Errorf("LastVersion: got %q, want c1.2.0", got)
	}
	if got, _ := g.LastRelease(); got != "v1.1.0" {
		t.Errorf("LastRelease: got %q, want v1.1.0", got)
	}
}

func TestLastVersionForDevelopment(t *testing.T) {
	r, g := newRepo(t)
	r.TaggedCommit("v1.2.0")
	r.TaggedCommit("c1.3.0")

	// no dev tag yet — returns empty string
	if got, _ := g.LastVersionForDevelopment("acme.123"); got != "" {
		t.Errorf("no dev tag: got %q, want empty", got)
	}

	r.TaggedCommit("d1.3.0-acme.123.0")

	if got, _ := g.LastVersionForDevelopment("acme.123"); got != "d1.3.0-acme.123.0" {
		t.Errorf("dev tag: got %q, want d1.3.0-acme.123.0", got)
	}
}

func TestCurrentBranch(t *testing.T) {
	r, g := newRepo(t)
	r.Commit()

	if got, err := g.CurrentBranch(); err != nil || got != "main" {
		t.Errorf("CurrentBranch: got %q, err %v", got, err)
	}
}

func TestCurrentTags(t *testing.T) {
	r, g := newRepo(t)
	r.TaggedCommit("v1.0.0")
	r.TaggedCommit("c1.1.0")

	tags, err := g.CurrentTags()
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0] != "c1.1.0" {
		t.Errorf("CurrentTags: got %v, want [c1.1.0]", tags)
	}
}

func TestListTagsOrdering(t *testing.T) {
	r, g := newRepo(t)
	r.TaggedCommit("v1.0.0")
	r.TaggedCommit("c1.1.0")
	r.TaggedCommit("v1.1.0")

	got, err := g.ListTags()
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.1.0\nc1.1.0\nv1.0.0\n" {
		t.Errorf("ListTags: got %q", got)
	}
}

func TestNotAGitRepository(t *testing.T) {
	if _, err := git.NewGitRepository(t.TempDir(), version.DefaultPrefixes()).LastVersion(); err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

// --- Reachability ---
//
// These tests verify the --merged boundary: each asserts which tags are and are
// not visible from a given branch, which is the property that makes branch-aware
// version calculation correct.

// TestReachability_AncestorTagsVisible confirms the positive case: a branch sees
// all tags on commits that are part of its ancestry.
func TestReachability_AncestorTagsVisible(t *testing.T) {
	r, g := newRepo(t)
	r.TaggedCommit("v1.0.0")
	r.TaggedCommit("c1.1.0")

	r.Branch("feature/acme-1")
	r.Commit()

	if got, _ := g.LastVersion(); got != "c1.1.0" {
		t.Errorf("dev branch should see pre-branch ancestor tags: got %q, want c1.1.0", got)
	}
}

// TestReachability_DevBranchIsolatedFromPostBranchMainTags confirms that a dev
// branch cannot see tags added to main after the branch was cut.
func TestReachability_DevBranchIsolatedFromPostBranchMainTags(t *testing.T) {
	r, g := newRepo(t)
	r.TaggedCommit("v1.0.0")

	r.Branch("feature/acme-1")
	r.Commit()

	// new release on main after the branch was cut
	r.Checkout("main")
	r.TaggedCommit("v1.1.0")

	r.Checkout("feature/acme-1")
	if got, _ := g.LastVersion(); got != "v1.0.0" {
		t.Errorf("dev branch should not see post-branch main tag: got %q, want v1.0.0", got)
	}
}

// TestReachability_ParallelDevBranchesIsolated confirms that two dev branches
// branched from the same point cannot see each other's tags.
func TestReachability_ParallelDevBranchesIsolated(t *testing.T) {
	r, g := newRepo(t)
	r.TaggedCommit("v1.0.0")

	r.Branch("feature/acme-1")
	r.TaggedCommit("d1.1.0-acme.1.0")

	r.Checkout("main")
	r.Branch("feature/acme-2")
	r.TaggedCommit("d1.1.0-acme.2.0")

	// acme-2 cannot see acme-1's dev tag
	if got, _ := g.LastVersionForDevelopment("acme.1"); got == "d1.1.0-acme.1.0" {
		t.Error("feature/acme-2 should not see feature/acme-1 dev tag")
	}

	r.Checkout("feature/acme-1")

	// acme-1 cannot see acme-2's dev tag
	if got, _ := g.LastVersionForDevelopment("acme.2"); got == "d1.1.0-acme.2.0" {
		t.Error("feature/acme-1 should not see feature/acme-2 dev tag")
	}
}

// TestReachability_PatchBranchIsolatedFromNewerMainTags confirms that a patch
// branch cannot see releases added to main after the branch was cut.
func TestReachability_PatchBranchIsolatedFromNewerMainTags(t *testing.T) {
	r, g := newRepo(t)
	r.TaggedCommit("v1.0.0")
	r.TaggedCommit("v1.1.0")

	r.Branch("support/1.1.0")
	r.Commit()
	r.Tag("v1.1.1")

	// newer releases on main after the patch branch was cut
	r.Checkout("main")
	r.TaggedCommit("v1.2.0")
	r.TaggedCommit("v1.3.0")

	r.Checkout("support/1.1.0")
	if got, _ := g.LastRelease(); got != "v1.1.1" {
		t.Errorf("patch branch should not see post-branch main releases: got %q, want v1.1.1", got)
	}
}

// TestReachability_PatchTagNotVisibleFromMain confirms that a tag created on a
// patch branch is not visible from main until the patch branch is merged.
func TestReachability_PatchTagNotVisibleFromMain(t *testing.T) {
	r, g := newRepo(t)
	r.TaggedCommit("v1.0.0")
	r.TaggedCommit("v1.1.0")

	r.Branch("support/1.1.0")
	r.Commit()
	r.Tag("v1.1.1")

	r.Checkout("main")
	if got, _ := g.LastRelease(); got != "v1.1.0" {
		t.Errorf("main should not see unmerged patch tag: got %q, want v1.1.0", got)
	}
}

// --- Multi-path versioning ---
//
// These tests verify that a path-scoped Git client sees only tags under its own
// path prefix and that the prefix is stripped from all returned version strings.

// TestPath_VersionsScopedToPath confirms that two path-scoped clients operating
// on the same repo see only their own version sequence.
func TestPath_VersionsScopedToPath(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("sockets/v1.0.0")
	r.TaggedCommit("http/v2.0.0")

	sockets := git.NewGitRepository(r.Dir, version.DefaultPrefixes()).WithPath("sockets")
	httpLib := git.NewGitRepository(r.Dir, version.DefaultPrefixes()).WithPath("http")

	if got, _ := sockets.LastRelease(); got != "v1.0.0" {
		t.Errorf("sockets.LastRelease: got %q, want v1.0.0", got)
	}
	if got, _ := httpLib.LastRelease(); got != "v2.0.0" {
		t.Errorf("http.LastRelease: got %q, want v2.0.0", got)
	}

	// each path is opaque to the other
	if got, _ := sockets.LastVersion(); got == "v2.0.0" {
		t.Error("sockets path must not see http/v2.0.0")
	}
	if got, _ := httpLib.LastVersion(); got == "v1.0.0" {
		t.Error("http path must not see sockets/v1.0.0")
	}
}

// TestPath_LastVersionForDevelopmentScoped confirms dev tag lookup is scoped to path.
func TestPath_LastVersionForDevelopmentScoped(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("sockets/v1.0.0")
	r.TaggedCommit("sockets/d1.1.0-acme.1.0")
	r.TaggedCommit("http/d1.1.0-acme.1.0")

	sockets := git.NewGitRepository(r.Dir, version.DefaultPrefixes()).WithPath("sockets")
	httpLib := git.NewGitRepository(r.Dir, version.DefaultPrefixes()).WithPath("http")

	if got, _ := sockets.LastVersionForDevelopment("acme.1"); got != "d1.1.0-acme.1.0" {
		t.Errorf("sockets dev tag: got %q, want d1.1.0-acme.1.0", got)
	}
	if got, _ := httpLib.LastVersionForDevelopment("acme.1"); got != "d1.1.0-acme.1.0" {
		t.Errorf("http dev tag: got %q, want d1.1.0-acme.1.0", got)
	}
}

// TestPath_CurrentTagsFilteredToPath confirms that CurrentTags returns only
// path-scoped tags and strips the path prefix from the returned names.
func TestPath_CurrentTagsFilteredToPath(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit()
	r.Tag("sockets/c1.0.0")
	r.Tag("http/c2.0.0")

	sockets := git.NewGitRepository(r.Dir, version.DefaultPrefixes()).WithPath("sockets")

	tags, err := sockets.CurrentTags()
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0] != "c1.0.0" {
		t.Errorf("CurrentTags: got %v, want [c1.0.0]", tags)
	}
}

// TestPath_ListTagsScopedToPath confirms that ListTags returns only path-scoped
// tags with the path prefix stripped, preserving date ordering.
func TestPath_ListTagsScopedToPath(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("sockets/v1.0.0")
	r.TaggedCommit("http/v2.0.0")
	r.TaggedCommit("sockets/c1.1.0")

	sockets := git.NewGitRepository(r.Dir, version.DefaultPrefixes()).WithPath("sockets")

	got, err := sockets.ListTags()
	if err != nil {
		t.Fatal(err)
	}
	want := "c1.1.0\nv1.0.0\n"
	if got != want {
		t.Errorf("ListTags: got %q, want %q", got, want)
	}
}

// TestPath_ReachabilityRespected confirms that the --merged boundary still applies
// when a path is set: a dev branch cannot see post-branch path-scoped main tags.
func TestPath_ReachabilityRespected(t *testing.T) {
	r, g := newPathRepo(t, "sockets")
	r.TaggedCommit("sockets/v1.0.0")

	r.Branch("dev/acme-1")
	r.Commit()

	r.Checkout("main")
	r.TaggedCommit("sockets/v1.1.0")

	r.Checkout("dev/acme-1")
	if got, _ := g.LastRelease(); got != "v1.0.0" {
		t.Errorf("dev branch should not see post-branch main path tag: got %q, want v1.0.0", got)
	}
}
