// Package enginetest lets tests that need the analysis engine skip when it is absent.
package enginetest

import (
	"testing"

	"github.com/lafke/cli/internal/engine"
)

// Require skips the test unless the engine executable can be located.
func Require(t testing.TB) {
	t.Helper()
	if !engine.Available() {
		t.Skip("lafke-engine not available; set LAFKE_ENGINE to run this test")
	}
}
