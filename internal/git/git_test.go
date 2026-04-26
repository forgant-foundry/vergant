package git_test

import (
	"testing"

	"github.com/forgant-foundry/vergant/internal/git"
	"github.com/forgant-foundry/vergant/internal/testutil"
)

// --- Basic operations ---

func TestLastVersionNoTags(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit()

	tag, err := git.New(r.Dir).LastVersion()
	if err != nil {
		t.Fatal(err)
	}
	if tag != "" {
		t.Errorf("expected empty, got %q", tag)
	}
}

func TestLastVersionAndRelease(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("r1.0.0")
	r.TaggedCommit("c1.1.0")
	r.TaggedCommit("r1.1.0")
	r.TaggedCommit("c1.2.0")

	g := git.New(r.Dir)
	if got, _ := g.LastVersion(); got != "c1.2.0" {
		t.Errorf("LastVersion: got %q, want c1.2.0", got)
	}
	if got, _ := g.LastRelease(); got != "r1.1.0" {
		t.Errorf("LastRelease: got %q, want r1.1.0", got)
	}
}

func TestLastVersionForDevelopment(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("r1.2.0")
	r.TaggedCommit("c1.3.0")

	g := git.New(r.Dir)

	// no dev tag yet — falls back to last release/candidate
	if got, _ := g.LastVersionForDevelopment("acme.123"); got != "c1.3.0" {
		t.Errorf("fallback: got %q, want c1.3.0", got)
	}

	r.TaggedCommit("d1.3.0+acme.123.0")

	if got, _ := g.LastVersionForDevelopment("acme.123"); got != "d1.3.0+acme.123.0" {
		t.Errorf("dev tag: got %q, want d1.3.0+acme.123.0", got)
	}
}

func TestCurrentBranch(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit()

	if got, err := git.New(r.Dir).CurrentBranch(); err != nil || got != "main" {
		t.Errorf("CurrentBranch: got %q, err %v", got, err)
	}
}

func TestCurrentTags(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("r1.0.0")
	r.TaggedCommit("c1.1.0")

	tags, err := git.New(r.Dir).CurrentTags()
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0] != "c1.1.0" {
		t.Errorf("CurrentTags: got %v, want [c1.1.0]", tags)
	}
}

func TestListTagsOrdering(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("r1.0.0")
	r.TaggedCommit("c1.1.0")
	r.TaggedCommit("r1.1.0")

	got, err := git.New(r.Dir).ListTags()
	if err != nil {
		t.Fatal(err)
	}
	if got != "r1.1.0\nc1.1.0\nr1.0.0\n" {
		t.Errorf("ListTags: got %q", got)
	}
}

func TestNotAGitRepository(t *testing.T) {
	if _, err := git.New(t.TempDir()).LastVersion(); err == nil {
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
	r := testutil.NewRepo(t)
	r.TaggedCommit("r1.0.0")
	r.TaggedCommit("c1.1.0")

	r.Branch("feature/acme-1")
	r.Commit()

	g := git.New(r.Dir)
	if got, _ := g.LastVersion(); got != "c1.1.0" {
		t.Errorf("dev branch should see pre-branch ancestor tags: got %q, want c1.1.0", got)
	}
}

// TestReachability_DevBranchIsolatedFromPostBranchMainTags confirms that a dev
// branch cannot see tags added to main after the branch was cut.
func TestReachability_DevBranchIsolatedFromPostBranchMainTags(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("r1.0.0")

	r.Branch("feature/acme-1")
	r.Commit()

	// new release on main after the branch was cut
	r.Checkout("main")
	r.TaggedCommit("r1.1.0")

	r.Checkout("feature/acme-1")
	g := git.New(r.Dir)
	if got, _ := g.LastVersion(); got != "r1.0.0" {
		t.Errorf("dev branch should not see post-branch main tag: got %q, want r1.0.0", got)
	}
}

// TestReachability_ParallelDevBranchesIsolated confirms that two dev branches
// branched from the same point cannot see each other's tags.
func TestReachability_ParallelDevBranchesIsolated(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("r1.0.0")

	r.Branch("feature/acme-1")
	r.TaggedCommit("d1.0.0+acme.1.0")

	r.Checkout("main")
	r.Branch("feature/acme-2")
	r.TaggedCommit("d1.0.0+acme.2.0")

	g := git.New(r.Dir)

	// acme-2 cannot see acme-1's dev tag
	if got, _ := g.LastVersionForDevelopment("acme.1"); got == "d1.0.0+acme.1.0" {
		t.Error("feature/acme-2 should not see feature/acme-1 dev tag")
	}

	r.Checkout("feature/acme-1")

	// acme-1 cannot see acme-2's dev tag
	if got, _ := g.LastVersionForDevelopment("acme.2"); got == "d1.0.0+acme.2.0" {
		t.Error("feature/acme-1 should not see feature/acme-2 dev tag")
	}
}

// TestReachability_PatchBranchIsolatedFromNewerMainTags confirms that a patch
// branch cannot see releases added to main after the branch was cut.
func TestReachability_PatchBranchIsolatedFromNewerMainTags(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("r1.0.0")
	r.TaggedCommit("r1.1.0")

	r.Branch("support/1.1.0")
	r.Commit()
	r.Tag("r1.1.1")

	// newer releases on main after the patch branch was cut
	r.Checkout("main")
	r.TaggedCommit("r1.2.0")
	r.TaggedCommit("r1.3.0")

	r.Checkout("support/1.1.0")
	g := git.New(r.Dir)
	if got, _ := g.LastRelease(); got != "r1.1.1" {
		t.Errorf("patch branch should not see post-branch main releases: got %q, want r1.1.1", got)
	}
}

// TestReachability_PatchTagNotVisibleFromMain confirms that a tag created on a
// patch branch is not visible from main until the patch branch is merged.
func TestReachability_PatchTagNotVisibleFromMain(t *testing.T) {
	r := testutil.NewRepo(t)
	r.TaggedCommit("r1.0.0")
	r.TaggedCommit("r1.1.0")

	r.Branch("support/1.1.0")
	r.Commit()
	r.Tag("r1.1.1")

	r.Checkout("main")
	g := git.New(r.Dir)
	if got, _ := g.LastRelease(); got != "r1.1.0" {
		t.Errorf("main should not see unmerged patch tag: got %q, want r1.1.0", got)
	}
}
