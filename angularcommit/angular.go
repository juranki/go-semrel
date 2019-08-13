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
	fullAngularHead    = regexp.MustCompile(`^\s*([a-zA-Z]+)\s*\(([^\)]+)\):\s*([^\n]*)`)
	minimalAngularHead = regexp.MustCompile(`^\s*([a-zA-Z]+):\s*([^\n]*)`)
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
	options := analyzer.options
	if options == nil {
		options = DefaultOptions
	}
	ac := parseAngularHead(message)
	if !ac.isAngular {
		return []error{errors.New("invalid message head")}
	}
	for _, t := range options.ChoreTypes {
		if ac.CommitType == t {
			return []error{}
		}
	}
	for _, t := range options.FeatureTypes {
		if ac.CommitType == t {
			return []error{}
		}
	}
	for _, t := range options.FixTypes {
		if ac.CommitType == t {
			return []error{}
		}
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
	ac.BreakingMessage = parseAngularBreakingChange(message, options.BreakingChangeMarkers)
	ac.commit = *commit
	ac.options = options
	ac.Hash = commit.SHA
	if len(ac.Category()) > 0 {
		changes = append(changes, ac)
	}
	return changes, nil
}

// Change captures commit message analysis
type Change struct {
	isAngular       bool
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

func parseAngularHead(text string) *Change {
	t := strings.Replace(text, "\r", "", -1)
	if match := fullAngularHead.FindStringSubmatch(t); len(match) > 0 {
		return &Change{
			isAngular:  true,
			CommitType: strings.ToLower(strings.Trim(match[1], " \t\n")),
			Scope:      strings.ToLower(strings.Trim(match[2], " \t\n")),
			Subject:    strings.Trim(match[3], " \t\n"),
		}
	}
	if match := minimalAngularHead.FindStringSubmatch(text); len(match) > 0 {
		return &Change{
			isAngular:  true,
			CommitType: strings.ToLower(strings.Trim(match[1], " \t\n")),
			Subject:    strings.Trim(match[2], " \t\n"),
		}
	}
	return &Change{
		isAngular: false,
		Subject:   strings.Trim(strings.Split(text, "\n")[0], " \n\t"),
	}
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
