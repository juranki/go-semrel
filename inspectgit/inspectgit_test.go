package inspectgit

import (
	"fmt"
	"log"
	"testing"
	"time"

	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func setupRepo(t *testing.T) (*git.Repository, *git.Worktree) {
	t.Helper()
	r, err := git.Init(memory.NewStorage(), memfs.New())
	if err != nil {
		t.Fatal(err)
	}
	w, err := r.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	return r, w
}

func commit(t *testing.T, w *git.Worktree, msg string) plumbing.Hash {
	t.Helper()
	hash, err := w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "a",
			Email: "a@b",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
func merge(t *testing.T, w *git.Worktree, msg string, parents []plumbing.Hash) plumbing.Hash {
	t.Helper()
	hash, err := w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "a",
			Email: "a@b",
			When:  time.Now(),
		},
		Parents: parents,
	})
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func tag(t *testing.T, r *git.Repository, hash plumbing.Hash, version string) {
	t.Helper()
	n := plumbing.ReferenceName(fmt.Sprintf("refs/tags/v%s", version))
	tag := plumbing.NewHashReference(n, hash)
	err := r.Storer.SetReference(tag)
	if err != nil {
		t.Fatal(err)
	}
}

func checkReleaseData(t *testing.T, r *git.Repository, n int, version string) {
	t.Helper()
	vs, err := getVersions(r)
	if err != nil {
		t.Error(err)
	}
	vcsData, err := getUnreleasedCommits(r, vs)
	if err != nil {
		log.Fatal(err)
	}
	if len(vcsData.UnreleasedCommits) != n {
		t.Logf("commits: %+v\n", vcsData.UnreleasedCommits)
		t.Logf("versions: %+v\n", vs)
		t.Logf("version: %+v\n", vcsData.CurrentVersion)
		t.Errorf("got %d commits, want %d", len(vcsData.UnreleasedCommits), n)
	}
	if vcsData.CurrentVersion.String() != version {
		t.Logf("commits: %+v\n", vcsData.UnreleasedCommits)
		t.Logf("versions: %+v\n", vs)
		t.Logf("version: %+v\n", vcsData.CurrentVersion)
		t.Errorf("got %s, want %s", vcsData.CurrentVersion.String(), version)
	}
}

func TestGetVersions(t *testing.T) {
	r, w := setupRepo(t)

	checkVersionCount := func(n int) {
		vs, err := getVersions(r)
		if err != nil {
			t.Error(err)
		}
		if len(vs) != n {
			t.Errorf("got %d versions, want %d", len(vs), n)
		}
	}

	checkVersionCount(0)

	hash := commit(t, w, "initial")
	tag(t, r, hash, "2.3.4")

	checkVersionCount(1)
}

func TestGetUnreleasedCommits(t *testing.T) {
	r, w := setupRepo(t)

	commit(t, w, "initial")
	checkReleaseData(t, r, 1, "0.0.0")
	hash := commit(t, w, "1")
	checkReleaseData(t, r, 2, "0.0.0")
	tag(t, r, hash, "1.0.0")
	checkReleaseData(t, r, 0, "1.0.0")
	commit(t, w, "2")
	checkReleaseData(t, r, 1, "1.0.0")
}

func TestIgnorePreRelease(t *testing.T) {
	r, w := setupRepo(t)

	commit(t, w, "initial")
	checkReleaseData(t, r, 1, "0.0.0")
	hash := commit(t, w, "1")
	checkReleaseData(t, r, 2, "0.0.0")
	tag(t, r, hash, "1.0.0-pre")
	checkReleaseData(t, r, 2, "0.0.0")
	commit(t, w, "2")
	checkReleaseData(t, r, 3, "0.0.0")
}

func TestMultipleTagsOnCommit(t *testing.T) {
	r, w := setupRepo(t)

	commit(t, w, "initial")
	checkReleaseData(t, r, 1, "0.0.0")
	hash := commit(t, w, "1")
	checkReleaseData(t, r, 2, "0.0.0")
	tag(t, r, hash, "1.0.0")
	tag(t, r, hash, "1.0.1")
	checkReleaseData(t, r, 0, "1.0.1")
	hash = commit(t, w, "2")
	tag(t, r, hash, "2.0.0")
	tag(t, r, hash, "2.0.1")
	checkReleaseData(t, r, 0, "2.0.1")
}

func TestMerge(t *testing.T) {
	r, w := setupRepo(t)

	commit(t, w, "initial")
	a1 := commit(t, w, "a1")
	a2 := commit(t, w, "a2")
	checkReleaseData(t, r, 3, "0.0.0")
	tag(t, r, a2, "1.0.0")
	checkReleaseData(t, r, 0, "1.0.0")
	a3 := commit(t, w, "a3")
	checkReleaseData(t, r, 1, "1.0.0")

	err := w.Checkout(&git.CheckoutOptions{
		Hash:   a1,
		Branch: "refs/heads/b",
		Create: true,
		Force:  true,
	})
	if err != nil {
		t.Error(err)
	}
	checkReleaseData(t, r, 2, "0.0.0")
	f, err := w.Filesystem.Create("foo")
	if err != nil {
		t.Error(err)
	}
	f.Write([]byte("hello"))
	f.Close()
	w.Add("foo")
	commit(t, w, "b1")
	// fmt.Printf("%+v\n", head)
	checkReleaseData(t, r, 3, "0.0.0")
	commit(t, w, "b2")
	b3 := commit(t, w, "b3")
	checkReleaseData(t, r, 5, "0.0.0")
	merge(t, w, "merge", []plumbing.Hash{b3, a3})
	checkReleaseData(t, r, 5, "1.0.0")
}
