// Package inspectgit collects version tags and unreleased commits from
// a git repository
package inspectgit

import (
	"io"
	"sort"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/juranki/go-semrel/semrel"
	"github.com/pkg/errors"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

// VCSData returns current version and list of unreleased changes
//
// Open repository at `path` and traverse parents of `HEAD` to find
// the tag that represents previous release and the commits that haven't
// been released yet.
func VCSData(path string) (*semrel.VCSData, error) {
	return VCSDataWithPrefix(path, "")
}

// VCSDataWithPrefix returns current version and list of unreleased changes
//
// The same as VCSData, but allows prefix before version, when searching earlier
// releases. Versions without the prefix are still recognized.
func VCSDataWithPrefix(path string, prefix string) (*semrel.VCSData, error) {

	r, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}

	versions, err := getVersions(r, prefix)
	if err != nil {
		return nil, err
	}

	data, err := getUnreleasedCommits(r, versions)
	if err != nil {
		return nil, err
	}

	t, err := getHeadTime(r)
	if err != nil {
		return nil, err
	}
	data.Time = *t

	return data, nil
}

func getHeadTime(r *git.Repository) (*time.Time, error) {
	h, err := r.Head()
	if err != nil {
		return nil, errors.Wrap(err, "get HEAD")
	}
	hCommit, err := r.CommitObject(h.Hash())
	if err != nil {
		return nil, err
	}

	return &hCommit.Author.When, nil
}

// Search semantic versions from tags, including pre-releases
// prefix is removed from the tag before trying to parse semantic version
func getVersions(r *git.Repository, prefix string) (map[string]semver.Version, error) {
	versions := make(map[string]semver.Version)

	addIfSemVer := func(sha string, version string) {
		s := strings.TrimPrefix(version, prefix)
		sv, err := semver.ParseTolerant(s)
		if err == nil {
			prevV, prevExists := versions[sha]
			if prevExists && prevV.GT(sv) {
				return
			}
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

func getUnreleasedCommits(r *git.Repository, versions map[string]semver.Version) (*semrel.VCSData, error) {
	var traverse func(*object.Commit, bool, bool) error
	currVersion := semver.MustParse("0.0.0")
	cache := newCache()
	traverse = func(c *object.Commit, isNew bool, isPreReleased bool) error {
		unReleased := isNew
		preReleased := isPreReleased
		tag, hasTag := versions[c.Hash.String()]
		if hasTag {
			if len(tag.Pre) > 0 || len(tag.Build) > 0 {
				preReleased = true
			} else if isNew {
				unReleased = false
				if tag.GT(currVersion) {
					currVersion = tag
				}
			}
		}
		if !cache.add(c, unReleased, preReleased) {
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
			traverse(cc, unReleased, preReleased)
		}
	}
	h, err := r.Head()
	if err != nil {
		return nil, errors.Wrap(err, "get HEAD")
	}
	hCommit, err := r.CommitObject(h.Hash())
	if err != nil {
		return nil, err
	}

	err = traverse(hCommit, true, false)
	if err != nil {
		return nil, err
	}

	newCommits := cache.newCommits()
	sort.Sort(semrel.ByTime(newCommits))

	return &semrel.VCSData{
		CurrentVersion:    currVersion,
		UnreleasedCommits: newCommits,
	}, nil
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

func (cache *commitCache) add(commit *object.Commit, isNew bool, isPreReleased bool) bool {
	entry, hasEntry := cache.commits[commit.Hash.String()]
	if !hasEntry {
		cache.commits[commit.Hash.String()] = &commitCacheEntry{
			isNew: isNew,
			commit: semrel.Commit{
				Msg:         commit.Message,
				SHA:         commit.Hash.String(),
				Time:        commit.Author.When,
				PreReleased: isPreReleased,
			},
		}
		return true
	}
	if isPreReleased {
		entry.commit.PreReleased = true
	}
	if !entry.isNew || entry.isNew == isNew {
		return false
	}
	entry.isNew = isNew
	return true
}
