package lock_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lafke/lafke/internal/lock"
)

func FuzzRead(f *testing.F) {
	f.Add([]byte("version: 2\nartifacts: []\n"))
	f.Add([]byte("{\"version\":1,\"artifacts\":[]}"))
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "lafke.lock")
		if len(data) > 0 && data[0] == '{' {
			path = filepath.Join(dir, "lafke.json")
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		_, _, _ = lock.Read(path)
	})
}
