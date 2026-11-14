package verify

import (
	"github.com/lafke/cli/internal/diff"
	"github.com/lafke/cli/pkg/lockfile"
)

func Compare(locked, current lockfile.Document) diff.Result {
	return diff.Compare(locked, current)
}
