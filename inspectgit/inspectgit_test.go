package inspectgit

import (
	"fmt"
	"log"
	"testing"
	"time"

	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

func setupRepo() (*git.Repository, *git.Worktree) {
	r, err := git.Init(memory.NewStorage(), memfs.New())
	if err != nil {
		log.Fatalln(err)
	}
	w, err := r.Worktree()
	if err != nil {
		log.Fatalln(err)
	}
	return r, w
}

func commit(w *git.Worktree, msg string) plumbing.Hash {
	hash, err := w.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "a",
			Email: "a@b",
			When:  time.Now(),
		},
	})
	// fmt.Println(msg, hash.String())
	if err != nil {
		log.Fatal(err)
	}
	return hash
}
func merge(w *git.Worktree, msg string, parents []plumbing.Hash) plumbing.Hash {
	hash, err := w.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "a",
			Email: "a@b",
			When:  time.Now(),
		},
		Parents: parents,
	})
	// fmt.Println(msg, hash.String())
	if err != nil {
		log.Fatal(err)
	}
	return hash
}

func tag(r *git.Repository, hash plumbing.Hash, version string) {
	n := plumbing.ReferenceName(fmt.Sprintf("refs/tags/v%s", version))
	tag := plumbing.NewHashReference(n, hash)
	err := r.Storer.SetReference(tag)
	if err != nil {
		log.Fatal(err)
	}
}

func checkReleaseData(t *testing.T, r *git.Repository, n int, version string) {
	vs, err := getVersions(r)
	if err != nil {
		t.Error(err)
	}
	v, cs, err := getUnreleasedCommits(r, vs)
	if err != nil {
		log.Fatal(err)
	}
	if len(cs) != n {
		fmt.Printf("commits: %+v\n", cs)
		fmt.Printf("versions: %+v\n", vs)
		fmt.Printf("version: %+v\n", v)
		t.Errorf("expected %d commits, got %d", n, len(cs))
	}
	if v.String() != version {
		fmt.Printf("commits: %+v\n", cs)
		fmt.Printf("versions: %+v\n", vs)
		fmt.Printf("version: %+v\n", v)
		t.Errorf("expected %s, got %s", version, v.String())
	}
}

func TestGetVersions(t *testing.T) {
	r, w := setupRepo()

	checkVersionCount := func(n int) {
		vs, err := getVersions(r)
		if err != nil {
			t.Error(err)
		}
		if len(vs) != n {
			t.Errorf("expected %d versions, got %d", n, len(vs))
		}
	}

	checkVersionCount(0)

	hash := commit(w, "initial")
	tag(r, hash, "2.3.4")

	checkVersionCount(1)
}

func TestGetUnreleasedCommits(t *testing.T) {
	r, w := setupRepo()

	commit(w, "initial")
	checkReleaseData(t, r, 1, "0.0.0")
	hash := commit(w, "1")
	checkReleaseData(t, r, 2, "0.0.0")
	tag(r, hash, "1.0.0")
	checkReleaseData(t, r, 0, "1.0.0")
	hash = commit(w, "2")
	checkReleaseData(t, r, 1, "1.0.0")
}

func TestMerge(t *testing.T) {
	r, w := setupRepo()

	commit(w, "initial")
	a1 := commit(w, "a1")
	a2 := commit(w, "a2")
	checkReleaseData(t, r, 3, "0.0.0")
	tag(r, a2, "1.0.0")
	checkReleaseData(t, r, 0, "1.0.0")
	a3 := commit(w, "a3")
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
	commit(w, "b1")
	// fmt.Printf("%+v\n", head)
	checkReleaseData(t, r, 3, "0.0.0")
	commit(w, "b2")
	b3 := commit(w, "b3")
	checkReleaseData(t, r, 5, "0.0.0")
	merge(w, "merge", []plumbing.Hash{b3, a3})
	checkReleaseData(t, r, 5, "1.0.0")
}
