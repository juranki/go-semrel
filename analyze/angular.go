/*
Analyze angular-style commit message

https://gist.github.com/stephenparish/9941e89d80e2bc58a153#format-of-the-commit-message
https://docs.google.com/document/d/1QrDFcIiPjSLDn3EL15IJygNPiHORgU1_OOAqWjiDU5Y/edit#
*/

package analyze

import (
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

func parseAngularBreakingChange() {

}
