package angularcommit

import (
	"reflect"
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
		{"fix(testing): angular head\nfix(testing): angular head", true, "fix", "testing", "angular head"},
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
		for _, cg := range *ah {
			if cg.isAngular != c.isAngular {
				t.Errorf("'%s': got isAngular=%t, want %t\n", c.msg, cg.isAngular, c.isAngular)
			}
			if cg.CommitType != c.commitType {
				t.Errorf("'%s': got type '%s', want '%s'\n", c.msg, cg.CommitType, c.commitType)
			}
			if cg.Scope != c.scope {
				t.Errorf("'%s': got scope '%s', want '%s'\n", c.msg, cg.Scope, c.scope)
			}
			if cg.Subject != c.subject {
				t.Errorf("'%s': got subject '%s', want '%s'\n", c.msg, cg.Subject, c.subject)
			}
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
			t.Errorf("got %s, want %s\n", result, c.result)
		}
	}
}

func TestAnalyzer_Lint(t *testing.T) {
	analyzer := New()
	tests := []struct {
		name    string
		message string
		want    []string
	}{
		{"simple ok", "chore: test", []string{}},
		{"no type", "test", []string{"invalid message head"}},
		{"invalid type", "foo: test", []string{"invalid type"}},
		{"with scope", "test(test): test", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := analyzer.Lint(tt.message)
			got := make([]string, len(errs))
			for i, s := range errs {
				got[i] = s.Error()
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Analyzer.Lint() = %v, want %v", got, tt.want)
			}
		})
	}
}
