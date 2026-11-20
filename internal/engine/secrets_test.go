package engine

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestSecretContentsWarnsWhenEngineFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("needs a shell script")
	}
	fake := filepath.Join(t.TempDir(), "lafke-engine")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho 'rulepack: broken' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvPath, fake)
	secretsWarning = sync.Once{}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	got := SecretContents([]string{"a.env"}, [][]byte{[]byte("x")})
	os.Stderr = orig
	w.Close()
	logged, _ := io.ReadAll(r)

	if len(got) != 1 || len(got[0]) != 0 {
		t.Fatalf("got %v, want one empty result", got)
	}
	if !strings.Contains(string(logged), "secret scan skipped") || !strings.Contains(string(logged), "rulepack: broken") {
		t.Fatalf("missing warning, stderr: %q", logged)
	}
}
