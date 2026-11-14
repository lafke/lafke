package provenance

import "github.com/lafke/lafke/pkg/lockfile"

func State() lockfile.EvidenceState {
	return lockfile.EvidenceUnknown
}
