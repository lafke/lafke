package verify

import (
	"github.com/lafke/lafke/internal/diff"
	"github.com/lafke/lafke/pkg/lockfile"
)

func Compare(locked, current lockfile.Document) diff.Result {
	return diff.Compare(locked, current)
}
