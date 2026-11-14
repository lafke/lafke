package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func runUninstallCmd(t *testing.T, exe string, args ...string) (string, error) {
	t.Helper()
	old := executablePath
	executablePath = func() (string, error) { return exe, nil }
	t.Cleanup(func() { executablePath = old })

	root := NewRoot("test", "none", "unknown")
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetIn(strings.NewReader(""))
	root.SetArgs(append([]string{"uninstall"}, args...))
	err := root.Execute()
	return buf.String(), err
}

func fakeInstall(t *testing.T) (dir, exe, engine string) {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("HOME", cache)
	t.Setenv("LocalAppData", cache)
	dir = t.TempDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	exe = filepath.Join(dir, "lafke"+ext)
	engine = filepath.Join(dir, "lafke-engine"+ext)
	for _, p := range []string{exe, engine, filepath.Join(dir, "other")} {
		if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir, exe, engine
}

func TestUninstallDryRun(t *testing.T) {
	_, exe, engine := fakeInstall(t)
	out, err := runUninstallCmd(t, exe, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, exe) || !strings.Contains(out, engine) {
		t.Fatalf("dry run must list both executables:\n%s", out)
	}
	for _, p := range []string{exe, engine} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("dry run removed %s", p)
		}
	}
}

func TestUninstallNeedsConfirmation(t *testing.T) {
	_, exe, _ := fakeInstall(t)
	if _, err := runUninstallCmd(t, exe); err == nil {
		t.Fatal("a non-interactive run without --yes must be refused")
	}
	if _, err := os.Stat(exe); err != nil {
		t.Fatal("executable was removed without confirmation")
	}
}

func TestUninstallRemovesOnlyOwnFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a running executable cannot be removed on Windows")
	}
	dir, exe, engine := fakeInstall(t)
	if _, err := runUninstallCmd(t, exe, "--yes"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{exe, engine} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s should be removed", p)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "other")); err != nil {
		t.Fatal("an unrelated file next to the executable must stay")
	}
}

func TestUninstallRefusesGoRunBuild(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "go-build123", "b001", "exe", "lafke")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := runUninstallCmd(t, exe, "--yes"); err == nil {
		t.Fatal("a go run build must not be removed")
	}
	if _, err := os.Stat(exe); err != nil {
		t.Fatal("go run build was removed")
	}
}
