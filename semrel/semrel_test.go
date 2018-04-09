package semrel

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/blang/semver"
)

type dummyCommit string

func (commit dummyCommit) Msg() string     { return string(commit) }
func (commit dummyCommit) SHA() string     { return string(commit) }
func (commit dummyCommit) Time() time.Time { return time.Now() }

func (change BumpLevel) Category() string              { return fmt.Sprintf("%d", int(change)) }
func (change BumpLevel) BumpLevel() BumpLevel          { return change }
func (change BumpLevel) Render(category string) string { return "" }

type analyzer struct{}

func (a analyzer) Analyze(commit Commit) ([]Change, error) {
	msg := commit.Msg()
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

func TestBump(t *testing.T) {
	if bump(semver.MustParse("0.0.0"), NoBump).String() != "0.0.0" {
		t.Error("NoBump")
	}
	if bump(semver.MustParse("0.0.0"), BumpPatch).String() != "0.0.1" {
		t.Error("BumpPatch")
	}
	if bump(semver.MustParse("0.2.1"), BumpMinor).String() != "0.3.0" {
		t.Error("BumpMinor")
	}
	if bump(semver.MustParse("0.2.1"), BumpMajor).String() != "0.3.0" {
		t.Error("BumpMajor alpha")
	}
	if bump(semver.MustParse("1.2.1"), BumpMajor).String() != "2.0.0" {
		t.Error("BumpMajor")
	}
}

func TestRelease1(t *testing.T) {
	input := &VCSData{
		CurrentVersion: semver.MustParse("0.0.0"),
		UnreleasedCommits: []Commit{
			dummyCommit("fix"),
		},
	}
	output, err := Release(input, dummyAnalyzer)
	if err != nil {
		t.Error(err)
	}
	if output.NextVersion.String() != "0.0.1" {
		t.Errorf("expected 0.0.1, got %s", output.NextVersion.String())
	}
	if len(output.Changes["1"]) != 1 {
		t.Errorf("expected one fix, got %d", len(output.Changes["1"]))
	}
}

func TestRelease2(t *testing.T) {
	input := &VCSData{
		CurrentVersion: semver.MustParse("1.2.3"),
		UnreleasedCommits: []Commit{
			dummyCommit("fix"),
			dummyCommit("fix"),
			dummyCommit("feat"),
			dummyCommit("break"),
		},
	}
	output, err := Release(input, dummyAnalyzer)
	if err != nil {
		t.Error(err)
	}
	if output.NextVersion.String() != "2.0.0" {
		t.Errorf("expected 0.0.1, got %s", output.NextVersion.String())
	}
	if len(output.Changes["1"]) != 2 {
		t.Errorf("expected two fixes, got %d", len(output.Changes["1"]))
	}
	if len(output.Changes["2"]) != 1 {
		t.Errorf("expected one feature, got %d", len(output.Changes["2"]))
	}
	if len(output.Changes["3"]) != 1 {
		t.Errorf("expected one breaking change, got %d", len(output.Changes["3"]))
	}
}

func TestRelease3(t *testing.T) {
	input := &VCSData{
		CurrentVersion: semver.MustParse("1.2.3"),
		UnreleasedCommits: []Commit{
			dummyCommit("fix"),
			dummyCommit("fix"),
			dummyCommit("fail"),
			dummyCommit("break"),
		},
	}
	_, err := Release(input, dummyAnalyzer)
	if err == nil || err.Error() != "fail" {
		t.Errorf("expected error, got %+v", err)
	}
}
