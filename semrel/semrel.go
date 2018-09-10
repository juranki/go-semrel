// Package semrel processes version control data using an analyser
// function, to produce data for a release note
package semrel

import (
	"fmt"
	"time"

	"github.com/blang/semver"
)

// BumpLevel of the release and/or individual commit
type BumpLevel int

// BumpLevel values
const (
	NoBump    BumpLevel = iota
	BumpPatch           = iota
	BumpMinor           = iota
	BumpMajor           = iota
)

// ChangeAnalyzer analyzes a commit message and returns 0 or more entries to release note
type ChangeAnalyzer interface {
	Analyze(commit *Commit) ([]Change, error)
}

// VCSData contains data collected from version control system
type VCSData struct {
	CurrentVersion    semver.Version
	UnreleasedCommits []Commit
	// Time of the commit being released
	Time time.Time
}

// Commit contains VCS commit data
type Commit struct {
	Msg         string
	SHA         string
	Time        time.Time
	PreReleased bool
}

// ByTime implements sort.Interface for []Commit based on Time().
type ByTime []Commit

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].Time.Before(a[j].Time) }

// Change captures ChangeAnalyzer results
type Change interface {
	Category() string
	BumpLevel() BumpLevel
	PreReleased() bool
}

// ReleaseData contains information for next release
type ReleaseData struct {
	CurrentVersion semver.Version
	NextVersion    semver.Version
	BumpLevel      BumpLevel
	Changes        map[string][]Change
	// Time of the commit being released
	Time time.Time
}

// Release processes the release data
func Release(input *VCSData, analyzer ChangeAnalyzer) (*ReleaseData, error) {
	output := &ReleaseData{
		CurrentVersion: input.CurrentVersion,
		NextVersion:    input.CurrentVersion,
		BumpLevel:      NoBump,
		Changes:        map[string][]Change{},
		Time:           input.Time,
	}
	for _, commit := range input.UnreleasedCommits {
		changes, err := analyzer.Analyze(&commit)
		if err != nil {
			return nil, err
		}
		for _, change := range changes {
			if category, catOK := output.Changes[change.Category()]; catOK {
				output.Changes[change.Category()] = append(category, change)
			} else {
				output.Changes[change.Category()] = []Change{change}
			}
			if change.BumpLevel() > output.BumpLevel {
				output.BumpLevel = change.BumpLevel()
			}
		}
	}
	output.NextVersion = bump(output.CurrentVersion, output.BumpLevel)
	return output, nil
}

func bump(curr semver.Version, bumpLevel BumpLevel) semver.Version {
	var major uint64
	var minor uint64
	var patch uint64
	if bumpLevel == NoBump {
		return semver.MustParse(curr.String())
	}
	if bumpLevel == BumpMajor && curr.Major > 0 {
		major = curr.Major + 1
	}
	if bumpLevel == BumpMinor || (curr.Major == 0 && bumpLevel == BumpMajor) {
		major = curr.Major
		minor = curr.Minor + 1
	}
	if bumpLevel == BumpPatch {
		major = curr.Major
		minor = curr.Minor
		patch = curr.Patch + 1
	}
	return semver.MustParse(fmt.Sprintf("%d.%d.%d", major, minor, patch))
}
