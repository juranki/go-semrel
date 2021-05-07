// Package angularcommit analyzes angular-style commit messages
//
// https://gist.github.com/stephenparish/9941e89d80e2bc58a153#format-of-the-commit-message
// https://docs.google.com/document/d/1QrDFcIiPjSLDn3EL15IJygNPiHORgU1_OOAqWjiDU5Y/edit#
package angularcommit

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/juranki/go-semrel/semrel"
)

var (

	// <header>
	// <BLANK LINE>
	// <body>
	// <BLANK LINE>
	// <footer>
	// ---
	// <header>
	// <BLANK LINE>
	// <body>
	// <BLANK LINE>
	// <footer>

	fullAngularHead    = regexp.MustCompile(`(?m)^\s*([a-zA-Z]+)\s*\(([^\)]+)\):\s*([^\n]*)`)
	minimalAngularHead = regexp.MustCompile(`(?m)^\s*([a-zA-Z]+):\s*([^\n]*)`)
	// DefaultOptions for angular commit Analyzer
	DefaultOptions = &Options{
		ChoreTypes:   []string{"chore", "docs", "test"},
		FixTypes:     []string{"fix", "refactor", "perf", "style"},
		FeatureTypes: []string{"feat"},
		BreakingChangeMarkers: []string{
			`BREAKING\s+CHANGE:`,
			`BREAKING\s+CHANGE`,
			`BREAKING:`,
		},
	}
)

// Options control how angular commit analyzer behaves
type Options struct {
	ChoreTypes            []string
	FixTypes              []string
	FeatureTypes          []string
	BreakingChangeMarkers []string
}

// Analyzer is a semrel.Analyzer instance that parses commits
// according to angularjs commit conventions
type Analyzer struct {
	options *Options
}

// NewWithOptions initializes Analyzer with options provided
func NewWithOptions(options *Options) *Analyzer {
	return &Analyzer{
		options: options,
	}
}

// New initializes Analyzer with DefaultOptions
func New() *Analyzer {
	return &Analyzer{}
}

// Lint checks if message is fomatted according to rules specified in analyzer.
// Currently only checs the format of head line and that type is found.
func (analyzer *Analyzer) Lint(message string) []error {
	var choreType = false
	var featureType = false
	var fixType = false
	options := analyzer.options
	if options == nil {
		options = DefaultOptions
	}
	ac := parseAngularHead(message)
	if !checkAllAngularChanges(ac) {
		return []error{errors.New("invalid message head")}
	}

	for _, t := range options.ChoreTypes {
		choreType = hasChangeType(ac, t)
		if choreType == true {
			break
		}
	}
	for _, t := range options.FeatureTypes {
		featureType = hasChangeType(ac, t)
		if featureType == true {
			break
		}
	}
	for _, t := range options.FixTypes {
		fixType = hasChangeType(ac, t)
		if fixType == true {
			break
		}
	}
	if choreType || featureType || fixType {
		return []error{}
	}
	return []error{errors.New("invalid type")}
}

// Analyze implements semrel.Analyzer interface for angularcommit.Analyzer
func (analyzer *Analyzer) Analyze(commit *semrel.Commit) ([]semrel.Change, error) {
	options := analyzer.options
	if analyzer.options == nil {
		options = DefaultOptions
	}
	changes := []semrel.Change{}
	message := commit.Msg
	ac := parseAngularHead(message)
	for _, change := range *ac {
		change.BreakingMessage = parseAngularBreakingChange(message, options.BreakingChangeMarkers)
		change.commit = *commit
		change.options = options
		change.Hash = commit.SHA
		if len(change.Category()) > 0 {
			changes = append(changes, change)
		}
	}
	return changes, nil
}

// Change captures commit message analysis
type Change struct {
	isAngular       bool
	FullHeader      string
	CommitType      string
	Scope           string
	Subject         string
	BreakingMessage string
	Hash            string
	commit          semrel.Commit
	options         *Options
}

// Category implements semrel.Change interface
func (commit *Change) Category() string {
	var categoryMap = map[semrel.BumpLevel]string{
		semrel.NoBump:    "other",
		semrel.BumpMajor: "breaking",
		semrel.BumpMinor: "feature",
		semrel.BumpPatch: "fix",
	}
	return categoryMap[commit.BumpLevel()]
}

// BumpLevel implements semrel.Change interface
func (commit *Change) BumpLevel() semrel.BumpLevel {
	if len(commit.BreakingMessage) > 0 {
		return semrel.BumpMajor
	}
	for _, fType := range commit.options.FeatureTypes {
		if fType == commit.CommitType {
			return semrel.BumpMinor
		}
	}
	for _, fType := range commit.options.FixTypes {
		if fType == commit.CommitType {
			return semrel.BumpPatch
		}
	}
	return semrel.NoBump
}

// PreReleased implements semrel.Change interface
func (commit *Change) PreReleased() bool {
	return commit.commit.PreReleased
}

func parseAngularHead(text string) *[]*Change {
	var allChanges []*Change
	t := strings.Replace(text, "\r", "", -1)
	mFull := fullAngularHead.FindAllStringSubmatch(t, -1)
	if len(mFull) > 0 {
		for _, match := range mFull {
			allChanges = append(allChanges,
				&Change{
					isAngular:  true,
					FullHeader: strings.ToLower(strings.Trim(match[0], " \t\n")),
					CommitType: strings.ToLower(strings.Trim(match[1], " \t\n")),
					Scope:      strings.ToLower(strings.Trim(match[2], " \t\n")),
					Subject:    strings.Trim(match[3], " \t\n"),
				},
			)
		}
	}

	mMinimal := minimalAngularHead.FindAllStringSubmatch(t, -1)
	if len(mMinimal) > 0 {
		for _, match := range mMinimal {
			allChanges = append(allChanges,
				&Change{
					isAngular:  true,
					FullHeader: strings.ToLower(strings.Trim(match[0], " \t\n")),
					CommitType: strings.ToLower(strings.Trim(match[1], " \t\n")),
					Subject:    strings.Trim(match[2], " \t\n"),
				},
			)
		}
	}

	if len(mFull) == 0 && len(mMinimal) == 0 {
		allChanges = append(allChanges,
			&Change{
				isAngular: false,
				Subject:   strings.Trim(strings.Split(text, "\n")[0], " \n\t"),
			},
		)
	}
	return &allChanges
}

func parseAngularBreakingChange(text string, markers []string) string {
	for _, marker := range markers {
		re, err := regexp.Compile(`(?ms)` + marker + `\s+(.*)`)
		if err != nil {
			fmt.Printf("WARNING: unable to compile regular expression for marker '%s'\n", marker)
			continue
		}
		if match := re.FindStringSubmatch(text); len(match) > 0 {
			return strings.Trim(match[1], " \n\t")
		}

	}
	return ""
}
