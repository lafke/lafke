package provenance

import "github.com/lafke/cli/pkg/lockfile"

func State() lockfile.EvidenceState {
	return lockfile.EvidenceUnknown
}
