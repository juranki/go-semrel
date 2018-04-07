// Package angularcommit analyzes angular-style commit messages
//
// https://gist.github.com/stephenparish/9941e89d80e2bc58a153#format-of-the-commit-message
// https://docs.google.com/document/d/1QrDFcIiPjSLDn3EL15IJygNPiHORgU1_OOAqWjiDU5Y/edit#
package angularcommit

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	fullAngularHead    = regexp.MustCompile(`^\s*([a-zA-Z]+)\s*\(([^\)]+)\):\s*([^\n]*)`)
	minimalAngularHead = regexp.MustCompile(`^\s*([a-zA-Z]+):\s*([^\n]*)`)
)

type angularHead struct {
	isAngular  bool
	commitType string
	scope      string
	subject    string
}

func parseAngularHead(text string) *angularHead {
	t := strings.Replace(text, "\r", "", -1)
	if match := fullAngularHead.FindStringSubmatch(t); len(match) > 0 {
		return &angularHead{
			isAngular:  true,
			commitType: strings.ToLower(strings.Trim(match[1], " \t\n")),
			scope:      strings.ToLower(strings.Trim(match[2], " \t\n")),
			subject:    strings.Trim(match[3], " \t\n"),
		}
	}
	if match := minimalAngularHead.FindStringSubmatch(text); len(match) > 0 {
		return &angularHead{
			isAngular:  true,
			commitType: strings.ToLower(strings.Trim(match[1], " \t\n")),
			subject:    strings.Trim(match[2], " \t\n"),
		}
	}
	return &angularHead{
		isAngular: false,
		subject:   strings.Trim(strings.Split(text, "\n")[0], " \n\t"),
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
