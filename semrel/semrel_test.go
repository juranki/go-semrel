package semrel

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/blang/semver"
)

// implement Change interface for BumpLevel
func (change BumpLevel) Category() string     { return fmt.Sprintf("%d", int(change)) }
func (change BumpLevel) BumpLevel() BumpLevel { return change }

type analyzer struct{}

func (a analyzer) Analyze(commit *Commit) ([]Change, error) {
	msg := commit.Msg
	if strings.HasPrefix(msg, "fix") {
		return []Change{BumpLevel(BumpPatch)}, nil
	}
	if strings.HasPrefix(msg, "feat") {
		return []Change{BumpLevel(BumpMinor)}, nil
	}
	if strings.HasPrefix(msg, "break") {
		return []Change{BumpLevel(BumpMajor)}, nil
	}
	if strings.HasPrefix(msg, "fail") {
		return nil, fmt.Errorf("fail")
	}
	return []Change{}, nil
}

var dummyAnalyzer = analyzer{}

func TestNoChangesBump(t *testing.T) {
	input := &VCSData{
		CurrentVersion: semver.MustParse("0.1.0"),
		UnreleasedCommits: []Commit{
			{"aaa", "", time.Now()},
		},
	}
	output, err := Release(input, dummyAnalyzer)
	if err != nil {
		t.Error(err)
	}
	fmt.Print(output)
	if output.NextVersion.String() != "0.1.0" {
		t.Error(output.NextVersion.String())
	}
}

func TestBump(t *testing.T) {
	data := []struct {
		origVersion string
		bumpLevel   BumpLevel
		newVersion  string
	}{
		{"0.0.0", NoBump, "0.0.0"},
		{"0.0.0", BumpPatch, "0.0.1"},
		{"0.2.1", BumpMinor, "0.3.0"},
		{"0.2.1", BumpMajor, "0.3.0"},
		{"1.2.1", BumpMajor, "2.0.0"},
	}
	for _, d := range data {
		bumpedVersion := bump(semver.MustParse(d.origVersion), d.bumpLevel).String()
		if bumpedVersion != d.newVersion {
			t.Errorf("with %+v, got %s, want %s", d, bumpedVersion, d.newVersion)
		}
	}
}

func TestRelease1(t *testing.T) {
	input := &VCSData{
		CurrentVersion: semver.MustParse("0.0.0"),
		UnreleasedCommits: []Commit{
			{"fix", "", time.Now()},
		},
	}
	output, err := Release(input, dummyAnalyzer)
	if err != nil {
		t.Error(err)
	}
	if output.NextVersion.String() != "0.0.1" {
		t.Errorf("got %s, want 0.0.1", output.NextVersion.String())
	}
	if len(output.Changes["1"]) != 1 {
		t.Errorf("got %d, want 1 fix", len(output.Changes["1"]))
	}
}

func TestRelease2(t *testing.T) {
	input := &VCSData{
		CurrentVersion: semver.MustParse("1.2.3"),
		UnreleasedCommits: []Commit{
			{"fix", "", time.Now()},
			{"fix", "", time.Now()},
			{"feat", "", time.Now()},
			{"break", "", time.Now()},
		},
	}
	output, err := Release(input, dummyAnalyzer)
	if err != nil {
		t.Error(err)
	}
	if output.NextVersion.String() != "2.0.0" {
		t.Errorf("got %s, want 2.0.0", output.NextVersion.String())
	}
	if len(output.Changes["1"]) != 2 {
		t.Errorf("got %d, want 2 fixes", len(output.Changes["1"]))
	}
	if len(output.Changes["2"]) != 1 {
		t.Errorf("got %d, want 1 features", len(output.Changes["2"]))
	}
	if len(output.Changes["3"]) != 1 {
		t.Errorf("got %d, want expected 1 breaking change", len(output.Changes["3"]))
	}
}

func TestRelease3(t *testing.T) {
	input := &VCSData{
		CurrentVersion: semver.MustParse("1.2.3"),
		UnreleasedCommits: []Commit{
			{"fix", "", time.Now()},
			{"fix", "", time.Now()},
			{"fail", "", time.Now()},
			{"break", "", time.Now()},
		},
	}
	_, err := Release(input, dummyAnalyzer)
	if err == nil || err.Error() != "fail" {
		t.Errorf("got %+v, want error", err)
	}
}
