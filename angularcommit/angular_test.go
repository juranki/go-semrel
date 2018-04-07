package angularcommit

import (
	"testing"
)

func TestAngularHead(t *testing.T) {
	cases := []struct {
		msg        string
		isAngular  bool
		commitType string
		scope      string
		subject    string
	}{
		{"fix(testing): angular head", true, "fix", "testing", "angular head"},
		{"fix(testing): angular head  ", true, "fix", "testing", "angular head"},
		{"Fix(Testing): Angular head  ", true, "fix", "testing", "Angular head"},
		{" fix (testing): angular head", true, "fix", "testing", "angular head"},
		{" fix ( testing ): angular head", true, "fix", "testing", "angular head"},
		{"fix: angular head", true, "fix", "", "angular head"},
		{"Fix: Angular head", true, "fix", "", "Angular head"},
		{"angular head", false, "", "", "angular head"},
		{"trailing newline\n", false, "", "", "trailing newline"},
		{" trim\n", false, "", "", "trim"},
		{" fix:trailing newline\nasdf", true, "fix", "", "trailing newline"},
	}
	for _, c := range cases {
		ah := parseAngularHead(c.msg)
		if ah.isAngular != c.isAngular {
			t.Errorf("'%s': expected isAngular=%t, got %t\n", c.msg, c.isAngular, ah.isAngular)
		}
		if ah.commitType != c.commitType {
			t.Errorf("'%s': expected type '%s', got '%s'\n", c.msg, c.commitType, ah.commitType)
		}
		if ah.scope != c.scope {
			t.Errorf("'%s': expected scope '%s', got '%s'\n", c.msg, c.scope, ah.scope)
		}
		if ah.subject != c.subject {
			t.Errorf("'%s': expected subject '%s', got '%s'\n", c.msg, c.subject, ah.subject)
		}
	}
}

func TestBreakingChange(t *testing.T) {
	markers := []string{"break:", "break"}
	cases := []struct {
		msg    string
		result string
	}{
		{"foo\n\nbreak message", "message"},
		{"foo\n\nbreak: message\nsecondline\n  ", "message\nsecondline"},
		{"foo\n\nbrek: message\nsecondline\n  ", ""},
	}
	for _, c := range cases {
		result := parseAngularBreakingChange(c.msg, markers)
		if result != c.result {
			t.Errorf("expected %s, got %s\n", c.result, result)
		}
	}
}
