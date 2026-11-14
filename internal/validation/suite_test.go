package validation_test

import (
	"github.com/lafke/cli/internal/finding"
	"github.com/lafke/cli/internal/validation"
	"strings"
	"testing"
)

func TestMissingFindingsAndUnsafeCommands(t *testing.T) {
	r := validation.Evaluate(validation.Step{Name: "x", Expected: []string{"risk"}}, finding.New("security", nil), nil)
	if r.Passed {
		t.Fatal("missing risk passed")
	}
	for _, s := range []string{`{"version":1,"steps":[{"name":"x","command":"watch"}]}`, `{"version":1,"steps":[{"name":"x","command":"security","args":["--output=/tmp/report"]}]}`} {
		if _, e := validation.Read(strings.NewReader(s)); e == nil {
			t.Fatal("unsafe suite accepted")
		}
	}
}
