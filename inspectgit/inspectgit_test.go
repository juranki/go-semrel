package inspectgit

import (
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

	hash, err := w.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "a",
			Email: "a@b",
			When:  time.Now(),
		},
	})
	n := plumbing.ReferenceName("refs/tags/v2.3.4")
	tag := plumbing.NewHashReference(n, hash)
	err = r.Storer.SetReference(tag)
	if err != nil {
		t.Error(err)
	}
	checkVersionCount(1)
}

func TestGetUnreleasedCommits(t *testing.T) {
	r, w := setupRepo()

	checkCommitCount := func(n int) {
		vs, err := getVersions(r)
		if err != nil {
			t.Error(err)
		}
		_, cs, err := getUnreleasedCommits(r, vs)
		if err != nil {
			log.Fatal(err)
		}
		if len(cs) != n {
			t.Errorf("expected %d commits, got %d", n, len(cs))
		}
	}

	_, err := w.Commit("initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "a",
			Email: "a@b",
			When:  time.Now(),
		},
	})
	// n := plumbing.ReferenceName("refs/tags/v2.3.4")
	// tag := plumbing.NewHashReference(n, hash)
	// err = r.Storer.SetReference(tag)
	if err != nil {
		t.Error(err)
	}
	checkCommitCount(1)
}
