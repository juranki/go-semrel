// Package inspectgit collects version tags and unreleased commits from
// a git repository
package inspectgit

import (
	"io"
	"time"

	"github.com/blang/semver"
	"github.com/juranki/go-semrel/semrel"
	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// implement sermver.Commit interface for object.Commit
type newCommit object.Commit

func (commit *newCommit) Msg() string     { return commit.Message }
func (commit *newCommit) SHA() string     { return commit.Hash.String() }
func (commit *newCommit) Time() time.Time { return commit.Author.When }

// InspectGit returns current version and list of unreleased changes
//
// Open repository at `path` and traverse parents of `HEAD` to find
// the tag that represents previous release and the commits that haven't
// been released yet.
func InspectGit(path string) (semver.Version, []semrel.Commit, error) {

	r, err := git.PlainOpen(path)
	if err != nil {
		return semver.MustParse("0.0.0"), nil, err
	}

	versions, err := getVersions(r)
	if err != nil {
		return semver.MustParse("0.0.0"), nil, err
	}

	return getUnreleasedCommits(r, versions)
}

// Search semantic versions from tags
func getVersions(r *git.Repository) (map[string]semver.Version, error) {
	versions := make(map[string]semver.Version)

	addIfSemVer := func(sha string, version string) {
		sv, err := semver.ParseTolerant(version)
		if err == nil {
			versions[sha] = sv
		}
	}

	tagRefs, err := r.Tags()
	if err != nil {
		return nil, err
	}
	err = tagRefs.ForEach(func(t *plumbing.Reference) error {
		addIfSemVer(t.Hash().String(), t.Name().Short())
		return nil
	})
	if err != nil {
		return nil, err
	}

	tagObjects, err := r.TagObjects()
	if err != nil {
		return nil, err
	}
	err = tagObjects.ForEach(func(t *object.Tag) error {
		addIfSemVer(t.Target.String(), t.Name)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return versions, nil
}

func getUnreleasedCommits(r *git.Repository, versions map[string]semver.Version) (semver.Version, []semrel.Commit, error) {
	var traverse func(*object.Commit, bool) error
	currVersion := semver.MustParse("0.0.0")
	cache := newCache()
	traverse = func(c *object.Commit, isNew bool) error {
		tag, hasTag := versions[c.Hash.String()]
		if hasTag && isNew {
			if tag.GT(currVersion) {
				currVersion = tag
			}
		}
		if !cache.add(c, isNew && !hasTag) {
			return nil
		}
		parents := c.Parents()
		defer parents.Close()
		for {
			cc, err := parents.Next()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			// fmt.Println(c.NumParents(), c.Hash, " -> ", cc.Hash)
			traverse(cc, isNew && !hasTag)
		}
	}
	h, err := r.Head()
	if err != nil {
		return currVersion, nil, errors.Wrap(err, "get HEAD")
	}
	hCommit, err := r.CommitObject(h.Hash())
	if err != nil {
		return currVersion, nil, err
	}

	err = traverse(hCommit, true)
	if err != nil {
		return currVersion, nil, err
	}
	return currVersion, cache.newCommits(), nil
}

type commitCacheEntry struct {
	isNew  bool
	commit semrel.Commit
}

type commitCache struct {
	commits map[string]*commitCacheEntry
}

func newCache() *commitCache {
	return &commitCache{
		commits: map[string]*commitCacheEntry{},
	}
}

func (cache *commitCache) newCommits() []semrel.Commit {
	rv := []semrel.Commit{}
	for _, entry := range cache.commits {
		if entry.isNew {
			rv = append(rv, entry.commit)
		}
	}
	return rv
}

func (cache *commitCache) add(commit *object.Commit, isNew bool) bool {
	entry, hasEntry := cache.commits[commit.Hash.String()]
	if !hasEntry {
		nc := newCommit(*commit)
		cache.commits[commit.Hash.String()] = &commitCacheEntry{
			isNew:  isNew,
			commit: &nc,
		}
		return true
	}
	if !entry.isNew {
		return false
	}
	entry.isNew = isNew
	return true
}
