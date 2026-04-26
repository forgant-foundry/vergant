package testutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// RepoHelper manages a temporary git repository for integration tests.
// Tag timestamps are assigned deterministically via GIT_COMMITTER_DATE, so no
// sleeping is needed to guarantee sort order.
type RepoHelper struct {
	t    *testing.T
	Dir  string
	seq  int
	base time.Time
}

// NewRepo initialises a fresh temporary git repository on branch main.
// The directory is cleaned up automatically when the test ends.
func NewRepo(t *testing.T) *RepoHelper {
	t.Helper()
	h := &RepoHelper{
		t:    t,
		Dir:  t.TempDir(),
		base: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	h.run("init", "-b", "main")
	h.run("config", "user.email", "test@example.com")
	h.run("config", "user.name", "Test")
	return h
}

// Commit stages a file change and creates a commit with a deterministic timestamp.
func (h *RepoHelper) Commit() {
	h.t.Helper()
	h.seq++
	f := filepath.Join(h.Dir, "file.txt")
	if err := os.WriteFile(f, []byte(fmt.Sprintf("%d", h.seq)), 0644); err != nil {
		h.t.Fatal(err)
	}
	h.run("add", "file.txt")
	h.runDated("commit", "-m", fmt.Sprintf("commit %d", h.seq))
}

// Tag creates an annotated tag at HEAD with a deterministic timestamp.
func (h *RepoHelper) Tag(name string) {
	h.t.Helper()
	h.seq++
	h.runDated("tag", "-a", name, "-m", name)
}

// TaggedCommit is shorthand for Commit followed by Tag.
func (h *RepoHelper) TaggedCommit(name string) {
	h.t.Helper()
	h.Commit()
	h.Tag(name)
}

// Branch creates a new branch and checks it out.
func (h *RepoHelper) Branch(name string) {
	h.t.Helper()
	h.run("checkout", "-b", name)
}

// Checkout switches to an existing branch.
func (h *RepoHelper) Checkout(name string) {
	h.t.Helper()
	h.run("checkout", name)
}

// runDated runs a git command with GIT_COMMITTER_DATE and GIT_AUTHOR_DATE set to
// an incrementing timestamp, giving each operation a unique, ordered time value.
func (h *RepoHelper) runDated(args ...string) {
	h.t.Helper()
	date := h.base.Add(time.Duration(h.seq) * time.Second).Format(time.RFC3339)
	cmd := exec.Command("git", args...)
	cmd.Dir = h.Dir
	cmd.Env = append(os.Environ(),
		"GIT_COMMITTER_DATE="+date,
		"GIT_AUTHOR_DATE="+date,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		h.t.Fatalf("git %v: %s", args, out)
	}
}

func (h *RepoHelper) run(args ...string) {
	h.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = h.Dir
	if out, err := cmd.CombinedOutput(); err != nil {
		h.t.Fatalf("git %v: %s", args, out)
	}
}
