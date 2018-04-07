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

// ReleaseInput contains data collected from VCS and an analyzer function
type ReleaseInput struct {
	CurrentVersion    semver.Version
	UnreleasedChanges []RawChange
	ChangeAnalyzer    func(Message string) ([]Change, error)
}

// RawChange provides accessors to VCS commit data
type RawChange interface {
	Msg() string
	SHA() string
	Time() time.Time
}

// Change captures ChangeAnalyzer results
type Change interface {
	Category() string
	BumpLevel() BumpLevel
	Render() string
}

// ReleaseOutput contains information for next release
type ReleaseOutput struct {
	CurrentVersion semver.Version
	NextVersion    semver.Version
	Changes        map[string][]Change
}

// Release processes the release data
func Release(input *ReleaseInput) (*ReleaseOutput, error) {
	output := &ReleaseOutput{
		CurrentVersion: input.CurrentVersion,
		NextVersion:    input.CurrentVersion,
		Changes:        map[string][]Change{},
	}
	bumpLevel := NoBump
	for _, rawChange := range input.UnreleasedChanges {
		changes, err := input.ChangeAnalyzer(rawChange.Msg())
		if err != nil {
			return nil, err
		}
		for _, change := range changes {
			if category, catOK := output.Changes[change.Category()]; catOK {
				output.Changes[change.Category()] = append(category, change)
			} else {
				output.Changes[change.Category()] = []Change{change}
			}
			if change.BumpLevel() > bumpLevel {
				bumpLevel = change.BumpLevel()
			}
		}
	}
	output.NextVersion = bump(output.CurrentVersion, bumpLevel)
	return output, nil
}

func bump(curr semver.Version, bumpLevel BumpLevel) semver.Version {
	var major uint64
	var minor uint64
	var patch uint64
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
