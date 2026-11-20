package engine

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// release is one fake engine release served by the test server.
type release struct {
	version  string
	protocol int
}

// fakeReleases serves releases in the layout of the engine-releases repository. The newest
// release is "latest". The engine in each archive is a shell script that reports its version.
func fakeReleases(t *testing.T, releases ...release) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake engine is a shell script")
	}
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("HOME", cache)
	t.Setenv("LocalAppData", cache)
	t.Setenv(EnvPath, "")
	t.Setenv(EnvVersion, "")

	files := map[string][]byte{}
	var apiList []string
	for i, r := range releases {
		script := fmt.Sprintf("#!/bin/sh\necho '{\"protocol\":%d,\"version\":\"%s\"}'\n", r.protocol, r.version)
		archive := tarGz(t, executableName(), []byte(script))
		sum := sha256.Sum256(archive)
		manifest := fmt.Sprintf(`{"version":%q,"protocol":%d,"archives":{%q:%q}}`,
			r.version, r.protocol, archiveName(), hex.EncodeToString(sum[:]))
		files["/releases/download/"+r.version+"/"+manifestName] = []byte(manifest)
		files["/releases/download/"+r.version+"/"+archiveName()] = archive
		if i == 0 {
			files["/releases/latest/download/"+manifestName] = []byte(manifest)
		}
		apiList = append(apiList, fmt.Sprintf(`{"tag_name":%q,"draft":false,"prerelease":false}`, r.version))
	}
	files["/api/releases"] = []byte("[" + strings.Join(apiList, ",") + "]")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)

	oldURL, oldAPI, oldVerify := releasesURL, releasesAPI, verifyAttestation
	releasesURL, releasesAPI = srv.URL+"/releases", srv.URL+"/api/releases"
	verifyAttestation = func(context.Context, string) error { return nil }
	t.Cleanup(func() { releasesURL, releasesAPI, verifyAttestation = oldURL, oldAPI, oldVerify })
}

func tarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	tw.Write(content)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestInstallLatestAndLocate(t *testing.T) {
	fakeReleases(t, release{"v1.2.0", ProtocolVersion})
	in, err := Install(context.Background(), InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if in.Version != "v1.2.0" || in.Protocol != ProtocolVersion {
		t.Fatalf("installed %+v", in)
	}
	if want := filepath.Join(InstallDir(), "v1.2.0", executableName()); in.Path != want {
		t.Fatalf("path %s, want %s", in.Path, want)
	}
	if st, err := os.Stat(in.Path); err != nil || st.Mode()&0o111 == 0 {
		t.Fatalf("engine is not executable: %v", err)
	}
	got, ok := installedPath()
	if !ok || got != in.Path {
		t.Fatalf("installedPath = %q, %v", got, ok)
	}
	if v, err := checkPath(got); err != nil || v != "v1.2.0" {
		t.Fatalf("checkPath = %q, %v", v, err)
	}
}

func TestInstallSkipsIncompatibleLatest(t *testing.T) {
	fakeReleases(t,
		release{"v2.0.0", ProtocolVersion + 1},
		release{"v1.5.0", ProtocolVersion},
		release{"v1.4.0", ProtocolVersion},
	)
	in, err := Install(context.Background(), InstallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if in.Version != "v1.5.0" {
		t.Fatalf("installed %s, want the newest compatible v1.5.0", in.Version)
	}
}

func TestInstallRejectsIncompatibleVersion(t *testing.T) {
	fakeReleases(t, release{"v2.0.0", ProtocolVersion + 1})
	if _, err := Install(context.Background(), InstallOptions{Version: "v2.0.0"}); err == nil || !strings.Contains(err.Error(), "protocol") {
		t.Fatalf("err = %v", err)
	}
	if _, err := Install(context.Background(), InstallOptions{}); err == nil {
		t.Fatal("no release speaks this protocol, but Install succeeded")
	}
}

func TestInstallChecksum(t *testing.T) {
	fakeReleases(t, release{"v1.0.0", ProtocolVersion})
	// Serve a manifest whose checksum does not match the archive.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, manifestName):
			fmt.Fprintf(w, `{"version":"v1.0.0","protocol":%d,"archives":{%q:%q}}`, ProtocolVersion, archiveName(), strings.Repeat("0", 64))
		default:
			w.Write(tarGz(t, executableName(), []byte("#!/bin/sh\n")))
		}
	}))
	defer srv.Close()
	releasesURL = srv.URL
	_, err := Install(context.Background(), InstallOptions{Version: "v1.0.0"})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v", err)
	}
	if len(List()) != 0 {
		t.Fatal("an engine with a bad checksum was installed")
	}
}

func TestInstallAttestationFailure(t *testing.T) {
	fakeReleases(t, release{"v1.0.0", ProtocolVersion})
	verifyAttestation = func(context.Context, string) error { return errors.New("no attestation") }
	if _, err := Install(context.Background(), InstallOptions{}); err == nil || !strings.Contains(err.Error(), "no attestation") {
		t.Fatalf("err = %v", err)
	}
	if len(List()) != 0 {
		t.Fatal("an unattested engine was installed")
	}
	// Skipping verification still installs, after the checksum passes.
	if _, err := Install(context.Background(), InstallOptions{SkipAttestation: true}); err != nil {
		t.Fatal(err)
	}
}

func TestInstalledPathPrefersNewestAndHonorsPin(t *testing.T) {
	fakeReleases(t, release{"v1.10.0", ProtocolVersion}, release{"v1.9.0", ProtocolVersion})
	for _, v := range []string{"v1.9.0", "v1.10.0"} {
		if _, err := Install(context.Background(), InstallOptions{Version: v}); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := installedPath()
	if !strings.Contains(got, "v1.10.0") {
		t.Fatalf("installedPath = %s, want v1.10.0 (semver order, not string order)", got)
	}
	t.Setenv(EnvVersion, "v1.9.0")
	got, ok := installedPath()
	if !ok || !strings.Contains(got, "v1.9.0") {
		t.Fatalf("pinned installedPath = %s, %v", got, ok)
	}
	t.Setenv(EnvVersion, "v0.1.0")
	if _, err := Locate(); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("Locate with a missing pin: %v", err)
	}
}

func TestListIgnoresOtherProtocols(t *testing.T) {
	fakeReleases(t, release{"v1.0.0", ProtocolVersion})
	if _, err := Install(context.Background(), InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(InstallDir(), "v9.0.0")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(other, executableName()), []byte("x"), 0o755)
	os.WriteFile(filepath.Join(other, metaName), []byte(fmt.Sprintf(`{"version":"v9.0.0","protocol":%d}`, ProtocolVersion+1)), 0o644)
	list := List()
	if len(list) != 1 || list[0].Version != "v1.0.0" {
		t.Fatalf("List = %+v", list)
	}
}

func TestRemove(t *testing.T) {
	fakeReleases(t, release{"v1.1.0", ProtocolVersion}, release{"v1.0.0", ProtocolVersion})
	for _, v := range []string{"v1.0.0", "v1.1.0"} {
		if _, err := Install(context.Background(), InstallOptions{Version: v}); err != nil {
			t.Fatal(err)
		}
	}
	removed, err := Remove("v1.1.0")
	if err != nil || len(removed) != 1 || removed[0] != "v1.0.0" {
		t.Fatalf("Remove = %v, %v", removed, err)
	}
	if list := List(); len(list) != 1 || list[0].Version != "v1.1.0" {
		t.Fatalf("List = %+v", list)
	}
}

func TestBundledIsPreferredOverInstalled(t *testing.T) {
	fakeReleases(t, release{"v1.0.0", ProtocolVersion})
	if _, err := Install(context.Background(), InstallOptions{}); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	bundled := filepath.Join(filepath.Dir(exe), executableName())
	if _, err := os.Stat(bundled); err == nil {
		t.Skip("an engine already sits next to the test executable")
	}
	if _, ok := Bundled(); ok {
		t.Fatal("Bundled found an engine that is not there")
	}
	if err := os.WriteFile(bundled, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(bundled)
	if p, ok := Bundled(); !ok || p != bundled {
		t.Fatalf("Bundled = %q, %v", p, ok)
	}
	if p, err := Locate(); err != nil || p != bundled {
		t.Fatalf("Locate = %q, %v; want the bundled engine", p, err)
	}
	os.Remove(bundled)
	if p, err := Locate(); err != nil || !strings.Contains(p, "v1.0.0") {
		t.Fatalf("Locate = %q, %v; want the installed engine", p, err)
	}
}
